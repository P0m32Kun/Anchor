package db

import "database/sql"

// migrateV32 adds tool_call_logs table for scan tool call traceability
// and source_task_id column to findings for Finding→ScanTask linkage.
func migrateV32(db *sql.DB) error {
	// 1. Create tool_call_logs table
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS tool_call_logs (
			id TEXT PRIMARY KEY,
			run_id TEXT NOT NULL,
			work_item_id TEXT,
			task_id TEXT,
			tool TEXT NOT NULL,
			action TEXT NOT NULL,
			asset_id TEXT,
			params_json TEXT NOT NULL DEFAULT '{}',
			started_at DATETIME NOT NULL,
			finished_at DATETIME,
			duration_ms INTEGER,
			exit_code INTEGER,
			status TEXT NOT NULL DEFAULT 'running',
			output_summary TEXT,
			error_message TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`); err != nil {
		return err
	}

	// 2. Create indexes
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_tool_call_logs_run_id ON tool_call_logs(run_id)`); err != nil {
		return err
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_tool_call_logs_task_id ON tool_call_logs(task_id)`); err != nil {
		return err
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_tool_call_logs_work_item_id ON tool_call_logs(work_item_id)`); err != nil {
		return err
	}

	// 3. Add source_task_id to findings
	if _, err := db.Exec(`ALTER TABLE findings ADD COLUMN source_task_id TEXT`); err != nil {
		// Column may already exist from a previous partial migration; ignore duplicate column errors
		// SQLite error: "duplicate column name: source_task_id"
		if !isDuplicateColumnError(err) {
			return err
		}
	}

	return nil
}

// isDuplicateColumnError checks if the error is SQLite's "duplicate column name" error.
func isDuplicateColumnError(err error) bool {
	if err == nil {
		return false
	}
	// SQLite returns: "duplicate column name: <column>"
	return contains(err.Error(), "duplicate column name")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
