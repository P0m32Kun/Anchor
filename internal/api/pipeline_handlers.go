package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/P0m32Kun/Anchor/internal/errors"
	"github.com/P0m32Kun/Anchor/internal/evaluator"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/scanconfig"
	"github.com/P0m32Kun/Anchor/internal/scanengine"
	"github.com/P0m32Kun/Anchor/internal/scanengine/core"
	"github.com/P0m32Kun/Anchor/internal/scanengine/seed"
	"github.com/P0m32Kun/Anchor/internal/util"
	"github.com/P0m32Kun/Anchor/internal/toolregistry"
)

// finalizePipelineRun sets the terminal pipeline run status after the scan engine exits.
// It never marks a run completed while work items are still pending.
func (s *Server) finalizePipelineRun(runID string, runErr error) {
	terminal, err := s.queries.AllWorkItemsTerminal(runID)
	if err != nil {
		log.Printf("[scan] finalize run %s: check work terminal: %v", runID, err)
	}

	if !terminal {
		pending, listErr := s.queries.ListScanWorkItemsByRunAndStatus(runID, models.WorkStatusPending)
		if listErr != nil {
			log.Printf("[scan] finalize run %s: list pending work: %v", runID, listErr)
		} else {
			now := time.Now().UTC()
			for _, w := range pending {
				_ = s.queries.UpdateScanWorkItemSkip(w.ID, models.WorkStatusSkipped, "unfinished", &now)
			}
			if len(pending) > 0 {
				log.Printf("[scan] finalize run %s: skipped %d orphan pending work items", runID, len(pending))
			}
		}
	}

	if runErr != nil {
		msg := runErr.Error()
		if runErr == scanengine.ErrAbsoluteTimeout {
			_, _, done, skipped, failed, _ := s.queries.CountScanWorkItemsByStatus(runID)
			msg = fmt.Sprintf("absolute timeout reached (%d done, %d skipped, %d failed)", done, skipped, failed)
		}
		_ = s.queries.UpdatePipelineRunError(runID, msg)
		_ = s.queries.UpdatePipelineRunStatus(runID, "failed")
		return
	}

	terminal, err = s.queries.AllWorkItemsTerminal(runID)
	if err != nil {
		log.Printf("[scan] finalize run %s: recheck work terminal: %v", runID, err)
	}
	if !terminal {
		_ = s.queries.UpdatePipelineRunError(runID, "scan finished with unfinished work items")
		_ = s.queries.UpdatePipelineRunStatus(runID, "failed")
		return
	}

	if err := s.queries.UpdatePipelineRunCompleted(runID, time.Now().UTC()); err != nil {
		log.Printf("[scan] finalize run %s completed: %v", runID, err)
	}
}


func (s *Server) handleListPipelineRuns(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	if projectID == "" {
		writeError(w, http.StatusBadRequest, errors.New("MISSING_PROJECT_ID", "Project ID is required"))
		return
	}

	runs, err := s.queries.ListPipelineRunsByProject(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("DB_ERROR", err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"runs": runs,
	})
}

func (s *Server) handleGetPipelineRun(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	runID := r.PathValue("runId")
	if projectID == "" || runID == "" {
		writeError(w, http.StatusBadRequest, errors.New("MISSING_PARAM", "Project ID and Run ID are required"))
		return
	}

	run, err := s.queries.GetPipelineRun(runID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("DB_ERROR", err.Error()))
		return
	}
	if run == nil {
		writeError(w, http.StatusNotFound, errors.New("NOT_FOUND", "Pipeline run not found"))
		return
	}
	if run.ProjectID != projectID {
		writeError(w, http.StatusNotFound, errors.New("NOT_FOUND", "Pipeline run not found"))
		return
	}

	writeJSON(w, http.StatusOK, run)
}

