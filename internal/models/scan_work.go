package models

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// WorkStatus represents the lifecycle state of a ScanWorkItem.
type WorkStatus string

const (
	WorkStatusPending WorkStatus = "pending"
	WorkStatusRunning WorkStatus = "running"
	WorkStatusDone    WorkStatus = "done"
	WorkStatusSkipped WorkStatus = "skipped"
	WorkStatusFailed  WorkStatus = "failed"
)

// ScanWorkItem represents a single (asset × action) unit of work within a
// pipeline run. It is the scheduling truth for the asset-driven scan engine.
type ScanWorkItem struct {
	ID          string     `json:"id" db:"id"`
	RunID       string     `json:"run_id" db:"run_id"`
	ProjectID   string     `json:"project_id" db:"project_id"`
	AssetID     string     `json:"asset_id" db:"asset_id"`
	Action      string     `json:"action" db:"action"`
	TaskID      *string    `json:"task_id,omitempty" db:"task_id"`
	Status      WorkStatus `json:"status" db:"status"`
	SkipReason  string     `json:"skip_reason,omitempty" db:"skip_reason"`
	Stage       string     `json:"stage,omitempty" db:"stage"`
	Error       string     `json:"error,omitempty" db:"error"`
	StartedAt   *time.Time `json:"started_at,omitempty" db:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
}

// --- JSON helpers for WorkStatus ---

func (s WorkStatus) Value() (driver.Value, error) { return string(s), nil }

func (s *WorkStatus) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case string:
		*s = WorkStatus(v)
		return nil
	case []byte:
		*s = WorkStatus(v)
		return nil
	default:
		return fmt.Errorf("cannot scan %T into WorkStatus", value)
	}
}

// ScanRunMetrics holds aggregated metrics for a single pipeline run,
// exposed via the GET /runs/{runId}/metrics endpoint.
type ScanRunMetrics struct {
	EngineState      string         `json:"engine_state"`
	AssetsDiscovered int            `json:"assets_discovered"`
	WorksPending     int            `json:"works_pending"`
	WorksDone        int            `json:"works_done"`
	WorksSkipped     int            `json:"works_skipped"`
	WorksRunning     int            `json:"works_running"`
	WorksFailed      int            `json:"works_failed"`
	QueueDepth       QueueDepthInfo `json:"queue_depth"`
	LastNewAssetAt   *time.Time     `json:"last_new_asset_at,omitempty"`
}

// QueueDepthInfo holds the number of pending work items per priority tier.
type QueueDepthInfo struct {
	High   int `json:"high"`
	Medium int `json:"medium"`
	Low    int `json:"low"`
}
