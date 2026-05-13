package db

import "database/sql"

// migrateV20 marks built-in dictionaries seeded from the worker/server image's
// /opt/dict tree so the API can refuse edits to them and the UI can render them
// as read-only. User-uploaded dictionaries keep builtin=0.
func migrateV20(db *sql.DB) error {
	stmts := []string{
		`ALTER TABLE dictionaries ADD COLUMN builtin INTEGER NOT NULL DEFAULT 0`,
		`CREATE INDEX IF NOT EXISTS idx_dictionaries_builtin ON dictionaries(builtin)`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}
