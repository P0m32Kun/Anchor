package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/toolguard"
)

// newTestWorkerServer creates a WorkerServer for handler tests with governor disabled.
func newTestWorkerServer(t *testing.T) (*WorkerServer, string) {
	t.Helper()
	dataDir := t.TempDir()
	ws := &WorkerServer{
		dataDir:        dataDir,
		coreURL:        "",
		token:          "",
		httpClient:     &http.Client{Timeout: 5 * time.Second},
		governor:       NewResourceGovernor(GovernorConfig{Enabled: false}, nil),
		allowlist:      toolguard.NewAllowlist(),
		procs:          make(map[string]*exec.Cmd),
		workdirs:       make(map[string]string),
		maxConcurrency: 10,
	}
	return ws, dataDir
}

// newTestWorkerServerMux creates a mux with all WorkerServer routes registered.
func newTestWorkerServerMux(t *testing.T, ws *WorkerServer) *http.ServeMux {
	t.Helper()
	mux := http.NewServeMux()
	ws.Register(mux)
	return mux
}

func TestNewWorkerServer(t *testing.T) {
	ws := NewWorkerServer("/tmp/test", "http://core:8080", "tok123")
	if ws.dataDir != "/tmp/test" {
		t.Errorf("dataDir = %q, want %q", ws.dataDir, "/tmp/test")
	}
	if ws.coreURL != "http://core:8080" {
		t.Errorf("coreURL = %q, want %q", ws.coreURL, "http://core:8080")
	}
	if ws.token != "tok123" {
		t.Errorf("token = %q, want %q", ws.token, "tok123")
	}
	if ws.httpClient == nil {
		t.Error("httpClient should not be nil")
	}
	if ws.governor == nil {
		t.Error("governor should not be nil")
	}
	if ws.allowlist == nil {
		t.Error("allowlist should not be nil")
	}
	if ws.procs == nil {
		t.Error("procs should not be nil")
	}
	if ws.workdirs == nil {
		t.Error("workdirs should not be nil")
	}
}

func TestWorkerServer_SetGovernor(t *testing.T) {
	ws, _ := newTestWorkerServer(t)
	customGov := NewResourceGovernor(GovernorConfig{Enabled: false}, nil)
	ws.SetGovernor(customGov)
	if ws.governor != customGov {
		t.Error("SetGovernor should swap the governor")
	}
}

func TestWorkerServer_handleHealth(t *testing.T) {
	ws, _ := newTestWorkerServer(t)
	mux := newTestWorkerServerMux(t, ws)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result["status"] != "ok" {
		t.Errorf("status = %v, want ok", result["status"])
	}
	tools, ok := result["tools"].([]interface{})
	if !ok {
		t.Fatal("tools should be an array")
	}
	if len(tools) == 0 {
		t.Error("tools array should not be empty")
	}
}

func TestWorkerServer_handleTask_accepted(t *testing.T) {
	ws, _ := newTestWorkerServer(t)
	mux := newTestWorkerServerMux(t, ws)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	body, _ := json.Marshal(map[string]interface{}{
		"task_id":    "task-1",
		"tool":       "echo",
		"command":    []string{"/bin/echo", "hello"},
		"workdir":    "",
		"scan_depth": "",
	})
	resp, err := http.Post(ts.URL+"/tasks", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /tasks: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusAccepted)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result["status"] != "accepted" {
		t.Errorf("response status = %q, want accepted", result["status"])
	}
}

