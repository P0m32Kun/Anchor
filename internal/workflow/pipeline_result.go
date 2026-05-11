package workflow

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/P0m32Kun/Anchor/internal/fingerprint"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/parser"
	"github.com/P0m32Kun/Anchor/internal/util"
)

func (p *Pipeline) saveFingerprints(fpResults []fingerprint.NmapServiceResult) {
	if len(fpResults) == 0 {
		return
	}
	fps := fingerprint.ConvertToServiceFingerprints(p.projectID, fpResults)
	for i := range fps {
		fps[i].ID = util.GenerateID()
		fps[i].ProjectID = p.projectID
		fps[i].CreatedAt = time.Now().UTC()
		if err := p.queries.SaveServiceFingerprint(&fps[i]); err != nil {
			log.Printf("[pipeline] save service fingerprint: %v", err)
		}
	}
}

func (p *Pipeline) runHTTPXAndNuclei(ctx context.Context, fpResults []fingerprint.NmapServiceResult, extraHTTPXTargets []string) error {
	webResults, nonWebResults := fingerprint.SplitByServiceType(fpResults)

	httpxTargets := makeHTTPTargets(webResults)
	for _, t := range extraHTTPXTargets {
		httpxTargets = append(httpxTargets, t)
	}
	httpxTargets = dedupStrings(httpxTargets)

	var webEndpoints []*models.WebEndpoint
	if len(httpxTargets) > 0 {
		p.setStage(StageHTTPX)
		endpoints, err := p.runHttpx(ctx, httpxTargets)
		if err != nil {
			log.Printf("httpx: %v", err)
			p.failStage(StageHTTPX, err.Error())
		} else {
			webEndpoints = endpoints
			p.completeStage(StageHTTPX)
		}
	}

	if p.config.EnableNuclei {
		p.setStage(StageVuln)
		if len(webEndpoints) > 0 {
			if err := p.runNucleiWeb(ctx, webEndpoints); err != nil {
				log.Printf("nuclei web: %v", err)
				p.failStage(StageVuln, "nuclei web: "+err.Error())
			}
		}
		if len(nonWebResults) > 0 {
			if err := p.runNucleiNonWeb(ctx, nonWebResults); err != nil {
				log.Printf("nuclei non-web: %v", err)
				p.failStage(StageVuln, "nuclei non-web: "+err.Error())
			}
		}
		p.completeStage(StageVuln)
	}

	return nil
}

func (p *Pipeline) saveNucleiFindings(stdout []byte, urlToEndpoint map[string]*models.WebEndpoint, ipToAsset map[string]*models.Asset) {
	if len(stdout) == 0 {
		return
	}
	results, parseErrs := parser.ParseNuclei(bytes.NewReader(stdout))
	for _, pe := range parseErrs {
		log.Printf("[saveNucleiFindings] parse error line=%d msg=%s", pe.Line, pe.Message)
	}
	for _, nr := range results {
		dedupKey := fmt.Sprintf("%s|%s|%s", nr.TemplateID, nr.Host, nr.MatcherName)
		existing, err := p.queries.GetFindingByDedupKey(p.projectID, dedupKey)
		if err != nil {
			continue
		}
		if existing != nil {
			continue
		}
		var assetID, webEndpointID *string
		if urlToEndpoint != nil {
			if ep, ok := urlToEndpoint[nr.Host]; ok {
				assetID = &ep.AssetID
				webEndpointID = &ep.ID
			}
		}
		if assetID == nil && ipToAsset != nil {
			if a, ok := ipToAsset[nr.Host]; ok {
				assetID = &a.ID
			}
		}
		f := &models.Finding{
			ID:            util.GenerateID(),
			ProjectID:     p.projectID,
			AssetID:       assetID,
			WebEndpointID: webEndpointID,
			SourceTool:    "nuclei",
			SourceRuleID:  nr.TemplateID,
			DedupKey:      dedupKey,
			Title:         nr.Name,
			Severity:      models.FindingSeverity(nr.Severity),
			Confidence:    80,
			Priority:      3,
			Status:        models.FindingPendingReview,
			Summary:       fmt.Sprintf("Host: %s\nMatched: %s\nMatcher: %s", nr.Host, nr.MatchedAt, nr.MatcherName),
			CreatedAt:     time.Now().UTC(),
			UpdatedAt:     time.Now().UTC(),
		}
		if err := p.queries.CreateFinding(f); err != nil {
			log.Printf("[pipeline] create finding %s: %v", nr.Name, err)
			continue
		}

		// Collect evidence inline
		p.collectNucleiEvidence(f.ID, nr)
	}
}

func (p *Pipeline) collectNucleiEvidence(findingID string, nr parser.NucleiResult) {
	// Save nuclei raw output as artifact
	if nr.RawLine != "" {
		artifactID := p.saveEvidenceFile(findingID, "nuclei_output.json", []byte(nr.RawLine))
		if artifactID != "" {
			p.queries.CreateEvidence(&models.Evidence{
				ID:         util.GenerateID(),
				FindingID:  findingID,
				Type:       models.EvidenceRawOutput,
				ArtifactID: &artifactID,
				Excerpt:    truncate(nr.RawLine, 500),
				CreatedBy:  "nuclei",
				CreatedAt:  time.Now().UTC(),
			})
		}
	}

	// Save request if available
	if nr.Request != "" {
		artifactID := p.saveEvidenceFile(findingID, "request.txt", []byte(nr.Request))
		if artifactID != "" {
			p.queries.CreateEvidence(&models.Evidence{
				ID:         util.GenerateID(),
				FindingID:  findingID,
				Type:       models.EvidenceRequest,
				ArtifactID: &artifactID,
				Excerpt:    truncate(nr.Request, 500),
				CreatedBy:  "nuclei",
				CreatedAt:  time.Now().UTC(),
			})
		}
	}

	// Save response if available
	if nr.Response != "" {
		artifactID := p.saveEvidenceFile(findingID, "response.txt", []byte(nr.Response))
		if artifactID != "" {
			p.queries.CreateEvidence(&models.Evidence{
				ID:         util.GenerateID(),
				FindingID:  findingID,
				Type:       models.EvidenceResponse,
				ArtifactID: &artifactID,
				Excerpt:    truncate(nr.Response, 500),
				CreatedBy:  "nuclei",
				CreatedAt:  time.Now().UTC(),
			})
		}
	}
}

func (p *Pipeline) saveEvidenceFile(findingID, filename string, data []byte) string {
	// Truncate to 1MB max
	if len(data) > 1024*1024 {
		data = data[:1024*1024]
	}

	evidenceDir := filepath.Join(p.dataDir, "evidence")
	if err := os.MkdirAll(evidenceDir, 0750); err != nil {
		log.Printf("[pipeline] create evidence dir: %v", err)
		return ""
	}

	artifactID := util.GenerateID()
	filePath := filepath.Join(evidenceDir, artifactID+"_"+filename)

	if err := os.WriteFile(filePath, data, 0640); err != nil {
		log.Printf("[pipeline] write evidence file: %v", err)
		return ""
	}

	hash := sha256.Sum256(data)
	artifact := &models.RawArtifact{
		ID:              artifactID,
		ProjectID:       p.projectID,
		Type:            models.ArtifactFile,
		Path:            filePath,
		SHA256:          fmt.Sprintf("%x", hash),
		Size:            int64(len(data)),
		RedactionStatus: "none",
		CreatedAt:       time.Now().UTC(),
	}
	if err := p.queries.CreateRawArtifact(artifact); err != nil {
		log.Printf("[pipeline] create raw artifact: %v", err)
		return ""
	}
	return artifactID
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
