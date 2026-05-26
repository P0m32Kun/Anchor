package workflow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/P0m32Kun/Anchor/internal/asset"
	"github.com/P0m32Kun/Anchor/internal/cdn"
	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/fingerprint"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/nuclei"
	"github.com/P0m32Kun/Anchor/internal/parser"
	"github.com/P0m32Kun/Anchor/internal/resolve"
	"github.com/P0m32Kun/Anchor/internal/scope"
	"github.com/P0m32Kun/Anchor/internal/search"
	"github.com/P0m32Kun/Anchor/internal/toolregistry"
	"github.com/P0m32Kun/Anchor/internal/toolrun"
	"github.com/P0m32Kun/Anchor/internal/util"
	"github.com/P0m32Kun/Anchor/internal/worker"
)

// DefaultWorkflowDir is empty because custom templates/workflows are injected
// at execution time by the worker (from the active custom bundle) rather than
// hard-coded into the Docker image. The worker's injectCustomNucleiTemplates
// layers -t (official + custom templates) and -w (custom workflows) onto the
// command depending on scanDepth.
const DefaultWorkflowDir = ""

// Pipeline orchestrates the complete scan workflow.
type Pipeline struct {
	queries          *db.Queries
	runner           *worker.Runner
	scope            *scope.Engine
	resolver         *resolve.Resolver
	cdnDet           *cdn.Detector
	fofa             *search.FofaClient
	merger           *asset.Merger
	dataDir          string
	projectID        string
	config           models.PipelineConfig
	runID            string
	onStageChange    StageEventCallback
	emitter          *StageEmitter
	findingBuf       *db.FindingBuffer
	seenDedupKeys    map[string]bool
	deferredEvidence []deferredEvidence
	tools            *toolregistry.Registry
}

