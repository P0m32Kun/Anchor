package api

import (
	"encoding/json"
	"net/http"

	"github.com/P0m32Kun/Anchor/internal/errors"
	"github.com/P0m32Kun/Anchor/internal/service"
)

func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	var req service.CreateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "invalid request body").WithDetail(err.Error()))
		return
	}

	project, err := s.projectSvc.Create(r.Context(), req)
	if s.handleServiceError(w, err, "create project failed") {
		return
	}

	writeJSON(w, http.StatusCreated, project)
}

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	page := parsePagination(r)
	result, err := s.projectSvc.ListPaginated(r.Context(), service.PaginationParams{Page: page.Page, PageSize: page.PageSize})
	if s.handleServiceError(w, err, "list projects failed") {
		return
	}
	writePaginatedJSON(w, result.Data, result.Total, page)
}

func (s *Server) handleGetProject(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	project, err := s.projectSvc.Get(r.Context(), id)
	if s.handleServiceError(w, err, "get project failed") {
		return
	}
	writeJSON(w, http.StatusOK, project)
}

// GET /projects/{id}/watch-config
func (s *Server) handleGetWatchConfig(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	// ListWatchEnabledProjects only returns enabled ones; we need to handle both cases.
	// For a specific project, read watch fields via raw query.
	proj, err := s.queries.GetWatchProject(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "get watch config: %v", err))
		return
	}
	if proj == nil {
		writeError(w, http.StatusNotFound, errors.New(errors.ErrNotFound, "Project not found"))
		return
	}
	writeJSON(w, http.StatusOK, proj)
}

// PATCH /projects/{id}/watch-config
func (s *Server) handleUpdateWatchConfig(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		WatchEnabled        bool `json:"watch_enabled"`
		WatchIntervalHours  int  `json:"watch_interval_hours"`
		WatchPassiveOnly    bool `json:"watch_passive_only"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "invalid body"))
		return
	}
	if req.WatchIntervalHours <= 0 {
		req.WatchIntervalHours = 24
	}
	if err := s.queries.UpdateProjectWatchConfig(id, req.WatchEnabled, req.WatchIntervalHours, req.WatchPassiveOnly); err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "update watch config: %v", err))
		return
	}
	proj, _ := s.queries.GetWatchProject(id)
	writeJSON(w, http.StatusOK, proj)
}

func (s *Server) handleDeleteProject(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.projectSvc.Delete(r.Context(), id); err != nil {
		if s.handleServiceError(w, err, "delete project failed") {
			return
		}
	}
	// Immediately clean up workdir files on project deletion.
	s.cleanupProjectWorkdir(id)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
