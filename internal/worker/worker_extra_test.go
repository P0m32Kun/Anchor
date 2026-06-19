package worker

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/scope"
)

// --- Cancel ---

func TestCancel_processNotRunning(t *testing.T) {
	r, q, _ := setupRunner(t)
	r.SetGovernor(NewResourceGovernor(GovernorConfig{Enabled: false}, nil))

	// Create a task in DB so UpdateScanTaskStatus works.
	now := time.Now().UTC()
	q.CreateScanTask(&models.ScanTask{
		ID: "task-cancel-norun", ProjectID: "proj-1", PlanID: "plan-1",
		Tool: "sh", CommandTemplate: "sh -c echo hi",
		Status: models.TaskCreated, CreatedAt: now,
	})

	err := r.Cancel("task-cancel-norun")
	if err != nil {
		t.Fatalf("Cancel should not error when process not running: %v", err)
	}

	task, _ := q.GetScanTask("task-cancel-norun")
	if task.Status != models.TaskCancelled {
		t.Errorf("status = %q, want %q", task.Status, models.TaskCancelled)
	}
}

func TestCancel_runningProcess(t *testing.T) {
	r, q, dataDir := setupRunner(t)
	r.SetGovernor(NewResourceGovernor(GovernorConfig{Enabled: false}, nil))

	// Create a task and manually register a long-running process.
	now := time.Now().UTC()
	q.CreateScanTask(&models.ScanTask{
		ID: "task-cancel-run", ProjectID: "proj-1", PlanID: "plan-1",
		Tool: "sh", CommandTemplate: "sh -c 'sleep 30'",
		Status: models.TaskRunning, CreatedAt: now,
	})

	workdir := filepath.Join(dataDir, "workdirs", "proj-1", "task-cancel-run")
	os.MkdirAll(workdir, 0750)

	ctx, ctxCancel := context.WithCancel(context.Background())
	defer ctxCancel()
	cmd := exec.CommandContext(ctx, "/bin/sleep", "30")
	cmd.Dir = workdir
	if err := cmd.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}

	doneCh := make(chan struct{})
	r.mu.Lock()
	r.procs["task-cancel-run"] = cmd
	r.doneChs["task-cancel-run"] = doneCh
	r.mu.Unlock()

	// Goroutine to close doneCh when process exits (mimics Run's tracking).
	// Run() deletes from procs/doneChs before closing doneCh.
	go func() {
		cmd.Wait()
		r.mu.Lock()
		delete(r.procs, "task-cancel-run")
		delete(r.doneChs, "task-cancel-run")
		r.mu.Unlock()
		close(doneCh)
	}()

	err := r.Cancel("task-cancel-run")
	if err != nil {
		t.Fatalf("Cancel: %v", err)
	}

	// Process should be dead.
	r.mu.RLock()
	_, tracked := r.procs["task-cancel-run"]
	r.mu.RUnlock()
	if tracked {
		t.Error("process should be removed from tracking after cancel")
	}

	task, _ := q.GetScanTask("task-cancel-run")
	if task.Status != models.TaskCancelled {
		t.Errorf("status = %q, want %q", task.Status, models.TaskCancelled)
	}
}

// --- Run with empty command template ---

func TestRun_emptyCommandTemplate(t *testing.T) {
	rawDB := openWorkerTestDB(t)
	q := db.New(rawDB)
	dataDir := t.TempDir()
	now := time.Now().UTC()

	q.CreateProject(&models.Project{
		ID: "proj-1", Name: "test", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	})
	q.CreateScanPlan(&models.ScanPlan{
		ID: "plan-1", ProjectID: "proj-1", WorkflowType: "manual",
		Profile: models.ProfileStandard, Status: "approved", CreatedBy: "test",
		CreatedAt: now,
	})
	q.CreateScanTask(&models.ScanTask{
		ID: "task-empty", ProjectID: "proj-1", PlanID: "plan-1",
		Tool: "sh", CommandTemplate: "",
		Status: models.TaskCreated, CreatedAt: now,
	})

	scopeEng := scope.NewEngine(q)
	r := NewRunner(q, scopeEng, dataDir)
	r.SetGovernor(NewResourceGovernor(GovernorConfig{Enabled: false}, nil))

	err := r.Run(t.Context(), "task-empty")
	if err == nil {
		t.Fatal("expected error for empty command template")
	}
	if !strings.Contains(err.Error(), "empty command template") {
		t.Errorf("error = %v", err)
	}
}

