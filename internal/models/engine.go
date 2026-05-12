package models

import "time"

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
	EnableNmapService        bool   `json:"enable_nmap_service"`
	NmapServiceTimeout       int    `json:"nmap_service_timeout"`
	EnableHttpx              bool   `json:"enable_httpx"`
	HttpxRateLimit           int    `json:"httpx_rate_limit"`
	HttpxThreads             int    `json:"httpx_threads"`
	EnableNuclei             bool   `json:"enable_nuclei"`
	NucleiRateLimit          int    `json:"nuclei_rate_limit"`          // -rl: requests per second
	NucleiRateLimitPerMinute int    `json:"nuclei_rate_limit_per_min"` // -rlm: requests per minute (for sensitive targets)
	NucleiConcurrency        int    `json:"nuclei_concurrency"`        // -c: parallel templates/hosts
	NucleiScanDepth          string `json:"nuclei_scan_depth"`         // "workflow" | "tags" | "both"
}

func DefaultPipelineConfig() PipelineConfig {
	return PipelineConfig{
		EnableFOFA:               true,
		FofaResultLimit:          500,
		FofaConcurrency:          5,
		EnableSubfinder:          true,
		SubfinderRateLimit:       50,
		SubfinderThreads:         10,
		SubfinderTimeout:         30, // seconds; aligns with subfinder CLI default
		EnableDNSx:               true,
		DNSxRateLimit:            100,
		DNSxThreads:              50,
		DNSxTimeout:              5,
		EnableCDNFilter:          true,
		PortRange:                "top1000",
		NaabuRate:                1000,
		NaabuThreads:             100,
		NaabuTimeout:             5, // seconds; converted to ms when passed to naabu CLI
		EnableNmapService:        true,
		NmapServiceTimeout:       180, // seconds; per-host --host-timeout for -sV scan
		EnableHttpx:              true,
		HttpxRateLimit:           150,
		HttpxThreads:             50,
		EnableNuclei:             true,
		NucleiRateLimit:          100,
		NucleiRateLimitPerMinute: 0, // disabled by default, set for sensitive targets
		NucleiConcurrency:        25,
		NucleiScanDepth:          "tags",
	}
}
