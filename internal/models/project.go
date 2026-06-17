package models

import "time"

// --- Project ---
type Project struct {
	ID                string    `json:"id" db:"id"`
	Name              string    `json:"name" db:"name"`
	Organization      string    `json:"organization" db:"organization"`
	Purpose           string    `json:"purpose" db:"purpose"`
	RateLimit         int       `json:"rate_limit" db:"rate_limit"`
	PortRange         *string   `json:"port_range,omitempty" db:"port_range"`
	DefaultProfile    string    `json:"default_profile" db:"default_profile"`
	ScopeBoundaryMode string    `json:"scope_boundary_mode" db:"scope_boundary_mode"`
	PipelineConfig    *string   `json:"pipeline_config,omitempty" db:"pipeline_config"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
}
