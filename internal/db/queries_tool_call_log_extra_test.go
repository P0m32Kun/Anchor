package db

import (
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
)

// createToolCallLogPrereqs creates the minimum prerequisite rows for tool_call_logs:
// project -> scan_plan -> pipeline_run -> scan_task.
func createToolCallLogPrereqs(t *testing.T, q *Queries) {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Second)

	p := &models.Project{
		ID:             "tcl-proj",
		Name:           "tcl-project",
		Organization:   "test-org",
		Purpose:        "testing",
		RateLimit:      100,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := q.CreateProject(p); err != nil {
		t.Fatalf("create project: %v", err)
	}

	plan := &models.ScanPlan{
		ID:           "tcl-plan",
		ProjectID:    "tcl-proj",
		WorkflowType: "full",
		Profile:      models.ProfileStandard,
		Status:       "approved",
		CreatedBy:    "test",
		CreatedAt:    now,
	}
	if err := q.CreateScanPlan(plan); err != nil {
		t.Fatalf("create scan plan: %v", err)
	}

	run := &models.PipelineRun{
		ID:        "tcl-run",
		ProjectID: "tcl-proj",
		Mode:      "quick",
		Status:    "running",
		Stage:     "scan",
		StartedAt: now,
		CreatedAt: now,
	}
	if err := q.CreatePipelineRun(run); err != nil {
		t.Fatalf("create pipeline run: %v", err)
	}

	task := &models.ScanTask{
		ID:              "tcl-task",
		ProjectID:       "tcl-proj",
		PlanID:          "tcl-plan",
		RunID:           strPtr("tcl-run"),
		Tool:            "nuclei",
		CommandTemplate: "nuclei -u {{target}}",
		Status:          models.TaskRunning,
		CreatedAt:       now,
	}
	if err := q.CreateScanTask(task); err != nil {
		t.Fatalf("create scan task: %v", err)
	}
}

func TestCreateToolCallLog_GetToolCallLog_RoundTrip(t *testing.T) {
	q := New(openTestDB(t))
	createToolCallLogPrereqs(t, q)
	now := time.Now().UTC().Truncate(time.Second)

	l := &models.ToolCallLog{
		ID:         "tcl-1",
		RunID:      "tcl-run",
		WorkItemID: strPtr("wi-1"),
		TaskID:     strPtr("tcl-task"),
		Tool:       "nuclei",
		Action:     "scan",
		AssetID:    strPtr("asset-1"),
		ParamsJSON: `{"target":"example.com","severity":"high"}`,
		StartedAt:  now,
		Status:     models.ToolCallRunning,
		CreatedAt:  now,
	}
	if err := q.CreateToolCallLog(l); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := q.GetToolCallLog("tcl-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got == nil {
		t.Fatal("get returned nil for existing row")
	}
	if got.ID != "tcl-1" {
		t.Errorf("id: want tcl-1, got %q", got.ID)
	}
	if got.RunID != "tcl-run" {
		t.Errorf("run_id: want tcl-run, got %q", got.RunID)
	}
	if got.WorkItemID == nil || *got.WorkItemID != "wi-1" {
		t.Errorf("work_item_id: want wi-1, got %v", got.WorkItemID)
	}
	if got.TaskID == nil || *got.TaskID != "tcl-task" {
		t.Errorf("task_id: want tcl-task, got %v", got.TaskID)
	}
	if got.Tool != "nuclei" {
		t.Errorf("tool: want nuclei, got %q", got.Tool)
	}
	if got.Action != "scan" {
		t.Errorf("action: want scan, got %q", got.Action)
	}
	if got.AssetID == nil || *got.AssetID != "asset-1" {
		t.Errorf("asset_id: want asset-1, got %v", got.AssetID)
	}
	if got.Status != models.ToolCallRunning {
		t.Errorf("status: want running, got %q", got.Status)
	}
	if got.FinishedAt != nil {
		t.Errorf("finished_at: want nil, got %v", got.FinishedAt)
	}
	if got.DurationMs != nil {
		t.Errorf("duration_ms: want nil, got %v", got.DurationMs)
	}
	if got.ExitCode != nil {
		t.Errorf("exit_code: want nil, got %v", got.ExitCode)
	}
	if got.OutputSummary != nil {
		t.Errorf("output_summary: want nil, got %v", got.OutputSummary)
	}
	if got.ErrorMessage != nil {
		t.Errorf("error_message: want nil, got %v", got.ErrorMessage)
	}
}

