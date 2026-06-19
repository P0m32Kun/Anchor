package worker

import (
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/models"
)

func openTestQueries(t *testing.T) *db.Queries {
	t.Helper()
	raw, err := db.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = raw.Close() })
	q := db.New(raw)
	now := time.Now().UTC()
	if err := q.CreateProject(&models.Project{
		ID: "proj-1", Name: "test", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create project: %v", err)
	}
	return q
}

func seedWorker(t *testing.T, q *db.Queries, id, endpoint string, maxConcurrency int) {
	t.Helper()
	now := time.Now().UTC()
	if err := q.CreateWorkerNode(&models.WorkerNode{
		ID:             id,
		Name:           id,
		Endpoint:       endpoint,
		Mode:           models.WorkerModeRemote,
		Status:         models.WorkerStatusOnline,
		TrustLevel:     "standard",
		MaxConcurrency: maxConcurrency,
		LastSeen:       &now,
		CreatedAt:      now,
	}); err != nil {
		t.Fatalf("create worker %s: %v", id, err)
	}
}

func seedRunningTask(t *testing.T, q *db.Queries, taskID, workerID string) {
	t.Helper()
	now := time.Now().UTC()
	if err := q.CreateScanTask(&models.ScanTask{
		ID:              taskID,
		ProjectID:       "proj-1",
		Tool:            "dnsx",
		CommandTemplate: "dnsx -d example.com",
		Status:          models.TaskCreated,
		CreatedAt:       now,
	}); err != nil {
		t.Fatalf("create task %s: %v", taskID, err)
	}
	if err := q.SetScanTaskRunning(taskID, now); err != nil {
		t.Fatalf("set running %s: %v", taskID, err)
	}
	if err := q.SetScanTaskWorker(taskID, workerID); err != nil {
		t.Fatalf("set worker %s: %v", taskID, err)
	}
}

func TestPickLeastLoaded_prefersLowerLoad(t *testing.T) {
	q := openTestQueries(t)
	seedWorker(t, q, "w-a", "http://worker-a:8080", 10)
	seedWorker(t, q, "w-b", "http://worker-b:8080", 10)
	seedRunningTask(t, q, "task-1", "w-a")

	d := NewDispatcher(q)
	got := d.pickLeastLoaded(nil)
	if got == nil || got.ID != "w-b" {
		t.Fatalf("expected w-b, got %v", got)
	}
}

func TestPickLeastLoaded_roundRobinAmongEqualLoad(t *testing.T) {
	q := openTestQueries(t)
	seedWorker(t, q, "w-a", "http://worker-a:8080", 10)
	seedWorker(t, q, "w-b", "http://worker-b:8080", 10)

	d := NewDispatcher(q)
	seen := map[string]int{}
	for i := 0; i < 6; i++ {
		w := d.PickOnlineWorkerExcluding(nil)
		if w == nil {
			t.Fatal("expected worker")
		}
		seen[w.ID]++
		d.ReleaseWorker(w.ID)
	}
	if seen["w-a"] == 0 || seen["w-b"] == 0 {
		t.Fatalf("expected both workers picked, got %v", seen)
	}
	if seen["w-a"] != seen["w-b"] {
		t.Fatalf("expected even round-robin split, got %v", seen)
	}
}

func TestPickLeastLoaded_newWorkerReceivesWork(t *testing.T) {
	q := openTestQueries(t)
	seedWorker(t, q, "w-a", "http://worker-a:8080", 10)
	seedWorker(t, q, "w-b", "http://worker-b:8080", 10)

	d := NewDispatcher(q)
	d.trackInflight("w-a", 2)
	d.trackInflight("w-b", 2)

	seedWorker(t, q, "w-new", "http://worker-new:8080", 10)
	got := d.pickLeastLoaded(nil)
	if got == nil || got.ID != "w-new" {
		t.Fatalf("expected new idle worker w-new, got %v", got)
	}
}

func TestPickLeastLoaded_respectsMaxConcurrency(t *testing.T) {
	q := openTestQueries(t)
	seedWorker(t, q, "w-full", "http://worker-full:8080", 2)
	seedWorker(t, q, "w-free", "http://worker-free:8080", 10)
	seedRunningTask(t, q, "task-1", "w-full")
	seedRunningTask(t, q, "task-2", "w-full")

	d := NewDispatcher(q)
	got := d.pickLeastLoaded(nil)
	if got == nil || got.ID != "w-free" {
		t.Fatalf("expected w-free, got %v", got)
	}
}

func TestReleaseWorker_clearsInflight(t *testing.T) {
	q := openTestQueries(t)
	d := NewDispatcher(q)
	d.trackInflight("w-a", 1)
	d.ReleaseWorker("w-a")
	if d.currentInflight("w-a") != 0 {
		t.Fatalf("expected inflight 0 after release, got %d", d.currentInflight("w-a"))
	}
}

func TestWorkerAtCapacity(t *testing.T) {
	w := &models.WorkerNode{MaxConcurrency: 3}
	if workerAtCapacity(w, 2) {
		t.Fatal("load 2 should be under cap 3")
	}
	if !workerAtCapacity(w, 3) {
		t.Fatal("load 3 should be at cap")
	}
	w.MaxConcurrency = 0
	if workerAtCapacity(w, 100) {
		t.Fatal("max_concurrency 0 means unlimited")
	}
}

