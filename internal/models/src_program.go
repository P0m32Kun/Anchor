package models

import (
	"encoding/json"
	"time"
)

// SRCProgram 表示一个 SRC 平台程序配置
type SRCProgram struct {
	ID                     string    `json:"id" db:"id"`
	ProjectID              string    `json:"project_id" db:"project_id"`
	Name                   string    `json:"name" db:"name"`
	Platform               string    `json:"platform" db:"platform"`
	ProgramURL             string    `json:"program_url" db:"program_url"`
	RulesURL               string    `json:"rules_url" db:"rules_url"`
	AllowAutomation        bool      `json:"allow_automation" db:"allow_automation"`
	AllowDirBrute          bool      `json:"allow_dir_brute" db:"allow_dir_brute"`
	AllowWeakPassword      bool      `json:"allow_weak_password" db:"allow_weak_password"`
	AllowAuthenticatedTest bool      `json:"allow_authenticated_test" db:"allow_authenticated_test"`
	MaxRPS                 int       `json:"max_rps" db:"max_rps"`
	MaxConcurrency         int       `json:"max_concurrency" db:"max_concurrency"`
	PreferredVulnTypes     []string  `json:"preferred_vuln_types" db:"preferred_vuln_types"`
	PayoutHint             any       `json:"payout_hint" db:"payout_hint"`
	Notes                  string    `json:"notes" db:"notes"`
	CreatedAt              time.Time `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time `json:"updated_at" db:"updated_at"`
}

// CreateSRCProgramRequest 创建 SRC 程序请求
type CreateSRCProgramRequest struct {
	Name                   string   `json:"name"`
	Platform               string   `json:"platform"`
	ProgramURL             string   `json:"program_url"`
	RulesURL               string   `json:"rules_url"`
	AllowAutomation        bool     `json:"allow_automation"`
	AllowDirBrute          bool     `json:"allow_dir_brute"`
	AllowWeakPassword      bool     `json:"allow_weak_password"`
	AllowAuthenticatedTest bool     `json:"allow_authenticated_test"`
	MaxRPS                 int      `json:"max_rps"`
	MaxConcurrency         int      `json:"max_concurrency"`
	PreferredVulnTypes     []string `json:"preferred_vuln_types"`
	PayoutHint             any      `json:"payout_hint"`
	Notes                  string   `json:"notes"`
}

// UpdateSRCProgramRequest 更新 SRC 程序请求
type UpdateSRCProgramRequest struct {
	Name                   *string  `json:"name,omitempty"`
	Platform               *string  `json:"platform,omitempty"`
	ProgramURL             *string  `json:"program_url,omitempty"`
	RulesURL               *string  `json:"rules_url,omitempty"`
	AllowAutomation        *bool    `json:"allow_automation,omitempty"`
	AllowDirBrute          *bool    `json:"allow_dir_brute,omitempty"`
	AllowWeakPassword      *bool    `json:"allow_weak_password,omitempty"`
	AllowAuthenticatedTest *bool    `json:"allow_authenticated_test,omitempty"`
	MaxRPS                 *int     `json:"max_rps,omitempty"`
	MaxConcurrency         *int     `json:"max_concurrency,omitempty"`
	PreferredVulnTypes     []string `json:"preferred_vuln_types,omitempty"`
	PayoutHint             any      `json:"payout_hint,omitempty"`
	Notes                  *string  `json:"notes,omitempty"`
}

// DefaultSRCProgram 返回默认的 SRC 程序配置
func DefaultSRCProgram(projectID string) *SRCProgram {
	return &SRCProgram{
		ProjectID:          projectID,
		Platform:           "other",
		MaxRPS:             5,
		MaxConcurrency:     3,
		PreferredVulnTypes: []string{},
		PayoutHint:         map[string]any{},
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
}

// SupportedPlatforms 支持的 SRC 平台列表
var SupportedPlatforms = []string{
	"360src",
	"butian",
	"hackerone",
	"bugcrowd",
	"intigriti",
	"yeswehack",
	"qianxin",
	"topsec",
	"dbappsecurity",
	"disijie",
	"other",
}

// IsSupportedPlatform 检查是否是支持的平台
func IsSupportedPlatform(platform string) bool {
	for _, p := range SupportedPlatforms {
		if p == platform {
			return true
		}
	}
	return false
}

// MarshalPreferredVulnTypes 序列化首选漏洞类型
func (p *SRCProgram) MarshalPreferredVulnTypes() (string, error) {
	if p.PreferredVulnTypes == nil {
		return "[]", nil
	}
	data, err := json.Marshal(p.PreferredVulnTypes)
	if err != nil {
		return "[]", err
	}
	return string(data), nil
}

// UnmarshalPreferredVulnTypes 反序列化首选漏洞类型
func (p *SRCProgram) UnmarshalPreferredVulnTypes(data string) error {
	if data == "" {
		p.PreferredVulnTypes = []string{}
		return nil
	}
	return json.Unmarshal([]byte(data), &p.PreferredVulnTypes)
}

// MarshalPayoutHint 序列化支付提示
func (p *SRCProgram) MarshalPayoutHint() (string, error) {
	if p.PayoutHint == nil {
		return "{}", nil
	}
	data, err := json.Marshal(p.PayoutHint)
	if err != nil {
		return "{}", err
	}
	return string(data), nil
}

// UnmarshalPayoutHint 反序列化支付提示
func (p *SRCProgram) UnmarshalPayoutHint(data string) error {
	if data == "" {
		p.PayoutHint = map[string]any{}
		return nil
	}
	return json.Unmarshal([]byte(data), &p.PayoutHint)
}
