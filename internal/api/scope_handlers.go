package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/P0m32Kun/Anchor/internal/errors"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/scope"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// ParseScopeRequest 解析scope规则值的请求
type ParseScopeRequest struct {
	Value string `json:"value"`
}

// ParseScopeResponse 解析scope规则值的响应
type ParseScopeResponse struct {
	Rules []struct {
		Type  string `json:"type"`
		Value string `json:"value"`
	} `json:"rules"`
}

func (s *Server) handleParseScopeValue(w http.ResponseWriter, r *http.Request) {
	var req ParseScopeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "invalid request body").WithDetail(err.Error()))
		return
	}

	// 复用scope包的parseLine逻辑来解析单个值
	parsed := scope.ParseLine(req.Value)
	response := ParseScopeResponse{Rules: make([]struct {
		Type  string `json:"type"`
		Value string `json:"value"`
	}, 0, len(parsed))}

	for _, p := range parsed {
		response.Rules = append(response.Rules, struct {
			Type  string `json:"type"`
			Value string `json:"value"`
		}{
			Type:  string(p.Type),
			Value: p.Value,
		})
	}

	writeJSON(w, http.StatusOK, response)
}


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

func (s *Server) handleDeleteScopeRule(w http.ResponseWriter, r *http.Request) {
	ruleID := r.PathValue("id")
	if err := s.queries.DeleteScopeRule(ruleID); err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "delete scope rule failed: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
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
