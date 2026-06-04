package db

import "database/sql"

// migrateV30 adds bounty_candidates table for SRC bounty queue.
func migrateV30(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS bounty_candidates (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
			program_id TEXT REFERENCES src_programs(id) ON DELETE SET NULL,
			finding_id TEXT REFERENCES findings(id) ON DELETE SET NULL,
			endpoint_id TEXT,
			source_kind TEXT NOT NULL DEFAULT 'finding',
			title TEXT NOT NULL DEFAULT '',
			vuln_type TEXT NOT NULL DEFAULT '',
			severity TEXT NOT NULL DEFAULT 'info',
			confidence TEXT NOT NULL DEFAULT 'low',
			value_score INTEGER NOT NULL DEFAULT 0,
			impact_score INTEGER NOT NULL DEFAULT 0,
			novelty_score INTEGER NOT NULL DEFAULT 0,
			repro_score INTEGER NOT NULL DEFAULT 0,
			scope_score INTEGER NOT NULL DEFAULT 0,
			safety_score INTEGER NOT NULL DEFAULT 0,
			duplicate_risk TEXT NOT NULL DEFAULT 'unknown',
			ranking_reason TEXT NOT NULL DEFAULT '',
			verify_status TEXT NOT NULL DEFAULT 'pending',
			submission_status TEXT NOT NULL DEFAULT 'not_ready',
			submission_url TEXT NOT NULL DEFAULT '',
			submission_id TEXT NOT NULL DEFAULT '',
			paid_amount INTEGER NOT NULL DEFAULT 0,
			notes TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(project_id, finding_id)
		);
		CREATE INDEX IF NOT EXISTS idx_bounty_candidates_project_id ON bounty_candidates(project_id);
		CREATE INDEX IF NOT EXISTS idx_bounty_candidates_program_id ON bounty_candidates(program_id);
		CREATE INDEX IF NOT EXISTS idx_bounty_candidates_finding_id ON bounty_candidates(finding_id);
		CREATE INDEX IF NOT EXISTS idx_bounty_candidates_value_score ON bounty_candidates(value_score DESC);
		CREATE INDEX IF NOT EXISTS idx_bounty_candidates_verify_status ON bounty_candidates(verify_status);
		CREATE INDEX IF NOT EXISTS idx_bounty_candidates_submission_status ON bounty_candidates(submission_status);
	`)
	return err
}
