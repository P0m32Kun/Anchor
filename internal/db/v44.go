package db

import "database/sql"

// migrateV44 adds the retest_runs table for tracking finding retest results.
func migrateV44(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS retest_runs (
			id          TEXT PRIMARY KEY,
			finding_id  TEXT NOT NULL REFERENCES findings(id) ON DELETE CASCADE,
			task_id     TEXT NOT NULL DEFAULT '',
			result      TEXT NOT NULL DEFAULT 'still_present'
				CHECK(result IN ('still_present','fixed','inconclusive','failed_to_test')),
			evidence_id TEXT REFERENCES evidence(id) ON DELETE SET NULL,
			created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_retest_runs_finding
			ON retest_runs(finding_id);
	`)
	return err
}
