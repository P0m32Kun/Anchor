package db

import "database/sql"

// migrateV27 adds the excluded_domains table for user-configurable domain exclusions.
func migrateV27(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS excluded_domains (
			id TEXT PRIMARY KEY,
			domain TEXT NOT NULL UNIQUE,
			reason TEXT NOT NULL DEFAULT '',
			builtin INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_excluded_domains_domain ON excluded_domains(domain)`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}
