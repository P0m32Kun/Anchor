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
