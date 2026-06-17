package db

import (
	"database/sql"
	"log"
)

func migrateV38(db *sql.DB) error {
	log.Println("[migrate] v38: adding worker metrics + asset_state table...")

	// Add system resource metrics columns to worker_nodes
	workerCols := []string{
		"ALTER TABLE worker_nodes ADD COLUMN cpu_percent REAL",
		"ALTER TABLE worker_nodes ADD COLUMN mem_percent REAL",
		"ALTER TABLE worker_nodes ADD COLUMN disk_percent REAL",
		"ALTER TABLE worker_nodes ADD COLUMN metrics_updated_at DATETIME",
	}
	for _, col := range workerCols {
		if _, err := db.Exec(col); err != nil {
			if !isDuplicateColumnError(err) {
				return err
			}
		}
	}

	// Create asset_state table for caching asset attrs
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS asset_state (
		asset_id TEXT PRIMARY KEY REFERENCES assets(id) ON DELETE CASCADE,
		state TEXT NOT NULL DEFAULT '{}',
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		return err
	}

	log.Println("[migrate] v38: done")
	return nil
}
