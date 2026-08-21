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
	EnableSubfinder          bool   `json:"enable_subfinder"`
	EnableDNSx               bool   `json:"enable_dnsx"`
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
	// Slow scanning tools
	EnableFfuf       bool   `json:"enable_ffuf"`
	FfufRateLimit    int    `json:"ffuf_rate_limit"`        // rps
	FfufTimeout      int    `json:"ffuf_timeout"`           // seconds
	// External-scan-only fields
	EnablePassiveSearch      bool   `json:"enable_passive_search"`
	EnableKatana             bool   `json:"enable_katana"`
	KatanaMaxDepth           int    `json:"katana_max_depth"`
	KatanaRateLimit          int    `json:"katana_rate_limit"`
	KatanaTimeout            int    `json:"katana_timeout"` // per-request seconds
	SkipPortscanOnCDNHost    bool   `json:"skip_portscan_on_cdn_host"`
	NucleiRequireFingerprint bool   `json:"nuclei_require_fingerprint"`
	PassiveSearchResultLimit int    `json:"passive_search_result_limit"`
	PassiveSearchConcurrency int    `json:"passive_search_concurrency"`
	EnablePassiveJunkFilter  bool   `json:"enable_passive_junk_filter"`
	PassiveJunkKeywords      string `json:"passive_junk_keywords"`
}



const DefaultFfufDictionaryID = "builtin:path/top100.txt"
// DefaultPipelineConfig is LEGACY: it carries the retired internal-mode baseline
// (high naabu rate/threads, no CDN skip, broad depth). It is retained only as the
// structural base for the external variants and for reading saved internal-shaped
// configs. New code must use DefaultExternalPipelineConfig for an internet scan.
func DefaultPipelineConfig() PipelineConfig {
	return PipelineConfig{
		EnableSubfinder:          true,
		EnableDNSx:               true,
		EnableCDNFilter:          true,
		PortRange:                "high-risk",
		NaabuRate:                1000,
		NaabuThreads:             100,
		NaabuTimeout:             5000, // milliseconds (naabu CLI default is 1000ms)
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
		// Slow scanning defaults — very low rate for background operation
		EnableFfuf:       true,
		FfufRateLimit:    6, // rps
		FfufTimeout:      30,
		EnableKatana:    true,
		KatanaMaxDepth:  2,
		KatanaRateLimit: 10,
		KatanaTimeout:   10,
	}
}

// DefaultExternalPipelineConfig returns the baseline configuration for an
// internet-mode scan — deliberately more conservative on port range,
// rate-limiting, and scan depth. This is the canonical internet scan default
// since P1-1; the retired "internal" mode must not fall back to it implicitly.
func DefaultExternalPipelineConfig() PipelineConfig {
	cfg := DefaultPipelineConfig()
	cfg.PortRange = "top100"
	cfg.NaabuRate = 150
	cfg.NaabuThreads = 30
	cfg.NucleiScanDepth = "workflow"
	cfg.NucleiRateLimit = 10
	cfg.NucleiConcurrency = 5
	cfg.NucleiRateLimitPerMinute = 30
	cfg.FfufRateLimit = 4
	cfg.EnablePassiveSearch = true
	cfg.EnableKatana = true
	cfg.KatanaMaxDepth = 2
	cfg.KatanaRateLimit = 10
	cfg.KatanaTimeout = 10
	cfg.SkipPortscanOnCDNHost = true
	cfg.NucleiRequireFingerprint = true
	cfg.PassiveSearchResultLimit = 500
	cfg.PassiveSearchConcurrency = 3
	cfg.EnablePassiveJunkFilter = true
	return cfg
}

// DefaultExternalLowNoisePipelineConfig returns a low-noise external config.
func DefaultExternalLowNoisePipelineConfig() PipelineConfig {
	cfg := DefaultExternalPipelineConfig()
	cfg.PortRange = "top100"
	cfg.NaabuRate = 100
	cfg.NaabuThreads = 20
	cfg.NucleiScanDepth = "tags"
	cfg.NucleiRateLimit = 5
	cfg.NucleiConcurrency = 3
	cfg.NucleiRateLimitPerMinute = 20
	cfg.NucleiRequireFingerprint = true
	cfg.EnableFfuf = false
	cfg.FfufRateLimit = 3
	cfg.FfufTimeout = 20
	cfg.EnableKatana = false
	cfg.EnablePassiveSearch = true
	cfg.PassiveSearchResultLimit = 300
	cfg.PassiveSearchConcurrency = 2
	return cfg
}

// DefaultExternalStandardPipelineConfig returns a standard external config.
func DefaultExternalStandardPipelineConfig() PipelineConfig {
	return DefaultExternalPipelineConfig()
}
