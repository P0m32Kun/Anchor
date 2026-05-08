package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/P0m32Kun/Anchor/internal/errors"
)

// --- Health ---

func (s *Server) handleToolHealth(w http.ResponseWriter, r *http.Request) {
	h, err := s.queries.ListToolHealth()
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "list tool health failed: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, h)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	if err := s.health.CheckAll(r.Context(), s.dataDir); err != nil {
		log.Printf("health check error: %v", err)
	}
	s.handleToolHealth(w, r)
}

// --- Helpers ---

func (s *Server) handleListWorkers(w http.ResponseWriter, r *http.Request) {
	workers := []map[string]interface{}{}

	dbWorkers, err := s.queries.ListWorkerNodes()
	if err == nil {
		for _, w := range dbWorkers {
			workers = append(workers, map[string]interface{}{
				"id":       w.ID,
				"name":     w.Name,
				"mode":     w.Mode,
				"status":   w.Status,
				"endpoint": w.Endpoint,
			})
		}
	}

	writeJSON(w, http.StatusOK, workers)
}

func (s *Server) handleListToolTemplates(w http.ResponseWriter, r *http.Request) {
	templates, err := s.queries.ListToolTemplates()
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.New(errors.ErrInternal, err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, templates)
}

func (s *Server) handleGetToolTemplate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	template, err := s.queries.GetToolTemplate(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.New(errors.ErrInternal, err.Error()))
		return
	}
	if template == nil {
		writeError(w, http.StatusNotFound, errors.New(errors.ErrNotFound, "tool-template not found: "+id))
		return
	}
	writeJSON(w, http.StatusOK, template)
}

func defaultToolTimeout(tool string) time.Duration {
	switch strings.ToLower(tool) {
	case "subfinder":
		return 300 * time.Second
	case "httpx":
		return 300 * time.Second
	case "naabu":
		return 600 * time.Second
	case "nuclei":
		return 1800 * time.Second
	case "nmap":
		return 600 * time.Second
	default:
		return 300 * time.Second
	}
}

// handleServiceError handles errors returned by service layer methods.
// It returns true if an error was written to the response (caller should return).
func (s *Server) handleServiceError(w http.ResponseWriter, err error, fallbackMsg string) bool {
	if err == nil {
		return false
	}
	if appErr, ok := err.(*errors.AppError); ok {
		writeError(w, appErr.StatusCode(), appErr)
		return true
	}
	writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "%s: %v", fallbackMsg, err))
	return true
}

func redactArgs(cmd string) string {
	parts := strings.Fields(cmd)
	for i, p := range parts {
		if strings.Contains(strings.ToLower(p), "key") ||
			strings.Contains(strings.ToLower(p), "token") ||
			strings.Contains(strings.ToLower(p), "secret") ||
			strings.Contains(strings.ToLower(p), "password") {
			parts[i] = "[REDACTED]"
		}
	}
	return strings.Join(parts, " ")
}
