package scanengine

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/asset"
	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/scanengine/core"
	"github.com/P0m32Kun/Anchor/internal/toolregistry"
	"github.com/P0m32Kun/Anchor/internal/toolrun"
)

// fakeExecutor implements executor.Executor for testing.
type fakeExecutor struct {
	mu       sync.Mutex
	calls    []executorCall
	behavior func(ctx context.Context, w *models.ScanWorkItem) (*toolrun.InvokeResult, error)
}

type executorCall struct {
	WorkID  string
	Action  string
	AssetID string
}

func (f *fakeExecutor) Execute(ctx context.Context, w *models.ScanWorkItem, params toolregistry.RenderParams) (*toolrun.InvokeResult, error) {
	f.mu.Lock()
	f.calls = append(f.calls, executorCall{WorkID: w.ID, Action: w.Action, AssetID: w.AssetID})
	f.mu.Unlock()
	if f.behavior != nil {
		return f.behavior(ctx, w)
	}
	return &toolrun.InvokeResult{Task: &models.ScanTask{ID: "fake-task"}, Stdout: nil}, nil
}

func (f *fakeExecutor) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

func (f *fakeExecutor) getCalls() []executorCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	cp := make([]executorCall, len(f.calls))
	copy(cp, f.calls)
	return cp
}

// newTestDB creates an in-memory SQLite database with all migrations applied.
func newTestDB(t *testing.T) *db.Queries {
	t.Helper()
	sqlDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	sqlDB.SetMaxOpenConns(1) // required for :memory: to share state
	if err := db.Migrate(sqlDB); err != nil {
		t.Fatal(err)
	}
	return db.New(sqlDB)
}

