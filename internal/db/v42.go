package db

import "database/sql"

// migrateV42 adds the notification_channels table for external alert delivery channels.
func migrateV42(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS notification_channels (
			id              TEXT PRIMARY KEY,
			project_id      TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
			name            TEXT NOT NULL DEFAULT '',
			channel_type    TEXT NOT NULL DEFAULT 'webhook',
			url             TEXT NOT NULL DEFAULT '',
			enabled         INTEGER NOT NULL DEFAULT 1,
			created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_notification_channels_project
			ON notification_channels(project_id);
	`)
	return err
}
