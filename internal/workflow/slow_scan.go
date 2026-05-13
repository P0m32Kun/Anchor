package workflow

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/nuclei"
	"github.com/P0m32Kun/Anchor/internal/parser"
	"github.com/P0m32Kun/Anchor/internal/util"
	"github.com/P0m32Kun/Anchor/internal/worker"
)

// SlowScanOrchestrator manages background slow scanning tasks (currently
// just ffuf — urlfinder was removed 2026-05-13 after its `-json` flag bug
// was discovered; the OSINT use case never fit internal-scan targets anyway).
// Slow scan runs after the main pipeline completes and feeds discovered URLs
// back into the httpx -> nuclei loop.
//
// Stage reporting: when an emitter is wired in via WithStageEmitter, the
// orchestrator reports ffuf as a first-class pipeline stage so the UI shows
// running/completed/failed under the same SSE channel as the main pipeline
// stages. The stage is reported once (one running event, one terminal event)
// regardless of how many endpoints fan out — individual endpoint results are
// aggregated into the terminal state, not exposed as N child stages.
type SlowScanOrchestrator struct {
	queries   *db.Queries
	runner    *worker.Runner
	dataDir   string
	config    models.PipelineConfig
	projectID string
	emitter   *StageEmitter
	// sem controls concurrency — max N slow scan tasks running at once.
	sem chan struct{}
}

// NewSlowScanOrchestrator creates a new orchestrator with default concurrency limit.
func NewSlowScanOrchestrator(queries *db.Queries, runner *worker.Runner, dataDir string) *SlowScanOrchestrator {
	return &SlowScanOrchestrator{
		queries: queries,
		runner:  runner,
		dataDir: dataDir,
		emitter: NewStageEmitter(queries, "", nil), // no-op until WithStageEmitter
		sem:     make(chan struct{}, 5),            // max 5 concurrent slow scans
	}
}

func (s *SlowScanOrchestrator) WithConfig(cfg models.PipelineConfig) *SlowScanOrchestrator {
	s.config = cfg
	return s
}

// WithStageEmitter wires in the same StageEmitter the main Pipeline uses, so
// slow scan stages flow through the same DB table and SSE callback. Pass
// p.emitter from Pipeline. Passing a nil or empty-runID emitter keeps stage
// reporting silently disabled (matches NewSlowScanOrchestrator default).
func (s *SlowScanOrchestrator) WithStageEmitter(e *StageEmitter) *SlowScanOrchestrator {
	if e != nil {
		s.emitter = e
	}
	return s
}

