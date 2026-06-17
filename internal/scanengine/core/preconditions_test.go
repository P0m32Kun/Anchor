package core

import (
	"testing"

	"github.com/P0m32Kun/Anchor/internal/models"
)

func TestIsHighValueHTTP(t *testing.T) {
	statusOK := 200
	status404 := 404

	tests := []struct {
		name string
		asset *DiscoveryAsset
		want bool
	}{
		{
			name: "technologies",
			asset: &DiscoveryAsset{
				Type: AssetHTTPService,
				Attrs: AssetAttrs{Technologies: []string{"nginx"}},
			},
			want: true,
		},
		{
			name: "status 2xx",
			asset: &DiscoveryAsset{
				Type: AssetHTTPService,
				Attrs: AssetAttrs{StatusCode: &statusOK},
			},
			want: true,
		},
		{
			name: "sensitivity high",
			asset: &DiscoveryAsset{
				Type: AssetHTTPPath,
				Attrs: AssetAttrs{Sensitivity: "high"},
			},
			want: true,
		},
		{
			name: "low signal",
			asset: &DiscoveryAsset{
				Type: AssetHTTPService,
				Attrs: AssetAttrs{StatusCode: &status404},
			},
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isHighValueHTTP(tc.asset, nil); got != tc.want {
				t.Errorf("isHighValueHTTP() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestKatana_ExternalRequiresHighValue(t *testing.T) {
	cfg := models.DefaultExternalPipelineConfig()
	cfg.EnableKatana = true
	profile := ProfileFromConfig("external", cfg)

	low := &DiscoveryAsset{ID: "http-low", Type: AssetHTTPService, DiscoveryDepth: 0}
	works := DeriveEligibleWorks(low, profile)
	for _, w := range works {
		if w.Action == ActionKatanaCrawl {
			t.Fatal("katana should not run on low-value HTTP without attrs")
		}
	}

	status := 200
	high := &DiscoveryAsset{
		ID: "http-high", Type: AssetHTTPService, DiscoveryDepth: 0,
		Attrs: AssetAttrs{StatusCode: &status, Technologies: []string{"nginx"}},
	}
	works = DeriveEligibleWorks(high, profile)
	var hasKatana bool
	for _, w := range works {
		if w.Action == ActionKatanaCrawl {
			hasKatana = true
		}
	}
	if !hasKatana {
		t.Fatal("katana should run on high-value HTTP when enabled")
	}
}
