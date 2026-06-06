package models

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// ToolCallStatus represents the lifecycle state of a tool call log entry.
type ToolCallStatus string

const (
	ToolCallRunning   ToolCallStatus = "running"
	ToolCallCompleted ToolCallStatus = "completed"
	ToolCallFailed    ToolCallStatus = "failed"
)

// ToolCallLog records a single tool invocation for traceability.
// It captures the full lifecycle: when a tool was called, with what parameters,
// how long it took, and what the outcome was.
type ToolCallLog struct {
	ID            string         `json:"id" db:"id"`
	RunID         string         `json:"run_id" db:"run_id"`
	WorkItemID    *string        `json:"work_item_id,omitempty" db:"work_item_id"`
	TaskID        *string        `json:"task_id,omitempty" db:"task_id"`
	Tool          string         `json:"tool" db:"tool"`
	Action        string         `json:"action" db:"action"`
	AssetID       *string        `json:"asset_id,omitempty" db:"asset_id"`
	ParamsJSON    string         `json:"params_json" db:"params_json"`
	StartedAt     time.Time      `json:"started_at" db:"started_at"`
	FinishedAt    *time.Time     `json:"finished_at,omitempty" db:"finished_at"`
	DurationMs    *int64         `json:"duration_ms,omitempty" db:"duration_ms"`
	ExitCode      *int           `json:"exit_code,omitempty" db:"exit_code"`
	Status        ToolCallStatus `json:"status" db:"status"`
	OutputSummary *string        `json:"output_summary,omitempty" db:"output_summary"`
	ErrorMessage  *string        `json:"error_message,omitempty" db:"error_message"`
	CreatedAt     time.Time      `json:"created_at" db:"created_at"`
}

// ToolCallTrace represents the full trace chain from a Finding back to the
// pipeline run. Used for the GET /findings/{id}/trace endpoint.
type ToolCallTrace struct {
	Finding      *Finding       `json:"finding"`
	WorkItem     *ScanWorkItem  `json:"work_item,omitempty"`
	Task         *ScanTask      `json:"task,omitempty"`
	ToolCallLog  *ToolCallLog   `json:"tool_call_log,omitempty"`
	Run          *PipelineRun   `json:"run,omitempty"`
}

// --- JSON helpers for ToolCallStatus ---

func (s ToolCallStatus) Value() (driver.Value, error) { return string(s), nil }

func (s *ToolCallStatus) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case string:
		*s = ToolCallStatus(v)
		return nil
	case []byte:
		*s = ToolCallStatus(v)
		return nil
	default:
		return fmt.Errorf("cannot scan %T into ToolCallStatus", value)
	}
}
