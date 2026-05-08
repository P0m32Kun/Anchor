package models

import "time"

// --- CDN ---

type CDNResult struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	IP        string    `json:"ip"`
	IsCDN     bool      `json:"is_cdn"`
	Provider  string    `json:"provider,omitempty"`
	Type      string    `json:"type,omitempty"` // cdn | waf | cloud
	CreatedAt time.Time `json:"created_at"`
}
