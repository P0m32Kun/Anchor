package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/P0m32Kun/Anchor/internal/errors"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

func (s *Server) handleRunTask(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProjectID string `json:"project_id"`
		PlanID    string `json:"plan_id"`
		Tool      string `json:"tool"`
		TargetID  string `json:"target_id"`
		Command   string `json:"command"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "invalid request body").WithDetail(err.Error()))
		return
	}

	project, err := s.queries.GetProject(req.ProjectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "get project failed: %v", err))
		return
	}
	if project == nil {
		writeError(w, http.StatusNotFound, errors.New(errors.ErrNotFound, "project not found"))
		return
	}
	if project.RateLimit < 0 {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "rate_limit must be >= 0"))
		return
	}

	if req.TargetID != "" {
		target, err := s.queries.GetTarget(req.TargetID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "get target failed: %v", err))
			return
		}
		if target == nil {
			writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "target not found"))
			return
		}
	}

	planID := req.PlanID
	if planID == "" {
		plan := &models.ScanPlan{
			ID:           util.GenerateID(),
			ProjectID:    req.ProjectID,
			WorkflowType: "ad-hoc",
			Profile:      models.ProfileStandard,
			Status:       "approved",
			CreatedBy:    "user",
			CreatedAt:    time.Now().UTC(),
		}
		if err := s.queries.CreateScanPlan(plan); err != nil {
			writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "create default plan failed: %v", err))
			return
		}
		planID = plan.ID
	}

	task := &models.ScanTask{
		ID:                util.GenerateID(),
		ProjectID:         req.ProjectID,
		PlanID:            planID,
		TargetID:          nil,
		Tool:              req.Tool,
		CommandTemplate:   req.Command,
		ArgumentsRedacted: redactArgs(req.Command),
		Status:            models.TaskQueued,
		CreatedAt:         time.Now().UTC(),
	}
	if req.TargetID != "" {
		task.TargetID = &req.TargetID
	}
	if err := s.queries.CreateScanTask(task); err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "create scan task failed: %v", err))
		return
	}

	timeout := defaultToolTimeout(task.Tool)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		if err := s.worker.Run(ctx, task.ID); err != nil {
			log.Printf("task %s failed: %v", task.ID, err)
			if ctx.Err() == context.DeadlineExceeded {
				now := time.Now().UTC()
				exitCode := -1
				_ = s.queries.UpdateScanTaskStatus(task.ID, models.TaskFailed, &exitCode, &now)
			}
		}
		s.broadcastProjectSSE(task.ProjectID, map[string]interface{}{
			"event":   "task_update",
			"task_id": task.ID,
		})
	}()

	writeJSON(w, http.StatusCreated, task)
}

func (s *Server) handleGetTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	task, err := s.queries.GetScanTask(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "get task failed: %v", err))
		return
	}
	if task == nil {
		writeError(w, http.StatusNotFound, errors.New(errors.ErrNotFound, "task not found"))
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (s *Server) handleCancelTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	now := time.Now().UTC()
	_ = s.queries.UpdateScanTaskStatus(id, models.TaskCancelled, nil, &now)
	_ = s.worker.Cancel(id)
	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

func (s *Server) handleListArtifacts(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	artifacts, err := s.queries.ListRawArtifactsByTask(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "list artifacts failed: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, artifacts)
}

func (s *Server) handleGetArtifactContent(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "missing artifact id"))
		return
	}
	artifact, err := s.queries.GetRawArtifact(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "get artifact failed: %v", err))
		return
	}
	if artifact == nil {
		writeError(w, http.StatusNotFound, errors.Newf(errors.ErrNotFound, "artifact not found: %s", id))
		return
	}

	data, err := os.ReadFile(artifact.Path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "read artifact file failed: %v", err))
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write(data)
}
