package models

import "time"

// NotificationChannel represents an external alert delivery channel.
type NotificationChannel struct {
	ID          string    `json:"id" db:"id"`
	ProjectID   string    `json:"project_id" db:"project_id"`
	Name        string    `json:"name" db:"name"`
	ChannelType string    `json:"channel_type" db:"channel_type"`
	URL         string    `json:"url" db:"url"`
	Enabled     bool      `json:"enabled" db:"enabled"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

const (
	NotificationChannelTypeWebhook = "webhook"
)

// NotificationPayload is the JSON payload sent to webhook endpoints.
type NotificationPayload struct {
	Event     string `json:"event"`
	ProjectID string `json:"project_id"`
	RunID     string `json:"run_id,omitempty"`
	Signal    *SignalNotification `json:"signal,omitempty"`
	Summary   *ScanSummaryNotification `json:"summary,omitempty"`
	Timestamp string `json:"timestamp"`
}

// SignalNotification is a single signal in notification payloads.
type SignalNotification struct {
	ID          string `json:"id"`
	SourceKind  string `json:"source_kind"`
	Title       string `json:"title"`
	Severity    string `json:"severity"`
	Score       int    `json:"score"`
	Status      string `json:"status"`
}

// ScanSummaryNotification carries scan-level summary for watch scan completion.
type ScanSummaryNotification struct {
	AssetCount    int `json:"asset_count"`
	PortCount     int `json:"port_count"`
	EndpointCount int `json:"endpoint_count"`
	ServiceCount  int `json:"service_count"`
	NewAssets     int `json:"new_assets"`
	GoneAssets    int `json:"gone_assets"`
}
