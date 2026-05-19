package db

import "database/sql"

// migrateV24 adds match_keys column to finding_templates and backfills
// existing single-value match_key entries into JSON arrays.
func migrateV24(db *sql.DB) error {
	// Step 1: add match_keys column with default empty JSON array
	if _, err := db.Exec(`ALTER TABLE finding_templates ADD COLUMN match_keys TEXT NOT NULL DEFAULT '[]'`); err != nil {
		return err
	}

	// Step 2: migrate existing match_key values into JSON arrays
	if _, err := db.Exec(`
		UPDATE finding_templates
		SET match_keys = json_array(match_key)
		WHERE match_key != ''
	`); err != nil {
		return err
	}

	return nil
}
