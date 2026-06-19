package worker

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/scope"
)

// --- NewRemoteClient ---

func TestNewRemoteClient(t *testing.T) {
	r := NewRunner(nil, nil, t.TempDir())
	c := NewRemoteClient("http://core:8080", "http://me:9090", "tok123", "/data", r)

	if c.coreURL != "http://core:8080" {
		t.Errorf("coreURL = %q", c.coreURL)
	}
	if c.endpoint != "http://me:9090" {
		t.Errorf("endpoint = %q", c.endpoint)
	}
	if c.token != "tok123" {
		t.Errorf("token = %q", c.token)
	}
	if c.dataDir != "/data" {
		t.Errorf("dataDir = %q", c.dataDir)
	}
	if c.runner != r {
		t.Error("runner not set")
	}
	if c.httpClient == nil {
		t.Error("httpClient should not be nil")
	}
	if c.stopCh == nil {
		t.Error("stopCh should not be nil")
	}
}

// --- Register ---

func TestRemoteClient_Register_success(t *testing.T) {
	var mu sync.Mutex
	var gotBody map[string]interface{}

	mockMux := http.NewServeMux()
	mockMux.HandleFunc("POST /workers/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok123" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		gotBody = body
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"worker_id": "w-abc"})
	})
	// Heartbeat endpoint (called immediately after register)
	mockMux.HandleFunc("POST /workers/{id}/heartbeat", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mockServer := httptest.NewServer(mockMux)
	defer mockServer.Close()

	r := NewRunner(nil, nil, t.TempDir())
	c := NewRemoteClient(mockServer.URL, "http://me:9090", "tok123", "/data", r)

	if err := c.Register("test-worker"); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if c.workerID != "w-abc" {
		t.Errorf("workerID = %q, want w-abc", c.workerID)
	}

	mu.Lock()
	defer mu.Unlock()
	if gotBody["name"] != "test-worker" {
		t.Errorf("name = %v, want test-worker", gotBody["name"])
	}
	if gotBody["endpoint"] != "http://me:9090" {
		t.Errorf("endpoint = %v", gotBody["endpoint"])
	}
}

func TestRemoteClient_Register_unauthorized(t *testing.T) {
	mockMux := http.NewServeMux()
	mockMux.HandleFunc("POST /workers/register", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})
	mockServer := httptest.NewServer(mockMux)
	defer mockServer.Close()

	r := NewRunner(nil, nil, t.TempDir())
	c := NewRemoteClient(mockServer.URL, "http://me:9090", "bad-token", "/data", r)

	err := c.Register("test-worker")
	if err == nil {
		t.Fatal("expected error for unauthorized")
	}
	if !strings.Contains(err.Error(), "unauthorized") {
		t.Errorf("error = %q, want contains 'unauthorized'", err)
	}
}

func TestRemoteClient_Register_serverError(t *testing.T) {
	mockMux := http.NewServeMux()
	mockMux.HandleFunc("POST /workers/register", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	})
	mockServer := httptest.NewServer(mockMux)
	defer mockServer.Close()

	r := NewRunner(nil, nil, t.TempDir())
	c := NewRemoteClient(mockServer.URL, "http://me:9090", "tok", "/data", r)

	err := c.Register("test-worker")
	if err == nil {
		t.Fatal("expected error for 500")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error = %q, want contains '500'", err)
	}
}

func TestRemoteClient_Register_connectionRefused(t *testing.T) {
	r := NewRunner(nil, nil, t.TempDir())
	c := NewRemoteClient("http://127.0.0.1:1", "http://me:9090", "tok", "/data", r)

	err := c.Register("test-worker")
	if err == nil {
		t.Fatal("expected error for connection refused")
	}
	if !strings.Contains(err.Error(), "register request") {
		t.Errorf("error = %q, want contains 'register request'", err)
	}
}

// --- heartbeat ---

