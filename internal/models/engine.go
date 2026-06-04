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
	// Slow scanning tools
	EnableFfuf       bool   `json:"enable_ffuf"`
	FfufRateLimit    int    `json:"ffuf_rate_limit"`        // rps
	FfufTimeout      int    `json:"ffuf_timeout"`           // seconds
	FfufDictionaryID string `json:"ffuf_dictionary_id"`     // optional
	// External-scan-only fields
	EnablePassiveSearch      bool   `json:"enable_passive_search"`
	EnablePassiveCert        bool   `json:"enable_passive_cert"`
	EnablePassiveURL         bool   `json:"enable_passive_url"`
	SubfinderMode            string `json:"subfinder_mode"`             // passive | active | off
	EnableKatana             bool   `json:"enable_katana"`
	KatanaMaxDepth           int    `json:"katana_max_depth"`
	KatanaRateLimit          int    `json:"katana_rate_limit"`
	KatanaTimeout            int    `json:"katana_timeout"` // per-request seconds
	FfufTier                 string `json:"ffuf_tier"`                  // small | medium | off
	SkipPortscanOnCDNHost    bool   `json:"skip_portscan_on_cdn_host"`
	NucleiRequireFingerprint bool   `json:"nuclei_require_fingerprint"`
	PassiveSearchResultLimit int    `json:"passive_search_result_limit"`
	PassiveSearchConcurrency int    `json:"passive_search_concurrency"`
	// SubfinderProviderConfig is an optional YAML string for subfinder's
	// provider-config.yaml. When non-empty, the pipeline writes it to a temp
	// file and passes -pc <path> to subfinder. The file is automatically
	// synced to remote workers via the dispatcher's input_files mechanism.
	// Leave empty to use the worker's default (~/.config/subfinder/provider-config.yaml).
	SubfinderProviderConfig string `json:"subfinder_provider_config,omitempty"`
	// ScanMode is the scan mode preset: "default" | "src_low_noise"
	// src_low_noise enables only bounded ffuf dictionaries and high-signal Nuclei workflows.
	ScanMode string `json:"scan_mode,omitempty"`
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
		FfufDictionaryID: "",
		EnableKatana:    true,
		KatanaMaxDepth:  2,
		KatanaRateLimit: 10,
		KatanaTimeout:   10,
	}
}

// DefaultExternalPipelineConfig returns the baseline configuration for
// external-mode scans — deliberately more conservative on port range,
// rate-limiting, and scan depth than the internal default.
func DefaultExternalPipelineConfig() PipelineConfig {
	cfg := DefaultPipelineConfig()
	cfg.PortRange = "top100"
	cfg.NaabuRate = 300
	cfg.NaabuThreads = 50
	cfg.NucleiScanDepth = "workflow"
	cfg.NucleiRateLimit = 20
	cfg.NucleiConcurrency = 5
	cfg.NucleiRateLimitPerMinute = 30
	cfg.FfufRateLimit = 4
	cfg.EnablePassiveSearch = true
	cfg.EnablePassiveCert = true
	cfg.EnablePassiveURL = true
	cfg.SubfinderMode = "passive"
	cfg.EnableKatana = true
	cfg.KatanaMaxDepth = 2
	cfg.KatanaRateLimit = 10
	cfg.KatanaTimeout = 10
	cfg.FfufTier = "small"
	cfg.SkipPortscanOnCDNHost = true
	cfg.NucleiRequireFingerprint = true
	cfg.PassiveSearchResultLimit = 500
	cfg.PassiveSearchConcurrency = 3
	return cfg
}

// DefaultLowNoisePipelineConfig returns the configuration for SRC low-noise scans.
// It enables only bounded ffuf dictionaries and high-signal Nuclei workflows.
// Use this for SRC bounty programs where noise must be minimized.
func DefaultLowNoisePipelineConfig() PipelineConfig {
	cfg := DefaultExternalPipelineConfig()
	cfg.ScanMode = "src_low_noise"
	// Port scanning: top100 only
	cfg.PortRange = "top100"
	cfg.NaabuRate = 200
	cfg.NaabuThreads = 30
	// Nuclei: high-signal only, low rate
	cfg.EnableNuclei = true
	cfg.NucleiScanDepth = "tags"        // tags only, no full workflows
	cfg.NucleiRateLimit = 5             // very low rate
	cfg.NucleiConcurrency = 3
	cfg.NucleiRateLimitPerMinute = 20
	cfg.NucleiRequireFingerprint = true
	// Ffuf: enabled with small dictionary only
	cfg.EnableFfuf = true
	cfg.FfufTier = "small"
	cfg.FfufRateLimit = 3
	cfg.FfufTimeout = 20
	// Katana: disabled (too noisy)
	cfg.EnableKatana = false
	// Passive: enabled
	cfg.EnablePassiveSearch = true
	cfg.EnablePassiveCert = true
	cfg.EnablePassiveURL = true
	cfg.SubfinderMode = "passive"
	cfg.PassiveSearchResultLimit = 300
	cfg.PassiveSearchConcurrency = 2
	return cfg
}

// IsLowNoiseMode returns true if the scan mode is src_low_noise.
func (c PipelineConfig) IsLowNoiseMode() bool {
	return c.ScanMode == "src_low_noise"
}
