package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/P0m32Kun/Anchor/internal/errors"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
	"github.com/P0m32Kun/Anchor/internal/workflow"
)

func (s *Server) handleRunPipeline(w http.ResponseWriter, r *http.Request) {
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

	runID := util.GenerateID()
	now := time.Now().UTC()
	if err := s.queries.CreatePipelineRun(&models.PipelineRun{
		ID:        runID,
		ProjectID: projectID,
		Status:    "running",
		StartedAt: now,
		CreatedAt: now,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("DB_ERROR", err.Error()))
		return
	}

	go func() {
		ctx := context.Background()
		pipeline := workflow.NewPipeline(s.queries, s.worker, s.scopeEng, s.dataDir).
			WithConfig(cfg).
			WithRunID(runID).
			WithStageCallback(func(rid string, stage workflow.StageID, status, errMsg string) {
				s.broadcastProjectSSE(projectID, map[string]interface{}{
					"event":   "pipeline_stage_change",
					"run_id":  rid,
					"stage":   string(stage),
					"status":  status,
					"error":   errMsg,
					"time":    time.Now().UTC().Format(time.RFC3339),
				})
			})
		if err := pipeline.Run(ctx, projectID); err != nil {
			log.Printf("pipeline run for project %s: %v", projectID, err)
		}
		// Broadcast final run status update.
		s.broadcastProjectSSE(projectID, map[string]interface{}{
			"event":  "pipeline_complete",
			"run_id": runID,
			"time":   time.Now().UTC().Format(time.RFC3339),
		})
	}()

	writeJSON(w, http.StatusAccepted, map[string]string{
		"status":  "accepted",
		"message": "Pipeline started",
		"run_id":  runID,
	})
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

// buildConfigForMode returns a PipelineConfig for external/internal scan modes.
// Speed parameters are loaded from the request body; defaults are applied for zero values.
func buildConfigForMode(mode string, cfg models.PipelineConfig) models.PipelineConfig {
	// Apply defaults for any zero speed values.
	defaults := models.DefaultPipelineConfig()
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

	switch mode {
	case "external":
		cfg.EnableFOFA = true
		cfg.EnableSubfinder = true
		cfg.EnableDNSx = true
		cfg.EnableCDNFilter = true
		cfg.EnableNerva = true
		cfg.EnableHttpx = true
		cfg.EnableNuclei = true
	case "internal":
		cfg.EnableFOFA = false
		cfg.EnableSubfinder = false
		cfg.EnableDNSx = false
		cfg.EnableCDNFilter = false
		cfg.EnableNerva = true
		cfg.EnableHttpx = true
		cfg.EnableNuclei = true
	}
	return cfg
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

	// Launch pipeline in background
	go func() {
		ctx := context.Background()
		pipeline := workflow.NewPipeline(s.queries, s.worker, s.scopeEng, s.dataDir).
			WithConfig(cfg).
			WithRunID(runID).
			WithStageCallback(func(rid string, stage workflow.StageID, status, errMsg string) {
				s.broadcastProjectSSE(projectID, map[string]interface{}{
					"event":   "pipeline_stage_change",
					"run_id":  rid,
					"stage":   string(stage),
					"status":  status,
					"error":   errMsg,
					"time":    time.Now().UTC().Format(time.RFC3339),
				})
			})

		if err := pipeline.Run(ctx, projectID); err != nil {
			log.Printf("scan run %s for project %s: %v", runID, projectID, err)
			s.queries.UpdatePipelineRunError(runID, err.Error())
		}
		completedAt := time.Now().UTC()
		s.queries.UpdatePipelineRunCompleted(runID, completedAt)

		s.broadcastProjectSSE(projectID, map[string]interface{}{
			"event":  "pipeline_complete",
			"run_id": runID,
			"time":   completedAt.Format(time.RFC3339),
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
