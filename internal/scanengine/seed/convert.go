package seed

import (
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/scanengine/core"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// ToDiscoveryAsset converts a seed into an engine discovery asset with lineage metadata.
func (s SeedAsset) ToDiscoveryAsset() *core.DiscoveryAsset {
	a := &core.DiscoveryAsset{
		ID:             util.GenerateID(),
		Type:           core.ClassifySeedTarget(s.Value),
		Value:          s.Value,
		DiscoveryDepth: 0,
		SourceTool:     s.Source,
	}
	if s.SourceRef != "" {
		a.LineageSourceType = models.RelationSourceTarget
		a.LineageSourceID = s.SourceRef
		a.LineageRelationType = models.RelationExpandedBy
	}
	if a.SourceTool == "" {
		a.SourceTool = "seed"
	}
	core.ReconcileDiscoveryAsset(a)
	return a
}

// DiscoveryAssetsFromValues builds minimal seed assets from plain strings (tests / legacy Run).
func DiscoveryAssetsFromValues(values []string) []SeedAsset {
	out := make([]SeedAsset, 0, len(values))
	for _, v := range values {
		if v != "" {
			out = append(out, SeedAsset{Value: v, Source: "seed"})
		}
	}
	return out
}
