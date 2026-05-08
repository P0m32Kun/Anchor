package models

import (
	"database/sql/driver"
	"fmt"
	"time"
)

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

// --- JSON helpers for FindingSeverity ---

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

// --- JSON helpers for FindingStatus ---

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

// --- JSON helpers for EvidenceType ---

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
