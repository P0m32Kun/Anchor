package db

import (
	"database/sql"
	"fmt"
)

func migrateV14(db *sql.DB) error {
	// Check if reports table already exists
	var hasTable int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='reports'`).Scan(&hasTable); err != nil {
		return fmt.Errorf("check reports table: %w", err)
	}
	if hasTable > 0 {
		return nil
	}

	if _, err := db.Exec(`
CREATE TABLE IF NOT EXISTS reports (
    id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL REFERENCES pipeline_runs(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'generating' CHECK(status IN ('generating','partial','complete','failed')),
    title TEXT,
    finding_count INTEGER DEFAULT 0,
    evidence_count INTEGER DEFAULT 0,
    file_path TEXT,
    file_size_bytes INTEGER DEFAULT 0,
    error_message TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME,
    UNIQUE(run_id)
);
CREATE INDEX IF NOT EXISTS idx_reports_run ON reports(run_id);
CREATE INDEX IF NOT EXISTS idx_reports_status ON reports(status);
`); err != nil {
		return fmt.Errorf("create reports table: %w", err)
	}
	return nil
}
