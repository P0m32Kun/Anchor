package seed

import (
	"log"

	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/scope"
)

// FilterSeedsByBoundary removes seeds that are out-of-scope according to the
// project's scope boundary mode. When mode is "off", all seeds pass through
// (same as current behavior). When mode is "strict", seeds must match include
// rules (if any) and must not match exclude rules.
//
// This is Gate A of the three-gate boundary enforcement:
//   - Gate A (here): filter seeds before they enter the engine
//   - Gate B: filter work items in the engine
//   - Gate C: mark findings scope status on persist
func FilterSeedsByBoundary(seeds []SeedAsset, eng *scope.Engine, rules []*models.ScopeRule, mode string) []SeedAsset {
	if mode == "" || mode == string(models.ScopeBoundaryOff) {
		return seeds
	}

	filtered := make([]SeedAsset, 0, len(seeds))
	for _, s := range seeds {
		target := seedToTarget(s)
		allow, _, reason := eng.EvaluateBoundary(target, rules, mode)
		if allow {
			filtered = append(filtered, s)
		} else {
			log.Printf("[scope] seed %q filtered out (mode=%s): %s", s.Value, mode, reason)
		}
	}
	return filtered
}

// seedToTarget converts a SeedAsset to a Target for scope evaluation.
func seedToTarget(s SeedAsset) *models.Target {
	typeMap := map[string]models.TargetType{
		"domain": models.TargetTypeDomain,
		"ip":     models.TargetTypeIP,
		"cidr":   models.TargetTypeCIDR,
		"url":    models.TargetTypeURL,
	}
	t, ok := typeMap[s.ValueType]
	if !ok {
		t = models.TargetTypeDomain
	}
	return &models.Target{
		Type:  t,
		Value: s.Value,
	}
}
