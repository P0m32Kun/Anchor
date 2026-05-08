package db

import (
	"database/sql"
	"fmt"
)

// migrateAddRateLimit adds the rate_limit column to projects table.
// SQLite does not support IF NOT EXISTS on ALTER TABLE ADD COLUMN,
// so we check pragma_table_info first.
func migrateAddRateLimit(db *sql.DB) error {
	rows, err := db.Query(`SELECT name FROM pragma_table_info('projects') WHERE name = 'rate_limit'`)
	if err != nil {
		return fmt.Errorf("check rate_limit column: %w", err)
	}
	defer rows.Close()
	if rows.Next() {
		return nil // column already exists
	}
	if _, err := db.Exec(`ALTER TABLE projects ADD COLUMN rate_limit INTEGER DEFAULT 0`); err != nil {
		return fmt.Errorf("add rate_limit column: %w", err)
	}
	return nil
}
