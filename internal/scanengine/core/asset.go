package core

import (
	"net"
	"strings"
)

// AssetType represents the kind of discovered asset.
type AssetType string

const (
	AssetSubdomain   AssetType = "SUBDOMAIN"
	AssetIP          AssetType = "IP"
	AssetCIDR        AssetType = "CIDR"
	AssetIPPort      AssetType = "IP_PORT"
	AssetHTTPService AssetType = "HTTP_SERVICE"
	AssetHTTPPath    AssetType = "HTTP_PATH"
	AssetJSURL       AssetType = "JS_URL"
)

// DiscoveryAsset is the engine-internal DTO for a node in the asset graph.
type DiscoveryAsset struct {
	ID              string     `json:"id"`
	Type            AssetType  `json:"type"`
	Value           string     `json:"value"`
	NormalizedValue string     `json:"normalized_value"`
	ParentID        string     `json:"parent_id,omitempty"`
	DiscoveryDepth  int        `json:"discovery_depth"`
	Attrs           AssetAttrs `json:"attrs"`
	SourceTool      string     `json:"source_tool,omitempty"`
}

// ReconcileDiscoveryAsset corrects asset type/normalized value before work derivation.
func ReconcileDiscoveryAsset(a *DiscoveryAsset) {
	if a == nil {
		return
	}
	switch InferStorageType(a.Value) {
	case "ip":
		a.Type = AssetIP
	case "cidr":
		a.Type = AssetCIDR
	case "url":
		a.Type = AssetHTTPService
	}
	a.NormalizedValue = normalizedDiscoveryValue(a)
}

func normalizedDiscoveryValue(a *DiscoveryAsset) string {
	switch a.Type {
	case AssetSubdomain:
		return strings.ToLower(strings.TrimSpace(a.Value))
	case AssetIP, AssetCIDR:
		if ip := net.ParseIP(strings.TrimSpace(a.Value)); ip != nil {
			if ip4 := ip.To4(); ip4 != nil {
				return ip4.String()
			}
			return ip.String()
		}
		if _, ipNet, err := net.ParseCIDR(strings.TrimSpace(a.Value)); err == nil {
			return ipNet.String()
		}
		return strings.TrimSpace(a.Value)
	default:
		return strings.TrimSpace(a.Value)
	}
}

// InferStorageType mirrors asset.InferStorageType for scanengine without importing asset.
func InferStorageType(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return "url"
	}
	if _, _, err := net.ParseCIDR(value); err == nil {
		return "cidr"
	}
	if host, _, err := net.SplitHostPort(value); err == nil {
		if net.ParseIP(host) != nil {
			return "ip"
		}
		return "domain"
	}
	if net.ParseIP(value) != nil {
		return "ip"
	}
	return "domain"
}

// ClassifySeedTarget infers asset type from a scan seed value.
func ClassifySeedTarget(target string) AssetType {
	target = strings.TrimSpace(target)
	if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
		return AssetHTTPService
	}
	if host, _, err := net.SplitHostPort(target); err == nil {
		if net.ParseIP(host) != nil {
			return AssetIP
		}
		return AssetSubdomain
	}
	// CIDR notation (e.g., 172.30.0.0/24)
	if _, _, err := net.ParseCIDR(target); err == nil {
		return AssetCIDR
	}
	if net.ParseIP(target) != nil {
		return AssetIP
	}
	return AssetSubdomain
}
