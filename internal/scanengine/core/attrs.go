package core

// AssetAttrs holds gating attributes for a DiscoveryAsset.
// Fields are set as tools complete; nil means "not yet known".
type AssetAttrs struct {
	Alive         *bool    `json:"alive,omitempty"`
	IsCDN         *bool    `json:"is_cdn,omitempty"`
	Fingerprinted bool     `json:"fingerprinted"`
	Technologies  []string `json:"technologies,omitempty"`
	StatusCode    *int     `json:"status_code,omitempty"`
	// SRC bounty fields (from Spoor)
	Sensitivity  string   `json:"sensitivity,omitempty"`   // high/medium/low
	RequiresAuth bool     `json:"requires_auth,omitempty"` // whether endpoint likely requires auth
	Tags         []string `json:"tags,omitempty"`          // endpoint tags (admin, user, token, etc.)
}

// MergeAttrs updates dst with non-zero values from src.
// For pointer fields, src wins if non-nil.
func MergeAttrs(dst *AssetAttrs, src AssetAttrs) {
	if src.Alive != nil {
		dst.Alive = src.Alive
	}
	if src.IsCDN != nil {
		dst.IsCDN = src.IsCDN
	}
	if src.Fingerprinted {
		dst.Fingerprinted = true
	}
	if len(src.Technologies) > 0 {
		dst.Technologies = append(dst.Technologies, src.Technologies...)
	}
	if src.StatusCode != nil {
		dst.StatusCode = src.StatusCode
	}
	// SRC bounty fields
	if src.Sensitivity != "" {
		dst.Sensitivity = src.Sensitivity
	}
	if src.RequiresAuth {
		dst.RequiresAuth = true
	}
	if len(src.Tags) > 0 {
		dst.Tags = append(dst.Tags, src.Tags...)
	}
}
