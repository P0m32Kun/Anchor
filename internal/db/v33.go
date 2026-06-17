package db

import "database/sql"

// migrateV33 adds asset_relations for scan lineage (target/asset graph edges).
func migrateV33(db *sql.DB) error {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS asset_relations (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL,
			run_id TEXT NOT NULL DEFAULT '',
			source_type TEXT NOT NULL CHECK(source_type IN ('target','asset')),
			source_id TEXT NOT NULL,
			target_type TEXT NOT NULL CHECK(target_type IN ('asset')),
			target_id TEXT NOT NULL,
			relation_type TEXT NOT NULL CHECK(relation_type IN ('expanded_by','discovered_from','resolves_to','contains')),
			source_engine TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(project_id, source_type, source_id, target_type, target_id, relation_type, run_id)
		)`); err != nil {
		return err
	}

	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_asset_relations_project ON asset_relations(project_id)`); err != nil {
		return err
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_asset_relations_target ON asset_relations(target_id)`); err != nil {
		return err
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_asset_relations_source ON asset_relations(source_id)`); err != nil {
		return err
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_asset_relations_run ON asset_relations(run_id)`); err != nil {
		return err
	}

	return nil
}
