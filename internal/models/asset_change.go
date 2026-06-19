package models

import "time"

// AssetChange records a granular change to an asset detected between scan runs.
type AssetChange struct {
	ID            string    `json:"id" db:"id"`
	ProjectID     string    `json:"project_id" db:"project_id"`
	RunID         string    `json:"run_id" db:"run_id"`
	AssetID       string    `json:"asset_id" db:"asset_id"`
	AssetValue    string    `json:"asset_value" db:"asset_value"`
	AssetType     string    `json:"asset_type" db:"asset_type"`
	ChangeType    string    `json:"change_type" db:"change_type"`
	ChangeSummary string    `json:"change_summary" db:"change_summary"`
	DetailJSON    string    `json:"detail_json" db:"detail_json"`
	Severity      string    `json:"severity" db:"severity"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

const (
	ChangeTypePortNew         = "port_new"
	ChangeTypePortGone        = "port_gone"
	ChangeTypeServiceNew      = "service_new"
	ChangeTypeServiceGone     = "service_gone"
	ChangeTypeServiceVersion  = "service_version_change"
	ChangeTypeEndpointNew     = "endpoint_new"
	ChangeTypeEndpointGone    = "endpoint_gone"
	ChangeTypeEndpointChanged = "endpoint_changed"
	ChangeTypeCertExpiring    = "cert_expiring"
	ChangeTypeCertChanged     = "cert_changed"
	ChangeTypeAssetNew        = "asset_new"
	ChangeTypeAssetGone       = "asset_gone"
)

// AlertWebhook configures a webhook endpoint for project alerts.
type AlertWebhook struct {
	ID               string    `json:"id" db:"id"`
	ProjectID        string    `json:"project_id" db:"project_id"`
	Enabled          bool      `json:"enabled" db:"enabled"`
	URL              string    `json:"url" db:"url"`
	Secret           string    `json:"secret,omitempty" db:"secret"`
	MinSeverity      string    `json:"min_severity" db:"min_severity"`
	OnNewAsset       bool      `json:"on_new_asset" db:"on_new_asset"`
	OnAssetGone      bool      `json:"on_asset_gone" db:"on_asset_gone"`
	OnPortChange     bool      `json:"on_port_change" db:"on_port_change"`
	OnServiceChange  bool      `json:"on_service_change" db:"on_service_change"`
	OnCertExpiry     bool      `json:"on_cert_expiry" db:"on_cert_expiry"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}
