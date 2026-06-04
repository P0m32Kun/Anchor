package models

import (
	"time"
)

// BountyCandidate 表示一个赏金候选
type BountyCandidate struct {
	ID               string    `json:"id" db:"id"`
	ProjectID        string    `json:"project_id" db:"project_id"`
	ProgramID        *string   `json:"program_id,omitempty" db:"program_id"`
	FindingID        *string   `json:"finding_id,omitempty" db:"finding_id"`
	EndpointID       *string   `json:"endpoint_id,omitempty" db:"endpoint_id"`
	SourceKind       string    `json:"source_kind" db:"source_kind"`
	Title            string    `json:"title" db:"title"`
	VulnType         string    `json:"vuln_type" db:"vuln_type"`
	Severity         string    `json:"severity" db:"severity"`
	Confidence       string    `json:"confidence" db:"confidence"`
	ValueScore       int       `json:"value_score" db:"value_score"`
	ImpactScore      int       `json:"impact_score" db:"impact_score"`
	NoveltyScore     int       `json:"novelty_score" db:"novelty_score"`
	ReproScore       int       `json:"repro_score" db:"repro_score"`
	ScopeScore       int       `json:"scope_score" db:"scope_score"`
	SafetyScore      int       `json:"safety_score" db:"safety_score"`
	DuplicateRisk    string    `json:"duplicate_risk" db:"duplicate_risk"`
	RankingReason    string    `json:"ranking_reason" db:"ranking_reason"`
	VerifyStatus     string    `json:"verify_status" db:"verify_status"`
	SubmissionStatus string    `json:"submission_status" db:"submission_status"`
	SubmissionURL    string    `json:"submission_url" db:"submission_url"`
	SubmissionID     string    `json:"submission_id" db:"submission_id"`
	PaidAmount       int       `json:"paid_amount" db:"paid_amount"`
	Notes            string    `json:"notes" db:"notes"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

// SourceKind 来源类型
const (
	SourceKindFinding   = "finding"
	SourceKindEndpoint  = "endpoint"
	SourceKindAsset     = "asset"
	SourceKindManual    = "manual"
)

// VerifyStatus 验证状态
const (
	VerifyStatusPending         = "pending"
	VerifyStatusVerifying       = "verifying"
	VerifyStatusConfirmed       = "confirmed"
	VerifyStatusFalsePositive   = "false_positive"
	VerifyStatusNotApplicable   = "not_applicable"
)

// SubmissionStatus 提交状态
const (
	SubmissionStatusNotReady   = "not_ready"
	SubmissionStatusReady      = "ready"
	SubmissionStatusSubmitted  = "submitted"
	SubmissionStatusDuplicate  = "duplicate"
	SubmissionStatusAccepted   = "accepted"
	SubmissionStatusRejected   = "rejected"
	SubmissionStatusPaid       = "paid"
)

// DuplicateRisk 重复风险
const (
	DuplicateRiskLow     = "low"
	DuplicateRiskMedium  = "medium"
	DuplicateRiskHigh    = "high"
	DuplicateRiskUnknown = "unknown"
)

// UpdateBountyCandidateRequest 更新赏金候选请求
type UpdateBountyCandidateRequest struct {
	VerifyStatus     *string `json:"verify_status,omitempty"`
	SubmissionStatus *string `json:"submission_status,omitempty"`
	SubmissionURL    *string `json:"submission_url,omitempty"`
	SubmissionID     *string `json:"submission_id,omitempty"`
	PaidAmount       *int    `json:"paid_amount,omitempty"`
	Notes            *string `json:"notes,omitempty"`
}

// BountyCandidateStats 赏金候选统计
type BountyCandidateStats struct {
	Total           int `json:"total"`
	Pending         int `json:"pending"`
	Verified        int `json:"verified"`
	Submitted       int `json:"submitted"`
	Accepted        int `json:"accepted"`
	Paid            int `json:"paid"`
	TotalValue      int `json:"total_value"`
	AverageValue    int `json:"average_value"`
}

// IsValidVerifyStatus 检查是否是有效的验证状态
func IsValidVerifyStatus(status string) bool {
	switch status {
	case VerifyStatusPending, VerifyStatusVerifying, VerifyStatusConfirmed,
		VerifyStatusFalsePositive, VerifyStatusNotApplicable:
		return true
	}
	return false
}

// IsValidSubmissionStatus 检查是否是有效的提交状态
func IsValidSubmissionStatus(status string) bool {
	switch status {
	case SubmissionStatusNotReady, SubmissionStatusReady, SubmissionStatusSubmitted,
		SubmissionStatusDuplicate, SubmissionStatusAccepted, SubmissionStatusRejected,
		SubmissionStatusPaid:
		return true
	}
	return false
}

// IsValidDuplicateRisk 检查是否是有效的重复风险
func IsValidDuplicateRisk(risk string) bool {
	switch risk {
	case DuplicateRiskLow, DuplicateRiskMedium, DuplicateRiskHigh, DuplicateRiskUnknown:
		return true
	}
	return false
}

// IsValidSourceKind 检查是否是有效的来源类型
func IsValidSourceKind(kind string) bool {
	switch kind {
	case SourceKindFinding, SourceKindEndpoint, SourceKindAsset, SourceKindManual:
		return true
	}
	return false
}
