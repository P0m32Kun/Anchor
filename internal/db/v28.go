package db

import "database/sql"

// migrateV28 adds task_id to scan_work_items for work→scan_task linkage.
func migrateV28(db *sql.DB) error {
	_, err := db.Exec(`ALTER TABLE scan_work_items ADD COLUMN task_id TEXT REFERENCES scan_tasks(id) ON DELETE SET NULL`)
	return err
}
