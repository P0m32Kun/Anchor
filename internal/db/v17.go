package db

import "database/sql"

// migrateV17 creates the slow_scan_tasks table for background slow scanning.
// Supports urlfinder (pingc0y/URLFinder) and ffuf slow-scan tools.
func migrateV17(db *sql.DB) error {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS slow_scan_tasks (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
			target_id TEXT REFERENCES targets(id) ON DELETE CASCADE,
			run_id TEXT REFERENCES pipeline_runs(id) ON DELETE SET NULL,
			tool TEXT NOT NULL CHECK(tool IN ('urlfinder','ffuf')),
			status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','running','completed','failed','cancelled')),
			config_json TEXT NOT NULL DEFAULT '{}',
			rate_limit INTEGER DEFAULT 0,
			timeout INTEGER DEFAULT 0,
			error_message TEXT,
			started_at DATETIME,
			finished_at DATETIME,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_slow_scan_project ON slow_scan_tasks(project_id);
		CREATE INDEX IF NOT EXISTS idx_slow_scan_target ON slow_scan_tasks(target_id);
		CREATE INDEX IF NOT EXISTS idx_slow_scan_status ON slow_scan_tasks(status);
	`); err != nil {
		return err
	}
	return nil
}
