package db

import (
	"database/sql"
	"fmt"
)

// migrateV45 extends the assets.type CHECK constraint to include 'cidr',
// aligning the DB schema with the engine's core.AssetCIDR type.
// SQLite does not support ALTER TABLE DROP CONSTRAINT, so we recreate the table.
func migrateV45(db *sql.DB) error {
	// 1. Create new assets table with extended CHECK constraint
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS assets_new (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
			type TEXT NOT NULL CHECK(type IN ('domain', 'ip', 'url', 'cidr')),
			value TEXT NOT NULL,
			normalized_value TEXT NOT NULL,
			source_tools TEXT,
			first_seen DATETIME DEFAULT CURRENT_TIMESTAMP,
			last_seen DATETIME DEFAULT CURRENT_TIMESTAMP,
			tags TEXT,
			UNIQUE(project_id, normalized_value)
		);
	`)
	if err != nil {
		return fmt.Errorf("create assets_new: %w", err)
	}

	// 2. Copy data from old table
	_, err = db.Exec(`
		INSERT INTO assets_new (id, project_id, type, value, normalized_value, source_tools, first_seen, last_seen, tags)
		SELECT id, project_id, type, value, normalized_value, source_tools, first_seen, last_seen, tags FROM assets;
	`)
	if err != nil {
		return fmt.Errorf("copy assets: %w", err)
	}

	// 3. Drop old table
	_, err = db.Exec(`DROP TABLE assets;`)
	if err != nil {
		return fmt.Errorf("drop old assets: %w", err)
	}

	// 4. Rename new table
	_, err = db.Exec(`ALTER TABLE assets_new RENAME TO assets;`)
	if err != nil {
		return fmt.Errorf("rename assets_new: %w", err)
	}

	// 5. Recreate indexes
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_assets_project ON assets(project_id);`)
	if err != nil {
		return fmt.Errorf("create idx_assets_project: %w", err)
	}
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_assets_project_normalized ON assets(project_id, normalized_value);`)
	if err != nil {
		return fmt.Errorf("create idx_assets_project_normalized: %w", err)
	}

	return nil
}
