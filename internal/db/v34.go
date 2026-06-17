package db

import "database/sql"

// migrateV34 adds state_json to assets for scan gating attribute persistence.
func migrateV34(db *sql.DB) error {
	if _, err := db.Exec(`ALTER TABLE assets ADD COLUMN state_json TEXT NOT NULL DEFAULT '{}'`); err != nil {
		if !isDuplicateColumnError(err) {
			return err
		}
	}
	return nil
}