func TestGetToolCallLog_NotFoundReturnsNil(t *testing.T) {
	q := New(openTestDB(t))

	got, err := q.GetToolCallLog("nonexistent")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got != nil {
		t.Errorf("want nil, got %+v", got)
	}
}

func TestUpdateToolCallLogOnComplete(t *testing.T) {
	q := New(openTestDB(t))
	createToolCallLogPrereqs(t, q)
	now := time.Now().UTC().Truncate(time.Second)

	l := &models.ToolCallLog{
		ID:        "tcl-complete",
		RunID:     "tcl-run",
		Tool:      "nuclei",
		Action:    "scan",
		StartedAt: now,
		Status:    models.ToolCallRunning,
		CreatedAt: now,
	}
	if err := q.CreateToolCallLog(l); err != nil {
		t.Fatalf("create: %v", err)
	}

	finishedAt := now.Add(30 * time.Second)
	exitCode := 0
	if err := q.UpdateToolCallLogOnComplete("tcl-complete", finishedAt, &exitCode, models.ToolCallCompleted, 30000, "found 5 issues", ""); err != nil {
		t.Fatalf("update on complete: %v", err)
	}

	got, err := q.GetToolCallLog("tcl-complete")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.FinishedAt == nil || !got.FinishedAt.Equal(finishedAt) {
		t.Errorf("finished_at: want %v, got %v", finishedAt, got.FinishedAt)
	}
	if got.ExitCode == nil || *got.ExitCode != 0 {
		t.Errorf("exit_code: want 0, got %v", got.ExitCode)
	}
	if got.Status != models.ToolCallCompleted {
		t.Errorf("status: want completed, got %q", got.Status)
	}
	if got.DurationMs == nil || *got.DurationMs != 30000 {
		t.Errorf("duration_ms: want 30000, got %v", got.DurationMs)
	}
	if got.OutputSummary == nil || *got.OutputSummary != "found 5 issues" {
		t.Errorf("output_summary: want 'found 5 issues', got %v", got.OutputSummary)
	}
	if got.ErrorMessage != nil && *got.ErrorMessage != "" {
		t.Errorf("error_message: want empty, got %q", *got.ErrorMessage)
	}
}

func TestUpdateToolCallLogOnComplete_Failure(t *testing.T) {
	q := New(openTestDB(t))
	createToolCallLogPrereqs(t, q)
	now := time.Now().UTC().Truncate(time.Second)

	l := &models.ToolCallLog{
		ID:        "tcl-fail",
		RunID:     "tcl-run",
		Tool:      "katana",
		Action:    "crawl",
		StartedAt: now,
		Status:    models.ToolCallRunning,
		CreatedAt: now,
	}
	if err := q.CreateToolCallLog(l); err != nil {
		t.Fatalf("create: %v", err)
	}

	finishedAt := now.Add(10 * time.Second)
	exitCode := 1
	if err := q.UpdateToolCallLogOnComplete("tcl-fail", finishedAt, &exitCode, models.ToolCallFailed, 10000, "", "timeout exceeded"); err != nil {
		t.Fatalf("update on complete: %v", err)
	}

	got, err := q.GetToolCallLog("tcl-fail")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status != models.ToolCallFailed {
		t.Errorf("status: want failed, got %q", got.Status)
	}
	if got.ExitCode == nil || *got.ExitCode != 1 {
		t.Errorf("exit_code: want 1, got %v", got.ExitCode)
	}
	if got.ErrorMessage == nil || *got.ErrorMessage != "timeout exceeded" {
		t.Errorf("error_message: want 'timeout exceeded', got %v", got.ErrorMessage)
	}
}

