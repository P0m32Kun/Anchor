package db

import "database/sql"

// migrateV26 adds the scan_work_items table and extends pipeline_runs,
// pipeline_run_stages, scan_tasks, assets, and web_endpoints for the
// asset-driven scan engine.
func migrateV26(db *sql.DB) error {
	stmts := []string{
		// --- scan_work_items (new table) ---
		`CREATE TABLE IF NOT EXISTS scan_work_items (
			id TEXT PRIMARY KEY,
			run_id TEXT NOT NULL,
			project_id TEXT NOT NULL,
			asset_id TEXT NOT NULL,
			action TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			skip_reason TEXT,
			stage TEXT,
			error TEXT,
			started_at DATETIME,
			completed_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(run_id, asset_id, action)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_scan_work_items_run_status ON scan_work_items(run_id, status)`,

		// --- pipeline_runs extensions ---
		`ALTER TABLE pipeline_runs ADD COLUMN engine_state TEXT NOT NULL DEFAULT 'running'`,
		`ALTER TABLE pipeline_runs ADD COLUMN last_new_asset_at TEXT`,

		// --- pipeline_run_stages extensions ---
		`ALTER TABLE pipeline_run_stages ADD COLUMN work_total INTEGER`,
		`ALTER TABLE pipeline_run_stages ADD COLUMN work_done INTEGER`,
		`ALTER TABLE pipeline_run_stages ADD COLUMN work_running INTEGER`,
		`ALTER TABLE pipeline_run_stages ADD COLUMN round INTEGER`,

		// --- scan_tasks extensions ---
		`ALTER TABLE scan_tasks ADD COLUMN work_id TEXT`,
		`ALTER TABLE scan_tasks ADD COLUMN action TEXT`,
		`ALTER TABLE scan_tasks ADD COLUMN asset_id TEXT`,
		`ALTER TABLE scan_tasks ADD COLUMN stage TEXT`,

		// --- assets / web_endpoints extensions ---
		`ALTER TABLE assets ADD COLUMN discovery_depth INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE web_endpoints ADD COLUMN discovery_depth INTEGER NOT NULL DEFAULT 0`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}
