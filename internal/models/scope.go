package models

import (
	"database/sql/driver"
	"fmt"
	"time"
)

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

// --- JSON helpers for ScopeAction ---

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

// --- JSON helpers for ScopeDecisionResult ---

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
