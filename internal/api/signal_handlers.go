package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/P0m32Kun/Anchor/internal/models"
)

func (s *Server) handleListSignals(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	if projectID == "" {
		http.Error(w, "missing project id", http.StatusBadRequest)
		return
	}

	signals, err := s.queries.ListSignalsByProject(projectID)
	if err != nil {
		log.Printf("[api] list signals: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	statusFilter := r.URL.Query().Get("status")
	severityFilter := r.URL.Query().Get("severity")
	sourceKindFilter := r.URL.Query().Get("source_kind")

	if statusFilter != "" || severityFilter != "" || sourceKindFilter != "" {
		filtered := make([]*models.Signal, 0)
		for _, sig := range signals {
			if statusFilter != "" && sig.Status != statusFilter {
				continue
			}
			if severityFilter != "" && sig.Severity != severityFilter {
				continue
			}
			if sourceKindFilter != "" && sig.SourceKind != sourceKindFilter {
				continue
			}
			filtered = append(filtered, sig)
		}
		signals = filtered
	}

	if signals == nil {
		signals = []*models.Signal{}
	}

	writeJSON(w, http.StatusOK, signals)
}

func (s *Server) handleUpdateSignalStatus(w http.ResponseWriter, r *http.Request) {
	signalID := r.PathValue("id")
	if signalID == "" {
		http.Error(w, "missing signal id", http.StatusBadRequest)
		return
	}

	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	if req.Status != models.SignalStatusNew && req.Status != models.SignalStatusAcknowledged && req.Status != models.SignalStatusResolved {
		http.Error(w, "invalid status; must be new, acknowledged, or resolved", http.StatusBadRequest)
		return
	}

	if err := s.queries.UpdateSignalStatus(signalID, req.Status); err != nil {
		log.Printf("[api] update signal status: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleBatchUpdateSignalStatus(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IDs    []string `json:"ids"`
		Status string   `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	if req.Status != models.SignalStatusNew && req.Status != models.SignalStatusAcknowledged && req.Status != models.SignalStatusResolved {
		http.Error(w, "invalid status", http.StatusBadRequest)
		return
	}

	if len(req.IDs) == 0 {
		http.Error(w, "ids is required", http.StatusBadRequest)
		return
	}

	for _, id := range req.IDs {
		if err := s.queries.UpdateSignalStatus(id, req.Status); err != nil {
			log.Printf("[api] batch update signal %s: %v", id, err)
		}
	}

	writeJSON(w, http.StatusOK, map[string]int{"updated": len(req.IDs)})
}

func (s *Server) handleSignalStats(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	if projectID == "" {
		http.Error(w, "missing project id", http.StatusBadRequest)
		return
	}

	signals, err := s.queries.ListSignalsByProject(projectID)
	if err != nil {
		log.Printf("[api] signal stats: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	stats := map[string]int{
		"total":        len(signals),
		"new":          0,
		"acknowledged": 0,
		"resolved":     0,
	}
	for _, sig := range signals {
		switch sig.Status {
		case models.SignalStatusNew:
			stats["new"]++
		case models.SignalStatusAcknowledged:
			stats["acknowledged"]++
		case models.SignalStatusResolved:
			stats["resolved"]++
		}
	}

	writeJSON(w, http.StatusOK, stats)
}

func (s *Server) handleSignalCount(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	if projectID == "" {
		http.Error(w, "missing project id", http.StatusBadRequest)
		return
	}

	var statusPtr *string
	if statusStr := r.URL.Query().Get("status"); statusStr != "" {
		statusPtr = &statusStr
	}

	count, err := s.queries.CountSignalsByProject(projectID, statusPtr)
	if err != nil {
		log.Printf("[api] count signals: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]int{"count": count})
}
