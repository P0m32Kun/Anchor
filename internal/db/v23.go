package db

import (
	"database/sql"
	"fmt"
)

// migrateV23 adds run_id to findings so run-scoped reports can filter
// findings by pipeline run. The column is nullable — findings created
// by project-level workflows (asset discovery, web screening) will
// have NULL run_id and still appear in project-scoped reports.
func migrateV23(db *sql.DB) error {
	var colCount int
	err := db.QueryRow(
		`SELECT COUNT(*) FROM pragma_table_info('findings') WHERE name = 'run_id'`,
	).Scan(&colCount)
	if err != nil {
		return fmt.Errorf("check run_id column: %w", err)
	}
	if colCount > 0 {
		return nil
	}

	if _, err := db.Exec(`ALTER TABLE findings ADD COLUMN run_id TEXT REFERENCES runs(id) ON DELETE SET NULL`); err != nil {
		return fmt.Errorf("add run_id column: %w", err)
	}

	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_findings_run ON findings(run_id)`); err != nil {
		return fmt.Errorf("create run_id index: %w", err)
	}

	return nil
}