// --- isEligible ---

func TestIsEligible(t *testing.T) {
	d := NewDispatcher(nil) // queries not needed for isEligible
	exclude := map[string]struct{}{}

	now := time.Now().UTC()

	tests := []struct {
		name   string
		w      *models.WorkerNode
		excl   map[string]struct{}
		expect bool
	}{
		{
			name:   "nil worker",
			w:      nil,
			excl:   exclude,
			expect: false,
		},
		{
			name:   "empty endpoint",
			w:      &models.WorkerNode{ID: "w-1", Endpoint: "", Status: models.WorkerStatusOnline},
			excl:   exclude,
			expect: false,
		},
		{
			name:   "revoked worker",
			w:      &models.WorkerNode{ID: "w-1", Endpoint: "http://x:8080", Status: models.WorkerStatusOnline, RevokedAt: &now},
			excl:   exclude,
			expect: false,
		},
		{
			name:   "offline status",
			w:      &models.WorkerNode{ID: "w-1", Endpoint: "http://x:8080", Status: models.WorkerStatusOffline},
			excl:   exclude,
			expect: false,
		},
		{
			name:   "error status",
			w:      &models.WorkerNode{ID: "w-1", Endpoint: "http://x:8080", Status: models.WorkerStatusError},
			excl:   exclude,
			expect: false,
		},
		{
			name:   "online status — eligible",
			w:      &models.WorkerNode{ID: "w-1", Endpoint: "http://x:8080", Status: models.WorkerStatusOnline},
			excl:   exclude,
			expect: true,
		},
		{
			name:   "busy status — eligible",
			w:      &models.WorkerNode{ID: "w-1", Endpoint: "http://x:8080", Status: models.WorkerStatusBusy},
			excl:   exclude,
			expect: true,
		},
		{
			name:   "excluded ID",
			w:      &models.WorkerNode{ID: "w-1", Endpoint: "http://x:8080", Status: models.WorkerStatusOnline},
			excl:   map[string]struct{}{"w-1": {}},
			expect: false,
		},
		{
			name:   "non-excluded ID — eligible",
			w:      &models.WorkerNode{ID: "w-2", Endpoint: "http://x:8080", Status: models.WorkerStatusOnline},
			excl:   map[string]struct{}{"w-1": {}},
			expect: true,
		},
		{
			name:   "empty status",
			w:      &models.WorkerNode{ID: "w-1", Endpoint: "http://x:8080", Status: ""},
			excl:   exclude,
			expect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := d.isEligible(tt.w, tt.excl)
			if got != tt.expect {
				t.Errorf("isEligible = %v, want %v", got, tt.expect)
			}
		})
	}
}

func TestPickOnlineWorker_noWorkers(t *testing.T) {
	q := openTestQueries(t)
	d := NewDispatcher(q)
	got := d.PickOnlineWorker()
	if got != nil {
		t.Fatalf("expected nil when no workers registered, got %v", got)
	}
}

func TestPickOnlineWorker_allRevoked(t *testing.T) {
	q := openTestQueries(t)
	seedWorker(t, q, "w-a", "http://worker-a:8080", 10)
	seedWorker(t, q, "w-b", "http://worker-b:8080", 10)

	now := time.Now().UTC()
	if err := q.RevokeWorkerNode("w-a", now); err != nil {
		t.Fatalf("revoke w-a: %v", err)
	}
	if err := q.RevokeWorkerNode("w-b", now); err != nil {
		t.Fatalf("revoke w-b: %v", err)
	}

	d := NewDispatcher(q)
	got := d.PickOnlineWorker()
	if got != nil {
		t.Fatalf("expected nil when all workers revoked, got %v", got)
	}
}

func TestPickOnlineWorker_allOffline(t *testing.T) {
	q := openTestQueries(t)
	seedWorker(t, q, "w-a", "http://worker-a:8080", 10)
	seedWorker(t, q, "w-b", "http://worker-b:8080", 10)

	now := time.Now().UTC()
	if err := q.UpdateWorkerNodeStatus("w-a", models.WorkerStatusOffline, now); err != nil {
		t.Fatalf("mark w-a offline: %v", err)
	}
	if err := q.UpdateWorkerNodeStatus("w-b", models.WorkerStatusOffline, now); err != nil {
		t.Fatalf("mark w-b offline: %v", err)
	}

	d := NewDispatcher(q)
	got := d.PickOnlineWorker()
	if got != nil {
		t.Fatalf("expected nil when all workers offline, got %v", got)
	}
}

func TestMarkWorkerOffline(t *testing.T) {
	q := openTestQueries(t)
	seedWorker(t, q, "w-a", "http://worker-a:8080", 10)

	d := NewDispatcher(q)
	if err := d.MarkWorkerOffline("w-a"); err != nil {
		t.Fatalf("MarkWorkerOffline: %v", err)
	}

	// Verify the worker is now offline and no longer picked.
	got := d.PickOnlineWorker()
	if got != nil {
		t.Fatalf("expected nil after marking worker offline, got %v", got)
	}

	// Verify DB status directly.
	w, err := q.GetWorkerNode("w-a")
	if err != nil {
		t.Fatalf("GetWorkerNode: %v", err)
	}
	if w.Status != models.WorkerStatusOffline {
		t.Errorf("status = %q, want %q", w.Status, models.WorkerStatusOffline)
	}
}