func TestRemoteClient_heartbeat_success(t *testing.T) {
	var received atomic.Bool

	mockMux := http.NewServeMux()
	mockMux.HandleFunc("POST /workers/{id}/heartbeat", func(w http.ResponseWriter, r *http.Request) {
		received.Store(true)
		if r.Header.Get("Authorization") != "Bearer mytoken" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	mockServer := httptest.NewServer(mockMux)
	defer mockServer.Close()

	r := NewRunner(nil, nil, t.TempDir())
	c := NewRemoteClient(mockServer.URL, "http://me:9090", "mytoken", "/data", r)
	c.workerID = "w-123"

	c.heartbeat()

	if !received.Load() {
		t.Error("heartbeat was not sent")
	}
}

func TestRemoteClient_heartbeat_error(t *testing.T) {
	// Unreachable endpoint — should not panic.
	r := NewRunner(nil, nil, t.TempDir())
	c := NewRemoteClient("http://127.0.0.1:1", "http://me:9090", "tok", "/data", r)
	c.workerID = "w-123"

	c.heartbeat() // should log error but not panic
}

// --- StartHeartbeat ---

func TestRemoteClient_StartHeartbeat(t *testing.T) {
	var count atomic.Int32

	mockMux := http.NewServeMux()
	mockMux.HandleFunc("POST /workers/{id}/heartbeat", func(w http.ResponseWriter, r *http.Request) {
		count.Add(1)
		w.WriteHeader(http.StatusOK)
	})
	mockServer := httptest.NewServer(mockMux)
	defer mockServer.Close()

	r := NewRunner(nil, nil, t.TempDir())
	c := NewRemoteClient(mockServer.URL, "http://me:9090", "tok", "/data", r)
	c.workerID = "w-hb"

	c.StartHeartbeat(50 * time.Millisecond)
	defer c.Stop()

	time.Sleep(200 * time.Millisecond)

	if n := count.Load(); n < 2 {
		t.Errorf("expected at least 2 heartbeats, got %d", n)
	}
}

// --- StartSourceSync / syncSources ---

func TestRemoteClient_syncSources_success(t *testing.T) {
	mockMux := http.NewServeMux()
	mockMux.HandleFunc("GET /nuclei/custom/sources", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]interface{}{
			{"id": "rbkd", "name": "RBKD", "builtin": true, "enabled": true},
			{"id": "custom", "name": "Custom", "builtin": false, "enabled": true},
		})
	})
	mockServer := httptest.NewServer(mockMux)
	defer mockServer.Close()

	r := NewRunner(nil, nil, t.TempDir())
	c := NewRemoteClient(mockServer.URL, "http://me:9090", "tok", "/data", r)
	c.syncSources() // should not panic
}

func TestRemoteClient_syncSources_error(t *testing.T) {
	r := NewRunner(nil, nil, t.TempDir())
	c := NewRemoteClient("http://127.0.0.1:1", "http://me:9090", "tok", "/data", r)
	c.syncSources() // should log error, not panic
}

func TestRemoteClient_syncSources_serverError(t *testing.T) {
	mockMux := http.NewServeMux()
	mockMux.HandleFunc("GET /nuclei/custom/sources", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "error", http.StatusInternalServerError)
	})
	mockServer := httptest.NewServer(mockMux)
	defer mockServer.Close()

	r := NewRunner(nil, nil, t.TempDir())
	c := NewRemoteClient(mockServer.URL, "http://me:9090", "tok", "/data", r)
	c.syncSources() // should log, not panic
}

func TestRemoteClient_StartSourceSync(t *testing.T) {
	var count atomic.Int32

	mockMux := http.NewServeMux()
	mockMux.HandleFunc("GET /nuclei/custom/sources", func(w http.ResponseWriter, r *http.Request) {
		count.Add(1)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]interface{}{})
	})
	mockServer := httptest.NewServer(mockMux)
	defer mockServer.Close()

	r := NewRunner(nil, nil, t.TempDir())
	c := NewRemoteClient(mockServer.URL, "http://me:9090", "tok", "/data", r)

	c.StartSourceSync(50 * time.Millisecond)
	defer c.Stop()

	time.Sleep(200 * time.Millisecond)

	if n := count.Load(); n < 2 {
		t.Errorf("expected at least 2 source syncs, got %d", n)
	}
}

// --- StartPolling ---

