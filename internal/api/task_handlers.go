package api

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/P0m32Kun/Anchor/internal/errors"
	"github.com/P0m32Kun/Anchor/internal/models"
)

func (s *Server) handleGetTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	task, err := s.queries.GetScanTask(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "get task failed: %v", err))
		return
	}
	if task == nil {
		writeError(w, http.StatusNotFound, errors.New(errors.ErrNotFound, "task not found"))
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (s *Server) handleCancelTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	now := time.Now().UTC()
	_ = s.queries.UpdateScanTaskStatus(id, models.TaskCancelled, nil, &now)
	_ = s.worker.Cancel(id)
	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

func (s *Server) handleListArtifacts(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	artifacts, err := s.queries.ListRawArtifactsByTask(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "list artifacts failed: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, artifacts)
}

func (s *Server) handleGetArtifactContent(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "missing artifact id"))
		return
	}
	artifact, err := s.queries.GetRawArtifact(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "get artifact failed: %v", err))
		return
	}
	if artifact == nil {
		writeError(w, http.StatusNotFound, errors.Newf(errors.ErrNotFound, "artifact not found: %s", id))
		return
	}

	data, err := os.ReadFile(artifact.Path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "read artifact file failed: %v", err))
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write(data)
}

// handleGetArtifactContentRange returns a byte range of an artifact.
// Query params: id (artifact ID), offset (default 0), limit (default 64KB, max 1MB).
func (s *Server) handleGetArtifactContentRange(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "missing artifact id"))
		return
	}
	artifact, err := s.queries.GetRawArtifact(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "get artifact failed: %v", err))
		return
	}
	if artifact == nil {
		writeError(w, http.StatusNotFound, errors.Newf(errors.ErrNotFound, "artifact not found: %s", id))
		return
	}

	offset := parseIntParam(r.URL.Query().Get("offset"), 0)
	limit := parseIntParam(r.URL.Query().Get("limit"), 65536) // 64KB default
	const maxLimit = 1048576                                   // 1MB max
	if limit > maxLimit {
		limit = maxLimit
	}

	f, err := os.Open(artifact.Path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "open artifact: %v", err))
		return
	}
	defer f.Close()

	if offset > 0 {
		if _, err := f.Seek(int64(offset), io.SeekStart); err != nil {
			writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "seek artifact: %v", err))
			return
		}
	}

	buf := make([]byte, limit)
	n, err := io.ReadFull(f, buf)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "read artifact: %v", err))
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", offset, offset+n-1, artifact.Size))
	w.Write(buf[:n])
}

// parseIntParam parses an integer query param with a default fallback.
func parseIntParam(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	val := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return defaultVal
		}
		val = val*10 + int(c-'0')
	}
	return val
}
