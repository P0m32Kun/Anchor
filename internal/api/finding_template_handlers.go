package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/errors"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

type findingTemplatePayload struct {
	SourceTool  *string   `json:"source_tool,omitempty"`
	MatchKeys   *[]string `json:"match_keys,omitempty"`
	Title       *string   `json:"title,omitempty"`
	Severity    *string   `json:"severity,omitempty"`
	Summary     *string   `json:"summary,omitempty"`
	Remediation *string   `json:"remediation,omitempty"`
	Enabled     *bool     `json:"enabled,omitempty"`
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
	if sourceTool == "" {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "source_tool is required"))
		return
	}

	var matchKeys []string
	if req.MatchKeys != nil {
		for _, k := range *req.MatchKeys {
			k = strings.TrimSpace(k)
			if k != "" {
				matchKeys = append(matchKeys, k)
			}
		}
	}
	if len(matchKeys) == 0 {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "match_keys is required (at least one non-empty key)"))
		return
	}

	// 应用层唯一性检查：同 source_tool 下不允许重复 match_key
	existingList, err := s.queries.ListFindingTemplates(sourceTool)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "list templates for duplicate check: %v", err))
		return
	}
	for _, existing := range existingList {
		for _, k := range matchKeys {
			if containsString(existing.MatchKeys, k) {
				writeError(w, http.StatusConflict, errors.Newf(errors.ErrConflict, "match_key '%s' already exists for tool '%s'", k, sourceTool))
				return
			}
		}
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
		MatchKey:    matchKeys[0], // 兼容老字段
		MatchKeys:   matchKeys,
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
	contentChanged := false
	if req.SourceTool != nil {
		v := strings.TrimSpace(*req.SourceTool)
		if v == "" {
			writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "source_tool cannot be empty"))
			return
		}
		if v != t.SourceTool {
			contentChanged = true
		}
		t.SourceTool = v
	}
	if req.MatchKeys != nil {
		var keys []string
		for _, k := range *req.MatchKeys {
			k = strings.TrimSpace(k)
			if k != "" {
				keys = append(keys, k)
			}
		}
		if len(keys) == 0 {
			writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "match_keys cannot be empty (at least one non-empty key)"))
			return
		}
		// 检查是否变更
		changed := len(keys) != len(t.MatchKeys)
		if !changed {
			for i, k := range keys {
				if k != t.MatchKeys[i] {
					changed = true
					break
				}
			}
		}
		if changed {
			contentChanged = true
		}
		t.MatchKeys = keys
		t.MatchKey = keys[0] // 兼容老字段
	}
	if req.Title != nil {
		v := strings.TrimSpace(*req.Title)
		if v != t.Title {
			contentChanged = true
		}
		t.Title = v
	}
	if req.Severity != nil {
		v := strings.TrimSpace(*req.Severity)
		if !validSeverity(v) {
			writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "severity must be info/low/medium/high/critical or empty"))
			return
		}
		if v != t.Severity {
			contentChanged = true
		}
		t.Severity = v
	}
	if req.Summary != nil {
		if *req.Summary != t.Summary {
			contentChanged = true
		}
		t.Summary = *req.Summary
	}
	if req.Remediation != nil {
		if *req.Remediation != t.Remediation {
			contentChanged = true
		}
		t.Remediation = *req.Remediation
	}
	if req.Enabled != nil {
		if *req.Enabled != t.Enabled {
			contentChanged = true
		}
		t.Enabled = *req.Enabled
	}
	// Builtin rows are locked from auto-overwrite as soon as a user edits them.
	if t.IsBuiltin && contentChanged {
		t.UserModified = true
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

// handleAcceptFindingTemplateUpstream overlays the row's builtin_payload onto
// its content fields and clears user_modified. Used when a user wants to drop
// their local edits and adopt the upstream (repo) version.
func (s *Server) handleAcceptFindingTemplateUpstream(w http.ResponseWriter, r *http.Request) {
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
	if !t.IsBuiltin || strings.TrimSpace(t.BuiltinPayload) == "" {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "this template has no upstream version to accept"))
		return
	}

	var seed db.SeedFindingTemplate
	if err := json.Unmarshal([]byte(t.BuiltinPayload), &seed); err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "decode builtin payload: %v", err))
		return
	}
	t.SourceTool = strings.TrimSpace(seed.SourceTool)
	t.MatchKey = strings.TrimSpace(seed.MatchKey)
	t.Title = strings.TrimSpace(seed.Title)
	t.Severity = strings.TrimSpace(seed.Severity)
	t.Summary = seed.Summary
	t.Remediation = seed.Remediation
	if seed.Enabled != nil {
		t.Enabled = *seed.Enabled
	} else {
		t.Enabled = true
	}
	t.UserModified = false
	t.UpdatedAt = time.Now().UTC()
	if err := s.queries.UpdateFindingTemplate(t); err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "apply upstream: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// handleExportFindingTemplates returns every template in the repo seed JSON
// shape, so team members can copy the array straight into
// docs/templates/vuln-templates.json.
func (s *Server) handleExportFindingTemplates(w http.ResponseWriter, r *http.Request) {
	list, err := s.queries.ListFindingTemplates("")
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "list finding templates: %v", err))
		return
	}
	seeds := make([]db.SeedFindingTemplate, 0, len(list))
	for _, t := range list {
		enabled := t.Enabled
		seeds = append(seeds, db.SeedFindingTemplate{
			SourceTool:  t.SourceTool,
			MatchKeys:   t.MatchKeys,
			Title:       t.Title,
			Severity:    t.Severity,
			Summary:     t.Summary,
			Remediation: t.Remediation,
			Enabled:     &enabled,
		})
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="vuln-templates.json"`)
	w.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(seeds)
}

func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
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
