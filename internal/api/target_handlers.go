package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/P0m32Kun/Anchor/internal/errors"
	"github.com/P0m32Kun/Anchor/internal/scope"
	"github.com/P0m32Kun/Anchor/internal/service"
)

func (s *Server) handleCreateTarget(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")

	var req service.CreateTargetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "invalid request body").WithDetail(err.Error()))
		return
	}

	resp, err := s.targetSvc.Create(r.Context(), projectID, req)
	if s.handleServiceError(w, err, "create target failed") {
		return
	}

	if resp.NeedsScopeConfirmation {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"needs_scope_confirmation": true,
			"message":                  resp.Message,
			"suggested_rule":           resp.SuggestedRule,
		})
		return
	}

	writeJSON(w, http.StatusCreated, resp.Target)
}

func (s *Server) handleListTargets(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	page := parsePagination(r)
	result, err := s.targetSvc.ListPaginated(r.Context(), projectID, service.PaginationParams{Page: page.Page, PageSize: page.PageSize})
	if s.handleServiceError(w, err, "list targets failed") {
		return
	}
	writePaginatedJSON(w, result.Data, result.Total, page)
}

func (s *Server) handleImportTargets(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "failed to parse multipart form").WithDetail(err.Error()))
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "missing file field").WithDetail(err.Error()))
		return
	}
	defer file.Close()

	var parsed []scope.ImportTarget
	name := strings.ToLower(header.Filename)
	if strings.HasSuffix(name, ".csv") {
		parsed, err = scope.ParseCSV(file)
	} else {
		parsed, err = scope.ParseTXT(file)
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrParse, "failed to parse file").WithDetail(err.Error()))
		return
	}

	if len(parsed) == 0 {
		writeJSON(w, http.StatusOK, service.ImportResult{})
		return
	}

	seen := make(map[string]bool)
	var targets []service.ImportTarget
	for _, t := range parsed {
		if t.Value == "" {
			continue
		}
		if seen[t.Value] {
			continue
		}
		seen[t.Value] = true
		targets = append(targets, service.ImportTarget{Type: t.Type, Value: t.Value})
	}

	result, err := s.targetSvc.Import(r.Context(), projectID, targets)
	if s.handleServiceError(w, err, "import targets failed") {
		return
	}

	writeJSON(w, http.StatusOK, result)
}
