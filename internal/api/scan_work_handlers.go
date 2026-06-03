package api

import (
	"net/http"
)

// handleListScanRunWorks returns all work items for a pipeline run.
// GET /projects/{id}/pipeline/runs/{runId}/works
func (s *Server) handleListScanRunWorks(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	runID := r.PathValue("runId")
	if runID == "" {
		http.Error(w, "runId required", http.StatusBadRequest)
		return
	}
	if _, ok := s.requireRunInProject(w, r, projectID, runID); !ok {
		return
	}

	works, err := s.queries.ListScanWorkItemsByRun(runID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": works,
		"total": len(works),
	})
}

// handleListAssetWorks returns work items for a specific asset in a run.
// GET /assets/{id}/works?run_id=xxx
func (s *Server) handleListAssetWorks(w http.ResponseWriter, r *http.Request) {
	assetID := r.PathValue("id")
	if assetID == "" {
		http.Error(w, "asset id required", http.StatusBadRequest)
		return
	}

	runID := r.URL.Query().Get("run_id")
	if runID == "" {
		http.Error(w, "run_id query param required", http.StatusBadRequest)
		return
	}
	projectID := r.URL.Query().Get("project_id")
	if projectID != "" {
		if _, ok := s.requireRunInProject(w, r, projectID, runID); !ok {
			return
		}
	}

	works, err := s.queries.ListScanWorkItemsByAsset(runID, assetID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"asset_id": assetID,
		"run_id":   runID,
		"items":    works,
		"total":    len(works),
	})
}
