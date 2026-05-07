package db

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
	_ "github.com/mattn/go-sqlite3"
)

// TestMigrateV12_FixesScanTasksRunIDForeignKey reproduces the bug that caused
// every internal scan to silently complete with no results: scan_tasks.run_id
// was declared REFERENCES runs(id) by an early v0.2 migration, but the
// application records pipeline run IDs from pipeline_runs (added in v6).
// Inserting a scan_task with a run_id from pipeline_runs failed with
// FOREIGN KEY constraint failed before the v12 fix.
func TestMigrateV12_FixesScanTasksRunIDForeignKey(t *testing.T) {
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

	// 1. Verify scan_tasks.run_id now references pipeline_runs(id).
	var refTable string
	row := rawDB.QueryRow(`SELECT "table" FROM pragma_foreign_key_list('scan_tasks') WHERE "from" = 'run_id'`)
	if err := row.Scan(&refTable); err != nil {
		t.Fatalf("scan FK list: %v", err)
	}
	if refTable != "pipeline_runs" {
		t.Fatalf("scan_tasks.run_id should reference pipeline_runs after v12, got %q", refTable)
	}

	// 2. End-to-end: insert a project, a pipeline_runs row, then a scan_task
	// pointing at the pipeline_runs.id. This is the exact INSERT that failed
	// with FOREIGN KEY constraint failed before the fix.
	q := New(rawDB)
	now := time.Now().UTC()

	project := &models.Project{
		ID:        "proj-v12-test",
		Name:      "v12 test",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := q.CreateProject(project); err != nil {
		t.Fatalf("create project: %v", err)
	}

	if err := q.CreatePipelineRun(&models.PipelineRun{
		ID:        "run-v12-test",
		ProjectID: project.ID,
		Mode:      "internal",
		Status:    "running",
		StartedAt: now,
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("create pipeline run: %v", err)
	}

	runID := "run-v12-test"
	task := &models.ScanTask{
		ID:              "task-v12-test",
		ProjectID:       project.ID,
		RunID:           &runID,
		Tool:            "naabu",
		CommandTemplate: "naabu -host 127.0.0.1",
		Status:          models.TaskCreated,
		CreatedAt:       now,
	}
	if err := q.CreateScanTask(task); err != nil {
		t.Fatalf("create scan task with pipeline_runs.run_id should succeed after v12, got: %v", err)
	}

	// 3. Verify the FK is enforced (run_id pointing at a non-existent
	// pipeline_runs.id is rejected).
	bogusRun := "no-such-run"
	bad := &models.ScanTask{
		ID:              "task-v12-bad",
		ProjectID:       project.ID,
		RunID:           &bogusRun,
		Tool:            "naabu",
		CommandTemplate: "naabu -host 127.0.0.1",
		Status:          models.TaskCreated,
		CreatedAt:       now,
	}
	err = q.CreateScanTask(bad)
	if err == nil {
		t.Fatalf("expected FOREIGN KEY constraint failed for non-existent run_id, got nil")
	}
	if !strings.Contains(err.Error(), "FOREIGN KEY constraint failed") {
		t.Fatalf("expected foreign-key error, got: %v", err)
	}
}
