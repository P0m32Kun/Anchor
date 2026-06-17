package core

// ExternalProfile implements the external scan profile with conservative settings.
type ExternalProfile struct {
	SkipPortscanOnCDN bool
}

// DefaultExternalProfile returns the default external scan profile.
func DefaultExternalProfile() *ExternalProfile {
	return &ExternalProfile{
		SkipPortscanOnCDN: true,
	}
}

func (p *ExternalProfile) RequireFingerprint() bool { return true }

func (p *ExternalProfile) Rules() []ActionRule {
	return []ActionRule{
		// Passive discovery is handled by seed injectors (crt/FOFA), not tool work items.
		{Action: ActionPassiveSearch, Enabled: false, MaxDepth: 0, Precondition: isSubdomain},
		{Action: ActionPassiveCert, Enabled: false, MaxDepth: 0, Precondition: isSubdomain},
		{Action: ActionPassiveURL, Enabled: false, MaxDepth: 0, Precondition: isSubdomain},

		// Active actions
		{Action: ActionSubdomainEnum, Enabled: true, MaxDepth: 1, Precondition: isSubdomain},
		{Action: ActionDNSResolve, Enabled: true, MaxDepth: 1, Precondition: isSubdomainOrIP},
		{Action: ActionCDNCheck, Enabled: true, MaxDepth: -1, Precondition: isIP},
		{Action: ActionPortScan, Enabled: true, MaxDepth: MaxDiscoveryDepth, Precondition: func(a *DiscoveryAsset, _ Profile) bool {
			if a.Type != AssetIP {
				return false
			}
			// CDN skip
			if p.SkipPortscanOnCDN && a.Attrs.IsCDN != nil && *a.Attrs.IsCDN {
				return false
			}
			if a.Attrs.Alive != nil && !*a.Attrs.Alive {
				return false
			}
			return true
		}},
		{Action: ActionServiceFingerprint, Enabled: true, MaxDepth: MaxDiscoveryDepth, Precondition: isIPPort},
		{Action: ActionHTTPXFingerprint, Enabled: true, MaxDepth: MaxDiscoveryDepth, Precondition: isWebEntryOrHTTPXCandidate},
		{Action: ActionKatanaCrawl, Enabled: true, MaxDepth: 1, Precondition: isHTTPServiceOrPath},
		{Action: ActionFFUFBrute, Enabled: false, MaxDepth: 1, Precondition: isHTTPService}, // disabled by default for external
		{Action: ActionNucleiScan, Enabled: true, MaxDepth: MaxDiscoveryDepth, Precondition: isHTTPAndFingerprinted},
	}
}
