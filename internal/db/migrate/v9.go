package db

import (
	"database/sql"
	"fmt"
	"log"
	"time"
)

func migrateV09(db *sql.DB) error {
	// 1. Create engine_credentials table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS engine_credentials (
			id TEXT PRIMARY KEY,
			engine TEXT NOT NULL UNIQUE,
			api_key TEXT NOT NULL,
			email TEXT,
			extra TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		return fmt.Errorf("create engine_credentials: %w", err)
	}

	// 2. Migrate existing FOFA credentials from projects table
	var fofaEmail, fofaAPIKey string
	err = db.QueryRow(`SELECT fofa_email, fofa_api_key FROM projects WHERE fofa_email IS NOT NULL AND fofa_email != '' AND fofa_api_key IS NOT NULL AND fofa_api_key != '' LIMIT 1`).Scan(&fofaEmail, &fofaAPIKey)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("migrate fofa credentials: %w", err)
	}
	if err == nil && fofaEmail != "" && fofaAPIKey != "" {
		now := time.Now().UTC()
		_, err = db.Exec(`
			INSERT INTO engine_credentials (id, engine, api_key, email, created_at, updated_at)
			VALUES (?, 'fofa', ?, ?, ?, ?)
			ON CONFLICT(engine) DO UPDATE SET
				api_key = excluded.api_key,
				email = excluded.email,
				updated_at = excluded.updated_at;
		`, fmt.Sprintf("cred-%d", now.UnixNano()), fofaAPIKey, fofaEmail, now, now)
		if err != nil {
			return fmt.Errorf("insert fofa credential: %w", err)
		}
		log.Printf("migration v9: migrated FOFA credentials from projects to engine_credentials")
	}

	return nil
}
