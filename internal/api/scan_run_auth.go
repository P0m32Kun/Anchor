package api

import (
	"net/http"

	"github.com/P0m32Kun/Anchor/internal/errors"
	"github.com/P0m32Kun/Anchor/internal/models"
)

// requireRunInProject verifies the pipeline run belongs to the path project.
func (s *Server) requireRunInProject(w http.ResponseWriter, _ *http.Request, projectID, runID string) (*models.PipelineRun, bool) {
	run, err := s.queries.GetPipelineRun(runID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("DB_ERROR", err.Error()))
		return nil, false
	}
	if run == nil || run.ProjectID != projectID {
		writeError(w, http.StatusNotFound, errors.New("NOT_FOUND", "run not found"))
		return nil, false
	}
	return run, true
}
