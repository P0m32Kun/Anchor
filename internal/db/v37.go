package db

import "database/sql"

// migrateV37 adds watch mode fields to the projects table (BW4).
func migrateV37(db *sql.DB) error {
	stmts := []string{
		`ALTER TABLE projects ADD COLUMN watch_enabled INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE projects ADD COLUMN watch_interval_hours INTEGER NOT NULL DEFAULT 24`,
		`ALTER TABLE projects ADD COLUMN watch_passive_only INTEGER NOT NULL DEFAULT 1`,
		`ALTER TABLE projects ADD COLUMN watch_last_tick_at DATETIME`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil && !isDuplicateColumnError(err) {
			return err
		}
	}
	return nil
}
