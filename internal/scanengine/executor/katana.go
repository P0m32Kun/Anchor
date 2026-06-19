package executor

import (
	"encoding/json"
	"net/url"
	"strings"

	"github.com/P0m32Kun/Anchor/internal/scanengine/core"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// ParseKatanaOutput parses katana JSONL output into discovered URL assets.
func ParseKatanaOutput(stdout []byte) ([]*core.DiscoveryAsset, error) {
	var assets []*core.DiscoveryAsset
	seen := make(map[string]bool)

	lines := strings.Split(strings.TrimSpace(string(stdout)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var entry katanaEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		if entry.URL == "" {
			continue
		}
		// Dedup within same output
		if seen[entry.URL] {
			continue
		}
		seen[entry.URL] = true

		assetType := core.AssetHTTPPath
		if isJSURL(entry.URL) {
			assetType = core.AssetJSURL
		}
		assets = append(assets, &core.DiscoveryAsset{
			ID:              util.GenerateID(),
			Type:            assetType,
			Value:           entry.URL,
			NormalizedValue: entry.URL,
			SourceTool:      "katana",
		})
	}
	return assets, nil
}

type katanaEntry struct {
	URL string `json:"url"`
}

// isJSURL returns true if the URL path ends with .js (ignoring query params and fragments).
func isJSURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return strings.HasSuffix(strings.ToLower(u.Path), ".js")
}
