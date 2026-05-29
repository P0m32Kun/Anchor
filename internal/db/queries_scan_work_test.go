package db

import (
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// setupScanWorkTest creates a project + pipeline run and returns the queries
// handle together with the run ID.
func setupScanWorkTest(t *testing.T) (*Queries, string) {
	t.Helper()
	q := New(openTestDB(t))
	now := time.Now().UTC()
	if err := q.CreateProject(&models.Project{
		ID: "proj-1", Name: "test", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create project: %v", err)
	}
	run := &models.PipelineRun{
		ID:          "run-1",
		ProjectID:   "proj-1",
		Mode:        "standard",
		Status:      "running",
		EngineState: "running",
		StartedAt:   now,
		CreatedAt:   now,
	}
	if err := q.CreatePipelineRun(run); err != nil {
		t.Fatalf("create pipeline run: %v", err)
	}
	return q, "run-1"
}

func TestScanWorkItem_CreateAndGet(t *testing.T) {
	q, runID := setupScanWorkTest(t)
	now := time.Now().UTC()

	w := &models.ScanWorkItem{
		ID:        util.GenerateID(),
		RunID:     runID,
		ProjectID: "proj-1",
		AssetID:   "asset-1",
		Action:    "HTTPX_FINGERPRINT",
		Status:    models.WorkStatusPending,
		CreatedAt: now,
	}
	if err := q.CreateScanWorkItem(w); err != nil {
		t.Fatalf("CreateScanWorkItem: %v", err)
	}

	got, err := q.GetScanWorkItem(w.ID)
	if err != nil {
		t.Fatalf("GetScanWorkItem: %v", err)
	}
	if got == nil {
		t.Fatal("GetScanWorkItem returned nil")
	}
	if got.Action != "HTTPX_FINGERPRINT" {
		t.Errorf("action = %q, want HTTPX_FINGERPRINT", got.Action)
	}
	if got.Status != models.WorkStatusPending {
		t.Errorf("status = %q, want pending", got.Status)
	}
}

func TestScanWorkItem_UniqueRunAssetAction(t *testing.T) {
	q, runID := setupScanWorkTest(t)
	now := time.Now().UTC()

	w := &models.ScanWorkItem{
		ID:        util.GenerateID(),
		RunID:     runID,
		ProjectID: "proj-1",
		AssetID:   "asset-1",
		Action:    "HTTPX_FINGERPRINT",
		Status:    models.WorkStatusPending,
		CreatedAt: now,
	}
	if err := q.CreateScanWorkItem(w); err != nil {
		t.Fatalf("CreateScanWorkItem: %v", err)
	}

	// Same (run_id, asset_id, action) should violate UNIQUE constraint.
	w2 := &models.ScanWorkItem{
		ID:        util.GenerateID(),
		RunID:     runID,
		ProjectID: "proj-1",
		AssetID:   "asset-1",
		Action:    "HTTPX_FINGERPRINT",
		Status:    models.WorkStatusPending,
		CreatedAt: now,
	}
	err := q.CreateScanWorkItem(w2)
	if err == nil {
		t.Fatal("expected unique constraint violation, got nil")
	}
}

func TestScanWorkItem_GetByRunAssetAction(t *testing.T) {
	q, runID := setupScanWorkTest(t)
	now := time.Now().UTC()

	w := &models.ScanWorkItem{
		ID:        util.GenerateID(),
		RunID:     runID,
		ProjectID: "proj-1",
		AssetID:   "asset-1",
		Action:    "NUCLEI_SCAN",
		Status:    models.WorkStatusPending,
		CreatedAt: now,
	}
	if err := q.CreateScanWorkItem(w); err != nil {
		t.Fatalf("CreateScanWorkItem: %v", err)
	}

	got, err := q.GetScanWorkItemByRunAssetAction(runID, "asset-1", "NUCLEI_SCAN")
	if err != nil {
		t.Fatalf("GetScanWorkItemByRunAssetAction: %v", err)
	}
	if got == nil {
		t.Fatal("GetScanWorkItemByRunAssetAction returned nil")
	}
	if got.ID != w.ID {
		t.Errorf("id = %q, want %q", got.ID, w.ID)
	}
}

func TestScanWorkItem_UpdateStatus(t *testing.T) {
	q, runID := setupScanWorkTest(t)
	now := time.Now().UTC()

	w := &models.ScanWorkItem{
		ID:        util.GenerateID(),
		RunID:     runID,
		ProjectID: "proj-1",
		AssetID:   "asset-1",
		Action:    "HTTPX_FINGERPRINT",
		Status:    models.WorkStatusPending,
		CreatedAt: now,
	}
	if err := q.CreateScanWorkItem(w); err != nil {
		t.Fatalf("CreateScanWorkItem: %v", err)
	}

	started := now.Add(time.Second)
	if err := q.UpdateScanWorkItemStatus(w.ID, models.WorkStatusRunning, &started, nil); err != nil {
		t.Fatalf("UpdateScanWorkItemStatus: %v", err)
	}

	got, _ := q.GetScanWorkItem(w.ID)
	if got.Status != models.WorkStatusRunning {
		t.Errorf("status = %q, want running", got.Status)
	}
	if got.StartedAt == nil {
		t.Error("started_at should be set")
	}

	completed := now.Add(5 * time.Second)
	if err := q.UpdateScanWorkItemStatus(w.ID, models.WorkStatusDone, &started, &completed); err != nil {
		t.Fatalf("UpdateScanWorkItemStatus to done: %v", err)
	}
	got, _ = q.GetScanWorkItem(w.ID)
	if got.Status != models.WorkStatusDone {
		t.Errorf("status = %q, want done", got.Status)
	}
	if got.CompletedAt == nil {
		t.Error("completed_at should be set")
	}
}

func TestScanWorkItem_UpdateSkip(t *testing.T) {
	q, runID := setupScanWorkTest(t)
	now := time.Now().UTC()

	w := &models.ScanWorkItem{
		ID:        util.GenerateID(),
		RunID:     runID,
		ProjectID: "proj-1",
		AssetID:   "asset-1",
		Action:    "PORT_SCAN",
		Status:    models.WorkStatusPending,
		CreatedAt: now,
	}
	if err := q.CreateScanWorkItem(w); err != nil {
		t.Fatalf("CreateScanWorkItem: %v", err)
	}

	completed := now.Add(time.Second)
	if err := q.UpdateScanWorkItemSkip(w.ID, models.WorkStatusSkipped, "cdn_host", &completed); err != nil {
		t.Fatalf("UpdateScanWorkItemSkip: %v", err)
	}

	got, _ := q.GetScanWorkItem(w.ID)
	if got.Status != models.WorkStatusSkipped {
		t.Errorf("status = %q, want skipped", got.Status)
	}
	if got.SkipReason != "cdn_host" {
		t.Errorf("skip_reason = %q, want cdn_host", got.SkipReason)
	}
}

func TestScanWorkItem_ListByRun(t *testing.T) {
	q, runID := setupScanWorkTest(t)
	now := time.Now().UTC()

	for _, action := range []string{"HTTPX_FINGERPRINT", "NUCLEI_SCAN", "PORT_SCAN"} {
		w := &models.ScanWorkItem{
			ID:        util.GenerateID(),
			RunID:     runID,
			ProjectID: "proj-1",
			AssetID:   "asset-1",
			Action:    action,
			Status:    models.WorkStatusPending,
			CreatedAt: now,
		}
		if err := q.CreateScanWorkItem(w); err != nil {
			t.Fatalf("CreateScanWorkItem(%s): %v", action, err)
		}
	}

	list, err := q.ListScanWorkItemsByRun(runID)
	if err != nil {
		t.Fatalf("ListScanWorkItemsByRun: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("len = %d, want 3", len(list))
	}
}

func TestScanWorkItem_ListByAsset(t *testing.T) {
	q, runID := setupScanWorkTest(t)
	now := time.Now().UTC()

	// Two works for asset-1, one for asset-2.
	for _, action := range []string{"HTTPX_FINGERPRINT", "NUCLEI_SCAN"} {
		w := &models.ScanWorkItem{
			ID: util.GenerateID(), RunID: runID, ProjectID: "proj-1",
			AssetID: "asset-1", Action: action, Status: models.WorkStatusPending, CreatedAt: now,
		}
		if err := q.CreateScanWorkItem(w); err != nil {
			t.Fatalf("CreateScanWorkItem(%s): %v", action, err)
		}
	}
	w := &models.ScanWorkItem{
		ID: util.GenerateID(), RunID: runID, ProjectID: "proj-1",
		AssetID: "asset-2", Action: "HTTPX_FINGERPRINT", Status: models.WorkStatusPending, CreatedAt: now,
	}
	if err := q.CreateScanWorkItem(w); err != nil {
		t.Fatalf("CreateScanWorkItem: %v", err)
	}

	list, err := q.ListScanWorkItemsByAsset(runID, "asset-1")
	if err != nil {
		t.Fatalf("ListScanWorkItemsByAsset: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("len = %d, want 2", len(list))
	}
}

func TestScanWorkItem_CountByStatus(t *testing.T) {
	q, runID := setupScanWorkTest(t)
	now := time.Now().UTC()

	statuses := []models.WorkStatus{
		models.WorkStatusPending, models.WorkStatusPending,
		models.WorkStatusRunning,
		models.WorkStatusDone, models.WorkStatusDone, models.WorkStatusDone,
		models.WorkStatusSkipped,
		models.WorkStatusFailed,
	}
	for i, st := range statuses {
		w := &models.ScanWorkItem{
			ID: util.GenerateID(), RunID: runID, ProjectID: "proj-1",
			AssetID: "asset-1", Action: string(rune('A' + i)), Status: st, CreatedAt: now,
		}
		if err := q.CreateScanWorkItem(w); err != nil {
			t.Fatalf("CreateScanWorkItem: %v", err)
		}
	}

	pending, running, done, skipped, failed, err := q.CountScanWorkItemsByStatus(runID)
	if err != nil {
		t.Fatalf("CountScanWorkItemsByStatus: %v", err)
	}
	if pending != 2 {
		t.Errorf("pending = %d, want 2", pending)
	}
	if running != 1 {
		t.Errorf("running = %d, want 1", running)
	}
	if done != 3 {
		t.Errorf("done = %d, want 3", done)
	}
	if skipped != 1 {
		t.Errorf("skipped = %d, want 1", skipped)
	}
	if failed != 1 {
		t.Errorf("failed = %d, want 1", failed)
	}
}

func TestScanWorkItem_AllTerminal(t *testing.T) {
	q, runID := setupScanWorkTest(t)
	now := time.Now().UTC()

	// Initially no work items — AllTerminal should return true.
	all, err := q.AllWorkItemsTerminal(runID)
	if err != nil {
		t.Fatalf("AllWorkItemsTerminal: %v", err)
	}
	if !all {
		t.Error("expected all terminal when no work items exist")
	}

	// Add a pending item.
	w := &models.ScanWorkItem{
		ID: util.GenerateID(), RunID: runID, ProjectID: "proj-1",
		AssetID: "asset-1", Action: "HTTPX_FINGERPRINT", Status: models.WorkStatusPending, CreatedAt: now,
	}
	if err := q.CreateScanWorkItem(w); err != nil {
		t.Fatalf("CreateScanWorkItem: %v", err)
	}

	all, err = q.AllWorkItemsTerminal(runID)
	if err != nil {
		t.Fatalf("AllWorkItemsTerminal: %v", err)
	}
	if all {
		t.Error("expected not all terminal with pending item")
	}

	// Mark done.
	completed := now.Add(time.Second)
	if err := q.UpdateScanWorkItemStatus(w.ID, models.WorkStatusDone, &now, &completed); err != nil {
		t.Fatalf("UpdateScanWorkItemStatus: %v", err)
	}

	all, err = q.AllWorkItemsTerminal(runID)
	if err != nil {
		t.Fatalf("AllWorkItemsTerminal: %v", err)
	}
	if !all {
		t.Error("expected all terminal after marking done")
	}
}

func TestScanRunMetrics_RoundTrip(t *testing.T) {
	q, runID := setupScanWorkTest(t)
	now := time.Now().UTC()

	// Add some work items with unique (asset_id, action) pairs.
	items := []struct {
		assetID string
		action  string
		status  models.WorkStatus
	}{
		{"asset-1", "HTTPX_FINGERPRINT", models.WorkStatusDone},
		{"asset-1", "NUCLEI_SCAN", models.WorkStatusDone},
		{"asset-2", "HTTPX_FINGERPRINT", models.WorkStatusPending},
	}
	for _, it := range items {
		w := &models.ScanWorkItem{
			ID: util.GenerateID(), RunID: runID, ProjectID: "proj-1",
			AssetID: it.assetID, Action: it.action, Status: it.status, CreatedAt: now,
		}
		if err := q.CreateScanWorkItem(w); err != nil {
			t.Fatalf("CreateScanWorkItem(%s/%s): %v", it.assetID, it.action, err)
		}
	}

	m, err := q.GetScanRunMetrics(runID)
	if err != nil {
		t.Fatalf("GetScanRunMetrics: %v", err)
	}
	if m == nil {
		t.Fatal("GetScanRunMetrics returned nil")
	}
	if m.EngineState != "running" {
		t.Errorf("engine_state = %q, want running", m.EngineState)
	}
	if m.WorksDone != 2 {
		t.Errorf("works_done = %d, want 2", m.WorksDone)
	}
	if m.WorksPending != 1 {
		t.Errorf("works_pending = %d, want 1", m.WorksPending)
	}
}

func TestPipelineRun_EngineState(t *testing.T) {
	q, runID := setupScanWorkTest(t)

	if err := q.UpdatePipelineRunEngineState(runID, "wind_down"); err != nil {
		t.Fatalf("UpdatePipelineRunEngineState: %v", err)
	}

	run, err := q.GetPipelineRun(runID)
	if err != nil {
		t.Fatalf("GetPipelineRun: %v", err)
	}
	if run.EngineState != "wind_down" {
		t.Errorf("engine_state = %q, want wind_down", run.EngineState)
	}
}

func TestPipelineRunStage_WorkCounts(t *testing.T) {
	q, runID := setupScanWorkTest(t)
	now := time.Now().UTC()

	stage := &models.PipelineRunStage{
		ID:        util.GenerateID(),
		RunID:     runID,
		Stage:     "httpx",
		Status:    models.StageStatusRunning,
		StartedAt: &now,
		CreatedAt: now,
	}
	if err := q.CreatePipelineRunStage(stage); err != nil {
		t.Fatalf("CreatePipelineRunStage: %v", err)
	}

	total, done, running, round := 10, 5, 2, 1
	if err := q.UpdatePipelineRunStageWorkCounts(runID, "httpx", total, done, running, round); err != nil {
		t.Fatalf("UpdatePipelineRunStageWorkCounts: %v", err)
	}

	got, err := q.GetPipelineRunStage(runID, "httpx")
	if err != nil {
		t.Fatalf("GetPipelineRunStage: %v", err)
	}
	if got.WorkTotal == nil || *got.WorkTotal != 10 {
		t.Errorf("work_total = %v, want 10", got.WorkTotal)
	}
	if got.WorkDone == nil || *got.WorkDone != 5 {
		t.Errorf("work_done = %v, want 5", got.WorkDone)
	}
	if got.WorkRunning == nil || *got.WorkRunning != 2 {
		t.Errorf("work_running = %v, want 2", got.WorkRunning)
	}
	if got.Round == nil || *got.Round != 1 {
		t.Errorf("round = %v, want 1", got.Round)
	}
}
