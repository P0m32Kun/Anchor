package models

import (
	"time"
)

// SubmissionPack 表示一个提交包
type SubmissionPack struct {
	ID              string    `json:"id" db:"id"`
	CandidateID     string    `json:"candidate_id" db:"candidate_id"`
	Format          string    `json:"format" db:"format"`
	Template        string    `json:"template" db:"template"`
	Content         string    `json:"content" db:"content"`
	ChecklistJSON   string    `json:"checklist_json" db:"checklist_json"`
	RedactionStatus string    `json:"redaction_status" db:"redaction_status"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

// SubmissionFormat 提交格式
const (
	SubmissionFormatMarkdown = "markdown"
)

// SubmissionTemplate 提交模板
const (
	SubmissionTemplateGeneric   = "generic"
	SubmissionTemplate360SRC    = "360src"
	SubmissionTemplateHackerOne = "hackerone"
	SubmissionTemplateBugcrowd  = "bugcrowd"
)

// RedactionStatus 脱敏状态
const (
	RedactionStatusRaw      = "raw"
	RedactionStatusReviewed  = "reviewed"
	RedactionStatusRedacted = "redacted"
)

// ChecklistItem 检查清单项
type ChecklistItem struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Checked bool   `json:"checked"`
	Note    string `json:"note,omitempty"`
}

// CreateSubmissionPackRequest 创建提交包请求
type CreateSubmissionPackRequest struct {
	Template string `json:"template"`
}

// UpdateSubmissionPackRequest 更新提交包请求
type UpdateSubmissionPackRequest struct {
	Content         *string `json:"content,omitempty"`
	ChecklistJSON   *string `json:"checklist_json,omitempty"`
	RedactionStatus *string `json:"redaction_status,omitempty"`
}

// DefaultChecklist 默认检查清单
func DefaultChecklist() []ChecklistItem {
	return []ChecklistItem{
		{ID: "reproduce", Label: "漏洞可复现", Checked: false},
		{ID: "impact", Label: "影响范围已确认", Checked: false},
		{ID: "evidence", Label: "证据完整（请求/响应）", Checked: false},
		{ID: "remediation", Label: "修复建议已提供", Checked: false},
		{ID: "scope", Label: "在授权范围内", Checked: false},
		{ID: "safe", Label: "测试方法安全", Checked: false},
		{ID: "dedup", Label: "已检查重复", Checked: false},
	}
}

// IsValidSubmissionTemplate 检查是否是有效的提交模板
func IsValidSubmissionTemplate(template string) bool {
	switch template {
	case SubmissionTemplateGeneric, SubmissionTemplate360SRC,
		SubmissionTemplateHackerOne, SubmissionTemplateBugcrowd:
		return true
	}
	return false
}

// IsValidRedactionStatus 检查是否是有效的脱敏状态
func IsValidRedactionStatus(status string) bool {
	switch status {
	case RedactionStatusRaw, RedactionStatusReviewed, RedactionStatusRedacted:
		return true
	}
	return false
}