func TestUpdateToolCallLogTaskID(t *testing.T) {
	q := New(openTestDB(t))
	createToolCallLogPrereqs(t, q)
	now := time.Now().UTC().Truncate(time.Second)

	l := &models.ToolCallLog{
		ID:        "tcl-link",
		RunID:     "tcl-run",
		Tool:      "nuclei",
		Action:    "scan",
		StartedAt: now,
		Status:    models.ToolCallRunning,
		CreatedAt: now,
	}
	if err := q.CreateToolCallLog(l); err != nil {
		t.Fatalf("create: %v", err)
	}

	// Verify task_id is initially nil
	got, err := q.GetToolCallLog("tcl-link")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.TaskID != nil {
		t.Errorf("task_id before link: want nil, got %v", got.TaskID)
	}

	if err := q.UpdateToolCallLogTaskID("tcl-link", "tcl-task"); err != nil {
		t.Fatalf("update task id: %v", err)
	}

	got2, err := q.GetToolCallLog("tcl-link")
	if err != nil {
		t.Fatalf("get after link: %v", err)
	}
	if got2.TaskID == nil || *got2.TaskID != "tcl-task" {
		t.Errorf("task_id after link: want tcl-task, got %v", got2.TaskID)
	}
}

func TestListToolCallLogsByRun(t *testing.T) {
	q := New(openTestDB(t))
	createToolCallLogPrereqs(t, q)
	now := time.Now().UTC().Truncate(time.Second)

	for i, id := range []string{"tcl-r1", "tcl-r2", "tcl-r3"} {
		l := &models.ToolCallLog{
			ID:        id,
			RunID:     "tcl-run",
			Tool:      "nuclei",
			Action:    "scan",
			StartedAt: now.Add(time.Duration(i) * time.Minute),
			Status:    models.ToolCallRunning,
			CreatedAt: now,
		}
		if err := q.CreateToolCallLog(l); err != nil {
			t.Fatalf("create %s: %v", id, err)
		}
	}

	list, err := q.ListToolCallLogsByRun("tcl-run")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("len: want 3, got %d", len(list))
	}
	// Ordered by started_at
	wantOrder := []string{"tcl-r1", "tcl-r2", "tcl-r3"}
	for i, want := range wantOrder {
		if list[i].ID != want {
			t.Errorf("position %d: want %q, got %q", i, want, list[i].ID)
		}
	}
}

func TestListToolCallLogsByRunPaginated(t *testing.T) {
	q := New(openTestDB(t))
	createToolCallLogPrereqs(t, q)
	now := time.Now().UTC().Truncate(time.Second)

	for i, c := range []rune{'a', 'b', 'c', 'd', 'e'} {
		l := &models.ToolCallLog{
			ID:        "tcl-pg-" + string(c),
			RunID:     "tcl-run",
			Tool:      "nuclei",
			Action:    "scan",
			StartedAt: now.Add(time.Duration(i) * time.Minute),
			Status:    models.ToolCallRunning,
			CreatedAt: now,
		}
		if err := q.CreateToolCallLog(l); err != nil {
			t.Fatalf("create pg-%c: %v", c, err)
		}
	}

	// Page 1: limit 2, offset 0
	page1, err := q.ListToolCallLogsByRunPaginated("tcl-run", 2, 0)
	if err != nil {
		t.Fatalf("page1: %v", err)
	}
	if len(page1) != 2 {
		t.Fatalf("page1 len: want 2, got %d", len(page1))
	}
	if page1[0].ID != "tcl-pg-a" {
		t.Errorf("page1[0]: want tcl-pg-a, got %q", page1[0].ID)
	}
	if page1[1].ID != "tcl-pg-b" {
		t.Errorf("page1[1]: want tcl-pg-b, got %q", page1[1].ID)
	}

	// Page 2: limit 2, offset 2
	page2, err := q.ListToolCallLogsByRunPaginated("tcl-run", 2, 2)
	if err != nil {
		t.Fatalf("page2: %v", err)
	}
	if len(page2) != 2 {
		t.Fatalf("page2 len: want 2, got %d", len(page2))
	}
	if page2[0].ID != "tcl-pg-c" {
		t.Errorf("page2[0]: want tcl-pg-c, got %q", page2[0].ID)
	}

	// Page 3: limit 2, offset 4 (last item)
	page3, err := q.ListToolCallLogsByRunPaginated("tcl-run", 2, 4)
	if err != nil {
		t.Fatalf("page3: %v", err)
	}
	if len(page3) != 1 {
		t.Fatalf("page3 len: want 1, got %d", len(page3))
	}
	if page3[0].ID != "tcl-pg-e" {
		t.Errorf("page3[0]: want tcl-pg-e, got %q", page3[0].ID)
	}
}

