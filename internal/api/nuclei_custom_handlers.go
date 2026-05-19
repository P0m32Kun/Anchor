package api

import (
	"encoding/json"
	stdErrors "errors"
	"io"
	"log"
	"net/http"
	"strings"

	apperrors "github.com/P0m32Kun/Anchor/internal/errors"
	"github.com/P0m32Kun/Anchor/internal/nuclei/custom"
)

// --- Nuclei Custom Sources ---

type createNucleiCustomGitRequest struct {
	Name          string `json:"name"`
	InstallPath   string `json:"install_path"`
	URI           string `json:"uri"`
	Branch        string `json:"branch,omitempty"`
	RoutingPolicy string `json:"routing_policy,omitempty"` // 可选，默认为 "manual"
}

type patchNucleiCustomSourceRequest struct {
	Name          *string `json:"name,omitempty"`
	Enabled       *bool   `json:"enabled,omitempty"`
	RoutingPolicy *string `json:"routing_policy,omitempty"`
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

func (s *Server) handleCreateNucleiCustomGitSource(w http.ResponseWriter, r *http.Request) {
	if s.nucleiCustomMgr == nil {
		writeError(w, http.StatusServiceUnavailable, apperrors.New(apperrors.ErrInternal, "nuclei custom manager not initialised"))
		return
	}

	var req createNucleiCustomGitRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, apperrors.New(apperrors.ErrBadRequest, "invalid request body").WithDetail(err.Error()))
		return
	}

	routingPolicy := req.RoutingPolicy
	if routingPolicy == "" {
		routingPolicy = "manual"
	}
	installPath := req.InstallPath
	if installPath == "" {
		installPath = req.Name
	}
	src, err := s.nucleiCustomMgr.CreateFromGit(r.Context(), req.Name, installPath, req.URI, req.Branch, routingPolicy)
	if err != nil {
		writeNucleiCustomError(w, err)
		return
	}

	// Auto-publish after creating source
	if _, pubErr := s.nucleiCustomMgr.Publish(); pubErr != nil {
		// Log publish error but don't fail the request - source was created successfully
		log.Printf("[api] auto-publish after git source creation: %v", pubErr)
	}

	writeJSON(w, http.StatusCreated, src)
}

func (s *Server) handleCreateNucleiCustomUploadSource(w http.ResponseWriter, r *http.Request) {
	if s.nucleiCustomMgr == nil {
		writeError(w, http.StatusServiceUnavailable, apperrors.New(apperrors.ErrInternal, "nuclei custom manager not initialised"))
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusBadRequest, apperrors.New(apperrors.ErrBadRequest, "failed to parse multipart form").WithDetail(err.Error()))
		return
	}

	name := r.FormValue("name")
	installPath := r.FormValue("install_path")
	if installPath == "" {
		installPath = name
	}
	routingPolicy := r.FormValue("routing_policy")
	if routingPolicy == "" {
		routingPolicy = "manual"
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, apperrors.New(apperrors.ErrBadRequest, "missing file field").WithDetail(err.Error()))
		return
	}
	defer file.Close()

	src, err := s.nucleiCustomMgr.CreateFromUpload(r.Context(), name, installPath, routingPolicy, header.Filename, file)
	if err != nil {
		writeNucleiCustomError(w, err)
		return
	}

	// Auto-publish after uploading source
	if _, pubErr := s.nucleiCustomMgr.Publish(); pubErr != nil {
		// Log publish error but don't fail the request - source was created successfully
		log.Printf("[api] auto-publish after upload: %v", pubErr)
	}

	writeJSON(w, http.StatusCreated, src)
}

func (s *Server) handleRefreshNucleiCustomSource(w http.ResponseWriter, r *http.Request) {
	if s.nucleiCustomMgr == nil {
		writeError(w, http.StatusServiceUnavailable, apperrors.New(apperrors.ErrInternal, "nuclei custom manager not initialised"))
		return
	}
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, apperrors.New(apperrors.ErrBadRequest, "id is required"))
		return
	}
	src, err := s.nucleiCustomMgr.Refresh(r.Context(), id)
	if err != nil {
		writeNucleiCustomError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, src)
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

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, apperrors.New(apperrors.ErrBadRequest, "invalid request body").WithDetail(err.Error()))
		return
	}

	src, err := s.nucleiCustomMgr.GetByID(id)
	if err != nil {
		writeNucleiCustomError(w, err)
		return
	}
	if !src.Builtin {
		writeError(w, http.StatusForbidden, apperrors.New(apperrors.ErrForbidden, "only builtin nuclei sources support enable toggle"))
		return
	}

	updated, err := s.nucleiCustomMgr.UpdateEnabled(id, req.Enabled)
	if err != nil {
		writeNucleiCustomError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) handlePatchNucleiCustomSource(w http.ResponseWriter, r *http.Request) {
	if s.nucleiCustomMgr == nil {
		writeError(w, http.StatusServiceUnavailable, apperrors.New(apperrors.ErrInternal, "nuclei custom manager not initialised"))
		return
	}
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, apperrors.New(apperrors.ErrBadRequest, "id is required"))
		return
	}

	var req patchNucleiCustomSourceRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, apperrors.New(apperrors.ErrBadRequest, "invalid request body").WithDetail(err.Error()))
		return
	}

	src, err := s.nucleiCustomMgr.Patch(id, custom.SourcePatch{
		Name:          req.Name,
		Enabled:       req.Enabled,
		RoutingPolicy: req.RoutingPolicy,
	})
	if err != nil {
		writeNucleiCustomError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, src)
}

