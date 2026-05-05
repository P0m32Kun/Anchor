package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/P0m32Kun/Anchor/internal/errors"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

func (s *Server) handleCreateScopeRule(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProjectID string `json:"project_id"`
		Action    string `json:"action"`
		Type      string `json:"type"`
		Value     string `json:"value"`
		Reason    string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "invalid request body").WithDetail(err.Error()))
		return
	}

	sr := &models.ScopeRule{
		ID:        util.GenerateID(),
		ProjectID: req.ProjectID,
		Action:    models.ScopeAction(req.Action),
		Type:      models.TargetType(req.Type),
		Value:     req.Value,
		Reason:    req.Reason,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := s.queries.CreateScopeRule(sr); err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "create scope rule failed: %v", err))
		return
	}

	writeJSON(w, http.StatusCreated, sr)
}

func (s *Server) handleListScopeRules(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")
	if projectID == "" {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "missing project_id"))
		return
	}
	page := parsePagination(r)
	total, err := s.queries.CountScopeRulesByProject(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "count scope rules failed: %v", err))
		return
	}
	rules, err := s.queries.ListScopeRulesByProjectPaginated(projectID, page.PageSize, page.Offset())
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "list scope rules failed: %v", err))
		return
	}
	writePaginatedJSON(w, rules, total, page)
}

func (s *Server) handleBatchCreateScopeRules(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")

	var req struct {
		Rules []struct {
			Action string `json:"action"`
			Type   string `json:"type"`
			Value  string `json:"value"`
			Reason string `json:"reason"`
		} `json:"rules"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "invalid request body").WithDetail(err.Error()))
		return
	}

	now := time.Now().UTC()
	var created []*models.ScopeRule
	for _, ruleReq := range req.Rules {
		if ruleReq.Action == "" {
			ruleReq.Action = "include"
		}
		sr := &models.ScopeRule{
			ID:        util.GenerateID(),
			ProjectID: projectID,
			Action:    models.ScopeAction(ruleReq.Action),
			Type:      models.TargetType(ruleReq.Type),
			Value:     ruleReq.Value,
			Reason:    ruleReq.Reason,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := s.queries.CreateScopeRule(sr); err != nil {
			writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "create scope rule failed: %v", err))
			return
		}
		created = append(created, sr)
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"created": len(created),
		"rules":   created,
	})
}

func (s *Server) handleCreateScanPlan(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProjectID    string `json:"project_id"`
		WorkflowType string `json:"workflow_type"`
		Profile      string `json:"profile"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "invalid request body").WithDetail(err.Error()))
		return
	}

	plan := &models.ScanPlan{
		ID:           util.GenerateID(),
		ProjectID:    req.ProjectID,
		WorkflowType: req.WorkflowType,
		Profile:      models.ScanProfile(req.Profile),
		Status:       "draft",
		CreatedBy:    "user",
		CreatedAt:    time.Now().UTC(),
	}
	if err := s.queries.CreateScanPlan(plan); err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "create scan plan failed: %v", err))
		return
	}

	writeJSON(w, http.StatusCreated, plan)
}

func (s *Server) handleApprovePlan(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "approved"})
}

func (s *Server) handleDryRun(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")
	if projectID == "" {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "missing project_id"))
		return
	}

	project, err := s.queries.GetProject(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "get project failed: %v", err))
		return
	}
	if project == nil {
		writeError(w, http.StatusNotFound, errors.New(errors.ErrNotFound, "project not found"))
		return
	}

	targets, err := s.queries.ListTargetsByProject(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "list targets failed: %v", err))
		return
	}

	timeWindowOK := true
	timeWindowReason := ""

	var results []map[string]interface{}
	allowCount := 0
	for _, t := range targets {
		decision, err := s.scopeEng.Check(r.Context(), projectID, t)
		if err != nil {
			results = append(results, map[string]interface{}{
				"target":   t.Value,
				"type":     t.Type,
				"decision": "error",
				"reason":   err.Error(),
			})
			continue
		}
		if decision.Decision == models.ScopeAllow {
			allowCount++
		}
		results = append(results, map[string]interface{}{
			"target":   t.Value,
			"type":     t.Type,
			"decision": decision.Decision,
			"reason":   decision.Reason,
		})
	}

	estimatedSeconds := estimateExecutionTime(len(targets), project.DefaultProfile)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"project_id":         projectID,
		"results":            results,
		"mode":               "dry-run",
		"time_window_valid":  timeWindowOK,
		"time_window_reason": timeWindowReason,
		"rate_limit":         project.RateLimit,
		"target_count":       len(targets),
		"allow_count":        allowCount,
		"estimated_seconds":  estimatedSeconds,
	})
}

func estimateExecutionTime(targetCount int, profile string) int {
	if targetCount == 0 {
		return 0
	}
	var perTarget int
	switch profile {
	case "light":
		perTarget = 30
	case "deep":
		perTarget = 120
	default:
		perTarget = 60
	}
	return targetCount * perTarget
}