func TestWorkerServer_handleTask_badRequest(t *testing.T) {
	ws, _ := newTestWorkerServer(t)
	mux := newTestWorkerServerMux(t, ws)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/tasks", "application/json", bytes.NewReader([]byte("not json")))
	if err != nil {
		t.Fatalf("POST /tasks: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestWorkerServer_handleTask_atCapacity(t *testing.T) {
	ws, _ := newTestWorkerServer(t)
	ws.maxConcurrency = 1
	ws.runningTasks.Store(1)
	mux := newTestWorkerServerMux(t, ws)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	body, _ := json.Marshal(map[string]interface{}{
		"task_id": "task-1",
		"tool":    "echo",
		"command": []string{"/bin/echo"},
	})
	resp, err := http.Post(ts.URL+"/tasks", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /tasks: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusServiceUnavailable)
	}
}

func TestWorkerServer_handleTask_executionReportsToCore(t *testing.T) {
	var mu sync.Mutex
	var reported []map[string]interface{}

	coreMux := http.NewServeMux()
	coreMux.HandleFunc("POST /tasks/{id}/result", func(w http.ResponseWriter, r *http.Request) {
		var result map[string]interface{}
		json.NewDecoder(r.Body).Decode(&result)
		mu.Lock()
		reported = append(reported, result)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	})
	coreServer := httptest.NewServer(coreMux)
	defer coreServer.Close()

	ws, _ := newTestWorkerServer(t)
	ws.coreURL = coreServer.URL
	ws.token = "test-token"
	mux := newTestWorkerServerMux(t, ws)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	body, _ := json.Marshal(map[string]interface{}{
		"task_id":    "task-exec-1",
		"tool":       "echo",
		"command":    []string{"/bin/echo", "hello"},
		"workdir":    "",
		"scan_depth": "",
	})
	resp, err := http.Post(ts.URL+"/tasks", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /tasks: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusAccepted)
	}

	// Wait for async execution to complete.
	time.Sleep(2 * time.Second)

	mu.Lock()
	defer mu.Unlock()
	if len(reported) == 0 {
		t.Fatal("expected result to be reported to core server")
	}
	result := reported[0]
	if result["status"] != "failed" {
		t.Errorf("reported status = %v, want failed (allowlist rejects echo)", result["status"])
	}
}

