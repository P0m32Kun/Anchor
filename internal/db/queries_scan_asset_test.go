package db

import (
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// --- Scan queries roundtrip ---

func setupScanTest(t *testing.T) (*Queries, string) {
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
	if err := q.CreateScanPlan(&models.ScanPlan{
		ID: "plan-1", ProjectID: "proj-1", WorkflowType: "manual",
		Profile: models.ProfileStandard, Status: "approved", CreatedBy: "test",
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("create scan plan: %v", err)
	}
	return q, "plan-1"
}

func TestCreateScanTask_RoundTrip(t *testing.T) {
	q, planID := setupScanTest(t)
	now := time.Now().UTC()

	task := &models.ScanTask{
		ID: "task-1", ProjectID: "proj-1", PlanID: planID,
		Tool: "nuclei", CommandTemplate: "nuclei -t test",
		Status: models.TaskQueued, CreatedAt: now,
	}
	if err := q.CreateScanTask(task); err != nil {
		t.Fatalf("CreateScanTask: %v", err)
	}

	got, err := q.GetScanTask("task-1")
	if err != nil {
		t.Fatalf("GetScanTask: %v", err)
	}
	if got == nil {
		t.Fatal("GetScanTask returned nil")
	}
	if got.Tool != "nuclei" {
		t.Errorf("tool = %q, want nuclei", got.Tool)
	}
	if got.Status != models.TaskQueued {
		t.Errorf("status = %q, want queued", got.Status)
	}
}

func TestUpdateScanTaskStatus(t *testing.T) {
	q, planID := setupScanTest(t)
	now := time.Now().UTC()

	q.CreateScanTask(&models.ScanTask{
		ID: "task-1", ProjectID: "proj-1", PlanID: planID,
		Tool: "nuclei", Status: models.TaskQueued, CreatedAt: now,
	})

	startedAt := now.Add(time.Second)
	if err := q.SetScanTaskRunning("task-1", startedAt); err != nil {
		t.Fatalf("SetScanTaskRunning: %v", err)
	}

	got, _ := q.GetScanTask("task-1")
	if got.Status != models.TaskRunning {
		t.Errorf("status = %q, want running", got.Status)
	}

	finishedAt := now.Add(10 * time.Second)
	exitCode := 0
	if err := q.UpdateScanTaskStatus("task-1", models.TaskCompleted, &exitCode, &finishedAt); err != nil {
		t.Fatalf("UpdateScanTaskStatus: %v", err)
	}

	got2, _ := q.GetScanTask("task-1")
	if got2.Status != models.TaskCompleted {
		t.Errorf("status = %q, want completed", got2.Status)
	}
	if got2.ExitCode == nil || *got2.ExitCode != 0 {
		t.Errorf("exit_code = %v, want 0", got2.ExitCode)
	}
}

func TestUpdateScanTaskErrorMessage(t *testing.T) {
	q, planID := setupScanTest(t)
	now := time.Now().UTC()

	q.CreateScanTask(&models.ScanTask{
		ID: "task-1", ProjectID: "proj-1", PlanID: planID,
		Tool: "nuclei", Status: models.TaskFailed, CreatedAt: now,
	})

	if err := q.UpdateScanTaskErrorMessage("task-1", "connection timeout"); err != nil {
		t.Fatalf("UpdateScanTaskErrorMessage: %v", err)
	}

	got, _ := q.GetScanTask("task-1")
	if got.ErrorMessage != "connection timeout" {
		t.Errorf("error_message = %q, want %q", got.ErrorMessage, "connection timeout")
	}
}

func TestListScanTasksByPlan(t *testing.T) {
	q, planID := setupScanTest(t)
	now := time.Now().UTC()

	for i := 0; i < 3; i++ {
		q.CreateScanTask(&models.ScanTask{
			ID: "task-" + string(rune('a'+i)), ProjectID: "proj-1", PlanID: planID,
			Tool: "nuclei", Status: models.TaskQueued, CreatedAt: now,
		})
	}

	tasks, err := q.ListScanTasksByPlan(planID)
	if err != nil {
		t.Fatalf("ListScanTasksByPlan: %v", err)
	}
	if len(tasks) != 3 {
		t.Errorf("len = %d, want 3", len(tasks))
	}
}

func TestResetScanTaskForRetry(t *testing.T) {
	q, planID := setupScanTest(t)
	now := time.Now().UTC()

	q.CreateScanTask(&models.ScanTask{
		ID: "task-1", ProjectID: "proj-1", PlanID: planID,
		Tool: "nuclei", Status: models.TaskFailed,
		ErrorMessage: "timeout", CreatedAt: now,
	})

	if err := q.ResetScanTaskForRetry("task-1"); err != nil {
		t.Fatalf("ResetScanTaskForRetry: %v", err)
	}

	got, _ := q.GetScanTask("task-1")
	if got.Status != models.TaskCreated {
		t.Errorf("status = %q, want created", got.Status)
	}
	if got.ErrorMessage != "" {
		t.Errorf("error_message = %q, want empty", got.ErrorMessage)
	}
}

// --- Run queries ---

func TestCreateRun_RoundTrip(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC()
	q.CreateProject(&models.Project{
		ID: "proj-1", Name: "test", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	})

	run := &models.Run{
		ID: "run-1", ProjectID: "proj-1", Name: "test-run",
		Status: "running", StartedAt: &now, CreatedAt: now,
	}
	if err := q.CreateRun(run); err != nil {
		t.Fatalf("CreateRun: %v", err)
	}

	got, err := q.GetRun("run-1")
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if got == nil {
		t.Fatal("GetRun returned nil")
	}
	if got.Name != "test-run" {
		t.Errorf("name = %q, want test-run", got.Name)
	}
}

func TestListRunsByProject(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC()
	q.CreateProject(&models.Project{
		ID: "proj-1", Name: "test", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	})

	for i := 0; i < 3; i++ {
		q.CreateRun(&models.Run{
			ID: "run-" + string(rune('a'+i)), ProjectID: "proj-1",
			Status: "completed", CreatedAt: now,
		})
	}

	runs, err := q.ListRunsByProject("proj-1")
	if err != nil {
		t.Fatalf("ListRunsByProject: %v", err)
	}
	if len(runs) != 3 {
		t.Errorf("len = %d, want 3", len(runs))
	}
}

func TestUpdateRunStatus(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC()
	q.CreateProject(&models.Project{
		ID: "proj-1", Name: "test", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	})

	q.CreateRun(&models.Run{
		ID: "run-1", ProjectID: "proj-1",
		Status: "running", CreatedAt: now,
	})

	finishedAt := now.Add(5 * time.Minute)
	if err := q.UpdateRunStatus("run-1", "completed", nil, &finishedAt); err != nil {
		t.Fatalf("UpdateRunStatus: %v", err)
	}

	got, _ := q.GetRun("run-1")
	if got.Status != "completed" {
		t.Errorf("status = %q, want completed", got.Status)
	}
}

// --- Asset queries roundtrip ---

func TestCreateAsset_RoundTrip(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC()
	q.CreateProject(&models.Project{
		ID: "proj-1", Name: "test", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	})

	asset := &models.Asset{
		ID: util.GenerateID(), ProjectID: "proj-1",
		Type: models.AssetTypeDomain, Value: "example.com",
		NormalizedValue: "example.com",
		SourceTools:     []string{"subfinder"},
		FirstSeen: now, LastSeen: now,
	}
	if err := q.CreateAsset(asset); err != nil {
		t.Fatalf("CreateAsset: %v", err)
	}

	got, err := q.GetAssetByNormalizedValue("proj-1", "example.com")
	if err != nil {
		t.Fatalf("GetAssetByNormalizedValue: %v", err)
	}
	if got == nil {
		t.Fatal("GetAssetByNormalizedValue returned nil")
	}
	if got.Value != "example.com" {
		t.Errorf("value = %q, want example.com", got.Value)
	}
}

func TestListAssetsByProject(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC()
	q.CreateProject(&models.Project{
		ID: "proj-1", Name: "test", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	})

	for _, v := range []string{"a.com", "b.com", "c.com"} {
		q.CreateAsset(&models.Asset{
			ID: util.GenerateID(), ProjectID: "proj-1",
			Type: models.AssetTypeDomain, Value: v,
			NormalizedValue: v, SourceTools: []string{"subfinder"},
			FirstSeen: now, LastSeen: now,
		})
	}

	assets, err := q.ListAssetsByProject("proj-1")
	if err != nil {
		t.Fatalf("ListAssetsByProject: %v", err)
	}
	if len(assets) != 3 {
		t.Errorf("len = %d, want 3", len(assets))
	}
}

func TestCountAssetsByProject(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC()
	q.CreateProject(&models.Project{
		ID: "proj-1", Name: "test", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	})

	for _, v := range []string{"a.com", "b.com"} {
		q.CreateAsset(&models.Asset{
			ID: util.GenerateID(), ProjectID: "proj-1",
			Type: models.AssetTypeDomain, Value: v,
			NormalizedValue: v, SourceTools: []string{"subfinder"},
			FirstSeen: now, LastSeen: now,
		})
	}

	count, err := q.CountAssetsByProject("proj-1")
	if err != nil {
		t.Fatalf("CountAssetsByProject: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}

// --- WebEndpoint queries ---

func setupWebEndpointTest(t *testing.T) (*Queries, string) {
	t.Helper()
	q := New(openTestDB(t))
	now := time.Now().UTC()
	q.CreateProject(&models.Project{
		ID: "proj-1", Name: "test", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	})
	// Create asset for FK constraint.
	q.CreateAsset(&models.Asset{
		ID: "asset-1", ProjectID: "proj-1",
		Type: models.AssetTypeDomain, Value: "example.com",
		NormalizedValue: "example.com", SourceTools: []string{"httpx"},
		FirstSeen: now, LastSeen: now,
	})
	return q, "proj-1"
}

func TestCreateWebEndpoint_Upsert(t *testing.T) {
	q, _ := setupWebEndpointTest(t)
	now := time.Now().UTC()

	ep := &models.WebEndpoint{
		ID: util.GenerateID(), ProjectID: "proj-1", AssetID: "asset-1",
		URL: "https://example.com", Scheme: "https", Host: "example.com",
		SourceTool: "httpx", CreatedAt: now,
	}
	if err := q.CreateWebEndpoint(ep); err != nil {
		t.Fatalf("CreateWebEndpoint: %v", err)
	}

	// Second insert with same URL should upsert (no error).
	ep2 := &models.WebEndpoint{
		ID: util.GenerateID(), ProjectID: "proj-1", AssetID: "asset-1",
		URL: "https://example.com", Scheme: "https", Host: "example.com",
		Title: "Updated Title", SourceTool: "httpx", CreatedAt: now,
	}
	if err := q.CreateWebEndpoint(ep2); err != nil {
		t.Fatalf("CreateWebEndpoint upsert: %v", err)
	}

	eps, err := q.ListWebEndpointsByProject("proj-1")
	if err != nil {
		t.Fatalf("ListWebEndpointsByProject: %v", err)
	}
	if len(eps) != 1 {
		t.Errorf("len = %d, want 1 (upsert)", len(eps))
	}
}

func TestWebEndpointExists(t *testing.T) {
	q, _ := setupWebEndpointTest(t)
	now := time.Now().UTC()

	exists, err := q.WebEndpointExists("proj-1", "https://example.com")
	if err != nil {
		t.Fatalf("WebEndpointExists: %v", err)
	}
	if exists {
		t.Error("expected false for non-existent endpoint")
	}

	q.CreateWebEndpoint(&models.WebEndpoint{
		ID: util.GenerateID(), ProjectID: "proj-1", AssetID: "asset-1",
		URL: "https://example.com", Scheme: "https", Host: "example.com",
		SourceTool: "httpx", CreatedAt: now,
	})

	exists, err = q.WebEndpointExists("proj-1", "https://example.com")
	if err != nil {
		t.Fatalf("WebEndpointExists: %v", err)
	}
	if !exists {
		t.Error("expected true for existing endpoint")
	}
}

// --- RawArtifact queries ---

func TestCreateRawArtifact_RoundTrip(t *testing.T) {
	q, planID := setupScanTest(t)
	now := time.Now().UTC()

	q.CreateScanTask(&models.ScanTask{
		ID: "task-1", ProjectID: "proj-1", PlanID: planID,
		Tool: "nuclei", Status: models.TaskCompleted, CreatedAt: now,
	})

	art := &models.RawArtifact{
		ID: util.GenerateID(), ProjectID: "proj-1", TaskID: strPtr("task-1"),
		Type: models.ArtifactStdout, Path: "/tmp/stdout.txt",
		SHA256: "abc123", Size: 1024, RedactionStatus: "unchecked",
		CreatedAt: now,
	}
	if err := q.CreateRawArtifact(art); err != nil {
		t.Fatalf("CreateRawArtifact: %v", err)
	}

	arts, err := q.ListRawArtifactsByTask("task-1")
	if err != nil {
		t.Fatalf("ListRawArtifactsByTask: %v", err)
	}
	if len(arts) != 1 {
		t.Fatalf("len = %d, want 1", len(arts))
	}
	if arts[0].Type != models.ArtifactStdout {
		t.Errorf("type = %q, want stdout", arts[0].Type)
	}
	if arts[0].Size != 1024 {
		t.Errorf("size = %d, want 1024", arts[0].Size)
	}
}
