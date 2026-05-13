package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/P0m32Kun/Anchor/internal/errors"
	"github.com/P0m32Kun/Anchor/internal/models"
)

func (s *Server) handleListHttpxFingerprints(w http.ResponseWriter, r *http.Request) {
	if s.httpxFpMgr == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New(errors.ErrInternal, "httpx fingerprint manager not initialised"))
		return
	}
	fpType := r.URL.Query().Get("type")
	list, err := s.httpxFpMgr.List(fpType)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "list fingerprints: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleCreateHttpxFingerprint(w http.ResponseWriter, r *http.Request) {
	if s.httpxFpMgr == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New(errors.ErrInternal, "httpx fingerprint manager not initialised"))
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "failed to parse multipart form").WithDetail(err.Error()))
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	fpType := r.FormValue("type")
	description := r.FormValue("description")

	if name == "" {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "name is required"))
		return
	}
	if fpType == "" {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "type is required"))
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "file is required").WithDetail(err.Error()))
		return
	}
	defer file.Close()

	content, err := io.ReadAll(io.LimitReader(file, 10<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "failed to read file").WithDetail(err.Error()))
		return
	}
	_ = header

	f, err := s.httpxFpMgr.Create(name, description, models.HttpxFingerprintType(fpType), content)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "create fingerprint: %v", err))
		return
	}
	writeJSON(w, http.StatusCreated, f)
}

func (s *Server) handleGetHttpxFingerprint(w http.ResponseWriter, r *http.Request) {
	if s.httpxFpMgr == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New(errors.ErrInternal, "httpx fingerprint manager not initialised"))
		return
	}
	id := r.PathValue("id")
	f, err := s.httpxFpMgr.Get(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "get fingerprint: %v", err))
		return
	}
	if f == nil {
		writeError(w, http.StatusNotFound, errors.Newf(errors.ErrNotFound, "fingerprint %s not found", id))
		return
	}
	writeJSON(w, http.StatusOK, f)
}

func (s *Server) handlePatchHttpxFingerprint(w http.ResponseWriter, r *http.Request) {
	if s.httpxFpMgr == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New(errors.ErrInternal, "httpx fingerprint manager not initialised"))
		return
	}
	id := r.PathValue("id")

	var req struct {
		Name        *string                        `json:"name,omitempty"`
		Description *string                        `json:"description,omitempty"`
		Enabled     *bool                          `json:"enabled,omitempty"`
		Type        *models.HttpxFingerprintType   `json:"type,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "invalid request body").WithDetail(err.Error()))
		return
	}

	// Fetch existing fingerprint to get current values
	f, err := s.httpxFpMgr.Get(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "get fingerprint: %v", err))
		return
	}
	if f == nil {
		writeError(w, http.StatusNotFound, errors.Newf(errors.ErrNotFound, "fingerprint %s not found", id))
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
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "update fingerprint: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) handleDeleteHttpxFingerprint(w http.ResponseWriter, r *http.Request) {
	if s.httpxFpMgr == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New(errors.ErrInternal, "httpx fingerprint manager not initialised"))
		return
	}
	id := r.PathValue("id")
	if err := s.httpxFpMgr.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "delete fingerprint: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleReadHttpxFingerprintContent(w http.ResponseWriter, r *http.Request) {
	if s.httpxFpMgr == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New(errors.ErrInternal, "httpx fingerprint manager not initialised"))
		return
	}
	id := r.PathValue("id")
	data, err := s.httpxFpMgr.ReadContent(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "read content: %v", err))
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (s *Server) handleWriteHttpxFingerprintContent(w http.ResponseWriter, r *http.Request) {
	if s.httpxFpMgr == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New(errors.ErrInternal, "httpx fingerprint manager not initialised"))
		return
	}
	id := r.PathValue("id")
	content, err := io.ReadAll(io.LimitReader(r.Body, 10<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "failed to read body").WithDetail(err.Error()))
		return
	}
	f, err := s.httpxFpMgr.UpdateContent(id, content)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "update content: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, f)
}
