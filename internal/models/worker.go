package models

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// --- WorkerNode ---

type WorkerMode string

const (
	WorkerModeRemote WorkerMode = "remote"
)

type WorkerStatus string

const (
	WorkerStatusOnline  WorkerStatus = "online"
	WorkerStatusOffline WorkerStatus = "offline"
	WorkerStatusBusy    WorkerStatus = "busy"
	WorkerStatusError   WorkerStatus = "error"
)

type WorkerNode struct {
	ID               string       `json:"id" db:"id"`
	Name             string       `json:"name" db:"name"`
	Endpoint         string       `json:"endpoint" db:"endpoint"`
	Mode             WorkerMode   `json:"mode" db:"mode"`
	Status           WorkerStatus `json:"status" db:"status"`
	TrustLevel       string       `json:"trust_level" db:"trust_level"`
	NetworkProfile   string       `json:"network_profile" db:"network_profile"`
	Capabilities     string       `json:"capabilities" db:"capabilities"`
	ToolVersions     string       `json:"tool_versions" db:"tool_versions"`
	TemplateVersions string       `json:"template_versions" db:"template_versions"`
	MaxConcurrency   int          `json:"max_concurrency" db:"max_concurrency"`
	LastSeen         *time.Time   `json:"last_seen" db:"last_seen"`
	// System resource metrics (updated via heartbeat)
	CPUPercent       *float64     `json:"cpu_percent" db:"cpu_percent"`
	MemPercent       *float64     `json:"mem_percent" db:"mem_percent"`
	DiskPercent      *float64     `json:"disk_percent" db:"disk_percent"`
	MetricsUpdatedAt *time.Time   `json:"metrics_updated_at" db:"metrics_updated_at"`
	CreatedAt        time.Time    `json:"created_at" db:"created_at"`
	RevokedAt        *time.Time   `json:"revoked_at" db:"revoked_at"`
}

// --- WorkerHealthCheck ---

type HealthCheckStatus string

const (
	HealthCheckReady           HealthCheckStatus = "ready"
	HealthCheckMissing         HealthCheckStatus = "missing"
	HealthCheckVersionMismatch HealthCheckStatus = "version_mismatch"
	HealthCheckConfigError     HealthCheckStatus = "config_error"
	HealthCheckPermissionError HealthCheckStatus = "permission_error"
)

type WorkerHealthCheck struct {
	ID        string            `json:"id" db:"id"`
	WorkerID  string            `json:"worker_id" db:"worker_id"`
	Tool      string            `json:"tool" db:"tool"`
	Status    HealthCheckStatus `json:"status" db:"status"`
	Version   string            `json:"version" db:"version"`
	Details   string            `json:"details" db:"details"`
	CheckedAt time.Time         `json:"checked_at" db:"checked_at"`
}

// --- JSON helpers for WorkerMode ---

func (w WorkerMode) Value() (driver.Value, error) { return string(w), nil }
func (w *WorkerMode) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case string:
		*w = WorkerMode(v)
		return nil
	case []byte:
		*w = WorkerMode(v)
		return nil
	default:
		return fmt.Errorf("cannot scan %T into WorkerMode", value)
	}
}

// --- JSON helpers for WorkerStatus ---

func (w WorkerStatus) Value() (driver.Value, error) { return string(w), nil }
func (w *WorkerStatus) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case string:
		*w = WorkerStatus(v)
		return nil
	case []byte:
		*w = WorkerStatus(v)
		return nil
	default:
		return fmt.Errorf("cannot scan %T into WorkerStatus", value)
	}
}

// --- JSON helpers for HealthCheckStatus ---

func (h HealthCheckStatus) Value() (driver.Value, error) { return string(h), nil }
func (h *HealthCheckStatus) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case string:
		*h = HealthCheckStatus(v)
		return nil
	case []byte:
		*h = HealthCheckStatus(v)
		return nil
	default:
		return fmt.Errorf("cannot scan %T into HealthCheckStatus", value)
	}
}
