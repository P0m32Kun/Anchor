package models

import "time"

// FindingTemplate is a vulnerability knowledge entry. Templates may be seeded
// from the repository's docs/templates/vuln-templates.json (is_builtin=1) or
// created in the UI (is_builtin=0). At report generation time, a finding is
// looked up by (source_tool, match_key); matching, enabled templates override
// the finding's title / severity / summary / remediation when those template
// fields are non-empty.
//
// Provenance fields:
//
//	IsBuiltin       — true when this row came from the repo seed JSON.
//	UserModified    — true when a builtin row was edited locally; locks it
//	                  from being auto-overwritten on the next image upgrade.
//	BuiltinPayload  — JSON of the latest upstream version of a builtin row,
//	                  used by the UI to show "upstream has a newer version"
//	                  and to power the "accept upstream" action.
type FindingTemplate struct {
	ID             string    `json:"id" db:"id"`
	SourceTool     string    `json:"source_tool" db:"source_tool"`
	// MatchKeys 是内存中的匹配键列表(一个词条可挂多个)。
	// MatchKeysJSON 是数据库存储格式(JSON 编码的字符串数组)。
	// 老字段 MatchKey 保留一个 release 以兼容回滚;新代码只读写 MatchKeys/MatchKeysJSON。
	MatchKeys      []string `json:"match_keys" db:"-"`
	MatchKeysJSON  string   `json:"-"            db:"match_keys"`
	MatchKey       string   `json:"match_key"    db:"match_key"`
	Title          string    `json:"title" db:"title"`
	Severity       string    `json:"severity" db:"severity"`
	Summary        string    `json:"summary" db:"summary"`
	Remediation    string    `json:"remediation" db:"remediation"`
	Enabled        bool      `json:"enabled" db:"enabled"`
	IsBuiltin      bool      `json:"is_builtin" db:"is_builtin"`
	UserModified   bool      `json:"user_modified" db:"user_modified"`
	BuiltinPayload string    `json:"builtin_payload" db:"builtin_payload"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}
