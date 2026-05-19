package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	ancherrors "github.com/P0m32Kun/Anchor/internal/errors"
	"github.com/P0m32Kun/Anchor/internal/httpxfp"
	"github.com/P0m32Kun/Anchor/internal/models"
)

func (s *Server) handleListHttpxFingerprints(w http.ResponseWriter, r *http.Request) {
	if s.httpxFpMgr == nil {
		writeError(w, http.StatusServiceUnavailable, ancherrors.New(ancherrors.ErrInternal, "httpx fingerprint manager not initialised"))
		return
	}
	fpType := r.URL.Query().Get("type")
	list, err := s.httpxFpMgr.List(fpType)
	if err != nil {
		writeError(w, http.StatusInternalServerError, ancherrors.Newf(ancherrors.ErrInternal, "list fingerprints: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleCreateHttpxFingerprint(w http.ResponseWriter, r *http.Request) {
	if s.httpxFpMgr == nil {
		writeError(w, http.StatusServiceUnavailable, ancherrors.New(ancherrors.ErrInternal, "httpx fingerprint manager not initialised"))
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusBadRequest, ancherrors.New(ancherrors.ErrBadRequest, "failed to parse multipart form").WithDetail(err.Error()))
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	fpType := r.FormValue("type")
	description := r.FormValue("description")

	if name == "" {
		writeError(w, http.StatusBadRequest, ancherrors.New(ancherrors.ErrBadRequest, "name is required"))
		return
	}
	if fpType == "" {
		writeError(w, http.StatusBadRequest, ancherrors.New(ancherrors.ErrBadRequest, "type is required"))
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, ancherrors.New(ancherrors.ErrBadRequest, "file is required").WithDetail(err.Error()))
		return
	}
	defer file.Close()

	content, err := io.ReadAll(io.LimitReader(file, 10<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, ancherrors.New(ancherrors.ErrBadRequest, "failed to read file").WithDetail(err.Error()))
		return
	}
	_ = header

	f, err := s.httpxFpMgr.Create(name, description, models.HttpxFingerprintType(fpType), content)
	if err != nil {
		writeError(w, http.StatusInternalServerError, ancherrors.Newf(ancherrors.ErrInternal, "create fingerprint: %v", err))
		return
	}
	writeJSON(w, http.StatusCreated, f)
}

func (s *Server) handleGetHttpxFingerprint(w http.ResponseWriter, r *http.Request) {
	if s.httpxFpMgr == nil {
		writeError(w, http.StatusServiceUnavailable, ancherrors.New(ancherrors.ErrInternal, "httpx fingerprint manager not initialised"))
		return
	}
	id := r.PathValue("id")
	f, err := s.httpxFpMgr.Get(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, ancherrors.Newf(ancherrors.ErrInternal, "get fingerprint: %v", err))
		return
	}
	if f == nil {
		writeError(w, http.StatusNotFound, ancherrors.Newf(ancherrors.ErrNotFound, "fingerprint %s not found", id))
		return
	}
	writeJSON(w, http.StatusOK, f)
}

func (s *Server) handlePatchHttpxFingerprint(w http.ResponseWriter, r *http.Request) {
	if s.httpxFpMgr == nil {
		writeError(w, http.StatusServiceUnavailable, ancherrors.New(ancherrors.ErrInternal, "httpx fingerprint manager not initialised"))
		return
	}
	id := r.PathValue("id")
	if !s.requireUserHttpxFingerprint(w, id) {
		return
	}

	var req struct {
		Name        *string                        `json:"name,omitempty"`
		Description *string                        `json:"description,omitempty"`
		Enabled     *bool                          `json:"enabled,omitempty"`
		Type        *models.HttpxFingerprintType   `json:"type,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ancherrors.New(ancherrors.ErrBadRequest, "invalid request body").WithDetail(err.Error()))
		return
	}

	// Fetch existing fingerprint to get current values
	f, err := s.httpxFpMgr.Get(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, ancherrors.Newf(ancherrors.ErrInternal, "get fingerprint: %v", err))
		return
	}
	if f == nil {
		writeError(w, http.StatusNotFound, ancherrors.Newf(ancherrors.ErrNotFound, "fingerprint %s not found", id))
		return
	}

	// Use provided values or fall back to existing ones
	name := f.Name
	description := f.Description
	enabled := f.Enabled
	fpType := f.Type

	if req.Name != nil {
		name = *req.Name
	}
	if req.Description != nil {
		description = *req.Description
	}
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	if req.Type != nil {
		fpType = *req.Type
	}

	updated, err := s.httpxFpMgr.Update(id, name, description, fpType, enabled)
	if err != nil {
		writeHttpxFpMgrError(w, err, "update fingerprint")
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) handlePatchHttpxFingerprintEnabled(w http.ResponseWriter, r *http.Request) {
	if s.httpxFpMgr == nil {
		writeError(w, http.StatusServiceUnavailable, ancherrors.New(ancherrors.ErrInternal, "httpx fingerprint manager not initialised"))
		return
	}
	id := r.PathValue("id")

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ancherrors.New(ancherrors.ErrBadRequest, "invalid request body").WithDetail(err.Error()))
		return
	}

	updated, err := s.httpxFpMgr.UpdateEnabled(id, req.Enabled)
	if err != nil {
		writeHttpxFpMgrError(w, err, "update fingerprint enabled")
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) handleDeleteHttpxFingerprint(w http.ResponseWriter, r *http.Request) {
	if s.httpxFpMgr == nil {
		writeError(w, http.StatusServiceUnavailable, ancherrors.New(ancherrors.ErrInternal, "httpx fingerprint manager not initialised"))
		return
	}
	id := r.PathValue("id")
	if !s.requireUserHttpxFingerprint(w, id) {
		return
	}
	if err := s.httpxFpMgr.Delete(r.Context(), id); err != nil {
		writeHttpxFpMgrError(w, err, "delete fingerprint")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleReadHttpxFingerprintContent(w http.ResponseWriter, r *http.Request) {
	if s.httpxFpMgr == nil {
		writeError(w, http.StatusServiceUnavailable, ancherrors.New(ancherrors.ErrInternal, "httpx fingerprint manager not initialised"))
		return
	}
	id := r.PathValue("id")
	data, err := s.httpxFpMgr.ReadContent(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, ancherrors.Newf(ancherrors.ErrInternal, "read content: %v", err))
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (s *Server) handleWriteHttpxFingerprintContent(w http.ResponseWriter, r *http.Request) {
	if s.httpxFpMgr == nil {
		writeError(w, http.StatusServiceUnavailable, ancherrors.New(ancherrors.ErrInternal, "httpx fingerprint manager not initialised"))
		return
	}
	id := r.PathValue("id")
	if !s.requireUserHttpxFingerprint(w, id) {
		return
	}
	content, err := io.ReadAll(io.LimitReader(r.Body, 10<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, ancherrors.New(ancherrors.ErrBadRequest, "failed to read body").WithDetail(err.Error()))
		return
	}
	f, err := s.httpxFpMgr.UpdateContent(id, content)
	if err != nil {
		writeHttpxFpMgrError(w, err, "update content")
		return
	}
	writeJSON(w, http.StatusOK, f)
}

// requireUserHttpxFingerprint blocks write/delete on builtin fingerprints.
func (s *Server) requireUserHttpxFingerprint(w http.ResponseWriter, id string) bool {
	f, err := s.httpxFpMgr.Get(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, ancherrors.Newf(ancherrors.ErrInternal, "get fingerprint: %v", err))
		return false
	}
	if f == nil {
		writeError(w, http.StatusNotFound, ancherrors.Newf(ancherrors.ErrNotFound, "fingerprint %s not found", id))
		return false
	}
	if f.Builtin {
		writeError(w, http.StatusForbidden, ancherrors.New(ancherrors.ErrForbidden, "builtin fingerprint is read-only"))
		return false
	}
	return true
}

func writeHttpxFpMgrError(w http.ResponseWriter, err error, action string) {
	switch {
	case errors.Is(err, httpxfp.ErrBuiltinReadOnly):
		writeError(w, http.StatusForbidden, ancherrors.New(ancherrors.ErrForbidden, err.Error()))
	case errors.Is(err, httpxfp.ErrNotBuiltin):
		writeError(w, http.StatusBadRequest, ancherrors.New(ancherrors.ErrBadRequest, err.Error()))
	default:
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, ancherrors.Newf(ancherrors.ErrNotFound, "%s: %v", action, err))
			return
		}
		writeError(w, http.StatusInternalServerError, ancherrors.Newf(ancherrors.ErrInternal, "%s: %v", action, err))
	}
}
