package models

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// --- ScanPlan ---

type ScanProfile string

const (
	ProfileLight    ScanProfile = "light"
	ProfileStandard ScanProfile = "standard"
	ProfileDeep     ScanProfile = "deep"
)

type ScanPlan struct {
	ID           string      `json:"id" db:"id"`
	ProjectID    string      `json:"project_id" db:"project_id"`
	WorkflowType string      `json:"workflow_type" db:"workflow_type"`
	Profile      ScanProfile `json:"profile" db:"profile"`
	Status       string      `json:"status" db:"status"`
	CreatedBy    string      `json:"created_by" db:"created_by"`
	CreatedAt    time.Time   `json:"created_at" db:"created_at"`
	ApprovedAt   *time.Time  `json:"approved_at" db:"approved_at"`
}

// --- ScanTask ---

type TaskStatus string

const (
	TaskCreated        TaskStatus = "created"
	TaskQueued         TaskStatus = "queued"
	TaskRunning        TaskStatus = "running"
	TaskCompleted      TaskStatus = "completed"
	TaskPartialSuccess TaskStatus = "partial_success"
	TaskFailed         TaskStatus = "failed"
	TaskCancelled      TaskStatus = "cancelled"
	TaskScopeDenied    TaskStatus = "scope_denied"
)

type ScanTask struct {
	ID                        string     `json:"id" db:"id"`
	ProjectID                 string     `json:"project_id" db:"project_id"`
	PlanID                    string     `json:"plan_id" db:"plan_id"`
	RunID                     *string    `json:"run_id" db:"run_id"`
	DependsOnTaskID           *string    `json:"depends_on_task_id" db:"depends_on_task_id"`
	TargetID                  *string    `json:"target_id" db:"target_id"`
	Tool                      string     `json:"tool" db:"tool"`
	CommandTemplate           string     `json:"command_template" db:"command_template"`
	ArgumentsRedacted         string     `json:"arguments_redacted" db:"arguments_redacted"`
	Status                    TaskStatus `json:"status" db:"status"`
	StartedAt                 *time.Time `json:"started_at" db:"started_at"`
	FinishedAt                *time.Time `json:"finished_at" db:"finished_at"`
	ExitCode                  *int       `json:"exit_code" db:"exit_code"`
	ErrorMessage              string     `json:"error_message,omitempty" db:"error_message"`
	WorkerID                  *string    `json:"worker_id" db:"worker_id"`
	NucleiCustomBundleVersion *string    `json:"nuclei_custom_bundle_version,omitempty" db:"nuclei_custom_bundle_version"`
	CreatedAt                 time.Time  `json:"created_at" db:"created_at"`
}

// --- ToolInvocation ---

type ToolInvocation struct {
	ID              string     `json:"id" db:"id"`
	ProjectID       string     `json:"project_id" db:"project_id"`
	TaskID          string     `json:"task_id" db:"task_id"`
	Tool            string     `json:"tool" db:"tool"`
	BinaryPath      string     `json:"binary_path" db:"binary_path"`
	Version         string     `json:"version" db:"version"`
	CommandRedacted string     `json:"command_redacted" db:"command_redacted"`
	Workdir         string     `json:"workdir" db:"workdir"`
	StartedAt       time.Time  `json:"started_at" db:"started_at"`
	FinishedAt      *time.Time `json:"finished_at" db:"finished_at"`
	ExitCode        *int       `json:"exit_code" db:"exit_code"`
}

// --- RawArtifact ---

type ArtifactType string

const (
	ArtifactStdout     ArtifactType = "stdout"
	ArtifactStderr     ArtifactType = "stderr"
	ArtifactJSONL      ArtifactType = "jsonl"
	ArtifactScreenshot ArtifactType = "screenshot"
	ArtifactRequest    ArtifactType = "request"
	ArtifactResponse   ArtifactType = "response"
	ArtifactFile       ArtifactType = "file"
)

type RawArtifact struct {
	ID              string       `json:"id" db:"id"`
	ProjectID       string       `json:"project_id" db:"project_id"`
	TaskID          *string      `json:"task_id" db:"task_id"`
	Type            ArtifactType `json:"type" db:"type"`
	Path            string       `json:"path" db:"path"`
	SHA256          string       `json:"sha256" db:"sha256"`
	Size            int64        `json:"size" db:"size"`
	RedactionStatus string       `json:"redaction_status" db:"redaction_status"`
	CreatedAt       time.Time    `json:"created_at" db:"created_at"`
}

// --- ScanStep ---

type StepName string

const (
	StepScopeCheck       StepName = "scope_check"
	StepPrepareInput     StepName = "prepare_input"
	StepRunTool          StepName = "run_tool"
	StepCollectArtifacts StepName = "collect_artifacts"
	StepParseOutput      StepName = "parse_output"
	StepNormalizeResult  StepName = "normalize_result"
	StepScoreResult      StepName = "score_result"
	StepCleanup          StepName = "cleanup"
)

type StepStatus string

const (
	StepPending   StepStatus = "pending"
	StepRunning   StepStatus = "running"
	StepCompleted StepStatus = "completed"
	StepFailed    StepStatus = "failed"
	StepSkipped   StepStatus = "skipped"
)

