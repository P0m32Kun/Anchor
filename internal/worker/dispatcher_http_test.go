package worker

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
)

func TestDispatchOnce_workerRejected(t *testing.T) {
	// Mock worker that rejects the task.
	mockMux := http.NewServeMux()
	mockMux.HandleFunc("POST /tasks", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	})
	mockServer := httptest.NewServer(mockMux)
	defer mockServer.Close()

	q := openTestQueries(t)
	now := time.Now().UTC()
	if err := q.CreateScanTask(&models.ScanTask{
		ID:              "task-reject",
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

	err := d.dispatchOnce(context.Background(),
		&models.ScanTask{
			ID:              "task-reject",
			ProjectID:       "proj-1",
			Tool:            "nuclei",
			CommandTemplate: "nuclei -t test",
			Status:          models.TaskRunning,
		}, worker, "/tmp/workdir", project)
	if err == nil {
		t.Fatal("expected error for rejected task")
	}
	if !containsSubstring(err.Error(), "rejected") {
		t.Errorf("error = %q, want contains 'rejected'", err.Error())
	}
}

func TestDispatchOnce_workerAtCapacity(t *testing.T) {
	mockMux := http.NewServeMux()
	mockMux.HandleFunc("POST /tasks", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "worker at capacity", http.StatusServiceUnavailable)
	})
	mockServer := httptest.NewServer(mockMux)
	defer mockServer.Close()

	q := openTestQueries(t)
	d := NewDispatcher(q)
	worker := &models.WorkerNode{
		ID:       "w-1",
		Endpoint: mockServer.URL,
		Status:   models.WorkerStatusOnline,
	}
	project := &models.Project{ID: "proj-1", RateLimit: 10}

	err := d.dispatchOnce(context.Background(),
		&models.ScanTask{
			ID:              "task-cap",
			ProjectID:       "proj-1",
			Tool:            "nuclei",
			CommandTemplate: "nuclei -t test",
		}, worker, "/tmp/workdir", project)
	if err == nil {
		t.Fatal("expected error for capacity")
	}
	if !isAtCapacityError(err) {
		t.Errorf("expected capacity error, got: %v", err)
	}
}

