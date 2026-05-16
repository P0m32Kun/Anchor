package db

import (
	"database/sql"
	"fmt"
)

// migrateV22 adds install_path to nuclei_custom_sources so each custom
// template source can declare its subdirectory under ~/nuclei-templates/.
// Defaults to the source name for existing rows.
//
// This column enables the worker to drop -t/-w injection entirely: all
// custom templates live under nuclei's default search path, so nuclei
// finds them natively without extra flags.
func migrateV22(db *sql.DB) error {
	var colCount int
	err := db.QueryRow(
		`SELECT COUNT(*) FROM pragma_table_info('nuclei_custom_sources') WHERE name = 'install_path'`,
	).Scan(&colCount)
	if err != nil {
		return fmt.Errorf("check install_path column: %w", err)
	}
	if colCount > 0 {
		return nil
	}

	if _, err := db.Exec(`ALTER TABLE nuclei_custom_sources ADD COLUMN install_path TEXT NOT NULL DEFAULT ''`); err != nil {
		return fmt.Errorf("add install_path column: %w", err)
	}

	// Backfill: set install_path = name for existing rows.
	if _, err := db.Exec(`UPDATE nuclei_custom_sources SET install_path = name WHERE install_path = ''`); err != nil {
		return fmt.Errorf("backfill install_path: %w", err)
	}

	return nil
}
