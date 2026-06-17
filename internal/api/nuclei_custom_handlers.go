package api

import (
	"encoding/json"
	stdErrors "errors"
	"net/http"

	apperrors "github.com/P0m32Kun/Anchor/internal/errors"
	"github.com/P0m32Kun/Anchor/internal/nuclei/custom"
)

type patchNucleiCustomEnabledRequest struct {
	Enabled bool `json:"enabled"`
}

func (s *Server) handleListNucleiCustomSources(w http.ResponseWriter, r *http.Request) {
	if s.nucleiCustomMgr == nil {
		writeError(w, http.StatusServiceUnavailable, apperrors.New(apperrors.ErrInternal, "nuclei custom manager not initialised"))
		return
	}
	list, err := s.nucleiCustomMgr.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, apperrors.Newf(apperrors.ErrInternal, "list sources: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handlePatchNucleiCustomSourceEnabled(w http.ResponseWriter, r *http.Request) {
	if s.nucleiCustomMgr == nil {
		writeError(w, http.StatusServiceUnavailable, apperrors.New(apperrors.ErrInternal, "nuclei custom manager not initialised"))
		return
	}
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, apperrors.New(apperrors.ErrBadRequest, "id is required"))
		return
	}

	var req patchNucleiCustomEnabledRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, apperrors.New(apperrors.ErrBadRequest, "invalid request body").WithDetail(err.Error()))
		return
	}

	updated, err := s.nucleiCustomMgr.UpdateEnabled(id, req.Enabled)
	if err != nil {
		writeNucleiCustomError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func writeNucleiCustomError(w http.ResponseWriter, err error) {
	var appErr *apperrors.AppError
	if stdErrors.As(err, &appErr) {
		writeError(w, appErr.StatusCode(), appErr)
		return
	}
	if stdErrors.Is(err, custom.ErrNotBuiltin) {
		writeError(w, http.StatusForbidden, apperrors.New(apperrors.ErrForbidden, err.Error()))
		return
	}
	writeError(w, http.StatusInternalServerError, apperrors.Newf(apperrors.ErrInternal, "%v", err))
}
