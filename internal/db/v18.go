package db

import "database/sql"

// migrateV18 creates the finding_templates table — a globally shared
// vulnerability knowledge base that overrides finding.title / severity /
// summary / remediation at report generation time when matched by
// (source_tool, match_key).
func migrateV18(db *sql.DB) error {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS finding_templates (
			id TEXT PRIMARY KEY,
			source_tool TEXT NOT NULL,
			match_key TEXT NOT NULL,
			title TEXT NOT NULL DEFAULT '',
			severity TEXT NOT NULL DEFAULT '',
			summary TEXT NOT NULL DEFAULT '',
			remediation TEXT NOT NULL DEFAULT '',
			enabled INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_finding_templates_match ON finding_templates(source_tool, match_key);
		CREATE INDEX IF NOT EXISTS idx_finding_templates_enabled ON finding_templates(enabled);
	`); err != nil {
		return err
	}
	return nil
}
