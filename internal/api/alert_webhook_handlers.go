package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

func (s *Server) handleGetAlertWebhook(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	if projectID == "" {
		http.Error(w, "missing project id", http.StatusBadRequest)
		return
	}

	webhook, err := s.queries.GetAlertWebhook(projectID)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"enabled": false, "url": "",
		})
		return
	}

	resp := map[string]interface{}{
		"id":                webhook.ID,
		"project_id":        webhook.ProjectID,
		"enabled":           webhook.Enabled,
		"url":               webhook.URL,
		"has_secret":        webhook.Secret != "",
		"min_severity":      webhook.MinSeverity,
		"on_new_asset":      webhook.OnNewAsset,
		"on_asset_gone":     webhook.OnAssetGone,
		"on_port_change":    webhook.OnPortChange,
		"on_service_change": webhook.OnServiceChange,
		"on_cert_expiry":    webhook.OnCertExpiry,
		"created_at":        webhook.CreatedAt,
		"updated_at":        webhook.UpdatedAt,
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleUpsertAlertWebhook(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	if projectID == "" {
		http.Error(w, "missing project id", http.StatusBadRequest)
		return
	}

	var req struct {
		Enabled         bool   `json:"enabled"`
		URL             string `json:"url"`
		Secret          string `json:"secret"`
		MinSeverity     string `json:"min_severity"`
		OnNewAsset      *bool  `json:"on_new_asset"`
		OnAssetGone     *bool  `json:"on_asset_gone"`
		OnPortChange    *bool  `json:"on_port_change"`
		OnServiceChange *bool  `json:"on_service_change"`
		OnCertExpiry    *bool  `json:"on_cert_expiry"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	existing, err := s.queries.GetAlertWebhook(projectID)
	webhook := &models.AlertWebhook{
		ID:              util.GenerateID(),
		ProjectID:       projectID,
		Enabled:         req.Enabled,
		URL:             req.URL,
		Secret:          req.Secret,
		MinSeverity:     "medium",
		OnNewAsset:      true,
		OnAssetGone:     true,
		OnPortChange:    true,
		OnServiceChange: true,
		OnCertExpiry:    true,
	}
	if err == nil && existing != nil {
		webhook.ID = existing.ID
		webhook.CreatedAt = existing.CreatedAt
		if req.MinSeverity != "" {
			webhook.MinSeverity = req.MinSeverity
		} else {
			webhook.MinSeverity = existing.MinSeverity
		}
		if req.OnNewAsset == nil {
			webhook.OnNewAsset = existing.OnNewAsset
		}
		if req.OnAssetGone == nil {
			webhook.OnAssetGone = existing.OnAssetGone
		}
		if req.OnPortChange == nil {
			webhook.OnPortChange = existing.OnPortChange
		}
		if req.OnServiceChange == nil {
			webhook.OnServiceChange = existing.OnServiceChange
		}
		if req.OnCertExpiry == nil {
			webhook.OnCertExpiry = existing.OnCertExpiry
		}
		if req.Secret == "" {
			webhook.Secret = existing.Secret
		}
	} else {
		if req.MinSeverity != "" {
			webhook.MinSeverity = req.MinSeverity
		}
		if req.OnNewAsset != nil {
			webhook.OnNewAsset = *req.OnNewAsset
		}
		if req.OnAssetGone != nil {
			webhook.OnAssetGone = *req.OnAssetGone
		}
		if req.OnPortChange != nil {
			webhook.OnPortChange = *req.OnPortChange
		}
		if req.OnServiceChange != nil {
			webhook.OnServiceChange = *req.OnServiceChange
		}
		if req.OnCertExpiry != nil {
			webhook.OnCertExpiry = *req.OnCertExpiry
		}
	}

	if err := s.queries.UpsertAlertWebhook(webhook); err != nil {
		log.Printf("[api] upsert alert webhook: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":         webhook.ID,
		"project_id": webhook.ProjectID,
		"enabled":    webhook.Enabled,
		"url":        webhook.URL,
	})
}

func (s *Server) handleDeleteAlertWebhook(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	if projectID == "" {
		http.Error(w, "missing project id", http.StatusBadRequest)
		return
	}

	if err := s.queries.DeleteAlertWebhook(projectID); err != nil {
		log.Printf("[api] delete alert webhook: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
