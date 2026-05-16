package workflow

import (
	"os"
	"sync"
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// setupEmitterTest spins up an in-process SQLite DB with a real
// pipeline_run + pipeline_run_stages schema. We use the project's
// db.Open helper (rather than raw sql.Open) so migrations match what
// production ships, including the CHECK constraint on stage status.
func setupEmitterTest(t *testing.T) (*db.Queries, string, func()) {
	t.Helper()
	dataDir, err := os.MkdirTemp("", "anchor-emitter-*")
	if err != nil {
		t.Fatalf("temp dir: %v", err)
	}
	sqlDB, err := db.Open(dataDir)
	if err != nil {
		os.RemoveAll(dataDir)
		t.Fatalf("open db: %v", err)
	}
	queries := db.New(sqlDB)

	// Need a parent pipeline_run row because pipeline_run_stages.run_id has
	// an ON DELETE CASCADE FK pointing at it.
	projectID := util.GenerateID()
	runID := util.GenerateID()
	if err := queries.CreateProject(&models.Project{
		ID:        projectID,
		Name:      "emitter-test",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("create project: %v", err)
	}
	if err := queries.CreatePipelineRun(&models.PipelineRun{
		ID:        runID,
		ProjectID: projectID,
		Status:    "running",
		StartedAt: time.Now().UTC(),
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("create pipeline_run: %v", err)
	}

	cleanup := func() {
		sqlDB.Close()
		os.RemoveAll(dataDir)
	}
	return queries, runID, cleanup
}

func TestStageEmitter_EmptyRunID_AllNoOp(t *testing.T) {
	queries, _, cleanup := setupEmitterTest(t)
	defer cleanup()

	var calls int
	cb := func(string, StageID, string, string) { calls++ }
	e := NewStageEmitter(queries, "", cb)

	e.Set(StageHTTPX)
	e.Complete(StageHTTPX)
	e.Fail(StageHTTPX, "boom")

	if calls != 0 {
		t.Fatalf("empty-runID emitter should not invoke callback, got %d calls", calls)
	}
}

func TestStageEmitter_NilCallback_NoPanic(t *testing.T) {
	queries, runID, cleanup := setupEmitterTest(t)
	defer cleanup()

	e := NewStageEmitter(queries, runID, nil)
	// Must not panic. DB writes should still happen.
	e.Set(StageHTTPX)
	e.Complete(StageHTTPX)

	row, err := queries.GetPipelineRunStage(runID, string(StageHTTPX))
	if err != nil {
		t.Fatalf("get stage: %v", err)
	}
	if row == nil {
		t.Fatalf("stage row not persisted")
	}
	if row.Status != models.StageStatusCompleted {
		t.Fatalf("status = %q, want completed", row.Status)
	}
}

func TestStageEmitter_SetIsIdempotent(t *testing.T) {
	queries, runID, cleanup := setupEmitterTest(t)
	defer cleanup()

	e := NewStageEmitter(queries, runID, nil)
	e.Set(StageHTTPX)
	first, _ := queries.GetPipelineRunStage(runID, string(StageHTTPX))

	// Second Set should reuse the existing row (no INSERT, no PK collision).
	e.Set(StageHTTPX)
	second, _ := queries.GetPipelineRunStage(runID, string(StageHTTPX))

	if first == nil || second == nil {
		t.Fatalf("stage row missing after Set")
	}
	if first.ID != second.ID {
		t.Fatalf("Set created a second row: first=%s second=%s", first.ID, second.ID)
	}
	if second.Status != models.StageStatusRunning {
		t.Fatalf("status after second Set = %q, want running", second.Status)
	}
}

func TestStageEmitter_FailWithoutSet_PersistsRow(t *testing.T) {
	// Fix 3 path: ffuf is enabled but no dictionary configured, so the
	// orchestrator calls Fail directly without ever calling Set. A user
	// reloading the run-detail page must still see the failure persisted.
	queries, runID, cleanup := setupEmitterTest(t)
	defer cleanup()

	var captured struct {
		mu     sync.Mutex
		status string
		err    string
	}
	cb := func(_ string, _ StageID, status, errMsg string) {
		captured.mu.Lock()
		defer captured.mu.Unlock()
		captured.status = status
		captured.err = errMsg
	}

	e := NewStageEmitter(queries, runID, cb)
	e.Fail(StageFfuf, "ffuf enabled but no dictionary configured")

	row, err := queries.GetPipelineRunStage(runID, string(StageFfuf))
	if err != nil {
		t.Fatalf("get stage: %v", err)
	}
	if row == nil {
		t.Fatalf("Fail-without-Set should persist a failed row, got nil")
	}
	if row.Status != models.StageStatusFailed {
		t.Fatalf("status = %q, want failed", row.Status)
	}
	if row.Error != "ffuf enabled but no dictionary configured" {
		t.Fatalf("error = %q, want the reason string", row.Error)
	}
	if row.CompletedAt == nil {
		t.Fatalf("CompletedAt should be set for a directly-failed stage")
	}

	captured.mu.Lock()
	defer captured.mu.Unlock()
	if captured.status != "failed" {
		t.Fatalf("callback status = %q, want failed", captured.status)
	}
	if captured.err == "" {
		t.Fatalf("callback errMsg empty, want reason")
	}
}

func TestStageEmitter_SetThenComplete_FullLifecycle(t *testing.T) {
	queries, runID, cleanup := setupEmitterTest(t)
	defer cleanup()

	var events []string
	cb := func(_ string, _ StageID, status, _ string) {
		events = append(events, status)
	}

	e := NewStageEmitter(queries, runID, cb)
	e.Set(StageFfuf)
	e.Complete(StageFfuf)

	if got, want := events, []string{"running", "completed"}; !sliceEq(got, want) {
		t.Fatalf("events = %v, want %v", got, want)
	}

	row, err := queries.GetPipelineRunStage(runID, string(StageFfuf))
	if err != nil || row == nil {
		t.Fatalf("get stage: %v / row=%v", err, row)
	}
	if row.Status != models.StageStatusCompleted {
		t.Fatalf("status = %q, want completed", row.Status)
	}
	if row.CompletedAt == nil {
		t.Fatalf("CompletedAt missing after Complete")
	}
}

func TestStageEmitter_SetThenFail_ErrorRecorded(t *testing.T) {
	queries, runID, cleanup := setupEmitterTest(t)
	defer cleanup()

	e := NewStageEmitter(queries, runID, nil)
	e.Set(StageFfuf)
	e.Fail(StageFfuf, "3/3 targets failed")

	row, err := queries.GetPipelineRunStage(runID, string(StageFfuf))
	if err != nil || row == nil {
		t.Fatalf("get stage: %v / row=%v", err, row)
	}
	if row.Status != models.StageStatusFailed {
		t.Fatalf("status = %q, want failed", row.Status)
	}
	if row.Error != "3/3 targets failed" {
		t.Fatalf("error = %q, want the reason string", row.Error)
	}
}

func sliceEq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
