package core

import "testing"

func TestClassifySeedTarget_IP(t *testing.T) {
	if got := ClassifySeedTarget("172.31.0.13"); got != AssetIP {
		t.Fatalf("ClassifySeedTarget(172.31.0.13) = %s, want IP", got)
	}
}

func TestReconcileDiscoveryAsset_IPNotSubdomain(t *testing.T) {
	a := &DiscoveryAsset{
		Type:            AssetSubdomain,
		Value:           "172.31.0.13",
		NormalizedValue: "172.31.0.13",
	}
	ReconcileDiscoveryAsset(a)
	if a.Type != AssetIP {
		t.Fatalf("type = %s, want IP", a.Type)
	}
	works := DeriveEligibleWorks(a, DefaultInternalProfile())
	for _, w := range works {
		if w.Action == ActionSubdomainEnum {
			t.Fatal("IP asset must not derive SUBDOMAIN_ENUM")
		}
	}
}