func (s *Server) handleDeleteNucleiCustomSource(w http.ResponseWriter, r *http.Request) {
	if s.nucleiCustomMgr == nil {
		writeError(w, http.StatusServiceUnavailable, apperrors.New(apperrors.ErrInternal, "nuclei custom manager not initialised"))
		return
	}
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, apperrors.New(apperrors.ErrBadRequest, "id is required"))
		return
	}
	if err := s.nucleiCustomMgr.Delete(r.Context(), id); err != nil {
		writeNucleiCustomError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListNucleiCustomFiles(w http.ResponseWriter, r *http.Request) {
	if s.nucleiCustomMgr == nil {
		writeError(w, http.StatusServiceUnavailable, apperrors.New(apperrors.ErrInternal, "nuclei custom manager not initialised"))
		return
	}
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, apperrors.New(apperrors.ErrBadRequest, "id is required"))
		return
	}
	files, err := s.nucleiCustomMgr.ListFiles(id)
	if err != nil {
		writeNucleiCustomError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, files)
}

func (s *Server) handleReadNucleiCustomFile(w http.ResponseWriter, r *http.Request) {
	if s.nucleiCustomMgr == nil {
		writeError(w, http.StatusServiceUnavailable, apperrors.New(apperrors.ErrInternal, "nuclei custom manager not initialised"))
		return
	}
	id, rel, ok := splitNucleiCustomFilePath(r)
	if !ok {
		writeError(w, http.StatusBadRequest, apperrors.New(apperrors.ErrBadRequest, "id and path are required"))
		return
	}
	data, err := s.nucleiCustomMgr.ReadFile(id, rel)
	if err != nil {
		writeNucleiCustomError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (s *Server) handleWriteNucleiCustomFile(w http.ResponseWriter, r *http.Request) {
	if s.nucleiCustomMgr == nil {
		writeError(w, http.StatusServiceUnavailable, apperrors.New(apperrors.ErrInternal, "nuclei custom manager not initialised"))
		return
	}
	id, rel, ok := splitNucleiCustomFilePath(r)
	if !ok {
		writeError(w, http.StatusBadRequest, apperrors.New(apperrors.ErrBadRequest, "id and path are required"))
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, custom.MaxWriteFileBytes)
	defer r.Body.Close()
	data, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, apperrors.New(apperrors.ErrValidation, "request body too large").WithDetail(err.Error()))
		return
	}

	if err := s.nucleiCustomMgr.WriteFile(id, rel, data); err != nil {
		writeNucleiCustomError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDeleteNucleiCustomFile(w http.ResponseWriter, r *http.Request) {
	if s.nucleiCustomMgr == nil {
		writeError(w, http.StatusServiceUnavailable, apperrors.New(apperrors.ErrInternal, "nuclei custom manager not initialised"))
		return
	}
	id, rel, ok := splitNucleiCustomFilePath(r)
	if !ok {
		writeError(w, http.StatusBadRequest, apperrors.New(apperrors.ErrBadRequest, "id and path are required"))
		return
	}
	if err := s.nucleiCustomMgr.DeleteFile(id, rel); err != nil {
		writeNucleiCustomError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// splitNucleiCustomFilePath extracts the source id and the {path...} wildcard
// segment from the request. Returns ok=false when either is empty.
func splitNucleiCustomFilePath(r *http.Request) (id, rel string, ok bool) {
	id = r.PathValue("id")
	rel = r.PathValue("path")
	rel = strings.TrimPrefix(rel, "/")
	if id == "" || rel == "" {
		return "", "", false
	}
	return id, rel, true
}

func (s *Server) handleDownloadNucleiCustomSourceBundle(w http.ResponseWriter, r *http.Request) {
	if s.nucleiCustomMgr == nil {
		writeError(w, http.StatusServiceUnavailable, apperrors.New(apperrors.ErrInternal, "nuclei custom manager not initialised"))
		return
	}
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, apperrors.New(apperrors.ErrBadRequest, "id is required"))
		return
	}

	// Get source info
	src, err := s.nucleiCustomMgr.GetByID(id)
	if err != nil {
		writeNucleiCustomError(w, err)
		return
	}

	// Build a bundle for just this source on-the-fly
	version, archivePath, err := s.nucleiCustomMgr.BuildSourceBundle(id)
	if err != nil {
		writeNucleiCustomError(w, err)
		return
	}

	log.Printf("[api] serving bundle for source %s (%s), version %s", src.Name, id, version)
	http.ServeFile(w, r, archivePath)
}

// --- Phase 2: Validation & Publishing ---

func (s *Server) handleValidateNucleiCustomSource(w http.ResponseWriter, r *http.Request) {
	if s.nucleiCustomMgr == nil {
		writeError(w, http.StatusServiceUnavailable, apperrors.New(apperrors.ErrInternal, "nuclei custom manager not initialised"))
		return
	}
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, apperrors.New(apperrors.ErrBadRequest, "id is required"))
		return
	}
	result, err := s.nucleiCustomMgr.ValidateSource(id)
	if err != nil {
		writeNucleiCustomError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleValidateAllNucleiCustom(w http.ResponseWriter, r *http.Request) {
	if s.nucleiCustomMgr == nil {
		writeError(w, http.StatusServiceUnavailable, apperrors.New(apperrors.ErrInternal, "nuclei custom manager not initialised"))
		return
	}
	results, err := s.nucleiCustomMgr.ValidateAll()
	if err != nil {
		writeNucleiCustomError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, results)
}

func (s *Server) handlePublishNucleiCustom(w http.ResponseWriter, r *http.Request) {
	if s.nucleiCustomMgr == nil {
		writeError(w, http.StatusServiceUnavailable, apperrors.New(apperrors.ErrInternal, "nuclei custom manager not initialised"))
		return
	}
	version, err := s.nucleiCustomMgr.Publish()
	if err != nil {
		writeNucleiCustomError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"version": version})
}

func (s *Server) handleGetNucleiCustomManifest(w http.ResponseWriter, r *http.Request) {
	if s.nucleiCustomMgr == nil {
		writeError(w, http.StatusServiceUnavailable, apperrors.New(apperrors.ErrInternal, "nuclei custom manager not initialised"))
		return
	}
	manifest, err := s.nucleiCustomMgr.GetActiveManifest()
	if err != nil {
		writeNucleiCustomError(w, err)
		return
	}
	if manifest == nil {
		writeError(w, http.StatusNotFound, apperrors.New(apperrors.ErrNotFound, "no active bundle"))
		return
	}
	writeJSON(w, http.StatusOK, manifest)
}

func (s *Server) handleDownloadNucleiCustomBundle(w http.ResponseWriter, r *http.Request) {
	if s.nucleiCustomMgr == nil {
		writeError(w, http.StatusServiceUnavailable, apperrors.New(apperrors.ErrInternal, "nuclei custom manager not initialised"))
		return
	}
	version := r.PathValue("version")
	if version == "" {
		writeError(w, http.StatusBadRequest, apperrors.New(apperrors.ErrBadRequest, "version is required"))
		return
	}
	bundle, err := s.nucleiCustomMgr.GetBundleManifest(version)
	if err != nil {
		writeNucleiCustomError(w, err)
		return
	}
	if bundle == nil {
		writeError(w, http.StatusNotFound, apperrors.New(apperrors.ErrNotFound, "bundle not found"))
		return
	}

	archivePath := s.nucleiCustomMgr.Layout().BundleArchivePath(version)
	http.ServeFile(w, r, archivePath)
}

// writeNucleiCustomError maps a Manager-layer error onto an HTTP response.
// AppError keeps its own status code; everything else becomes 500.
func writeNucleiCustomError(w http.ResponseWriter, err error) {
	var appErr *apperrors.AppError
	if stdErrors.As(err, &appErr) {
		writeError(w, appErr.StatusCode(), appErr)
		return
	}
	if stdErrors.Is(err, custom.ErrBuiltinReadOnly) {
		writeError(w, http.StatusForbidden, apperrors.New(apperrors.ErrForbidden, err.Error()))
		return
	}
	if stdErrors.Is(err, custom.ErrNotBuiltin) {
		writeError(w, http.StatusBadRequest, apperrors.New(apperrors.ErrBadRequest, err.Error()))
		return
	}
	writeError(w, http.StatusInternalServerError, apperrors.Newf(apperrors.ErrInternal, "%v", err))
}
