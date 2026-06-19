package db

import (
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
)

// setupScanTestData creates a project, scan plan, and run for tests that need them.
// Returns (Queries, projectID, runID).
func setupScanTestData(t *testing.T) (*Queries, string, string) {
	t.Helper()
	q := New(openTestDB(t))
	now := time.Now().UTC().Truncate(time.Second)

	if err := q.CreateProject(&models.Project{
		ID:             "proj-1",
		Name:           "test",
		RateLimit:      10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt:      now,
		UpdatedAt:      now,
	}); err != nil {
		t.Fatalf("create project: %v", err)
	}

	if err := q.CreateScanPlan(&models.ScanPlan{
		ID:           "plan-1",
		ProjectID:    "proj-1",
		WorkflowType: "manual",
		Profile:      models.ProfileStandard,
		Status:       "approved",
		CreatedBy:    "test",
		CreatedAt:    now,
	}); err != nil {
		t.Fatalf("create scan plan: %v", err)
	}

	if err := q.CreateRun(&models.Run{
		ID:        "run-1",
		ProjectID: "proj-1",
		Name:      "test-run",
		Status:    models.RunRunning,
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("create run: %v", err)
	}

	return q, "proj-1", "run-1"
}

// createTestScanTask inserts a scan task for use in tests.
func createTestScanTask(t *testing.T, q *Queries, taskID, runID string) {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Second)
	runPtr := &runID
	if err := q.CreateScanTask(&models.ScanTask{
		ID:              taskID,
		ProjectID:       "proj-1",
		PlanID:          "plan-1",
		RunID:           runPtr,
		Tool:            "nuclei",
		CommandTemplate: "nuclei -u {{target}}",
		Status:          models.TaskCreated,
		CreatedAt:       now,
	}); err != nil {
		t.Fatalf("create scan task: %v", err)
	}
}

// --- 1. SetScanTaskWorker ---

func TestSetScanTaskWorker(t *testing.T) {
	q, _, runID := setupScanTestData(t)
	createTestScanTask(t, q, "task-1", runID)

	if err := q.SetScanTaskWorker("task-1", "worker-42"); err != nil {
		t.Fatalf("SetScanTaskWorker: %v", err)
	}

	task, err := q.GetScanTask("task-1")
	if err != nil {
		t.Fatalf("GetScanTask: %v", err)
	}
	if task == nil {
		t.Fatal("expected task, got nil")
	}
	if task.WorkerID == nil {
		t.Fatal("expected WorkerID to be set")
	}
	if *task.WorkerID != "worker-42" {
		t.Errorf("WorkerID = %q, want %q", *task.WorkerID, "worker-42")
	}
}

func TestSetScanTaskWorker_UpdatesExistingWorker(t *testing.T) {
	q, _, runID := setupScanTestData(t)
	createTestScanTask(t, q, "task-1", runID)

	if err := q.SetScanTaskWorker("task-1", "worker-A"); err != nil {
		t.Fatalf("first SetScanTaskWorker: %v", err)
	}
	if err := q.SetScanTaskWorker("task-1", "worker-B"); err != nil {
		t.Fatalf("second SetScanTaskWorker: %v", err)
	}

	task, err := q.GetScanTask("task-1")
	if err != nil {
		t.Fatalf("GetScanTask: %v", err)
	}
	if *task.WorkerID != "worker-B" {
		t.Errorf("WorkerID = %q, want %q", *task.WorkerID, "worker-B")
	}
}

// --- 2. CreateScanStep / UpdateScanStepStatus / ListScanStepsByTask ---

