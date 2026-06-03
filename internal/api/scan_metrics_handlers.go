package api

import (
	"net/http"
)

// handleGetScanRunMetrics returns aggregated metrics for a pipeline run.
// GET /projects/{id}/pipeline/runs/{runId}/metrics
func (s *Server) handleGetScanRunMetrics(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	runID := r.PathValue("runId")
	if runID == "" {
		http.Error(w, "runId required", http.StatusBadRequest)
		return
	}
	if _, ok := s.requireRunInProject(w, r, projectID, runID); !ok {
		return
	}

	m, err := s.queries.GetScanRunMetrics(runID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if m == nil {
		http.Error(w, "run not found", http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, m)
}
