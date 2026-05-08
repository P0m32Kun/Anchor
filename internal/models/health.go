package models

import "time"

// --- ToolHealth ---

type ToolHealth struct {
	ID               string    `json:"id" db:"id"`
	Tool             string    `json:"tool" db:"tool"`
	BinaryPath       string    `json:"binary_path" db:"binary_path"`
	Version          string    `json:"version" db:"version"`
	TemplatePath     *string   `json:"template_path" db:"template_path"`
	WorkdirWritable  bool      `json:"workdir_writable" db:"workdir_writable"`
	NetworkAvailable bool      `json:"network_available" db:"network_available"`
	DNSAvailable     bool      `json:"dns_available" db:"dns_available"`
	ProxyReachable   *bool     `json:"proxy_reachable" db:"proxy_reachable"`
	LastCheckAt      time.Time `json:"last_check_at" db:"last_check_at"`
}

// --- ToolTemplate ---

type ToolTemplate struct {
	ID                         string    `json:"id" db:"id"`
	Name                       string    `json:"name" db:"name"`
	Description                string    `json:"description" db:"description"`
	ProfileType                string    `json:"profile_type" db:"profile_type"`
	ToolsJSON                  string    `json:"tools_json" db:"tools_json"`
	DefaultMaxConcurrency      int       `json:"default_max_concurrency" db:"default_max_concurrency"`
	ScreenshotEnabled          bool      `json:"screenshot_enabled" db:"screenshot_enabled"`
	DirectoryBruteforceEnabled bool      `json:"directory_bruteforce_enabled" db:"directory_bruteforce_enabled"`
	NucleiSeverityFilter       string    `json:"nuclei_severity_filter" db:"nuclei_severity_filter"`
	CreatedAt                  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt                  time.Time `json:"updated_at" db:"updated_at"`
}

type TemplateTool struct {
	Tool    string `json:"tool"`
	Enabled bool   `json:"enabled"`
	Rate    int    `json:"rate"`
}

func (t *ToolTemplate) Tools() ([]TemplateTool, error) {
	var tools []TemplateTool
	if err := json.Unmarshal([]byte(t.ToolsJSON), &tools); err != nil {
		return nil, err
	}
	return tools, nil
}
