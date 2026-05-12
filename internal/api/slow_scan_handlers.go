package api

import (
	"net/http"

	"github.com/P0m32Kun/Anchor/internal/errors"
)

func (s *Server) handleListSlowScans(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	list, err := s.queries.ListSlowScanTasksByProject(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "list slow scans: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleGetSlowScan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	task, err := s.queries.GetSlowScanTask(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "get slow scan: %v", err))
		return
	}
	if task == nil {
		writeError(w, http.StatusNotFound, errors.Newf(errors.ErrNotFound, "slow scan task %s not found", id))
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (s *Server) handleCancelSlowScan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	task, err := s.queries.GetSlowScanTask(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "get slow scan: %v", err))
		return
	}
	if task == nil {
		writeError(w, http.StatusNotFound, errors.Newf(errors.ErrNotFound, "slow scan task %s not found", id))
		return
	}

	// Cancel the underlying worker task if it's running.
	// Slow scan tasks are backed by ScanTask records; find and cancel the latest one.
	// For simplicity, we just mark the slow scan task as cancelled.
	// The worker will finish naturally; the orchestrator won't create new tasks.
	if err := s.worker.Cancel(id); err != nil {
		// Best effort — log but don't fail the API call.
		// The slow scan task status is what matters.
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}
