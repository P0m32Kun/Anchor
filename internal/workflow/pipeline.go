package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/P0m32Kun/Anchor/internal/asset"
	"github.com/P0m32Kun/Anchor/internal/cdn"
	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/fingerprint"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/resolve"
	"github.com/P0m32Kun/Anchor/internal/scope"
	"github.com/P0m32Kun/Anchor/internal/search"
	"github.com/P0m32Kun/Anchor/internal/worker"
)

// DefaultWorkflowDir is empty because custom templates/workflows are injected
// at execution time by the worker (from the active custom bundle) rather than
// hard-coded into the Docker image.
const DefaultWorkflowDir = ""

// Pipeline orchestrates the complete scan workflow.
type Pipeline struct {
	queries       *db.Queries
	runner        *worker.Runner
	scope         *scope.Engine
	resolver      *resolve.Resolver
	cdnDet        *cdn.Detector
	fofa          *search.FofaClient
	merger        *asset.Merger
	dataDir       string
	projectID     string
	config        models.PipelineConfig
	runID         string
	onStageChange StageEventCallback
}

// NewPipeline creates a new Pipeline instance.
func NewPipeline(queries *db.Queries, runner *worker.Runner, scopeEng *scope.Engine, dataDir string) *Pipeline {
	return &Pipeline{
		queries:  queries,
		runner:   runner,
		scope:    scopeEng,
		resolver: resolve.NewResolver(),
		cdnDet:   cdn.NewDetector(),
		merger:   asset.NewMerger(queries),
		dataDir:  dataDir,
	}
}

// WithConfig sets the pipeline configuration.
func (p *Pipeline) WithConfig(cfg models.PipelineConfig) *Pipeline {
	p.config = cfg
	return p
}

// WithFOFA sets the FOFA client.
func (p *Pipeline) WithFOFA(apiKey string) *Pipeline {
	if apiKey != "" {
		p.fofa = search.NewFofaClient(apiKey)
	}
	return p
}

func (p *Pipeline) WithRunID(runID string) *Pipeline {
	p.runID = runID
	return p
}

func (p *Pipeline) WithStageCallback(cb StageEventCallback) *Pipeline {
	p.onStageChange = cb
	return p
}

var requiredTools = []string{"subfinder", "naabu", "cdncheck", "httpx", "nuclei", "dnsx", "nmap"}

func (p *Pipeline) checkTools() []string {
	var missing []string
	for _, tool := range requiredTools {
		if _, err := exec.LookPath(tool); err != nil {
			missing = append(missing, tool)
		}
	}
	return missing
}

// Run executes the pipeline for a project.
func (p *Pipeline) Run(ctx context.Context, projectID string) error {
	p.projectID = projectID

	// Load project config if not set
	if p.config == (models.PipelineConfig{}) {
		project, err := p.queries.GetProject(projectID)
		if err != nil {
			return fmt.Errorf("get project: %w", err)
		}
		if project != nil && project.PipelineConfig != nil && *project.PipelineConfig != "" {
			if err := json.Unmarshal([]byte(*project.PipelineConfig), &p.config); err != nil {
				log.Printf("[pipeline] unmarshal pipeline config: %v", err)
				p.config = models.DefaultPipelineConfig()
			}
		} else {
			p.config = models.DefaultPipelineConfig()
		}
	}

	// Initialize FOFA if enabled and not already set
	if p.config.EnableFOFA && p.fofa == nil {
		cred, err := p.queries.GetEngineCredential("fofa")
		if err == nil && cred != nil && cred.APIKey != "" {
			p.fofa = search.NewFofaClient(cred.APIKey)
		}
	}

	// Check required tools (non-blocking: tools run on workers, not server)
	if missing := p.checkTools(); len(missing) > 0 {
		log.Printf("[pipeline] warning: required tools not found on server (OK if workers have them): %s", strings.Join(missing, ", "))
	}

	// Get all targets
	targets, err := p.queries.ListTargetsByProject(projectID)
	if err != nil {
		if p.runID != "" {
			p.queries.UpdatePipelineRunError(p.runID, err.Error())
			p.queries.UpdatePipelineRunStatus(p.runID, "failed")
		}
		return fmt.Errorf("list targets: %w", err)
	}

	// Group by type
	groups := groupTargetsByType(targets)

	// Execute flows for each target type
	var flowErr error
	for _, group := range groups {
		if err := p.runFlow(ctx, group); err != nil {
			log.Printf("pipeline flow error for type %s: %v", group.Type, err)
			flowErr = err
		}
	}

	if p.runID != "" {
		now := time.Now().UTC()

		// Check if any stage failed.
		stages, _ := p.queries.ListPipelineRunStages(p.runID)
		hasFailedStage := false
		for _, s := range stages {
			if s.Status == models.StageStatusFailed {
				hasFailedStage = true
				break
			}
		}

		var errMsg string
		if flowErr != nil {
			errMsg = flowErr.Error()
		} else if hasFailedStage {
			errMsg = "one or more stages failed"
		}

		if flowErr != nil || hasFailedStage {
			if errMsg != "" {
				p.queries.UpdatePipelineRunError(p.runID, errMsg)
			}
			p.queries.UpdatePipelineRunStatus(p.runID, "failed")
		} else {
			p.queries.UpdatePipelineRunCompleted(p.runID, now)
		}

		// Mark any still-running stages as completed (or failed if overall failed).
		for _, s := range stages {
			if s.Status == models.StageStatusRunning {
				if flowErr != nil || hasFailedStage {
					p.queries.UpdatePipelineRunStageRecord(s.ID, models.StageStatusFailed, "pipeline aborted", &now)
				} else {
					p.queries.UpdatePipelineRunStageRecord(s.ID, models.StageStatusCompleted, "", &now)
				}
			}
		}
	}

	return flowErr
}

// --- Utility functions ---

func extractTargetValues(targets []*models.Target) []string {
	var vals []string
	for _, t := range targets {
		vals = append(vals, t.Value)
	}
	return vals
}

func makeTargets(values []string, typ models.TargetType) []*models.Target {
	var targets []*models.Target
	for _, v := range values {
		targets = append(targets, &models.Target{Type: typ, Value: v})
	}
	return targets
}

func dedupStrings(s []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, v := range s {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	return result
}

func makeHTTPTargets(results []fingerprint.NervaResult) []string {
	var targets []string
	for _, r := range results {
		host := r.IP
		if host == "" {
			host = r.Host
		}
		if r.Port == 443 {
			targets = append(targets, fmt.Sprintf("https://%s", host))
		} else {
			targets = append(targets, fmt.Sprintf("http://%s:%d", host, r.Port))
		}
	}
	return targets
}
