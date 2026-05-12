package db

import "database/sql"

// migrateV19 extends finding_templates with provenance tracking so the
// templates can be seeded from the repository's docs/templates/vuln-templates.json
// while user edits in the UI are preserved across image upgrades.
//
//   is_builtin       — 1 when the row was seeded from the repo JSON; 0 when created in UI.
//   user_modified    — 1 when a builtin row was edited locally (locks it from auto-overwrite).
//   builtin_payload  — JSON of the latest upstream version of a builtin row, used by UI
//                      to show "upstream has a newer version" and to power the
//                      "accept upstream" action.
func migrateV19(db *sql.DB) error {
	stmts := []string{
		`ALTER TABLE finding_templates ADD COLUMN is_builtin INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE finding_templates ADD COLUMN user_modified INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE finding_templates ADD COLUMN builtin_payload TEXT NOT NULL DEFAULT ''`,
		`CREATE INDEX IF NOT EXISTS idx_finding_templates_builtin ON finding_templates(is_builtin)`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}
