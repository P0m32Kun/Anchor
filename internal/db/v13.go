package db

import (
	"database/sql"
	"fmt"
)

func migrateV13(db *sql.DB) error {
	var hasCol int
	if err := db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('scan_tasks') WHERE name = 'error_message'`).Scan(&hasCol); err != nil {
		return fmt.Errorf("check error_message column: %w", err)
	}
	if hasCol > 0 {
		return nil
	}
	if _, err := db.Exec(`ALTER TABLE scan_tasks ADD COLUMN error_message TEXT`); err != nil {
		return fmt.Errorf("add error_message column: %w", err)
	}
	return nil
}
