package db

import (
	"database/sql"
	"fmt"
)

func migrateV07(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS pipeline_run_stages (
			id TEXT PRIMARY KEY,
			run_id TEXT NOT NULL REFERENCES pipeline_runs(id) ON DELETE CASCADE,
			stage TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','running','completed','failed','skipped')),
			error TEXT,
			started_at DATETIME,
			completed_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_pipeline_run_stages_run ON pipeline_run_stages(run_id);
		CREATE INDEX IF NOT EXISTS idx_pipeline_run_stages_stage ON pipeline_run_stages(run_id, stage);
	`)
	if err != nil {
		return fmt.Errorf("create pipeline_run_stages: %w", err)
	}
	return nil
}