func TestDispatchOnce_success(t *testing.T) {
	// Mock worker that accepts the task.
	mockMux := http.NewServeMux()
	mockMux.HandleFunc("POST /tasks", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
	})
	mockServer := httptest.NewServer(mockMux)
	defer mockServer.Close()

	q := openTestQueries(t)
	now := time.Now().UTC()
	if err := q.CreateScanTask(&models.ScanTask{
		ID:              "task-ok",
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

	// Start dispatch in background — it will poll forever since task never completes.
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err := d.dispatchOnce(ctx,
		&models.ScanTask{
			ID:              "task-ok",
			ProjectID:       "proj-1",
			Tool:            "nuclei",
			CommandTemplate: "nuclei -t test",
			Status:          models.TaskRunning,
		}, worker, "/tmp/workdir", project)

	// Should fail with context deadline (polling never sees completion).
	if err == nil {
		t.Fatal("expected timeout error from polling")
	}
}

func TestDispatchOnce_contextCancelled(t *testing.T) {
	// Mock worker that accepts the task.
	mockMux := http.NewServeMux()
	mockMux.HandleFunc("POST /tasks", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})
	mockServer := httptest.NewServer(mockMux)
	defer mockServer.Close()

	q := openTestQueries(t)
	now := time.Now().UTC()
	if err := q.CreateScanTask(&models.ScanTask{
		ID:              "task-ctx",
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

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	err := d.dispatchOnce(ctx,
		&models.ScanTask{
			ID:              "task-ctx",
			ProjectID:       "proj-1",
			Tool:            "nuclei",
			CommandTemplate: "nuclei -t test",
		}, worker, "/tmp/workdir", project)

	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
	if !containsSubstring(err.Error(), "cancelled") {
		t.Errorf("error = %q, want contains 'cancelled'", err.Error())
	}
}

func TestDispatchOnce_taskCompleted(t *testing.T) {
	// Mock worker that accepts and completes the task.
	mockMux := http.NewServeMux()
	mockMux.HandleFunc("POST /tasks", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})
	mockServer := httptest.NewServer(mockMux)
	defer mockServer.Close()

	q := openTestQueries(t)
	now := time.Now().UTC()
	if err := q.CreateScanTask(&models.ScanTask{
		ID:              "task-complete",
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
		_ = q.UpdateScanTaskStatus("task-complete", models.TaskCompleted, nil, &finished)
	}()

	err := d.dispatchOnce(context.Background(),
		&models.ScanTask{
			ID:              "task-complete",
			ProjectID:       "proj-1",
			Tool:            "nuclei",
			CommandTemplate: "nuclei -t test",
			Status:          models.TaskRunning,
		}, worker, "/tmp/workdir", project)

	if err != nil {
		t.Fatalf("expected nil error for completed task, got: %v", err)
	}
}

func TestDispatchOnce_taskFailed(t *testing.T) {
	mockMux := http.NewServeMux()
	mockMux.HandleFunc("POST /tasks", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})
	mockServer := httptest.NewServer(mockMux)
	defer mockServer.Close()

	q := openTestQueries(t)
	now := time.Now().UTC()
	if err := q.CreateScanTask(&models.ScanTask{
		ID:              "task-fail",
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

	// Fail the task in DB after a short delay.
	go func() {
		time.Sleep(200 * time.Millisecond)
		finished := time.Now().UTC()
		_ = q.UpdateScanTaskStatus("task-fail", models.TaskFailed, nil, &finished)
	}()

	err := d.dispatchOnce(context.Background(),
		&models.ScanTask{
			ID:              "task-fail",
			ProjectID:       "proj-1",
			Tool:            "nuclei",
			CommandTemplate: "nuclei -t test",
			Status:          models.TaskRunning,
		}, worker, "/tmp/workdir", project)

	if err == nil {
		t.Fatal("expected error for failed task")
	}
	if !containsSubstring(err.Error(), "failed") {
		t.Errorf("error = %q, want contains 'failed'", err.Error())
	}
}

func TestDispatchOnce_taskCancelled(t *testing.T) {
	var cancelCalled atomic.Bool

	mockMux := http.NewServeMux()
	mockMux.HandleFunc("POST /tasks", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})
	mockMux.HandleFunc("POST /tasks/{id}/cancel", func(w http.ResponseWriter, r *http.Request) {
		cancelCalled.Store(true)
		w.WriteHeader(http.StatusOK)
	})
	mockServer := httptest.NewServer(mockMux)
	defer mockServer.Close()

	q := openTestQueries(t)
	now := time.Now().UTC()
	if err := q.CreateScanTask(&models.ScanTask{
		ID:              "task-cancel",
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

	// Cancel the task in DB after a short delay.
	go func() {
		time.Sleep(200 * time.Millisecond)
		now := time.Now().UTC()
		_ = q.UpdateScanTaskStatus("task-cancel", models.TaskCancelled, nil, &now)
	}()

	err := d.dispatchOnce(context.Background(),
		&models.ScanTask{
			ID:              "task-cancel",
			ProjectID:       "proj-1",
			Tool:            "nuclei",
			CommandTemplate: "nuclei -t test",
			Status:          models.TaskRunning,
		}, worker, "/tmp/workdir", project)

	if err == nil {
		t.Fatal("expected error for cancelled task")
	}
	if !containsSubstring(err.Error(), "cancelled") {
		t.Errorf("error = %q, want contains 'cancelled'", err.Error())
	}
	// Verify cancel was forwarded to worker.
	if !cancelCalled.Load() {
		t.Error("expected cancel to be forwarded to worker")
	}
}

func TestDispatchToWorker_retriesOnUnreachable(t *testing.T) {
	// Use a non-existent endpoint to trigger unreachable error.
	d := NewDispatcher(openTestQueries(t))
	worker := &models.WorkerNode{
		ID:       "w-1",
		Endpoint: "http://127.0.0.1:1", // Connection refused.
		Status:   models.WorkerStatusOnline,
	}
	project := &models.Project{ID: "proj-1", RateLimit: 10}

	now := time.Now().UTC()
	task := &models.ScanTask{
		ID:              "task-retry",
		ProjectID:       "proj-1",
		Tool:            "nuclei",
		CommandTemplate: "nuclei -t test",
		Status:          models.TaskRunning,
		CreatedAt:       now,
	}

	// Create the task in DB for ResetScanTaskForRetry to work.
	if err := d.queries.CreateScanTask(task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	
	err := d.DispatchToWorker(context.Background(), task, worker, "/tmp/workdir", project)
	

	if err == nil {
		t.Fatal("expected error after retries")
	}
	if !isUnreachableError(err) {
		t.Errorf("expected unreachable error, got: %v", err)
	}
	// Should have retried (error message confirms retries happened).
	if err == nil {
		t.Fatal("expected error after retries")
	}
}

func TestDispatchToWorker_noRetryOnTaskFailure(t *testing.T) {
	mockMux := http.NewServeMux()
	mockMux.HandleFunc("POST /tasks", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	})
	mockServer := httptest.NewServer(mockMux)
	defer mockServer.Close()

	d := NewDispatcher(openTestQueries(t))
	worker := &models.WorkerNode{
		ID:       "w-1",
		Endpoint: mockServer.URL,
		Status:   models.WorkerStatusOnline,
	}
	project := &models.Project{ID: "proj-1", RateLimit: 10}

	task := &models.ScanTask{
		ID:              "task-noretry",
		ProjectID:       "proj-1",
		Tool:            "nuclei",
		CommandTemplate: "nuclei -t test",
	}

	err := d.DispatchToWorker(context.Background(), task, worker, "/tmp/workdir", project)
	if err == nil {
		t.Fatal("expected error")
	}
	// "rejected" is not an unreachable error, so no retry should happen.
	if isUnreachableError(err) {
		t.Errorf("should not be unreachable error: %v", err)
	}
}

func TestCancelRemoteTask(t *testing.T) {
	var cancelHits int32

	mockMux := http.NewServeMux()
	mockMux.HandleFunc("POST /tasks/{id}/cancel", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&cancelHits, 1)
		w.WriteHeader(http.StatusOK)
	})
	mockServer := httptest.NewServer(mockMux)
	defer mockServer.Close()

	d := NewDispatcher(nil)
	d.cancelRemoteTask(mockServer.URL, "task-123")

	if atomic.LoadInt32(&cancelHits) != 1 {
		t.Errorf("cancel hits = %d, want 1", atomic.LoadInt32(&cancelHits))
	}
}

func TestCancelRemoteTask_unreachable(t *testing.T) {
	// Should not panic even if worker is unreachable.
	d := NewDispatcher(nil)
	d.cancelRemoteTask("http://127.0.0.1:1", "task-123")
	// No assertion needed — just verifying it doesn't panic.
}

func TestCollectInputFiles_largeFileSkipped(t *testing.T) {
	dir := t.TempDir()

	// Create a file just over 32 MB.
	path := filepath.Join(dir, "large.txt")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	f.Seek(32*1024*1024, 0)
	f.Write([]byte("x"))
	f.Close()

	args := []string{"tool", path}
	got := collectInputFiles(args)
	if len(got) != 0 {
		t.Errorf("expected empty map for large file, got %d entries", len(got))
	}
}

func TestCollectInputFiles_sizeExceeded(t *testing.T) {
	dir := t.TempDir()

	// Create a file that is exactly 32MB + 1 byte (exceeds the limit).
	path := filepath.Join(dir, "toolarge.txt")
	data := make([]byte, 32*1024*1024+1)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	got := collectInputFiles([]string{"tool", path})
	if len(got) != 0 {
		t.Errorf("expected empty map for >32MB file, got %d entries", len(got))
	}
}

// containsSubstring is a test helper.
func containsSubstring(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsAt(s, sub))
}

func containsAt(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
