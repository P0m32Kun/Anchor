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
	if net.ParseIP(target) != nil {
		return AssetIP
	}
	return AssetSubdomain
}
