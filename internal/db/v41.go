package db

import "database/sql"

// migrateV41 adds asset_changes and alert_webhooks tables for tracking
// granular asset changes between scan runs and webhook alert configuration.
func migrateV41(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS asset_changes (
			id              TEXT PRIMARY KEY,
			project_id      TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
			run_id          TEXT NOT NULL DEFAULT '',
			asset_id        TEXT NOT NULL DEFAULT '',
			asset_value     TEXT NOT NULL DEFAULT '',
			asset_type      TEXT NOT NULL DEFAULT '',
			change_type     TEXT NOT NULL DEFAULT '',
			change_summary  TEXT NOT NULL DEFAULT '',
			detail_json     TEXT NOT NULL DEFAULT '{}',
			severity        TEXT NOT NULL DEFAULT 'info',
			created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_asset_changes_project
			ON asset_changes(project_id, created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_asset_changes_asset
			ON asset_changes(asset_id);

		CREATE TABLE IF NOT EXISTS alert_webhooks (
			id              TEXT PRIMARY KEY,
			project_id      TEXT NOT NULL UNIQUE REFERENCES projects(id) ON DELETE CASCADE,
			enabled         INTEGER NOT NULL DEFAULT 1,
			url             TEXT NOT NULL DEFAULT '',
			secret          TEXT NOT NULL DEFAULT '',
			min_severity    TEXT NOT NULL DEFAULT 'info',
			on_new_asset    INTEGER NOT NULL DEFAULT 1,
			on_asset_gone   INTEGER NOT NULL DEFAULT 1,
			on_port_change  INTEGER NOT NULL DEFAULT 1,
			on_service_change INTEGER NOT NULL DEFAULT 1,
			on_cert_expiry  INTEGER NOT NULL DEFAULT 1,
			created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`)
	return err
}