type ScanStep struct {
	ID           string     `json:"id" db:"id"`
	TaskID       string     `json:"task_id" db:"task_id"`
	Name         StepName   `json:"name" db:"name"`
	Status       StepStatus `json:"status" db:"status"`
	StartedAt    *time.Time `json:"started_at" db:"started_at"`
	FinishedAt   *time.Time `json:"finished_at" db:"finished_at"`
	ErrorCode    string     `json:"error_code" db:"error_code"`
	ErrorSummary string     `json:"error_summary" db:"error_summary"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
}

// --- Run ---

type RunStatus string

const (
	RunPending   RunStatus = "pending"
	RunRunning   RunStatus = "running"
	RunCompleted RunStatus = "completed"
	RunFailed    RunStatus = "failed"
	RunCancelled RunStatus = "cancelled"
)

type Run struct {
	ID             string     `json:"id" db:"id"`
	ProjectID      string     `json:"project_id" db:"project_id"`
	ToolTemplateID *string    `json:"tool_template_id" db:"tool_template_id"`
	Name           string     `json:"name" db:"name"`
	Status         RunStatus  `json:"status" db:"status"`
	StartedAt      *time.Time `json:"started_at" db:"started_at"`
	FinishedAt     *time.Time `json:"finished_at" db:"finished_at"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
}

// --- Pipeline Run ---

type PipelineRun struct {
	ID          string    `json:"id" db:"id"`
	ProjectID   string    `json:"project_id" db:"project_id"`
	Mode        string    `json:"mode" db:"mode"`     // quick | standard | deep | custom
	Status      string    `json:"status" db:"status"` // running | completed | failed | cancelled
	Stage       string    `json:"stage,omitempty" db:"stage"`
	Error       string    `json:"error,omitempty" db:"error"`
	StartedAt   time.Time `json:"started_at" db:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// --- Pipeline Run Stage ---

type PipelineRunStageStatus string

const (
	StageStatusPending   PipelineRunStageStatus = "pending"
	StageStatusRunning   PipelineRunStageStatus = "running"
	StageStatusCompleted PipelineRunStageStatus = "completed"
	StageStatusFailed    PipelineRunStageStatus = "failed"
	StageStatusSkipped   PipelineRunStageStatus = "skipped"
)

type PipelineRunStage struct {
	ID          string                 `json:"id" db:"id"`
	RunID       string                 `json:"run_id" db:"run_id"`
	Stage       string                 `json:"stage" db:"stage"`
	Status      PipelineRunStageStatus `json:"status" db:"status"`
	Error       string                 `json:"error,omitempty" db:"error"`
	StartedAt   *time.Time             `json:"started_at,omitempty" db:"started_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty" db:"completed_at"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
}

// --- Screenshot ---

type Screenshot struct {
	ID            string    `json:"id" db:"id"`
	ProjectID     string    `json:"project_id" db:"project_id"`
	AssetID       *string   `json:"asset_id" db:"asset_id"`
	TaskID        *string   `json:"task_id" db:"task_id"`
	URL           string    `json:"url" db:"url"`
	OriginalPath  string    `json:"original_path" db:"original_path"`
	ThumbnailPath string    `json:"thumbnail_path" db:"thumbnail_path"`
	Width         int       `json:"width" db:"width"`
	Height        int       `json:"height" db:"height"`
	TakenAt       time.Time `json:"taken_at" db:"taken_at"`
}

// --- AuditLog ---

type AuditLog struct {
	ID           string    `json:"id" db:"id"`
	ProjectID    string    `json:"project_id" db:"project_id"`
	Actor        string    `json:"actor" db:"actor"`
	Action       string    `json:"action" db:"action"`
	ResourceType string    `json:"resource_type" db:"resource_type"`
	ResourceID   string    `json:"resource_id" db:"resource_id"`
	Summary      string    `json:"summary" db:"summary"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// --- JSON helpers for TaskStatus ---

func (s TaskStatus) Value() (driver.Value, error) { return string(s), nil }
func (s *TaskStatus) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case string:
		*s = TaskStatus(v)
		return nil
	case []byte:
		*s = TaskStatus(v)
		return nil
	default:
		return fmt.Errorf("cannot scan %T into TaskStatus", value)
	}
}

// --- JSON helpers for ArtifactType ---

func (a ArtifactType) Value() (driver.Value, error) { return string(a), nil }
func (a *ArtifactType) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case string:
		*a = ArtifactType(v)
		return nil
	case []byte:
		*a = ArtifactType(v)
		return nil
	default:
		return fmt.Errorf("cannot scan %T into ArtifactType", value)
	}
}

// --- JSON helpers for StepStatus ---

func (s StepStatus) Value() (driver.Value, error) { return string(s), nil }
func (s *StepStatus) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case string:
		*s = StepStatus(v)
		return nil
	case []byte:
		*s = StepStatus(v)
		return nil
	default:
		return fmt.Errorf("cannot scan %T into StepStatus", value)
	}
}

// --- JSON helpers for StepName ---

func (n StepName) Value() (driver.Value, error) { return string(n), nil }
func (n *StepName) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case string:
		*n = StepName(v)
		return nil
	case []byte:
		*n = StepName(v)
		return nil
	default:
		return fmt.Errorf("cannot scan %T into StepName", value)
	}
}

// --- JSON helpers for RunStatus ---

func (r RunStatus) Value() (driver.Value, error) { return string(r), nil }
func (r *RunStatus) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case string:
		*r = RunStatus(v)
		return nil
	case []byte:
		*r = RunStatus(v)
		return nil
	default:
		return fmt.Errorf("cannot scan %T into RunStatus", value)
	}
}

// --- JSON helpers for PipelineRunStageStatus ---

func (s PipelineRunStageStatus) Value() (driver.Value, error) { return string(s), nil }
func (s *PipelineRunStageStatus) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case string:
		*s = PipelineRunStageStatus(v)
		return nil
	case []byte:
		*s = PipelineRunStageStatus(v)
		return nil
	default:
		return fmt.Errorf("cannot scan %T into PipelineRunStageStatus", value)
	}
}
