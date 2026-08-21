package scanengine

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/scanengine/core"
	"github.com/P0m32Kun/Anchor/internal/toolrun"
)

// quiescenceTestConfig returns a config where the scheduler ticks far faster
// than the pool flush timeout, so pooled members only become work through the
// engine's flush-before-stop path — the exact shape of the production bug.
func quiescenceTestConfig() EngineConfig {
	cfg := DefaultEngineConfig()
	cfg.SchedulerTick = 20 * time.Millisecond
	cfg.IdleTimeout = 2 * time.Second
	return cfg
}

func actionCalls(fake *fakeExecutor, action core.TaskAction) []int {
	idxs := make([]int, 0, 2)
	for i, c := range fake.getCalls() {
		if c.Action == string(action) {
			idxs = append(idxs, i)
		}
	}
	return idxs
}

// taskLinker creates a real scan_tasks row for each executed work item, so
// work-item task_id links satisfy the foreign key exactly like the
// production executor (which dispatches a task before returning it).
type taskLinker struct {
	t       *testing.T
	queries *db.Queries
	seq     uint64
}

func (l *taskLinker) taskFor(w *models.ScanWorkItem) *models.ScanTask {
	l.t.Helper()
	task := &models.ScanTask{
		ID:        fmt.Sprintf("task-%s-%d", w.ID, atomic.AddUint64(&l.seq, 1)),
		ProjectID: "proj1",
		Tool:      core.ActionToTool[core.TaskAction(w.Action)],
		Status:    models.TaskCompleted,
		CreatedAt: time.Now().UTC(),
	}
	if err := l.queries.CreateScanTask(task); err != nil {
		l.t.Errorf("create scan task for work %s: %v", w.ID, err)
	}
	return task
}

