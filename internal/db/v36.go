package db

import "database/sql"

// migrateV36 adds the signals table for the Signal Inbox (BW2).
// Signals are auto-generated from findings, Spoor output, new assets, and endpoints.
func migrateV36(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS signals (
			id             TEXT PRIMARY KEY,
			project_id     TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
			source_kind    TEXT NOT NULL DEFAULT 'finding',
			source_id      TEXT NOT NULL DEFAULT '',
			title          TEXT NOT NULL DEFAULT '',
			severity       TEXT NOT NULL DEFAULT 'info',
			score          INTEGER NOT NULL DEFAULT 0,
			scope_status   TEXT NOT NULL DEFAULT 'in_scope',
			status         TEXT NOT NULL DEFAULT 'new',
			metadata       TEXT NOT NULL DEFAULT '{}',
			first_seen     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			last_seen      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			created_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_signals_project_id ON signals(project_id);
		CREATE INDEX IF NOT EXISTS idx_signals_score ON signals(score DESC);
		CREATE INDEX IF NOT EXISTS idx_signals_status ON signals(status);
		CREATE INDEX IF NOT EXISTS idx_signals_source ON signals(source_kind, source_id);
	`)
	if err != nil {
		return err
	}
	return nil
}
