package asset

import (
	"encoding/json"

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

// EncodeAssetState marshals attrs to a JSON string.
func EncodeAssetState(attrs core.AssetAttrs) (string, error) {
	data, err := json.Marshal(attrs)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// MergeAndSaveState loads existing attrs for assetID, merges non-zero fields
// from attrs (existing fields preserved unless overwritten), then saves back.
func MergeAndSaveState(q *db.Queries, assetID string, attrs core.AssetAttrs) error {
	existing, err := q.GetAssetState(assetID)
	if err != nil {
		return err
	}
	core.MergeAttrs(&existing, attrs)
	return q.UpdateAssetState(assetID, existing)
}
