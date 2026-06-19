package models

import "testing"

func TestDefaultExternalPipelineConfig(t *testing.T) {
	cfg := DefaultExternalPipelineConfig()
	if cfg.PortRange != "top100" {
		t.Errorf("PortRange = %q, want top100", cfg.PortRange)
	}
	if cfg.NucleiScanDepth != "workflow" {
		t.Errorf("NucleiScanDepth = %q, want workflow", cfg.NucleiScanDepth)
	}
	if cfg.NaabuRate != 150 {
		t.Errorf("NaabuRate = %d, want 150", cfg.NaabuRate)
	}
	if !cfg.EnablePassiveSearch {
		t.Error("EnablePassiveSearch should be true")
	}
	if cfg.SubfinderMode != "passive" {
		t.Errorf("SubfinderMode = %q, want passive", cfg.SubfinderMode)
	}
	if cfg.NucleiRequireFingerprint != true {
		t.Error("NucleiRequireFingerprint should be true")
	}
}