// --- Run with tool not found (local fallback) ---

func TestRun_toolNotFound(t *testing.T) {
	rawDB := openWorkerTestDB(t)
	q := db.New(rawDB)
	dataDir := t.TempDir()
	now := time.Now().UTC()

	q.CreateProject(&models.Project{
		ID: "proj-1", Name: "test", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	})
	q.CreateScanPlan(&models.ScanPlan{
		ID: "plan-1", ProjectID: "proj-1", WorkflowType: "manual",
		Profile: models.ProfileStandard, Status: "approved", CreatedBy: "test",
		CreatedAt: now,
	})
	q.CreateScanTask(&models.ScanTask{
		ID: "task-notfound", ProjectID: "proj-1", PlanID: "plan-1",
		Tool: "nonexistent-tool-xyz", CommandTemplate: "nonexistent-tool-xyz --help",
		Status: models.TaskCreated, CreatedAt: now,
	})

	scopeEng := scope.NewEngine(q)
	r := NewRunner(q, scopeEng, dataDir)
	r.SetGovernor(NewResourceGovernor(GovernorConfig{Enabled: false}, nil))

	err := r.Run(t.Context(), "task-notfound")
	if err == nil {
		t.Fatal("expected error for nonexistent tool")
	}
	if !strings.Contains(err.Error(), "tool") {
		t.Errorf("error = %v", err)
	}
}

// --- Run successful local execution ---

func TestRun_successLocalExecution(t *testing.T) {
	rawDB := openWorkerTestDB(t)
	q := db.New(rawDB)
	dataDir := t.TempDir()
	now := time.Now().UTC()

	q.CreateProject(&models.Project{
		ID: "proj-1", Name: "test", RateLimit: 0,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	})
	q.CreateScanPlan(&models.ScanPlan{
		ID: "plan-1", ProjectID: "proj-1", WorkflowType: "manual",
		Profile: models.ProfileStandard, Status: "approved", CreatedBy: "test",
		CreatedAt: now,
	})
	q.CreateScanTask(&models.ScanTask{
		ID: "task-ok", ProjectID: "proj-1", PlanID: "plan-1",
		Tool: "sh", CommandTemplate: "sh -c echo hello",
		Status: models.TaskCreated, CreatedAt: now,
	})

	scopeEng := scope.NewEngine(q)
	r := NewRunner(q, scopeEng, dataDir)
	r.SetGovernor(NewResourceGovernor(GovernorConfig{Enabled: false}, nil))

	err := r.Run(t.Context(), "task-ok")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	task, _ := q.GetScanTask("task-ok")
	if task.Status != models.TaskCompleted {
		t.Errorf("status = %q, want %q", task.Status, models.TaskCompleted)
	}
}

// --- Run failed local execution ---

func TestRun_failedLocalExecution(t *testing.T) {
	rawDB := openWorkerTestDB(t)
	q := db.New(rawDB)
	dataDir := t.TempDir()
	now := time.Now().UTC()

	q.CreateProject(&models.Project{
		ID: "proj-1", Name: "test", RateLimit: 0,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	})
	q.CreateScanPlan(&models.ScanPlan{
		ID: "plan-1", ProjectID: "proj-1", WorkflowType: "manual",
		Profile: models.ProfileStandard, Status: "approved", CreatedBy: "test",
		CreatedAt: now,
	})
	q.CreateScanTask(&models.ScanTask{
		ID: "task-fail", ProjectID: "proj-1", PlanID: "plan-1",
		Tool: "sh", CommandTemplate: "sh -c 'exit 1'",
		Status: models.TaskCreated, CreatedAt: now,
	})

	scopeEng := scope.NewEngine(q)
	r := NewRunner(q, scopeEng, dataDir)
	r.SetGovernor(NewResourceGovernor(GovernorConfig{Enabled: false}, nil))

	err := r.Run(t.Context(), "task-fail")
	if err == nil {
		t.Fatal("expected error for failed command")
	}

	task, _ := q.GetScanTask("task-fail")
	if task.Status != models.TaskFailed {
		t.Errorf("status = %q, want %q", task.Status, models.TaskFailed)
	}
}

