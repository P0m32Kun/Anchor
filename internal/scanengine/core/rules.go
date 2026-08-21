package core

// MaxDiscoveryDepth is the global default maximum discovery depth.
const MaxDiscoveryDepth = 2

// ActionRule defines when an action is eligible for a given asset.
type ActionRule struct {
	Action       TaskAction
	Enabled      bool
	MaxDepth     int // -1 means no depth limit
	Precondition func(a *DiscoveryAsset, profile Profile) bool
}

// Profile is the interface for scan profile rules (external, url_only).
type Profile interface {
	// Rules returns the action rules for this profile.
	Rules() []ActionRule
	// RequireFingerprint returns whether Nuclei requires a fingerprint first.
	RequireFingerprint() bool
}

// DeriveEligibleWorks returns the list of works that should be enqueued for
// the given asset, based on the profile's rules and the asset's current state.
func DeriveEligibleWorks(a *DiscoveryAsset, profile Profile) []DerivedWork {
	var works []DerivedWork
	for _, rule := range profile.Rules() {
		if !rule.Enabled {
			continue
		}
		if rule.MaxDepth >= 0 && a.DiscoveryDepth > rule.MaxDepth {
			continue
		}
		if rule.Precondition != nil && !rule.Precondition(a, profile) {
			continue
		}
		stage := ActionToStage[rule.Action]
		works = append(works, DerivedWork{
			Action:  rule.Action,
			AssetID: a.ID,
			Stage:   stage,
		})
	}
	return works
}

// --- Precondition functions ---

func isSubdomain(a *DiscoveryAsset, _ Profile) bool {
	return a.Type == AssetSubdomain
}

func isSubdomainOrIP(a *DiscoveryAsset, _ Profile) bool {
	return a.Type == AssetSubdomain || a.Type == AssetIP || a.Type == AssetCIDR
}

func isIP(a *DiscoveryAsset, _ Profile) bool {
	return a.Type == AssetIP
}

func isIPOrCIDR(a *DiscoveryAsset, _ Profile) bool {
	return a.Type == AssetIP || a.Type == AssetCIDR
}

func isIPAndAlive(a *DiscoveryAsset, _ Profile) bool {
	if a.Type != AssetIP {
		return false
	}
	// CDN check: skip port scan on CDN hosts
	if a.Attrs.IsCDN != nil && *a.Attrs.IsCDN {
		return false
	}
	// Must be explicitly alive.
	if a.Attrs.Alive == nil || !*a.Attrs.Alive {
		return false
	}
	return true
}

func isIPPort(a *DiscoveryAsset, _ Profile) bool {
	return a.Type == AssetIPPort
}

func isWebEntry(a *DiscoveryAsset, _ Profile) bool {
	return a.Type == AssetHTTPService || a.Type == AssetHTTPPath
}

// isHTTPXCandidate covers assets that should be probed by httpx before becoming HTTP services.
func isHTTPXCandidate(a *DiscoveryAsset, _ Profile) bool {
	switch a.Type {
	case AssetSubdomain, AssetIPPort, AssetCIDR:
		return true
	case AssetIP:
		return a.Attrs.Alive != nil && *a.Attrs.Alive
	default:
		return false
	}
}

func isWebEntryOrHTTPXCandidate(a *DiscoveryAsset, p Profile) bool {
	return isWebEntry(a, p) || isHTTPXCandidate(a, p)
}

func isHTTPServiceOrPath(a *DiscoveryAsset, _ Profile) bool {
	return a.Type == AssetHTTPService || a.Type == AssetHTTPPath
}

func isHTTPService(a *DiscoveryAsset, _ Profile) bool {
	return a.Type == AssetHTTPService
}

func isHTTPAndFingerprinted(a *DiscoveryAsset, profile Profile) bool {
	if a.Type != AssetHTTPService && a.Type != AssetHTTPPath {
		return false
	}
	if profile.RequireFingerprint() && !a.Attrs.Fingerprinted {
		return false
	}
	return true
}
