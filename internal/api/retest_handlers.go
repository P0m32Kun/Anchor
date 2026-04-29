package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/errors"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// POST /findings/{id}/retest
func (s *Server) handleRetestFinding(w http.ResponseWriter, r *http.Request) {
	findingID := r.PathValue("id")

	// Get finding
	finding, err := s.queries.GetFinding(findingID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "get finding: %v", err))
		return
	}
	if finding == nil {
		writeError(w, http.StatusNotFound, errors.New(errors.ErrNotFound, "finding not found"))
		return
	}

	// Create retest run record
	retest := &models.RetestRun{
		ID:        util.GenerateID(),
		FindingID: findingID,
		Result:    models.RetestInconclusive,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.queries.CreateRetestRun(retest); err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "create retest: %v", err))
		return
	}

	writeJSON(w, http.StatusCreated, retest)
}

// GET /findings/{id}/retests
func (s *Server) handleListRetests(w http.ResponseWriter, r *http.Request) {
	findingID := r.PathValue("id")
	retests, err := s.queries.ListRetestRunsByFinding(findingID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "list retests: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, retests)
}

// PATCH /findings/batch-status
func (s *Server) handleBatchUpdateFindingStatus(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IDs    []string `json:"ids"`
		Status string   `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "invalid body"))
		return
	}

	if len(req.IDs) == 0 {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "ids required"))
		return
	}
	if len(req.IDs) > 1000 {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "max 1000 ids"))
		return
	}
	validStatuses := map[string]bool{
		"pending_review": true,
		"confirmed":      true,
		"false_positive": true,
		"accepted_risk":  true,
		"resolved":       true,
	}
	if !validStatuses[req.Status] {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "invalid status"))
		return
	}

	tx, err := s.rawDB.Begin()
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "begin tx: %v", err))
		return
	}
	defer tx.Rollback()

	txQueries := db.New(tx)
	now := time.Now().UTC()
	for _, id := range req.IDs {
		if err := txQueries.UpdateFindingStatus(id, models.FindingStatus(req.Status), now); err != nil {
			writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "update finding %s: %v", id, err))
			return
		}
	}

	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "commit tx: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"updated": len(req.IDs),
	})
}

// GET /findings/{id}/curl
func (s *Server) handleGetFindingCurl(w http.ResponseWriter, r *http.Request) {
	findingID := r.PathValue("id")
	finding, err := s.queries.GetFinding(findingID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "get finding: %v", err))
		return
	}
	if finding == nil {
		writeError(w, http.StatusNotFound, errors.New(errors.ErrNotFound, "finding not found"))
		return
	}

	curl := ""
	if finding.RawRequest != "" {
		curl = "# curl command placeholder\n" + finding.RawRequest
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(curl))
}
