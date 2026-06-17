package db

import "database/sql"

// migrateV35 adds scope_boundary_mode to projects and scope_status to findings.
//   - projects.scope_boundary_mode: "off" (default) or "strict"
//   - findings.scope_status: "in_scope" (default), "out_of_scope"
func migrateV35(db *sql.DB) error {
	if _, err := db.Exec(`ALTER TABLE projects ADD COLUMN scope_boundary_mode TEXT NOT NULL DEFAULT 'off'`); err != nil && !isDuplicateColumnError(err) {
		return err
	}
	if _, err := db.Exec(`ALTER TABLE findings ADD COLUMN scope_status TEXT NOT NULL DEFAULT 'in_scope'`); err != nil && !isDuplicateColumnError(err) {
		return err
	}
	return nil
}
