package db

import (
	"database/sql"
	"fmt"
)

func migrateV04(db *sql.DB) error {
	// 1. Update targets table CHECK constraint to support 'company' type
	// SQLite doesn't support ALTER TABLE DROP CONSTRAINT, so we need to recreate
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS targets_new (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
			type TEXT NOT NULL CHECK(type IN ('domain','url','ip','cidr','company')),
			value TEXT NOT NULL,
			source TEXT DEFAULT 'manual',
			status TEXT DEFAULT 'active',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		return fmt.Errorf("create targets_new: %w", err)
	}

	_, err = db.Exec(`
		INSERT INTO targets_new (id, project_id, type, value, source, status, created_at)
		SELECT id, project_id, type, value, source, status, created_at FROM targets;
	`)
	if err != nil {
		return fmt.Errorf("copy targets: %w", err)
	}

	_, err = db.Exec(`DROP TABLE targets;`)
	if err != nil {
		return fmt.Errorf("drop old targets: %w", err)
	}

	_, err = db.Exec(`ALTER TABLE targets_new RENAME TO targets;`)
	if err != nil {
		return fmt.Errorf("rename targets_new: %w", err)
	}

	// Recreate index
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_targets_project ON targets(project_id);`)
	if err != nil {
		return fmt.Errorf("create index: %w", err)
	}

	// 2. Add FOFA config to projects
	var colCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('projects') WHERE name = 'fofa_email'`).Scan(&colCount)
	if err != nil {
		return fmt.Errorf("check fofa_email column: %w", err)
	}
	if colCount == 0 {
		_, err = db.Exec(`ALTER TABLE projects ADD COLUMN fofa_email TEXT;`)
		if err != nil {
			return fmt.Errorf("add fofa_email: %w", err)
		}
		_, err = db.Exec(`ALTER TABLE projects ADD COLUMN fofa_api_key TEXT;`)
		if err != nil {
			return fmt.Errorf("add fofa_api_key: %w", err)
		}
		_, err = db.Exec(`ALTER TABLE projects ADD COLUMN pipeline_config TEXT;`)
		if err != nil {
			return fmt.Errorf("add pipeline_config: %w", err)
		}
	}

	// 3. Create DNS records table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS dns_records (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
			domain TEXT NOT NULL,
			ips TEXT NOT NULL,
			cnames TEXT,
			ttl INTEGER,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(project_id, domain)
		);
	`)
	if err != nil {
		return fmt.Errorf("create dns_records: %w", err)
	}

	// 4. Create CDN results table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS cdn_results (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
			ip TEXT NOT NULL,
			is_cdn BOOLEAN DEFAULT FALSE,
			cdn_provider TEXT,
			cdn_type TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(project_id, ip)
		);
	`)
	if err != nil {
		return fmt.Errorf("create cdn_results: %w", err)
	}

	// 5. Create service fingerprints table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS service_fingerprints (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
			ip TEXT NOT NULL,
			port INTEGER NOT NULL,
			protocol TEXT DEFAULT 'tcp',
			is_web BOOLEAN DEFAULT FALSE,
			service TEXT NOT NULL,
			metadata TEXT,
			source TEXT DEFAULT 'nerva',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(project_id, ip, port)
		);
	`)
	if err != nil {
		return fmt.Errorf("create service_fingerprints: %w", err)
	}

	return nil
}
