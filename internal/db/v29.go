package db

import "database/sql"

// migrateV29 adds src_programs table for SRC bounty workflow.
func migrateV29(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS src_programs (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
			name TEXT NOT NULL DEFAULT '',
			platform TEXT NOT NULL DEFAULT 'other',
			program_url TEXT NOT NULL DEFAULT '',
			rules_url TEXT NOT NULL DEFAULT '',
			allow_automation INTEGER NOT NULL DEFAULT 0,
			allow_dir_brute INTEGER NOT NULL DEFAULT 0,
			allow_weak_password INTEGER NOT NULL DEFAULT 0,
			allow_authenticated_test INTEGER NOT NULL DEFAULT 0,
			max_rps INTEGER NOT NULL DEFAULT 5,
			max_concurrency INTEGER NOT NULL DEFAULT 3,
			preferred_vuln_types TEXT NOT NULL DEFAULT '[]',
			payout_hint TEXT NOT NULL DEFAULT '{}',
			notes TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(project_id)
		);
		CREATE INDEX IF NOT EXISTS idx_src_programs_project_id ON src_programs(project_id);
		CREATE INDEX IF NOT EXISTS idx_src_programs_platform ON src_programs(platform);
	`)
	return err
}
