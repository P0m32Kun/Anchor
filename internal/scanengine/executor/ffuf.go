package executor

import (
	"encoding/json"
	"strings"

	"github.com/P0m32Kun/Anchor/internal/scanengine/core"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// ParseFFUFOutput parses ffuf JSON output into discovered URL assets.
func ParseFFUFOutput(stdout []byte) ([]*core.DiscoveryAsset, error) {
	var assets []*core.DiscoveryAsset

	// ffuf outputs a JSON object with "results" array
	var output ffufOutput
	if err := json.Unmarshal(stdout, &output); err != nil {
		// Try line-by-line JSONL fallback
		return parseFFUFJsonl(stdout)
	}

	seen := make(map[string]bool)
	for _, r := range output.Results {
		if r.URL == "" {
			continue
		}
		if seen[r.URL] {
			continue
		}
		seen[r.URL] = true
		assets = append(assets, &core.DiscoveryAsset{
			ID:              util.GenerateID(),
			Type:            core.AssetHTTPPath,
			Value:           r.URL,
			NormalizedValue: r.URL,
			SourceTool:      "ffuf",
		})
	}
	return assets, nil
}

func parseFFUFJsonl(stdout []byte) ([]*core.DiscoveryAsset, error) {
	var assets []*core.DiscoveryAsset
	seen := make(map[string]bool)

	lines := strings.Split(strings.TrimSpace(string(stdout)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var entry ffufResult
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		if entry.URL == "" {
			continue
		}
		if seen[entry.URL] {
			continue
		}
		seen[entry.URL] = true
		assets = append(assets, &core.DiscoveryAsset{
			ID:              util.GenerateID(),
			Type:            core.AssetHTTPPath,
			Value:           entry.URL,
			NormalizedValue: entry.URL,
			SourceTool:      "ffuf",
		})
	}
	return assets, nil
}

type ffufOutput struct {
	Results []ffufResult `json:"results"`
}

type ffufResult struct {
	URL        string `json:"url"`
	Status     int    `json:"status"`
	ContentLen int    `json:"content-length"`
}
