package core

import (
	"testing"
)

func TestPortScan_SkippedOnCDN_External(t *testing.T) {
	profile := DefaultExternalProfile()

	a := &DiscoveryAsset{
		ID: "ip-1", Type: AssetIP, DiscoveryDepth: 0,
		Attrs: AssetAttrs{Alive: boolPtr(true), IsCDN: boolPtr(true)},
	}
	works := DeriveEligibleWorks(a, profile)
	for _, w := range works {
		if w.Action == ActionPortScan {
			t.Fatal("should not enqueue port scan on CDN (external profile)")
		}
	}
}

func TestPortScan_AllowedOnNonCDN_External(t *testing.T) {
	profile := DefaultExternalProfile()

	a := &DiscoveryAsset{
		ID: "ip-1", Type: AssetIP, DiscoveryDepth: 0,
		Attrs: AssetAttrs{Alive: boolPtr(true), IsCDN: boolPtr(false)},
	}
	works := DeriveEligibleWorks(a, profile)
	var hasPortScan bool
	for _, w := range works {
		if w.Action == ActionPortScan {
			hasPortScan = true
		}
	}
	if !hasPortScan {
		t.Fatal("port scan should be allowed on non-CDN (external profile)")
	}
}

func TestFFUF_DisabledByDefault_External(t *testing.T) {
	profile := DefaultExternalProfile()

	a := &DiscoveryAsset{ID: "http-1", Type: AssetHTTPService, DiscoveryDepth: 0}
	works := DeriveEligibleWorks(a, profile)
	for _, w := range works {
		if w.Action == ActionFFUFBrute {
			t.Fatal("ffuf should be disabled by default in external profile")
		}
	}
}

func TestPassiveActions_External(t *testing.T) {
	profile := DefaultExternalProfile()

	a := &DiscoveryAsset{ID: "sub-1", Type: AssetSubdomain, DiscoveryDepth: 0}
	works := DeriveEligibleWorks(a, profile)

	actions := make(map[TaskAction]bool)
	for _, w := range works {
		actions[w.Action] = true
	}

	// Passive discovery runs via seed injectors, not tool work items.
	if actions[ActionPassiveSearch] {
		t.Error("passive search should not enqueue work items")
	}
	if actions[ActionPassiveCert] {
		t.Error("passive cert should not enqueue work items")
	}
	if actions[ActionPassiveURL] {
		t.Error("passive URL should not enqueue work items")
	}
}
