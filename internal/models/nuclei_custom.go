package models

import "time"

// NucleiCustomSourceType enumerates the supported source ingest mechanisms.
type NucleiCustomSourceType string

const (
	NucleiCustomSourceTypeGit    NucleiCustomSourceType = "git"
	NucleiCustomSourceTypeUpload NucleiCustomSourceType = "upload"
	NucleiCustomSourceTypeFile   NucleiCustomSourceType = "file"
)

// NucleiCustomSourceStatus enumerates the lifecycle states of a source.
type NucleiCustomSourceStatus string

const (
	NucleiCustomSourceStatusDraft NucleiCustomSourceStatus = "draft"
	NucleiCustomSourceStatusReady NucleiCustomSourceStatus = "ready"
	NucleiCustomSourceStatusError NucleiCustomSourceStatus = "error"
)

// NucleiCustomSource describes a user-managed Nuclei template source.
type NucleiCustomSource struct {
	ID              string                   `json:"id" db:"id"`
	Name            string                   `json:"name" db:"name"`
	InstallPath     string                   `json:"install_path" db:"install_path"`
	Type            NucleiCustomSourceType   `json:"type" db:"type"`
	URI             *string                  `json:"uri,omitempty" db:"uri"`
	Branch          *string                  `json:"branch,omitempty" db:"branch"`
	Enabled         bool                     `json:"enabled" db:"enabled"`
	RoutingPolicy   string                   `json:"routing_policy" db:"routing_policy"`
	Status          NucleiCustomSourceStatus `json:"status" db:"status"`
	LastSyncAt      *time.Time               `json:"last_sync_at,omitempty" db:"last_sync_at"`
	LastValidateAt  *time.Time               `json:"last_validate_at,omitempty" db:"last_validate_at"`
	LastError       *string                  `json:"last_error,omitempty" db:"last_error"`
	CreatedAt       time.Time                `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time                `json:"updated_at" db:"updated_at"`
}

// NucleiCustomBundle describes an immutable published bundle. Phase 1 ships
// the schema only; bundle creation and publishing are introduced in Phase 2.
type NucleiCustomBundle struct {
	Version      string     `json:"version" db:"version"`
	ManifestJSON string     `json:"manifest_json" db:"manifest_json"`
	ArchivePath  string     `json:"archive_path" db:"archive_path"`
	Status       string     `json:"status" db:"status"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	ActivatedAt  *time.Time `json:"activated_at,omitempty" db:"activated_at"`
}

// NucleiCustomFileEntry represents a node in the per-source file tree.
type NucleiCustomFileEntry struct {
	Path    string    `json:"path"`
	IsDir   bool      `json:"is_dir"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"mod_time"`
}

// NucleiCustomManifest is the manifest that workers will fetch in Phase 3 to
// learn the active bundle. Phase 1 keeps the type as a placeholder.
type NucleiCustomManifest struct {
	ActiveVersion string               `json:"active_version"`
	DownloadURL   string               `json:"download_url"`
	Sources       []NucleiCustomSource `json:"sources"`
	CreatedAt     time.Time            `json:"created_at"`
}

// NucleiCustomValidationResult is reserved for Phase 2 validation output.
type NucleiCustomValidationResult struct {
	SourceID string   `json:"source_id"`
	OK       bool     `json:"ok"`
	Errors   []string `json:"errors,omitempty"`
}
