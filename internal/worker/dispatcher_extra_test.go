package worker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
)

// --- DispatchToWorker additional coverage ---

func TestDispatchToWorker_taskCompletedViaPoll(t *testing.T) {
	// Mock worker accepts the task.
	mockMux := http.NewServeMux()
	mockMux.HandleFunc("POST /tasks", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})
	mockServer := httptest.NewServer(mockMux)
	defer mockServer.Close()

	q := openTestQueries(t)
	now := time.Now().UTC()
	if err := q.CreateScanTask(&models.ScanTask{
		ID:              "task-dispatch-ok",
		ProjectID:       "proj-1",
		Tool:            "nuclei",
		CommandTemplate: "nuclei -t test",
		Status:          models.TaskRunning,
		CreatedAt:       now,
	}); err != nil {
		t.Fatalf("create task: %v", err)
	}

	d := NewDispatcher(q)
	worker := &models.WorkerNode{
		ID:       "w-1",
		Endpoint: mockServer.URL,
		Status:   models.WorkerStatusOnline,
	}
	project := &models.Project{ID: "proj-1", RateLimit: 10}

	// Complete the task in DB after a short delay.
	go func() {
		time.Sleep(200 * time.Millisecond)
		finished := time.Now().UTC()
		_ = q.UpdateScanTaskStatus("task-dispatch-ok", models.TaskCompleted, nil, &finished)
	}()

	err := d.DispatchToWorker(context.Background(),
		&models.ScanTask{
			ID:              "task-dispatch-ok",
			ProjectID:       "proj-1",
			Tool:            "nuclei",
			CommandTemplate: "nuclei -t test",
			Status:          models.TaskRunning,
		}, worker, "/tmp/workdir", project)

	if err != nil {
		t.Fatalf("DispatchToWorker: %v", err)
	}
}

func TestDispatchToWorker_resetFailsOnRetry(t *testing.T) {
	// Use a non-existent endpoint to trigger unreachable.
	d := NewDispatcher(openTestQueries(t))
	worker := &models.WorkerNode{
		ID:       "w-1",
		Endpoint: "http://127.0.0.1:1",
		Status:   models.WorkerStatusOnline,
	}
	project := &models.Project{ID: "proj-1", RateLimit: 10}

	// Do NOT create the task in DB — ResetScanTaskForRetry will fail.
	task := &models.ScanTask{
		ID:              "task-reset-fail",
		ProjectID:       "proj-1",
		Tool:            "nuclei",
		CommandTemplate: "nuclei -t test",
	}

	err := d.DispatchToWorker(context.Background(), task, worker, "/tmp/workdir", project)
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- dispatchOnce scope denied path ---

func TestDispatchOnce_scopeDenied(t *testing.T) {
	mockMux := http.NewServeMux()
	mockMux.HandleFunc("POST /tasks", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})
	mockServer := httptest.NewServer(mockMux)
	defer mockServer.Close()

	q := openTestQueries(t)
	now := time.Now().UTC()
	if err := q.CreateScanTask(&models.ScanTask{
		ID:              "task-scope",
		ProjectID:       "proj-1",
		Tool:            "nuclei",
		CommandTemplate: "nuclei -t test",
		Status:          models.TaskRunning,
		CreatedAt:       now,
	}); err != nil {
		t.Fatalf("create task: %v", err)
	}

	d := NewDispatcher(q)
	worker := &models.WorkerNode{
		ID:       "w-1",
		Endpoint: mockServer.URL,
		Status:   models.WorkerStatusOnline,
	}
	project := &models.Project{ID: "proj-1", RateLimit: 10}

	// Mark task as scope denied after a short delay.
	go func() {
		time.Sleep(200 * time.Millisecond)
		now := time.Now().UTC()
		_ = q.UpdateScanTaskStatus("task-scope", models.TaskScopeDenied, nil, &now)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := d.dispatchOnce(ctx,
		&models.ScanTask{
			ID:              "task-scope",
			ProjectID:       "proj-1",
			Tool:            "nuclei",
			CommandTemplate: "nuclei -t test",
		}, worker, "/tmp/workdir", project)

	if err == nil {
		t.Fatal("expected error for scope denied")
	}
	if !containsSubstring(err.Error(), "cancelled") && !containsSubstring(err.Error(), "scope_denied") {
		t.Errorf("error = %q, want contains 'cancelled' or 'scope_denied'", err)
	}
}

// --- dispatchOnce with network error ---

func TestDispatchOnce_networkError(t *testing.T) {
	d := NewDispatcher(openTestQueries(t))
	worker := &models.WorkerNode{
		ID:       "w-1",
		Endpoint: "http://127.0.0.1:1", // Connection refused.
		Status:   models.WorkerStatusOnline,
	}
	project := &models.Project{ID: "proj-1", RateLimit: 10}

	err := d.dispatchOnce(context.Background(),
		&models.ScanTask{
			ID:              "task-net-err",
			ProjectID:       "proj-1",
			Tool:            "nuclei",
			CommandTemplate: "nuclei -t test",
		}, worker, "/tmp/workdir", project)

	if err == nil {
		t.Fatal("expected error for network error")
	}
	if !isUnreachableError(err) {
		t.Errorf("expected unreachable error: %v", err)
	}
}

// --- collectInputFiles additional ---

func TestCollectInputFiles_multipleFiles(t *testing.T) {
	dir := t.TempDir()
	path1 := filepath.Join(dir, "a.txt")
	path2 := filepath.Join(dir, "b.txt")
	os.WriteFile(path1, []byte("aaa"), 0644)
	os.WriteFile(path2, []byte("bbb"), 0644)

	got := collectInputFiles([]string{"tool", path1, path2})
	if len(got) != 2 {
		t.Errorf("expected 2 files, got %d", len(got))
	}
}

func TestCollectInputFiles_emptyArgs(t *testing.T) {
	got := collectInputFiles(nil)
	if len(got) != 0 {
		t.Error("expected empty for nil args")
	}
}

func TestCollectInputFiles_noArgs(t *testing.T) {
	got := collectInputFiles([]string{"tool"})
	if len(got) != 0 {
		t.Error("expected empty for args with no file paths")
	}
}

// --- trackInflight edge cases ---

func TestTrackInflight_emptyWorkerID(t *testing.T) {
	d := NewDispatcher(nil)
	d.trackInflight("", 1) // should not panic
	if d.currentInflight("") != 0 {
		t.Error("empty workerID should not be tracked")
	}
}

func TestTrackInflight_negativeGoesToZero(t *testing.T) {
	d := NewDispatcher(nil)
	d.trackInflight("w-1", 1)
	d.trackInflight("w-1", -2) // more decrements than increments
	if d.currentInflight("w-1") != 0 {
		t.Errorf("inflight = %d, want 0 (cleaned up)", d.currentInflight("w-1"))
	}
}
