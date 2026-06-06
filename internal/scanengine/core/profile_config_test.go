package core

import (
	"testing"

	"github.com/P0m32Kun/Anchor/internal/models"
)

func TestProfileFromConfig_DisablesNuclei(t *testing.T) {
	cfg := models.DefaultPipelineConfig()
	cfg.EnableNuclei = false
	p := ProfileFromConfig("internal", cfg)
	works := DeriveEligibleWorks(&DiscoveryAsset{
		ID: "a1", Type: AssetHTTPService, DiscoveryDepth: 0,
		Attrs: AssetAttrs{Fingerprinted: true},
	}, p)
	for _, w := range works {
		if w.Action == ActionNucleiScan {
			t.Fatal("nuclei work should be disabled by config")
		}
	}
}

func TestProfileFromConfig_PassiveNotEnqueued(t *testing.T) {
	cfg := models.DefaultExternalPipelineConfig()
	p := ProfileFromConfig("external", cfg)
	works := DeriveEligibleWorks(&DiscoveryAsset{
		ID: "s1", Type: AssetSubdomain, Value: "example.com", DiscoveryDepth: 0,
	}, p)
	for _, w := range works {
		switch w.Action {
		case ActionPassiveSearch, ActionPassiveCert, ActionPassiveURL:
			t.Fatalf("passive action %s must not be enqueued as work item", w.Action)
		}
	}
}

func TestProfileFromConfig_HTTPXCandidateSubdomain(t *testing.T) {
	cfg := models.DefaultPipelineConfig()
	p := ProfileFromConfig("internal", cfg)
	works := DeriveEligibleWorks(&DiscoveryAsset{
		ID: "s1", Type: AssetSubdomain, Value: "sub.example.com", DiscoveryDepth: 0,
	}, p)
	var hasHttpx bool
	for _, w := range works {
		if w.Action == ActionHTTPXFingerprint {
			hasHttpx = true
		}
	}
	if !hasHttpx {
		t.Fatal("subdomain should derive httpx fingerprint work")
	}
}
