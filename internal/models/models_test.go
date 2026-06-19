package models

import (
	"encoding/json"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// engine.go — DefaultPipelineConfig
// ---------------------------------------------------------------------------

func TestDefaultPipelineConfig(t *testing.T) {
	cfg := DefaultPipelineConfig()

	checks := []struct {
		name string
		got  interface{}
		want interface{}
	}{
		{"EnableFOFA", cfg.EnableFOFA, true},
		{"FofaResultLimit", cfg.FofaResultLimit, 500},
		{"FofaConcurrency", cfg.FofaConcurrency, 5},
		{"EnableSubfinder", cfg.EnableSubfinder, true},
		{"SubfinderRateLimit", cfg.SubfinderRateLimit, 50},
		{"SubfinderThreads", cfg.SubfinderThreads, 10},
		{"SubfinderTimeout", cfg.SubfinderTimeout, 30},
		{"EnableDNSx", cfg.EnableDNSx, true},
		{"DNSxRateLimit", cfg.DNSxRateLimit, 100},
		{"DNSxThreads", cfg.DNSxThreads, 50},
		{"DNSxTimeout", cfg.DNSxTimeout, 5},
		{"EnableCDNFilter", cfg.EnableCDNFilter, true},
		{"PortRange", cfg.PortRange, "high-risk"},
		{"NaabuRate", cfg.NaabuRate, 1000},
		{"NaabuThreads", cfg.NaabuThreads, 100},
		{"NaabuTimeout", cfg.NaabuTimeout, 5000},
		{"EnableNmapService", cfg.EnableNmapService, true},
		{"NmapServiceTimeout", cfg.NmapServiceTimeout, 180},
		{"EnableHttpx", cfg.EnableHttpx, true},
		{"HttpxRateLimit", cfg.HttpxRateLimit, 150},
		{"HttpxThreads", cfg.HttpxThreads, 50},
		{"EnableNuclei", cfg.EnableNuclei, true},
		{"NucleiRateLimit", cfg.NucleiRateLimit, 100},
		{"NucleiRateLimitPerMinute", cfg.NucleiRateLimitPerMinute, 0},
		{"NucleiConcurrency", cfg.NucleiConcurrency, 25},
		{"NucleiScanDepth", cfg.NucleiScanDepth, "tags"},
		{"EnableFfuf", cfg.EnableFfuf, true},
		{"FfufRateLimit", cfg.FfufRateLimit, 6},
		{"FfufTimeout", cfg.FfufTimeout, 30},
		{"EnableKatana", cfg.EnableKatana, true},
		{"KatanaMaxDepth", cfg.KatanaMaxDepth, 2},
		{"KatanaRateLimit", cfg.KatanaRateLimit, 10},
		{"KatanaTimeout", cfg.KatanaTimeout, 10},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("DefaultPipelineConfig().%s = %v, want %v", c.name, c.got, c.want)
		}
	}
}

// ---------------------------------------------------------------------------
// engine.go — DefaultExternalPipelineConfig
// ---------------------------------------------------------------------------

func TestDefaultExternalPipelineConfigFull(t *testing.T) {
	cfg := DefaultExternalPipelineConfig()

	// Inherits from DefaultPipelineConfig
	if !cfg.EnableFOFA {
		t.Error("EnableFOFA should be inherited as true")
	}
	if !cfg.EnableSubfinder {
		t.Error("EnableSubfinder should be inherited as true")
	}

	// External-specific overrides
	checks := []struct {
		name string
		got  interface{}
		want interface{}
	}{
		{"PortRange", cfg.PortRange, "top100"},
		{"NaabuRate", cfg.NaabuRate, 150},
		{"NaabuThreads", cfg.NaabuThreads, 30},
		{"NucleiScanDepth", cfg.NucleiScanDepth, "workflow"},
		{"NucleiRateLimit", cfg.NucleiRateLimit, 10},
		{"NucleiConcurrency", cfg.NucleiConcurrency, 5},
		{"NucleiRateLimitPerMinute", cfg.NucleiRateLimitPerMinute, 30},
		{"FfufRateLimit", cfg.FfufRateLimit, 4},
		{"EnablePassiveSearch", cfg.EnablePassiveSearch, true},
		{"EnablePassiveCert", cfg.EnablePassiveCert, true},
		{"EnablePassiveURL", cfg.EnablePassiveURL, true},
		{"SubfinderMode", cfg.SubfinderMode, "passive"},
		{"EnableKatana", cfg.EnableKatana, true},
		{"KatanaMaxDepth", cfg.KatanaMaxDepth, 2},
		{"KatanaRateLimit", cfg.KatanaRateLimit, 10},
		{"KatanaTimeout", cfg.KatanaTimeout, 10},
		{"FfufTier", cfg.FfufTier, "small"},
		{"SkipPortscanOnCDNHost", cfg.SkipPortscanOnCDNHost, true},
		{"NucleiRequireFingerprint", cfg.NucleiRequireFingerprint, true},
		{"PassiveSearchResultLimit", cfg.PassiveSearchResultLimit, 500},
		{"PassiveSearchConcurrency", cfg.PassiveSearchConcurrency, 3},
		{"EnablePassiveJunkFilter", cfg.EnablePassiveJunkFilter, true},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("DefaultExternalPipelineConfig().%s = %v, want %v", c.name, c.got, c.want)
		}
	}
}

