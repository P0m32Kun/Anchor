package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/P0m32Kun/Anchor/internal/errors"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/search"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// --- Engine Credentials ---

type saveEngineCredentialRequest struct {
	Engine string  `json:"engine"`
	APIKey string  `json:"api_key"`
	Extra  *string `json:"extra,omitempty"`
}

func maskKey(key string) string {
	if len(key) > 4 {
		return key[:4] + "****"
	}
	return key
}

func (s *Server) handleListEngineCredentials(w http.ResponseWriter, r *http.Request) {
	creds, err := s.queries.ListEngineCredentials()
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "list credentials: %v", err))
		return
	}

	for _, c := range creds {
		c.APIKey = maskKey(c.APIKey)
	}

	writeJSON(w, http.StatusOK, creds)
}

func (s *Server) handleSaveEngineCredential(w http.ResponseWriter, r *http.Request) {
	var req saveEngineCredentialRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "invalid request body"))
		return
	}

	if req.Engine == "" || req.APIKey == "" {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "engine and api_key are required"))
		return
	}

	now := time.Now().UTC()
	cred := &models.EngineCredential{
		ID:        util.GenerateID(),
		Engine:    req.Engine,
		APIKey:    req.APIKey,
		Extra:     req.Extra,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.queries.SaveEngineCredential(cred); err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "save credential: %v", err))
		return
	}

	cred.APIKey = maskKey(cred.APIKey)
	writeJSON(w, http.StatusOK, cred)
}

func (s *Server) handleDeleteEngineCredential(w http.ResponseWriter, r *http.Request) {
	engine := r.PathValue("engine")
	if engine == "" {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "engine is required"))
		return
	}

	if err := s.queries.DeleteEngineCredential(engine); err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "delete credential: %v", err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- Engine Search ---

type searchEngineResponse struct {
	Engine string                `json:"engine"`
	Query  string                `json:"query"`
	Total  int                   `json:"total"`
	Data   []search.SearchResult `json:"data"`
}

func (s *Server) handleSearchEngine(w http.ResponseWriter, r *http.Request) {
	engine := r.URL.Query().Get("engine")
	query := r.URL.Query().Get("query")
	if engine == "" || query == "" {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "engine and query are required"))
		return
	}

	page := parseIntQuery(r, "page", 1)
	size := parseIntQuery(r, "size", 20)
	if size > 100 {
		size = 100
	}

	cred, err := s.queries.GetEngineCredential(engine)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "get credential: %v", err))
		return
	}
	if cred == nil {
		writeError(w, http.StatusBadRequest, errors.Newf(errors.ErrBadRequest, "no API key configured for %s", engine))
		return
	}

	var results []search.SearchResult
	switch engine {
	case "fofa":
		client := search.NewFofaClient(cred.APIKey)
		fofaResults, err := client.Search(r.Context(), query, page, size)
		if err != nil {
			writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "fofa search: %v", err))
			return
		}
		results = fofaResults
	case "hunter":
		client := search.NewHunterClient(cred.APIKey)
		hunterResults, err := client.Search(r.Context(), query, page, size)
		if err != nil {
			writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "hunter search: %v", err))
			return
		}
		results = hunterResults
	case "quake":
		client := search.NewQuakeClient(cred.APIKey)
		start := (page - 1) * size
		quakeResults, err := client.Search(r.Context(), query, start, size)
		if err != nil {
			writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "quake search: %v", err))
			return
		}
		results = quakeResults
	default:
		writeError(w, http.StatusBadRequest, errors.Newf(errors.ErrBadRequest, "unsupported engine: %s", engine))
		return
	}

	writeJSON(w, http.StatusOK, searchEngineResponse{
		Engine: engine,
		Query:  query,
		Total:  len(results),
		Data:   results,
	})
}

func parseIntQuery(r *http.Request, key string, defaultVal int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return defaultVal
	}
	var n int
	fmt.Sscanf(v, "%d", &n)
	if n <= 0 {
		return defaultVal
	}
	return n
}
