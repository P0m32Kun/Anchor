package db

import (
	"database/sql"
	"fmt"
)

// migrateV11 drops legacy FOFA email columns now that the FOFA API no longer
// requires an email parameter. It rebuilds projects without fofa_email/
// fofa_api_key and engine_credentials without email.
func migrateV11(db *sql.DB) error {
	// 1. Rebuild projects table without fofa_email and fofa_api_key
	var fofaEmailCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('projects') WHERE name = 'fofa_email'`).Scan(&fofaEmailCount); err != nil {
		return fmt.Errorf("check fofa_email column: %w", err)
	}
	if fofaEmailCount > 0 {
		_, err := db.Exec(`
			CREATE TABLE projects_new (
				id TEXT PRIMARY KEY,
				name TEXT NOT NULL,
				organization TEXT,
				purpose TEXT,
				default_profile TEXT DEFAULT 'standard',
				port_range TEXT,
				rate_limit INTEGER DEFAULT 0,
				pipeline_config TEXT,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
			);
		`)
		if err != nil {
			return fmt.Errorf("create projects_new: %w", err)
		}

		_, err = db.Exec(`
			INSERT INTO projects_new (id, name, organization, purpose, default_profile, port_range, rate_limit, pipeline_config, created_at, updated_at)
			SELECT id, name, organization, purpose, default_profile, port_range, COALESCE(rate_limit, 0), pipeline_config, created_at, updated_at
			FROM projects;
		`)
		if err != nil {
			return fmt.Errorf("copy projects: %w", err)
		}

		if _, err := db.Exec(`DROP TABLE projects;`); err != nil {
			return fmt.Errorf("drop old projects: %w", err)
		}

		if _, err := db.Exec(`ALTER TABLE projects_new RENAME TO projects;`); err != nil {
			return fmt.Errorf("rename projects_new: %w", err)
		}
	}

	// 2. Rebuild engine_credentials table without email
	var emailCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('engine_credentials') WHERE name = 'email'`).Scan(&emailCount); err != nil {
		return fmt.Errorf("check engine_credentials.email column: %w", err)
	}
	if emailCount > 0 {
		_, err := db.Exec(`
			CREATE TABLE engine_credentials_new (
				id TEXT PRIMARY KEY,
				engine TEXT NOT NULL UNIQUE,
				api_key TEXT NOT NULL,
				extra TEXT,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
			);
		`)
		if err != nil {
			return fmt.Errorf("create engine_credentials_new: %w", err)
		}

		_, err = db.Exec(`
			INSERT INTO engine_credentials_new (id, engine, api_key, extra, created_at, updated_at)
			SELECT id, engine, api_key, extra, created_at, updated_at
			FROM engine_credentials;
		`)
		if err != nil {
			return fmt.Errorf("copy engine_credentials: %w", err)
		}

		if _, err := db.Exec(`DROP TABLE engine_credentials;`); err != nil {
			return fmt.Errorf("drop old engine_credentials: %w", err)
		}

		if _, err := db.Exec(`ALTER TABLE engine_credentials_new RENAME TO engine_credentials;`); err != nil {
			return fmt.Errorf("rename engine_credentials_new: %w", err)
		}
	}

	return nil
}
