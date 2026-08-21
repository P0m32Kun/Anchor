package scanconfig

import (
	"testing"

	"github.com/P0m32Kun/Anchor/internal/models"
)

func TestIsInternalMode(t *testing.T) {
	cases := []struct {
		mode string
		want bool
	}{
		{"internal", true},
		{"INTERNAL", true},
		{" internal ", true},
		{"external", false},
		{"", false},
	}
	for _, c := range cases {
		if got := IsInternalMode(c.mode); got != c.want {
			t.Errorf("IsInternalMode(%q) = %v, want %v", c.mode, got, c.want)
		}
	}
}

func TestMigrateInternalConfigToExternal(t *testing.T) {
	legacy := models.PipelineConfig{
		EnableSubfinder:   false,
		EnableDNSx:        false,
		EnableCDNFilter:   false,
		EnableNmapService: true,
		EnableHttpx:       true,
		EnableNuclei:      true,
		NaabuRate:         1000,
		PortRange:         "high-risk",
	}
	got := MigrateInternalConfigToExternal(legacy)
	if !got.EnableSubfinder || !got.EnableCDNFilter || !got.EnableDNSx {
		t.Error("migration must enable subfinder, dnsx, and cdn filter")
	}
	if !got.EnablePassiveSearch || !got.SkipPortscanOnCDNHost || !got.NucleiRequireFingerprint {
		t.Error("migration must apply the conservative internet baseline guards")
	}
	if got.NaabuRate > 150 {
		t.Errorf("NaabuRate = %d, want <= 150 (internet rate)", got.NaabuRate)
	}
	if got.PortRange != "top100" {
		t.Errorf("PortRange = %q, want internet default top100", got.PortRange)
	}
}

func TestIsLegacyInternalConfig(t *testing.T) {
	legacy := models.PipelineConfig{EnableSubfinder: false, EnableCDNFilter: false, NaabuRate: 1000, PortRange: "high-risk"}
	if !IsLegacyInternalConfig(legacy) {
		t.Error("expected legacy internal config to be detected")
	}
	// An internet config that merely toggles one tool off must not be flagged.
	ext := models.DefaultExternalPipelineConfig()
	ext.EnableFfuf = false
	if IsLegacyInternalConfig(ext) {
		t.Error("external config with ffuf toggled off must not be flagged as legacy internal")
	}
	// A config with subfinder on (even high rate) is not internal-shaped.
	notInternal := models.PipelineConfig{EnableSubfinder: true, EnableCDNFilter: false, NaabuRate: 1000, PortRange: "high-risk"}
	if IsLegacyInternalConfig(notInternal) {
		t.Error("config with subfinder enabled must not be flagged as legacy internal")
	}
}