func TestCountToolCallLogsByRun(t *testing.T) {
	q := New(openTestDB(t))
	createToolCallLogPrereqs(t, q)
	now := time.Now().UTC().Truncate(time.Second)

	count, err := q.CountToolCallLogsByRun("tcl-run")
	if err != nil {
		t.Fatalf("count empty: %v", err)
	}
	if count != 0 {
		t.Errorf("count empty: want 0, got %d", count)
	}

	for i, c := range []rune{'a', 'b', 'c'} {
		l := &models.ToolCallLog{
			ID:        "tcl-cnt-" + string(c),
			RunID:     "tcl-run",
			Tool:      "nuclei",
			Action:    "scan",
			StartedAt: now.Add(time.Duration(i) * time.Minute),
			Status:    models.ToolCallRunning,
			CreatedAt: now,
		}
		if err := q.CreateToolCallLog(l); err != nil {
			t.Fatalf("create: %v", err)
		}
	}

	count, err = q.CountToolCallLogsByRun("tcl-run")
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 3 {
		t.Errorf("count: want 3, got %d", count)
	}
}

func TestListToolCallLogsByTaskID(t *testing.T) {
	q := New(openTestDB(t))
	createToolCallLogPrereqs(t, q)
	now := time.Now().UTC().Truncate(time.Second)

	l := &models.ToolCallLog{
		ID:        "tcl-task-filter",
		RunID:     "tcl-run",
		TaskID:    strPtr("tcl-task"),
		Tool:      "nuclei",
		Action:    "scan",
		StartedAt: now,
		Status:    models.ToolCallRunning,
		CreatedAt: now,
	}
	if err := q.CreateToolCallLog(l); err != nil {
		t.Fatalf("create: %v", err)
	}

	// Also create one without task_id
	l2 := &models.ToolCallLog{
		ID:        "tcl-no-task",
		RunID:     "tcl-run",
		Tool:      "katana",
		Action:    "crawl",
		StartedAt: now.Add(time.Minute),
		Status:    models.ToolCallRunning,
		CreatedAt: now,
	}
	if err := q.CreateToolCallLog(l2); err != nil {
		t.Fatalf("create l2: %v", err)
	}

	list, err := q.ListToolCallLogsByTaskID("tcl-task")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("len: want 1, got %d", len(list))
	}
	if list[0].ID != "tcl-task-filter" {
		t.Errorf("id: want tcl-task-filter, got %q", list[0].ID)
	}
}

func TestListToolCallLogsByWorkItem(t *testing.T) {
	q := New(openTestDB(t))
	createToolCallLogPrereqs(t, q)
	now := time.Now().UTC().Truncate(time.Second)

	l := &models.ToolCallLog{
		ID:         "tcl-wi-filter",
		RunID:      "tcl-run",
		WorkItemID: strPtr("wi-abc"),
		Tool:       "nuclei",
		Action:     "scan",
		StartedAt:  now,
		Status:     models.ToolCallRunning,
		CreatedAt:  now,
	}
	if err := q.CreateToolCallLog(l); err != nil {
		t.Fatalf("create: %v", err)
	}

	// Also create one with different work_item_id
	l2 := &models.ToolCallLog{
		ID:         "tcl-wi-other",
		RunID:      "tcl-run",
		WorkItemID: strPtr("wi-xyz"),
		Tool:       "katana",
		Action:     "crawl",
		StartedAt:  now.Add(time.Minute),
		Status:     models.ToolCallRunning,
		CreatedAt:  now,
	}
	if err := q.CreateToolCallLog(l2); err != nil {
		t.Fatalf("create l2: %v", err)
	}

	list, err := q.ListToolCallLogsByWorkItem("wi-abc")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("len: want 1, got %d", len(list))
	}
	if list[0].ID != "tcl-wi-filter" {
		t.Errorf("id: want tcl-wi-filter, got %q", list[0].ID)
	}
}

func TestListToolCallLogsByRun_Empty(t *testing.T) {
	q := New(openTestDB(t))

	list, err := q.ListToolCallLogsByRun("no-such-run")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("want 0, got %d", len(list))
	}
}
