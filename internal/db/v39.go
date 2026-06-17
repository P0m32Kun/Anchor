package db

import (
	"database/sql"
	"log"
)

func migrateV39(db *sql.DB) error {
	log.Println("[migrate] v39: scan_work_items batch columns...")

	cols := []string{
		"ALTER TABLE scan_work_items ADD COLUMN input_file TEXT",
		"ALTER TABLE scan_work_items ADD COLUMN member_asset_ids TEXT",
		"ALTER TABLE scan_work_items ADD COLUMN bucket_key TEXT",
		"ALTER TABLE scan_work_items ADD COLUMN generation INTEGER",
		"ALTER TABLE scan_work_items ADD COLUMN batch_mode INTEGER NOT NULL DEFAULT 0",
	}
	for _, stmt := range cols {
		if _, err := db.Exec(stmt); err != nil {
			if !isDuplicateColumnError(err) {
				return err
			}
		}
	}

	log.Println("[migrate] v39: done")
	return nil
}