func TestWorkerServer_handleCancelTask_notFound(t *testing.T) {
	ws, _ := newTestWorkerServer(t)
	mux := newTestWorkerServerMux(t, ws)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/tasks/nonexistent/cancel", "", nil)
	if err != nil {
		t.Fatalf("POST /cancel: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestWorkerServer_handleCancelTask_runningProcess(t *testing.T) {
	ws, dataDir := newTestWorkerServer(t)
	mux := newTestWorkerServerMux(t, ws)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	// Create a long-running process to cancel.
	workdir := filepath.Join(dataDir, "workdirs", "task-cancel")
	if err := os.MkdirAll(workdir, 0750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/bin/sleep", "30")

	if err := cmd.Start(); err != nil {
		t.Fatalf("start process: %v", err)
	}

	ws.mu.Lock()
	ws.procs["task-cancel"] = cmd
	ws.workdirs["task-cancel"] = workdir
	ws.mu.Unlock()

	t.Cleanup(func() {
		ws.mu.Lock()
		delete(ws.procs, "task-cancel")
		delete(ws.workdirs, "task-cancel")
		ws.mu.Unlock()
		if cmd.Process != nil {
			cmd.Process.Kill()
			cmd.Wait()
		}
	})

	resp, err := http.Post(ts.URL+"/tasks/task-cancel/cancel", "", nil)
	if err != nil {
		t.Fatalf("POST /cancel: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	if result["status"] != "cancelled" {
		t.Errorf("response status = %q, want cancelled", result["status"])
	}
}

func TestWorkerServer_handleProgress(t *testing.T) {
	ws, _ := newTestWorkerServer(t)
	mux := newTestWorkerServerMux(t, ws)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/tasks/task-1/progress", "", nil)
	if err != nil {
		t.Fatalf("POST /progress: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestWorkerServer_handleTaskOutput_stdout(t *testing.T) {
	ws, dataDir := newTestWorkerServer(t)
	mux := newTestWorkerServerMux(t, ws)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	workdir := filepath.Join(dataDir, "workdirs", "task-output")
	if err := os.MkdirAll(workdir, 0750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workdir, "stdout.txt"), []byte("hello output"), 0640); err != nil {
		t.Fatalf("write stdout: %v", err)
	}

	resp, err := http.Get(ts.URL + "/tasks/task-output/output?stream=stdout&offset=0")
	if err != nil {
		t.Fatalf("GET /output: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result["stream"] != "stdout" {
		t.Errorf("stream = %v, want stdout", result["stream"])
	}
	if result["content"] != "hello output" {
		t.Errorf("content = %v, want 'hello output'", result["content"])
	}
}

func TestWorkerServer_handleTaskOutput_noFile(t *testing.T) {
	ws, _ := newTestWorkerServer(t)
	mux := newTestWorkerServerMux(t, ws)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/tasks/nonexistent/output?stream=stdout&offset=0")
	if err != nil {
		t.Fatalf("GET /output: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["content"] != "" {
		t.Errorf("content should be empty for missing file, got %v", result["content"])
	}
}

func TestWorkerServer_handleTaskOutput_invalidStream(t *testing.T) {
	ws, _ := newTestWorkerServer(t)
	mux := newTestWorkerServerMux(t, ws)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/tasks/task-1/output?stream=invalid")
	if err != nil {
		t.Fatalf("GET /output: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestWorkerServer_handleTaskOutput_invalidOffset(t *testing.T) {
	ws, _ := newTestWorkerServer(t)
	mux := newTestWorkerServerMux(t, ws)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/tasks/task-1/output?stream=stdout&offset=abc")
	if err != nil {
		t.Fatalf("GET /output: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestWorkerServer_handleFile(t *testing.T) {
	ws, dataDir := newTestWorkerServer(t)
	mux := newTestWorkerServerMux(t, ws)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	testData := []byte("file content here")
	path := filepath.Join(dataDir, "test.txt")
	if err := os.WriteFile(path, testData, 0640); err != nil {
		t.Fatalf("write file: %v", err)
	}

	resp, err := http.Get(ts.URL + "/files/test.txt")
	if err != nil {
		t.Fatalf("GET /files: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var buf bytes.Buffer
	buf.ReadFrom(resp.Body)
	if !bytes.Equal(buf.Bytes(), testData) {
		t.Errorf("body = %q, want %q", buf.Bytes(), testData)
	}
}

func TestWorkerServer_handleFile_notFound(t *testing.T) {
	ws, _ := newTestWorkerServer(t)
	mux := newTestWorkerServerMux(t, ws)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/files/nonexistent.txt")
	if err != nil {
		t.Fatalf("GET /files: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestWorkerServer_handleResult(t *testing.T) {
	ws, _ := newTestWorkerServer(t)
	mux := newTestWorkerServerMux(t, ws)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	body, _ := json.Marshal(map[string]interface{}{
		"task_id": "task-1",
		"status":  "completed",
	})
	resp, err := http.Post(ts.URL+"/tasks/task-1/result", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /result: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestWorkerServer_reportResult_writesFile(t *testing.T) {
	ws, dataDir := newTestWorkerServer(t)

	artifacts := []map[string]interface{}{
		{"type": "stdout", "data": []byte("output")},
	}
	ws.reportResult("task-report", "completed", artifacts, "")

	resultPath := filepath.Join(dataDir, "workdirs", "task-report", "_result.json")
	data, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatalf("read result file: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result["task_id"] != "task-report" {
		t.Errorf("task_id = %v, want task-report", result["task_id"])
	}
	if result["status"] != "completed" {
		t.Errorf("status = %v, want completed", result["status"])
	}
}

func TestWorkerServer_reportResult_reportsToCoreServer(t *testing.T) {
	var mu sync.Mutex
	var received map[string]interface{}

	coreMux := http.NewServeMux()
	coreMux.HandleFunc("POST /tasks/{id}/result", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		received = body
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	})
	coreServer := httptest.NewServer(coreMux)
	defer coreServer.Close()

	ws, _ := newTestWorkerServer(t)
	ws.coreURL = coreServer.URL
	ws.token = "test-token"

	ws.reportResult("task-core", "failed", nil, "something went wrong")

	// Give the HTTP request time to complete.
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if received == nil {
		t.Fatal("expected result to be reported to core server")
	}
	if received["task_id"] != "task-core" {
		t.Errorf("task_id = %v, want task-core", received["task_id"])
	}
	if received["status"] != "failed" {
		t.Errorf("status = %v, want failed", received["status"])
	}
	if received["error"] != "something went wrong" {
		t.Errorf("error = %v, want 'something went wrong'", received["error"])
	}
}

func TestWorkerServer_injectCustomNucleiTemplates_noop(t *testing.T) {
	ws, _ := newTestWorkerServer(t)
	cmd := []string{"nuclei", "-t", "test.yaml"}
	got := ws.injectCustomNucleiTemplates(cmd, "task-1", "tags")
	if len(got) != len(cmd) {
		t.Errorf("expected no-op, got different length")
	}
}

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(w, map[string]string{"key": "value"})

	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", w.Header().Get("Content-Type"))
	}

	var result map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result["key"] != "value" {
		t.Errorf("key = %q, want value", result["key"])
	}
}