// ---------------------------------------------------------------------------
// engine.go — DefaultExternalLowNoisePipelineConfig
// ---------------------------------------------------------------------------

func TestDefaultExternalLowNoisePipelineConfig(t *testing.T) {
	cfg := DefaultExternalLowNoisePipelineConfig()

	checks := []struct {
		name string
		got  interface{}
		want interface{}
	}{
		{"ScanMode", cfg.ScanMode, "external"},
		{"NoiseLevel", cfg.NoiseLevel, "low"},
		{"PortRange", cfg.PortRange, "top100"},
		{"NaabuRate", cfg.NaabuRate, 100},
		{"NaabuThreads", cfg.NaabuThreads, 20},
		{"NucleiScanDepth", cfg.NucleiScanDepth, "tags"},
		{"NucleiRateLimit", cfg.NucleiRateLimit, 5},
		{"NucleiConcurrency", cfg.NucleiConcurrency, 3},
		{"NucleiRateLimitPerMinute", cfg.NucleiRateLimitPerMinute, 20},
		{"NucleiRequireFingerprint", cfg.NucleiRequireFingerprint, true},
		{"EnableFfuf", cfg.EnableFfuf, false},
		{"FfufTier", cfg.FfufTier, "small"},
		{"FfufRateLimit", cfg.FfufRateLimit, 3},
		{"FfufTimeout", cfg.FfufTimeout, 20},
		{"EnableKatana", cfg.EnableKatana, false},
		{"EnablePassiveSearch", cfg.EnablePassiveSearch, true},
		{"EnablePassiveCert", cfg.EnablePassiveCert, true},
		{"EnablePassiveURL", cfg.EnablePassiveURL, true},
		{"SubfinderMode", cfg.SubfinderMode, "passive"},
		{"PassiveSearchResultLimit", cfg.PassiveSearchResultLimit, 300},
		{"PassiveSearchConcurrency", cfg.PassiveSearchConcurrency, 2},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("DefaultExternalLowNoisePipelineConfig().%s = %v, want %v", c.name, c.got, c.want)
		}
	}
}

// ---------------------------------------------------------------------------
// engine.go — DefaultExternalStandardPipelineConfig
// ---------------------------------------------------------------------------

func TestDefaultExternalStandardPipelineConfig(t *testing.T) {
	cfg := DefaultExternalStandardPipelineConfig()
	ext := DefaultExternalPipelineConfig()

	// Should be identical to DefaultExternalPipelineConfig
	if cfg.PortRange != ext.PortRange {
		t.Errorf("Standard PortRange = %q, want %q (same as External)", cfg.PortRange, ext.PortRange)
	}
	if cfg.NucleiRateLimit != ext.NucleiRateLimit {
		t.Errorf("Standard NucleiRateLimit = %d, want %d", cfg.NucleiRateLimit, ext.NucleiRateLimit)
	}
}

// ---------------------------------------------------------------------------
// engine.go — NormalizeScanMode
// ---------------------------------------------------------------------------

