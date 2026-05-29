package core

// MaxDiscoveryDepth is the global default maximum discovery depth.
const MaxDiscoveryDepth = 2

// ActionRule defines when an action is eligible for a given asset.
type ActionRule struct {
	Action       TaskAction
	Enabled      bool
	MaxDepth     int          // -1 means no depth limit
	Precondition func(a *DiscoveryAsset, profile Profile) bool
}

// Profile is the interface for scan profile rules (internal, external, url_only).
type Profile interface {
	// Rules returns the action rules for this profile.
	Rules() []ActionRule
	// RequireFingerprint returns whether Nuclei requires a fingerprint first.
	RequireFingerprint() bool
}

// DefaultInternalProfile returns the default internal scan profile.
func DefaultInternalProfile() Profile {
	return &internalProfile{}
}

type internalProfile struct{}

func (p *internalProfile) RequireFingerprint() bool { return true }

func (p *internalProfile) Rules() []ActionRule {
	return []ActionRule{
		{Action: ActionSubdomainEnum, Enabled: true, MaxDepth: 1, Precondition: isSubdomain},
		{Action: ActionDNSResolve, Enabled: true, MaxDepth: -1, Precondition: isSubdomainOrIP},
		{Action: ActionCDNCheck, Enabled: true, MaxDepth: -1, Precondition: isSubdomainOrIP},
		{Action: ActionPortScan, Enabled: true, MaxDepth: MaxDiscoveryDepth, Precondition: isIPAndAlive},
		{Action: ActionServiceFingerprint, Enabled: true, MaxDepth: MaxDiscoveryDepth, Precondition: isIPPort},
		{Action: ActionHTTPXFingerprint, Enabled: true, MaxDepth: MaxDiscoveryDepth, Precondition: isWebEntry},
		{Action: ActionKatanaCrawl, Enabled: true, MaxDepth: 1, Precondition: isHTTPServiceOrPath},
		{Action: ActionFFUFBrute, Enabled: true, MaxDepth: 1, Precondition: isHTTPService},
		{Action: ActionNucleiScan, Enabled: true, MaxDepth: MaxDiscoveryDepth, Precondition: isHTTPAndFingerprinted},
	}
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
	return a.Type == AssetSubdomain || a.Type == AssetIP
}

func isIPAndAlive(a *DiscoveryAsset, _ Profile) bool {
	if a.Type != AssetIP {
		return false
	}
	// CDN check: skip port scan on CDN hosts
	if a.Attrs.IsCDN != nil && *a.Attrs.IsCDN {
		return false
	}
	// Must be alive (or alive status unknown — allow it)
	if a.Attrs.Alive != nil && !*a.Attrs.Alive {
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
