package db

import (
	"database/sql"
	"fmt"
)

// migrateV12 fixes a long-standing bug where scan_tasks.run_id was declared as
// REFERENCES runs(id) by an early v0.2 migration. The application now records
// pipeline run IDs from pipeline_runs (added in v6), so every scan_tasks INSERT
// fails with FOREIGN KEY constraint failed and the entire scan silently does
// nothing. This rebuilds scan_tasks with run_id REFERENCES pipeline_runs(id).
func migrateV12(db *sql.DB) error {
	// Detect whether the bad FK is present. If the column is missing or already
	// points at pipeline_runs there is nothing to do.
	rows, err := db.Query(`SELECT "table" FROM pragma_foreign_key_list('scan_tasks') WHERE "from" = 'run_id'`)
	if err != nil {
		return fmt.Errorf("query scan_tasks foreign keys: %w", err)
	}
	var refTable string
	if rows.Next() {
		_ = rows.Scan(&refTable)
	}
	rows.Close()
	if refTable == "" || refTable == "pipeline_runs" {
		return nil
	}

	// Rebuild scan_tasks with the correct FK. SQLite requires foreign_keys=OFF
	// during the table rebuild because other tables reference scan_tasks.
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
			worker_id TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			steps_json TEXT,
			tool_template_id TEXT,
			run_id TEXT REFERENCES pipeline_runs(id) ON DELETE CASCADE,
			nuclei_custom_bundle_version TEXT
		);
	`); err != nil {
		return fmt.Errorf("create scan_tasks_new: %w", err)
	}

	if _, err := tx.Exec(`
		INSERT INTO scan_tasks_new (id, project_id, plan_id, depends_on_task_id, target_id, tool, command_template, arguments_redacted, status, started_at, finished_at, exit_code, worker_id, created_at, steps_json, tool_template_id, run_id, nuclei_custom_bundle_version)
		SELECT id, project_id, plan_id, depends_on_task_id, target_id, tool, command_template, arguments_redacted, status, started_at, finished_at, exit_code, worker_id, created_at, steps_json, tool_template_id,
			CASE WHEN run_id IN (SELECT id FROM pipeline_runs) THEN run_id ELSE NULL END,
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
	if _, err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_tasks_plan ON scan_tasks(plan_id)`); err != nil {
		return fmt.Errorf("recreate idx_tasks_plan: %w", err)
	}
	if _, err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_tasks_project ON scan_tasks(project_id)`); err != nil {
		return fmt.Errorf("recreate idx_tasks_project: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	committed = true

	return nil
}
