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
			WithRunID(runID)
		if project.FofaEmail != nil && project.FofaAPIKey != nil && *project.FofaEmail != "" && *project.FofaAPIKey != "" {
			pipeline.WithFOFA(*project.FofaEmail, *project.FofaAPIKey)
		}
		if err := pipeline.Run(ctx, projectID); err != nil {
			log.Printf("pipeline run for project %s: %v", projectID, err)
		}
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
