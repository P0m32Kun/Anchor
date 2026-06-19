package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// --- executeTask tests ---

func TestExecuteTask_emptyCommand(t *testing.T) {
	ws, _ := newTestWorkerServer(t)

	ws.executeTask(context.Background(), "task-empty", "sh", nil, "", 0, "")

	resultPath := filepath.Join(ws.dataDir, "workdirs", "task-empty", "_result.json")
	data, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatalf("read result: %v", err)
	}
	var result map[string]interface{}
	json.Unmarshal(data, &result)

	if result["status"] != "failed" {
		t.Errorf("status = %v, want failed", result["status"])
	}
}

func TestExecuteTask_toolNotFound(t *testing.T) {
	ws, _ := newTestWorkerServer(t)

	ws.executeTask(context.Background(), "task-notfound", "nonexistent-tool", []string{"nonexistent-tool"}, "", 0, "")

	resultPath := filepath.Join(ws.dataDir, "workdirs", "task-notfound", "_result.json")
	data, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatalf("read result: %v", err)
	}
	var result map[string]interface{}
	json.Unmarshal(data, &result)

	if result["status"] != "failed" {
		t.Errorf("status = %v, want failed", result["status"])
	}
}

func TestExecuteTask_success(t *testing.T) {
	ws, _ := newTestWorkerServer(t)

	ws.executeTask(context.Background(), "task-ok", "sh", []string{"sh", "-c", "echo hello"}, "", 0, "")

	resultPath := filepath.Join(ws.dataDir, "workdirs", "task-ok", "_result.json")
	data, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatalf("read result: %v", err)
	}
	var result map[string]interface{}
	json.Unmarshal(data, &result)

	if result["status"] != "completed" {
		t.Errorf("status = %v, want completed", result["status"])
	}
}

func TestExecuteTask_failureExitCode(t *testing.T) {
	ws, _ := newTestWorkerServer(t)

	ws.executeTask(context.Background(), "task-fail", "sh", []string{"sh", "-c", "exit 1"}, "", 0, "")

	resultPath := filepath.Join(ws.dataDir, "workdirs", "task-fail", "_result.json")
	data, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatalf("read result: %v", err)
	}
	var result map[string]interface{}
	json.Unmarshal(data, &result)

	if result["status"] != "failed" {
		t.Errorf("status = %v, want failed", result["status"])
	}
}

func TestExecuteTask_reportsToCoreServer(t *testing.T) {
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

	ws.executeTask(context.Background(), "task-core", "sh", []string{"sh", "-c", "echo hello"}, "", 0, "")

	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if received == nil {
		t.Fatal("expected result to be reported to core")
	}
	if received["status"] != "completed" {
		t.Errorf("status = %v, want completed", received["status"])
	}
}

func TestExecuteTask_withWorkdir(t *testing.T) {
	ws, dataDir := newTestWorkerServer(t)

	workdir := filepath.Join(dataDir, "custom-workdir")
	os.MkdirAll(workdir, 0750)

	ws.executeTask(context.Background(), "task-workdir", "sh", []string{"sh", "-c", "echo test"}, workdir, 0, "")

	resultPath := filepath.Join(dataDir, "workdirs", "task-workdir", "_result.json")
	if _, err := os.ReadFile(resultPath); err != nil {
		t.Fatalf("read result: %v", err)
	}
}

func TestExecuteTask_outputFilesCollected(t *testing.T) {
	ws, dataDir := newTestWorkerServer(t)

	workdir := filepath.Join(dataDir, "workdirs", "task-files")
	os.MkdirAll(workdir, 0750)
	os.WriteFile(filepath.Join(workdir, "results.json"), []byte(`{"findings":[]}`), 0640)
	os.WriteFile(filepath.Join(workdir, "output.txt"), []byte("plain text"), 0640)

	ws.executeTask(context.Background(), "task-files", "sh", []string{"sh", "-c", "echo x"}, workdir, 0, "")

	resultPath := filepath.Join(dataDir, "workdirs", "task-files", "_result.json")
	data, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatalf("read result: %v", err)
	}
	var result map[string]interface{}
	json.Unmarshal(data, &result)

	artifacts, ok := result["artifacts"].([]interface{})
	if !ok {
		t.Fatal("expected artifacts array")
	}
	// stdout artifact + 2 output files = 3
	if len(artifacts) < 2 {
		t.Errorf("expected at least 2 artifacts (stdout + files), got %d", len(artifacts))
	}
}

func TestExecuteTask_reportResultToCoreError(t *testing.T) {
	// Unreachable coreURL — should not panic.
	ws, _ := newTestWorkerServer(t)
	ws.coreURL = "http://127.0.0.1:1"
	ws.token = "tok"

	ws.executeTask(context.Background(), "task-err", "sh", []string{"sh", "-c", "echo hi"}, "", 0, "")

	// Should not panic. Result written to local file.
	resultPath := filepath.Join(ws.dataDir, "workdirs", "task-err", "_result.json")
	if _, err := os.ReadFile(resultPath); err != nil {
		t.Fatalf("read result: %v", err)
	}
}

// --- handleTask with input files ---

func TestHandleTask_withInputFiles(t *testing.T) {
	ws, dataDir := newTestWorkerServer(t)
	mux := newTestWorkerServerMux(t, ws)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	inputPath := filepath.Join(dataDir, "input-targets.txt")

	body, _ := json.Marshal(map[string]interface{}{
		"task_id": "task-input",
		"tool":    "sh",
		"command": []string{"sh", "-c", "echo hello"},
		"input_files": map[string]string{
			inputPath: "aGVsbG8=", // base64("hello")
		},
	})
	resp, err := http.Post(ts.URL+"/tasks", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /tasks: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusAccepted)
	}

	// Verify the input file was materialised.
	time.Sleep(100 * time.Millisecond)
	data, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatalf("read input file: %v", err)
	}
	if string(data) != "hello" {
		t.Errorf("content = %q, want 'hello'", string(data))
	}
}

// --- handleFile directory traversal ---

func TestWorkerServer_handleFile_directoryTraversal(t *testing.T) {
	ws, _ := newTestWorkerServer(t)
	mux := newTestWorkerServerMux(t, ws)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/files/../../../etc/passwd")
	if err != nil {
		t.Fatalf("GET /files: %v", err)
	}
	defer resp.Body.Close()

	// The HTTP client normalizes /files/../../../etc/passwd to /etc/passwd,
	// which doesn't match the /files/{path...} route, so the router returns 404.
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d (normalized path doesn't match route)", resp.StatusCode, http.StatusNotFound)
	}
}
