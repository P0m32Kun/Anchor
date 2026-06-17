package api

import (
	"net/http"
)

const scanDetailDefaultPageSize = 50

func parseScanDetailPagination(r *http.Request) PaginationParams {
	pg := parsePagination(r)
	if r.URL.Query().Get("page_size") == "" {
		pg.PageSize = scanDetailDefaultPageSize
	}
	return pg
}

// handleListScanRunWorks returns paginated work items for a pipeline run.
// GET /projects/{id}/pipeline/runs/{runId}/works?page=1&page_size=50
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

	pg := parseScanDetailPagination(r)
	total, err := s.queries.CountScanWorkItemsByRun(runID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	works, err := s.queries.ListScanWorkItemsByRunPaginated(runID, pg.PageSize, pg.Offset())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items":     works,
		"total":     total,
		"page":      pg.Page,
		"page_size": pg.PageSize,
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

// handleListToolCallLogs returns paginated tool call logs for a pipeline run.
// GET /projects/{id}/pipeline/runs/{runId}/tool-calls?page=1&page_size=50
func (s *Server) handleListToolCallLogs(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	runID := r.PathValue("runId")
	if runID == "" {
		http.Error(w, "runId required", http.StatusBadRequest)
		return
	}
	if _, ok := s.requireRunInProject(w, r, projectID, runID); !ok {
		return
	}

	pg := parseScanDetailPagination(r)
	total, err := s.queries.CountToolCallLogsByRun(runID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	logs, err := s.queries.ListToolCallLogsByRunPaginated(runID, pg.PageSize, pg.Offset())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items":     logs,
		"total":     total,
		"page":      pg.Page,
		"page_size": pg.PageSize,
	})
}

// handleGetFindingTrace returns the full trace chain for a finding.
// GET /findings/{findingId}/trace
func (s *Server) handleGetFindingTrace(w http.ResponseWriter, r *http.Request) {
	findingID := r.PathValue("findingId")
	if findingID == "" {
		http.Error(w, "findingId required", http.StatusBadRequest)
		return
	}

	trace, err := s.queries.GetToolCallTraceByFinding(findingID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if trace == nil || trace.Finding == nil {
		http.Error(w, "finding not found", http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, trace)
}
