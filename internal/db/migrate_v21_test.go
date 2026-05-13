package db

import (
	"database/sql"
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
	_ "github.com/mattn/go-sqlite3"
)

func TestMigrateV21_RevertsRunIDFK(t *testing.T) {
	rawDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer rawDB.Close()
	rawDB.SetMaxOpenConns(1)

	if _, err := rawDB.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		t.Fatalf("enable fk: %v", err)
	}

	if err := Migrate(rawDB); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// 1. Verify run_id has no FK constraint (two systems share the column).
	var refTable string
	row := rawDB.QueryRow(`SELECT "table" FROM pragma_foreign_key_list('scan_tasks') WHERE "from" = 'run_id'`)
	if err := row.Scan(&refTable); err != sql.ErrNoRows {
		t.Fatalf("expected no FK on run_id, got refTable=%q err=%v", refTable, err)
	}

	// 2. Verify scan_task with pipeline_runs.id succeeds (no FK block).
	q := New(rawDB)
	now := time.Now().UTC()

	if err := q.CreateProject(&models.Project{
		ID: "proj-1", Name: "test", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create project: %v", err)
	}

	if err := q.CreatePipelineRun(&models.PipelineRun{
		ID: "prun-1", ProjectID: "proj-1",
		Status: "running", StartedAt: now, CreatedAt: now,
	}); err != nil {
		t.Fatalf("create pipeline_run: %v", err)
	}

	pRunID := "prun-1"
	if err := q.CreateScanTask(&models.ScanTask{
		ID: "task-1", ProjectID: "proj-1", RunID: &pRunID,
		Tool: "nuclei", Status: models.TaskRunning, CreatedAt: now,
	}); err != nil {
		t.Fatalf("create scan_task with pipeline_runs run_id: %v", err)
	}

	// 3. Verify scan_task with arbitrary run_id also succeeds (no FK).
	bogus := "no-such-run"
	if err := q.CreateScanTask(&models.ScanTask{
		ID: "task-bad", ProjectID: "proj-1", RunID: &bogus,
		Tool: "nuclei", Status: models.TaskRunning, CreatedAt: now,
	}); err != nil {
		t.Fatalf("create scan_task with arbitrary run_id should succeed (no FK): %v", err)
	}

	// 4. Verify ListScanTasksByRun returns tasks for valid run.
	tasks, err := q.ListScanTasksByRun("prun-1")
	if err != nil {
		t.Fatalf("ListScanTasksByRun: %v", err)
	}
	if len(tasks) != 1 {
		t.Errorf("len = %d, want 1", len(tasks))
	}

	// 5. Verify scan_task with nil run_id succeeds.
	if err := q.CreateScanTask(&models.ScanTask{
		ID: "task-nil", ProjectID: "proj-1", RunID: nil,
		Tool: "nuclei", Status: models.TaskRunning, CreatedAt: now,
	}); err != nil {
		t.Fatalf("create scan_task with nil run_id: %v", err)
	}
}

func TestMigrateV21_Idempotent(t *testing.T) {
	// Running migrate twice should not error.
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
