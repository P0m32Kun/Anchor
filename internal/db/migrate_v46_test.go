package db

import (
	"database/sql"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// TestMigrateV46_InternalToolTemplatesBecomeExternal verifies that the P1-1
// compatibility exit migrates any tool template seeded with profile_type
// 'internal' to the internet profile and annotates its description, without
// widening a saved project's pipeline_config.
func TestMigrateV46_InternalToolTemplatesBecomeExternal(t *testing.T) {
	rawDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer rawDB.Close()
	rawDB.SetMaxOpenConns(1)

	if err := Migrate(rawDB); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// v3 seeds an 'internal-slow' template with profile_type 'internal'. After
	// migrateV46 it must be external and annotated.
	var gotProfile, gotDesc string
	err = rawDB.QueryRow(
		`SELECT profile_type, description FROM tool_templates WHERE id = 'internal-slow'`,
	).Scan(&gotProfile, &gotDesc)
	if err != nil {
		t.Fatalf("query internal-slow template: %v", err)
	}
	if gotProfile != "external" {
		t.Errorf("profile_type = %q, want 'external' after P1-1 migration", gotProfile)
	}
	if !strings.Contains(gotDesc, "migrated from internal") {
		t.Errorf("description = %q, want annotation mentioning internal-mode migration", gotDesc)
	}

	// No template may still carry the retired internal profile.
	var count int
	if err := rawDB.QueryRow(`SELECT COUNT(*) FROM tool_templates WHERE profile_type = 'internal'`).Scan(&count); err != nil {
		t.Fatalf("count internal templates: %v", err)
	}
	if count != 0 {
		t.Errorf("%d template(s) still have profile_type='internal'", count)
	}
}

// TestMigrateV46_Idempotent ensures migrateV46 (and the whole chain) can run twice.
func TestMigrateV46_Idempotent(t *testing.T) {
	rawDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer rawDB.Close()
	rawDB.SetMaxOpenConns(1)

	if err := Migrate(rawDB); err != nil {
		t.Fatalf("first migrate: %v", err)
	}
	if err := Migrate(rawDB); err != nil {
		t.Fatalf("second migrate: %v", err)
	}
}

// TestMigrateV46_PreservesSavedConfigAndLegacyRuns verifies the P1-1 invariant
// that migration never silently widens scope: a saved project pipeline_config and
// a historical pipeline_runs row with mode='internal' are left untouched.
func TestMigrateV46_PreservesSavedConfigAndLegacyRuns(t *testing.T) {
	rawDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer rawDB.Close()
	rawDB.SetMaxOpenConns(1)

	if err := Migrate(rawDB); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	legacyJSON := `{"enable_subfinder":false,"enable_cdn_filter":false,"naabu_rate":1000,"port_range":"high-risk"}`
	if _, err := rawDB.Exec(
		`INSERT INTO projects (id, name, pipeline_config, created_at, updated_at) VALUES ('p-saved','legacy-saved', ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		legacyJSON,
	); err != nil {
		t.Fatalf("insert legacy project: %v", err)
	}
	if _, err := rawDB.Exec(
		`INSERT INTO pipeline_runs (id, project_id, mode, status, created_at) VALUES ('r-internal','p-saved','internal','completed',CURRENT_TIMESTAMP)`,
	); err != nil {
		t.Fatalf("insert legacy internal run: %v", err)
	}

	// Re-run only the P1-1 migration and confirm it preserves both rows.
	if err := migrateV46(rawDB); err != nil {
		t.Fatalf("migrateV46: %v", err)
	}

	var preserved string
	if err := rawDB.QueryRow(`SELECT pipeline_config FROM projects WHERE id='p-saved'`).Scan(&preserved); err != nil {
		t.Fatalf("read saved config: %v", err)
	}
	if preserved != legacyJSON {
		t.Errorf("project pipeline_config was altered by migration:\n got %q\nwant %q", preserved, legacyJSON)
	}

	var mode string
	if err := rawDB.QueryRow(`SELECT mode FROM pipeline_runs WHERE id='r-internal'`).Scan(&mode); err != nil {
		t.Fatalf("read legacy run mode: %v", err)
	}
	if mode != "internal" {
		t.Errorf("historical run mode = %q, want 'internal' preserved as audit row", mode)
	}
}