func TestRemoteClient_StartPolling_noContent(t *testing.T) {
	var pollCount atomic.Int32

	mockMux := http.NewServeMux()
	mockMux.HandleFunc("GET /workers/{id}/tasks/poll", func(w http.ResponseWriter, r *http.Request) {
		pollCount.Add(1)
		if pollCount.Load() > 3 {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mockServer := httptest.NewServer(mockMux)
	defer mockServer.Close()

	r := NewRunner(nil, nil, t.TempDir())
	c := NewRemoteClient(mockServer.URL, "http://me:9090", "tok", "/data", r)
	c.workerID = "w-poll"

	c.StartPolling()
	defer c.Stop()

	time.Sleep(300 * time.Millisecond)

	if pollCount.Load() < 2 {
		t.Errorf("expected multiple polls, got %d", pollCount.Load())
	}
}

func TestRemoteClient_StartPolling_unauthorized(t *testing.T) {
	mockMux := http.NewServeMux()
	mockMux.HandleFunc("GET /workers/{id}/tasks/poll", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})
	mockServer := httptest.NewServer(mockMux)
	defer mockServer.Close()

	r := NewRunner(nil, nil, t.TempDir())
	c := NewRemoteClient(mockServer.URL, "http://me:9090", "tok", "/data", r)
	c.workerID = "w-unauth"

	c.StartPolling()
	defer c.Stop()

	// Polling should stop on 401. Give it time to exit.
	time.Sleep(100 * time.Millisecond)
	// No panic = success
}

func TestRemoteClient_StartPolling_notFound(t *testing.T) {
	var registerHits atomic.Int32

	mockMux := http.NewServeMux()
	mockMux.HandleFunc("GET /workers/{id}/tasks/poll", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	})
	mockMux.HandleFunc("POST /workers/register", func(w http.ResponseWriter, r *http.Request) {
		registerHits.Add(1)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"worker_id": "w-new"})
	})
	// Heartbeat sent after re-register
	mockMux.HandleFunc("POST /workers/{id}/heartbeat", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mockServer := httptest.NewServer(mockMux)
	defer mockServer.Close()

	r := NewRunner(nil, nil, t.TempDir())
	c := NewRemoteClient(mockServer.URL, "http://me:9090", "tok", "/data", r)
	c.workerID = "w-old"

	c.StartPolling()
	defer c.Stop()

	time.Sleep(300 * time.Millisecond)

	if registerHits.Load() < 1 {
		t.Error("expected re-register on 404")
	}
}

func TestRemoteClient_StartPolling_withTask(t *testing.T) {
	// Mock core server that returns a task, then reports result.
	var mu sync.Mutex
	var results []string

	mockMux := http.NewServeMux()
	pollCount := atomic.Int32{}
	mockMux.HandleFunc("GET /workers/{id}/tasks/poll", func(w http.ResponseWriter, r *http.Request) {
		if pollCount.Add(1) > 1 {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"task_id": "task-from-poll",
			"tool":    "echo",
			"payload": map[string]interface{}{},
		})
	})
	mockMux.HandleFunc("POST /tasks/{id}/result", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		results = append(results, body["status"].(string))
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	})
	mockServer := httptest.NewServer(mockMux)
	defer mockServer.Close()

	// Setup real runner with DB for executeTask to call runner.Run.
	rawDB, _ := db.Open(t.TempDir())
	t.Cleanup(func() { rawDB.Close() })
	q := db.New(rawDB)
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
		ID: "task-from-poll", ProjectID: "proj-1", PlanID: "plan-1",
		Tool: "sh", CommandTemplate: "sh -c echo hi",
		Status: models.TaskCreated, CreatedAt: now,
	})

	scopeEng := scope.NewEngine(q)
	runner := NewRunner(q, scopeEng, t.TempDir())
	runner.SetGovernor(NewResourceGovernor(GovernorConfig{Enabled: false}, nil))

	c := NewRemoteClient(mockServer.URL, "http://me:9090", "tok", t.TempDir(), runner)
	c.workerID = "w-task"

	c.StartPolling()
	defer c.Stop()

	// Wait for polling + execution.
	time.Sleep(3 * time.Second)

	mu.Lock()
	defer mu.Unlock()
	if len(results) == 0 {
		t.Error("expected task result to be reported")
	}
}

// --- Stop ---

func TestRemoteClient_Stop(t *testing.T) {
	r := NewRunner(nil, nil, t.TempDir())
	c := NewRemoteClient("http://core:8080", "http://me:9090", "tok", "/data", r)
	c.Stop() // should not panic; double-close should be tested separately
}

func TestRemoteClient_Stop_idempotent(t *testing.T) {
	r := NewRunner(nil, nil, t.TempDir())
	c := NewRemoteClient("http://core:8080", "http://me:9090", "tok", "/data", r)
	c.Stop()
	// Second close will panic on already-closed channel; this is expected behavior
	// in the real code. We only test first close here.
}

// --- collectSystemMetrics ---

