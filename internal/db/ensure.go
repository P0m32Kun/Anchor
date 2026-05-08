package db

import (
	"database/sql"
	"fmt"
)

// ensureProjectsColumns verifies that all expected columns exist on the
// projects table and adds any that are missing. This acts as a final
// safety net for databases that may have skipped an intermediate
// migration or had their schema altered outside the normal flow.
func ensureProjectsColumns(db *sql.DB) error {
	columns := []struct {
		name string
		def  string
	}{
		{"rate_limit", "INTEGER DEFAULT 0"},
	}

	for _, col := range columns {
		var count int
		err := db.QueryRow(
			"SELECT COUNT(*) FROM pragma_table_info('projects') WHERE name = ?",
			col.name,
		).Scan(&count)
		if err != nil {
			return fmt.Errorf("check column %s: %w", col.name, err)
		}
		if count == 0 {
			_, err := db.Exec(fmt.Sprintf("ALTER TABLE projects ADD COLUMN %s %s", col.name, col.def))
			if err != nil {
				return fmt.Errorf("add column %s: %w", col.name, err)
			}
		}
	}
	return nil
}
