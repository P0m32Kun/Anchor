package db

import "database/sql"

// migrateV25 adds per-row enable flags for builtin dictionaries and marks
// seeded httpx fingerprints and nuclei custom sources as builtin assets.
func migrateV25(db *sql.DB) error {
	stmts := []string{
		`ALTER TABLE dictionaries ADD COLUMN enabled INTEGER NOT NULL DEFAULT 1`,
		`ALTER TABLE httpx_fingerprints ADD COLUMN builtin INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE nuclei_custom_sources ADD COLUMN builtin INTEGER NOT NULL DEFAULT 0`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}
