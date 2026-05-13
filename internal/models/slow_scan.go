package models

import "time"

type SlowScanTool string

const (
	SlowScanToolFfuf      SlowScanTool = "ffuf"
	SlowScanToolURLFinder SlowScanTool = "urlfinder"
)

type SlowScanStatus string

const (
	SlowScanPending   SlowScanStatus = "pending"
	SlowScanRunning   SlowScanStatus = "running"
	SlowScanCompleted SlowScanStatus = "completed"
	SlowScanFailed    SlowScanStatus = "failed"
	SlowScanCancelled SlowScanStatus = "cancelled"
)

type SlowScanTask struct {
	ID           string         `json:"id" db:"id"`
	ProjectID    string         `json:"project_id" db:"project_id"`
	TargetID     *string        `json:"target_id,omitempty" db:"target_id"`
	RunID        *string        `json:"run_id,omitempty" db:"run_id"`
	Tool         SlowScanTool   `json:"tool" db:"tool"`
	Status       SlowScanStatus `json:"status" db:"status"`
	ConfigJSON   string         `json:"config_json" db:"config_json"`
	RateLimit    int            `json:"rate_limit" db:"rate_limit"`
	Timeout      int            `json:"timeout" db:"timeout"`
	ErrorMessage string         `json:"error_message,omitempty" db:"error_message"`
	StartedAt    *time.Time     `json:"started_at,omitempty" db:"started_at"`
	FinishedAt   *time.Time     `json:"finished_at,omitempty" db:"finished_at"`
	CreatedAt    time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at" db:"updated_at"`
}
