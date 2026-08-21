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
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/scanconfig"
	"github.com/P0m32Kun/Anchor/internal/scanengine"
	"github.com/P0m32Kun/Anchor/internal/scanengine/core"
	"github.com/P0m32Kun/Anchor/internal/scanengine/seed"
	"github.com/P0m32Kun/Anchor/internal/util"
	"github.com/P0m32Kun/Anchor/internal/toolregistry"
)

// handleScanDiff compares two pipeline runs: base vs target.
// GET /projects/{id}/scan/diff?base={runId1}&target={runId2}
func (s *Server) handleScanDiff(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	baseRunID := r.URL.Query().Get("base")
	targetRunID := r.URL.Query().Get("target")
	if projectID == "" || baseRunID == "" || targetRunID == "" {
		writeError(w, http.StatusBadRequest, errors.New("MISSING_PARAMS", "Project ID, base and target run IDs are required"))
		return
	}
	if baseRunID == targetRunID {
		writeError(w, http.StatusBadRequest, errors.New("SAME_RUN", "base and target must be different runs"))
		return
	}

	// Fetch both runs
	baseRun, err := s.queries.GetPipelineRun(baseRunID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("DB_ERROR", err.Error()))
		return
	}
	if baseRun == nil || baseRun.ProjectID != projectID {
		writeError(w, http.StatusNotFound, errors.New("NOT_FOUND", "base run not found"))
		return
	}

	targetRun, err := s.queries.GetPipelineRun(targetRunID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("DB_ERROR", err.Error()))
		return
	}
	if targetRun == nil || targetRun.ProjectID != projectID {
		writeError(w, http.StatusNotFound, errors.New("NOT_FOUND", "target run not found"))
		return
	}

	// Fetch assets for both runs
	baseAssets, err := s.queries.ListAssetsByRun(projectID, baseRunID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("DB_ERROR", err.Error()))
		return
	}
	targetAssets, err := s.queries.ListAssetsByRun(projectID, targetRunID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("DB_ERROR", err.Error()))
		return
	}

	// Fetch findings for both runs
	baseFindings, err := s.queries.ListFindingsByRun(projectID, baseRunID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("DB_ERROR", err.Error()))
		return
	}
	targetFindings, err := s.queries.ListFindingsByRun(projectID, targetRunID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("DB_ERROR", err.Error()))
		return
	}

	// Build lookup maps for diff
	baseAssetMap := make(map[string]*models.Asset)
	for _, a := range baseAssets {
		baseAssetMap[a.NormalizedValue] = a
	}
	targetAssetMap := make(map[string]*models.Asset)
	for _, a := range targetAssets {
		targetAssetMap[a.NormalizedValue] = a
	}

	baseFindingMap := make(map[string]*models.Finding)
	for _, f := range baseFindings {
		baseFindingMap[f.DedupKey] = f
	}
	targetFindingMap := make(map[string]*models.Finding)
	for _, f := range targetFindings {
		targetFindingMap[f.DedupKey] = f
	}

	// Compute asset diff
	var assetsAdded, assetsRemoved, assetsUnchanged []models.ScanAssetDiffItem
	for nv, a := range targetAssetMap {
		if _, exists := baseAssetMap[nv]; exists {
			assetsUnchanged = append(assetsUnchanged, models.ScanAssetDiffItem{Asset: *a, RunID: targetRunID})
		} else {
			assetsAdded = append(assetsAdded, models.ScanAssetDiffItem{Asset: *a, RunID: targetRunID})
		}
	}
	for nv, a := range baseAssetMap {
		if _, exists := targetAssetMap[nv]; !exists {
			assetsRemoved = append(assetsRemoved, models.ScanAssetDiffItem{Asset: *a, RunID: baseRunID})
		}
	}

	// Compute finding diff
	var findingsAdded, findingsRemoved, findingsUnchanged []models.ScanFindingDiffItem
	for dk, f := range targetFindingMap {
		if _, exists := baseFindingMap[dk]; exists {
			findingsUnchanged = append(findingsUnchanged, models.ScanFindingDiffItem{Finding: *f})
		} else {
			findingsAdded = append(findingsAdded, models.ScanFindingDiffItem{Finding: *f})
		}
	}
	for dk, f := range baseFindingMap {
		if _, exists := targetFindingMap[dk]; !exists {
			findingsRemoved = append(findingsRemoved, models.ScanFindingDiffItem{Finding: *f})
		}
	}

	result := models.ScanDiffResult{
		BaseRunID:   baseRunID,
		TargetRunID: targetRunID,
		BaseRun:     baseRun,
		TargetRun:   targetRun,
		Assets: models.ScanDiffAssets{
			Added:     assetsAdded,
			Removed:   assetsRemoved,
			Unchanged: assetsUnchanged,
		},
		Findings: models.ScanDiffFindings{
			Added:     findingsAdded,
			Removed:   findingsRemoved,
			Unchanged: findingsUnchanged,
		},
		Summary: models.ScanDiffSummary{
			AssetsAdded:       len(assetsAdded),
			AssetsRemoved:     len(assetsRemoved),
			AssetsUnchanged:   len(assetsUnchanged),
			FindingsAdded:     len(findingsAdded),
			FindingsRemoved:   len(findingsRemoved),
			FindingsUnchanged: len(findingsUnchanged),
		},
	}

	writeJSON(w, http.StatusOK, result)
}

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

		running, listErr := s.queries.ListScanWorkItemsByRunAndStatus(runID, models.WorkStatusRunning)
		if listErr != nil {
			log.Printf("[scan] finalize run %s: list running work: %v", runID, listErr)
		} else {
			now := time.Now().UTC()
			for _, w := range running {
				_ = s.queries.UpdateScanWorkItemError(w.ID, models.WorkStatusFailed, "orphan: engine exited before completion", &now)
			}
			if len(running) > 0 {
				log.Printf("[scan] finalize run %s: failed %d orphan running work items", runID, len(running))
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

	cfg := models.DefaultExternalPipelineConfig()
	if project.PipelineConfig != nil && *project.PipelineConfig != "" {
		if err := json.Unmarshal([]byte(*project.PipelineConfig), &cfg); err != nil {
			log.Printf("parse pipeline config: %v", err)
		}
	}

	// P1-1: surface a compatibility note (header only) so clients know a stored
	// config predates the internal→internet exit. The payload is returned as-is;
	// the server never silently rewrites scope.
	if scanconfig.IsLegacyInternalConfig(cfg) {
		w.Header().Set("X-Anchor-Compatibility", "internal-mode-migrated")
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

	// P1-1: saving a legacy internal-shaped config would silently widen scope on a
	// later scan. Require an explicit migration flag; otherwise reject.
	if scanconfig.IsLegacyInternalConfig(cfg) && r.URL.Query().Get("migrate") != "external" {
		writeError(w, http.StatusGone, scanconfig.InternalModeError())
		return
	}
	if scanconfig.IsLegacyInternalConfig(cfg) {
		cfg = scanconfig.MigrateInternalConfigToExternal(cfg)
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

// Compatibility helpers for the retired internal scan mode live in
// internal/scanconfig/compat.go (IsInternalMode, IsLegacyInternalConfig,
// MigrateInternalConfigToExternal, ErrInternalModeRemoved), not in the handler.

// buildConfigForMode returns a PipelineConfig for the given scan mode.
// Only the internet scan mode is supported; "internal" is rejected upstream.
// Speed parameters are loaded from the request body; defaults are applied for zero values.
// Tool toggles (enable_spoor, enable_katana, enable_ffuf) are NOT overridden — the
// frontend controls these via the ScanModal tool section, and the backend respects
// whatever the user submitted.
func buildConfigForMode(mode string, cfg models.PipelineConfig) models.PipelineConfig {
	defaults := presetDefaults(mode)
	if cfg.PortRange == "" {
		cfg.PortRange = defaults.PortRange
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

	// Internet mode: enable the standard mapping toolset with external defaults.
	cfg.EnableSubfinder = true
	cfg.EnableDNSx = true
	cfg.EnableCDNFilter = true
	cfg.EnableNmapService = true
	cfg.EnableHttpx = true
	cfg.EnableNuclei = true
	cfg.EnablePassiveSearch = defaults.EnablePassiveSearch
	if cfg.PassiveSearchResultLimit == 0 {
		cfg.PassiveSearchResultLimit = defaults.PassiveSearchResultLimit
	}
	if cfg.PassiveSearchConcurrency == 0 {
		cfg.PassiveSearchConcurrency = defaults.PassiveSearchConcurrency
	}
	cfg.EnablePassiveJunkFilter = defaults.EnablePassiveJunkFilter
	cfg.SkipPortscanOnCDNHost = defaults.SkipPortscanOnCDNHost
	cfg.NucleiRequireFingerprint = defaults.NucleiRequireFingerprint
	// Tool toggles (enable_spoor, enable_katana, enable_ffuf) are NOT forced here.
	// The frontend controls these via the ScanModal tool section; defaults are
	// provided by the preset, but the user's explicit choice is always respected.
	return cfg
}

// presetDefaults returns the default PipelineConfig for the given mode.
// Only the internet ("external") preset is available.
func presetDefaults(mode string) models.PipelineConfig {
	if sc := scanconfig.Get(); sc != nil {
		return sc.Preset(mode)
	}
	return models.DefaultExternalPipelineConfig()
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

	// P1-1: reject the retired internal mode. A saved internal-shaped config may be
	// explicitly migrated only when the caller passes ?migrate=external.
	if scanconfig.IsInternalMode(req.Mode) {
		if r.URL.Query().Get("migrate") != "external" {
			writeError(w, http.StatusGone, scanconfig.InternalModeError())
			return
		}
		// Explicit migration request.
		req.Mode = "external"
		req.Config = scanconfig.MigrateInternalConfigToExternal(req.Config)
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

	// The run always uses the validated internet profile, regardless of any saved
	// legacy internal config. P1-1: never silently overwrite a saved legacy
	// internal config with the internet baseline unless the caller explicitly
	// requested migration (?migrate=external) — that would widen scope without an
	// explicit act.
	cfg := buildConfigForMode(req.Mode, req.Config)

	persist := true
	if project.PipelineConfig != nil && *project.PipelineConfig != "" {
		var saved models.PipelineConfig
		if json.Unmarshal([]byte(*project.PipelineConfig), &saved) == nil && scanconfig.IsLegacyInternalConfig(saved) {
			if r.URL.Query().Get("migrate") != "external" {
				persist = false
				log.Printf("[scan] preserved saved legacy internal config for project %s; pass ?migrate=external to persist the internet baseline", projectID)
			}
		}
	}
	// Persist config to project so the ScanModal can reload it next time.
	// Side effects (save failure) are non-fatal — the pipeline still runs.
	if persist {
		if cfgJSON, err := json.Marshal(cfg); err == nil {
			if err := s.queries.UpdateProjectPipelineConfig(projectID, string(cfgJSON)); err != nil {
				log.Printf("[scan] persist pipeline config for project %s: %v", projectID, err)
			}
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
		if proj, _ := s.queries.GetProject(projectID); proj != nil && proj.ScopeBoundaryMode == string(models.ScopeBoundaryStrict) {
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

