package db

import (
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
)

// --- ListScanTasksByPlan ---

func TestListScanTasksByPlan_Coverage(t *testing.T) {
	q, _, _ := setupScanTestData(t)
	now := time.Now().UTC().Truncate(time.Second)

	for i := 0; i < 3; i++ {
		if err := q.CreateScanTask(&models.ScanTask{
			ID: "task-plan-" + string(rune('a'+i)), ProjectID: "proj-1",
			PlanID: "plan-1", Tool: "nuclei",
			CommandTemplate: "nuclei -u {{target}}", Status: models.TaskCreated,
			CreatedAt: now.Add(time.Duration(i) * time.Second),
		}); err != nil {
			t.Fatalf("create task %d: %v", i, err)
		}
	}

	tasks, err := q.ListScanTasksByPlan("plan-1")
	if err != nil {
		t.Fatalf("ListScanTasksByPlan: %v", err)
	}
	if len(tasks) != 3 {
		t.Fatalf("expected 3, got %d", len(tasks))
	}

	// Empty plan
	empty, err := q.ListScanTasksByPlan("nonexistent")
	if err != nil {
		t.Fatalf("ListScanTasksByPlan empty: %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("expected 0, got %d", len(empty))
	}
}

// --- ListScanTasksByRun ---

func TestListScanTasksByRun(t *testing.T) {
	q, _, runID := setupScanTestData(t)
	now := time.Now().UTC().Truncate(time.Second)

	for i := 0; i < 2; i++ {
		runPtr := &runID
		if err := q.CreateScanTask(&models.ScanTask{
			ID: "task-run-" + string(rune('a'+i)), ProjectID: "proj-1",
			PlanID: "plan-1", RunID: runPtr, Tool: "nuclei",
			CommandTemplate: "nuclei -u {{target}}", Status: models.TaskCreated,
			CreatedAt: now.Add(time.Duration(i) * time.Second),
		}); err != nil {
			t.Fatalf("create task %d: %v", i, err)
		}
	}

	tasks, err := q.ListScanTasksByRun(runID)
	if err != nil {
		t.Fatalf("ListScanTasksByRun: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("expected 2, got %d", len(tasks))
	}

	// Verify nullable fields are handled
	for _, t2 := range tasks {
		if t2.RunID == nil || *t2.RunID != runID {
			t.Errorf("run_id = %v, want %q", t2.RunID, runID)
		}
	}
}

func TestListScanTasksByRun_WithNullableFields(t *testing.T) {
	q, _, runID := setupScanTestData(t)
	now := time.Now().UTC().Truncate(time.Second)
	started := now.Add(time.Second)
	finished := now.Add(10 * time.Second)
	exitCode := 0
	workerID := "worker-1"
	customVersion := "v1.0"
	depTaskID := "task-dep"
	targetID := "tgt-1"
	errMsg := "some error"

	// Create target first (FK on target_id)
	if err := q.CreateTarget(&models.Target{
		ID: targetID, ProjectID: "proj-1",
		Type: models.TargetTypeDomain, Value: "example.com",
		Source: "manual", Status: "active", CreatedAt: now,
	}); err != nil {
		t.Fatalf("create target: %v", err)
	}

	// Create dependency task first (FK on depends_on_task_id)
	runPtr := &runID
	if err := q.CreateScanTask(&models.ScanTask{
		ID: depTaskID, ProjectID: "proj-1", PlanID: "plan-1", RunID: runPtr,
		Tool: "nuclei", Status: models.TaskCompleted, CreatedAt: now,
	}); err != nil {
		t.Fatalf("create dep task: %v", err)
	}

	if err := q.CreateScanTask(&models.ScanTask{
		ID: "task-full", ProjectID: "proj-1",
		PlanID: "plan-1", RunID: runPtr, DependsOnTaskID: &depTaskID,
		TargetID: &targetID, Tool: "nuclei",
		CommandTemplate: "nuclei -u {{target}}", ArgumentsRedacted: "-u [redacted]",
		Status:                    models.TaskCreated,
		NucleiCustomBundleVersion: &customVersion,
		CreatedAt:                 now,
	}); err != nil {
		t.Fatalf("create task: %v", err)
	}
	// Set worker_id
	if err := q.SetScanTaskWorker("task-full", workerID); err != nil {
		t.Fatalf("SetScanTaskWorker: %v", err)
	}
	// Set status=running and started_at
	if err := q.SetScanTaskRunning("task-full", started); err != nil {
		t.Fatalf("SetScanTaskRunning: %v", err)
	}
	// Set status=completed, exit_code, finished_at
	if err := q.UpdateScanTaskStatus("task-full", models.TaskCompleted, &exitCode, &finished); err != nil {
		t.Fatalf("UpdateScanTaskStatus: %v", err)
	}
	// Set error_message
	if err := q.UpdateScanTaskErrorMessage("task-full", errMsg); err != nil {
		t.Fatalf("UpdateScanTaskErrorMessage: %v", err)
	}

	tasks, err := q.ListScanTasksByRun(runID)
	if err != nil {
		t.Fatalf("ListScanTasksByRun: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("expected 2, got %d", len(tasks))
	}

	// Find the "task-full" entry (order is non-deterministic with same created_at)
	var tk *models.ScanTask
	for _, t2 := range tasks {
		if t2.ID == "task-full" {
			tk = t2
			break
		}
	}
	if tk == nil {
		t.Fatal("task-full not found in results")
	}
	if tk.DependsOnTaskID == nil || *tk.DependsOnTaskID != depTaskID {
		t.Errorf("depends_on_task_id = %v, want %q", tk.DependsOnTaskID, depTaskID)
	}
	if tk.TargetID == nil || *tk.TargetID != targetID {
		t.Errorf("target_id = %v, want %q", tk.TargetID, targetID)
	}
	if tk.WorkerID == nil || *tk.WorkerID != workerID {
		t.Errorf("worker_id = %v, want %q", tk.WorkerID, workerID)
	}
	if tk.NucleiCustomBundleVersion == nil || *tk.NucleiCustomBundleVersion != customVersion {
		t.Errorf("nuclei_custom_bundle_version = %v, want %q", tk.NucleiCustomBundleVersion, customVersion)
	}
	if tk.ErrorMessage != errMsg {
		t.Errorf("error_message = %q, want %q", tk.ErrorMessage, errMsg)
	}
	if tk.StartedAt == nil {
		t.Error("expected started_at to be set")
	}
	if tk.FinishedAt == nil {
		t.Error("expected finished_at to be set")
	}
	if tk.ExitCode == nil || *tk.ExitCode != 0 {
		t.Errorf("exit_code = %v, want 0", tk.ExitCode)
	}
}

// --- GetRun ---

func TestGetRun_Found(t *testing.T) {
	q, _, _ := setupScanTestData(t)

	got, err := q.GetRun("run-1")
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if got == nil {
		t.Fatal("expected run, got nil")
	}
	if got.Name != "test-run" {
		t.Errorf("name = %q, want test-run", got.Name)
	}
}

func TestGetRun_NotFound(t *testing.T) {
	q := New(openTestDB(t))

	got, err := q.GetRun("nonexistent")
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if got != nil {
		t.Error("expected nil for nonexistent run")
	}
}

func TestGetRun_WithTemplateID(t *testing.T) {
	q, _, _ := setupScanTestData(t)
	now := time.Now().UTC().Truncate(time.Second)

	// Create tool_template first (FK on tool_template_id)
	_, err := q.db.Exec(`INSERT INTO tool_templates (id, name, description, profile_type, tools_json, default_max_concurrency, screenshot_enabled, directory_bruteforce_enabled, nuclei_severity_filter, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"tmpl-1", "test", "test template", "external", "[]", 5, false, false, "medium", now, now)
	if err != nil {
		t.Fatalf("create tool_template: %v", err)
	}

	templateID := "tmpl-1"
	if err := q.CreateRun(&models.Run{
		ID: "run-tmpl", ProjectID: "proj-1", Name: "with-template",
		ToolTemplateID: &templateID, Status: models.RunRunning, CreatedAt: now,
	}); err != nil {
		t.Fatalf("CreateRun: %v", err)
	}

	got, err := q.GetRun("run-tmpl")
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if got.ToolTemplateID == nil || *got.ToolTemplateID != "tmpl-1" {
		t.Errorf("tool_template_id = %v, want tmpl-1", got.ToolTemplateID)
	}
}

// --- ListRunsByProject ---

func TestListRunsByProject_Coverage(t *testing.T) {
	q, projID, _ := setupScanTestData(t)
	now := time.Now().UTC().Truncate(time.Second)

	for i := 2; i <= 4; i++ {
		q.CreateRun(&models.Run{
			ID: "run-" + string(rune('0'+i)), ProjectID: projID,
			Name: "run", Status: models.RunPending,
			CreatedAt: now.Add(time.Duration(i) * time.Minute),
		})
	}

	list, err := q.ListRunsByProject(projID)
	if err != nil {
		t.Fatalf("ListRunsByProject: %v", err)
	}
	if len(list) != 4 {
		t.Fatalf("expected 4, got %d", len(list))
	}

	// Empty project
	empty, err := q.ListRunsByProject("nonexistent")
	if err != nil {
		t.Fatalf("ListRunsByProject empty: %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("expected 0, got %d", len(empty))
	}
}

// --- ListPipelineRunsByProject ---

func TestListPipelineRunsByProject_Coverage(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC().Truncate(time.Second)

	q.CreateProject(&models.Project{
		ID: "proj-pr", Name: "pr-test", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	})

	createTestPipelineRun(t, q, "pr-1", "proj-pr", "running", now)
	createTestPipelineRun(t, q, "pr-2", "proj-pr", "completed", now.Add(time.Minute))

	list, err := q.ListPipelineRunsByProject("proj-pr")
	if err != nil {
		t.Fatalf("ListPipelineRunsByProject: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2, got %d", len(list))
	}

	// Empty project
	empty, err := q.ListPipelineRunsByProject("nonexistent")
	if err != nil {
		t.Fatalf("ListPipelineRunsByProject empty: %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("expected 0, got %d", len(empty))
	}
}

// --- ListPipelineRunsByStatus ---

func TestListPipelineRunsByStatus_Filtered(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC().Truncate(time.Second)

	q.CreateProject(&models.Project{
		ID: "proj-ps", Name: "ps-test", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	})

	createTestPipelineRun(t, q, "pr-1", "proj-ps", "running", now)
	createTestPipelineRun(t, q, "pr-2", "proj-ps", "completed", now.Add(time.Minute))
	createTestPipelineRun(t, q, "pr-3", "proj-ps", "running", now.Add(2*time.Minute))

	list, err := q.ListPipelineRunsByStatus("running")
	if err != nil {
		t.Fatalf("ListPipelineRunsByStatus: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 running, got %d", len(list))
	}

	// No matches
	empty, err := q.ListPipelineRunsByStatus("cancelled")
	if err != nil {
		t.Fatalf("ListPipelineRunsByStatus cancelled: %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("expected 0, got %d", len(empty))
	}
}

// --- ListPipelineRunsByProjectPaginated ---

func TestListPipelineRunsByProjectPaginated_EmptyProject(t *testing.T) {
	q := New(openTestDB(t))

	list, err := q.ListPipelineRunsByProjectPaginated("nonexistent", 10, 0)
	if err != nil {
		t.Fatalf("ListPipelineRunsByProjectPaginated: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected 0, got %d", len(list))
	}
}

// --- ListRecentRuns ---

func TestListRecentRuns_WithProjectName(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC().Truncate(time.Second)

	q.CreateProject(&models.Project{
		ID: "proj-rr", Name: "my-project", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	})

	for i := 1; i <= 3; i++ {
		q.CreateRun(&models.Run{
			ID: "run-" + string(rune('0'+i)), ProjectID: "proj-rr",
			Name: "run", Status: models.RunRunning,
			StartedAt: timePtr(now.Add(time.Duration(i) * time.Minute)),
			CreatedAt:  now.Add(time.Duration(i) * time.Minute),
		})
	}

	list, err := q.ListRecentRuns(2)
	if err != nil {
		t.Fatalf("ListRecentRuns: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2, got %d", len(list))
	}
	if list[0].ProjectName != "my-project" {
		t.Errorf("project_name = %q, want my-project", list[0].ProjectName)
	}
	if list[0].StartedAt == nil {
		t.Error("expected started_at to be set")
	}
}

func TestListRecentRuns_ZeroLimit(t *testing.T) {
	q := New(openTestDB(t))

	list, err := q.ListRecentRuns(0)
	if err != nil {
		t.Fatalf("ListRecentRuns(0): %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected 0, got %d", len(list))
	}
}

func timePtr(t time.Time) *time.Time { return &t }
