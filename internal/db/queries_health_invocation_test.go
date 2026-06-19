package db

import (
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// --- ToolInvocation tests ---

func TestToolInvocation_CreateUpdateList(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC()

	if err := q.CreateProject(&models.Project{
		ID: "proj-inv", Name: "inv-test", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create project: %v", err)
	}

	// Create scan_task first (FK on task_id)
	if err := q.CreateScanTask(&models.ScanTask{
		ID: "task-inv-1", ProjectID: "proj-inv", Tool: "nuclei",
		Status: models.TaskRunning, CreatedAt: now,
	}); err != nil {
		t.Fatalf("create scan_task: %v", err)
	}

	inv := &models.ToolInvocation{
		ID: util.GenerateID(), ProjectID: "proj-inv", TaskID: "task-inv-1",
		Tool: "nuclei", BinaryPath: "/usr/bin/nuclei", Version: "3.0.0",
		CommandRedacted: "nuclei -u [redacted]", Workdir: "/tmp/work",
		StartedAt: now,
	}
	if err := q.CreateToolInvocation(inv); err != nil {
		t.Fatalf("CreateToolInvocation: %v", err)
	}

	// Update
	finishedAt := now.Add(30 * time.Second)
	if err := q.UpdateToolInvocation("task-inv-1", finishedAt, 0); err != nil {
		t.Fatalf("UpdateToolInvocation: %v", err)
	}

	// List
	list, err := q.ListToolInvocationsByProject("proj-inv")
	if err != nil {
		t.Fatalf("ListToolInvocationsByProject: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 invocation, got %d", len(list))
	}
	if list[0].Tool != "nuclei" {
		t.Errorf("tool = %q, want nuclei", list[0].Tool)
	}
	if list[0].ExitCode == nil || *list[0].ExitCode != 0 {
		t.Errorf("exit_code = %v, want 0", list[0].ExitCode)
	}
	if list[0].FinishedAt == nil {
		t.Error("expected finished_at to be set")
	}

	// Empty project
	empty, err := q.ListToolInvocationsByProject("nonexistent")
	if err != nil {
		t.Fatalf("ListToolInvocationsByProject empty: %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("expected 0, got %d", len(empty))
	}
}
