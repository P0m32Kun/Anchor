package db

import (
	"database/sql"
	"encoding/json"

	"github.com/P0m32Kun/Anchor/internal/scanengine/core"
)

// GetAssetState returns the stored attrs for an asset.
func (q *Queries) GetAssetState(assetID string) (core.AssetAttrs, error) {
	var stateJSON sql.NullString
	err := q.db.QueryRow(`SELECT state FROM asset_state WHERE asset_id = ?`, assetID).Scan(&stateJSON)
	if err == sql.ErrNoRows {
		return core.AssetAttrs{}, nil
	}
	if err != nil {
		return core.AssetAttrs{}, err
	}
	if !stateJSON.Valid {
		return core.AssetAttrs{}, nil
	}
	var attrs core.AssetAttrs
	if err := json.Unmarshal([]byte(stateJSON.String), &attrs); err != nil {
		return core.AssetAttrs{}, err
	}
	return attrs, nil
}

// UpdateAssetState upserts the attrs for an asset.
func (q *Queries) UpdateAssetState(assetID string, attrs core.AssetAttrs) error {
	data, err := json.Marshal(attrs)
	if err != nil {
		return err
	}
	_, err = q.db.Exec(`
		INSERT INTO asset_state (asset_id, state, updated_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(asset_id) DO UPDATE SET state = excluded.state, updated_at = excluded.updated_at`,
		assetID, string(data))
	return err
}
