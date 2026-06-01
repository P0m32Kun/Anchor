package models

import "time"

// --- ExcludedDomain ---

// ExcludedDomain represents a domain that should be excluded from scanning.
// This includes both built-in defaults and user-configured custom exclusions.
type ExcludedDomain struct {
	ID        string    `json:"id" db:"id"`
	Domain    string    `json:"domain" db:"domain"`
	Reason    string    `json:"reason" db:"reason"`
	Builtin   bool      `json:"builtin" db:"builtin"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}
