package workflow

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/nuclei"
	"github.com/P0m32Kun/Anchor/internal/parser"
	"github.com/P0m32Kun/Anchor/internal/scope"
	"github.com/P0m32Kun/Anchor/internal/scoring"
	"github.com/P0m32Kun/Anchor/internal/util"
	"github.com/P0m32Kun/Anchor/internal/worker"
)

type WebScreeningWorkflow struct {
	queries *db.Queries
	runner  *worker.Runner
	scope   *scope.Engine
	scoring *scoring.ScoringEngine
	dataDir string
}

type ScreeningResult struct {
	EndpointsScanned int `json:"endpoints_scanned"`
	FindingsCreated  int `json:"findings_created"`
	FindingsUpdated  int `json:"findings_updated"`
}

func NewWebScreeningWorkflow(queries *db.Queries, runner *worker.Runner, scopeEng *scope.Engine, dataDir string) *WebScreeningWorkflow {
	return &WebScreeningWorkflow{
		queries: queries,
		runner:  runner,
		scope:   scopeEng,
		scoring: scoring.NewScoringEngine(),
		dataDir: dataDir,
	}
}

func (w *WebScreeningWorkflow) Run(ctx context.Context, projectID string) (*ScreeningResult, error) {
	result := &ScreeningResult{}

	endpoints, err := w.queries.ListWebEndpointsByProject(projectID)
	if err != nil {
		return nil, fmt.Errorf("list web endpoints: %w", err)
	}

	// Filter by scope.
	var scopedEndpoints []*models.WebEndpoint
	urlToEndpoint := make(map[string]*models.WebEndpoint)
	for _, ep := range endpoints {
		tmpTarget := &models.Target{Type: models.TargetTypeURL, Value: ep.URL}
		decision, err := w.scope.Check(ctx, projectID, tmpTarget)
		if err != nil || decision.Decision == models.ScopeDeny {
			continue
		}
		scopedEndpoints = append(scopedEndpoints, ep)
		urlToEndpoint[ep.URL] = ep
	}

	if len(scopedEndpoints) == 0 {
		return result, nil
	}

	// Group endpoints by precise tags.
	groups := nuclei.GroupEndpointsByTags(scopedEndpoints)

	if len(groups) == 0 {
		log.Printf("WebScreeningWorkflow: no fingerprinted endpoints for project %s, skipping", projectID)
		return result, nil
	}

	result.EndpointsScanned = len(scopedEndpoints)

	project, err := w.queries.GetProject(projectID)
	if err != nil {
		return nil, fmt.Errorf("get project: %w", err)
	}
	rateLimit := 0
	if project != nil {
		rateLimit = project.RateLimit
	}

	// Run one Nuclei task per tag group.
	for tagKey, urls := range groups {
		if len(urls) == 0 {
			continue
		}

		tags := strings.Split(tagKey, ",")
		targetFile, err := writeTargetsFile(w.dataDir, projectID, urls)
		if err != nil {
			log.Printf("write targets file for tags %s: %v", tagKey, err)
			continue
		}

		task, err := w.createAndRunTask(ctx, projectID, "nuclei", worker.BuildNucleiCommand(targetFile, "standard", rateLimit, 0, 0, tags, "tags", ""))
		if err != nil {
			log.Printf("nuclei task for tags %s failed: %v", tagKey, err)
			continue
		}

		nucleiResults, err := w.parseNucleiOutput(task.ID)
		if err != nil {
			log.Printf("parse nuclei output for tags %s: %v", tagKey, err)
			continue
		}

		// Process findings.
		for _, nr := range nucleiResults {
			dedupKey := computeDedupKey(nr.TemplateID, nr.Host, nr.MatcherName)

			existing, err := w.queries.GetFindingByDedupKey(projectID, dedupKey)
			if err != nil {
				continue
			}

			confidence, priority, _ := w.scoring.ScoreFinding(&nr)

			var findingID string
			if existing != nil {
				findingID = existing.ID
				result.FindingsUpdated++
				now := time.Now().UTC()
				severity := models.FindingSeverity(nr.Severity)
				if nr.Severity == "" {
					severity = existing.Severity
				}
				summary := fmt.Sprintf("Host: %s\nMatched: %s\nMatcher: %s", nr.Host, nr.MatchedAt, nr.MatcherName)
				_ = w.queries.UpdateFindingEvidence(findingID, severity, confidence, priority, summary, existing.Remediation, now)
			} else {
				var assetID, webEndpointID *string
				if ep, ok := urlToEndpoint[nr.Host]; ok {
					assetID = &ep.AssetID
					webEndpointID = &ep.ID
				}

				f := &models.Finding{
					ID:            util.GenerateID(),
					ProjectID:     projectID,
					AssetID:       assetID,
					WebEndpointID: webEndpointID,
					SourceTool:    "nuclei",
					SourceRuleID:  nr.TemplateID,
					DedupKey:      dedupKey,
					Title:         nr.Name,
					Severity:      models.FindingSeverity(nr.Severity),
					Confidence:    confidence,
					Priority:      priority,
					Status:        models.FindingPendingReview,
					Summary:       fmt.Sprintf("Host: %s\nMatched: %s\nMatcher: %s", nr.Host, nr.MatchedAt, nr.MatcherName),
					CreatedAt:     time.Now().UTC(),
					UpdatedAt:     time.Now().UTC(),
				}
				if err := w.queries.CreateFinding(f); err != nil {
					continue
				}
				findingID = f.ID
				result.FindingsCreated++
			}

			// Save request/response as evidence with sanitization.
			if nr.Request != "" || nr.Response != "" {
				workdir := filepath.Join(w.dataDir, "workdirs", projectID, task.ID)
				_ = os.MkdirAll(workdir, 0750)
				if nr.Request != "" {
					_ = w.saveEvidenceArtifact(workdir, projectID, task.ID, findingID, models.EvidenceRequest, nr.Request)
				}
				if nr.Response != "" {
					_ = w.saveEvidenceArtifact(workdir, projectID, task.ID, findingID, models.EvidenceResponse, nr.Response)
				}
			}
		}
	}

	// --- Network service scan (non-Web ports: Redis, MySQL, etc.) ---
	if err := w.runNetworkServiceScan(ctx, projectID, rateLimit, result); err != nil {
		log.Printf("network service scan for project %s: %v", projectID, err)
	}

	return result, nil
}

