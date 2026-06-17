package recovery

import (
	"database/sql"
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/models"
)

func TestRecoverOrphanRuns_MarksRunAndWorkTerminal(t *testing.T) {
	sqlDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	sqlDB.SetMaxOpenConns(1)
	if err := db.Migrate(sqlDB); err != nil {
		t.Fatal(err)
	}
	q := db.New(sqlDB)

	now := time.Now().UTC()
	if err := q.CreateProject(&models.Project{ID: "p1", Name: "orphan", CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	runID := "run-orphan-1"
	if err := q.CreatePipelineRun(&models.PipelineRun{
		ID: runID, ProjectID: "p1", Status: "running", EngineState: "running", CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	pending := &models.ScanWorkItem{
		ID: "w-pending", RunID: runID, ProjectID: "p1", AssetID: "a1",
		Action: "DNS_RESOLVE", Status: models.WorkStatusPending, CreatedAt: now,
	}
	running := &models.ScanWorkItem{
		ID: "w-running", RunID: runID, ProjectID: "p1", AssetID: "a2",
		Action: "HTTPX_FINGERPRINT", Status: models.WorkStatusRunning, CreatedAt: now,
	}
	for _, w := range []*models.ScanWorkItem{pending, running} {
		if err := q.CreateScanWorkItem(w); err != nil {
			t.Fatal(err)
		}
	}

	n, err := RecoverOrphanRuns(q)
	if err != nil {
		t.Fatalf("RecoverOrphanRuns: %v", err)
	}
	if n != 1 {
		t.Fatalf("recovered = %d, want 1", n)
	}

	run, err := q.GetPipelineRun(runID)
	if err != nil || run == nil {
		t.Fatalf("GetPipelineRun: %v", err)
	}
	if run.Status != "failed" {
		t.Fatalf("run status = %q, want failed", run.Status)
	}
	if run.EngineState != "stopped" {
		t.Fatalf("engine_state = %q, want stopped", run.EngineState)
	}

	gotPending, _ := q.GetScanWorkItem("w-pending")
	if gotPending == nil || gotPending.Status != models.WorkStatusSkipped {
		t.Fatalf("pending work status = %v", gotPending)
	}
	gotRunning, _ := q.GetScanWorkItem("w-running")
	if gotRunning == nil || gotRunning.Status != models.WorkStatusFailed {
		t.Fatalf("running work status = %v", gotRunning)
	}
}
