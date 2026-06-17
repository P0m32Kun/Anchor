package db

import "database/sql"

// migrateV40 adds asset_snapshots table for tracking asset changes between scan runs.
func migrateV40(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS asset_snapshots (
			id              TEXT PRIMARY KEY,
			project_id      TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
			run_id          TEXT NOT NULL,
			asset_count     INTEGER NOT NULL DEFAULT 0,
			port_count      INTEGER NOT NULL DEFAULT 0,
			endpoint_count  INTEGER NOT NULL DEFAULT 0,
			service_count   INTEGER NOT NULL DEFAULT 0,
			asset_changes_json TEXT NOT NULL DEFAULT '{}',
			created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_asset_snapshots_project
			ON asset_snapshots(project_id, created_at DESC);
	`)
	return err
}
