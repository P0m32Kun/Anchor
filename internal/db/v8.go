package db

import (
	"database/sql"
	"log"
)

func migrateV08(db *sql.DB) error {
	// Add mode column to pipeline_runs
	_, err := db.Exec(`ALTER TABLE pipeline_runs ADD COLUMN mode TEXT NOT NULL DEFAULT 'standard'`)
	if err != nil {
		// Column may already exist
		log.Printf("migration v8 (mode column): %v", err)
	}
	return nil
}
