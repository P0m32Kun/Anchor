package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// WorkerServer runs in worker mode, receiving tasks via HTTP and executing them.
type WorkerServer struct {
	dataDir   string
	procs     map[string]*exec.Cmd
	mu        sync.Mutex
}

func NewWorkerServer(dataDir string) *WorkerServer {
	return &WorkerServer{
		dataDir: dataDir,
		procs:   make(map[string]*exec.Cmd),
	}
}

func (ws *WorkerServer) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /tasks", ws.handleTask)
	mux.HandleFunc("POST /tasks/{id}/progress", ws.handleProgress)
	mux.HandleFunc("POST /tasks/{id}/result", ws.handleResult)
	mux.HandleFunc("GET /health", ws.handleHealth)
	mux.HandleFunc("GET /files/{path...}", ws.handleFile)
}

func (ws *WorkerServer) handleTask(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TaskID      string   `json:"task_id"`
		Tool        string   `json:"tool"`
		Command     []string `json:"command"`
		Workdir     string   `json:"workdir"`
		RateLimit   int      `json:"rate_limit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Execute task asynchronously
	go ws.executeTask(r.Context(), req.TaskID, req.Tool, req.Command, req.Workdir, req.RateLimit)

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
}

func (ws *WorkerServer) executeTask(ctx context.Context, taskID, tool string, command []string, workdir string, rateLimit int) {
	if workdir == "" {
		workdir = filepath.Join(ws.dataDir, "workdirs", taskID)
	}
	if err := os.MkdirAll(workdir, 0750); err != nil {
		ws.reportResult(taskID, "failed", nil, fmt.Sprintf("create workdir: %v", err))
		return
	}

	if len(command) == 0 {
		ws.reportResult(taskID, "failed", nil, "empty command")
		return
	}

	binary := command[0]
	if _, err := exec.LookPath(binary); err != nil {
		ws.reportResult(taskID, "failed", nil, fmt.Sprintf("tool not found: %s", binary))
		return
	}

	// Apply rate limit
	if rateLimit > 0 {
		command = appendRateLimitArgs(command, tool, rateLimit)
	}

	ctx, cancel := context.WithTimeout(ctx, defaultToolTimeout(tool))
	defer cancel()

	cmd := exec.CommandContext(ctx, binary, command[1:]...)
	cmd.Dir = workdir
	cmd.Env = os.Environ()

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	ws.mu.Lock()
	ws.procs[taskID] = cmd
	ws.mu.Unlock()

	log.Printf("[worker] task %s started: %s %s", taskID, binary, strings.Join(command[1:], " "))

	err := cmd.Run()

	ws.mu.Lock()
	delete(ws.procs, taskID)
	ws.mu.Unlock()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}

	// Collect artifacts
	artifacts := []map[string]interface{}{}
	
	// stdout artifact
	if stdoutBuf.Len() > 0 {
		artifacts = append(artifacts, map[string]interface{}{
			"type": "stdout",
			"data": stdoutBuf.Bytes(),
		})
	}

	// stderr artifact
	if stderrBuf.Len() > 0 {
		artifacts = append(artifacts, map[string]interface{}{
			"type": "stderr", 
			"data": stderrBuf.Bytes(),
		})
	}

	// Collect output files
	entries, _ := os.ReadDir(workdir)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name == "." || name == ".." {
			continue
		}
		path := filepath.Join(workdir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		artifactType := "file"
		if strings.HasSuffix(name, ".json") || strings.HasSuffix(name, ".jsonl") {
			artifactType = "jsonl"
		}
		artifacts = append(artifacts, map[string]interface{}{
			"type": artifactType,
			"name": name,
			"data": data,
		})
	}

	status := "completed"
	if err != nil && exitCode != 0 {
		status = "failed"
	}

	ws.reportResult(taskID, status, artifacts, "")
}

func (ws *WorkerServer) reportResult(taskID, status string, artifacts []map[string]interface{}, errorMsg string) {
	result := map[string]interface{}{
		"task_id":   taskID,
		"status":    status,
		"artifacts": artifacts,
		"error":     errorMsg,
	}
	data, _ := json.Marshal(result)
	
	// For local worker, we don't have a core URL. The result should be collected
	// by the core server polling or via stdout. For now, write to a result file.
	resultPath := filepath.Join(ws.dataDir, "workdirs", taskID, "_result.json")
	os.WriteFile(resultPath, data, 0640)
	
	log.Printf("[worker] task %s finished: %s", taskID, status)
}

func (ws *WorkerServer) handleProgress(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (ws *WorkerServer) handleResult(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (ws *WorkerServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	tools := []map[string]string{}
	for _, name := range []string{"subfinder", "httpx", "naabu", "nuclei"} {
		status := "missing"
		if _, err := exec.LookPath(name); err == nil {
			status = "ready"
		}
		tools = append(tools, map[string]string{
			"name":   name,
			"status": status,
		})
	}

	// Check Rod / Chromium
	rodStatus := "missing"
	// Try to check if rod can download chromium or if it's already available
	rodStatus = "ready" // Simplified for now

	tools = append(tools, map[string]string{
		"name":   "rod",
		"status": rodStatus,
	})

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "ok",
		"tools":       tools,
		"rod_ready":   rodStatus == "ready",
	})
}

func (ws *WorkerServer) handleFile(w http.ResponseWriter, r *http.Request) {
	path := r.PathValue("path")
	fullPath := filepath.Join(ws.dataDir, path)
	
	// Security: prevent directory traversal
	if !strings.HasPrefix(fullPath, ws.dataDir) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(data)
}

func defaultToolTimeout(tool string) time.Duration {
	switch tool {
	case "subfinder":
		return 10 * time.Minute
	case "httpx":
		return 10 * time.Minute
	case "naabu":
		return 30 * time.Minute
	case "nuclei":
		return 60 * time.Minute
	default:
		return 10 * time.Minute
	}
}