// Run consumes web endpoints discovered by the main pipeline (httpx) and runs
// ffuf against them. Works for any target type (domain, URL, IP, CIDR) — what
// matters is that httpx exposed a live HTTP service, not how the target was
// originally specified. ffuf is run per endpoint URL since it brute-forces
// paths under a specific scheme+host+port.
//
// Stage lifecycle for the UI:
//   - ffuf: one Set(StageFfuf) before fan-out, then Complete or Fail once
//     every goroutine returns. A panic in any goroutine is recovered and
//     counted as a failure for that endpoint so the stage still terminates.
//     If the config enables ffuf without a dictionary (Fix 3 path), we Fail
//     StageFfuf inline without ever running the goroutines.
//   - A defer at the top of Run() catches a panic in the orchestrator itself
//     and marks any not-yet-terminated stages as failed so the UI never sees a
//     stage stuck on running.
func (s *SlowScanOrchestrator) Run(ctx context.Context, projectID string, runID string) error {
	s.projectID = projectID

	// Track which stages were started (Set called) but not yet terminated
	// (Complete/Fail called). If the orchestrator panics, the top-level defer
	// uses this set to fail every dangling stage so the UI doesn't hang on
	// "running" forever.
	startedStages := make(map[StageID]bool)
	var stagesMu sync.Mutex
	markStarted := func(stage StageID) {
		stagesMu.Lock()
		startedStages[stage] = true
		stagesMu.Unlock()
	}
	markTerminated := func(stage StageID) {
		stagesMu.Lock()
		delete(startedStages, stage)
		stagesMu.Unlock()
	}
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[slow-scan] panic in Run: %v", r)
			stagesMu.Lock()
			for stage := range startedStages {
				s.emitter.Fail(stage, fmt.Sprintf("slow scan panic: %v", r))
			}
			stagesMu.Unlock()
		}
	}()

	endpoints, err := s.queries.ListWebEndpointsByProject(projectID)
	if err != nil {
		return fmt.Errorf("list web endpoints: %w", err)
	}

	// Decide what to run BEFORE touching stages, so we never emit a stage for
	// work that won't actually fan out. If ffuf is enabled but no dictionary
	// is selected, emit a failed stage so the UI shows the misconfiguration
	// even when a caller bypassed the frontend guard (e.g. API direct call).
	var wantFfuf bool
	if s.config.EnableFfuf {
		if s.config.FfufDictionaryID == "" {
			s.emitter.Fail(StageFfuf, "ffuf enabled but no dictionary configured")
		} else {
			wantFfuf = true
		}
	}

	if len(endpoints) == 0 {
		log.Printf("[slow-scan] no web endpoints discovered, skipping ffuf")
		return nil
	}

	// Per-stage aggregation state so we can emit one terminal event per stage
	// regardless of how many endpoint goroutines ran.
	type aggregate struct {
		mu       sync.Mutex
		attempts int
		failures int
		errs     []string
	}
	ffufAgg := &aggregate{}

	if wantFfuf {
		s.emitter.Set(StageFfuf)
		markStarted(StageFfuf)
	}

	var wg sync.WaitGroup
	var discoveredURLs []string
	var mu sync.Mutex

	for _, ep := range endpoints {
		if wantFfuf {
			ffufAgg.mu.Lock()
			ffufAgg.attempts++
			ffufAgg.mu.Unlock()
			wg.Add(1)
			go func(endpoint *models.WebEndpoint) {
				defer wg.Done()
				defer func() {
					if r := recover(); r != nil {
						ffufAgg.mu.Lock()
						ffufAgg.failures++
						if len(ffufAgg.errs) < 10 {
							ffufAgg.errs = append(ffufAgg.errs, fmt.Sprintf("%s: panic %v", endpoint.URL, r))
						}
						ffufAgg.mu.Unlock()
						log.Printf("[slow-scan] ffuf panic %s: %v", endpoint.URL, r)
					}
				}()
				urls, err := s.runFfuf(ctx, endpoint, runID)
				if err != nil {
					ffufAgg.mu.Lock()
					ffufAgg.failures++
					if len(ffufAgg.errs) < 10 {
						ffufAgg.errs = append(ffufAgg.errs, fmt.Sprintf("%s: %v", endpoint.URL, err))
					}
					ffufAgg.mu.Unlock()
					log.Printf("[slow-scan] ffuf %s: %v", endpoint.URL, err)
					return
				}
				mu.Lock()
				discoveredURLs = append(discoveredURLs, urls...)
				mu.Unlock()
			}(ep)
		}
	}

	wg.Wait()

	// Emit terminal stage event. Partial failure is still "completed" — the
	// stage produced some results — but we attach the failed-endpoint count so
	// the UI shows context. Total failure is "failed".
	if wantFfuf {
		finalizeStage(s.emitter, StageFfuf, ffufAgg.attempts, ffufAgg.failures, ffufAgg.errs)
		markTerminated(StageFfuf)
	}

	// Deduplicate and feed to httpx -> nuclei
	if len(discoveredURLs) > 0 {
		unique := dedupStrings(discoveredURLs)
		if err := s.feedToHttpxNuclei(ctx, unique, endpoints); err != nil {
			log.Printf("[slow-scan] feed to httpx/nuclei: %v", err)
		}
	}

	return nil
}

