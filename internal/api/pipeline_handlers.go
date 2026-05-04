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
		if project.FofaEmail != nil && project.FofaAPIKey != nil && *project.FofaEmail != "" && *project.FofaAPIKey != "" {
			pipeline.WithFOFA(*project.FofaEmail, *project.FofaAPIKey)
		}
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

// ScanMode represents a predefined scan configuration
type ScanMode string

const (
	ScanModeQuick    ScanMode = "quick"
	ScanModeStandard ScanMode = "standard"
	ScanModeDeep     ScanMode = "deep"
	ScanModeCustom   ScanMode = "custom"
)

// buildConfigForMode returns a PipelineConfig preset for the given scan mode
func buildConfigForMode(mode ScanMode, base models.PipelineConfig) models.PipelineConfig {
	switch mode {
	case ScanModeQuick:
		return models.PipelineConfig{
			EnableFOFA:          false,
			EnableSubfinder:     false,
			EnableCDNFilter:     false,
			PortRange:           "top-1000",
			PortScanTimeout:     10,
			PortScanConcurrency: 25,
			EnableNerva:         false,
			EnableNuclei:        false,
		}
	case ScanModeStandard:
		return models.PipelineConfig{
			EnableFOFA:          false,
			EnableSubfinder:     true,
			SubfinderTimeout:    30,
			DNSConcurrency:      50,
			DNSTimeout:          5,
			EnableCDNFilter:     false,
			PortRange:           "top-1000",
			PortScanTimeout:     10,
			PortScanConcurrency: 25,
			EnableNerva:         true,
			NervaTimeout:        10,
			NervaConcurrency:    10,
			EnableNuclei:        true,
			NucleiRateLimit:     150,
			NucleiConcurrency:   25,
		}
	case ScanModeDeep:
		return models.PipelineConfig{
			EnableFOFA:          base.EnableFOFA,
			FofaResultLimit:     base.FofaResultLimit,
			FofaConcurrency:     base.FofaConcurrency,
			EnableSubfinder:     true,
			SubfinderTimeout:    60,
			DNSConcurrency:      100,
			DNSTimeout:          10,
			EnableCDNFilter:     true,
			PortRange:           base.PortRange,
			PortScanTimeout:     15,
			PortScanConcurrency: 50,
			EnableNerva:         true,
			NervaTimeout:        15,
			NervaConcurrency:    20,
			EnableNuclei:        true,
			NucleiRateLimit:     300,
			NucleiConcurrency:   50,
		}
	case ScanModeCustom:
		return base
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
		Mode ScanMode `json:"mode"`
		Name string   `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New("INVALID_BODY", "Invalid request body"))
		return
	}

	if req.Mode == "" {
		req.Mode = ScanModeStandard
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

	// Build config based on mode
	baseConfig := models.DefaultPipelineConfig()
	if project.PipelineConfig != nil && *project.PipelineConfig != "" {
		json.Unmarshal([]byte(*project.PipelineConfig), &baseConfig)
	}
	cfg := buildConfigForMode(req.Mode, baseConfig)

	// Create pipeline run with mode
	runID := util.GenerateID()
	now := time.Now().UTC()
	if err := s.queries.CreatePipelineRun(&models.PipelineRun{
		ID:        runID,
		ProjectID: projectID,
		Mode:      string(req.Mode),
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

		// Set FOFA credentials if deep mode and configured
		if req.Mode == ScanModeDeep && project.FofaEmail != nil && project.FofaAPIKey != nil &&
			*project.FofaEmail != "" && *project.FofaAPIKey != "" {
			pipeline.WithFOFA(*project.FofaEmail, *project.FofaAPIKey)
		}

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
		"mode":   string(req.Mode),
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
