package finding

import (
	"crypto/sha256"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/parser"
	"github.com/P0m32Kun/Anchor/internal/scoring"
	"github.com/P0m32Kun/Anchor/internal/util"
)

const maxEvidenceSize = 10 * 1024 * 1024

// NucleiPersister saves Nuclei JSONL results as Findings and Evidence artifacts.
type NucleiPersister struct {
	queries *db.Queries
	scorer  *scoring.ScoringEngine
	dataDir string
}

func NewNucleiPersister(queries *db.Queries, dataDir string) *NucleiPersister {
	return &NucleiPersister{
		queries: queries,
		scorer:  scoring.NewScoringEngine(),
		dataDir: dataDir,
	}
}

// Persist parses stdout and upserts findings for the given run/task.
func (p *NucleiPersister) Persist(projectID, runID, taskID, assetID string, stdout []byte) (created, updated int, err error) {
	results, parseErrs := parser.ParseNuclei(strings.NewReader(string(stdout)))
	for _, pe := range parseErrs {
		_ = pe
	}
	workdir := filepath.Join(p.dataDir, "workdirs", projectID, taskID)
	for _, nr := range results {
		if nr.TemplateID == "" {
			continue
		}
		dedupKey := DedupKey(nr.TemplateID, nr.Host, nr.MatcherName)
		existing, err := p.queries.GetFindingByDedupKey(projectID, dedupKey)
		if err != nil {
			continue
		}
		confidence, priority, _ := p.scorer.ScoreFinding(&nr)
		var findingID string
		if existing != nil {
			findingID = existing.ID
			updated++
			now := time.Now().UTC()
			severity := models.FindingSeverity(nr.Severity)
			if nr.Severity == "" {
				severity = existing.Severity
			}
			summary := fmt.Sprintf("Host: %s\nMatched: %s\nMatcher: %s", nr.Host, nr.MatchedAt, nr.MatcherName)
			_ = p.queries.UpdateFindingEvidence(findingID, severity, confidence, priority, summary, existing.Remediation, now)
		} else {
			var aid, runIDPtr *string
			if assetID != "" {
				aid = &assetID
			}
			if runID != "" {
				runIDPtr = &runID
			}
			f := &models.Finding{
				ID:             util.GenerateID(),
				ProjectID:      projectID,
				RunID:          runIDPtr,
				AssetID:        aid,
				SourceTool:     "nuclei",
				SourceRuleID:   nr.TemplateID,
				DedupKey:       dedupKey,
				Title:          nr.Name,
				Severity:       models.FindingSeverity(nr.Severity),
				Confidence:     confidence,
				Priority:       priority,
				Status:         models.FindingPendingReview,
				Summary:        fmt.Sprintf("Host: %s\nMatched: %s\nMatcher: %s", nr.Host, nr.MatchedAt, nr.MatcherName),
				MatchedTemplate: nr.TemplateID,
				CreatedAt:      time.Now().UTC(),
				UpdatedAt:      time.Now().UTC(),
			}
			if err := p.queries.CreateFinding(f); err != nil {
				continue
			}
			findingID = f.ID
			created++
		}
		if nr.Request != "" || nr.Response != "" {
			_ = os.MkdirAll(workdir, 0750)
			if nr.Request != "" {
				_ = p.saveEvidence(workdir, projectID, taskID, findingID, models.EvidenceRequest, nr.Request)
			}
			if nr.Response != "" {
				_ = p.saveEvidence(workdir, projectID, taskID, findingID, models.EvidenceResponse, nr.Response)
			}
		}
	}
	return created, updated, nil
}

func (p *NucleiPersister) saveEvidence(workdir, projectID, taskID, findingID string, evType models.EvidenceType, data string) error {
	if len(data) > maxEvidenceSize {
		data = data[:maxEvidenceSize] + "\n... [truncated]"
	}
	filename := fmt.Sprintf("%s_%s_%d.txt", findingID, evType, time.Now().UnixNano())
	path := filepath.Join(workdir, filename)
	if err := os.WriteFile(path, []byte(data), 0640); err != nil {
		return err
	}
	sum := sha256.Sum256([]byte(data))
	a := &models.RawArtifact{
		ID:              util.GenerateID(),
		ProjectID:       projectID,
		TaskID:          &taskID,
		Type:            models.ArtifactRequest,
		Path:            path,
		SHA256:          fmt.Sprintf("%x", sum),
		Size:            int64(len(data)),
		RedactionStatus: "raw",
		CreatedAt:       time.Now().UTC(),
	}
	if evType == models.EvidenceResponse {
		a.Type = models.ArtifactResponse
	}
	if err := p.queries.CreateRawArtifact(a); err != nil {
		return err
	}
	sanitized := util.SanitizeHTTPHeaders(data)
	excerpt := sanitized
	if len(excerpt) > 500 {
		excerpt = excerpt[:500]
	}
	ev := &models.Evidence{
		ID:         util.GenerateID(),
		FindingID:  findingID,
		Type:       evType,
		ArtifactID: &a.ID,
		Excerpt:    excerpt,
		CreatedAt:  time.Now().UTC(),
	}
	return p.queries.CreateEvidence(ev)
}

// DedupKey hashes template + scan origin + matcher (same semantics as legacy workflow).
func DedupKey(templateID, host, matcherName string) string {
	origin := scanOrigin(host)
	h := sha256.New()
	fmt.Fprintf(h, "%s|%s|%s", templateID, origin, matcherName)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func scanOrigin(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.Contains(raw, "://") {
		u, err := url.Parse(raw)
		if err == nil && u.Host != "" {
			host := u.Hostname()
			port := u.Port()
			if port == "" {
				switch strings.ToLower(u.Scheme) {
				case "http":
					port = "80"
				case "https":
					port = "443"
				}
			}
			if port != "" {
				return strings.ToLower(net.JoinHostPort(host, port))
			}
			return strings.ToLower(u.Host)
		}
	}
	if idx := strings.Index(raw, "/"); idx > 0 {
		raw = raw[:idx]
	}
	host, port, err := net.SplitHostPort(raw)
	if err != nil {
		return strings.ToLower(raw)
	}
	return strings.ToLower(net.JoinHostPort(host, port))
}
