package models

import "time"

type DictionaryCategory string

const (
	DictionaryCategoryDirscan   DictionaryCategory = "dirscan"
	DictionaryCategorySubdomain DictionaryCategory = "subdomain"
	DictionaryCategoryVhost     DictionaryCategory = "vhost"
	DictionaryCategoryCustom    DictionaryCategory = "custom"
)

type Dictionary struct {
	ID          string             `json:"id" db:"id"`
	Name        string             `json:"name" db:"name"`
	Description string             `json:"description,omitempty" db:"description"`
	Category    DictionaryCategory `json:"category" db:"category"`
	FilePath    string             `json:"file_path" db:"file_path"`
	LineCount   int                `json:"line_count" db:"line_count"`
	SizeBytes   int64              `json:"size_bytes" db:"size_bytes"`
	CreatedAt   time.Time          `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at" db:"updated_at"`
}
