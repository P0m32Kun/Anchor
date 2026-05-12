package models

import "time"

// --- Service Fingerprint ---

type ServiceFingerprint struct {
	ID        string                 `json:"id"`
	ProjectID string                 `json:"project_id"`
	IP        string                 `json:"ip"`
	Port      int                    `json:"port"`
	Protocol  string                 `json:"protocol"`
	IsWeb     bool                   `json:"is_web"`
	Service   string                 `json:"service"`
	Product   string                 `json:"product,omitempty"`
	Version   string                 `json:"version,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Source    string                 `json:"source"`
	CreatedAt time.Time              `json:"created_at"`
}
