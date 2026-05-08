package api

import (
	"encoding/json"
	"net/http"

	"github.com/P0m32Kun/Anchor/internal/errors"
	"github.com/P0m32Kun/Anchor/internal/service"
)

func (s *Server) handleListFindings(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	status := r.URL.Query().Get("status")
	page := parsePagination(r)

	result, err := s.findingSvc.ListPaginated(r.Context(), projectID, status, service.PaginationParams{Page: page.Page, PageSize: page.PageSize})
	if s.handleServiceError(w, err, "list findings failed") {
		return
	}
	writePaginatedJSON(w, result.Data, result.Total, page)
}

func (s *Server) handleGetFinding(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	finding, err := s.findingSvc.Get(r.Context(), id)
	if s.handleServiceError(w, err, "get finding failed") {
		return
	}
	writeJSON(w, http.StatusOK, finding)
}

func (s *Server) handlePatchFindingStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "invalid request body").WithDetail(err.Error()))
		return
	}

	if err := s.findingSvc.UpdateStatus(r.Context(), id, req.Status); err != nil {
		if s.handleServiceError(w, err, "update finding status failed") {
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": req.Status})
}

func (s *Server) handleAddEvidence(w http.ResponseWriter, r *http.Request) {
	findingID := r.PathValue("id")
	var req service.AddEvidenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "invalid request body").WithDetail(err.Error()))
		return
	}

	ev, err := s.findingSvc.AddEvidence(r.Context(), findingID, req)
	if s.handleServiceError(w, err, "create evidence failed") {
		return
	}
	writeJSON(w, http.StatusCreated, ev)
}