// TestEngine_Quiescence_PoolBacklogIsFlushedBeforeStop locks in that an IP
// seed sitting in the alive pool (SchedulerTick << Tier1FlushTimeout) is
// flushed and executed while the run context is still alive, instead of the
// engine declaring completion and flushing during shutdown.
func TestEngine_Quiescence_PoolBacklogIsFlushedBeforeStop(t *testing.T) {
	var queries *db.Queries
	var linker taskLinker
	fake := &fakeExecutor{
		behavior: func(ctx context.Context, w *models.ScanWorkItem) (*toolrun.InvokeResult, error) {
			return &toolrun.InvokeResult{Task: linker.taskFor(w), Stdout: nil}, nil
		},
	}
	cfg := quiescenceTestConfig()
	engine, queries := setupTestEngine(t, fake, cfg)
	linker = taskLinker{t: t, queries: queries}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := engine.Run(ctx, []string{"10.0.0.5"}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	aliveCalls := actionCalls(fake, core.ActionAliveCheck)
	if len(aliveCalls) != 1 {
		t.Fatalf("ALIVE_CHECK executed %d times, want exactly 1 (calls: %+v)", len(aliveCalls), fake.getCalls())
	}

	works, err := queries.ListScanWorkItemsByRun("run1")
	if err != nil {
		t.Fatal(err)
	}
	if len(works) == 0 {
		t.Fatal("expected work items from flushed pools")
	}
	for _, w := range works {
		switch w.Status {
		case models.WorkStatusDone:
		default:
			t.Errorf("work %s (%s): status %s, want done (error: %s)", w.ID, w.Action, w.Status, w.Error)
		}
		if w.TaskID == nil || *w.TaskID == "" {
			t.Errorf("work %s (%s): task_id not linked", w.ID, w.Action)
		}
	}
}

// TestEngine_AliveStdoutReinjectTriggersPortScan locks the alive → port scan
// lineage: after the alive batch reports the IP up, a PORT_SCAN work item must
// be created, flushed, and executed — all before the engine stops.
func TestEngine_AliveStdoutReinjectTriggersPortScan(t *testing.T) {
	var queries *db.Queries
	var linker taskLinker
	fake := &fakeExecutor{
		behavior: func(ctx context.Context, w *models.ScanWorkItem) (*toolrun.InvokeResult, error) {
			stdout := []byte(nil)
			if core.TaskAction(w.Action) == core.ActionAliveCheck {
				stdout = []byte("Host: 10.0.0.5 (scanme)\tStatus: Up\n")
			}
			return &toolrun.InvokeResult{Task: linker.taskFor(w), Stdout: stdout}, nil
		},
	}
	cfg := quiescenceTestConfig()
	engine, queries := setupTestEngine(t, fake, cfg)
	linker = taskLinker{t: t, queries: queries}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := engine.Run(ctx, []string{"10.0.0.5"}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	aliveCalls := actionCalls(fake, core.ActionAliveCheck)
	portCalls := actionCalls(fake, core.ActionPortScan)
	if len(aliveCalls) != 1 {
		t.Fatalf("ALIVE_CHECK executed %d times, want 1", len(aliveCalls))
	}
	if len(portCalls) != 1 {
		t.Fatalf("PORT_SCAN executed %d times, want 1 (calls: %+v)", len(portCalls), fake.getCalls())
	}
	if portCalls[0] < aliveCalls[0] {
		t.Fatalf("PORT_SCAN (call %d) ran before ALIVE_CHECK (call %d)", portCalls[0], aliveCalls[0])
	}

	works, err := queries.ListScanWorkItemsByRun("run1")
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range works {
		if w.Status != models.WorkStatusDone {
			t.Errorf("work %s (%s): status %s, want done (error: %s)", w.ID, w.Action, w.Status, w.Error)
		}
	}
}

// TestEngine_PooledWorkRunsAcrossSchedulerTicks_ContextAlive locks in that a
// slow (blocking) executor crossing several scheduler ticks keeps the engine
// context alive: the engine must wait for the alive check instead of
// stopping and dispatching with an about-to-be-cancelled context.
func TestEngine_PooledWorkRunsAcrossSchedulerTicks_ContextAlive(t *testing.T) {
	var entryErr, exitErr error
	var ran bool
	fake := &fakeExecutor{
		behavior: func(ctx context.Context, w *models.ScanWorkItem) (*toolrun.InvokeResult, error) {
			if core.TaskAction(w.Action) != core.ActionAliveCheck {
				return &toolrun.InvokeResult{Task: &models.ScanTask{ID: "unlinked-task"}, Stdout: nil}, nil
			}
			ran = true
			entryErr = ctx.Err()
			time.Sleep(150 * time.Millisecond) // spans ~7 scheduler ticks
			exitErr = ctx.Err()
			return &toolrun.InvokeResult{Task: &models.ScanTask{ID: "unlinked-task"}, Stdout: nil}, nil
		},
	}
	cfg := quiescenceTestConfig()
	engine, queries := setupTestEngine(t, fake, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := engine.Run(ctx, []string{"10.0.0.5"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !ran {
		t.Fatal("ALIVE_CHECK was never executed")
	}
	if entryErr != nil {
		t.Errorf("engine context already cancelled when alive check started: %v", entryErr)
	}
	if exitErr != nil {
		t.Errorf("engine context cancelled while alive check was still running: %v", exitErr)
	}

	works, err := queries.ListScanWorkItemsByRun("run1")
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range works {
		if w.Status != models.WorkStatusDone {
			t.Errorf("work %s (%s): status %s, want done (error: %s)", w.ID, w.Action, w.Status, w.Error)
		}
	}
}

// TestEngine_CancelDiscardsPooledMembers_NoWorkGrowth locks in that an
// explicit cancel does not turn buffered pool members into new work items:
// they are discarded instead of being dispatched with a dead context.
func TestEngine_CancelDiscardsPooledMembers_NoWorkGrowth(t *testing.T) {
	fake := &fakeExecutor{
		behavior: func(ctx context.Context, w *models.ScanWorkItem) (*toolrun.InvokeResult, error) {
			return &toolrun.InvokeResult{Task: &models.ScanTask{ID: "unlinked-task"}, Stdout: nil}, nil
		},
	}
	cfg := DefaultEngineConfig()
	cfg.SchedulerTick = 1 * time.Second // no tick before cancel
	cfg.IdleTimeout = 5 * time.Second
	engine, queries := setupTestEngine(t, fake, cfg)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- engine.Run(ctx, []string{"10.0.0.5"}) }()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		if err != context.Canceled {
			t.Fatalf("Run error = %v, want context.Canceled", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Run did not return after cancel")
	}

	works, err := queries.ListScanWorkItemsByRun("run1")
	if err != nil {
		t.Fatal(err)
	}
	if len(works) != 0 {
		for _, w := range works {
			t.Logf("work %s (%s): status=%s error=%s", w.ID, w.Action, w.Status, w.Error)
		}
		t.Fatalf("cancel created %d work items from pooled members, want 0", len(works))
	}
	if engine.poolsHavePendingMembers() {
		t.Error("pools still hold pending members after cancelled shutdown")
	}
}
