package db

import (
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// --- UpdateScanWorkItemTaskID ---

func TestUpdateScanWorkItemTaskID(t *testing.T) {
	q, runID := setupScanWorkTest(t)
	now := time.Now().UTC()

	// Create scan_task first (FK on task_id)
	if err := q.CreateScanTask(&models.ScanTask{
		ID: "task-for-work", ProjectID: "proj-1", RunID: &runID,
		Tool: "httpx", Status: models.TaskCreated, CreatedAt: now,
	}); err != nil {
		t.Fatalf("CreateScanTask: %v", err)
	}

	w := &models.ScanWorkItem{
		ID: util.GenerateID(), RunID: runID, ProjectID: "proj-1",
		AssetID: "asset-1", Action: "HTTPX_FINGERPRINT",
		Status: models.WorkStatusPending, CreatedAt: now,
	}
	if err := q.CreateScanWorkItem(w); err != nil {
		t.Fatalf("CreateScanWorkItem: %v", err)
	}

	if err := q.UpdateScanWorkItemTaskID(w.ID, "task-for-work"); err != nil {
		t.Fatalf("UpdateScanWorkItemTaskID: %v", err)
	}

	got, _ := q.GetScanWorkItem(w.ID)
	if got.TaskID == nil || *got.TaskID != "task-for-work" {
		t.Errorf("task_id = %v, want task-for-work", got.TaskID)
	}
}

// --- UpdateScanWorkItemError ---

func TestUpdateScanWorkItemError(t *testing.T) {
	q, runID := setupScanWorkTest(t)
	now := time.Now().UTC()

	w := &models.ScanWorkItem{
		ID: util.GenerateID(), RunID: runID, ProjectID: "proj-1",
		AssetID: "asset-1", Action: "NUCLEI_SCAN",
		Status: models.WorkStatusRunning, CreatedAt: now,
	}
	if err := q.CreateScanWorkItem(w); err != nil {
		t.Fatalf("CreateScanWorkItem: %v", err)
	}

	completed := now.Add(10 * time.Second)
	if err := q.UpdateScanWorkItemError(w.ID, models.WorkStatusFailed, "connection timeout", &completed); err != nil {
		t.Fatalf("UpdateScanWorkItemError: %v", err)
	}

	got, _ := q.GetScanWorkItem(w.ID)
	if got.Status != models.WorkStatusFailed {
		t.Errorf("status = %q, want failed", got.Status)
	}
	if got.Error != "connection timeout" {
		t.Errorf("error = %q, want connection timeout", got.Error)
	}
	if got.CompletedAt == nil {
		t.Error("expected completed_at to be set")
	}
}

// --- CountScanWorkItemsByRun ---

func TestCountScanWorkItemsByRun(t *testing.T) {
	q, runID := setupScanWorkTest(t)
	now := time.Now().UTC()

	count, err := q.CountScanWorkItemsByRun(runID)
	if err != nil {
		t.Fatalf("CountScanWorkItemsByRun empty: %v", err)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}

	for i := 0; i < 3; i++ {
		w := &models.ScanWorkItem{
			ID: util.GenerateID(), RunID: runID, ProjectID: "proj-1",
			AssetID: "asset-1", Action: string(rune('A' + i)),
			Status: models.WorkStatusPending, CreatedAt: now,
		}
		q.CreateScanWorkItem(w)
	}

	count, err = q.CountScanWorkItemsByRun(runID)
	if err != nil {
		t.Fatalf("CountScanWorkItemsByRun: %v", err)
	}
	if count != 3 {
		t.Errorf("count = %d, want 3", count)
	}
}

// --- ListScanWorkItemsByRunAndStatus ---

func TestListScanWorkItemsByRunAndStatus(t *testing.T) {
	q, runID := setupScanWorkTest(t)
	now := time.Now().UTC()

	items := []struct {
		action string
		status models.WorkStatus
	}{
		{"HTTPX_FINGERPRINT", models.WorkStatusDone},
		{"NUCLEI_SCAN", models.WorkStatusPending},
		{"PORT_SCAN", models.WorkStatusDone},
	}
	for _, it := range items {
		w := &models.ScanWorkItem{
			ID: util.GenerateID(), RunID: runID, ProjectID: "proj-1",
			AssetID: "asset-1", Action: it.action, Status: it.status, CreatedAt: now,
		}
		q.CreateScanWorkItem(w)
	}

	done, err := q.ListScanWorkItemsByRunAndStatus(runID, models.WorkStatusDone)
	if err != nil {
		t.Fatalf("ListScanWorkItemsByRunAndStatus: %v", err)
	}
	if len(done) != 2 {
		t.Fatalf("expected 2 done, got %d", len(done))
	}

	pending, err := q.ListScanWorkItemsByRunAndStatus(runID, models.WorkStatusPending)
	if err != nil {
		t.Fatalf("ListScanWorkItemsByRunAndStatus pending: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending, got %d", len(pending))
	}
}

// --- UpdatePipelineRunLastNewAssetAt ---

func TestUpdatePipelineRunLastNewAssetAt(t *testing.T) {
	q, runID := setupScanWorkTest(t)

	lastAsset := time.Now().UTC().Add(-5 * time.Minute)
	if err := q.UpdatePipelineRunLastNewAssetAt(runID, lastAsset); err != nil {
		t.Fatalf("UpdatePipelineRunLastNewAssetAt: %v", err)
	}

	run, err := q.GetPipelineRun(runID)
	if err != nil {
		t.Fatalf("GetPipelineRun: %v", err)
	}
	if run.LastNewAssetAt == nil {
		t.Fatal("expected last_new_asset_at to be set")
	}
}

// --- ListAssetsByRun ---

func TestListAssetsByRun(t *testing.T) {
	q, runID := setupScanWorkTest(t)
	now := time.Now().UTC()

	// Create assets
	if err := q.CreateAsset(&models.Asset{
		ID: "asset-1", ProjectID: "proj-1", Type: models.AssetTypeDomain,
		Value: "a.example.com", NormalizedValue: "a.example.com",
		FirstSeen: now, LastSeen: now,
	}); err != nil {
		t.Fatalf("create asset-1: %v", err)
	}
	if err := q.CreateAsset(&models.Asset{
		ID: "asset-2", ProjectID: "proj-1", Type: models.AssetTypeDomain,
		Value: "b.example.com", NormalizedValue: "b.example.com",
		FirstSeen: now, LastSeen: now,
	}); err != nil {
		t.Fatalf("create asset-2: %v", err)
	}

	// Create work items linking assets to run
	for _, assetID := range []string{"asset-1", "asset-2"} {
		w := &models.ScanWorkItem{
			ID: util.GenerateID(), RunID: runID, ProjectID: "proj-1",
			AssetID: assetID, Action: "HTTPX_FINGERPRINT",
			Status: models.WorkStatusDone, CreatedAt: now,
		}
		q.CreateScanWorkItem(w)
	}

	assets, err := q.ListAssetsByRun("proj-1", runID)
	if err != nil {
		t.Fatalf("ListAssetsByRun: %v", err)
	}
	if len(assets) != 2 {
		t.Fatalf("expected 2 assets, got %d", len(assets))
	}
}

func TestListAssetsByRun_Empty(t *testing.T) {
	q := New(openTestDB(t))

	assets, err := q.ListAssetsByRun("proj-none", "run-none")
	if err != nil {
		t.Fatalf("ListAssetsByRun empty: %v", err)
	}
	if len(assets) != 0 {
		t.Errorf("expected 0, got %d", len(assets))
	}
}

// --- GetToolStatsByRun ---

func TestGetToolStatsByRun(t *testing.T) {
	q, runID := setupScanWorkTest(t)
	now := time.Now().UTC()

	started := now.Add(-10 * time.Second)
	completed := now

	// Use unique (asset_id, action) pairs to avoid UNIQUE constraint violation
	items := []struct {
		assetID     string
		action      string
		status      models.WorkStatus
		startedAt   *time.Time
		completedAt *time.Time
	}{
		{"asset-ts-1", "HTTPX_FINGERPRINT", models.WorkStatusDone, &started, &completed},
		{"asset-ts-2", "HTTPX_FINGERPRINT", models.WorkStatusDone, &started, &completed},
		{"asset-ts-3", "NUCLEI_SCAN", models.WorkStatusFailed, &started, &completed},
		{"asset-ts-4", "NUCLEI_SCAN", models.WorkStatusSkipped, nil, nil},
	}
	for _, it := range items {
		w := &models.ScanWorkItem{
			ID: util.GenerateID(), RunID: runID, ProjectID: "proj-1",
			AssetID: it.assetID, Action: it.action, Status: it.status,
			StartedAt: it.startedAt, CompletedAt: it.completedAt, CreatedAt: now,
		}
		q.CreateScanWorkItem(w)
	}

	stats, err := q.GetToolStatsByRun(runID)
	if err != nil {
		t.Fatalf("GetToolStatsByRun: %v", err)
	}
	if len(stats) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(stats))
	}

	statsMap := map[string]*ToolStats{}
	for _, s := range stats {
		statsMap[s.Tool] = s
	}

	httpx := statsMap["HTTPX_FINGERPRINT"]
	if httpx == nil {
		t.Fatal("expected HTTPX_FINGERPRINT stats")
	}
	if httpx.TotalCalls != 2 {
		t.Errorf("httpx total = %d, want 2", httpx.TotalCalls)
	}
	if httpx.SuccessCount != 2 {
		t.Errorf("httpx success = %d, want 2", httpx.SuccessCount)
	}

	nuclei := statsMap["NUCLEI_SCAN"]
	if nuclei == nil {
		t.Fatal("expected NUCLEI_SCAN stats")
	}
	if nuclei.TotalCalls != 2 {
		t.Errorf("nuclei total = %d, want 2", nuclei.TotalCalls)
	}
	if nuclei.FailedCount != 1 {
		t.Errorf("nuclei failed = %d, want 1", nuclei.FailedCount)
	}
	if nuclei.SkippedCount != 1 {
		t.Errorf("nuclei skipped = %d, want 1", nuclei.SkippedCount)
	}
}

