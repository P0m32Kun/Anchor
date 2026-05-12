package db

import (
	"database/sql"
	"fmt"
	"strings"
)

// migrateV15 promotes nmap -sV product/version from the metadata JSON blob to
// first-class columns on service_fingerprints. Before this migration the
// fingerprint pipeline wrote `metadata={"product":"nginx","version":"1.27.5"}`
// but the service-ports aggregation layer only read `service`, leaving the
// product/version fields permanently empty in API responses (observed during
// run id-1778560237476298804-140).
//
// Per design decision (plan-eng-review 2026-05-12): existing rows are NOT
// backfilled from metadata. The new columns default to '' on old rows; new
// scans populate them directly. Historical metadata is left intact for
// forensic value and possible later reuse.
func migrateV15(db *sql.DB) error {
	for _, col := range []string{"product", "version"} {
		exists, err := hasServiceFingerprintColumn(db, col)
		if err != nil {
			return fmt.Errorf("check service_fingerprints.%s: %w", col, err)
		}
		if exists {
			continue
		}
		stmt := fmt.Sprintf(`ALTER TABLE service_fingerprints ADD COLUMN %s TEXT NOT NULL DEFAULT ''`, col)
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("add service_fingerprints.%s: %w", col, err)
		}
	}
	return nil
}

func hasServiceFingerprintColumn(db *sql.DB, name string) (bool, error) {
	rows, err := db.Query(`PRAGMA table_info(service_fingerprints)`)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var col, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &col, &ctype, &notnull, &dflt, &pk); err != nil {
			return false, err
		}
		if strings.EqualFold(col, name) {
			return true, nil
		}
	}
	return false, rows.Err()
}
