package db

import (
	"database/sql"
	"fmt"
)

func migrateV06(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS pipeline_runs (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
			status TEXT NOT NULL DEFAULT 'running',
			stage TEXT,
			error TEXT,
			started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			completed_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_pipeline_runs_project ON pipeline_runs(project_id);
	`)
	if err != nil {
		return fmt.Errorf("create pipeline_runs: %w", err)
	}
	return nil
}
