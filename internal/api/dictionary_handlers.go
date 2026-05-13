package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/P0m32Kun/Anchor/internal/errors"
	"github.com/P0m32Kun/Anchor/internal/models"
)

func (s *Server) handleListDictionaries(w http.ResponseWriter, r *http.Request) {
	if s.dictMgr == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New(errors.ErrInternal, "dictionary manager not initialised"))
		return
	}
	category := r.URL.Query().Get("category")
	list, err := s.dictMgr.List(category)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "list dictionaries: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleCreateDictionary(w http.ResponseWriter, r *http.Request) {
	if s.dictMgr == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New(errors.ErrInternal, "dictionary manager not initialised"))
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "failed to parse multipart form").WithDetail(err.Error()))
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	category := r.FormValue("category")
	description := r.FormValue("description")

	if name == "" {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "name is required"))
		return
	}
	if category == "" {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "category is required"))
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

	d, err := s.dictMgr.Create(name, description, models.DictionaryCategory(category), content)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "create dictionary: %v", err))
		return
	}
	writeJSON(w, http.StatusCreated, d)
}

func (s *Server) handleGetDictionary(w http.ResponseWriter, r *http.Request) {
	if s.dictMgr == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New(errors.ErrInternal, "dictionary manager not initialised"))
		return
	}
	id := r.PathValue("id")
	d, err := s.dictMgr.Get(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "get dictionary: %v", err))
		return
	}
	if d == nil {
		writeError(w, http.StatusNotFound, errors.Newf(errors.ErrNotFound, "dictionary %s not found", id))
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func (s *Server) handlePatchDictionary(w http.ResponseWriter, r *http.Request) {
	if s.dictMgr == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New(errors.ErrInternal, "dictionary manager not initialised"))
		return
	}
	id := r.PathValue("id")
	if !s.requireUserDictionary(w, id) {
		return
	}

	var req struct {
		Name        string `json:"name,omitempty"`
		Description string `json:"description,omitempty"`
		Category    string `json:"category,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "invalid request body").WithDetail(err.Error()))
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "name is required"))
		return
	}
	if req.Category == "" {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "category is required"))
		return
	}

	d, err := s.dictMgr.Update(id, req.Name, req.Description, models.DictionaryCategory(req.Category))
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "update dictionary: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func (s *Server) handleDeleteDictionary(w http.ResponseWriter, r *http.Request) {
	if s.dictMgr == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New(errors.ErrInternal, "dictionary manager not initialised"))
		return
	}
	id := r.PathValue("id")
	if !s.requireUserDictionary(w, id) {
		return
	}
	if err := s.dictMgr.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "delete dictionary: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleReadDictionaryContent(w http.ResponseWriter, r *http.Request) {
	if s.dictMgr == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New(errors.ErrInternal, "dictionary manager not initialised"))
		return
	}
	id := r.PathValue("id")
	data, err := s.dictMgr.ReadContent(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "read content: %v", err))
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (s *Server) handleWriteDictionaryContent(w http.ResponseWriter, r *http.Request) {
	if s.dictMgr == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New(errors.ErrInternal, "dictionary manager not initialised"))
		return
	}
	id := r.PathValue("id")
	if !s.requireUserDictionary(w, id) {
		return
	}
	content, err := io.ReadAll(io.LimitReader(r.Body, 10<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "failed to read body").WithDetail(err.Error()))
		return
	}
	d, err := s.dictMgr.UpdateContent(id, content)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "update content: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, d)
}

// requireUserDictionary blocks write/delete requests against builtin dictionaries.
// Writes the HTTP response and returns false if the request was rejected.
// Returns true if the caller should proceed. When the dictionary lookup fails,
// the response is written and false is returned — the downstream handler should
// simply return without further action.
func (s *Server) requireUserDictionary(w http.ResponseWriter, id string) bool {
	d, err := s.dictMgr.Get(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "get dictionary: %v", err))
		return false
	}
	if d == nil {
		writeError(w, http.StatusNotFound, errors.Newf(errors.ErrNotFound, "dictionary %s not found", id))
		return false
	}
	if d.Builtin {
		writeError(w, http.StatusForbidden, errors.New(errors.ErrForbidden, "builtin dictionary is read-only"))
		return false
	}
	return true
}
