package scanconfig

import (
	"github.com/P0m32Kun/Anchor/internal/exclude"
	"github.com/P0m32Kun/Anchor/internal/scanengine/seed"
	"github.com/P0m32Kun/Anchor/internal/toolregistry"
)

func compiledHighRiskPorts() string {
	return toolregistry.HighRiskPortsFunc()
}

func compiledExcludeDomains() []string {
	return exclude.BuiltinDomainsFunc()
}

func compiledJunkKeywords() []JunkKeyword {
	return fallbackJunkKeywords()
}

// Apply pushes loaded config into runtime packages (call after Load).
func Apply(cfg *Config) {
	if cfg == nil {
		return
	}
	toolregistry.SetHighRiskPorts(cfg.HighRiskPorts)
	exclude.SetBuiltinDomains(cfg.ExcludeDomains)
	seed.SetJunkKeywords(toSeedJunkRules(cfg.JunkKeywords))
	seed.SetPassiveConfig(seed.PassiveRuntimeConfig{
		ResultLimit:   cfg.Passive.ResultLimit,
		Concurrency:   cfg.Passive.Concurrency,
		FofaQueries:   cfg.Passive.FofaQueries,
		HunterQueries: cfg.Passive.HunterQueries,
		QuakeQuery:    cfg.Passive.QuakeQuery,
	})
}

func toSeedJunkRules(in []JunkKeyword) []seed.JunkKeywordRule {
	out := make([]seed.JunkKeywordRule, len(in))
	for i, kw := range in {
		out[i] = seed.JunkKeywordRule{Keyword: kw.Keyword, WordBoundary: kw.WordBoundary}
	}
	return out
}
