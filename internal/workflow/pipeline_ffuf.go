package workflow

import (
	"sort"
	"strings"

	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/nuclei"
)

// ffufTierActive reports whether post-phase ffuf should run based on tier and legacy dict ID.
func (p *Pipeline) ffufTierActive() bool {
	if !p.config.EnableFfuf {
		return false
	}
	tier := normalizeFfufTier(p.config.FfufTier)
	if tier == "off" {
		return false
	}
	if tier == "" {
		return p.config.FfufDictionaryID != ""
	}
	return true
}

func normalizeFfufTier(tier string) string {
	return strings.TrimSpace(strings.ToLower(tier))
}

// resolveFfufDictionaryID picks the dictionary for ffuf. Explicit FfufDictionaryID wins;
// otherwise tier maps to the smallest (small) or largest (medium) enabled dirscan dictionary.
func (p *Pipeline) resolveFfufDictionaryID() string {
	if p.config.FfufDictionaryID != "" {
		return p.config.FfufDictionaryID
	}
	tier := normalizeFfufTier(p.config.FfufTier)
	if tier == "" || tier == "off" {
		return ""
	}
	dicts, err := p.queries.ListEnabledDictionaries("dirscan")
	if err != nil || len(dicts) == 0 {
		return ""
	}
	sort.Slice(dicts, func(i, j int) bool {
		if dicts[i].LineCount == dicts[j].LineCount {
			return dicts[i].ID < dicts[j].ID
		}
		return dicts[i].LineCount < dicts[j].LineCount
	})
	switch tier {
	case "medium":
		return dicts[len(dicts)-1].ID
	default:
		return dicts[0].ID
	}
}

// shouldFfufEndpoint applies per-endpoint tier rules after httpx.
func (p *Pipeline) shouldFfufEndpoint(ep *models.WebEndpoint) bool {
	tier := normalizeFfufTier(p.config.FfufTier)
	if tier == "" {
		return true
	}
	switch tier {
	case "off":
		return false
	case "medium":
		return len(nuclei.MapPreciseTags(ep.Technologies, "")) > 0
	case "small":
		if ep.StatusCode == nil {
			return true
		}
		sc := *ep.StatusCode
		return sc == 200 || sc == 403
	default:
		return true
	}
}
