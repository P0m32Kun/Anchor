package models

import "time"

// FindingTemplate is a globally shared vulnerability knowledge entry.
// At report generation time, a finding is looked up by (source_tool, match_key)
// against this table; matching, enabled templates override the finding's title,
// severity, summary and remediation when those template fields are non-empty.
type FindingTemplate struct {
	ID          string    `json:"id" db:"id"`
	SourceTool  string    `json:"source_tool" db:"source_tool"`
	MatchKey    string    `json:"match_key" db:"match_key"`
	Title       string    `json:"title" db:"title"`
	Severity    string    `json:"severity" db:"severity"`
	Summary     string    `json:"summary" db:"summary"`
	Remediation string    `json:"remediation" db:"remediation"`
	Enabled     bool      `json:"enabled" db:"enabled"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}
