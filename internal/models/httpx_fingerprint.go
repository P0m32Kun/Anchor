package models

import "time"

type HttpxFingerprintType string

const (
	HttpxFingerprintTypeFavicon    HttpxFingerprintType = "favicon"
	HttpxFingerprintTypeTechDetect HttpxFingerprintType = "tech_detect"
)

type HttpxFingerprint struct {
	ID          string               `json:"id" db:"id"`
	Name        string               `json:"name" db:"name"`
	Description string               `json:"description,omitempty" db:"description"`
	Type        HttpxFingerprintType `json:"type" db:"type"`
	FilePath    string               `json:"file_path" db:"file_path"`
	Enabled     bool                 `json:"enabled" db:"enabled"`
	Builtin     bool                 `json:"builtin" db:"builtin"`
	CreatedAt   time.Time            `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time            `json:"updated_at" db:"updated_at"`
}