func TestNormalizeScanMode(t *testing.T) {
	tests := []struct {
		name       string
		mode       string
		noise      string
		wantMode   string
		wantNoise  string
	}{
		// external passthrough
		{"external with noise", "external", "low", "external", "low"},
		{"external empty noise", "external", "", "external", "standard"},
		// standard → external, default noise
		{"standard empty noise", "standard", "", "external", "standard"},
		{"standard with noise", "standard", "high", "external", "high"},
		// external_low → external, low noise default
		{"external_low empty noise", "external_low", "", "external", "low"},
		{"external_low with noise", "external_low", "high", "external", "high"},
		// src_low_noise → external, low noise default
		{"src_low_noise empty noise", "src_low_noise", "", "external", "low"},
		{"src_low_noise with noise", "src_low_noise", "standard", "external", "standard"},
		// internal passthrough
		{"internal no noise", "internal", "", "internal", ""},
		{"internal with noise", "internal", "low", "internal", "low"},
		// watch → external
		{"watch empty noise", "watch", "", "external", "standard"},
		{"watch with noise", "watch", "low", "external", "low"},
		// unknown → external
		{"unknown mode", "something", "", "external", ""},
		{"unknown with noise", "random", "high", "external", "high"},
		// empty mode
		{"empty mode", "", "", "external", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMode, gotNoise := NormalizeScanMode(tt.mode, tt.noise)
			if gotMode != tt.wantMode || gotNoise != tt.wantNoise {
				t.Errorf("NormalizeScanMode(%q, %q) = (%q, %q), want (%q, %q)",
					tt.mode, tt.noise, gotMode, gotNoise, tt.wantMode, tt.wantNoise)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// certificate.go — DaysUntilExpiry / IsExpiringSoon
// ---------------------------------------------------------------------------

func TestDaysUntilExpiry(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name    string
		notAfter time.Time
		wantMin int // at least this many days
		wantMax int // at most this many days
	}{
		{"zero time", time.Time{}, 0, 0},
		{"expired 30d ago", now.AddDate(0, 0, -30), -31, -30},
		{"expires in 365d", now.AddDate(1, 0, 0), 364, 366},
		{"expires in 2 days", now.AddDate(0, 0, 2), 1, 2},
		{"expires in 1h", now.Add(time.Hour), 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Certificate{NotAfter: tt.notAfter}
			got := c.DaysUntilExpiry()
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("DaysUntilExpiry() = %d, want between %d and %d", got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestIsExpiringSoon(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name string
		notAfter time.Time
		days int
		want bool
	}{
		{"not expiring, 90d window", now.AddDate(1, 0, 0), 90, false},
		{"expiring in 30d, 90d window", now.AddDate(0, 0, 30), 90, true},
		{"expiring in 2d, 7d window", now.AddDate(0, 0, 2), 7, true},
		{"already expired", now.AddDate(0, 0, -5), 30, false},
		{"zero time", time.Time{}, 30, false},
		{"expires in 10d, 7d window", now.AddDate(0, 0, 10), 7, false},
		{"expires in 3d, 7d window", now.AddDate(0, 0, 3), 7, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Certificate{NotAfter: tt.notAfter}
			got := c.IsExpiringSoon(tt.days)
			if got != tt.want {
				t.Errorf("IsExpiringSoon(%d) = %v, want %v (DaysUntilExpiry=%d)",
					tt.days, got, tt.want, c.DaysUntilExpiry())
			}
		})
	}
}

// ---------------------------------------------------------------------------
// common.go — ToJSON
// ---------------------------------------------------------------------------

func TestToJSON(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  string
	}{
		{"nil", nil, "null"},
		{"string", "hello", `"hello"`},
		{"int", 42, "42"},
		{"bool", true, "true"},
		{"empty map", map[string]string{}, "{}"},
		{"struct", struct{ Name string }{Name: "test"}, `{"Name":"test"}`},
		{"slice", []int{1, 2, 3}, "[1,2,3]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToJSON(tt.input)
			if string(got) != tt.want {
				t.Errorf("ToJSON(%v) = %q, want %q", tt.input, string(got), tt.want)
			}
		})
	}
}

func TestToJSONInvalidReturnsEmpty(t *testing.T) {
	// channels cannot be marshaled — should return empty
	ch := make(chan int)
	got := ToJSON(ch)
	if got != nil {
		t.Errorf("ToJSON(chan) = %q, want nil", string(got))
	}
}

// ---------------------------------------------------------------------------
// health.go — ToolTemplate.Tools
// ---------------------------------------------------------------------------

func TestToolTemplateTools(t *testing.T) {
	tests := []struct {
		name      string
		toolsJSON string
		wantLen   int
		wantErr   bool
	}{
		{
			name:      "valid tools",
			toolsJSON: `[{"tool":"nuclei","enabled":true,"rate":100},{"tool":"httpx","enabled":false,"rate":50}]`,
			wantLen:   2,
			wantErr:   false,
		},
		{
			name:      "empty array",
			toolsJSON: `[]`,
			wantLen:   0,
			wantErr:   false,
		},
		{
			name:      "invalid json",
			toolsJSON: `not json`,
			wantLen:   0,
			wantErr:   true,
		},
		{
			name:      "empty string",
			toolsJSON: ``,
			wantLen:   0,
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl := &ToolTemplate{ToolsJSON: tt.toolsJSON}
			tools, err := tmpl.Tools()
			if (err != nil) != tt.wantErr {
				t.Errorf("Tools() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(tools) != tt.wantLen {
				t.Errorf("Tools() returned %d tools, want %d", len(tools), tt.wantLen)
			}
		})
	}
}

func TestToolTemplateToolsValues(t *testing.T) {
	tmpl := &ToolTemplate{
		ToolsJSON: `[{"tool":"subfinder","enabled":true,"rate":200}]`,
	}
	tools, err := tmpl.Tools()
	if err != nil {
		t.Fatalf("Tools() unexpected error: %v", err)
	}
	if len(tools) != 1 {
		t.Fatalf("len(tools) = %d, want 1", len(tools))
	}
	if tools[0].Tool != "subfinder" {
		t.Errorf("tools[0].Tool = %q, want subfinder", tools[0].Tool)
	}
	if !tools[0].Enabled {
		t.Error("tools[0].Enabled should be true")
	}
	if tools[0].Rate != 200 {
		t.Errorf("tools[0].Rate = %d, want 200", tools[0].Rate)
	}
}

// ---------------------------------------------------------------------------
// asset.go — AssetType Value/Scan
// ---------------------------------------------------------------------------

func TestAssetTypeValue(t *testing.T) {
	tests := []struct {
		name string
		at   AssetType
		want string
	}{
		{"domain", AssetTypeDomain, "domain"},
		{"ip", AssetTypeIP, "ip"},
		{"cidr", AssetTypeCIDR, "cidr"},
		{"url", AssetTypeURL, "url"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := tt.at.Value()
			if err != nil {
				t.Errorf("Value() error: %v", err)
			}
			if v != tt.want {
				t.Errorf("Value() = %v, want %v", v, tt.want)
			}
		})
	}
}

func TestAssetTypeScan(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    AssetType
		wantErr bool
	}{
		{"string", "domain", AssetTypeDomain, false},
		{"bytes", []byte("ip"), AssetTypeIP, false},
		{"nil", nil, "", false},
		{"int", 42, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var at AssetType
			err := at.Scan(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Scan(%v) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if at != tt.want {
				t.Errorf("Scan(%v) → %v, want %v", tt.input, at, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// finding.go — FindingSeverity Value/Scan
// ---------------------------------------------------------------------------

func TestFindingSeverityValue(t *testing.T) {
	tests := []struct {
		name string
		fs   FindingSeverity
		want string
	}{
		{"info", SeverityInfo, "info"},
		{"low", SeverityLow, "low"},
		{"medium", SeverityMedium, "medium"},
		{"high", SeverityHigh, "high"},
		{"critical", SeverityCritical, "critical"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := tt.fs.Value()
			if err != nil {
				t.Errorf("Value() error: %v", err)
			}
			if v != tt.want {
				t.Errorf("Value() = %v, want %v", v, tt.want)
			}
		})
	}
}

func TestFindingSeverityScan(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    FindingSeverity
		wantErr bool
	}{
		{"string", "high", SeverityHigh, false},
		{"bytes", []byte("critical"), SeverityCritical, false},
		{"nil", nil, "", false},
		{"int", 42, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var fs FindingSeverity
			err := fs.Scan(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Scan(%v) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if fs != tt.want {
				t.Errorf("Scan(%v) → %v, want %v", tt.input, fs, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// finding.go — FindingStatus Value/Scan
// ---------------------------------------------------------------------------

func TestFindingStatusValue(t *testing.T) {
	tests := []struct {
		name string
		fs   FindingStatus
		want string
	}{
		{"new", FindingNew, "new"},
		{"pending_review", FindingPendingReview, "pending_review"},
		{"confirmed", FindingConfirmed, "confirmed"},
		{"false_positive", FindingFalsePositive, "false_positive"},
		{"accepted_risk", FindingAcceptedRisk, "accepted_risk"},
		{"ignored", FindingIgnored, "ignored"},
		{"reported", FindingReported, "reported"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := tt.fs.Value()
			if err != nil {
				t.Errorf("Value() error: %v", err)
			}
			if v != tt.want {
				t.Errorf("Value() = %v, want %v", v, tt.want)
			}
		})
	}
}

func TestFindingStatusScan(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    FindingStatus
		wantErr bool
	}{
		{"string", "confirmed", FindingConfirmed, false},
		{"bytes", []byte("new"), FindingNew, false},
		{"nil", nil, "", false},
		{"float", 3.14, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var fs FindingStatus
			err := fs.Scan(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Scan(%v) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if fs != tt.want {
				t.Errorf("Scan(%v) → %v, want %v", tt.input, fs, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// finding.go — EvidenceType Value/Scan
// ---------------------------------------------------------------------------

func TestEvidenceTypeValue(t *testing.T) {
	tests := []struct {
		name string
		et   EvidenceType
		want string
	}{
		{"request", EvidenceRequest, "request"},
		{"response", EvidenceResponse, "response"},
		{"screenshot", EvidenceScreenshot, "screenshot"},
		{"raw_output", EvidenceRawOutput, "raw_output"},
		{"note", EvidenceNote, "note"},
		{"file", EvidenceFile, "file"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := tt.et.Value()
			if err != nil {
				t.Errorf("Value() error: %v", err)
			}
			if v != tt.want {
				t.Errorf("Value() = %v, want %v", v, tt.want)
			}
		})
	}
}

func TestEvidenceTypeScan(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    EvidenceType
		wantErr bool
	}{
		{"string", "screenshot", EvidenceScreenshot, false},
		{"bytes", []byte("note"), EvidenceNote, false},
		{"nil", nil, "", false},
		{"bool", true, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var et EvidenceType
			err := et.Scan(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Scan(%v) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if et != tt.want {
				t.Errorf("Scan(%v) → %v, want %v", tt.input, et, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// scope.go — ScopeAction Value/Scan
// ---------------------------------------------------------------------------

func TestScopeActionValue(t *testing.T) {
	tests := []struct {
		name string
		sa   ScopeAction
		want string
	}{
		{"include", ScopeActionInclude, "include"},
		{"exclude", ScopeActionExclude, "exclude"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := tt.sa.Value()
			if err != nil {
				t.Errorf("Value() error: %v", err)
			}
			if v != tt.want {
				t.Errorf("Value() = %v, want %v", v, tt.want)
			}
		})
	}
}

func TestScopeActionScan(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    ScopeAction
		wantErr bool
	}{
		{"string", "include", ScopeActionInclude, false},
		{"bytes", []byte("exclude"), ScopeActionExclude, false},
		{"nil", nil, "", false},
		{"int", 99, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sa ScopeAction
			err := sa.Scan(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Scan(%v) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if sa != tt.want {
				t.Errorf("Scan(%v) → %v, want %v", tt.input, sa, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// scope.go — ScopeDecisionResult Value/Scan
// ---------------------------------------------------------------------------

func TestScopeDecisionResultValue(t *testing.T) {
	tests := []struct {
		name string
		sd   ScopeDecisionResult
		want string
	}{
		{"allow", ScopeAllow, "allow"},
		{"deny", ScopeDeny, "deny"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := tt.sd.Value()
			if err != nil {
				t.Errorf("Value() error: %v", err)
			}
			if v != tt.want {
				t.Errorf("Value() = %v, want %v", v, tt.want)
			}
		})
	}
}

func TestScopeDecisionResultScan(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    ScopeDecisionResult
		wantErr bool
	}{
		{"string", "allow", ScopeAllow, false},
		{"bytes", []byte("deny"), ScopeDeny, false},
		{"nil", nil, "", false},
		{"float", 1.5, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sd ScopeDecisionResult
			err := sd.Scan(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Scan(%v) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if sd != tt.want {
				t.Errorf("Scan(%v) → %v, want %v", tt.input, sd, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// target.go — TargetType Value/Scan
// ---------------------------------------------------------------------------

func TestTargetTypeValue(t *testing.T) {
	tests := []struct {
		name string
		tt   TargetType
		want string
	}{
		{"domain", TargetTypeDomain, "domain"},
		{"url", TargetTypeURL, "url"},
		{"ip", TargetTypeIP, "ip"},
		{"cidr", TargetTypeCIDR, "cidr"},
		{"company", TargetTypeCompany, "company"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := tt.tt.Value()
			if err != nil {
				t.Errorf("Value() error: %v", err)
			}
			if v != tt.want {
				t.Errorf("Value() = %v, want %v", v, tt.want)
			}
		})
	}
}

func TestTargetTypeScan(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    TargetType
		wantErr bool
	}{
		{"string", "domain", TargetTypeDomain, false},
		{"bytes", []byte("cidr"), TargetTypeCIDR, false},
		{"nil", nil, "", false},
		{"bool", false, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var tp TargetType
			err := tp.Scan(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Scan(%v) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if tp != tt.want {
				t.Errorf("Scan(%v) → %v, want %v", tt.input, tp, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// scan.go — TaskStatus Value/Scan
// ---------------------------------------------------------------------------

func TestTaskStatusValue(t *testing.T) {
	tests := []struct {
		name string
		ts   TaskStatus
		want string
	}{
		{"created", TaskCreated, "created"},
		{"queued", TaskQueued, "queued"},
		{"running", TaskRunning, "running"},
		{"completed", TaskCompleted, "completed"},
		{"failed", TaskFailed, "failed"},
		{"cancelled", TaskCancelled, "cancelled"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := tt.ts.Value()
			if err != nil {
				t.Errorf("Value() error: %v", err)
			}
			if v != tt.want {
				t.Errorf("Value() = %v, want %v", v, tt.want)
			}
		})
	}
}

func TestTaskStatusScan(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    TaskStatus
		wantErr bool
	}{
		{"string", "running", TaskRunning, false},
		{"bytes", []byte("completed"), TaskCompleted, false},
		{"nil", nil, "", false},
		{"int", 7, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ts TaskStatus
			err := ts.Scan(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Scan(%v) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if ts != tt.want {
				t.Errorf("Scan(%v) → %v, want %v", tt.input, ts, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// scan.go — ArtifactType Value/Scan
// ---------------------------------------------------------------------------

func TestArtifactTypeValue(t *testing.T) {
	tests := []struct {
		name string
		at   ArtifactType
		want string
	}{
		{"stdout", ArtifactStdout, "stdout"},
		{"stderr", ArtifactStderr, "stderr"},
		{"jsonl", ArtifactJSONL, "jsonl"},
		{"screenshot", ArtifactScreenshot, "screenshot"},
		{"request", ArtifactRequest, "request"},
		{"response", ArtifactResponse, "response"},
		{"file", ArtifactFile, "file"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := tt.at.Value()
			if err != nil {
				t.Errorf("Value() error: %v", err)
			}
			if v != tt.want {
				t.Errorf("Value() = %v, want %v", v, tt.want)
			}
		})
	}
}

func TestArtifactTypeScan(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    ArtifactType
		wantErr bool
	}{
		{"string", "screenshot", ArtifactScreenshot, false},
		{"bytes", []byte("file"), ArtifactFile, false},
		{"nil", nil, "", false},
		{"float64", 2.7, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var at ArtifactType
			err := at.Scan(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Scan(%v) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if at != tt.want {
				t.Errorf("Scan(%v) → %v, want %v", tt.input, at, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// scan.go — StepStatus Value/Scan
// ---------------------------------------------------------------------------

func TestStepStatusValue(t *testing.T) {
	tests := []struct {
		name string
		ss   StepStatus
		want string
	}{
		{"pending", StepPending, "pending"},
		{"running", StepRunning, "running"},
		{"completed", StepCompleted, "completed"},
		{"failed", StepFailed, "failed"},
		{"skipped", StepSkipped, "skipped"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := tt.ss.Value()
			if err != nil {
				t.Errorf("Value() error: %v", err)
			}
			if v != tt.want {
				t.Errorf("Value() = %v, want %v", v, tt.want)
			}
		})
	}
}

func TestStepStatusScan(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    StepStatus
		wantErr bool
	}{
		{"string", "failed", StepFailed, false},
		{"bytes", []byte("skipped"), StepSkipped, false},
		{"nil", nil, "", false},
		{"bool", true, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ss StepStatus
			err := ss.Scan(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Scan(%v) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if ss != tt.want {
				t.Errorf("Scan(%v) → %v, want %v", tt.input, ss, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// scan.go — StepName Value/Scan
// ---------------------------------------------------------------------------

func TestStepNameValue(t *testing.T) {
	tests := []struct {
		name string
		sn   StepName
		want string
	}{
		{"scope_check", StepScopeCheck, "scope_check"},
		{"run_tool", StepRunTool, "run_tool"},
		{"parse_output", StepParseOutput, "parse_output"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := tt.sn.Value()
			if err != nil {
				t.Errorf("Value() error: %v", err)
			}
			if v != tt.want {
				t.Errorf("Value() = %v, want %v", v, tt.want)
			}
		})
	}
}

func TestStepNameScan(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    StepName
		wantErr bool
	}{
		{"string", "run_tool", StepRunTool, false},
		{"bytes", []byte("cleanup"), StepCleanup, false},
		{"nil", nil, "", false},
		{"int", 42, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sn StepName
			err := sn.Scan(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Scan(%v) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if sn != tt.want {
				t.Errorf("Scan(%v) → %v, want %v", tt.input, sn, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// scan.go — RunStatus Value/Scan
// ---------------------------------------------------------------------------

func TestRunStatusValue(t *testing.T) {
	tests := []struct {
		name string
		rs   RunStatus
		want string
	}{
		{"pending", RunPending, "pending"},
		{"running", RunRunning, "running"},
		{"completed", RunCompleted, "completed"},
		{"failed", RunFailed, "failed"},
		{"cancelled", RunCancelled, "cancelled"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := tt.rs.Value()
			if err != nil {
				t.Errorf("Value() error: %v", err)
			}
			if v != tt.want {
				t.Errorf("Value() = %v, want %v", v, tt.want)
			}
		})
	}
}

func TestRunStatusScan(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    RunStatus
		wantErr bool
	}{
		{"string", "completed", RunCompleted, false},
		{"bytes", []byte("cancelled"), RunCancelled, false},
		{"nil", nil, "", false},
		{"struct", struct{}{}, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var rs RunStatus
			err := rs.Scan(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Scan(%v) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if rs != tt.want {
				t.Errorf("Scan(%v) → %v, want %v", tt.input, rs, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// scan.go — PipelineRunStageStatus Value/Scan
// ---------------------------------------------------------------------------

func TestPipelineRunStageStatusValue(t *testing.T) {
	tests := []struct {
		name string
		ss   PipelineRunStageStatus
		want string
	}{
		{"pending", StageStatusPending, "pending"},
		{"running", StageStatusRunning, "running"},
		{"completed", StageStatusCompleted, "completed"},
		{"failed", StageStatusFailed, "failed"},
		{"skipped", StageStatusSkipped, "skipped"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := tt.ss.Value()
			if err != nil {
				t.Errorf("Value() error: %v", err)
			}
			if v != tt.want {
				t.Errorf("Value() = %v, want %v", v, tt.want)
			}
		})
	}
}

func TestPipelineRunStageStatusScan(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    PipelineRunStageStatus
		wantErr bool
	}{
		{"string", "running", StageStatusRunning, false},
		{"bytes", []byte("skipped"), StageStatusSkipped, false},
		{"nil", nil, "", false},
		{"int", 0, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ss PipelineRunStageStatus
			err := ss.Scan(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Scan(%v) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if ss != tt.want {
				t.Errorf("Scan(%v) → %v, want %v", tt.input, ss, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// worker.go — WorkerMode Value/Scan
// ---------------------------------------------------------------------------

func TestWorkerModeValue(t *testing.T) {
	v, err := WorkerModeRemote.Value()
	if err != nil {
		t.Errorf("Value() error: %v", err)
	}
	if v != "remote" {
		t.Errorf("Value() = %v, want remote", v)
	}
}

func TestWorkerModeScan(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    WorkerMode
		wantErr bool
	}{
		{"string", "remote", WorkerModeRemote, false},
		{"bytes", []byte("remote"), WorkerModeRemote, false},
		{"nil", nil, "", false},
		{"int", 1, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var wm WorkerMode
			err := wm.Scan(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Scan(%v) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if wm != tt.want {
				t.Errorf("Scan(%v) → %v, want %v", tt.input, wm, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// worker.go — WorkerStatus Value/Scan
// ---------------------------------------------------------------------------

func TestWorkerStatusValue(t *testing.T) {
	tests := []struct {
		name string
		ws   WorkerStatus
		want string
	}{
		{"online", WorkerStatusOnline, "online"},
		{"offline", WorkerStatusOffline, "offline"},
		{"busy", WorkerStatusBusy, "busy"},
		{"error", WorkerStatusError, "error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := tt.ws.Value()
			if err != nil {
				t.Errorf("Value() error: %v", err)
			}
			if v != tt.want {
				t.Errorf("Value() = %v, want %v", v, tt.want)
			}
		})
	}
}

func TestWorkerStatusScan(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    WorkerStatus
		wantErr bool
	}{
		{"string", "busy", WorkerStatusBusy, false},
		{"bytes", []byte("error"), WorkerStatusError, false},
		{"nil", nil, "", false},
		{"bool", true, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ws WorkerStatus
			err := ws.Scan(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Scan(%v) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if ws != tt.want {
				t.Errorf("Scan(%v) → %v, want %v", tt.input, ws, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// worker.go — HealthCheckStatus Value/Scan
// ---------------------------------------------------------------------------

func TestHealthCheckStatusValue(t *testing.T) {
	tests := []struct {
		name string
		hcs  HealthCheckStatus
		want string
	}{
		{"ready", HealthCheckReady, "ready"},
		{"missing", HealthCheckMissing, "missing"},
		{"version_mismatch", HealthCheckVersionMismatch, "version_mismatch"},
		{"config_error", HealthCheckConfigError, "config_error"},
		{"permission_error", HealthCheckPermissionError, "permission_error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := tt.hcs.Value()
			if err != nil {
				t.Errorf("Value() error: %v", err)
			}
			if v != tt.want {
				t.Errorf("Value() = %v, want %v", v, tt.want)
			}
		})
	}
}

func TestHealthCheckStatusScan(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    HealthCheckStatus
		wantErr bool
	}{
		{"string", "ready", HealthCheckReady, false},
		{"bytes", []byte("missing"), HealthCheckMissing, false},
		{"nil", nil, "", false},
		{"float", 9.9, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var hcs HealthCheckStatus
			err := hcs.Scan(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Scan(%v) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if hcs != tt.want {
				t.Errorf("Scan(%v) → %v, want %v", tt.input, hcs, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// scan_work.go — WorkStatus Value/Scan
// ---------------------------------------------------------------------------

func TestWorkStatusValue(t *testing.T) {
	tests := []struct {
		name string
		ws   WorkStatus
		want string
	}{
		{"pending", WorkStatusPending, "pending"},
		{"running", WorkStatusRunning, "running"},
		{"done", WorkStatusDone, "done"},
		{"skipped", WorkStatusSkipped, "skipped"},
		{"failed", WorkStatusFailed, "failed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := tt.ws.Value()
			if err != nil {
				t.Errorf("Value() error: %v", err)
			}
			if v != tt.want {
				t.Errorf("Value() = %v, want %v", v, tt.want)
			}
		})
	}
}

func TestWorkStatusScan(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    WorkStatus
		wantErr bool
	}{
		{"string", "done", WorkStatusDone, false},
		{"bytes", []byte("failed"), WorkStatusFailed, false},
		{"nil", nil, "", false},
		{"int", 0, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ws WorkStatus
			err := ws.Scan(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Scan(%v) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if ws != tt.want {
				t.Errorf("Scan(%v) → %v, want %v", tt.input, ws, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// tool_call_log.go — ToolCallStatus Value/Scan
// ---------------------------------------------------------------------------

func TestToolCallStatusValue(t *testing.T) {
	tests := []struct {
		name string
		ts   ToolCallStatus
		want string
	}{
		{"running", ToolCallRunning, "running"},
		{"completed", ToolCallCompleted, "completed"},
		{"failed", ToolCallFailed, "failed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := tt.ts.Value()
			if err != nil {
				t.Errorf("Value() error: %v", err)
			}
			if v != tt.want {
				t.Errorf("Value() = %v, want %v", v, tt.want)
			}
		})
	}
}

func TestToolCallStatusScan(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    ToolCallStatus
		wantErr bool
	}{
		{"string", "completed", ToolCallCompleted, false},
		{"bytes", []byte("failed"), ToolCallFailed, false},
		{"nil", nil, "", false},
		{"int", 5, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ts ToolCallStatus
			err := ts.Scan(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Scan(%v) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if ts != tt.want {
				t.Errorf("Scan(%v) → %v, want %v", tt.input, ts, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// DefaultFfufDictionaryID constant
// ---------------------------------------------------------------------------

func TestDefaultFfufDictionaryIDConstant(t *testing.T) {
	if DefaultFfufDictionaryID != "builtin:path/top100.txt" {
		t.Errorf("DefaultFfufDictionaryID = %q, want %q", DefaultFfufDictionaryID, "builtin:path/top100.txt")
	}
}

// ---------------------------------------------------------------------------
// Verify ToJSON produces valid JSON that round-trips
// ---------------------------------------------------------------------------

func TestToJSONRoundTrip(t *testing.T) {
	type sample struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	orig := sample{Name: "test", Count: 42}
	raw := ToJSON(orig)

	var decoded sample
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if decoded.Name != orig.Name || decoded.Count != orig.Count {
		t.Errorf("round-trip mismatch: got %+v, want %+v", decoded, orig)
	}
}