// finalizeStage emits the terminal event for a slow scan stage based on the
// aggregated fan-out result.
//   - attempts == 0: nothing actually ran (e.g. all hosts deduped to zero) →
//     Complete so the stage doesn't sit running.
//   - failures == attempts: every goroutine failed → Fail with the aggregate
//     reason. The UI shows the stage in red.
//   - 0 < failures < attempts: partial success → Complete, but the per-host
//     errors are still logged so an operator can dig in.
func finalizeStage(emitter *StageEmitter, stage StageID, attempts, failures int, errs []string) {
	if attempts == 0 || failures == 0 {
		emitter.Complete(stage)
		return
	}
	if failures == attempts {
		reason := fmt.Sprintf("%d/%d targets failed", failures, attempts)
		if len(errs) > 0 {
			reason = fmt.Sprintf("%s: %s", reason, strings.Join(errs, "; "))
		}
		emitter.Fail(stage, reason)
		return
	}
	// Partial success — still call it complete so the user sees results, but
	// log so the partial failure isn't invisible.
	log.Printf("[slow-scan] %s: %d/%d targets failed (partial success): %s", stage, failures, attempts, strings.Join(errs, "; "))
	emitter.Complete(stage)
}

func (s *SlowScanOrchestrator) runFfuf(ctx context.Context, endpoint *models.WebEndpoint, runID string) ([]string, error) {
	s.sem <- struct{}{}
	defer func() { <-s.sem }()

	// Get dictionary
	dict, err := s.queries.GetDictionary(s.config.FfufDictionaryID)
	if err != nil || dict == nil {
		return nil, fmt.Errorf("dictionary not found: %s", s.config.FfufDictionaryID)
	}

	// Build target URL with FUZZ placeholder from the live web endpoint
	base := strings.TrimSuffix(endpoint.URL, "/")
	targetURL := base + "/FUZZ"

	// Create slow scan task record
	task := &models.SlowScanTask{
		ID:        util.GenerateID(),
		ProjectID: s.projectID,
		TargetID:  nil, // sourced from web_endpoint, not a project target
		RunID:     &runID,
		Tool:      models.SlowScanToolFfuf,
		Status:    models.SlowScanPending,
		RateLimit: s.config.FfufRateLimit,
		Timeout:   s.config.FfufTimeout,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := s.queries.CreateSlowScanTask(task); err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}

	// Build and run via worker
	cmd := worker.BuildFfufCommand(targetURL, dict.FilePath, s.config.FfufRateLimit, s.config.FfufTimeout)
	scanTask := &models.ScanTask{
		ID:              util.GenerateID(),
		ProjectID:       s.projectID,
		RunID:           &runID,
		Tool:            "ffuf",
		CommandTemplate: strings.Join(cmd, " "),
		Status:          models.TaskCreated,
		CreatedAt:       time.Now().UTC(),
	}
	if err := s.queries.CreateScanTask(scanTask); err != nil {
		return nil, fmt.Errorf("create scan task: %w", err)
	}

	now := time.Now().UTC()
	s.queries.SetSlowScanRunning(task.ID, now)

	if err := s.runner.Run(ctx, scanTask.ID); err != nil {
		s.queries.UpdateSlowScanStatus(task.ID, models.SlowScanFailed, err.Error(), &now)
		return nil, err
	}

	stdout, err := s.readTaskStdout(scanTask.ID)
	if err != nil {
		s.queries.UpdateSlowScanStatus(task.ID, models.SlowScanFailed, err.Error(), &now)
		return nil, err
	}

	results, _ := parser.ParseFfufOutput(bytes.NewReader(stdout))
	var urls []string
	for _, r := range results {
		if r.URL != "" {
			urls = append(urls, r.URL)
		}
	}

	s.queries.UpdateSlowScanStatus(task.ID, models.SlowScanCompleted, "", &now)
	return urls, nil
}