// PhaseCoverage holds per-phase coverage statistics.
type PhaseCoverage struct {
	Stage string `json:"stage"`
	Total int    `json:"total"`
	Done  int    `json:"done"`
	Pct   int    `json:"pct"`
}

// handleGetRunSummary returns delta counts and per-phase coverage for a pipeline run.
func (s *Server) handleGetRunSummary(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	runID := r.PathValue("runId")
	if projectID == "" || runID == "" {
		writeError(w, http.StatusBadRequest, errors.New("MISSING_PARAMS", "Project ID and Run ID are required"))
		return
	}

	// Verify run belongs to project
	run, err := s.queries.GetPipelineRun(runID)
	if err != nil || run == nil || run.ProjectID != projectID {
		writeError(w, http.StatusNotFound, errors.New("NOT_FOUND", "Run not found"))
		return
	}

	// Count findings for this run
	findings, err := s.queries.ListFindingsByRun(projectID, runID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("DB_ERROR", err.Error()))
		return
	}

	// Count signals created during this run's time window
	var signalCount int
	if !run.StartedAt.IsZero() {
		signals, err := s.queries.ListSignalsByProjectSince(projectID, run.StartedAt)
		if err == nil {
			signalCount = len(signals)
		}
	}

	// Build per-phase coverage from pipeline_run_stages
	stages, err := s.queries.ListPipelineRunStages(runID)
	if err != nil {
		log.Printf("[summary] list stages for run %s: %v", runID, err)
	}

	var phases []PhaseCoverage
	totalPending := 0
	var pendingStages []string
	for _, st := range stages {
		total := 0
		done := 0
		if st.WorkTotal != nil {
			total = *st.WorkTotal
		}
		if st.WorkDone != nil {
			done = *st.WorkDone
		}
		pct := 0
		if total > 0 {
			pct = int(float64(done) / float64(total) * 100)
		}
		phases = append(phases, PhaseCoverage{
			Stage: st.Stage,
			Total: total,
			Done:  done,
			Pct:   pct,
		})

		pending := total - done
		if pending > 0 {
			totalPending += pending
			pendingStages = append(pendingStages, st.Stage)
		}
	}

	isComplete := totalPending == 0
	var incompleteReason string
	if !isComplete {
		incompleteReason = fmt.Sprintf("%d work items still pending in: %s", totalPending, strings.Join(pendingStages, ", "))
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"run_id":            runID,
		"new_findings":      len(findings),
		"new_signals":       signalCount,
		"status":            run.Status,
		"phases":            phases,
		"complete":          isComplete,
		"incomplete_reason": incompleteReason,
	})
}

func (s *Server) handleGetPipelineConfig(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	if projectID == "" {
		writeError(w, http.StatusBadRequest, errors.New("MISSING_PROJECT_ID", "Project ID is required"))
		return
	}

	project, err := s.queries.GetProject(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("DB_ERROR", err.Error()))
		return
	}
	if project == nil {
		writeError(w, http.StatusNotFound, errors.New("NOT_FOUND", "Project not found"))
		return
	}

	cfg := models.DefaultPipelineConfig()
	if project.PipelineConfig != nil && *project.PipelineConfig != "" {
		if err := json.Unmarshal([]byte(*project.PipelineConfig), &cfg); err != nil {
			log.Printf("parse pipeline config: %v", err)
		}
	}

	writeJSON(w, http.StatusOK, cfg)
}