func TestGetToolStatsByRun_Empty(t *testing.T) {
	q := New(openTestDB(t))

	stats, err := q.GetToolStatsByRun("nonexistent")
	if err != nil {
		t.Fatalf("GetToolStatsByRun empty: %v", err)
	}
	if len(stats) != 0 {
		t.Errorf("expected 0, got %d", len(stats))
	}
}

// --- GetToolErrorStatsByRun ---

func TestGetToolErrorStatsByRun(t *testing.T) {
	q, runID := setupScanWorkTest(t)
	now := time.Now().UTC()

	// Use unique (asset_id, action) pairs to avoid UNIQUE constraint violation
	items := []struct {
		assetID string
		action  string
		errMsg  string
	}{
		{"asset-err-1", "NUCLEI_SCAN", "connection timeout"},
		{"asset-err-2", "NUCLEI_SCAN", "connection timeout"},
		{"asset-err-3", "NUCLEI_SCAN", "dns resolution failed"},
		{"asset-err-4", "HTTPX_FINGERPRINT", ""},
	}
	for _, it := range items {
		w := &models.ScanWorkItem{
			ID: util.GenerateID(), RunID: runID, ProjectID: "proj-1",
			AssetID: it.assetID, Action: it.action,
			Status: models.WorkStatusFailed, Error: it.errMsg, CreatedAt: now,
		}
		q.CreateScanWorkItem(w)
	}

	stats, err := q.GetToolErrorStatsByRun(runID)
	if err != nil {
		t.Fatalf("GetToolErrorStatsByRun: %v", err)
	}
	if len(stats) != 2 {
		t.Fatalf("expected 2 error groups, got %d", len(stats))
	}

	// Verify the connection timeout group has count 2
	for _, s := range stats {
		if s.Tool == "NUCLEI_SCAN" && s.Error == "connection timeout" {
			if s.Count != 2 {
				t.Errorf("connection timeout count = %d, want 2", s.Count)
			}
		}
	}
}

