package db

import (
	"database/sql"
	"fmt"
)

// migrateV46 retires the dedicated internal scan scenario (P1-1).
//   - Tool templates seeded with profile_type 'internal' are migrated to
//     'external' and annotated so they remain discoverable but are clearly
//     retrofitted for the internet scan mode.
//   - Historical pipeline_runs with mode='internal' are kept as immutable audit
//     rows (never silently widened); callers that enumerate runs may render them
//     with a "legacy" label.
//
// Project pipeline_config strings are intentionally NOT rewritten here: a saved
// pipeline config stays byte-for-byte intact so the API layer can decide whether
// to reject or explicitly migrate it (see internal/api/pipeline_handlers.go).
func migrateV46(db *sql.DB) error {
	_, err := db.Exec(`
		UPDATE tool_templates
		SET profile_type = 'external',
		    description = COALESCE(NULLIF(description, ''), '') ||
		                  CASE
		                    WHEN COALESCE(NULLIF(description, ''), '') = '' THEN '[migrated from internal mode (P1-1)]'
		                    ELSE ' [migrated from internal mode (P1-1)]'
		                  END
		WHERE profile_type = 'internal';
	`)
	if err != nil {
		return fmt.Errorf("migrate internal tool templates to external: %w", err)
	}
	return nil
}