func (s *Server) handleUpdatePipelineConfig(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	if projectID == "" {
		writeError(w, http.StatusBadRequest, errors.New("MISSING_PROJECT_ID", "Project ID is required"))
		return
	}

	var cfg models.PipelineConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeError(w, http.StatusBadRequest, errors.New("INVALID_BODY", err.Error()))
		return
	}

	cfgJSON, err := json.Marshal(cfg)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("SERIALIZE_ERROR", err.Error()))
		return
	}

	if err := s.queries.UpdateProjectPipelineConfig(projectID, string(cfgJSON)); err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("DB_ERROR", err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) handleGetPipelineRunStages(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	runID := r.PathValue("runId")
	if projectID == "" || runID == "" {
		writeError(w, http.StatusBadRequest, errors.New("MISSING_PARAM", "Project ID and Run ID are required"))
		return
	}

	// Verify run belongs to project.
	run, err := s.queries.GetPipelineRun(runID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("DB_ERROR", err.Error()))
		return
	}
	if run == nil || run.ProjectID != projectID {
		writeError(w, http.StatusNotFound, errors.New("NOT_FOUND", "Pipeline run not found"))
		return
	}

	stages, err := s.queries.ListPipelineRunStages(runID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("DB_ERROR", err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"stages": stages,
	})
}

// --- Unified Scan ---

// buildConfigForMode returns a PipelineConfig for the given scan mode.
// Speed parameters are loaded from the request body; defaults are applied for zero values.
// Tool toggles (enable_spoor, enable_katana, enable_ffuf) are NOT overridden — the
// frontend controls these via the ScanModal tool section, and the backend respects
// whatever the user submitted.
func buildConfigForMode(mode string, cfg models.PipelineConfig) models.PipelineConfig {
	// Normalize legacy mode values.
	nMode, nNoise := models.NormalizeScanMode(mode, cfg.NoiseLevel)
	cfg.ScanMode = nMode
	if nNoise != "" {
		cfg.NoiseLevel = nNoise
	}
	defaults := presetDefaults(nMode, nNoise)
	if cfg.PortRange == "" {
		cfg.PortRange = defaults.PortRange
	}
	if cfg.SubfinderRateLimit == 0 {
		cfg.SubfinderRateLimit = defaults.SubfinderRateLimit
	}
	if cfg.SubfinderThreads == 0 {
		cfg.SubfinderThreads = defaults.SubfinderThreads
	}
	if cfg.SubfinderTimeout == 0 {
		cfg.SubfinderTimeout = defaults.SubfinderTimeout
	}
	if cfg.DNSxRateLimit == 0 {
		cfg.DNSxRateLimit = defaults.DNSxRateLimit
	}
	if cfg.DNSxThreads == 0 {
		cfg.DNSxThreads = defaults.DNSxThreads
	}
	if cfg.DNSxTimeout == 0 {
		cfg.DNSxTimeout = defaults.DNSxTimeout
	}
	if cfg.NaabuRate == 0 {
		cfg.NaabuRate = defaults.NaabuRate
	}
	if cfg.NaabuThreads == 0 {
		cfg.NaabuThreads = defaults.NaabuThreads
	}
	if cfg.NaabuTimeout == 0 {
		cfg.NaabuTimeout = defaults.NaabuTimeout
	}
	if cfg.NmapServiceTimeout == 0 {
		cfg.NmapServiceTimeout = defaults.NmapServiceTimeout
	}
	if cfg.HttpxRateLimit == 0 {
		cfg.HttpxRateLimit = defaults.HttpxRateLimit
	}
	if cfg.HttpxThreads == 0 {
		cfg.HttpxThreads = defaults.HttpxThreads
	}
	if cfg.NucleiRateLimit == 0 {
		cfg.NucleiRateLimit = defaults.NucleiRateLimit
	}
	if cfg.NucleiConcurrency == 0 {
		cfg.NucleiConcurrency = defaults.NucleiConcurrency
	}
	if cfg.NucleiScanDepth == "" {
		cfg.NucleiScanDepth = defaults.NucleiScanDepth
	}
	if cfg.FofaResultLimit == 0 {
		cfg.FofaResultLimit = defaults.FofaResultLimit
	}
	if cfg.FofaConcurrency == 0 {
		cfg.FofaConcurrency = defaults.FofaConcurrency
	}

	switch nMode {
	case "external":
		cfg.EnableFOFA = true
		cfg.EnableSubfinder = true
		cfg.EnableDNSx = true
		cfg.EnableCDNFilter = true
		cfg.EnableNmapService = true
		cfg.EnableHttpx = true
		cfg.EnableNuclei = true
		cfg.EnablePassiveSearch = defaults.EnablePassiveSearch
		cfg.EnablePassiveCert = defaults.EnablePassiveCert
		cfg.EnablePassiveURL = defaults.EnablePassiveURL
		if cfg.SubfinderMode == "" {
			cfg.SubfinderMode = defaults.SubfinderMode
		}
		if cfg.PassiveSearchResultLimit == 0 {
			cfg.PassiveSearchResultLimit = defaults.PassiveSearchResultLimit
		}
		if cfg.PassiveSearchConcurrency == 0 {
			cfg.PassiveSearchConcurrency = defaults.PassiveSearchConcurrency
		}
		cfg.EnablePassiveJunkFilter = defaults.EnablePassiveJunkFilter
		cfg.SkipPortscanOnCDNHost = defaults.SkipPortscanOnCDNHost
		cfg.NucleiRequireFingerprint = defaults.NucleiRequireFingerprint
	case "internal":
		cfg.EnableFOFA = false
		cfg.EnableSubfinder = false
		cfg.EnableDNSx = false
		cfg.EnableCDNFilter = false
		cfg.EnableNmapService = true
		cfg.EnableHttpx = true
		cfg.EnableNuclei = true
	case "watch":
		cfg.EnableFOFA = true
		cfg.EnableSubfinder = true
		cfg.EnableDNSx = true
		cfg.EnableCDNFilter = true
		cfg.EnableNmapService = true
		cfg.EnableHttpx = true
		cfg.EnableNuclei = true
		cfg.EnablePassiveSearch = defaults.EnablePassiveSearch
		cfg.EnablePassiveCert = defaults.EnablePassiveCert
		cfg.EnablePassiveURL = defaults.EnablePassiveURL
		if cfg.SubfinderMode == "" {
			cfg.SubfinderMode = defaults.SubfinderMode
		}
		if cfg.PassiveSearchResultLimit == 0 {
			cfg.PassiveSearchResultLimit = defaults.PassiveSearchResultLimit
		}
		if cfg.PassiveSearchConcurrency == 0 {
			cfg.PassiveSearchConcurrency = defaults.PassiveSearchConcurrency
		}
		cfg.EnablePassiveJunkFilter = defaults.EnablePassiveJunkFilter
		cfg.SkipPortscanOnCDNHost = defaults.SkipPortscanOnCDNHost
		cfg.NucleiRequireFingerprint = defaults.NucleiRequireFingerprint
	}
	// Tool toggles (enable_spoor, enable_katana, enable_ffuf) are NOT forced here.
	// The frontend controls these via the ScanModal tool section; defaults are
	// provided by the preset, but the user's explicit choice is always respected.
	return cfg
}