// deferredEvidence holds nuclei evidence data to be created after findings are flushed.
type deferredEvidence struct {
	findingID string
	nr        parser.NucleiResult
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

// WithTools sets the tool registry for argv generation and allowlist enforcement.
func (p *Pipeline) WithTools(reg *toolregistry.Registry) *Pipeline {
	p.tools = reg
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

// flushFindingsAndEvidence flushes the finding buffer and creates deferred evidence.
// Called via defer in Run() and explicitly at pipeline boundaries.
func (p *Pipeline) flushFindingsAndEvidence() {
	if p.findingBuf != nil {
		if err := p.findingBuf.Flush(); err != nil {
			log.Printf("[pipeline] flush findings: %v", err)
		}
	}
	for _, de := range p.deferredEvidence {
		p.collectNucleiEvidence(de.findingID, de.nr)
	}
	p.deferredEvidence = nil
}

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
	p.findingBuf = db.NewFindingBuffer(p.queries, 500, 2*time.Second)
	p.seenDedupKeys = make(map[string]bool)
	p.deferredEvidence = nil
	defer p.flushFindingsAndEvidence()

	// Register shutdown hook to flush buffer on SIGTERM/SIGINT.
	buf := p.findingBuf
	util.OnShutdown(func() error {
		if buf != nil {
			return buf.Flush()
		}
		return nil
	})

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

	// Post-phase: Katana crawl + ffuf against first-pass
	// web endpoints, then feed any new URLs through a second httpx → nuclei
	// round. This runs regardless of whether the main flow had failures —
	// scanning should be as complete as possible. The run status is determined
	// by ALL stages equally after everything finishes.
	p.runPostPhase(ctx, projectID)

	// Final status settlement: query all stages (including ffuf/crawl/
	// httpx_2/vuln_2) and determine run outcome. Any failed stage fails
	// the run, including the second pass.
	if p.runID != "" {
		now := time.Now().UTC()
		stages, _ := p.queries.ListPipelineRunStages(p.runID)
		for _, s := range stages {
			if s.Status == models.StageStatusRunning {
				if flowErr != nil {
					p.queries.UpdatePipelineRunStageRecord(s.ID, models.StageStatusFailed, "pipeline aborted", &now)
				} else {
					p.queries.UpdatePipelineRunStageRecord(s.ID, models.StageStatusCompleted, "", &now)
				}
			}
		}
		// Re-query after cleanup for accurate status
		stages, _ = p.queries.ListPipelineRunStages(p.runID)
		hasFailed := flowErr != nil
		for _, s := range stages {
			if s.Status == models.StageStatusFailed || s.Status == "failed" {
				hasFailed = true
				break
			}
		}
		if hasFailed {
			var errMsg string
			if flowErr != nil {
				errMsg = flowErr.Error()
			} else {
				errMsg = "one or more stages failed"
			}
			p.queries.UpdatePipelineRunError(p.runID, errMsg)
			p.queries.UpdatePipelineRunStatus(p.runID, "failed")
		} else {
			p.queries.UpdatePipelineRunCompleted(p.runID, now)
		}
	}

	return flowErr
}

// runPostPhase runs Katana (optional) and ffuf (optional) against first-pass web
// endpoints, then feeds newly discovered URLs through httpx_2 → vuln_2.
func (p *Pipeline) runPostPhase(ctx context.Context, projectID string) {
	endpoints, err := p.queries.ListWebEndpointsByProject(projectID)
	if err != nil || len(endpoints) == 0 {
		return
	}

	var discoveredURLs []string
	var mu sync.Mutex

	if p.config.EnableKatana {
		allURLs := make([]string, len(endpoints))
		for i, ep := range endpoints {
			allURLs[i] = ep.URL
		}
		crawlURLs, err := p.runKatana(ctx, allURLs)
		if err != nil {
			log.Printf("[pipeline] katana: %v", err)
		} else if len(crawlURLs) > 0 {
			mu.Lock()
			discoveredURLs = append(discoveredURLs, crawlURLs...)
			mu.Unlock()
		}
	}

	wantFfuf := p.ffufTierActive()
	ffufDictID := p.resolveFfufDictionaryID()
	if wantFfuf && ffufDictID == "" {
		wantFfuf = false
	}

	if wantFfuf {
		p.setStage(StageFfuf)
		var wg sync.WaitGroup
		var ffufFailures, ffufAttempts int
		const maxFfufConcurrency = 5
		ffufSem := make(chan struct{}, maxFfufConcurrency)

		for _, ep := range endpoints {
			if !p.shouldFfufEndpoint(ep) {
				continue
			}
			wg.Add(1)
			ffufAttempts++
			ffufSem <- struct{}{}
			go func(endpoint *models.WebEndpoint) {
				defer wg.Done()
				defer func() { <-ffufSem }()
				urls, err := p.runFfuf(ctx, endpoint, ffufDictID)
				if err != nil {
					log.Printf("[pipeline] ffuf %s: %v", endpoint.URL, err)
					mu.Lock()
					ffufFailures++
					mu.Unlock()
					return
				}
				mu.Lock()
				discoveredURLs = append(discoveredURLs, urls...)
				mu.Unlock()
			}(ep)
		}

		wg.Wait()
		if ffufFailures > 0 {
			p.failStage(StageFfuf, fmt.Sprintf("%d/%d endpoints failed", ffufFailures, ffufAttempts))
		} else {
			p.completeStage(StageFfuf)
		}
	}

	if len(discoveredURLs) > 0 {
		unique := dedupStrings(discoveredURLs)
		p.feedToHttpxNuclei(ctx, unique, endpoints)
	}
}

// feedToHttpxNuclei runs httpx (with custom fingerprints) on discovered URLs,
// saves new WebEndpoints, then runs nuclei and persists findings.
func (p *Pipeline) feedToHttpxNuclei(ctx context.Context, urls []string, existingEndpoints []*models.WebEndpoint) {
	if len(urls) == 0 {
		return
	}

	newURLs := filterURLsForSecondaryScan(urls, existingEndpoints)
	if len(newURLs) == 0 {
		log.Printf("[pipeline] all %d discovered URLs already covered (url or origin), skipping", len(urls))
		return
	}
	log.Printf("[pipeline] feeding %d new URLs to httpx/nuclei (deduped from %d)", len(newURLs), len(urls))

	// Write host file
	hostFile := filepath.Join(p.dataDir, "workdirs", p.projectID, fmt.Sprintf("httpx2-%s.txt", util.GenerateID()))
	if err := os.MkdirAll(filepath.Dir(hostFile), 0750); err != nil {
		return
	}
	if err := os.WriteFile(hostFile, []byte(strings.Join(newURLs, "\n")), 0640); err != nil {
		return
	}

	// Prepare custom fingerprint file
	customFpFile, err := p.prepareHttpxFingerprints()
	if err != nil {
		log.Printf("[pipeline] prepare httpx fingerprints: %v", err)
	}
	if customFpFile != "" {
		defer os.Remove(customFpFile)
	}

	// --- httpx_2 stage ---
	p.setStage(StageHTTPX2)
	resHT := toolrun.Invoke(ctx, p.queries, p.runner, p.tools, toolrun.InvokeInput{
		ProjectID: p.projectID,
		RunID:     &p.runID,
		ToolID:    "httpx",
		Params: toolregistry.RenderParams{
			"host_file":      hostFile,
			"rate_limit":     p.config.HttpxRateLimit,
			"threads":        p.config.HttpxThreads,
			"custom_fp_file": customFpFile,
		},
	})
	if resHT.Err != nil {
		p.failStage(StageHTTPX2, resHT.Err.Error())
		return
	}

	endpoints := parser.ParseHttpxOutput(bytes.NewReader(resHT.Stdout))
	var savedEndpoints []*models.WebEndpoint
	for _, ep := range endpoints {
		host := ep.Host
		if host == "" {
			continue
		}
		assetType := "domain"
		if net.ParseIP(host) != nil {
			assetType = "ip"
		}
		hostAsset, _, err := p.merger.MergeOrCreateAsset(p.projectID, assetType, host, "httpx")
		if err != nil {
			log.Printf("[pipeline] merge/create asset %s: %v", host, err)
			continue
		}
		we, _, err := p.merger.CreateWebEndpointIfNotExists(
			p.projectID, hostAsset.ID, ep.URL, ep.Scheme, ep.Host,
			ep.Port, ep.Path, ep.Title, ep.StatusCode, ep.Technologies, "httpx",
		)
		if err != nil {
			log.Printf("[pipeline] save endpoint %s: %v", ep.URL, err)
			continue
		}
		if we != nil {
			savedEndpoints = append(savedEndpoints, we)
		}
	}
	p.completeStage(StageHTTPX2)
	log.Printf("[pipeline] httpx_2: saved %d new web endpoints", len(savedEndpoints))

	if len(savedEndpoints) == 0 {
		return
	}

	// --- vuln_2 stage ---
	p.setStage(StageVuln2)

	urlToEndpoint := make(map[string]*models.WebEndpoint)
	for _, ep := range savedEndpoints {
		urlToEndpoint[ep.URL] = ep
	}

	groups := make(map[string][]string)
	for _, ep := range savedEndpoints {
		key := strings.Join(nuclei.MapPreciseTags(ep.Technologies, ""), ",")
		if key == "" {
			key = "generic"
		}
		groups[key] = append(groups[key], ep.URL)
	}

	var vulnErr error
	for tagKey, urls := range groups {
		tags := strings.Split(tagKey, ",")
		targetFile := filepath.Join(p.dataDir, "workdirs", p.projectID, fmt.Sprintf("nuclei2-%s.txt", util.GenerateID()))
		scanURLs := dedupHTTPTargetsByOrigin(dedupStrings(urls))
		if err := os.WriteFile(targetFile, []byte(strings.Join(scanURLs, "\n")), 0640); err != nil {
			continue
		}

		resNU := toolrun.Invoke(ctx, p.queries, p.runner, p.tools, toolrun.InvokeInput{
			ProjectID: p.projectID,
			RunID:     &p.runID,
			ToolID:    "nuclei",
			Params: toolregistry.RenderParams{
				"target_file":       targetFile,
				"profile":           "deep",
				"tags":              tags,
				"scan_depth":        p.config.NucleiScanDepth,
				"concurrency":       p.config.NucleiConcurrency,
				"rate_limit":        p.config.NucleiRateLimit,
				"rate_limit_per_min": p.config.NucleiRateLimitPerMinute,
			},
		})
		if resNU.Err != nil {
			log.Printf("[pipeline] vuln_2 nuclei error for tags %s: %v", tagKey, resNU.Err)
			vulnErr = resNU.Err
			continue
		}
		p.saveNucleiFindings(resNU.Stdout, urlToEndpoint, nil)
	}

	// Flush findings before creating evidence (FK constraint).
	p.flushFindingsAndEvidence()

	if vulnErr != nil {
		p.failStage(StageVuln2, vulnErr.Error())
	} else {
		p.completeStage(StageVuln2)
	}
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