// setupTestEngine creates a ScanEngine with a fake executor and in-memory DB.
func setupTestEngine(t *testing.T, fakeExec *fakeExecutor, config EngineConfig) (*ScanEngine, *db.Queries) {
	t.Helper()
	queries := newTestDB(t)

	// Create project
	if err := queries.CreateProject(&models.Project{ID: "proj1", Name: "test", CreatedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	// Create pipeline run
	if err := queries.CreatePipelineRun(&models.PipelineRun{
		ID:        "run1",
		ProjectID: "proj1",
		Status:    "running",
		CreatedAt: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	merger := asset.NewMerger(queries)
	profile := core.DefaultInternalProfile()
	engine := NewWithExecutor(queries, merger, profile, nil, nil, t.TempDir(), "run1", "proj1", config, nil, fakeExec)
	return engine, queries
}

func TestEngine_BasicRun_StopsOnCancel(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	cfg.SchedulerTick = 50 * time.Millisecond
	cfg.IdleTimeout = 200 * time.Millisecond
	engine, _ := setupTestEngine(t, fake, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := engine.Run(ctx, []string{"10.0.0.1"})
	if err != nil && err != context.DeadlineExceeded && err != context.Canceled {
		t.Fatalf("unexpected error: %v", err)
	}
	// Engine should have processed at least the seed asset
	t.Logf("executor called %d times", fake.callCount())
}

func TestEngine_IdleTimeout_WindDown(t *testing.T) {
	fake := &fakeExecutor{
		behavior: func(ctx context.Context, w *models.ScanWorkItem) (*toolrun.InvokeResult, error) {
			return &toolrun.InvokeResult{Task: &models.ScanTask{ID: "t"}, Stdout: nil}, nil
		},
	}
	cfg := DefaultEngineConfig()
	cfg.SchedulerTick = 20 * time.Millisecond
	cfg.IdleTimeout = 100 * time.Millisecond
	cfg.AbsoluteTimeout = 5 * time.Second
	engine, queries := setupTestEngine(t, fake, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := engine.Run(ctx, []string{"10.0.0.1"})
	if err != nil && err != context.DeadlineExceeded && err != context.Canceled {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check engine state
	run, _ := queries.GetPipelineRun("run1")
	if run != nil && run.EngineState != "stopped" {
		t.Errorf("expected engine_state=stopped, got %s", run.EngineState)
	}
}

func TestEngine_ConcurrencyBound(t *testing.T) {
	var maxConcurrent int64
	var currentConcurrent int64

	fake := &fakeExecutor{
		behavior: func(ctx context.Context, w *models.ScanWorkItem) (*toolrun.InvokeResult, error) {
			cur := atomic.AddInt64(&currentConcurrent, 1)
			// Track max concurrency
			for {
				old := atomic.LoadInt64(&maxConcurrent)
				if cur <= old || atomic.CompareAndSwapInt64(&maxConcurrent, old, cur) {
					break
				}
			}
			time.Sleep(50 * time.Millisecond) // simulate work
			atomic.AddInt64(&currentConcurrent, -1)
			return &toolrun.InvokeResult{Task: &models.ScanTask{ID: "t"}, Stdout: nil}, nil
		},
	}

	cfg := DefaultEngineConfig()
	cfg.BatchSize = 3
	cfg.SchedulerTick = 10 * time.Millisecond
	cfg.IdleTimeout = 500 * time.Millisecond
	engine, _ := setupTestEngine(t, fake, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Seed multiple targets to generate work
	engine.Run(ctx, []string{"10.0.0.1", "10.0.0.2", "10.0.0.3", "10.0.0.4", "10.0.0.5"})

	got := atomic.LoadInt64(&maxConcurrent)
	if got > int64(cfg.BatchSize) {
		t.Errorf("max concurrent %d exceeded batch size %d", got, cfg.BatchSize)
	}
	t.Logf("max concurrent: %d (limit: %d)", got, cfg.BatchSize)
}

func TestEngine_CancelDrainsGoroutines(t *testing.T) {
	var activeGoroutines int64

	fake := &fakeExecutor{
		behavior: func(ctx context.Context, w *models.ScanWorkItem) (*toolrun.InvokeResult, error) {
			atomic.AddInt64(&activeGoroutines, 1)
			defer atomic.AddInt64(&activeGoroutines, -1)
			time.Sleep(30 * time.Millisecond)
			return &toolrun.InvokeResult{Task: &models.ScanTask{ID: "t"}, Stdout: nil}, nil
		},
	}

	cfg := DefaultEngineConfig()
	cfg.SchedulerTick = 10 * time.Millisecond
	cfg.IdleTimeout = 200 * time.Millisecond
	engine, _ := setupTestEngine(t, fake, cfg)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- engine.Run(ctx, []string{"10.0.0.1"}) }()

	time.Sleep(200 * time.Millisecond)
	cancel()
	<-done

	// After Run returns, all goroutines should be drained
	remaining := atomic.LoadInt64(&activeGoroutines)
	if remaining != 0 {
		t.Errorf("expected 0 active goroutines after cancel, got %d", remaining)
	}
}

func TestEngine_FailedTaskDoesNotBlock(t *testing.T) {
	callCount := int64(0)
	fake := &fakeExecutor{
		behavior: func(ctx context.Context, w *models.ScanWorkItem) (*toolrun.InvokeResult, error) {
			n := atomic.AddInt64(&callCount, 1)
			if n == 1 {
				return nil, fmt.Errorf("simulated failure")
			}
			return &toolrun.InvokeResult{Task: &models.ScanTask{ID: "t"}, Stdout: nil}, nil
		},
	}

	cfg := DefaultEngineConfig()
	cfg.BatchSize = 1
	cfg.Tier1FlushTimeout = 30 * time.Millisecond
	cfg.SchedulerTick = 20 * time.Millisecond
	cfg.IdleTimeout = 3 * time.Second
	engine, queries := setupTestEngine(t, fake, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	engine.Run(ctx, []string{"10.0.0.1"})

	// With retry logic, the first failure is retried and succeeds
	// So we expect all work items to be done (no failures)
	works, _ := queries.ListScanWorkItemsByRun("run1")
	failed := 0
	done := 0
	for _, w := range works {
		switch w.Status {
		case models.WorkStatusFailed:
			failed++
		case models.WorkStatusDone:
			done++
		}
	}
	t.Logf("total works: %d, failed: %d, done: %d, calls: %d", len(works), failed, done, atomic.LoadInt64(&callCount))
	// With retry, the first call fails but retry succeeds
	// We should see more calls than works due to retries
	if len(works) > 0 && atomic.LoadInt64(&callCount) <= int64(len(works)) {
		t.Error("expected retry calls > work items due to first-call failure")
	}
}

func TestEngine_DedupAcrossTicks(t *testing.T) {
	fake := &fakeExecutor{
		behavior: func(ctx context.Context, w *models.ScanWorkItem) (*toolrun.InvokeResult, error) {
			return &toolrun.InvokeResult{Task: &models.ScanTask{ID: "t"}, Stdout: nil}, nil
		},
	}

	cfg := DefaultEngineConfig()
	cfg.SchedulerTick = 20 * time.Millisecond
	cfg.IdleTimeout = 200 * time.Millisecond
	engine, queries := setupTestEngine(t, fake, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Same target twice — should be deduped
	engine.Run(ctx, []string{"10.0.0.1", "10.0.0.1"})

	works, _ := queries.ListScanWorkItemsByRun("run1")
	// Each asset should produce unique (run_id, asset_id, action) work items
	seen := make(map[string]bool)
	for _, w := range works {
		key := w.AssetID + ":" + w.Action
		if seen[key] {
			t.Errorf("duplicate work item: %s", key)
		}
		seen[key] = true
	}
}

func TestEngine_WindDown_SkipsNonEssential(t *testing.T) {
	fake := &fakeExecutor{
		behavior: func(ctx context.Context, w *models.ScanWorkItem) (*toolrun.InvokeResult, error) {
			return &toolrun.InvokeResult{Task: &models.ScanTask{ID: "t"}, Stdout: nil}, nil
		},
	}

	cfg := DefaultEngineConfig()
	cfg.SchedulerTick = 20 * time.Millisecond
	cfg.IdleTimeout = 80 * time.Millisecond // very short idle
	cfg.AbsoluteTimeout = 5 * time.Second
	engine, queries := setupTestEngine(t, fake, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	engine.Run(ctx, []string{"10.0.0.1"})

	// After wind_down, non-essential actions should be skipped
	works, _ := queries.ListScanWorkItemsByRun("run1")
	skipped := 0
	for _, w := range works {
		if w.Status == models.WorkStatusSkipped {
			skipped++
		}
	}
	t.Logf("total: %d, skipped: %d", len(works), skipped)
}

func TestEngine_PriorityOrder(t *testing.T) {
	var actionOrder []string
	var mu sync.Mutex

	fake := &fakeExecutor{
		behavior: func(ctx context.Context, w *models.ScanWorkItem) (*toolrun.InvokeResult, error) {
			mu.Lock()
			actionOrder = append(actionOrder, w.Action)
			mu.Unlock()
			time.Sleep(10 * time.Millisecond)
			return &toolrun.InvokeResult{Task: &models.ScanTask{ID: "t"}, Stdout: nil}, nil
		},
	}

	cfg := DefaultEngineConfig()
	cfg.BatchSize = 1
	cfg.Tier1FlushTimeout = 30 * time.Millisecond
	cfg.SchedulerTick = 20 * time.Millisecond
	cfg.IdleTimeout = 300 * time.Millisecond
	engine, _ := setupTestEngine(t, fake, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	engine.Run(ctx, []string{"10.0.0.1"})

	mu.Lock()
	defer mu.Unlock()
	t.Logf("execution order: %v", actionOrder)

	// Stage order: DNS/CDN/PORT before HTTPX/NUCLEI
	firstLate := -1
	firstEarly := -1
	for i, a := range actionOrder {
		if a == "HTTPX_FINGERPRINT" || a == "NUCLEI_SCAN" {
			if firstLate == -1 {
				firstLate = i
			}
		}
		if a == "SUBDOMAIN_ENUM" || a == "CDN_CHECK" || a == "DNS_RESOLVE" || a == "PORT_SCAN" {
			if firstEarly == -1 {
				firstEarly = i
			}
		}
	}
	if firstEarly >= 0 && firstLate >= 0 && firstLate < firstEarly {
		t.Errorf("late-stage action at index %d ran before early-stage at %d: %v", firstLate, firstEarly, actionOrder)
	}
}

func TestEngine_WorkItemLifecycle(t *testing.T) {
	fake := &fakeExecutor{
		behavior: func(ctx context.Context, w *models.ScanWorkItem) (*toolrun.InvokeResult, error) {
			return &toolrun.InvokeResult{Task: &models.ScanTask{ID: "t"}, Stdout: nil}, nil
		},
	}

	cfg := DefaultEngineConfig()
	cfg.SchedulerTick = 20 * time.Millisecond
	cfg.IdleTimeout = 200 * time.Millisecond
	engine, queries := setupTestEngine(t, fake, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	engine.Run(ctx, []string{"10.0.0.1"})

	// All work items should be in terminal state
	works, _ := queries.ListScanWorkItemsByRun("run1")
	for _, w := range works {
		switch w.Status {
		case models.WorkStatusDone, models.WorkStatusFailed, models.WorkStatusSkipped:
			// OK
		default:
			t.Errorf("work %s has non-terminal status: %s", w.ID, w.Status)
		}
	}
}

func TestEngine_SeedCreatesWorkItems(t *testing.T) {
	fake := &fakeExecutor{
		behavior: func(ctx context.Context, w *models.ScanWorkItem) (*toolrun.InvokeResult, error) {
			return &toolrun.InvokeResult{Task: &models.ScanTask{ID: "t"}, Stdout: nil}, nil
		},
	}

	cfg := DefaultEngineConfig()
	cfg.SchedulerTick = 20 * time.Millisecond
	cfg.IdleTimeout = 200 * time.Millisecond
	engine, queries := setupTestEngine(t, fake, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	engine.Run(ctx, []string{"10.0.0.1", "10.0.0.2"})

	// Should have work items for both targets
	works, _ := queries.ListScanWorkItemsByRun("run1")
	if len(works) == 0 {
		t.Fatal("expected work items to be created")
	}

	// Check that assets were created
	assets1, _ := queries.ListScanWorkItemsByAsset("run1", works[0].AssetID)
	if len(assets1) == 0 {
		t.Error("expected work items for first asset")
	}
	t.Logf("total work items: %d", len(works))
}

// TestEngine_Stress_ShortTasks runs many fast-completing tasks.
func TestEngine_Stress_ShortTasks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	var completed int64
	fake := &fakeExecutor{
		behavior: func(ctx context.Context, w *models.ScanWorkItem) (*toolrun.InvokeResult, error) {
			atomic.AddInt64(&completed, 1)
			return &toolrun.InvokeResult{Task: &models.ScanTask{ID: "t"}, Stdout: nil}, nil
		},
	}

	cfg := DefaultEngineConfig()
	cfg.BatchSize = 10
	cfg.SchedulerTick = 10 * time.Millisecond
	cfg.IdleTimeout = 500 * time.Millisecond
	engine, _ := setupTestEngine(t, fake, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Seed many targets
	targets := make([]string, 20)
	for i := range targets {
		targets[i] = fmt.Sprintf("10.0.%d.1", i)
	}

	engine.Run(ctx, targets)

	got := atomic.LoadInt64(&completed)
	t.Logf("completed %d work items", got)
	if got == 0 {
		t.Fatal("expected some work items to complete")
	}
}

func TestEngine_ConcurrentPushPopSafety(t *testing.T) {
	// This test verifies no panics or data races under concurrent operations
	if testing.Short() {
		t.Skip("skipping concurrent safety test in short mode")
	}

	fake := &fakeExecutor{
		behavior: func(ctx context.Context, w *models.ScanWorkItem) (*toolrun.InvokeResult, error) {
			time.Sleep(5 * time.Millisecond)
			return &toolrun.InvokeResult{Task: &models.ScanTask{ID: "t"}, Stdout: nil}, nil
		},
	}

	cfg := DefaultEngineConfig()
	cfg.BatchSize = 5
	cfg.SchedulerTick = 5 * time.Millisecond
	cfg.IdleTimeout = 300 * time.Millisecond
	engine, _ := setupTestEngine(t, fake, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	targets := make([]string, 10)
	for i := range targets {
		targets[i] = fmt.Sprintf("192.168.%d.1", i)
	}

	engine.Run(ctx, targets)
	// If we get here without panics or race detector failures, the test passes
}

func TestEngine_LastNewAssetAt_ReadSafety(t *testing.T) {
	// Verifies the read-lock fix for lastNewAssetAt
	fake := &fakeExecutor{
		behavior: func(ctx context.Context, w *models.ScanWorkItem) (*toolrun.InvokeResult, error) {
			return &toolrun.InvokeResult{Task: &models.ScanTask{ID: "t"}, Stdout: nil}, nil
		},
	}

	cfg := DefaultEngineConfig()
	cfg.SchedulerTick = 5 * time.Millisecond
	cfg.IdleTimeout = 100 * time.Millisecond
	cfg.AbsoluteTimeout = 5 * time.Second
	engine, _ := setupTestEngine(t, fake, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Run with rapid ticks to stress the read path
	engine.Run(ctx, []string{"10.0.0.1"})
}

func TestEngine_AggregatorWorkCounts(t *testing.T) {
	fake := &fakeExecutor{
		behavior: func(ctx context.Context, w *models.ScanWorkItem) (*toolrun.InvokeResult, error) {
			return &toolrun.InvokeResult{Task: &models.ScanTask{ID: "t"}, Stdout: nil}, nil
		},
	}

	cfg := DefaultEngineConfig()
	cfg.SchedulerTick = 20 * time.Millisecond
	cfg.IdleTimeout = 200 * time.Millisecond
	engine, queries := setupTestEngine(t, fake, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	engine.Run(ctx, []string{"10.0.0.1"})

	// Check that stage records were created with work counts
	stages, _ := queries.ListPipelineRunStages("run1")
	for _, s := range stages {
		total, done, running, round := 0, 0, 0, 0
		if s.WorkTotal != nil {
			total = *s.WorkTotal
		}
		if s.WorkDone != nil {
			done = *s.WorkDone
		}
		if s.WorkRunning != nil {
			running = *s.WorkRunning
		}
		if s.Round != nil {
			round = *s.Round
		}
		t.Logf("stage %s: total=%d done=%d running=%d round=%d",
			s.Stage, total, done, running, round)
		if total > 0 && done != total {
			t.Errorf("stage %s: work_done(%d) != work_total(%d)", s.Stage, done, total)
		}
	}
}
