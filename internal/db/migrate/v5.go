package db

import (
	"database/sql"
	"fmt"
)

func migrateV05(db *sql.DB) error {
	// Check if start_time column exists in projects
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('projects') WHERE name = 'start_time'`).Scan(&count)
	if err != nil {
		return err
	}
	if count == 0 {
		// Column already removed, nothing to do
		return nil
	}

	// SQLite doesn't support DROP COLUMN, so we need to recreate the table
	_, err = db.Exec(`
		CREATE TABLE projects_new (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			organization TEXT,
			purpose TEXT,
			default_profile TEXT DEFAULT 'standard',
			port_range TEXT,
			fofa_email TEXT,
			fofa_api_key TEXT,
			pipeline_config TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		return fmt.Errorf("create projects_new: %w", err)
	}

	_, err = db.Exec(`
		INSERT INTO projects_new (id, name, organization, purpose, default_profile, port_range, fofa_email, fofa_api_key, pipeline_config, created_at, updated_at)
		SELECT id, name, organization, purpose, default_profile, port_range, fofa_email, fofa_api_key, pipeline_config, created_at, updated_at
		FROM projects;
	`)
	if err != nil {
		return fmt.Errorf("copy projects: %w", err)
	}

	_, err = db.Exec(`DROP TABLE projects;`)
	if err != nil {
		return fmt.Errorf("drop old projects: %w", err)
	}

	_, err = db.Exec(`ALTER TABLE projects_new RENAME TO projects;`)
	if err != nil {
		return fmt.Errorf("rename projects_new: %w", err)
	}

	return nil
}
