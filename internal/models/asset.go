package models

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// --- Asset ---

type AssetType string

const (
	AssetTypeDomain AssetType = "domain"
	AssetTypeIP     AssetType = "ip"
	AssetTypeURL    AssetType = "url"
)

type Asset struct {
	ID              string            `json:"id" db:"id"`
	ProjectID       string            `json:"project_id" db:"project_id"`
	Type            AssetType         `json:"type" db:"type"`
	Value           string            `json:"value" db:"value"`
	NormalizedValue string            `json:"normalized_value" db:"normalized_value"`
	SourceTools     []string          `json:"source_tools" db:"source_tools"`
	FirstSeen       time.Time         `json:"first_seen" db:"first_seen"`
	LastSeen        time.Time         `json:"last_seen" db:"last_seen"`
	Tags            map[string]string `json:"tags" db:"tags"`
}

// --- Port ---

type Port struct {
	ID         string    `json:"id" db:"id"`
	AssetID    string    `json:"asset_id" db:"asset_id"`
	Port       int       `json:"port" db:"port"`
	Protocol   string    `json:"protocol" db:"protocol"`
	State      string    `json:"state" db:"state"`
	SourceTool string    `json:"source_tool" db:"source_tool"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// --- Service ---

type Service struct {
	ID         string    `json:"id" db:"id"`
	AssetID    string    `json:"asset_id" db:"asset_id"`
	PortID     *string   `json:"port_id" db:"port_id"`
	Name       string    `json:"name" db:"name"`
	Product    string    `json:"product" db:"product"`
	Version    string    `json:"version" db:"version"`
	Banner     string    `json:"banner" db:"banner"`
	Confidence int       `json:"confidence" db:"confidence"`
	SourceTool string    `json:"source_tool" db:"source_tool"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// --- WebEndpoint ---

type WebEndpoint struct {
	ID                   string    `json:"id" db:"id"`
	ProjectID            string    `json:"project_id" db:"project_id"`
	AssetID              string    `json:"asset_id" db:"asset_id"`
	URL                  string    `json:"url" db:"url"`
	Scheme               string    `json:"scheme" db:"scheme"`
	Host                 string    `json:"host" db:"host"`
	Port                 *int      `json:"port" db:"port"`
	Path                 string    `json:"path" db:"path"`
	StatusCode           *int      `json:"status_code" db:"status_code"`
	Title                string    `json:"title" db:"title"`
	Technologies         []string  `json:"technologies" db:"technologies"`
	ScreenshotArtifactID *string   `json:"screenshot_artifact_id" db:"screenshot_artifact_id"`
	SourceTool           string    `json:"source_tool" db:"source_tool"`
	CreatedAt            time.Time `json:"created_at" db:"created_at"`
}

// --- ServicePort (aggregated view) ---

type ServicePort struct {
	ID           string    `json:"id"`
	ProjectID    string    `json:"project_id"`
	AssetID      string    `json:"asset_id"`
	IP           string    `json:"ip"`
	Port         int       `json:"port"`
	Protocol     string    `json:"protocol"`
	State        string    `json:"state"`
	ServiceName  string    `json:"service_name"`
	Title        string    `json:"title"`
	Technologies []string  `json:"technologies"`
	URL          string    `json:"url"`
	SourceTools  []string  `json:"source_tools"`
	IsWeb        bool      `json:"is_web"`
	CreatedAt    time.Time `json:"created_at"`
}

// --- JSON helpers for AssetType ---

func (a AssetType) Value() (driver.Value, error) { return string(a), nil }
func (a *AssetType) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case string:
		*a = AssetType(v)
		return nil
	case []byte:
		*a = AssetType(v)
		return nil
	default:
		return fmt.Errorf("cannot scan %T into AssetType", value)
	}
}
