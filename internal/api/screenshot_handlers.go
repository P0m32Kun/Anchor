package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/P0m32Kun/Anchor/internal/errors"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/screenshot"
)

func (s *Server) handleScreenshotCapture(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	project, err := s.queries.GetProject(projectID)
	if err != nil || project == nil {
		writeError(w, http.StatusNotFound, errors.New(errors.ErrNotFound, "project not found"))
		return
	}

	var req struct {
		URL     string `json:"url"`
		AssetID string `json:"asset_id"`
		TaskID  string `json:"task_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "invalid body"))
		return
	}
	if req.URL == "" {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "url is required"))
		return
	}

	mgr := screenshot.NewManager(s.queries, s.dataDir)

	var assetIDPtr *string
	if req.AssetID != "" {
		assetIDPtr = &req.AssetID
	}
	var taskIDPtr *string
	if req.TaskID != "" {
		taskIDPtr = &req.TaskID
	}

	ss, err := mgr.CaptureAndStore(r.Context(), projectID, req.URL, assetIDPtr, taskIDPtr, 30*time.Second)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "capture screenshot failed: %v", err))
		return
	}

	writeJSON(w, http.StatusCreated, ss)
}

func (s *Server) handleListScreenshots(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	mgr := screenshot.NewManager(s.queries, s.dataDir)

	screenshots, err := mgr.ListByProject(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "list screenshots failed: %v", err))
		return
	}
	if screenshots == nil {
		screenshots = []*models.Screenshot{}
	}
	writeJSON(w, http.StatusOK, screenshots)
}

func (s *Server) handleScreenshotFile(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	kind := r.PathValue("kind")

	ss, err := s.queries.GetScreenshot(id)
	if err != nil || ss == nil {
		writeError(w, http.StatusNotFound, errors.New(errors.ErrNotFound, "screenshot not found"))
		return
	}

	var targetPath string
	var contentType string

	switch kind {
	case "thumbnail":
		targetPath = ss.ThumbnailPath
		contentType = "image/jpeg"
	default:
		targetPath = ss.OriginalPath
		contentType = "image/png"
	}

	if targetPath == "" {
		writeError(w, http.StatusNotFound, errors.New(errors.ErrNotFound, "screenshot file not available"))
		return
	}

	data, err := os.ReadFile(targetPath)
	if err != nil {
		writeError(w, http.StatusNotFound, errors.Newf(errors.ErrNotFound, "screenshot file not found: %v", err))
		return
	}

	ext := filepath.Ext(targetPath)
	if ext == ".jpg" || ext == ".jpeg" {
		contentType = "image/jpeg"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (s *Server) handleScreenshotContent(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "path query parameter is required"))
		return
	}

	dataDir := s.dataDir
	if !strings.HasPrefix(filepath.Clean(path), filepath.Clean(dataDir)) {
		writeError(w, http.StatusForbidden, errors.New(errors.ErrForbidden, "access denied"))
		return
	}

	data, err := os.ReadFile(path)
	if err != nil {
		writeError(w, http.StatusNotFound, errors.Newf(errors.ErrNotFound, "file not found: %v", err))
		return
	}

	ext := strings.ToLower(filepath.Ext(path))
	contentType := "image/png"
	if ext == ".jpg" || ext == ".jpeg" {
		contentType = "image/jpeg"
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}
