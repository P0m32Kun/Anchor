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

// BuiltinWorkflowDir is the path to the official nuclei-templates workflows
// directory on the worker. Used by the pipeline to pass -w for "workflow" and
// "both" scan depths. Custom bundles are layered on top by the worker at
// execution time via injectCustomNucleiTemplates.
const BuiltinWorkflowDir = "/root/nuclei-templates/workflows"

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
	emitter       *StageEmitter
}

// NewPipeline creates a new Pipeline instance.
func NewPipeline(queries *db.Queries, runner *worker.Runner, scopeEng *scope.Engine, dataDir string) *Pipeline {
	p := &Pipeline{
		queries:  queries,
		runner:   runner,
		scope:    scopeEng,
		resolver: resolve.NewResolver(),
		cdnDet:   cdn.NewDetector(),
		merger:   asset.NewMerger(queries),
		dataDir:  dataDir,
	}
	// Start with an empty-runID emitter so setStage/completeStage/failStage
	// are safely no-ops until WithRunID is called.
	p.emitter = NewStageEmitter(queries, "", nil)
	return p
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
	p.emitter = NewStageEmitter(p.queries, runID, p.onStageChange)
	return p
}

func (p *Pipeline) WithStageCallback(cb StageEventCallback) *Pipeline {
	p.onStageChange = cb
	p.emitter = NewStageEmitter(p.queries, p.runID, cb)
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

	// Scope filter: single enforcement point.
	//   * CIDR targets are expanded to atomic IP targets (with a /16 cap to
	//     prevent OOM on absurdly wide inputs like 0.0.0.0/0).
	//   * Every IP/Domain/URL target is evaluated against the project's
	//     scope rules; denied targets are dropped here so downstream tools
	//     (nmap / naabu / httpx / nuclei) can stay scope-unaware.
	//   * Company targets pass through — FOFA expansion runs scope per
	//     derived target later in runCompanyFlow.
	filtered, scopeErr := p.scope.FilterTargets(ctx, projectID, targets)
	if scopeErr != nil {
		if p.runID != "" {
			p.queries.UpdatePipelineRunError(p.runID, scopeErr.Error())
			p.queries.UpdatePipelineRunStatus(p.runID, "failed")
		}
		return fmt.Errorf("scope filter: %w", scopeErr)
	}
	if len(filtered) == 0 && len(targets) > 0 {
		// Every configured target was rejected — surface this explicitly so
		// the user sees the cause in the UI instead of a quiet 0-finding run.
		msg := fmt.Sprintf("all %d targets denied by scope rules (no work to do)", len(targets))
		log.Printf("[pipeline] %s", msg)
		if p.runID != "" {
			p.queries.UpdatePipelineRunError(p.runID, msg)
			p.queries.UpdatePipelineRunStatus(p.runID, "failed")
		}
		return fmt.Errorf("%s", msg)
	}
	log.Printf("[pipeline] scope filter: %d targets in, %d after filter (project %s)", len(targets), len(filtered), projectID)
	targets = filtered

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

	// Pipeline lifecycle has two phases:
	//   1. Main flow stages (alive/portscan/.../vuln) just finished.
	//   2. Slow scan stages (ffuf) — part of the same pipeline_run lifecycle,
	//      not a fire-and-forget background task.
	// We DO NOT set pipeline_run.status to completed until phase 2 also
	// finishes, so consumers reading "status=completed" can trust the run is
	// fully done. The handleCreateScan handler already wraps Pipeline.Run in
	// a goroutine, so the longer wall-clock time here only blocks that
	// background goroutine, not the API response.

	// Phase 1 cleanup: finalize any main-flow stages still marked running and
	// decide whether the main flow itself succeeded. Slow-scan stages haven't
	// been emitted yet, so this check is purely about main pipeline health.
	var mainHasFailed bool
	if p.runID != "" {
		now := time.Now().UTC()
		stages, _ := p.queries.ListPipelineRunStages(p.runID)
		for _, s := range stages {
			if isSlowScanStage(s.Stage) {
				continue
			}
			if s.Status == models.StageStatusFailed {
				mainHasFailed = true
			}
			if s.Status == models.StageStatusRunning {
				// A stage left in "running" after the main flow returned means
				// the flow itself exited (panic, early return) without cleanup.
				// Mark it failed/completed in line with the overall outcome.
				if flowErr != nil {
					p.queries.UpdatePipelineRunStageRecord(s.ID, models.StageStatusFailed, "pipeline aborted", &now)
					mainHasFailed = true
				} else {
					p.queries.UpdatePipelineRunStageRecord(s.ID, models.StageStatusCompleted, "", &now)
				}
			}
		}
	}

	// Slow scan trigger: main flow must be clean. If any main-flow stage failed
	// (or the flow returned an error) we skip slow scan entirely — partial
	// data isn't worth burning brute-force budget on, and the previous
	// behavior of running slow scan after a half-broken pipeline produced
	// confusing UI (post-fail ffuf rows showing up). This resolves the
	// historical TODO around slow-scan trigger gating.
	shouldRunSlowScan := flowErr == nil && !mainHasFailed
	if shouldRunSlowScan {
		slowScan := NewSlowScanOrchestrator(p.queries, p.runner, p.dataDir).
			WithConfig(p.config).
			WithMerger(p.merger).
			WithStageEmitter(p.emitter)
		if err := slowScan.Run(ctx, p.projectID, p.runID); err != nil {
			log.Printf("[pipeline] slow scan orchestrator: %v", err)
		}
	}

	// Phase 2 final settlement: now that everything (main + slow scan) is
	// done, write the run's terminal status. Slow scan stage failures
	// (e.g. ffuf brute-force hit no valid paths against an internal target)
	// are expected in some target types and DO NOT mark the run as failed —
	// they show as a failed stage in the UI but the run as a whole still
	// counts as completed. Only main-flow failures fail the run.
	if p.runID != "" {
		now := time.Now().UTC()
		var errMsg string
		if flowErr != nil {
			errMsg = flowErr.Error()
		} else if mainHasFailed {
			errMsg = "one or more stages failed"
		}

		if flowErr != nil || mainHasFailed {
			if errMsg != "" {
				p.queries.UpdatePipelineRunError(p.runID, errMsg)
			}
			p.queries.UpdatePipelineRunStatus(p.runID, "failed")
		} else {
			p.queries.UpdatePipelineRunCompleted(p.runID, now)
		}
	}

	return flowErr
}

// isSlowScanStage reports whether a stage name belongs to the post-pipeline
// slow scan phase. Slow-scan stages are tracked in pipeline_run_stages alongside
// main flow stages but are treated as soft — their failure does not promote
// the run to failed. Adding a new slow-scan tool means adding its stage here.
func isSlowScanStage(stage string) bool {
	return stage == string(StageFfuf) || stage == string(StageURLFinder)
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

func makeHTTPTargets(results []fingerprint.NmapServiceResult) []string {
	return fingerprint.MakeHTTPTargets(results)
}
