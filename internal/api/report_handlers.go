package api

import (
	"fmt"
	"net/http"

	"github.com/P0m32Kun/Anchor/internal/errors"
	"github.com/P0m32Kun/Anchor/internal/report"
)

func (s *Server) handleExportReportMD(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")

	project, err := s.queries.GetProject(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "get project failed: %v", err))
		return
	}
	if project == nil {
		writeError(w, http.StatusNotFound, errors.New(errors.ErrNotFound, "project not found"))
		return
	}

	data, err := report.Aggregate(r.Context(), s.queries, project)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "report aggregation failed: %v", err))
		return
	}

	md := report.GenerateMarkdown(data)
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=report_%s.md", projectID))
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(md))
}

func (s *Server) handleExportReportJSON(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")

	project, err := s.queries.GetProject(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "get project failed: %v", err))
		return
	}
	if project == nil {
		writeError(w, http.StatusNotFound, errors.New(errors.ErrNotFound, "project not found"))
		return
	}

	data, err := report.Aggregate(r.Context(), s.queries, project)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "report aggregation failed: %v", err))
		return
	}

	jsonData, err := report.GenerateJSON(data)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "json generation failed: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=report_%s.json", projectID))
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)
}
