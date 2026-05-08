package db

import (
	"database/sql"
	"fmt"
)

// migrateV10 introduces server-side custom Nuclei template management:
//   - nuclei_custom_sources: user-defined source registry
//   - nuclei_custom_bundles: immutable published bundles (Phase 2 fills this)
//   - scan_tasks.nuclei_custom_bundle_version: per-task bundle attribution
//     (Phase 4 populates; Phase 1 ships the column to avoid a follow-up migration)
func migrateV10(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS nuclei_custom_sources (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			type TEXT NOT NULL CHECK(type IN ('git','upload','file')),
			uri TEXT,
			branch TEXT,
			enabled INTEGER NOT NULL DEFAULT 1,
			routing_policy TEXT NOT NULL DEFAULT 'manual',
			status TEXT NOT NULL DEFAULT 'draft',
			last_sync_at DATETIME,
			last_validate_at DATETIME,
			last_error TEXT,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_nuclei_custom_sources_status ON nuclei_custom_sources(status);
	`)
	if err != nil {
		return fmt.Errorf("create nuclei_custom_sources: %w", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS nuclei_custom_bundles (
			version TEXT PRIMARY KEY,
			manifest_json TEXT NOT NULL,
			archive_path TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'active',
			created_at DATETIME NOT NULL,
			activated_at DATETIME
		);
	`)
	if err != nil {
		return fmt.Errorf("create nuclei_custom_bundles: %w", err)
	}

	var colCount int
	err = db.QueryRow(
		`SELECT COUNT(*) FROM pragma_table_info('scan_tasks') WHERE name = 'nuclei_custom_bundle_version'`,
	).Scan(&colCount)
	if err != nil {
		return fmt.Errorf("check nuclei_custom_bundle_version column: %w", err)
	}
	if colCount == 0 {
		if _, err := db.Exec(`ALTER TABLE scan_tasks ADD COLUMN nuclei_custom_bundle_version TEXT`); err != nil {
			return fmt.Errorf("add nuclei_custom_bundle_version column: %w", err)
		}
	}

	return nil
}
