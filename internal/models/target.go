package models

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// --- Target ---

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

// --- IPDiscoveryResult ---

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

// --- JSON helpers for TargetType ---

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
