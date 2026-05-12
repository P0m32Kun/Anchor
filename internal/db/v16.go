package db

import "database/sql"

// migrateV16 creates tables for dictionary management and httpx custom fingerprint
// management. These support the ffuf directory scanner and custom httpx tech-detect
// / favicon fingerprint files respectively.
func migrateV16(db *sql.DB) error {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS dictionaries (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			category TEXT NOT NULL CHECK(category IN ('dirscan','subdomain','vhost','custom')),
			file_path TEXT NOT NULL,
			line_count INTEGER DEFAULT 0,
			size_bytes INTEGER DEFAULT 0,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_dictionaries_category ON dictionaries(category);
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS httpx_fingerprints (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			type TEXT NOT NULL CHECK(type IN ('favicon','tech_detect')),
			file_path TEXT NOT NULL,
			enabled INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_httpx_fp_type ON httpx_fingerprints(type);
		CREATE INDEX IF NOT EXISTS idx_httpx_fp_enabled ON httpx_fingerprints(enabled);
	`); err != nil {
		return err
	}

	return nil
}