func TestGetToolErrorStatsByRun_Empty(t *testing.T) {
	q := New(openTestDB(t))

	stats, err := q.GetToolErrorStatsByRun("nonexistent")
	if err != nil {
		t.Fatalf("GetToolErrorStatsByRun empty: %v", err)
	}
	if len(stats) != 0 {
		t.Errorf("expected 0, got %d", len(stats))
	}
}

// --- ListScanWorkItemsByRunPaginated with limit ---

func TestListScanWorkItemsByRunPaginated_WithLimit(t *testing.T) {
	q, runID := setupScanWorkTest(t)
	now := time.Now().UTC()

	for i := 0; i < 5; i++ {
		w := &models.ScanWorkItem{
			ID: util.GenerateID(), RunID: runID, ProjectID: "proj-1",
			AssetID: "asset-pag-" + string(rune('a'+i)), Action: "HTTPX_FINGERPRINT",
			Status: models.WorkStatusPending, CreatedAt: now,
		}
		q.CreateScanWorkItem(w)
	}

	// With limit
	page, err := q.ListScanWorkItemsByRunPaginated(runID, 2, 0)
	if err != nil {
		t.Fatalf("ListScanWorkItemsByRunPaginated: %v", err)
	}
	if len(page) != 2 {
		t.Fatalf("expected 2, got %d", len(page))
	}
}

