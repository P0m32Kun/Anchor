package api

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

func (s *Server) handleCreateNotificationChannel(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	if projectID == "" {
		http.Error(w, "missing project id", http.StatusBadRequest)
		return
	}

	var req struct {
		Name        string `json:"name"`
		ChannelType string `json:"channel_type"`
		URL         string `json:"url"`
		Enabled     *bool  `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if req.URL == "" {
		http.Error(w, "url is required", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		req.Name = "Webhook"
	}
	if req.ChannelType == "" {
		req.ChannelType = models.NotificationChannelTypeWebhook
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	ch := &models.NotificationChannel{
		ID:          util.GenerateID(),
		ProjectID:   projectID,
		Name:        req.Name,
		ChannelType: req.ChannelType,
		URL:         req.URL,
		Enabled:     enabled,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if err := s.queries.CreateNotificationChannel(ch); err != nil {
		log.Printf("[api] create notification channel: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, ch)
}

func (s *Server) handleListNotificationChannels(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	if projectID == "" {
		http.Error(w, "missing project id", http.StatusBadRequest)
		return
	}

	channels, err := s.queries.ListNotificationChannelsByProject(projectID)
	if err != nil {
		log.Printf("[api] list notification channels: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if channels == nil {
		channels = []*models.NotificationChannel{}
	}

	writeJSON(w, http.StatusOK, channels)
}

func (s *Server) handleUpdateNotificationChannel(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	channelID := r.PathValue("channelId")
	if projectID == "" || channelID == "" {
		http.Error(w, "missing project or channel id", http.StatusBadRequest)
		return
	}

	var req struct {
		Name    *string `json:"name"`
		URL     *string `json:"url"`
		Enabled *bool   `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	ch, err := s.queries.GetNotificationChannelByID(channelID, projectID)
	if err != nil {
		http.Error(w, "channel not found", http.StatusNotFound)
		return
	}

	if req.Name != nil {
		ch.Name = *req.Name
	}
	if req.URL != nil {
		ch.URL = *req.URL
	}
	if req.Enabled != nil {
		ch.Enabled = *req.Enabled
	}

	if err := s.queries.UpdateNotificationChannel(ch); err != nil {
		log.Printf("[api] update notification channel: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, ch)
}

func (s *Server) handleDeleteNotificationChannel(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	channelID := r.PathValue("channelId")
	if projectID == "" || channelID == "" {
		http.Error(w, "missing project or channel id", http.StatusBadRequest)
		return
	}

	if err := s.queries.DeleteNotificationChannel(channelID, projectID); err != nil {
		log.Printf("[api] delete notification channel: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
