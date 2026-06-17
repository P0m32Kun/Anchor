package models

import "time"

// Signal represents an alert signal in the BW2 Signal Inbox.
type Signal struct {
	ID          string    `json:"id" db:"id"`
	ProjectID   string    `json:"project_id" db:"project_id"`
	SourceKind  string    `json:"source_kind" db:"source_kind"`
	SourceID    string    `json:"source_id" db:"source_id"`
	Title       string    `json:"title" db:"title"`
	Severity    string    `json:"severity" db:"severity"`
	Score       int       `json:"score" db:"score"`
	ScopeStatus string    `json:"scope_status" db:"scope_status"`
	Status      string    `json:"status" db:"status"`
	Metadata    string    `json:"metadata" db:"metadata"`
	FirstSeen   time.Time `json:"first_seen" db:"first_seen"`
	LastSeen    time.Time `json:"last_seen" db:"last_seen"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

const (
	SignalSourceKindFinding     = "finding"
	SignalSourceKindAssetNew    = "new_asset"
	SignalSourceKindAssetGone   = "disappeared_asset"
	SignalSourceKindAssetChange = "asset_change"
	SignalSourceKindEndpoint    = "new_endpoint"
	SignalSourceKindPort        = "new_port"
	SignalSourceKindService     = "new_service"

	SignalStatusNew        = "new"
	SignalStatusAcknowledged = "acknowledged"
	SignalStatusResolved   = "resolved"

	SignalSeverityInfo     = "info"
	SignalSeverityLow      = "low"
	SignalSeverityMedium   = "medium"
	SignalSeverityHigh     = "high"
	SignalSeverityCritical = "critical"
)
