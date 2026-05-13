package db

import (
	"database/sql"
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
	_ "github.com/mattn/go-sqlite3"
)

// TestMigrateV12_FixesScanTasksRunIDForeignKey verifies that scan_tasks.run_id
// no longer has a FK constraint after v12+v21 migrations. The column is shared
// by two code paths (runs.id and pipeline_runs.id), so a FK is not viable.
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

	// 1. Verify scan_tasks.run_id has no FK constraint.
	var refTable string
	row := rawDB.QueryRow(`SELECT "table" FROM pragma_foreign_key_list('scan_tasks') WHERE "from" = 'run_id'`)
	if err := row.Scan(&refTable); err != sql.ErrNoRows {
		t.Fatalf("expected no FK on run_id, got refTable=%q err=%v", refTable, err)
	}

	// 2. Both pipeline_runs.id and runs.id are accepted as run_id.
	q := New(rawDB)
	now := time.Now().UTC()

	project := &models.Project{
		ID:        "proj-v12-test",
		Name:      "v12 test",
		RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := q.CreateProject(project); err != nil {
		t.Fatalf("create project: %v", err)
	}

	// Insert via pipeline_runs path
	if err := q.CreatePipelineRun(&models.PipelineRun{
		ID: "prun-v12", ProjectID: project.ID,
		Status: "running", StartedAt: now, CreatedAt: now,
	}); err != nil {
		t.Fatalf("create pipeline_run: %v", err)
	}
	pRunID := "prun-v12"
	if err := q.CreateScanTask(&models.ScanTask{
		ID: "task-prun", ProjectID: project.ID, RunID: &pRunID,
		Tool: "naabu", Status: models.TaskCreated, CreatedAt: now,
	}); err != nil {
		t.Fatalf("create scan_task with pipeline_runs.run_id: %v", err)
	}

	// Insert via old runs path
	runID := "run-v12-test"
	if err := q.CreateRun(&models.Run{
		ID: runID, ProjectID: project.ID,
		Status: models.RunRunning, CreatedAt: now,
	}); err != nil {
		t.Fatalf("create run: %v", err)
	}
	if err := q.CreateScanTask(&models.ScanTask{
		ID: "task-run", ProjectID: project.ID, RunID: &runID,
		Tool: "naabu", Status: models.TaskCreated, CreatedAt: now,
	}); err != nil {
		t.Fatalf("create scan_task with runs.run_id: %v", err)
	}
}
