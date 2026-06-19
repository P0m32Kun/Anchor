package api

import (
	"log"
	"net/http"
	"strconv"

	"github.com/P0m32Kun/Anchor/internal/models"
)

func (s *Server) handleListAssetChanges(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	if projectID == "" {
		http.Error(w, "missing project id", http.StatusBadRequest)
		return
	}

	limit := 50
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 500 {
			limit = v
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}

	changes, err := s.queries.ListAssetChangesByProject(projectID, limit, offset)
	if err != nil {
		log.Printf("[api] list asset changes: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if changes == nil {
		changes = []*models.AssetChange{}
	}

	writeJSON(w, http.StatusOK, changes)
}

func (s *Server) handleAssetChangeTimeline(w http.ResponseWriter, r *http.Request) {
	assetID := r.PathValue("id")
	if assetID == "" {
		http.Error(w, "missing asset id", http.StatusBadRequest)
		return
	}

	changes, err := s.queries.ListAssetChangesByAsset(assetID)
	if err != nil {
		log.Printf("[api] asset change timeline: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if changes == nil {
		changes = []*models.AssetChange{}
	}

	writeJSON(w, http.StatusOK, changes)
}