func (w *WebScreeningWorkflow) runNetworkServiceScan(ctx context.Context, projectID string, rateLimit int, result *ScreeningResult) error {
	assets, err := w.queries.ListAssetsByProject(projectID)
	if err != nil {
		return fmt.Errorf("list assets: %w", err)
	}

	var portTargets []nuclei.PortTarget
	hostToAssetID := make(map[string]string)

	for _, asset := range assets {
		if asset.Type != models.AssetTypeIP {
			continue
		}

		decision, err := w.scope.CheckIP(ctx, projectID, asset.Value)
		if err != nil || decision.Decision == models.ScopeDeny {
			continue
		}

		hostToAssetID[asset.Value] = asset.ID

		ports, err := w.queries.ListPortsByAsset(asset.ID)
		if err != nil {
			continue
		}
		for _, p := range ports {
			if p.State != "open" {
				continue
			}
			if tag := nuclei.MapPortToTag(p.Port); tag != "" {
				portTargets = append(portTargets, nuclei.PortTarget{
					IP:      asset.Value,
					Port:    p.Port,
					Tag:     tag,
					AssetID: asset.ID,
				})
			}
		}
	}

	if len(portTargets) == 0 {
		return nil
	}

	groups := nuclei.GroupPortsByTags(portTargets)

	for tag, targets := range groups {
		if len(targets) == 0 {
			continue
		}

		targetFile, err := writeTargetsFile(w.dataDir, projectID, targets)
		if err != nil {
			log.Printf("write network targets file for tag %s: %v", tag, err)
			continue
		}

		task, err := w.createAndRunTask(ctx, projectID, "nuclei", worker.BuildNucleiCommand(targetFile, "standard", rateLimit, 0, 0, []string{tag}, "tags", ""))
		if err != nil {
			log.Printf("nuclei network task for tag %s failed: %v", tag, err)
			continue
		}

		nucleiResults, err := w.parseNucleiOutput(task.ID)
		if err != nil {
			log.Printf("parse nuclei network output for tag %s: %v", tag, err)
			continue
		}

		for _, nr := range nucleiResults {
			dedupKey := computeDedupKey(nr.TemplateID, nr.Host, nr.MatcherName)

			existing, err := w.queries.GetFindingByDedupKey(projectID, dedupKey)
			if err != nil {
				continue
			}

			confidence, priority, _ := w.scoring.ScoreFinding(&nr)

			var findingID string
			if existing != nil {
				findingID = existing.ID
				result.FindingsUpdated++
				now := time.Now().UTC()
				severity := models.FindingSeverity(nr.Severity)
				if nr.Severity == "" {
					severity = existing.Severity
				}
				summary := fmt.Sprintf("Host: %s\nMatched: %s\nMatcher: %s", nr.Host, nr.MatchedAt, nr.MatcherName)
				_ = w.queries.UpdateFindingEvidence(findingID, severity, confidence, priority, summary, existing.Remediation, now)
			} else {
				var assetID *string
				if parts := strings.Split(nr.Host, ":"); len(parts) >= 1 {
					if id, ok := hostToAssetID[parts[0]]; ok {
						assetID = &id
					}
				}

				f := &models.Finding{
					ID:         util.GenerateID(),
					ProjectID:  projectID,
					AssetID:    assetID,
					SourceTool: "nuclei",
					SourceRuleID: nr.TemplateID,
					DedupKey:   dedupKey,
					Title:      nr.Name,
					Severity:   models.FindingSeverity(nr.Severity),
					Confidence: confidence,
					Priority:   priority,
					Status:     models.FindingPendingReview,
					Summary:    fmt.Sprintf("Host: %s\nMatched: %s\nMatcher: %s", nr.Host, nr.MatchedAt, nr.MatcherName),
					CreatedAt:  time.Now().UTC(),
					UpdatedAt:  time.Now().UTC(),
				}
				if err := w.queries.CreateFinding(f); err != nil {
					continue
				}
				findingID = f.ID
				result.FindingsCreated++
			}

			if nr.Request != "" || nr.Response != "" {
				workdir := filepath.Join(w.dataDir, "workdirs", projectID, task.ID)
				_ = os.MkdirAll(workdir, 0750)
				if nr.Request != "" {
					_ = w.saveEvidenceArtifact(workdir, projectID, task.ID, findingID, models.EvidenceRequest, nr.Request)
				}
				if nr.Response != "" {
					_ = w.saveEvidenceArtifact(workdir, projectID, task.ID, findingID, models.EvidenceResponse, nr.Response)
				}
			}
		}
	}

	return nil
}

