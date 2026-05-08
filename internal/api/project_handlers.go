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
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			writeError(w, appErr.StatusCode(), appErr)
			return
		}
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "list projects failed: %v", err))
		return
	}
	writePaginatedJSON(w, result.Data, result.Total, page)
}

func (s *Server) handleGetProject(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	project, err := s.projectSvc.Get(r.Context(), id)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			writeError(w, appErr.StatusCode(), appErr)
			return
		}
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "get project failed: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, project)
}

func (s *Server) handleDeleteProject(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.projectSvc.Delete(r.Context(), id); err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			writeError(w, appErr.StatusCode(), appErr)
			return
		}
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "delete project failed: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
