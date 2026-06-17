package asset

import (
	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/scanengine/core"
)

// LoadAttrsForEngine hydrates gating attrs for an asset before work derivation.
func LoadAttrsForEngine(q *db.Queries, assetID string) (core.AssetAttrs, error) {
	if q == nil || assetID == "" {
		return core.AssetAttrs{}, nil
	}
	state, err := q.GetAssetState(assetID)
	if err != nil {
		return core.AssetAttrs{}, err
	}
	return state, nil
}