func TestScanStep_CRUD_RoundTrip(t *testing.T) {
	q, _, runID := setupScanTestData(t)
	createTestScanTask(t, q, "task-1", runID)
	now := time.Now().UTC().Truncate(time.Second)

	// Create two steps
	if err := q.CreateScanStep(&models.ScanStep{
		ID:        "step-1",
		TaskID:    "task-1",
		Name:      models.StepRunTool,
		Status:    models.StepPending,
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("create step-1: %v", err)
	}
	if err := q.CreateScanStep(&models.ScanStep{
		ID:        "step-2",
		TaskID:    "task-1",
		Name:      models.StepCollectArtifacts,
		Status:    models.StepPending,
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("create step-2: %v", err)
	}

	// List
	steps, err := q.ListScanStepsByTask("task-1")
	if err != nil {
		t.Fatalf("ListScanStepsByTask: %v", err)
	}
	if len(steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(steps))
	}
	if steps[0].ID != "step-1" || steps[1].ID != "step-2" {
		t.Errorf("unexpected step order: [%s, %s]", steps[0].ID, steps[1].ID)
	}

	// Update step status
	completedAt := now.Add(5 * time.Second)
	if err := q.UpdateScanStepStatus("step-1", models.StepCompleted, &completedAt, "", ""); err != nil {
		t.Fatalf("UpdateScanStepStatus: %v", err)
	}

	steps, err = q.ListScanStepsByTask("task-1")
	if err != nil {
		t.Fatalf("ListScanStepsByTask after update: %v", err)
	}
	if steps[0].Status != models.StepCompleted {
		t.Errorf("step-1 status = %q, want %q", steps[0].Status, models.StepCompleted)
	}
	if steps[0].FinishedAt == nil {
		t.Error("expected FinishedAt to be set after update")
	}
}

func TestScanStep_UpdateWithError(t *testing.T) {
	q, _, runID := setupScanTestData(t)
	createTestScanTask(t, q, "task-1", runID)
	now := time.Now().UTC().Truncate(time.Second)

	if err := q.CreateScanStep(&models.ScanStep{
		ID:        "step-1",
		TaskID:    "task-1",
		Name:      models.StepRunTool,
		Status:    models.StepRunning,
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("create step: %v", err)
	}

	completedAt := now.Add(2 * time.Second)
	if err := q.UpdateScanStepStatus("step-1", models.StepFailed, &completedAt, "ERR_TIMEOUT", "tool timed out"); err != nil {
		t.Fatalf("UpdateScanStepStatus with error: %v", err)
	}

	steps, err := q.ListScanStepsByTask("task-1")
	if err != nil {
		t.Fatalf("ListScanStepsByTask: %v", err)
	}
	if steps[0].Status != models.StepFailed {
		t.Errorf("status = %q, want %q", steps[0].Status, models.StepFailed)
	}
	if steps[0].ErrorCode != "ERR_TIMEOUT" {
		t.Errorf("ErrorCode = %q, want %q", steps[0].ErrorCode, "ERR_TIMEOUT")
	}
	if steps[0].ErrorSummary != "tool timed out" {
		t.Errorf("ErrorSummary = %q, want %q", steps[0].ErrorSummary, "tool timed out")
	}
}

func TestListScanStepsByTask_Empty(t *testing.T) {
	q, _, runID := setupScanTestData(t)
	createTestScanTask(t, q, "task-1", runID)

	steps, err := q.ListScanStepsByTask("task-1")
	if err != nil {
		t.Fatalf("ListScanStepsByTask: %v", err)
	}
	if len(steps) != 0 {
		t.Errorf("expected 0 steps, got %d", len(steps))
	}
}

// --- 3. CountRunsByProject ---

func TestCountRunsByProject(t *testing.T) {
	q, projID, _ := setupScanTestData(t)
	now := time.Now().UTC().Truncate(time.Second)

	count, err := q.CountRunsByProject(projID)
	if err != nil {
		t.Fatalf("CountRunsByProject: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}

	// Add two more runs
	for i := 2; i <= 3; i++ {
		if err := q.CreateRun(&models.Run{
			ID:        "run-" + string(rune('0'+i)),
			ProjectID: projID,
			Name:      "extra-run",
			Status:    models.RunPending,
			CreatedAt: now.Add(time.Duration(i) * time.Minute),
		}); err != nil {
			t.Fatalf("create run-%d: %v", i, err)
		}
	}

	count, err = q.CountRunsByProject(projID)
	if err != nil {
		t.Fatalf("CountRunsByProject after insert: %v", err)
	}
	if count != 3 {
		t.Errorf("count = %d, want 3", count)
	}
}

func TestCountRunsByProject_OtherProject(t *testing.T) {
	q, _, _ := setupScanTestData(t)

	count, err := q.CountRunsByProject("nonexistent")
	if err != nil {
		t.Fatalf("CountRunsByProject: %v", err)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}
}

// --- 4. ListRunsByProjectPaginated ---

func TestListRunsByProjectPaginated(t *testing.T) {
	q, projID, _ := setupScanTestData(t)
	now := time.Now().UTC().Truncate(time.Second)

	// Insert 4 more runs (total 5)
	for i := 2; i <= 5; i++ {
		if err := q.CreateRun(&models.Run{
			ID:        fmtRunID(i),
			ProjectID: projID,
			Name:      "run",
			Status:    models.RunPending,
			CreatedAt: now.Add(time.Duration(i) * time.Minute),
		}); err != nil {
			t.Fatalf("create run %d: %v", i, err)
		}
	}

	// Page 1: limit 2, offset 0
	page1, err := q.ListRunsByProjectPaginated(projID, 2, 0)
	if err != nil {
		t.Fatalf("page 1: %v", err)
	}
	if len(page1) != 2 {
		t.Fatalf("page 1: expected 2, got %d", len(page1))
	}

	// Page 2: limit 2, offset 2
	page2, err := q.ListRunsByProjectPaginated(projID, 2, 2)
	if err != nil {
		t.Fatalf("page 2: %v", err)
	}
	if len(page2) != 2 {
		t.Fatalf("page 2: expected 2, got %d", len(page2))
	}

	// Page 3: limit 2, offset 4 — only 1 remaining
	page3, err := q.ListRunsByProjectPaginated(projID, 2, 4)
	if err != nil {
		t.Fatalf("page 3: %v", err)
	}
	if len(page3) != 1 {
		t.Fatalf("page 3: expected 1, got %d", len(page3))
	}

	// Verify no overlap between pages
	ids := make(map[string]bool)
	for _, r := range append(append(page1, page2...), page3...) {
		if ids[r.ID] {
			t.Errorf("duplicate run ID across pages: %s", r.ID)
		}
		ids[r.ID] = true
	}
	if len(ids) != 5 {
		t.Errorf("expected 5 unique runs, got %d", len(ids))
	}
}

func fmtRunID(i int) string {
	return "run-" + string(rune('0'+i))
}

// --- 5. GetRawArtifact ---

func TestGetRawArtifact_Found(t *testing.T) {
	q, _, runID := setupScanTestData(t)
	createTestScanTask(t, q, "task-1", runID)
	now := time.Now().UTC().Truncate(time.Second)
	taskPtr := "task-1"

	if err := q.CreateRawArtifact(&models.RawArtifact{
		ID:              "art-1",
		ProjectID:       "proj-1",
		TaskID:          &taskPtr,
		Type:            models.ArtifactJSONL,
		Path:            "/tmp/output.jsonl",
		SHA256:          "abc123",
		Size:            1024,
		RedactionStatus: "none",
		CreatedAt:       now,
	}); err != nil {
		t.Fatalf("CreateRawArtifact: %v", err)
	}

	got, err := q.GetRawArtifact("art-1")
	if err != nil {
		t.Fatalf("GetRawArtifact: %v", err)
	}
	if got == nil {
		t.Fatal("expected artifact, got nil")
	}
	if got.ID != "art-1" {
		t.Errorf("ID = %q, want %q", got.ID, "art-1")
	}
	if got.Type != models.ArtifactJSONL {
		t.Errorf("Type = %q, want %q", got.Type, models.ArtifactJSONL)
	}
	if got.SHA256 != "abc123" {
		t.Errorf("SHA256 = %q, want %q", got.SHA256, "abc123")
	}
	if got.Size != 1024 {
		t.Errorf("Size = %d, want 1024", got.Size)
	}
}

func TestGetRawArtifact_NotFound(t *testing.T) {
	q := New(openTestDB(t))

	got, err := q.GetRawArtifact("nonexistent")
	if err != nil {
		t.Fatalf("GetRawArtifact: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

// --- 6. Screenshot CRUD ---

func TestScreenshot_CRUD(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC().Truncate(time.Second)

	// Create project first
	if err := q.CreateProject(&models.Project{
		ID: "proj-1", Name: "test", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create project: %v", err)
	}

	assetID := "asset-1"
	taskID := "task-1"
	takenAt := now.Add(-time.Hour)

	// Create prerequisite asset and scan_task for FK constraints
	if err := q.CreateAsset(&models.Asset{ID: assetID, ProjectID: "proj-1", Type: "url", Value: "https://example.com", NormalizedValue: "https://example.com", FirstSeen: now, LastSeen: now}); err != nil {
		t.Fatalf("create asset: %v", err)
	}
	if err := q.CreateScanTask(&models.ScanTask{ID: taskID, ProjectID: "proj-1", Tool: "nuclei", Status: models.TaskCompleted, CreatedAt: now}); err != nil {
		t.Fatalf("create scan_task: %v", err)
	}

	if err := q.CreateScreenshot(&models.Screenshot{
		ID:            "ss-1",
		ProjectID:     "proj-1",
		AssetID:       &assetID,
		TaskID:        &taskID,
		URL:           "https://example.com",
		OriginalPath:  "/tmp/ss1.png",
		ThumbnailPath: "/tmp/ss1_thumb.png",
		Width:         1920,
		Height:        1080,
		TakenAt:       takenAt,
	}); err != nil {
		t.Fatalf("CreateScreenshot: %v", err)
	}

	// Get by ID
	got, err := q.GetScreenshot("ss-1")
	if err != nil {
		t.Fatalf("GetScreenshot: %v", err)
	}
	if got == nil {
		t.Fatal("expected screenshot, got nil")
	}
	if got.URL != "https://example.com" {
		t.Errorf("URL = %q, want %q", got.URL, "https://example.com")
	}
	if got.Width != 1920 || got.Height != 1080 {
		t.Errorf("dimensions = %dx%d, want 1920x1080", got.Width, got.Height)
	}

	// Get by URL
	byURL, err := q.GetScreenshotByURL("proj-1", "https://example.com")
	if err != nil {
		t.Fatalf("GetScreenshotByURL: %v", err)
	}
	if byURL == nil {
		t.Fatal("expected screenshot by URL, got nil")
	}
	if byURL.ID != "ss-1" {
		t.Errorf("GetScreenshotByURL ID = %q, want %q", byURL.ID, "ss-1")
	}

	// List by project
	list, err := q.ListScreenshotsByProject("proj-1")
	if err != nil {
		t.Fatalf("ListScreenshotsByProject: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 screenshot, got %d", len(list))
	}
}

func TestGetScreenshot_NotFound(t *testing.T) {
	q := New(openTestDB(t))

	got, err := q.GetScreenshot("nonexistent")
	if err != nil {
		t.Fatalf("GetScreenshot: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestGetScreenshotByURL_NotFound(t *testing.T) {
	q := New(openTestDB(t))

	got, err := q.GetScreenshotByURL("proj-1", "https://nope.example.com")
	if err != nil {
		t.Fatalf("GetScreenshotByURL: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestListScreenshotsByProject_Empty(t *testing.T) {
	q := New(openTestDB(t))

	list, err := q.ListScreenshotsByProject("proj-none")
	if err != nil {
		t.Fatalf("ListScreenshotsByProject: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected 0, got %d", len(list))
	}
}

// --- 7. SaveDNSRecord / ListDNSRecordsByProject ---

func TestDNSRecord_SaveAndList(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC().Truncate(time.Second)

	if err := q.CreateProject(&models.Project{
		ID: "proj-1", Name: "test", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create project: %v", err)
	}

	if err := q.SaveDNSRecord(&models.DNSRecord{
		ID:        "dns-1",
		ProjectID: "proj-1",
		Domain:    "example.com",
		IPs:       []string{"1.2.3.4", "5.6.7.8"},
		CNAMEs:    []string{""},
		TTL:       300,
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("SaveDNSRecord: %v", err)
	}

	list, err := q.ListDNSRecordsByProject("proj-1")
	if err != nil {
		t.Fatalf("ListDNSRecordsByProject: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 record, got %d", len(list))
	}
	if list[0].Domain != "example.com" {
		t.Errorf("Domain = %q, want %q", list[0].Domain, "example.com")
	}
	if len(list[0].IPs) != 2 {
		t.Errorf("IPs count = %d, want 2", len(list[0].IPs))
	}
	if list[0].TTL != 300 {
		t.Errorf("TTL = %d, want 300", list[0].TTL)
	}
}

func TestDNSRecord_Upsert(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC().Truncate(time.Second)

	if err := q.CreateProject(&models.Project{
		ID: "proj-1", Name: "test", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create project: %v", err)
	}

	// Insert
	if err := q.SaveDNSRecord(&models.DNSRecord{
		ID: "dns-1", ProjectID: "proj-1", Domain: "example.com",
		IPs: []string{"1.2.3.4"}, CNAMEs: []string{""}, TTL: 300, CreatedAt: now,
	}); err != nil {
		t.Fatalf("first save: %v", err)
	}

	// Upsert with updated IPs
	if err := q.SaveDNSRecord(&models.DNSRecord{
		ID: "dns-1-updated", ProjectID: "proj-1", Domain: "example.com",
		IPs: []string{"10.0.0.1"}, CNAMEs: []string{"cname.example.com"}, TTL: 600, CreatedAt: now,
	}); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	list, err := q.ListDNSRecordsByProject("proj-1")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	// Upsert on (project_id, domain) should still yield 1 row
	if len(list) != 1 {
		t.Fatalf("expected 1 record after upsert, got %d", len(list))
	}
	if list[0].IPs[0] != "10.0.0.1" {
		t.Errorf("IP after upsert = %q, want %q", list[0].IPs[0], "10.0.0.1")
	}
	if list[0].TTL != 600 {
		t.Errorf("TTL after upsert = %d, want 600", list[0].TTL)
	}
}

// --- 8. SaveCDNResult / ListCDNResultsByProject ---

func TestCDNResult_SaveAndList(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC().Truncate(time.Second)

	if err := q.CreateProject(&models.Project{
		ID: "proj-1", Name: "test", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create project: %v", err)
	}

	if err := q.SaveCDNResult(&models.CDNResult{
		ID: "cdn-1", ProjectID: "proj-1", IP: "1.2.3.4",
		IsCDN: true, Provider: "cloudflare", Type: "cdn", CreatedAt: now,
	}); err != nil {
		t.Fatalf("SaveCDNResult: %v", err)
	}
	if err := q.SaveCDNResult(&models.CDNResult{
		ID: "cdn-2", ProjectID: "proj-1", IP: "5.6.7.8",
		IsCDN: false, CreatedAt: now,
	}); err != nil {
		t.Fatalf("SaveCDNResult 2: %v", err)
	}

	list, err := q.ListCDNResultsByProject("proj-1")
	if err != nil {
		t.Fatalf("ListCDNResultsByProject: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 results, got %d", len(list))
	}

	// Verify fields
	foundCDN := false
	for _, r := range list {
		if r.IsCDN {
			foundCDN = true
			if r.Provider != "cloudflare" {
				t.Errorf("Provider = %q, want %q", r.Provider, "cloudflare")
			}
		}
	}
	if !foundCDN {
		t.Error("expected at least one CDN=true result")
	}
}

func TestCDNResult_Upsert(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC().Truncate(time.Second)

	if err := q.CreateProject(&models.Project{
		ID: "proj-1", Name: "test", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create project: %v", err)
	}

	// Insert
	if err := q.SaveCDNResult(&models.CDNResult{
		ID: "cdn-1", ProjectID: "proj-1", IP: "10.0.0.1",
		IsCDN: false, CreatedAt: now,
	}); err != nil {
		t.Fatalf("first save: %v", err)
	}

	// Upsert: same (project_id, ip), now detected as CDN
	if err := q.SaveCDNResult(&models.CDNResult{
		ID: "cdn-1b", ProjectID: "proj-1", IP: "10.0.0.1",
		IsCDN: true, Provider: "akamai", Type: "cdn", CreatedAt: now,
	}); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	list, err := q.ListCDNResultsByProject("proj-1")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 after upsert, got %d", len(list))
	}
	if !list[0].IsCDN {
		t.Error("expected IsCDN=true after upsert")
	}
	if list[0].Provider != "akamai" {
		t.Errorf("Provider = %q, want %q", list[0].Provider, "akamai")
	}
}

// --- 9. SaveServiceFingerprint / ListServiceFingerprintsByProject ---

func TestServiceFingerprint_SaveAndList(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC().Truncate(time.Second)

	if err := q.CreateProject(&models.Project{
		ID: "proj-1", Name: "test", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create project: %v", err)
	}

	if err := q.SaveServiceFingerprint(&models.ServiceFingerprint{
		ID: "fp-1", ProjectID: "proj-1", IP: "1.2.3.4", Port: 443,
		Protocol: "tcp", IsWeb: true, Service: "https", Product: "nginx",
		Version: "1.21", Metadata: map[string]interface{}{"header": "Server: nginx"},
		Source: "httpx", CreatedAt: now,
	}); err != nil {
		t.Fatalf("SaveServiceFingerprint: %v", err)
	}

	if err := q.SaveServiceFingerprint(&models.ServiceFingerprint{
		ID: "fp-2", ProjectID: "proj-1", IP: "1.2.3.4", Port: 22,
		Protocol: "tcp", IsWeb: false, Service: "ssh", Product: "OpenSSH",
		Source: "naabu", CreatedAt: now,
	}); err != nil {
		t.Fatalf("SaveServiceFingerprint 2: %v", err)
	}

	list, err := q.ListServiceFingerprintsByProject("proj-1")
	if err != nil {
		t.Fatalf("ListServiceFingerprintsByProject: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 fingerprints, got %d", len(list))
	}

	// Verify the nginx fingerprint metadata round-trips
	var nginx *models.ServiceFingerprint
	for _, fp := range list {
		if fp.Port == 443 {
			nginx = fp
		}
	}
	if nginx == nil {
		t.Fatal("expected nginx fingerprint on port 443")
	}
	if nginx.Product != "nginx" {
		t.Errorf("Product = %q, want %q", nginx.Product, "nginx")
	}
	if nginx.Metadata == nil {
		t.Fatal("expected Metadata to be non-nil")
	}
	if nginx.Metadata["header"] != "Server: nginx" {
		t.Errorf("Metadata[header] = %v, want %q", nginx.Metadata["header"], "Server: nginx")
	}
}

func TestServiceFingerprint_Upsert(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC().Truncate(time.Second)

	if err := q.CreateProject(&models.Project{
		ID: "proj-1", Name: "test", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create project: %v", err)
	}

	// Insert
	if err := q.SaveServiceFingerprint(&models.ServiceFingerprint{
		ID: "fp-1", ProjectID: "proj-1", IP: "10.0.0.1", Port: 80,
		Protocol: "tcp", IsWeb: true, Service: "http", Product: "apache",
		Source: "httpx", CreatedAt: now,
	}); err != nil {
		t.Fatalf("first save: %v", err)
	}

	// Upsert: same (project_id, ip, port), updated version
	if err := q.SaveServiceFingerprint(&models.ServiceFingerprint{
		ID: "fp-1b", ProjectID: "proj-1", IP: "10.0.0.1", Port: 80,
		Protocol: "tcp", IsWeb: true, Service: "http", Product: "nginx",
		Version: "2.0", Source: "httpx", CreatedAt: now,
	}); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	list, err := q.ListServiceFingerprintsByProject("proj-1")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 after upsert, got %d", len(list))
	}
	if list[0].Product != "nginx" {
		t.Errorf("Product = %q, want %q (upserted)", list[0].Product, "nginx")
	}
	if list[0].Version != "2.0" {
		t.Errorf("Version = %q, want %q (upserted)", list[0].Version, "2.0")
	}
}

// --- 10. PipelineRun updates ---

func TestPipelineRun_UpdateStatus(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC().Truncate(time.Second)

	if err := q.CreateProject(&models.Project{
		ID: "proj-1", Name: "test", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create project: %v", err)
	}

	pr := &models.PipelineRun{
		ID: "pr-1", ProjectID: "proj-1", Mode: "standard",
		Status: "running", Stage: "discover", EngineState: "running",
		StartedAt: now, CreatedAt: now,
	}
	if err := q.CreatePipelineRun(pr); err != nil {
		t.Fatalf("CreatePipelineRun: %v", err)
	}

	// Update status
	if err := q.UpdatePipelineRunStatus("pr-1", "completed"); err != nil {
		t.Fatalf("UpdatePipelineRunStatus: %v", err)
	}

	got, err := q.GetPipelineRun("pr-1")
	if err != nil {
		t.Fatalf("GetPipelineRun: %v", err)
	}
	if got.Status != "completed" {
		t.Errorf("Status = %q, want %q", got.Status, "completed")
	}
}

func TestPipelineRun_UpdateStage(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC().Truncate(time.Second)

	if err := q.CreateProject(&models.Project{
		ID: "proj-1", Name: "test", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create project: %v", err)
	}

	pr := &models.PipelineRun{
		ID: "pr-1", ProjectID: "proj-1", Mode: "standard",
		Status: "running", Stage: "discover", EngineState: "running",
		StartedAt: now, CreatedAt: now,
	}
	if err := q.CreatePipelineRun(pr); err != nil {
		t.Fatalf("CreatePipelineRun: %v", err)
	}

	if err := q.UpdatePipelineRunStage("pr-1", "scan"); err != nil {
		t.Fatalf("UpdatePipelineRunStage: %v", err)
	}

	got, err := q.GetPipelineRun("pr-1")
	if err != nil {
		t.Fatalf("GetPipelineRun: %v", err)
	}
	if got.Stage != "scan" {
		t.Errorf("Stage = %q, want %q", got.Stage, "scan")
	}
}

func TestPipelineRun_UpdateError(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC().Truncate(time.Second)

	if err := q.CreateProject(&models.Project{
		ID: "proj-1", Name: "test", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create project: %v", err)
	}

	pr := &models.PipelineRun{
		ID: "pr-1", ProjectID: "proj-1", Mode: "standard",
		Status: "running", Stage: "discover", EngineState: "running",
		StartedAt: now, CreatedAt: now,
	}
	if err := q.CreatePipelineRun(pr); err != nil {
		t.Fatalf("CreatePipelineRun: %v", err)
	}

	if err := q.UpdatePipelineRunError("pr-1", "worker timeout"); err != nil {
		t.Fatalf("UpdatePipelineRunError: %v", err)
	}

	got, err := q.GetPipelineRun("pr-1")
	if err != nil {
		t.Fatalf("GetPipelineRun: %v", err)
	}
	if got.Error != "worker timeout" {
		t.Errorf("Error = %q, want %q", got.Error, "worker timeout")
	}
}

func TestPipelineRun_UpdateCompleted(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC().Truncate(time.Second)

	if err := q.CreateProject(&models.Project{
		ID: "proj-1", Name: "test", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create project: %v", err)
	}

	pr := &models.PipelineRun{
		ID: "pr-1", ProjectID: "proj-1", Mode: "standard",
		Status: "running", Stage: "complete", EngineState: "running",
		StartedAt: now, CreatedAt: now,
	}
	if err := q.CreatePipelineRun(pr); err != nil {
		t.Fatalf("CreatePipelineRun: %v", err)
	}

	completedAt := now.Add(10 * time.Minute)
	if err := q.UpdatePipelineRunCompleted("pr-1", completedAt); err != nil {
		t.Fatalf("UpdatePipelineRunCompleted: %v", err)
	}

	got, err := q.GetPipelineRun("pr-1")
	if err != nil {
		t.Fatalf("GetPipelineRun: %v", err)
	}
	if got.Status != "completed" {
		t.Errorf("Status = %q, want %q", got.Status, "completed")
	}
	if got.CompletedAt == nil {
		t.Fatal("expected CompletedAt to be set")
	}
}

// --- 11. PipelineRun list/count/paginate ---

func createTestPipelineRun(t *testing.T, q *Queries, id, projectID, status string, createdAt time.Time) {
	t.Helper()
	if err := q.CreatePipelineRun(&models.PipelineRun{
		ID: id, ProjectID: projectID, Mode: "standard",
		Status: status, Stage: "discover", EngineState: "running",
		StartedAt: createdAt, CreatedAt: createdAt,
	}); err != nil {
		t.Fatalf("CreatePipelineRun %s: %v", id, err)
	}
}

func TestListPipelineRunsByProject(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC().Truncate(time.Second)

	if err := q.CreateProject(&models.Project{
		ID: "proj-1", Name: "test", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create project: %v", err)
	}

	createTestPipelineRun(t, q, "pr-1", "proj-1", "running", now)
	createTestPipelineRun(t, q, "pr-2", "proj-1", "completed", now.Add(time.Minute))

	list, err := q.ListPipelineRunsByProject("proj-1")
	if err != nil {
		t.Fatalf("ListPipelineRunsByProject: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2, got %d", len(list))
	}
	// DESC order: pr-2 first
	if list[0].ID != "pr-2" {
		t.Errorf("first ID = %q, want %q (DESC order)", list[0].ID, "pr-2")
	}
}

func TestListPipelineRunsByStatus(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC().Truncate(time.Second)

	if err := q.CreateProject(&models.Project{
		ID: "proj-1", Name: "test", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create project: %v", err)
	}

	createTestPipelineRun(t, q, "pr-1", "proj-1", "running", now)
	createTestPipelineRun(t, q, "pr-2", "proj-1", "completed", now.Add(time.Minute))
	createTestPipelineRun(t, q, "pr-3", "proj-1", "running", now.Add(2*time.Minute))

	list, err := q.ListPipelineRunsByStatus("running")
	if err != nil {
		t.Fatalf("ListPipelineRunsByStatus: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 running, got %d", len(list))
	}
}

func TestCountPipelineRunsByProject(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC().Truncate(time.Second)

	if err := q.CreateProject(&models.Project{
		ID: "proj-1", Name: "test", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create project: %v", err)
	}

	count, err := q.CountPipelineRunsByProject("proj-1")
	if err != nil {
		t.Fatalf("count before: %v", err)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}

	createTestPipelineRun(t, q, "pr-1", "proj-1", "running", now)
	createTestPipelineRun(t, q, "pr-2", "proj-1", "completed", now.Add(time.Minute))

	count, err = q.CountPipelineRunsByProject("proj-1")
	if err != nil {
		t.Fatalf("count after: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}

func TestListPipelineRunsByProjectPaginated(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC().Truncate(time.Second)

	if err := q.CreateProject(&models.Project{
		ID: "proj-1", Name: "test", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create project: %v", err)
	}

	for i := 1; i <= 5; i++ {
		createTestPipelineRun(t, q, fmtPRID(i), "proj-1", "running", now.Add(time.Duration(i)*time.Minute))
	}

	page1, err := q.ListPipelineRunsByProjectPaginated("proj-1", 2, 0)
	if err != nil {
		t.Fatalf("page1: %v", err)
	}
	if len(page1) != 2 {
		t.Fatalf("page1: expected 2, got %d", len(page1))
	}

	page2, err := q.ListPipelineRunsByProjectPaginated("proj-1", 2, 2)
	if err != nil {
		t.Fatalf("page2: %v", err)
	}
	if len(page2) != 2 {
		t.Fatalf("page2: expected 2, got %d", len(page2))
	}

	page3, err := q.ListPipelineRunsByProjectPaginated("proj-1", 2, 4)
	if err != nil {
		t.Fatalf("page3: %v", err)
	}
	if len(page3) != 1 {
		t.Fatalf("page3: expected 1, got %d", len(page3))
	}
}

func fmtPRID(i int) string {
	return "pr-" + string(rune('0'+i))
}

// --- 12. PipelineRunStage updates and listing ---

func TestPipelineRunStage_UpdateAndList(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC().Truncate(time.Second)

	if err := q.CreateProject(&models.Project{
		ID: "proj-1", Name: "test", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create project: %v", err)
	}

	createTestPipelineRun(t, q, "pr-1", "proj-1", "running", now)

	workTotal := 10
	workDone := 5
	workRunning := 2
	round := 1
	if err := q.CreatePipelineRunStage(&models.PipelineRunStage{
		ID: "stage-1", RunID: "pr-1", Stage: "discover",
		Status: models.StageStatusRunning, WorkTotal: &workTotal,
		WorkDone: &workDone, WorkRunning: &workRunning, Round: &round,
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("CreatePipelineRunStage: %v", err)
	}

	workDone2 := 10
	workRunning2 := 0
	completedAt := now.Add(30 * time.Second)
	if err := q.CreatePipelineRunStage(&models.PipelineRunStage{
		ID: "stage-2", RunID: "pr-1", Stage: "scan",
		Status: models.StageStatusCompleted, WorkTotal: &workTotal,
		WorkDone: &workDone2, WorkRunning: &workRunning2, Round: &round,
		StartedAt: &now, CompletedAt: &completedAt, CreatedAt: now,
	}); err != nil {
		t.Fatalf("CreatePipelineRunStage 2: %v", err)
	}

	// List
	list, err := q.ListPipelineRunStages("pr-1")
	if err != nil {
		t.Fatalf("ListPipelineRunStages: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 stages, got %d", len(list))
	}

	// Update stage-1 status
	completedAt2 := now.Add(time.Minute)
	if err := q.UpdatePipelineRunStageRecord("stage-1", models.StageStatusCompleted, "", &completedAt2); err != nil {
		t.Fatalf("UpdatePipelineRunStageRecord: %v", err)
	}

	list, err = q.ListPipelineRunStages("pr-1")
	if err != nil {
		t.Fatalf("ListPipelineRunStages after update: %v", err)
	}
	if list[0].Status != models.StageStatusCompleted {
		t.Errorf("stage-1 status = %q, want %q", list[0].Status, models.StageStatusCompleted)
	}
	if list[0].CompletedAt == nil {
		t.Error("expected CompletedAt to be set")
	}
}

func TestPipelineRunStage_UpdateWithError(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC().Truncate(time.Second)

	if err := q.CreateProject(&models.Project{
		ID: "proj-1", Name: "test", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create project: %v", err)
	}

	createTestPipelineRun(t, q, "pr-1", "proj-1", "running", now)

	if err := q.CreatePipelineRunStage(&models.PipelineRunStage{
		ID: "stage-1", RunID: "pr-1", Stage: "discover",
		Status: models.StageStatusRunning, CreatedAt: now,
	}); err != nil {
		t.Fatalf("CreatePipelineRunStage: %v", err)
	}

	completedAt := now.Add(10 * time.Second)
	if err := q.UpdatePipelineRunStageRecord("stage-1", models.StageStatusFailed, "connection refused", &completedAt); err != nil {
		t.Fatalf("UpdatePipelineRunStageRecord: %v", err)
	}

	list, err := q.ListPipelineRunStages("pr-1")
	if err != nil {
		t.Fatalf("ListPipelineRunStages: %v", err)
	}
	if list[0].Status != models.StageStatusFailed {
		t.Errorf("status = %q, want %q", list[0].Status, models.StageStatusFailed)
	}
	if list[0].Error != "connection refused" {
		t.Errorf("Error = %q, want %q", list[0].Error, "connection refused")
	}
}

// --- 13. Dashboard aggregate counts ---

func TestCountActiveRuns(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC().Truncate(time.Second)

	if err := q.CreateProject(&models.Project{
		ID: "proj-1", Name: "test", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create project: %v", err)
	}

	// No runs yet
	count, err := q.CountActiveRuns()
	if err != nil {
		t.Fatalf("CountActiveRuns: %v", err)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}

	// Add one running and one completed run
	if err := q.CreateRun(&models.Run{
		ID: "run-r", ProjectID: "proj-1", Name: "running",
		Status: models.RunRunning, CreatedAt: now,
	}); err != nil {
		t.Fatalf("create running run: %v", err)
	}
	if err := q.CreateRun(&models.Run{
		ID: "run-c", ProjectID: "proj-1", Name: "completed",
		Status: models.RunCompleted, CreatedAt: now,
	}); err != nil {
		t.Fatalf("create completed run: %v", err)
	}

	count, err = q.CountActiveRuns()
	if err != nil {
		t.Fatalf("CountActiveRuns: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
}

func TestCountPendingFindings(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC().Truncate(time.Second)

	if err := q.CreateProject(&models.Project{
		ID: "proj-1", Name: "test", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create project: %v", err)
	}

	count, err := q.CountPendingFindings()
	if err != nil {
		t.Fatalf("CountPendingFindings empty: %v", err)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}

	// Add a pending_review finding
	if err := q.CreateFinding(&models.Finding{
		ID: "f-1", ProjectID: "proj-1", SourceTool: "nuclei",
		SourceRuleID: "r-1", DedupKey: "dk-1", Title: "XSS",
		Severity: models.SeverityHigh, Confidence: 90, Priority: 80,
		Status: models.FindingPendingReview, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("CreateFinding: %v", err)
	}
	// Add a non-pending finding
	if err := q.CreateFinding(&models.Finding{
		ID: "f-2", ProjectID: "proj-1", SourceTool: "nuclei",
		SourceRuleID: "r-2", DedupKey: "dk-2", Title: "Info",
		Severity: models.SeverityInfo, Confidence: 50, Priority: 10,
		Status: models.FindingConfirmed, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("CreateFinding 2: %v", err)
	}

	count, err = q.CountPendingFindings()
	if err != nil {
		t.Fatalf("CountPendingFindings: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
}

func TestCountRunningScanTasksPerWorker(t *testing.T) {
	q, _, runID := setupScanTestData(t)
	now := time.Now().UTC().Truncate(time.Second)

	// Create tasks with different workers and statuses
	workerA := "worker-A"
	workerB := "worker-B"
	if err := q.CreateScanTask(&models.ScanTask{
		ID: "t-1", ProjectID: "proj-1", PlanID: "plan-1", RunID: &runID,
		Tool: "nuclei", Status: models.TaskRunning, CreatedAt: now,
	}); err != nil {
		t.Fatalf("task 1: %v", err)
	}
	if err := q.SetScanTaskWorker("t-1", workerA); err != nil {
		t.Fatalf("set worker t-1: %v", err)
	}
	if err := q.CreateScanTask(&models.ScanTask{
		ID: "t-2", ProjectID: "proj-1", PlanID: "plan-1", RunID: &runID,
		Tool: "nuclei", Status: models.TaskRunning, CreatedAt: now,
	}); err != nil {
		t.Fatalf("task 2: %v", err)
	}
	if err := q.SetScanTaskWorker("t-2", workerA); err != nil {
		t.Fatalf("set worker t-2: %v", err)
	}
	if err := q.CreateScanTask(&models.ScanTask{
		ID: "t-3", ProjectID: "proj-1", PlanID: "plan-1", RunID: &runID,
		Tool: "nuclei", Status: models.TaskRunning, CreatedAt: now,
	}); err != nil {
		t.Fatalf("task 3: %v", err)
	}
	if err := q.SetScanTaskWorker("t-3", workerB); err != nil {
		t.Fatalf("set worker t-3: %v", err)
	}
	// Completed task should not count
	if err := q.CreateScanTask(&models.ScanTask{
		ID: "t-4", ProjectID: "proj-1", PlanID: "plan-1", RunID: &runID,
		Tool: "nuclei", Status: models.TaskCompleted, CreatedAt: now,
	}); err != nil {
		t.Fatalf("task 4: %v", err)
	}
	if err := q.SetScanTaskWorker("t-4", workerA); err != nil {
		t.Fatalf("set worker t-4: %v", err)
	}

	counts, err := q.CountRunningScanTasksPerWorker()
	if err != nil {
		t.Fatalf("CountRunningScanTasksPerWorker: %v", err)
	}
	if counts["worker-A"] != 2 {
		t.Errorf("worker-A = %d, want 2", counts["worker-A"])
	}
	if counts["worker-B"] != 1 {
		t.Errorf("worker-B = %d, want 1", counts["worker-B"])
	}
}

func TestCountOnlineWorkers(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC().Truncate(time.Second)

	count, err := q.CountOnlineWorkers()
	if err != nil {
		t.Fatalf("CountOnlineWorkers empty: %v", err)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}

	lastSeen := now
	// Online worker
	if err := q.CreateWorkerNode(&models.WorkerNode{
		ID: "w-1", Name: "worker-1", Endpoint: "http://localhost:9090",
		Mode: models.WorkerModeRemote, Status: models.WorkerStatusOnline,
		TrustLevel: "standard", MaxConcurrency: 4,
		LastSeen: &lastSeen, CreatedAt: now,
	}); err != nil {
		t.Fatalf("create w-1: %v", err)
	}
	// Busy worker
	if err := q.CreateWorkerNode(&models.WorkerNode{
		ID: "w-2", Name: "worker-2", Endpoint: "http://localhost:9091",
		Mode: models.WorkerModeRemote, Status: models.WorkerStatusBusy,
		TrustLevel: "standard", MaxConcurrency: 4,
		LastSeen: &lastSeen, CreatedAt: now,
	}); err != nil {
		t.Fatalf("create w-2: %v", err)
	}
	// Offline worker — should not count
	if err := q.CreateWorkerNode(&models.WorkerNode{
		ID: "w-3", Name: "worker-3", Endpoint: "http://localhost:9092",
		Mode: models.WorkerModeRemote, Status: models.WorkerStatusOffline,
		TrustLevel: "standard", MaxConcurrency: 4,
		LastSeen: &lastSeen, CreatedAt: now,
	}); err != nil {
		t.Fatalf("create w-3: %v", err)
	}

	count, err = q.CountOnlineWorkers()
	if err != nil {
		t.Fatalf("CountOnlineWorkers: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}

// --- 14. ListRecentRuns / ListRecentCompletedRunsByProject ---

func TestListRecentRuns(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC().Truncate(time.Second)

	if err := q.CreateProject(&models.Project{
		ID: "proj-1", Name: "test", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create project: %v", err)
	}

	for i := 1; i <= 3; i++ {
		if err := q.CreateRun(&models.Run{
			ID: fmtRunID(i), ProjectID: "proj-1", Name: "run",
			Status: models.RunRunning, CreatedAt: now.Add(time.Duration(i) * time.Minute),
		}); err != nil {
			t.Fatalf("create run %d: %v", i, err)
		}
	}

	list, err := q.ListRecentRuns(2)
	if err != nil {
		t.Fatalf("ListRecentRuns: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2, got %d", len(list))
	}
	// Verify DashboardRunItem fields
	if list[0].ProjectName != "test" {
		t.Errorf("ProjectName = %q, want %q", list[0].ProjectName, "test")
	}
	// DESC order: run-3 first
	if list[0].ID != fmtRunID(3) {
		t.Errorf("first ID = %q, want %q", list[0].ID, fmtRunID(3))
	}
}

func TestListRecentCompletedRunsByProject(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC().Truncate(time.Second)

	if err := q.CreateProject(&models.Project{
		ID: "proj-1", Name: "test", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create project: %v", err)
	}

	completedAt := now.Add(5 * time.Minute)

	// Running — should not appear
	createTestPipelineRun(t, q, "pr-run", "proj-1", "running", now)
	// Completed
	if err := q.CreatePipelineRun(&models.PipelineRun{
		ID: "pr-done", ProjectID: "proj-1", Mode: "standard",
		Status: "completed", Stage: "complete", EngineState: "stopped",
		StartedAt: now, CompletedAt: &completedAt, CreatedAt: now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("create completed pr: %v", err)
	}
	// Failed
	if err := q.CreatePipelineRun(&models.PipelineRun{
		ID: "pr-fail", ProjectID: "proj-1", Mode: "standard",
		Status: "failed", Stage: "scan", EngineState: "stopped",
		Error: "crash", StartedAt: now, CreatedAt: now.Add(2 * time.Minute),
	}); err != nil {
		t.Fatalf("create failed pr: %v", err)
	}

	list, err := q.ListRecentCompletedRunsByProject("proj-1", 10)
	if err != nil {
		t.Fatalf("ListRecentCompletedRunsByProject: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 (completed + failed), got %d", len(list))
	}
	// Running should not be included
	for _, r := range list {
		if r.Status == "running" {
			t.Errorf("running run %s should not be in completed list", r.ID)
		}
	}
}

// --- 15. GetPipelineRunStageStats ---

func TestGetPipelineRunStageStats(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC().Truncate(time.Second)

	if err := q.CreateProject(&models.Project{
		ID: "proj-1", Name: "test", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create project: %v", err)
	}

	createTestPipelineRun(t, q, "pr-1", "proj-1", "running", now)

	startedAt := now.Add(time.Second)
	completedAt := now.Add(11 * time.Second) // 10 second duration

	if err := q.CreatePipelineRunStage(&models.PipelineRunStage{
		ID: "stage-1", RunID: "pr-1", Stage: "discover",
		Status: models.StageStatusCompleted, StartedAt: &startedAt,
		CompletedAt: &completedAt, CreatedAt: now,
	}); err != nil {
		t.Fatalf("CreatePipelineRunStage: %v", err)
	}

	stats, err := q.GetPipelineRunStageStats("pr-1")
	if err != nil {
		t.Fatalf("GetPipelineRunStageStats: %v", err)
	}
	if len(stats) != 1 {
		t.Fatalf("expected 1 stat, got %d", len(stats))
	}
	if stats[0].Stage != "discover" {
		t.Errorf("Stage = %q, want %q", stats[0].Stage, "discover")
	}
	if stats[0].Status != "completed" {
		t.Errorf("Status = %q, want %q", stats[0].Status, "completed")
	}
	if stats[0].Duration < 9.0 || stats[0].Duration > 11.0 {
		t.Errorf("Duration = %.2f, want ~10.0", stats[0].Duration)
	}
}

func TestGetPipelineRunStageStats_Empty(t *testing.T) {
	q := New(openTestDB(t))

	stats, err := q.GetPipelineRunStageStats("nonexistent")
	if err != nil {
		t.Fatalf("GetPipelineRunStageStats: %v", err)
	}
	if len(stats) != 0 {
		t.Errorf("expected 0 stats, got %d", len(stats))
	}
}

// --- 16. ListRecentFindingsByStatus ---

func TestListRecentFindingsByStatus(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC().Truncate(time.Second)

	if err := q.CreateProject(&models.Project{
		ID: "proj-1", Name: "test", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create project: %v", err)
	}

	// pending_review findings
	if err := q.CreateFinding(&models.Finding{
		ID: "f-1", ProjectID: "proj-1", SourceTool: "nuclei",
		SourceRuleID: "r-1", DedupKey: "dk-1", Title: "SQL Injection",
		Severity: models.SeverityCritical, Confidence: 95, Priority: 95,
		Status: models.FindingPendingReview, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("CreateFinding f-1: %v", err)
	}
	if err := q.CreateFinding(&models.Finding{
		ID: "f-2", ProjectID: "proj-1", SourceTool: "nuclei",
		SourceRuleID: "r-2", DedupKey: "dk-2", Title: "XSS",
		Severity: models.SeverityHigh, Confidence: 80, Priority: 70,
		Status: models.FindingPendingReview, CreatedAt: now.Add(time.Minute), UpdatedAt: now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("CreateFinding f-2: %v", err)
	}
	// confirmed finding — should not appear
	if err := q.CreateFinding(&models.Finding{
		ID: "f-3", ProjectID: "proj-1", SourceTool: "nuclei",
		SourceRuleID: "r-3", DedupKey: "dk-3", Title: "Old Bug",
		Severity: models.SeverityMedium, Confidence: 60, Priority: 40,
		Status: models.FindingConfirmed, CreatedAt: now.Add(2 * time.Minute), UpdatedAt: now.Add(2 * time.Minute),
	}); err != nil {
		t.Fatalf("CreateFinding f-3: %v", err)
	}

	list, err := q.ListRecentFindingsByStatus(models.FindingPendingReview, 10)
	if err != nil {
		t.Fatalf("ListRecentFindingsByStatus: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 pending_review, got %d", len(list))
	}

	// Verify DashboardFindingItem fields
	if list[0].ProjectName != "test" {
		t.Errorf("ProjectName = %q, want %q", list[0].ProjectName, "test")
	}
	// Higher priority first (SQL Injection priority=95 > XSS priority=70)
	if list[0].Title != "SQL Injection" {
		t.Errorf("first title = %q, want %q (higher priority)", list[0].Title, "SQL Injection")
	}
}

func TestListRecentFindingsByStatus_Empty(t *testing.T) {
	q := New(openTestDB(t))

	list, err := q.ListRecentFindingsByStatus(models.FindingPendingReview, 10)
	if err != nil {
		t.Fatalf("ListRecentFindingsByStatus: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected 0, got %d", len(list))
	}
}
