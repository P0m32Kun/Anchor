package seed

import (
	"testing"

	"github.com/P0m32Kun/Anchor/internal/scanengine/core"
)

// ============================================================
// ToDiscoveryAsset
// ============================================================

func TestToDiscoveryAsset_IP(t *testing.T) {
	s := SeedAsset{Value: "192.168.1.1", Source: "target", SourceRef: "tgt-1"}
	a := s.ToDiscoveryAsset()
	if a.Type != core.AssetIP {
		t.Errorf("type = %s, want IP", a.Type)
	}
	if a.SourceTool != "target" {
		t.Errorf("source = %s, want target", a.SourceTool)
	}
	if a.LineageSourceType != "target" {
		t.Errorf("lineage source type = %s, want target", a.LineageSourceType)
	}
	if a.LineageSourceID != "tgt-1" {
		t.Errorf("lineage source id = %s, want tgt-1", a.LineageSourceID)
	}
	if a.DiscoveryDepth != 0 {
		t.Errorf("depth = %d, want 0", a.DiscoveryDepth)
	}
	if a.ID == "" {
		t.Error("ID should be generated")
	}
}

func TestToDiscoveryAsset_Domain(t *testing.T) {
	s := SeedAsset{Value: "example.com", Source: "target"}
	a := s.ToDiscoveryAsset()
	if a.Type != core.AssetSubdomain {
		t.Errorf("type = %s, want SUBDOMAIN", a.Type)
	}
	if a.NormalizedValue != "example.com" {
		t.Errorf("normalized = %q, want example.com", a.NormalizedValue)
	}
}

func TestToDiscoveryAsset_URL(t *testing.T) {
	s := SeedAsset{Value: "http://example.com", Source: "fofa"}
	a := s.ToDiscoveryAsset()
	if a.Type != core.AssetHTTPService {
		t.Errorf("type = %s, want HTTP_SERVICE", a.Type)
	}
}

func TestToDiscoveryAsset_CIDR(t *testing.T) {
	s := SeedAsset{Value: "10.0.0.0/24", Source: "target"}
	a := s.ToDiscoveryAsset()
	if a.Type != core.AssetCIDR {
		t.Errorf("type = %s, want CIDR", a.Type)
	}
}

func TestToDiscoveryAsset_EmptySource(t *testing.T) {
	s := SeedAsset{Value: "example.com"}
	a := s.ToDiscoveryAsset()
	if a.SourceTool != "seed" {
		t.Errorf("source = %s, want seed (default)", a.SourceTool)
	}
}

func TestToDiscoveryAsset_NoLineageWithoutSourceRef(t *testing.T) {
	s := SeedAsset{Value: "example.com", Source: "target"}
	a := s.ToDiscoveryAsset()
	if a.LineageSourceType != "" {
		t.Errorf("lineage should be empty without SourceRef, got %s", a.LineageSourceType)
	}
}

// ============================================================
// DiscoveryAssetsFromValues
// ============================================================

func TestDiscoveryAssetsFromValues(t *testing.T) {
	vals := []string{"example.com", "10.0.0.1", "", "http://test.com"}
	seeds := DiscoveryAssetsFromValues(vals)
	if len(seeds) != 3 {
		t.Fatalf("expected 3 seeds, got %d", len(seeds))
	}
	for _, s := range seeds {
		if s.Source != "seed" {
			t.Errorf("source = %s, want seed", s.Source)
		}
	}
	if seeds[0].Value != "example.com" {
		t.Errorf("first seed = %q, want example.com", seeds[0].Value)
	}
}

func TestDiscoveryAssetsFromValues_Empty(t *testing.T) {
	seeds := DiscoveryAssetsFromValues(nil)
	if len(seeds) != 0 {
		t.Errorf("expected 0 seeds, got %d", len(seeds))
	}
}

// ============================================================
// SeedValues
// ============================================================

func TestSeedValues(t *testing.T) {
	seeds := []SeedAsset{
		{Value: "example.com"},
		{Value: ""},
		{Value: "10.0.0.1"},
	}
	vals := SeedValues(seeds)
	if len(vals) != 2 {
		t.Fatalf("expected 2 values, got %d", len(vals))
	}
	if vals[0] != "example.com" || vals[1] != "10.0.0.1" {
		t.Errorf("unexpected values: %v", vals)
	}
}

// ============================================================
// inferSeedValueType
// ============================================================

func TestInferSeedValueType(t *testing.T) {
	cases := []struct {
		value string
		want  string
	}{
		{"http://example.com", "url"},
		{"https://example.com/path", "url"},
		{"10.0.0.0/24", "cidr"},
		{"192.168.1.1", "ip"},
		{"example.com", "domain"},
		{"sub.example.com", "domain"},
		{"  10.0.0.1  ", "ip"},
	}
	for _, tc := range cases {
		t.Run(tc.value, func(t *testing.T) {
			got := inferSeedValueType(tc.value)
			if got != tc.want {
				t.Errorf("inferSeedValueType(%q) = %q, want %q", tc.value, got, tc.want)
			}
		})
	}
}
