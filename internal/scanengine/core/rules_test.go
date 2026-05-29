package core

import (
	"testing"
)

func boolPtr(b bool) *bool { return &b }

func TestDeriveEligibleWorks_NucleiRequiresFingerprint(t *testing.T) {
	profile := DefaultInternalProfile()

	a := &DiscoveryAsset{
		ID: "asset-1", Type: AssetHTTPService, DiscoveryDepth: 0,
		Attrs: AssetAttrs{Fingerprinted: false},
	}
	works := DeriveEligibleWorks(a, profile)
	for _, w := range works {
		if w.Action == ActionNucleiScan {
			t.Fatal("nuclei should not be eligible without fingerprint")
		}
	}

	a.Attrs.Fingerprinted = true
	works = DeriveEligibleWorks(a, profile)
	var hasNuclei bool
	for _, w := range works {
		if w.Action == ActionNucleiScan {
			hasNuclei = true
		}
	}
	if !hasNuclei {
		t.Fatal("expected nuclei after fingerprint")
	}
}

func TestDeriveEligibleWorks_KatanaMaxDepth1(t *testing.T) {
	profile := DefaultInternalProfile()

	// Depth 0: katana eligible
	a := &DiscoveryAsset{ID: "a1", Type: AssetHTTPService, DiscoveryDepth: 0}
	works := DeriveEligibleWorks(a, profile)
	var hasKatana bool
	for _, w := range works {
		if w.Action == ActionKatanaCrawl {
			hasKatana = true
		}
	}
	if !hasKatana {
		t.Fatal("katana should be eligible at depth 0")
	}

	// Depth 1: katana still eligible (max_depth=1 means depth<=1)
	a.DiscoveryDepth = 1
	works = DeriveEligibleWorks(a, profile)
	hasKatana = false
	for _, w := range works {
		if w.Action == ActionKatanaCrawl {
			hasKatana = true
		}
	}
	if !hasKatana {
		t.Fatal("katana should be eligible at depth 1")
	}

	// Depth 2: katana NOT eligible
	a.DiscoveryDepth = 2
	works = DeriveEligibleWorks(a, profile)
	for _, w := range works {
		if w.Action == ActionKatanaCrawl {
			t.Fatal("katana should not be eligible at depth 2")
		}
	}
}

func TestDeriveEligibleWorks_PortScanSkippedOnCDN(t *testing.T) {
	profile := DefaultInternalProfile()

	a := &DiscoveryAsset{
		ID: "ip-1", Type: AssetIP, DiscoveryDepth: 0,
		Attrs: AssetAttrs{Alive: boolPtr(true), IsCDN: boolPtr(true)},
	}
	works := DeriveEligibleWorks(a, profile)
	for _, w := range works {
		if w.Action == ActionPortScan {
			t.Fatal("port scan should be skipped on CDN host")
		}
	}

	// Non-CDN: port scan eligible
	a.Attrs.IsCDN = boolPtr(false)
	works = DeriveEligibleWorks(a, profile)
	var hasPortScan bool
	for _, w := range works {
		if w.Action == ActionPortScan {
			hasPortScan = true
		}
	}
	if !hasPortScan {
		t.Fatal("port scan should be eligible on non-CDN host")
	}
}

func TestDeriveEligibleWorks_PortScanSkippedOnDeadHost(t *testing.T) {
	profile := DefaultInternalProfile()

	a := &DiscoveryAsset{
		ID: "ip-1", Type: AssetIP, DiscoveryDepth: 0,
		Attrs: AssetAttrs{Alive: boolPtr(false)},
	}
	works := DeriveEligibleWorks(a, profile)
	for _, w := range works {
		if w.Action == ActionPortScan {
			t.Fatal("port scan should be skipped on dead host")
		}
	}
}

func TestDeriveEligibleWorks_SubdomainOnlyAtDepth01(t *testing.T) {
	profile := DefaultInternalProfile()

	// Depth 0: subdomain enum eligible
	a := &DiscoveryAsset{ID: "sub-1", Type: AssetSubdomain, DiscoveryDepth: 0}
	works := DeriveEligibleWorks(a, profile)
	var hasSubdomain bool
	for _, w := range works {
		if w.Action == ActionSubdomainEnum {
			hasSubdomain = true
		}
	}
	if !hasSubdomain {
		t.Fatal("subdomain enum should be eligible at depth 0")
	}

	// Depth 1: still eligible
	a.DiscoveryDepth = 1
	works = DeriveEligibleWorks(a, profile)
	hasSubdomain = false
	for _, w := range works {
		if w.Action == ActionSubdomainEnum {
			hasSubdomain = true
		}
	}
	if !hasSubdomain {
		t.Fatal("subdomain enum should be eligible at depth 1")
	}

	// Depth 2: not eligible
	a.DiscoveryDepth = 2
	works = DeriveEligibleWorks(a, profile)
	for _, w := range works {
		if w.Action == ActionSubdomainEnum {
			t.Fatal("subdomain enum should not be eligible at depth 2")
		}
	}
}

func TestDeriveEligibleWorks_HTTPXOnHTTPService(t *testing.T) {
	profile := DefaultInternalProfile()

	a := &DiscoveryAsset{ID: "http-1", Type: AssetHTTPService, DiscoveryDepth: 0}
	works := DeriveEligibleWorks(a, profile)
	var hasHTTPX bool
	for _, w := range works {
		if w.Action == ActionHTTPXFingerprint {
			hasHTTPX = true
		}
	}
	if !hasHTTPX {
		t.Fatal("httpx should be eligible on HTTP_SERVICE")
	}

	// IP only: httpx not eligible
	ip := &DiscoveryAsset{ID: "ip-1", Type: AssetIP, DiscoveryDepth: 0}
	works = DeriveEligibleWorks(ip, profile)
	for _, w := range works {
		if w.Action == ActionHTTPXFingerprint {
			t.Fatal("httpx should not be eligible on IP asset")
		}
	}
}
