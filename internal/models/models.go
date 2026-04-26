package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// --- Project ---

type Project struct {
	ID             string     `json:"id" db:"id"`
	Name           string     `json:"name" db:"name"`
	Organization   string     `json:"organization" db:"organization"`
	Purpose        string     `json:"purpose" db:"purpose"`
	StartTime      *time.Time `json:"start_time" db:"start_time"`
	EndTime        *time.Time `json:"end_time" db:"end_time"`
	RateLimit      int        `json:"rate_limit" db:"rate_limit"`
	DefaultProfile string     `json:"default_profile" db:"default_profile"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
}

// --- Target ---

type TargetType string

const (
	TargetTypeDomain TargetType = "domain"
	TargetTypeURL    TargetType = "url"
	TargetTypeIP     TargetType = "ip"
	TargetTypeCIDR   TargetType = "cidr"
)

type Target struct {
	ID        string     `json:"id" db:"id"`
	ProjectID string     `json:"project_id" db:"project_id"`
	Type      TargetType `json:"type" db:"type"`
	Value     string     `json:"value" db:"value"`
	Source    string     `json:"source" db:"source"` // manual | import | tool
	Status    string     `json:"status" db:"status"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
}

// --- ScopeRule ---

type ScopeAction string

const (
	ScopeActionInclude ScopeAction = "include"
	ScopeActionExclude ScopeAction = "exclude"
)

type ScopeRule struct {
	ID        string      `json:"id" db:"id"`
	ProjectID string      `json:"project_id" db:"project_id"`
	Action    ScopeAction `json:"action" db:"action"`
	Type      TargetType  `json:"type" db:"type"`
	Value     string      `json:"value" db:"value"`
	Reason    string      `json:"reason" db:"reason"`
	CreatedAt time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt time.Time   `json:"updated_at" db:"updated_at"`
}

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
	ID                string     `json:"id" db:"id"`
	ProjectID         string     `json:"project_id" db:"project_id"`
	PlanID            string     `json:"plan_id" db:"plan_id"`
	DependsOnTaskID   *string    `json:"depends_on_task_id" db:"depends_on_task_id"`
	TargetID          *string    `json:"target_id" db:"target_id"`
	Tool              string     `json:"tool" db:"tool"`
	CommandTemplate   string     `json:"command_template" db:"command_template"`
	ArgumentsRedacted string     `json:"arguments_redacted" db:"arguments_redacted"`
	Status            TaskStatus `json:"status" db:"status"`
	StartedAt         *time.Time `json:"started_at" db:"started_at"`
	FinishedAt        *time.Time `json:"finished_at" db:"finished_at"`
	ExitCode          *int       `json:"exit_code" db:"exit_code"`
	WorkerID          *string    `json:"worker_id" db:"worker_id"`
	CreatedAt         time.Time  `json:"created_at" db:"created_at"`
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

// --- ScopeDecision ---

type ScopeDecisionResult string

const (
	ScopeAllow ScopeDecisionResult = "allow"
	ScopeDeny  ScopeDecisionResult = "deny"
)

type ScopeDecision struct {
	ID            string              `json:"id" db:"id"`
	ProjectID     string              `json:"project_id" db:"project_id"`
	TargetValue   string              `json:"target_value" db:"target_value"`
	TaskID        *string             `json:"task_id" db:"task_id"`
	Decision      ScopeDecisionResult `json:"decision" db:"decision"`
	MatchedRuleID *string             `json:"matched_rule_id" db:"matched_rule_id"`
	Reason        string              `json:"reason" db:"reason"`
	CreatedAt     time.Time           `json:"created_at" db:"created_at"`
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

// --- ToolHealth ---

type ToolHealth struct {
	ID               string    `json:"id" db:"id"`
	Tool             string    `json:"tool" db:"tool"`
	BinaryPath       string    `json:"binary_path" db:"binary_path"`
	Version          string    `json:"version" db:"version"`
	TemplatePath     *string   `json:"template_path" db:"template_path"`
	WorkdirWritable  bool      `json:"workdir_writable" db:"workdir_writable"`
	NetworkAvailable bool      `json:"network_available" db:"network_available"`
	DNSAvailable     bool      `json:"dns_available" db:"dns_available"`
	ProxyReachable   *bool     `json:"proxy_reachable" db:"proxy_reachable"`
	LastCheckAt      time.Time `json:"last_check_at" db:"last_check_at"`
}

// --- JSON helpers ---

func (t TargetType) Value() (driver.Value, error) { return string(t), nil }
func (t *TargetType) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case string:
		*t = TargetType(v)
		return nil
	case []byte:
		*t = TargetType(v)
		return nil
	default:
		return fmt.Errorf("cannot scan %T into TargetType", value)
	}
}

func (a ScopeAction) Value() (driver.Value, error) { return string(a), nil }
func (a *ScopeAction) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case string:
		*a = ScopeAction(v)
		return nil
	case []byte:
		*a = ScopeAction(v)
		return nil
	default:
		return fmt.Errorf("cannot scan %T into ScopeAction", value)
	}
}

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

func (d ScopeDecisionResult) Value() (driver.Value, error) { return string(d), nil }
func (d *ScopeDecisionResult) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case string:
		*d = ScopeDecisionResult(v)
		return nil
	case []byte:
		*d = ScopeDecisionResult(v)
		return nil
	default:
		return fmt.Errorf("cannot scan %T into ScopeDecisionResult", value)
	}
}

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

func ToJSON(v interface{}) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
