package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// --- Project ---

type Project struct {
	ID             string    `json:"id" db:"id"`
	Name           string    `json:"name" db:"name"`
	Organization   string    `json:"organization" db:"organization"`
	Purpose        string    `json:"purpose" db:"purpose"`
	RateLimit      int       `json:"rate_limit" db:"rate_limit"`
	PortRange      *string   `json:"port_range,omitempty" db:"port_range"`
	DefaultProfile string    `json:"default_profile" db:"default_profile"`
	PipelineConfig *string   `json:"pipeline_config,omitempty" db:"pipeline_config"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

// --- Target ---

type IPDiscoveryResult struct {
	ID        string    `json:"id" db:"id"`
	ProjectID string    `json:"project_id" db:"project_id"`
	TargetID  string    `json:"target_id" db:"target_id"`
	IP        string    `json:"ip" db:"ip"`
	Hostname  *string   `json:"hostname,omitempty" db:"hostname"`
	Source    string    `json:"source" db:"source"` // naabu | nmap | manual
	Alive     bool      `json:"alive" db:"alive"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type TargetType string

const (
	TargetTypeDomain  TargetType = "domain"
	TargetTypeURL     TargetType = "url"
	TargetTypeIP      TargetType = "ip"
	TargetTypeCIDR    TargetType = "cidr"
	TargetTypeCompany TargetType = "company"
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

// --- Finding ---

type FindingSeverity string

const (
	SeverityInfo     FindingSeverity = "info"
	SeverityLow      FindingSeverity = "low"
	SeverityMedium   FindingSeverity = "medium"
	SeverityHigh     FindingSeverity = "high"
	SeverityCritical FindingSeverity = "critical"
)

type FindingStatus string

const (
	FindingNew           FindingStatus = "new"
	FindingPendingReview FindingStatus = "pending_review"
	FindingConfirmed     FindingStatus = "confirmed"
	FindingFalsePositive FindingStatus = "false_positive"
	FindingAcceptedRisk  FindingStatus = "accepted_risk"
	FindingIgnored       FindingStatus = "ignored"
	FindingReported      FindingStatus = "reported"
)

type Finding struct {
	ID              string          `json:"id" db:"id"`
	ProjectID       string          `json:"project_id" db:"project_id"`
	AssetID         *string         `json:"asset_id" db:"asset_id"`
	ServiceID       *string         `json:"service_id" db:"service_id"`
	WebEndpointID   *string         `json:"web_endpoint_id" db:"web_endpoint_id"`
	SourceTool      string          `json:"source_tool" db:"source_tool"`
	SourceRuleID    string          `json:"source_rule_id" db:"source_rule_id"`
	DedupKey        string          `json:"dedup_key" db:"dedup_key"`
	Title           string          `json:"title" db:"title"`
	Severity        FindingSeverity `json:"severity" db:"severity"`
	Confidence      int             `json:"confidence" db:"confidence"`
	Priority        int             `json:"priority" db:"priority"`
	Status          FindingStatus   `json:"status" db:"status"`
	Summary         string          `json:"summary" db:"summary"`
	Remediation     string          `json:"remediation" db:"remediation"`
	RawRequest      string          `json:"raw_request" db:"raw_request"`
	RawResponse     string          `json:"raw_response" db:"raw_response"`
	MatchedTemplate string          `json:"matched_template" db:"matched_template"`
	CreatedAt       time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at" db:"updated_at"`
}

// --- Evidence ---

type EvidenceType string

const (
	EvidenceRequest    EvidenceType = "request"
	EvidenceResponse   EvidenceType = "response"
	EvidenceScreenshot EvidenceType = "screenshot"
	EvidenceRawOutput  EvidenceType = "raw_output"
	EvidenceNote       EvidenceType = "note"
	EvidenceFile       EvidenceType = "file"
)

type Evidence struct {
	ID         string       `json:"id" db:"id"`
	FindingID  string       `json:"finding_id" db:"finding_id"`
	Type       EvidenceType `json:"type" db:"type"`
	ArtifactID *string      `json:"artifact_id" db:"artifact_id"`
	Excerpt    string       `json:"excerpt" db:"excerpt"`
	CreatedBy  string       `json:"created_by" db:"created_by"`
	CreatedAt  time.Time    `json:"created_at" db:"created_at"`
}

func (f FindingSeverity) Value() (driver.Value, error) { return string(f), nil }
func (f *FindingSeverity) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case string:
		*f = FindingSeverity(v)
		return nil
	case []byte:
		*f = FindingSeverity(v)
		return nil
	default:
		return fmt.Errorf("cannot scan %T into FindingSeverity", value)
	}
}