// --- Run governor blocks task ---

func TestRun_governorBlocks(t *testing.T) {
	rawDB := openWorkerTestDB(t)
	q := db.New(rawDB)
	dataDir := t.TempDir()
	now := time.Now().UTC()

	q.CreateProject(&models.Project{
		ID: "proj-1", Name: "test", RateLimit: 0,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	})
	q.CreateScanPlan(&models.ScanPlan{
		ID: "plan-1", ProjectID: "proj-1", WorkflowType: "manual",
		Profile: models.ProfileStandard, Status: "approved", CreatedBy: "test",
		CreatedAt: now,
	})
	q.CreateScanTask(&models.ScanTask{
		ID: "task-gov", ProjectID: "proj-1", PlanID: "plan-1",
		Tool: "sh", CommandTemplate: "sh -c echo hi",
		Status: models.TaskCreated, CreatedAt: now,
	})

	scopeEng := scope.NewEngine(q)
	r := NewRunner(q, scopeEng, dataDir)

	// Governor with impossible memory threshold that blocks forever.
	sampler := &fakeSampler{mem: 99, cpu: 50}
	cfg := GovernorConfig{
		Enabled:            true,
		MemoryThresholdPct: 0.01,
		CPUThresholdPct:    100,
		MemoryPollInterval: 10 * time.Millisecond,
		CPUDelay:           10 * time.Millisecond,
	}
	r.SetGovernor(NewResourceGovernor(cfg, sampler))

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := r.Run(ctx, "task-gov")
	if err == nil {
		t.Fatal("expected error from governor blocking")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !strings.Contains(err.Error(), "governor") {
		t.Errorf("error = %v, want governor-related or DeadlineExceeded", err)
	}
}

// --- isAtCapacityError additional ---

func TestIsAtCapacityError_various(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"capacity 503", errors.New("worker at capacity: 503"), true},
		{"capacity mixed case", errors.New("Worker At Capacity"), true},
		{"connection refused", errors.New("connection refused"), false},
		{"generic error", errors.New("something else"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isAtCapacityError(tt.err)
			if got != tt.want {
				t.Errorf("isAtCapacityError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

// --- Run with cancelled context ---

func TestRun_contextCancelled(t *testing.T) {
	rawDB := openWorkerTestDB(t)
	q := db.New(rawDB)
	dataDir := t.TempDir()
	now := time.Now().UTC()

	q.CreateProject(&models.Project{
		ID: "proj-1", Name: "test", RateLimit: 0,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	})
	q.CreateScanPlan(&models.ScanPlan{
		ID: "plan-1", ProjectID: "proj-1", WorkflowType: "manual",
		Profile: models.ProfileStandard, Status: "approved", CreatedBy: "test",
		CreatedAt: now,
	})
	q.CreateScanTask(&models.ScanTask{
		ID: "task-ctx", ProjectID: "proj-1", PlanID: "plan-1",
		Tool: "sh", CommandTemplate: "sh -c 'sleep 30'",
		Status: models.TaskCreated, CreatedAt: now,
	})

	scopeEng := scope.NewEngine(q)
	r := NewRunner(q, scopeEng, dataDir)
	r.SetGovernor(NewResourceGovernor(GovernorConfig{Enabled: false}, nil))

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel after a short delay so the process starts then gets killed.
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	err := r.Run(ctx, "task-ctx")
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}
