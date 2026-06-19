package db

import (
	"os"
	"path/filepath"
	"testing"
)

// --- Open ---

func TestOpen_CreatesDirAndDB(t *testing.T) {
	tmpDir := filepath.Join(t.TempDir(), "subdir", "anchor")

	db, err := Open(tmpDir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	// Verify the directory was created
	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		t.Error("expected data dir to be created")
	}

	// Verify the DB file exists
	dbPath := filepath.Join(tmpDir, "anchor.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("expected anchor.db to exist")
	}

	// Verify migration ran (check user_version)
	var version int
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		t.Fatalf("read user_version: %v", err)
	}
	if version != 44 {
		t.Errorf("user_version = %d, want 44", version)
	}
}

// --- Migrate idempotent ---

func TestMigrate_Idempotent(t *testing.T) {
	rawDB := openTestDB(t)

	// Migrate again — should be a no-op since version is already 44
	if err := Migrate(rawDB); err != nil {
		t.Fatalf("second Migrate: %v", err)
	}

	var version int
	if err := rawDB.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		t.Fatalf("read user_version: %v", err)
	}
	if version != 44 {
		t.Errorf("user_version = %d, want 44", version)
	}
}

// --- ensureProjectsColumns ---

func TestEnsureProjectsColumns_AlreadyExists(t *testing.T) {
	rawDB := openTestDB(t)

	// rate_limit column should already exist from migration
	if err := ensureProjectsColumns(rawDB); err != nil {
		t.Fatalf("ensureProjectsColumns: %v", err)
	}

	// Verify the column exists
	var count int
	err := rawDB.QueryRow(
		"SELECT COUNT(*) FROM pragma_table_info('projects') WHERE name = 'rate_limit'",
	).Scan(&count)
	if err != nil {
		t.Fatalf("check column: %v", err)
	}
	if count != 1 {
		t.Errorf("rate_limit column count = %d, want 1", count)
	}
}
