package db

import (
	"time"

	"github.com/P0m32Kun/Anchor/internal/util"
)

// AssetSnapshot records the state of a project's assets at the end of a scan.
type AssetSnapshot struct {
	ID               string    `json:"id"`
	ProjectID        string    `json:"project_id"`
	RunID            string    `json:"run_id"`
	AssetCount       int       `json:"asset_count"`
	PortCount        int       `json:"port_count"`
	EndpointCount    int       `json:"endpoint_count"`
	ServiceCount     int       `json:"service_count"`
	AssetChangesJSON string    `json:"asset_changes_json"`
	CreatedAt        time.Time `json:"created_at"`
}

// CreateAssetSnapshot inserts a new asset snapshot.
func (q *Queries) CreateAssetSnapshot(snap *AssetSnapshot) error {
	if snap.ID == "" {
		snap.ID = util.GenerateID()
	}
	if snap.AssetChangesJSON == "" {
		snap.AssetChangesJSON = "{}"
	}
	_, err := q.db.Exec(`
		INSERT INTO asset_snapshots (id, project_id, run_id, asset_count, port_count, endpoint_count, service_count, asset_changes_json, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		snap.ID, snap.ProjectID, snap.RunID, snap.AssetCount, snap.PortCount, snap.EndpointCount, snap.ServiceCount,
		snap.AssetChangesJSON, snap.CreatedAt,
	)
	return err
}

// GetLatestAssetSnapshot returns the most recent snapshot for a project (before the given run).
func (q *Queries) GetLatestAssetSnapshot(projectID, excludeRunID string) (*AssetSnapshot, error) {
	row := q.db.QueryRow(`
		SELECT id, project_id, run_id, asset_count, port_count, endpoint_count, service_count, asset_changes_json, created_at
		FROM asset_snapshots
		WHERE project_id = ?
		ORDER BY created_at DESC
		LIMIT 1`, projectID)
	snap := &AssetSnapshot{}
	err := row.Scan(&snap.ID, &snap.ProjectID, &snap.RunID, &snap.AssetCount, &snap.PortCount,
		&snap.EndpointCount, &snap.ServiceCount, &snap.AssetChangesJSON, &snap.CreatedAt)
	if err != nil {
		return nil, err
	}
	return snap, nil
}
