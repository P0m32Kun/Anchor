package core

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
