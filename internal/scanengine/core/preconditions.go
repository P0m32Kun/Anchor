package core

// isHighValueHTTP gates expensive HTTP follow-up actions (katana/ffuf/spoor) on external profiles.
func isHighValueHTTP(a *DiscoveryAsset, _ Profile) bool {
	if a.Type != AssetHTTPService && a.Type != AssetHTTPPath {
		return false
	}
	if a.Attrs.Sensitivity == "high" {
		return true
	}
	if len(a.Attrs.Technologies) > 0 {
		return true
	}
	if a.Attrs.StatusCode != nil && *a.Attrs.StatusCode >= 200 && *a.Attrs.StatusCode < 400 {
		return true
	}
	return false
}

func isHTTPServiceOrPathHighValue(a *DiscoveryAsset, p Profile) bool {
	return isHTTPServiceOrPath(a, p) && isHighValueHTTP(a, p)
}

// isSpoorEligible returns true for assets that Spoor can scan:
// HTTP services, paths, and JS URLs.
func isSpoorEligible(a *DiscoveryAsset, _ Profile) bool {
	return a.Type == AssetHTTPService || a.Type == AssetHTTPPath || a.Type == AssetJSURL
}
