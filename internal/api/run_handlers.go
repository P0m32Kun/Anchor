package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/P0m32Kun/Anchor/internal/errors"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// POST /projects/{id}/runs
func (s *Server) handleCreateRun(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	var req struct {
		ToolTemplateID string `json:"tool_template_id"`
		Name           string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "invalid body"))
		return
	}

	if req.Name == "" {
		req.Name = "未命名扫描"
	}

	run := &models.Run{
		ID:        util.GenerateID(),
		ProjectID: projectID,
		Name:      req.Name,
		Status:    models.RunPending,
		CreatedAt: time.Now().UTC(),
	}
	if req.ToolTemplateID != "" {
		run.ToolTemplateID = &req.ToolTemplateID
	}

	if err := s.queries.CreateRun(run); err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "create run: %v", err))
		return
	}

	writeJSON(w, http.StatusCreated, run)
}

// GET /projects/{id}/runs
func (s *Server) handleListRuns(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	runs, err := s.queries.ListRunsByProject(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "list runs: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, runs)
}
