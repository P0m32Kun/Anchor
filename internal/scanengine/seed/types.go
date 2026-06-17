package seed

import "github.com/P0m32Kun/Anchor/internal/search"

// SeedAsset is a normalized seed produced before ScanEngine ingestion.
type SeedAsset struct {
	Value     string               // domain / ip / url string passed to the engine
	ValueType string               // domain | ip | url
	Source    string               // fofa | hunter | quake | target | seed
	SourceRef string               // originating target id when known
	Raw       *search.SearchResult // optional passive metadata
}

// SeedValues extracts scan seed strings from SeedAssets preserving order.
func SeedValues(seeds []SeedAsset) []string {
	vals := make([]string, 0, len(seeds))
	for _, s := range seeds {
		if s.Value != "" {
			vals = append(vals, s.Value)
		}
	}
	return vals
}