func (f FindingStatus) Value() (driver.Value, error) { return string(f), nil }
func (f *FindingStatus) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case string:
		*f = FindingStatus(v)
		return nil
	case []byte:
		*f = FindingStatus(v)
		return nil
	default:
		return fmt.Errorf("cannot scan %T into FindingStatus", value)
	}
}

func (e EvidenceType) Value() (driver.Value, error) { return string(e), nil }
func (e *EvidenceType) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case string:
		*e = EvidenceType(v)
		return nil
	case []byte:
		*e = EvidenceType(v)
		return nil
	default:
		return fmt.Errorf("cannot scan %T into EvidenceType", value)
	}
}

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

func ToJSON(v interface{}) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

// --- ToolTemplate ---

type ToolTemplate struct {
	ID                         string    `json:"id" db:"id"`
	Name                       string    `json:"name" db:"name"`
	Description                string    `json:"description" db:"description"`
	ProfileType                string    `json:"profile_type" db:"profile_type"`
	ToolsJSON                  string    `json:"tools_json" db:"tools_json"`
	DefaultMaxConcurrency      int       `json:"default_max_concurrency" db:"default_max_concurrency"`
	ScreenshotEnabled          bool      `json:"screenshot_enabled" db:"screenshot_enabled"`
	DirectoryBruteforceEnabled bool      `json:"directory_bruteforce_enabled" db:"directory_bruteforce_enabled"`
	NucleiSeverityFilter       string    `json:"nuclei_severity_filter" db:"nuclei_severity_filter"`
	CreatedAt                  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt                  time.Time `json:"updated_at" db:"updated_at"`
}

type TemplateTool struct {
	Tool    string `json:"tool"`
	Enabled bool   `json:"enabled"`
	Rate    int    `json:"rate"`
}