func (w *WebScreeningWorkflow) createAndRunTask(ctx context.Context, projectID string, tool string, args []string) (*models.ScanTask, error) {
	// Create a default scan plan to satisfy FK constraint.
	plan := &models.ScanPlan{
		ID:           util.GenerateID(),
		ProjectID:    projectID,
		WorkflowType: "web-screening",
		Profile:      models.ProfileStandard,
		Status:       "approved",
		CreatedBy:    "system",
		CreatedAt:    time.Now().UTC(),
	}
	if err := w.queries.CreateScanPlan(plan); err != nil {
		return nil, fmt.Errorf("create default plan: %w", err)
	}

	taskID := util.GenerateID()
	task := &models.ScanTask{
		ID:              taskID,
		ProjectID:       projectID,
		PlanID:          plan.ID,
		Tool:            tool,
		CommandTemplate: joinArgs(args),
		Status:          models.TaskCreated,
		CreatedAt:       time.Now().UTC(),
	}
	// Record active custom bundle version for nuclei tasks
	if tool == "nuclei" {
		if version, err := w.queries.GetActiveNucleiCustomBundleVersion(); err == nil && version != "" {
			task.NucleiCustomBundleVersion = &version
		}
	}
	if err := w.queries.CreateScanTask(task); err != nil {
		return nil, err
	}
	if err := w.runner.Run(ctx, taskID); err != nil {
		return nil, err
	}
	return w.queries.GetScanTask(taskID)
}

func (w *WebScreeningWorkflow) parseNucleiOutput(taskID string) ([]parser.NucleiResult, error) {
	artifacts, err := w.queries.ListRawArtifactsByTask(taskID)
	if err != nil {
		return nil, err
	}
	for _, a := range artifacts {
		if a.Type == models.ArtifactStdout {
			f, err := os.Open(a.Path)
			if err != nil {
				continue
			}
			defer f.Close()
			results, parseErrs := parser.ParseNuclei(f)
			for _, pe := range parseErrs {
				log.Printf("nuclei parse error line %d: %s", pe.Line, pe.Message)
			}
			return results, nil
		}
	}
	return nil, nil
}

const maxEvidenceSize = 10 * 1024 * 1024 // 10 MB

func (w *WebScreeningWorkflow) saveEvidenceArtifact(workdir, projectID, taskID, findingID string, evType models.EvidenceType, data string) error {
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
	if err := w.queries.CreateRawArtifact(a); err != nil {
		return err
	}

	// Evidence excerpt uses sanitized data for safe display.
	sanitized := util.SanitizeHTTPHeaders(data)
	ev := &models.Evidence{
		ID:         util.GenerateID(),
		FindingID:  findingID,
		Type:       evType,
		ArtifactID: &a.ID,
		Excerpt:    truncateString(sanitized, 500),
		CreatedAt:  time.Now().UTC(),
	}
	return w.queries.CreateEvidence(ev)
}

func computeDedupKey(templateID, host, matcherName string) string {
	h := sha256.New()
	fmt.Fprintf(h, "%s|%s|%s", templateID, host, matcherName)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func writeTargetsFile(dataDir, projectID string, urls []string) (string, error) {
	workdir := filepath.Join(dataDir, "workdirs", projectID, "screening")
	if err := os.MkdirAll(workdir, 0750); err != nil {
		return "", err
	}
	path := filepath.Join(workdir, "targets.txt")
	if err := os.WriteFile(path, []byte(strings.Join(urls, "\n")+"\n"), 0640); err != nil {
		return "", err
	}
	return path, nil
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