// presetDefaults returns the default PipelineConfig for the given mode and noise level.
// It normalizes legacy mode values before lookup.
func presetDefaults(mode, noiseLevel string) models.PipelineConfig {
	if sc := scanconfig.Get(); sc != nil {
		// Construct the full preset name: "external_low", "external_standard", "watch", "internal"
		presetName := mode
		if mode == "external" && noiseLevel != "" {
			presetName = "external_" + noiseLevel
		}
		return sc.Preset(presetName)
	}
	switch mode {
	case "external":
		if noiseLevel == "low" {
			return models.DefaultExternalLowNoisePipelineConfig()
		}
		return models.DefaultExternalStandardPipelineConfig()
	case "watch":
		return models.DefaultWatchPipelineConfig()
	default:
		return models.DefaultPipelineConfig()
	}
}

func (s *Server) handleCreateScan(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	if projectID == "" {
		writeError(w, http.StatusBadRequest, errors.New("MISSING_PROJECT_ID", "Project ID is required"))
		return
	}

	var req struct {
		Mode   string                `json:"mode"`
		Config models.PipelineConfig `json:"config"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New("INVALID_BODY", "Invalid request body"))
		return
	}

	if req.Mode == "" {
		req.Mode = "external"
	}

	project, err := s.queries.GetProject(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("DB_ERROR", err.Error()))
		return
	}
	if project == nil {
		writeError(w, http.StatusNotFound, errors.New("NOT_FOUND", "Project not found"))
		return
	}

	cfg := buildConfigForMode(req.Mode, req.Config)

	// Persist config to project so the ScanModal can reload it next time.
	// Side effects (save failure) are non-fatal — the pipeline still runs.
	if cfgJSON, err := json.Marshal(cfg); err == nil {
		if err := s.queries.UpdateProjectPipelineConfig(projectID, string(cfgJSON)); err != nil {
			log.Printf("[scan] persist pipeline config for project %s: %v", projectID, err)
		}
	}

	// Create pipeline run with mode
	runID := util.GenerateID()
	now := time.Now().UTC()
	if err := s.queries.CreatePipelineRun(&models.PipelineRun{
		ID:        runID,
		ProjectID: projectID,
		Mode:      req.Mode,
		Status:    "running",
		StartedAt: now,
		CreatedAt: now,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("DB_ERROR", err.Error()))
		return
	}

	// Launch pipeline in background with a cancelable context so the
	// force-stop handler can signal the pipeline goroutine to exit.
	ctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.pipelineCancels[runID] = cancel
	s.mu.Unlock()

	go func() {
		defer func() {
			s.mu.Lock()
			delete(s.pipelineCancels, runID)
			s.mu.Unlock()
			cancel()
			// Safety net: engine may stop while run status is still "running".
			run, err := s.queries.GetPipelineRun(runID)
			if err == nil && run != nil && run.Status == "running" &&
				(run.EngineState == "stopped" || run.EngineState == "wind_down") {
				s.finalizePipelineRun(runID, nil)
			}
		}()

		stageCallback := func(rid string, stage string, status string, errMsg string) {
			s.broadcastProjectSSE(projectID, map[string]interface{}{
				"event":   "pipeline_stage_change",
				"run_id":  rid,
				"stage":   stage,
				"status":  status,
				"error":   errMsg,
				"time":    time.Now().UTC().Format(time.RFC3339),
			})
		}

		// Asset-driven scan engine
		profile := core.ProfileFromConfig(req.Mode, cfg)
		engineCfg := scanengine.DefaultEngineConfig()
		engineCfg.Pipeline = cfg
		engine := scanengine.New(
			s.queries, s.worker, toolregistry.DefaultRegistry(),
			s.assetMerger, profile, s.excludeMgr, s.scopeEng, s.dataDir, runID, projectID, engineCfg,
			func(rid, stage, status, errMsg string) {
				stageCallback(rid, stage, status, errMsg)
			},
		)
		// BW4: SSE callback for new assets
		engine.SetOnNewAsset(func(assetID, value, assetType string) {
			s.broadcastProjectSSE(projectID, map[string]interface{}{
				"event":      "asset.new",
				"run_id":     runID,
				"asset_id":   assetID,
				"value":      value,
				"asset_type": assetType,
				"time":       time.Now().UTC().Format(time.RFC3339),
			})
		})
		// Get targets for seeding (expand company targets via passive search when configured)
		targets, _ := s.queries.ListTargetsByProject(projectID)
		seeds := seed.ExpandTargets(ctx, s.queries, cfg, targets)
		if len(seeds) == 0 {
			for _, t := range targets {
				if t == nil || t.Value == "" {
					continue
				}
				seeds = append(seeds, seed.SeedAsset{
					Value:     t.Value,
					ValueType: string(t.Type),
					Source:    "target",
					SourceRef: t.ID,
				})
			}
		}
		// Gate A: filter seeds by scope boundary (strict mode only)
		if proj, _ := s.queries.GetProject(projectID); proj != nil && proj.ScopeBoundaryMode == models.ScopeBoundaryStrict {
			scopeRules, _ := s.queries.ListScopeRulesByProject(projectID)
			before := len(seeds)
			seeds = seed.FilterSeedsByBoundary(seeds, s.scopeEng, scopeRules, proj.ScopeBoundaryMode)
			if filtered := before - len(seeds); filtered > 0 {
				log.Printf("[scope] Gate A filtered %d seeds for project %s (mode=strict)", filtered, projectID)
			}
		}
		runErr := engine.RunWithSeeds(ctx, seeds)
		if runErr != nil {
			log.Printf("scan engine run %s for project %s: %v", runID, projectID, runErr)
		}
		s.finalizePipelineRun(runID, runErr)
		// Trigger evaluation asynchronously
		go func() {
			eval := evaluator.NewEvaluator(s.queries, s.dataDir, projectID, runID)
			_, evalErr := eval.Evaluate(context.Background())
			if evalErr != nil {
				log.Printf("[evaluation] failed for run %s: %v", runID, evalErr)
			} else {
				log.Printf("[evaluation] report generated for run %s", runID)
			}
		}()

		s.broadcastProjectSSE(projectID, map[string]interface{}{
			"event":  "pipeline_complete",
			"run_id": runID,
			"time":   time.Now().UTC().Format(time.RFC3339),
		})
	}()

	writeJSON(w, http.StatusAccepted, map[string]string{
		"run_id": runID,
		"status": "accepted",
		"mode":   req.Mode,
	})
}

func (s *Server) handleListScanRuns(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	if projectID == "" {
		writeError(w, http.StatusBadRequest, errors.New("MISSING_PROJECT_ID", "Project ID is required"))
		return
	}

	pg := parsePagination(r)

	total, err := s.queries.CountPipelineRunsByProject(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("DB_ERROR", err.Error()))
		return
	}

	runs, err := s.queries.ListPipelineRunsByProjectPaginated(projectID, pg.PageSize, pg.Offset())
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("DB_ERROR", err.Error()))
		return
	}

	writePaginatedJSON(w, runs, total, pg)
}

func (s *Server) handleCancelPipelineRun(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	runID := r.PathValue("runId")
	if projectID == "" || runID == "" {
		writeError(w, http.StatusBadRequest, errors.New("MISSING_PARAM", "Project ID and Run ID are required"))
		return
	}

	run, err := s.queries.GetPipelineRun(runID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("DB_ERROR", err.Error()))
		return
	}
	if run == nil || run.ProjectID != projectID {
		writeError(w, http.StatusNotFound, errors.New("NOT_FOUND", "pipeline run not found"))
		return
	}

	if run.Status == "completed" || run.Status == "failed" || run.Status == "cancelled" {
		writeError(w, http.StatusBadRequest, errors.New("ALREADY_FINISHED", "run already finished"))
		return
	}

	tasks, err := s.queries.ListScanTasksByRun(runID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("DB_ERROR", err.Error()))
		return
	}

	now := time.Now().UTC()
	for _, task := range tasks {
		if task.Status == models.TaskCompleted || task.Status == models.TaskFailed || task.Status == models.TaskCancelled {
			continue
		}
		_ = s.queries.UpdateScanTaskStatus(task.ID, models.TaskCancelled, nil, &now)
		_ = s.worker.Cancel(task.ID)
	}

	s.queries.UpdatePipelineRunStatus(runID, "cancelled")

	// Cancel the pipeline goroutine's context so it exits at the next stage
	// boundary instead of continuing to run remaining stages.
	s.mu.Lock()
	if cancel, ok := s.pipelineCancels[runID]; ok {
		cancel()
		delete(s.pipelineCancels, runID)
	}
	s.mu.Unlock()

	s.broadcastProjectSSE(projectID, map[string]interface{}{
		"event":  "pipeline_complete",
		"run_id": runID,
		"time":   now.Format(time.RFC3339),
	})

	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

// triggerWatchRun is called by the watch scheduler when a project's interval has elapsed.
// It creates a passive-only pipeline run for the project.
func (s *Server) triggerWatchRun(ctx context.Context, project *models.Project) error {
	// Build passive-only config
	cfg := models.DefaultExternalPipelineConfig()
	cfg.EnablePassiveSearch = true
	cfg.EnablePassiveCert = true
	cfg.EnablePassiveURL = true
	cfg.SubfinderMode = "passive"
	cfg.EnableKatana = false
	cfg.EnableFfuf = false
	cfg.EnableNuclei = false
	cfg.EnableNmapService = false
	cfg.PortRange = "top100"

	// Create pipeline run
	rid := util.GenerateID()
	now := time.Now().UTC()
	if err := s.queries.CreatePipelineRun(&models.PipelineRun{
		ID:        rid,
		ProjectID: project.ID,
		Mode:      "watch_passive",
		Status:    "running",
		StartedAt: now,
		CreatedAt: now,
	}); err != nil {
		return fmt.Errorf("create pipeline run: %w", err)
	}

	// Run the scan in a goroutine
	go func() {
		log.Printf("[watch] starting passive scan for project %s (run %s)", project.ID, rid)

		s.broadcastProjectSSE(project.ID, map[string]interface{}{
			"event":  "pipeline_start",
			"run_id": rid,
			"mode":   "watch_passive",
			"time":   time.Now().UTC().Format(time.RFC3339),
		})

		profile := core.ProfileFromConfig("external", cfg)
		engineCfg := scanengine.DefaultEngineConfig()
		engineCfg.Pipeline = cfg
		engine := scanengine.New(
			s.queries, s.worker, toolregistry.DefaultRegistry(),
			s.assetMerger, profile, s.excludeMgr, s.scopeEng, s.dataDir, rid, project.ID, engineCfg,
			func(rid, stage, status, errMsg string) {
				s.broadcastProjectSSE(project.ID, map[string]interface{}{
					"event":   "pipeline_stage_change",
					"run_id":  rid,
					"stage":   stage,
					"status":  status,
					"error":   errMsg,
					"time":    time.Now().UTC().Format(time.RFC3339),
				})
			},
		)
		// BW4: SSE callback for new assets in watch mode
		engine.SetOnNewAsset(func(assetID, value, assetType string) {
			s.broadcastProjectSSE(project.ID, map[string]interface{}{
				"event":      "asset.new",
				"run_id":     rid,
				"asset_id":   assetID,
				"value":      value,
				"asset_type": assetType,
				"time":       time.Now().UTC().Format(time.RFC3339),
			})
		})

		targets, _ := s.queries.ListTargetsByProject(project.ID)
		seeds := seed.ExpandTargets(ctx, s.queries, cfg, targets)
		if len(seeds) == 0 {
			for _, t := range targets {
				if t == nil || t.Value == "" {
					continue
				}
				seeds = append(seeds, seed.SeedAsset{
					Value:     t.Value,
					ValueType: string(t.Type),
					Source:    "target",
					SourceRef: t.ID,
				})
			}
		}

		runErr := engine.RunWithSeeds(ctx, seeds)
		if runErr != nil {
			log.Printf("[watch] scan engine run %s for project %s: %v", rid, project.ID, runErr)
		}
		s.finalizePipelineRun(rid, runErr)

		s.broadcastProjectSSE(project.ID, map[string]interface{}{
			"event":  "pipeline_complete",
			"run_id": rid,
			"time":   time.Now().UTC().Format(time.RFC3339),
		})

		log.Printf("[watch] completed passive scan for project %s (run %s)", project.ID, rid)
	}()

	return nil
}
