package db

import "database/sql"

// migrateV43 adds TLS certificate columns to web_endpoints for certificate expiry monitoring.
func migrateV43(db *sql.DB) error {
	_, err := db.Exec(`
		ALTER TABLE web_endpoints ADD COLUMN tls_issuer TEXT NOT NULL DEFAULT '';
		ALTER TABLE web_endpoints ADD COLUMN tls_subject TEXT NOT NULL DEFAULT '';
		ALTER TABLE web_endpoints ADD COLUMN tls_not_before DATETIME;
		ALTER TABLE web_endpoints ADD COLUMN tls_not_after DATETIME;
	`)
	return err
}
