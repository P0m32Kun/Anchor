package models

import (
	"encoding/json"
	"time"
)

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

// --- WorkerNode ---

type WorkerMode string

const (
	WorkerModeRemote WorkerMode = "remote"
)

type WorkerStatus string

const (
	WorkerStatusOnline  WorkerStatus = "online"
	WorkerStatusOffline WorkerStatus = "offline"
	WorkerStatusBusy    WorkerStatus = "busy"
	WorkerStatusError   WorkerStatus = "error"
)

type WorkerNode struct {
	ID               string       `json:"id" db:"id"`
	Name             string       `json:"name" db:"name"`
	Endpoint         string       `json:"endpoint" db:"endpoint"`
	Mode             WorkerMode   `json:"mode" db:"mode"`
	Status           WorkerStatus `json:"status" db:"status"`
	TrustLevel       string       `json:"trust_level" db:"trust_level"`
	NetworkProfile   string       `json:"network_profile" db:"network_profile"`
	Capabilities     string       `json:"capabilities" db:"capabilities"`
	ToolVersions     string       `json:"tool_versions" db:"tool_versions"`
	TemplateVersions string       `json:"template_versions" db:"template_versions"`
	MaxConcurrency   int          `json:"max_concurrency" db:"max_concurrency"`
	LastSeen         *time.Time   `json:"last_seen" db:"last_seen"`
	CreatedAt        time.Time    `json:"created_at" db:"created_at"`
	RevokedAt        *time.Time   `json:"revoked_at" db:"revoked_at"`
}

// --- WorkerHealthCheck ---

type HealthCheckStatus string

const (
	HealthCheckReady           HealthCheckStatus = "ready"
	HealthCheckMissing         HealthCheckStatus = "missing"
	HealthCheckVersionMismatch HealthCheckStatus = "version_mismatch"
	HealthCheckConfigError     HealthCheckStatus = "config_error"
	HealthCheckPermissionError HealthCheckStatus = "permission_error"
)

type WorkerHealthCheck struct {
	ID        string            `json:"id" db:"id"`
	WorkerID  string            `json:"worker_id" db:"worker_id"`
	Tool      string            `json:"tool" db:"tool"`
	Status    HealthCheckStatus `json:"status" db:"status"`
	Version   string            `json:"version" db:"version"`
	Details   string            `json:"details" db:"details"`
	CheckedAt time.Time         `json:"checked_at" db:"checked_at"`
}

// --- Engine Credential ---

