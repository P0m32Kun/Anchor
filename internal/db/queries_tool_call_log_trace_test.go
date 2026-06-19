package db

import (
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
)

// createTraceChain creates the full prerequisite chain for GetToolCallTraceByFinding:
// project → scan_plan → pipeline_run → scan_task → finding → scan_work_item → tool_call_log.
func createTraceChain(t *testing.T, q *Queries) {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Second)

	// Project
	if err := q.CreateProject(&models.Project{
		ID: "trace-proj", Name: "trace-project", Organization: "test-org",
		Purpose: "testing", RateLimit: 100,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create project: %v", err)
	}

	// Scan plan
	if err := q.CreateScanPlan(&models.ScanPlan{
		ID: "trace-plan", ProjectID: "trace-proj",
		WorkflowType: "full", Profile: models.ProfileStandard,
		Status: "approved", CreatedBy: "test", CreatedAt: now,
	}); err != nil {
		t.Fatalf("create scan plan: %v", err)
	}

	// Pipeline run
	if err := q.CreatePipelineRun(&models.PipelineRun{
		ID: "trace-run", ProjectID: "trace-proj",
		Mode: "quick", Status: "running", Stage: "scan",
		StartedAt: now, CreatedAt: now,
	}); err != nil {
		t.Fatalf("create pipeline run: %v", err)
	}

	// Run (required by findings.run_id FK)
	if err := q.CreateRun(&models.Run{
		ID: "trace-run", ProjectID: "trace-proj",
		Name: "trace-run", Status: models.RunRunning,
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("create run: %v", err)
	}

	// Scan task
	if err := q.CreateScanTask(&models.ScanTask{
		ID: "trace-task", ProjectID: "trace-proj", PlanID: "trace-plan",
		RunID: strPtr("trace-run"), Tool: "nuclei",
		CommandTemplate: "nuclei -u {{target}}",
		Status: models.TaskRunning, CreatedAt: now,
	}); err != nil {
		t.Fatalf("create scan task: %v", err)
	}

	// Finding with source_task_id and run_id
	if err := q.CreateFinding(&models.Finding{
		ID: "trace-finding", ProjectID: "trace-proj",
		RunID: strPtr("trace-run"), SourceTaskID: strPtr("trace-task"),
		SourceTool: "nuclei", SourceRuleID: "rule-1",
		DedupKey: "dedup-trace-1", Title: "XSS in login",
		Severity: models.SeverityHigh, Confidence: 90, Priority: 80,
		Status: models.FindingNew, Summary: "Reflected XSS",
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create finding: %v", err)
	}

	// Scan work item linked to the task
	if err := q.CreateScanWorkItem(&models.ScanWorkItem{
		ID: "trace-wi", RunID: "trace-run", ProjectID: "trace-proj",
		AssetID: "asset-1", Action: "scan",
		TaskID: strPtr("trace-task"), Status: models.WorkStatusDone,
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("create scan work item: %v", err)
	}

	// Tool call log linked to the work item
	if err := q.CreateToolCallLog(&models.ToolCallLog{
		ID: "trace-log", RunID: "trace-run",
		WorkItemID: strPtr("trace-wi"), TaskID: strPtr("trace-task"),
		Tool: "nuclei", Action: "scan",
		StartedAt: now, Status: models.ToolCallCompleted, CreatedAt: now,
	}); err != nil {
		t.Fatalf("create tool call log: %v", err)
	}
}

func TestGetToolCallTraceByFinding_FullChain(t *testing.T) {
	q := New(openTestDB(t))
	createTraceChain(t, q)

	trace, err := q.GetToolCallTraceByFinding("trace-finding")
	if err != nil {
		t.Fatalf("GetToolCallTraceByFinding: %v", err)
	}
	if trace == nil {
		t.Fatal("trace is nil")
	}

	// Finding
	if trace.Finding == nil {
		t.Fatal("trace.Finding is nil")
	}
	if trace.Finding.ID != "trace-finding" {
		t.Errorf("finding id: want trace-finding, got %q", trace.Finding.ID)
	}

	// Task (via source_task_id)
	if trace.Task == nil {
		t.Fatal("trace.Task is nil")
	}
	if trace.Task.ID != "trace-task" {
		t.Errorf("task id: want trace-task, got %q", trace.Task.ID)
	}

	// WorkItem (via getScanWorkItemByTaskID — was 0% coverage)
	if trace.WorkItem == nil {
		t.Fatal("trace.WorkItem is nil")
	}
	if trace.WorkItem.ID != "trace-wi" {
		t.Errorf("work item id: want trace-wi, got %q", trace.WorkItem.ID)
	}

	// Run (via finding.run_id)
	if trace.Run == nil {
		t.Fatal("trace.Run is nil")
	}
	if trace.Run.ID != "trace-run" {
		t.Errorf("run id: want trace-run, got %q", trace.Run.ID)
	}

	// ToolCallLog (via work_item_id)
	if trace.ToolCallLog == nil {
		t.Fatal("trace.ToolCallLog is nil")
	}
	if trace.ToolCallLog.ID != "trace-log" {
		t.Errorf("tool call log id: want trace-log, got %q", trace.ToolCallLog.ID)
	}
}

func TestGetToolCallTraceByFinding_NotFound(t *testing.T) {
	q := New(openTestDB(t))

	trace, err := q.GetToolCallTraceByFinding("nonexistent")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if trace != nil {
		t.Errorf("want nil, got %+v", trace)
	}
}

func TestGetToolCallTraceByFinding_FindingOnly(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC().Truncate(time.Second)

	// Project (required by FK)
	if err := q.CreateProject(&models.Project{
		ID: "fo-proj", Name: "fo-project", Organization: "test-org",
		Purpose: "testing", RateLimit: 100,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create project: %v", err)
	}

	// Finding with NO source_task_id and NO run_id
	if err := q.CreateFinding(&models.Finding{
		ID: "fo-finding", ProjectID: "fo-proj",
		SourceTool: "nuclei", SourceRuleID: "rule-2",
		DedupKey: "dedup-fo-1", Title: "Open redirect",
		Severity: models.SeverityMedium, Confidence: 70, Priority: 50,
		Status: models.FindingNew, Summary: "Redirect on param",
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create finding: %v", err)
	}

	trace, err := q.GetToolCallTraceByFinding("fo-finding")
	if err != nil {
		t.Fatalf("GetToolCallTraceByFinding: %v", err)
	}
	if trace == nil {
		t.Fatal("trace is nil")
	}
	if trace.Finding == nil || trace.Finding.ID != "fo-finding" {
		t.Errorf("finding: want fo-finding, got %v", trace.Finding)
	}
	if trace.Task != nil {
		t.Errorf("task: want nil, got %+v", trace.Task)
	}
	if trace.WorkItem != nil {
		t.Errorf("work_item: want nil, got %+v", trace.WorkItem)
	}
	if trace.Run != nil {
		t.Errorf("run: want nil, got %+v", trace.Run)
	}
	if trace.ToolCallLog != nil {
		t.Errorf("tool_call_log: want nil, got %+v", trace.ToolCallLog)
	}
}

func TestGetToolCallTraceByFinding_FallbackToTaskIDLog(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC().Truncate(time.Second)

	// Minimal chain: project → plan → run → task → finding → log (via task_id, no work_item)
	if err := q.CreateProject(&models.Project{
		ID: "fb-proj", Name: "fb-project", Organization: "test-org",
		Purpose: "testing", RateLimit: 100,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create project: %v", err)
	}
	if err := q.CreateScanPlan(&models.ScanPlan{
		ID: "fb-plan", ProjectID: "fb-proj",
		WorkflowType: "full", Profile: models.ProfileStandard,
		Status: "approved", CreatedBy: "test", CreatedAt: now,
	}); err != nil {
		t.Fatalf("create scan plan: %v", err)
	}
	if err := q.CreatePipelineRun(&models.PipelineRun{
		ID: "fb-run", ProjectID: "fb-proj",
		Mode: "quick", Status: "running", Stage: "scan",
		StartedAt: now, CreatedAt: now,
	}); err != nil {
		t.Fatalf("create pipeline run: %v", err)
	}
	if err := q.CreateRun(&models.Run{
		ID: "fb-run", ProjectID: "fb-proj",
		Name: "fb-run", Status: models.RunRunning,
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("create run: %v", err)
	}
	if err := q.CreateScanTask(&models.ScanTask{
		ID: "fb-task", ProjectID: "fb-proj", PlanID: "fb-plan",
		RunID: strPtr("fb-run"), Tool: "katana",
		CommandTemplate: "katana -u {{target}}",
		Status: models.TaskRunning, CreatedAt: now,
	}); err != nil {
		t.Fatalf("create scan task: %v", err)
	}
	if err := q.CreateFinding(&models.Finding{
		ID: "fb-finding", ProjectID: "fb-proj",
		RunID: strPtr("fb-run"), SourceTaskID: strPtr("fb-task"),
		SourceTool: "katana", SourceRuleID: "rule-fb",
		DedupKey: "dedup-fb-1", Title: "Info leak",
		Severity: models.SeverityLow, Confidence: 60, Priority: 30,
		Status: models.FindingNew, Summary: "Leaked version",
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create finding: %v", err)
	}

	// Tool call log with task_id but NO work_item_id — triggers fallback path
	if err := q.CreateToolCallLog(&models.ToolCallLog{
		ID: "fb-log", RunID: "fb-run",
		TaskID: strPtr("fb-task"),
		Tool: "katana", Action: "crawl",
		StartedAt: now, Status: models.ToolCallCompleted, CreatedAt: now,
	}); err != nil {
		t.Fatalf("create tool call log: %v", err)
	}

	trace, err := q.GetToolCallTraceByFinding("fb-finding")
	if err != nil {
		t.Fatalf("GetToolCallTraceByFinding: %v", err)
	}
	if trace == nil {
		t.Fatal("trace is nil")
	}
	if trace.Task == nil || trace.Task.ID != "fb-task" {
		t.Errorf("task: want fb-task, got %v", trace.Task)
	}
	// No work item → WorkItem should be nil
	if trace.WorkItem != nil {
		t.Errorf("work_item: want nil, got %+v", trace.WorkItem)
	}
	// ToolCallLog should be found via task_id fallback
	if trace.ToolCallLog == nil {
		t.Fatal("tool_call_log: want non-nil (fallback), got nil")
	}
	if trace.ToolCallLog.ID != "fb-log" {
		t.Errorf("tool_call_log id: want fb-log, got %q", trace.ToolCallLog.ID)
	}
}