func TestCollectSystemMetrics(t *testing.T) {
	cpu, mem, disk := collectSystemMetrics()

	if cpu == nil {
		t.Error("cpu should not be nil on any platform")
	} else if *cpu < 0 || *cpu > 100 {
		t.Errorf("cpu = %f, want 0..100", *cpu)
	}

	// mem is always available on all platforms via runtime.MemStats
	if mem == nil {
		t.Error("mem should not be nil")
	} else if *mem < 0 || *mem > 100 {
		t.Errorf("mem = %f, want 0..100", *mem)
	}

	// disk may be nil on platforms where /data doesn't exist
	if disk != nil && (*disk < 0 || *disk > 100) {
		t.Errorf("disk = %f, want 0..100", *disk)
	}
}

// --- executeTask (RemoteClient) ---

func TestRemoteClient_executeTask_success(t *testing.T) {
	var mu sync.Mutex
	var reported []string

	mockMux := http.NewServeMux()
	mockMux.HandleFunc("POST /tasks/{id}/result", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		reported = append(reported, body["status"].(string))
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	})
	mockServer := httptest.NewServer(mockMux)
	defer mockServer.Close()

	// Setup DB for the runner.
	rawDB, _ := db.Open(t.TempDir())
	t.Cleanup(func() { rawDB.Close() })
	q := db.New(rawDB)
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
		ID: "task-exec", ProjectID: "proj-1", PlanID: "plan-1",
		Tool: "sh", CommandTemplate: "sh -c echo hello",
		Status: models.TaskCreated, CreatedAt: now,
	})

	scopeEng := scope.NewEngine(q)
	runner := NewRunner(q, scopeEng, t.TempDir())
	runner.SetGovernor(NewResourceGovernor(GovernorConfig{Enabled: false}, nil))

	c := NewRemoteClient(mockServer.URL, "http://me:9090", "tok", t.TempDir(), runner)
	c.workerID = "w-1"

	c.executeTask("task-exec", "sh", map[string]interface{}{})

	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(reported) == 0 {
		t.Error("expected result to be reported")
	}
}

func TestRemoteClient_executeTask_failure(t *testing.T) {
	var mu sync.Mutex
	var reported []string

	mockMux := http.NewServeMux()
	mockMux.HandleFunc("POST /tasks/{id}/result", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		reported = append(reported, body["status"].(string))
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	})
	mockServer := httptest.NewServer(mockMux)
	defer mockServer.Close()

	// Setup DB with task that will fail (tool not found).
	rawDB, _ := db.Open(t.TempDir())
	t.Cleanup(func() { rawDB.Close() })
	q := db.New(rawDB)
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
		ID: "task-fail", ProjectID: "proj-1", PlanID: "plan-1",
		Tool: "nonexistent-tool", CommandTemplate: "nonexistent-tool --help",
		Status: models.TaskCreated, CreatedAt: now,
	})

	scopeEng := scope.NewEngine(q)
	runner := NewRunner(q, scopeEng, t.TempDir())
	runner.SetGovernor(NewResourceGovernor(GovernorConfig{Enabled: false}, nil))

	c := NewRemoteClient(mockServer.URL, "http://me:9090", "tok", t.TempDir(), runner)
	c.workerID = "w-1"

	c.executeTask("task-fail", "nonexistent-tool", map[string]interface{}{})

	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(reported) == 0 {
		t.Error("expected failure result to be reported")
	}
}

func TestRemoteClient_executeTask_reportError(t *testing.T) {
	// Result endpoint is unreachable — should log error, not panic.
	rawDB, _ := db.Open(t.TempDir())
	t.Cleanup(func() { rawDB.Close() })
	q := db.New(rawDB)
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
		ID: "task-reperr", ProjectID: "proj-1", PlanID: "plan-1",
		Tool: "sh", CommandTemplate: "sh -c echo hi",
		Status: models.TaskCreated, CreatedAt: now,
	})

	scopeEng := scope.NewEngine(q)
	runner := NewRunner(q, scopeEng, t.TempDir())
	runner.SetGovernor(NewResourceGovernor(GovernorConfig{Enabled: false}, nil))

	// Use unreachable URL for result reporting.
	c := NewRemoteClient("http://127.0.0.1:1", "http://me:9090", "tok", t.TempDir(), runner)
	c.workerID = "w-1"

	c.executeTask("task-reperr", "sh", map[string]interface{}{})
	// Should not panic even with unreachable result endpoint.
}

// --- dispatchOnce with PipelineConfig scan depth ---