// --- GetScanRunMetrics nil run ---

func TestGetScanRunMetrics_NilRun(t *testing.T) {
	q := New(openTestDB(t))

	m, err := q.GetScanRunMetrics("nonexistent")
	if err != nil {
		t.Fatalf("GetScanRunMetrics: %v", err)
	}
	if m != nil {
		t.Error("expected nil for nonexistent run")
	}
}

// --- scanWorkItemRow with all nullable fields ---

func TestScanWorkItem_AllNullableFields(t *testing.T) {
	q, runID := setupScanWorkTest(t)
	now := time.Now().UTC()

	inputFile := "/tmp/input.txt"
	memberIDs := "a,b,c"
	bucketKey := "bucket-1"

	w := &models.ScanWorkItem{
		ID: util.GenerateID(), RunID: runID, ProjectID: "proj-1",
		AssetID: "asset-full", Action: "NUCLEI_SCAN",
		Status: models.WorkStatusDone, SkipReason: "", Stage: "scan",
		InputFile: inputFile, MemberAssetIDs: memberIDs, BucketKey: bucketKey,
		Generation: 3, BatchMode: true,
		StartedAt: &now, CompletedAt: &now, CreatedAt: now,
	}
	if err := q.CreateScanWorkItem(w); err != nil {
		t.Fatalf("CreateScanWorkItem: %v", err)
	}

	got, err := q.GetScanWorkItem(w.ID)
	if err != nil {
		t.Fatalf("GetScanWorkItem: %v", err)
	}
	if got.InputFile != inputFile {
		t.Errorf("input_file = %q, want %q", got.InputFile, inputFile)
	}
	if got.MemberAssetIDs != memberIDs {
		t.Errorf("member_asset_ids = %q, want %q", got.MemberAssetIDs, memberIDs)
	}
	if got.BucketKey != bucketKey {
		t.Errorf("bucket_key = %q, want %q", got.BucketKey, bucketKey)
	}
	if got.Generation != 3 {
		t.Errorf("generation = %d, want 3", got.Generation)
	}
	if !got.BatchMode {
		t.Error("expected batch_mode=true")
	}
	if got.Stage != "scan" {
		t.Errorf("stage = %q, want scan", got.Stage)
	}
}