func (t *ToolTemplate) Tools() ([]TemplateTool, error) {
	var tools []TemplateTool
	if err := json.Unmarshal([]byte(t.ToolsJSON), &tools); err != nil {
		return nil, err
	}
	return tools, nil
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

// --- RetestRun ---

type RetestResult string

const (
	RetestStillPresent RetestResult = "still_present"
	RetestFixed        RetestResult = "fixed"
	RetestInconclusive RetestResult = "inconclusive"
	RetestFailedToTest RetestResult = "failed_to_test"
)

type RetestRun struct {
	ID         string       `json:"id" db:"id"`
	FindingID  string       `json:"finding_id" db:"finding_id"`
	TaskID     string       `json:"task_id" db:"task_id"`
	Result     RetestResult `json:"result" db:"result"`
	EvidenceID *string      `json:"evidence_id" db:"evidence_id"`
	CreatedAt  time.Time    `json:"created_at" db:"created_at"`
}

// --- Engine Credential ---

type EngineCredential struct {
	ID        string    `json:"id" db:"id"`
	Engine    string    `json:"engine" db:"engine"`
	APIKey    string    `json:"api_key" db:"api_key"`
	Email     *string   `json:"email,omitempty" db:"email"`
	Extra     *string   `json:"extra,omitempty" db:"extra"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// --- Pipeline Config ---

type PipelineConfig struct {
	EnableFOFA          bool   `json:"enable_fofa"`
	FofaResultLimit     int    `json:"fofa_result_limit"`
	FofaConcurrency     int    `json:"fofa_concurrency"`
	EnableSubfinder     bool   `json:"enable_subfinder"`
	SubfinderRateLimit  int    `json:"subfinder_rate_limit"`
	SubfinderThreads    int    `json:"subfinder_threads"`
	SubfinderTimeout    int    `json:"subfinder_timeout"`
	EnableDNSx          bool   `json:"enable_dnsx"`
	DNSxRateLimit       int    `json:"dnsx_rate_limit"`
	DNSxThreads         int    `json:"dnsx_threads"`
	DNSxTimeout         int    `json:"dnsx_timeout"`
	EnableCDNFilter     bool   `json:"enable_cdn_filter"`
	PortRange           string `json:"port_range"`
	NaabuRate           int    `json:"naabu_rate"`
	NaabuThreads        int    `json:"naabu_threads"`
	NaabuTimeout        int    `json:"naabu_timeout"`
	EnableNerva         bool   `json:"enable_nerva"`
	NervaRateLimit      int    `json:"nerva_rate_limit"`
	NervaWorkers        int    `json:"nerva_workers"`
	NervaTimeout        int    `json:"nerva_timeout"`
	EnableHttpx         bool   `json:"enable_httpx"`
	HttpxRateLimit      int    `json:"httpx_rate_limit"`
	HttpxThreads        int    `json:"httpx_threads"`
	EnableNuclei            bool   `json:"enable_nuclei"`
	NucleiRateLimit         int    `json:"nuclei_rate_limit"`          // -rl: requests per second
	NucleiRateLimitPerMinute int   `json:"nuclei_rate_limit_per_min"` // -rlm: requests per minute (for sensitive targets)
	NucleiConcurrency       int    `json:"nuclei_concurrency"`        // -c: parallel templates/hosts
	NucleiScanDepth         string `json:"nuclei_scan_depth"`         // "workflow" | "tags" | "both"
}

func DefaultPipelineConfig() PipelineConfig {
	return PipelineConfig{
		EnableFOFA:         true,
		FofaResultLimit:    500,
		FofaConcurrency:    5,
		EnableSubfinder:    true,
		SubfinderRateLimit: 50,
		SubfinderThreads:   10,
		SubfinderTimeout:   300,
		EnableDNSx:         true,
		DNSxRateLimit:      100,
		DNSxThreads:        50,
		DNSxTimeout:        5,
		EnableCDNFilter:    true,
		PortRange:          "top1000",
		NaabuRate:          1000,
		NaabuThreads:       100,
		NaabuTimeout:       600,
		EnableNerva:        true,
		NervaRateLimit:     100,
		NervaWorkers:       50,
		NervaTimeout:       10,
		EnableHttpx:        true,
		HttpxRateLimit:     150,
		HttpxThreads:       50,
		EnableNuclei:            true,
		NucleiRateLimit:         100,
		NucleiRateLimitPerMinute: 0, // disabled by default, set for sensitive targets
		NucleiConcurrency:       25,
		NucleiScanDepth:         "tags",
	}
}

// --- DNS ---

type DNSRecord struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	Domain    string    `json:"domain"`
	IPs       []string  `json:"ips"`
	CNAMEs    []string  `json:"cnames,omitempty"`
	TTL       uint32    `json:"ttl"`
	Resolver  string    `json:"resolver"`
	CreatedAt time.Time `json:"created_at"`
}

// --- CDN ---

type CDNResult struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	IP        string    `json:"ip"`
	IsCDN     bool      `json:"is_cdn"`
	Provider  string    `json:"provider,omitempty"`
	Type      string    `json:"type,omitempty"` // cdn | waf | cloud
	CreatedAt time.Time `json:"created_at"`
}

// --- Service Fingerprint ---

type ServiceFingerprint struct {
	ID        string                 `json:"id"`
	ProjectID string                 `json:"project_id"`
	IP        string                 `json:"ip"`
	Port      int                    `json:"port"`
	Protocol  string                 `json:"protocol"`
	IsWeb     bool                   `json:"is_web"`
	Service   string                 `json:"service"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Source    string                 `json:"source"`
	CreatedAt time.Time              `json:"created_at"`
}

// --- Pipeline Run ---

type PipelineRun struct {
	ID          string     `json:"id" db:"id"`
	ProjectID   string     `json:"project_id" db:"project_id"`
	Mode        string     `json:"mode" db:"mode"` // quick | standard | deep | custom
	Status      string     `json:"status" db:"status"` // running | completed | failed | cancelled
	Stage       string     `json:"stage,omitempty" db:"stage"`
	Error       string     `json:"error,omitempty" db:"error"`
	StartedAt   time.Time  `json:"started_at" db:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
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

// --- Dashboard ---

type DashboardStats struct {
	TotalProjects   int                    `json:"total_projects"`
	ActiveRuns      int                    `json:"active_runs"`
	PendingFindings int                    `json:"pending_findings"`
	OnlineWorkers   int                    `json:"online_workers"`
	RecentRuns      []*DashboardRunItem     `json:"recent_runs"`
	RecentFindings  []*DashboardFindingItem `json:"recent_findings"`
}

type DashboardRunItem struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	ProjectName string    `json:"project_name"`
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type DashboardFindingItem struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	ProjectName string    `json:"project_name"`
	Title       string    `json:"title"`
	Severity    string    `json:"severity"`
	CreatedAt   time.Time `json:"created_at"`
}

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
