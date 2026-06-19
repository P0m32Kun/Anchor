package seed

import (
	"log"
	"strings"

	"github.com/P0m32Kun/Anchor/internal/models"
)

var passiveSearchSources = map[string]bool{
	"fofa":   true,
	"hunter": true,
	"quake":  true,
}

// JunkFilterStats summarizes passive junk filtering for a seed batch.
type JunkFilterStats struct {
	Raw     int
	Kept    int
	Dropped int
}

// FilterPassiveSeeds removes high-confidence junk seeds from passive search engines.
// Manual targets and non-passive sources are always kept. Default policy is permissive:
// only drop on keyword hits in title, domain, or URL host/path.
func FilterPassiveSeeds(seeds []SeedAsset, cfg models.PipelineConfig) ([]SeedAsset, JunkFilterStats) {
	stats := JunkFilterStats{Raw: len(seeds)}
	if !cfg.EnablePassiveJunkFilter || len(seeds) == 0 {
		stats.Kept = len(seeds)
		return seeds, stats
	}

	keywords := activeJunkKeywords(nil)
	if cfg.PassiveJunkKeywords != "" {
		for _, extra := range strings.Split(cfg.PassiveJunkKeywords, ",") {
			extra = strings.TrimSpace(extra)
			if extra == "" {
				continue
			}
			keywords = append(keywords, junkKeywordRule{Keyword: extra})
		}
	}

	out := make([]SeedAsset, 0, len(seeds))
	for _, s := range seeds {
		if !passiveSearchSources[s.Source] {
			out = append(out, s)
			continue
		}
		if field, kw := matchSeedJunk(s, keywords); kw != "" {
			stats.Dropped++
			log.Printf("[seed] junk-filter dropped value=%q source=%s field=%s keyword=%q", s.Value, s.Source, field, kw)
			continue
		}
		out = append(out, s)
	}
	stats.Kept = len(out)
	if stats.Dropped > 0 {
		log.Printf("[seed] junk-filter summary raw=%d kept=%d dropped=%d", stats.Raw, stats.Kept, stats.Dropped)
	}
	return out, stats
}

// matchSeedJunk checks if a seed matches any junk keyword rule.
// It checks s.Value, s.Raw.Title, and s.Raw.Domain (when s.Raw is non-nil).
func matchSeedJunk(s SeedAsset, rules []junkKeywordRule) (string, string) {
	for _, rule := range rules {
		kw := strings.ToLower(rule.Keyword)
		if strings.Contains(strings.ToLower(s.Value), kw) {
			return "value", rule.Keyword
		}
		if s.Raw != nil {
			if strings.Contains(strings.ToLower(s.Raw.Title), kw) {
				return "title", rule.Keyword
			}
			if strings.Contains(strings.ToLower(s.Raw.Domain), kw) {
				return "domain", rule.Keyword
			}
		}
	}
	return "", ""
}
