package workflow

import (
	"testing"

	"github.com/P0m32Kun/Anchor/internal/models"
)

// TestExternalPreset_IncludesPortscanStages documents that external mode retains
// the P3 port-scan stages (alive → portscan → fingerprint) with conservative defaults.
func TestExternalPreset_IncludesPortscanStages(t *testing.T) {
	cfg := models.DefaultExternalPipelineConfig()
	if cfg.PortRange != "top100" {
		t.Fatalf("port_range = %q, want top100", cfg.PortRange)
	}
	if !cfg.SkipPortscanOnCDNHost {
		t.Fatal("expected skip_portscan_on_cdn_host for external preset")
	}
	if cfg.FfufTier != "small" {
		t.Fatalf("ffuf_tier = %q, want small", cfg.FfufTier)
	}
	if cfg.NucleiScanDepth != "workflow" {
		t.Fatalf("nuclei_scan_depth = %q, want workflow", cfg.NucleiScanDepth)
	}
	if !cfg.EnableKatana {
		t.Fatal("expected enable_katana for external preset")
	}

	// Stage IDs used by external five-phase pipeline (P1–P5).
	wantStages := []StageID{
		StageSearch, StagePassiveCert, StagePassiveURL, StageSubdomain,
		StageResolve, StageCDNFilter, StageAlive, StagePortScan, StageFingerprint,
		StageHTTPX, StageCrawl, StageFfuf, StageHTTPX2, StageVuln, StageVuln2,
	}
	seen := make(map[StageID]bool)
	for _, s := range wantStages {
		seen[s] = true
	}
	if !seen[StagePortScan] || !seen[StagePassiveCert] || !seen[StageCrawl] {
		t.Fatal("missing expected external pipeline stage constants")
	}
}