// feedToHttpxNuclei runs httpx on discovered URLs, saves new WebEndpoints, then runs nuclei.
func (s *SlowScanOrchestrator) feedToHttpxNuclei(ctx context.Context, urls []string, existingEndpoints []*models.WebEndpoint) error {
	if len(urls) == 0 {
		return nil
	}

	// Deduplicate against existing web endpoints
	existingSet := make(map[string]bool, len(existingEndpoints))
	for _, ep := range existingEndpoints {
		existingSet[ep.URL] = true
	}

	var newURLs []string
	for _, u := range urls {
		if !existingSet[u] {
			newURLs = append(newURLs, u)
		}
	}
	if len(newURLs) == 0 {
		log.Printf("[slow-scan] all %d discovered URLs already exist, skipping", len(urls))
		return nil
	}
	log.Printf("[slow-scan] feeding %d new URLs to httpx/nuclei (deduped from %d)", len(newURLs), len(urls))

	// Write to temp file
	hostFile := filepath.Join(s.dataDir, "workdirs", s.projectID, fmt.Sprintf("slowscan-httpx-%s.txt", util.GenerateID()))
	if err := os.MkdirAll(filepath.Dir(hostFile), 0750); err != nil {
		return err
	}
	if err := os.WriteFile(hostFile, []byte(strings.Join(newURLs, "\n")), 0640); err != nil {
		return err
	}

	// Run httpx
	cmd := worker.BuildHttpxCommand(hostFile, s.config.HttpxRateLimit, s.config.HttpxThreads)
	scanTask := &models.ScanTask{
		ID:              util.GenerateID(),
		ProjectID:       s.projectID,
		Tool:            "httpx",
		CommandTemplate: strings.Join(cmd, " "),
		Status:          models.TaskCreated,
		CreatedAt:       time.Now().UTC(),
	}
	if err := s.queries.CreateScanTask(scanTask); err != nil {
		return fmt.Errorf("create httpx task: %w", err)
	}
	if err := s.runner.Run(ctx, scanTask.ID); err != nil {
		log.Printf("[slow-scan] httpx run error: %v", err)
		return err
	}

	stdout, err := s.readTaskStdout(scanTask.ID)
	if err != nil {
		return fmt.Errorf("read httpx stdout: %w", err)
	}

	endpoints := parser.ParseHttpxOutput(bytes.NewReader(stdout))
	var savedEndpoints []*models.WebEndpoint
	for _, ep := range endpoints {
		ep.ID = util.GenerateID()
		ep.ProjectID = s.projectID
		ep.CreatedAt = time.Now().UTC()
		if err := s.queries.CreateWebEndpoint(ep); err != nil {
			log.Printf("[slow-scan] save endpoint %s: %v", ep.URL, err)
			continue
		}
		savedEndpoints = append(savedEndpoints, ep)
	}
	log.Printf("[slow-scan] saved %d new web endpoints", len(savedEndpoints))

	if len(savedEndpoints) == 0 {
		return nil
	}

	// Run nuclei on new endpoints
	groups := make(map[string][]string)
	for _, ep := range savedEndpoints {
		key := strings.Join(nuclei.MapPreciseTags(ep.Technologies, ""), ",")
		if key == "" {
			key = "generic"
		}
		groups[key] = append(groups[key], ep.URL)
	}

	for tagKey, urls := range groups {
		tags := strings.Split(tagKey, ",")
		targetFile := filepath.Join(s.dataDir, "workdirs", s.projectID, fmt.Sprintf("slowscan-nuclei-%s.txt", util.GenerateID()))
		if err := os.WriteFile(targetFile, []byte(strings.Join(urls, "\n")), 0640); err != nil {
			continue
		}

		cmd := worker.BuildNucleiCommand(targetFile, "deep", s.config.NucleiRateLimit, s.config.NucleiRateLimitPerMinute, s.config.NucleiConcurrency, tags, s.config.NucleiScanDepth, "")
		scanTask := &models.ScanTask{
			ID:              util.GenerateID(),
			ProjectID:       s.projectID,
			Tool:            "nuclei",
			CommandTemplate: strings.Join(cmd, " "),
			Status:          models.TaskCreated,
			CreatedAt:       time.Now().UTC(),
		}
		if err := s.queries.CreateScanTask(scanTask); err != nil {
			continue
		}
		if err := s.runner.Run(ctx, scanTask.ID); err != nil {
			log.Printf("[slow-scan] nuclei run error for tags %s: %v", tagKey, err)
			continue
		}
	}

	return nil
}

func (s *SlowScanOrchestrator) readTaskStdout(taskID string) ([]byte, error) {
	artifacts, err := s.queries.ListRawArtifactsByTask(taskID)
	if err != nil {
		return nil, err
	}
	for _, a := range artifacts {
		if a.Type == models.ArtifactStdout {
			return os.ReadFile(a.Path)
		}
	}
	return nil, fmt.Errorf("no stdout artifact for task %s", taskID)
}
