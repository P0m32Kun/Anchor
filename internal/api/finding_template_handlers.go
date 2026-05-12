package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/P0m32Kun/Anchor/internal/errors"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

type findingTemplatePayload struct {
	SourceTool  *string `json:"source_tool,omitempty"`
	MatchKey    *string `json:"match_key,omitempty"`
	Title       *string `json:"title,omitempty"`
	Severity    *string `json:"severity,omitempty"`
	Summary     *string `json:"summary,omitempty"`
	Remediation *string `json:"remediation,omitempty"`
	Enabled     *bool   `json:"enabled,omitempty"`
}

func validSeverity(s string) bool {
	switch s {
	case "", "info", "low", "medium", "high", "critical":
		return true
	}
	return false
}

func (s *Server) handleListFindingTemplates(w http.ResponseWriter, r *http.Request) {
	sourceTool := r.URL.Query().Get("source_tool")
	list, err := s.queries.ListFindingTemplates(sourceTool)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "list finding templates: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleCreateFindingTemplate(w http.ResponseWriter, r *http.Request) {
	var req findingTemplatePayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "invalid request body").WithDetail(err.Error()))
		return
	}

	sourceTool := strings.TrimSpace(deref(req.SourceTool))
	matchKey := strings.TrimSpace(deref(req.MatchKey))
	if sourceTool == "" {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "source_tool is required"))
		return
	}
	if matchKey == "" {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "match_key is required"))
		return
	}
	severity := strings.TrimSpace(deref(req.Severity))
	if !validSeverity(severity) {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "severity must be info/low/medium/high/critical or empty"))
		return
	}

	now := time.Now().UTC()
	t := &models.FindingTemplate{
		ID:          util.GenerateID(),
		SourceTool:  sourceTool,
		MatchKey:    matchKey,
		Title:       strings.TrimSpace(deref(req.Title)),
		Severity:    severity,
		Summary:     deref(req.Summary),
		Remediation: deref(req.Remediation),
		Enabled:     derefBool(req.Enabled, true),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.queries.CreateFindingTemplate(t); err != nil {
		writeError(w, http.StatusConflict, errors.Newf(errors.ErrInternal, "create finding template (duplicate match key?): %v", err))
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

func (s *Server) handleGetFindingTemplate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	t, err := s.queries.GetFindingTemplate(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "get finding template: %v", err))
		return
	}
	if t == nil {
		writeError(w, http.StatusNotFound, errors.Newf(errors.ErrNotFound, "finding template %s not found", id))
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (s *Server) handlePatchFindingTemplate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	t, err := s.queries.GetFindingTemplate(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "get finding template: %v", err))
		return
	}
	if t == nil {
		writeError(w, http.StatusNotFound, errors.Newf(errors.ErrNotFound, "finding template %s not found", id))
		return
	}

	var req findingTemplatePayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "invalid request body").WithDetail(err.Error()))
		return
	}
	if req.SourceTool != nil {
		v := strings.TrimSpace(*req.SourceTool)
		if v == "" {
			writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "source_tool cannot be empty"))
			return
		}
		t.SourceTool = v
	}
	if req.MatchKey != nil {
		v := strings.TrimSpace(*req.MatchKey)
		if v == "" {
			writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "match_key cannot be empty"))
			return
		}
		t.MatchKey = v
	}
	if req.Title != nil {
		t.Title = strings.TrimSpace(*req.Title)
	}
	if req.Severity != nil {
		v := strings.TrimSpace(*req.Severity)
		if !validSeverity(v) {
			writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "severity must be info/low/medium/high/critical or empty"))
			return
		}
		t.Severity = v
	}
	if req.Summary != nil {
		t.Summary = *req.Summary
	}
	if req.Remediation != nil {
		t.Remediation = *req.Remediation
	}
	if req.Enabled != nil {
		t.Enabled = *req.Enabled
	}
	t.UpdatedAt = time.Now().UTC()

	if err := s.queries.UpdateFindingTemplate(t); err != nil {
		writeError(w, http.StatusConflict, errors.Newf(errors.ErrInternal, "update finding template (duplicate match key?): %v", err))
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (s *Server) handleDeleteFindingTemplate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.queries.DeleteFindingTemplate(id); err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "delete finding template: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func derefBool(p *bool, def bool) bool {
	if p == nil {
		return def
	}
	return *p
}