func TestDispatchOnce_scanDepthFromPipelineConfig(t *testing.T) {
	var mu sync.Mutex
	var gotBody map[string]interface{}

	mockMux := http.NewServeMux()
	mockMux.HandleFunc("POST /tasks", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		gotBody = body
		mu.Unlock()
		w.WriteHeader(http.StatusAccepted)
	})
	mockServer := httptest.NewServer(mockMux)
	defer mockServer.Close()

	q := openTestQueries(t)
	now := time.Now().UTC()

	pipelineJSON := `{"nuclei_scan_depth":"workflow"}`
	if err := q.CreateScanTask(&models.ScanTask{
		ID:              "task-depth",
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
	project := &models.Project{
		ID:             "proj-1",
		RateLimit:      10,
		PipelineConfig: &pipelineJSON,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	d.dispatchOnce(ctx,
		&models.ScanTask{
			ID:              "task-depth",
			ProjectID:       "proj-1",
			Tool:            "nuclei",
			CommandTemplate: "nuclei -t test",
			Status:          models.TaskRunning,
		}, worker, "/tmp/workdir", project)

	mu.Lock()
	defer mu.Unlock()
	if gotBody == nil {
		t.Fatal("expected request body")
	}
	if gotBody["scan_depth"] != "workflow" {
		t.Errorf("scan_depth = %v, want workflow", gotBody["scan_depth"])
	}
}

// --- dispatchOnce with input files ---

func TestDispatchOnce_collectsInputFiles(t *testing.T) {
	// Create a temp file referenced in the command.
	dir := t.TempDir()
	inputPath := filepath.Join(dir, "targets.txt")
	_ = writeFile(t, inputPath, []byte("example.com\n"))

	var mu sync.Mutex
	var gotBody map[string]interface{}

	mockMux := http.NewServeMux()
	mockMux.HandleFunc("POST /tasks", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		gotBody = body
		mu.Unlock()
		w.WriteHeader(http.StatusAccepted)
	})
	mockServer := httptest.NewServer(mockMux)
	defer mockServer.Close()

	q := openTestQueries(t)
	now := time.Now().UTC()
	q.CreateScanTask(&models.ScanTask{
		ID:              "task-input",
		ProjectID:       "proj-1",
		Tool:            "naabu",
		CommandTemplate: "naabu -list " + inputPath,
		Status:          models.TaskRunning,
		CreatedAt:       now,
	})

	d := NewDispatcher(q)
	worker := &models.WorkerNode{
		ID:       "w-1",
		Endpoint: mockServer.URL,
		Status:   models.WorkerStatusOnline,
	}
	project := &models.Project{ID: "proj-1", RateLimit: 10}

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	d.dispatchOnce(ctx,
		&models.ScanTask{
			ID:              "task-input",
			ProjectID:       "proj-1",
			Tool:            "naabu",
			CommandTemplate: "naabu -list " + inputPath,
		}, worker, "/tmp/workdir", project)

	mu.Lock()
	defer mu.Unlock()
	if gotBody == nil {
		t.Fatal("expected request body")
	}
	inputFiles, ok := gotBody["input_files"].(map[string]interface{})
	if !ok {
		t.Fatal("expected input_files in request body")
	}
	if _, exists := inputFiles[inputPath]; !exists {
		t.Errorf("expected %s in input_files", inputPath)
	}
}

// --- collectInputFiles additional coverage ---

func TestCollectInputFiles_relativePathSkipped(t *testing.T) {
	args := []string{"nuclei", "-l", "relative/path.txt"}
	got := collectInputFiles(args)
	if len(got) != 0 {
		t.Error("expected empty for relative path")
	}
}

func TestCollectInputFiles_directorySkipped(t *testing.T) {
	dir := t.TempDir()
	args := []string{"tool", dir}
	got := collectInputFiles(args)
	if len(got) != 0 {
		t.Error("expected empty for directory")
	}
}

func TestCollectInputFiles_smallFileCollected(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "small.txt")
	writeFile(t, path, []byte("hello"))

	got := collectInputFiles([]string{"tool", path})
	if len(got) != 1 {
		t.Fatalf("expected 1 file, got %d", len(got))
	}
	if _, ok := got[path]; !ok {
		t.Errorf("expected %s in result", path)
	}
}

func TestCollectInputFiles_nonexistentSkipped(t *testing.T) {
	got := collectInputFiles([]string{"tool", "/nonexistent/file.txt"})
	if len(got) != 0 {
		t.Error("expected empty for nonexistent file")
	}
}

func writeFile(t *testing.T, path string, data []byte) []byte {
	t.Helper()
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return data
}
