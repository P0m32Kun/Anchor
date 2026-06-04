package db

import "database/sql"

// migrateV31 adds submission_packs table for SRC submission workflow.
func migrateV31(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS submission_packs (
			id TEXT PRIMARY KEY,
			candidate_id TEXT NOT NULL REFERENCES bounty_candidates(id) ON DELETE CASCADE,
			format TEXT NOT NULL DEFAULT 'markdown',
			template TEXT NOT NULL DEFAULT 'generic',
			content TEXT NOT NULL DEFAULT '',
			checklist_json TEXT NOT NULL DEFAULT '{}',
			redaction_status TEXT NOT NULL DEFAULT 'raw',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_submission_packs_candidate_id ON submission_packs(candidate_id);
	`)
	return err
}
