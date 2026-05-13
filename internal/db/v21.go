package db

import (
	"database/sql"
	"fmt"
)

// migrateV21 removes the FK constraint on scan_tasks.run_id.
//
// Two code paths write scan_tasks.run_id:
//   - dispatchRun (run_handlers.go) uses runs.id
//   - handleRunPipeline (pipeline_handlers.go) uses pipeline_runs.id
//
// A single FK column cannot reference two tables. v12 set it to pipeline_runs,
// v21 (original) reverted to runs — both break one path. The correct fix is to
// drop the FK entirely and let application logic handle referential integrity.
func migrateV21(db *sql.DB) error {
	// Detect current FK target for run_id.
	rows, err := db.Query(`SELECT "table" FROM pragma_foreign_key_list('scan_tasks') WHERE "from" = 'run_id'`)
	if err != nil {
		return fmt.Errorf("query scan_tasks foreign keys: %w", err)
	}
	var refTable string
	if rows.Next() {
		_ = rows.Scan(&refTable)
	}
	rows.Close()

	// If no FK at all, nothing to do.
	if refTable == "" {
		return nil
	}

	if _, err := db.Exec(`PRAGMA foreign_keys = OFF`); err != nil {
		return fmt.Errorf("disable foreign keys: %w", err)
	}
	defer func() { _, _ = db.Exec(`PRAGMA foreign_keys = ON`) }()

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	// Create new table with corrected FK.
	if _, err := tx.Exec(`
		CREATE TABLE scan_tasks_new (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
			plan_id TEXT REFERENCES scan_plans(id) ON DELETE CASCADE,
			depends_on_task_id TEXT REFERENCES scan_tasks(id) ON DELETE SET NULL,
			target_id TEXT REFERENCES targets(id) ON DELETE SET NULL,
			tool TEXT NOT NULL,
			command_template TEXT,
			arguments_redacted TEXT,
			status TEXT DEFAULT 'created' CHECK(status IN ('created','queued','running','completed','partial_success','failed','cancelled','scope_denied')),
			started_at DATETIME,
			finished_at DATETIME,
			exit_code INTEGER,
			error_message TEXT,
			worker_id TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			steps_json TEXT,
			tool_template_id TEXT,
			run_id TEXT,
			nuclei_custom_bundle_version TEXT
		);
	`); err != nil {
		return fmt.Errorf("create scan_tasks_new: %w", err)
	}

	// Copy data, NULLing run_id values that don't exist in runs.
	if _, err := tx.Exec(`
		INSERT INTO scan_tasks_new (id, project_id, plan_id, depends_on_task_id, target_id, tool, command_template, arguments_redacted, status, started_at, finished_at, exit_code, error_message, worker_id, created_at, steps_json, tool_template_id, run_id, nuclei_custom_bundle_version)
		SELECT id, project_id, plan_id, depends_on_task_id, target_id, tool, command_template, arguments_redacted, status, started_at, finished_at, exit_code, error_message, worker_id, created_at, steps_json, tool_template_id,
			CASE WHEN run_id IN (SELECT id FROM runs) THEN run_id ELSE NULL END,
			nuclei_custom_bundle_version
		FROM scan_tasks;
	`); err != nil {
		return fmt.Errorf("copy scan_tasks: %w", err)
	}

	if _, err := tx.Exec(`DROP TABLE scan_tasks`); err != nil {
		return fmt.Errorf("drop scan_tasks: %w", err)
	}
	if _, err := tx.Exec(`ALTER TABLE scan_tasks_new RENAME TO scan_tasks`); err != nil {
		return fmt.Errorf("rename scan_tasks_new: %w", err)
	}

	// Recreate indexes.
	for _, idx := range []string{
		`CREATE INDEX IF NOT EXISTS idx_tasks_plan ON scan_tasks(plan_id)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_project ON scan_tasks(project_id)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_run ON scan_tasks(run_id)`,
	} {
		if _, err := tx.Exec(idx); err != nil {
			return fmt.Errorf("create index: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	committed = true

	return nil
}