type EngineCredential struct {
	ID        string    `json:"id" db:"id"`
	Engine    string    `json:"engine" db:"engine"`
	APIKey    string    `json:"api_key" db:"api_key"`
	Extra     *string   `json:"extra,omitempty" db:"extra"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// --- Pipeline Config ---

type PipelineConfig struct {
	EnableFOFA               bool   `json:"enable_fofa"`
	FofaResultLimit          int    `json:"fofa_result_limit"`
	FofaConcurrency          int    `json:"fofa_concurrency"`
	EnableSubfinder          bool   `json:"enable_subfinder"`
	SubfinderRateLimit       int    `json:"subfinder_rate_limit"`
	SubfinderThreads         int    `json:"subfinder_threads"`
	SubfinderTimeout         int    `json:"subfinder_timeout"`
	EnableDNSx               bool   `json:"enable_dnsx"`
	DNSxRateLimit            int    `json:"dnsx_rate_limit"`
	DNSxThreads              int    `json:"dnsx_threads"`
	DNSxTimeout              int    `json:"dnsx_timeout"`
	EnableCDNFilter          bool   `json:"enable_cdn_filter"`
	PortRange                string `json:"port_range"`
	NaabuRate                int    `json:"naabu_rate"`
	NaabuThreads             int    `json:"naabu_threads"`
	NaabuTimeout             int    `json:"naabu_timeout"`
	EnableNerva              bool   `json:"enable_nerva"`
	NervaFastMode            bool   `json:"nerva_fast_mode"`
	NervaRateLimit           int    `json:"nerva_rate_limit"`
	NervaWorkers             int    `json:"nerva_workers"`
	NervaTimeout             int    `json:"nerva_timeout"`
	EnableHttpx              bool   `json:"enable_httpx"`
	HttpxRateLimit           int    `json:"httpx_rate_limit"`
	HttpxThreads             int    `json:"httpx_threads"`
	EnableNuclei             bool   `json:"enable_nuclei"`
	NucleiRateLimit          int    `json:"nuclei_rate_limit"`
	NucleiRateLimitPerMinute int    `json:"nuclei_rate_limit_per_min"`
	NucleiConcurrency        int    `json:"nuclei_concurrency"`
	NucleiScanDepth          string `json:"nuclei_scan_depth"`
}

func DefaultPipelineConfig() PipelineConfig {
	return PipelineConfig{
		EnableFOFA:               true,
		FofaResultLimit:          500,
		FofaConcurrency:          5,
		EnableSubfinder:          true,
		SubfinderRateLimit:       50,
		SubfinderThreads:         10,
		SubfinderTimeout:         300,
		EnableDNSx:               true,
		DNSxRateLimit:            100,
		DNSxThreads:              50,
		DNSxTimeout:              5,
		EnableCDNFilter:          true,
		PortRange:                "top1000",
		NaabuRate:                1000,
		NaabuThreads:             100,
		NaabuTimeout:             600,
		EnableNerva:              true,
		NervaFastMode:            true,
		NervaRateLimit:           100,
		NervaWorkers:             50,
		NervaTimeout:             10,
		EnableHttpx:              true,
		HttpxRateLimit:           150,
		HttpxThreads:             50,
		EnableNuclei:             true,
		NucleiRateLimit:          100,
		NucleiRateLimitPerMinute: 0,
		NucleiConcurrency:        25,
		NucleiScanDepth:          "tags",
	}
}

// --- DNS ---

type DNSRecord struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	Domain    string    `json:"domain"`
	IPs       []string  `json:"ips"`
	CNAMEs    []string  `json:"cnames,omitempty"`
	TTL       uint32    `json:"ttl"`
	Resolver  string    `json:"resolver"`
	CreatedAt time.Time `json:"created_at"`
}

// --- CDN ---

type CDNResult struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	IP        string    `json:"ip"`
	IsCDN     bool      `json:"is_cdn"`
	Provider  string    `json:"provider,omitempty"`
	Type      string    `json:"type,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// --- Service Fingerprint ---

type ServiceFingerprint struct {
	ID        string                 `json:"id"`
	ProjectID string                 `json:"project_id"`
	IP        string                 `json:"ip"`
	Port      int                    `json:"port"`
	Protocol  string                 `json:"protocol"`
	IsWeb     bool                   `json:"is_web"`
	Service   string                 `json:"service"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Source    string                 `json:"source"`
	CreatedAt time.Time              `json:"created_at"`
}

// --- Dashboard ---

type DashboardStats struct {
	TotalProjects   int                    `json:"total_projects"`
	ActiveRuns      int                    `json:"active_runs"`
	PendingFindings int                    `json:"pending_findings"`
	OnlineWorkers   int                    `json:"online_workers"`
	RecentRuns      []*DashboardRunItem    `json:"recent_runs"`
	RecentFindings  []*DashboardFindingItem `json:"recent_findings"`
}

type DashboardRunItem struct {
	ID          string     `json:"id"`
	ProjectID   string     `json:"project_id"`
	ProjectName string     `json:"project_name"`
	Name        string     `json:"name"`
	Status      string     `json:"status"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

type DashboardFindingItem struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	ProjectName string    `json:"project_name"`
	Title       string    `json:"title"`
	Severity    string    `json:"severity"`
	CreatedAt   time.Time `json:"created_at"`
}
