package scheduler

import (
	"strings"

	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/scanengine/seed"
)

// SeedBucketKey returns a stable grouping key for fair scheduling.
func SeedBucketKey(s seed.SeedAsset) string {
	if s.SourceRef != "" {
		return "target:" + s.SourceRef
	}
	v := strings.ToLower(strings.TrimSpace(s.Value))
	if v == "" {
		return "seed:unknown"
	}
	return "seed:" + v
}

// TargetBucketKey returns the bucket for a project target record.
func TargetBucketKey(t *models.Target) string {
	if t == nil {
		return "target:unknown"
	}
	if t.ID != "" {
		return "target:" + t.ID
	}
	return "seed:" + strings.ToLower(strings.TrimSpace(t.Value))
}

// CountSeedBuckets returns distinct bucket keys among seeds.
func CountSeedBuckets(seeds []seed.SeedAsset) int {
	if len(seeds) == 0 {
		return 1
	}
	seen := make(map[string]struct{}, len(seeds))
	for _, s := range seeds {
		seen[SeedBucketKey(s)] = struct{}{}
	}
	return len(seen)
}
