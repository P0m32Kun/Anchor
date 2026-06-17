package worker

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/P0m32Kun/Anchor/internal/toolguard"
)

// WorkerServer runs in worker mode, receiving tasks via HTTP and executing them.
type WorkerServer struct {
	dataDir        string
	coreURL        string
	token          string
	httpClient     *http.Client
	governor       *ResourceGovernor
	allowlist      *toolguard.Allowlist
	procs          map[string]*exec.Cmd
	workdirs       map[string]string // taskID -> workdir for live output tail
	maxConcurrency int
	runningTasks   atomic.Int32
	mu             sync.Mutex
}

func NewWorkerServer(dataDir string, coreURL string, token string) *WorkerServer {
	return &WorkerServer{
		dataDir:        dataDir,
		coreURL:        coreURL,
		token:          token,
		httpClient:     &http.Client{Timeout: 30 * time.Second},
		governor:       NewResourceGovernor(LoadGovernorConfigFromEnv(), nil),
		allowlist:      toolguard.NewAllowlist(),
		procs:          make(map[string]*exec.Cmd),
		workdirs:       make(map[string]string),
		maxConcurrency: LoadMaxConcurrencyFromEnv(),
	}
}

// SetGovernor swaps the resource governor (used by tests).
func (ws *WorkerServer) SetGovernor(g *ResourceGovernor) {
	ws.governor = g
}

func (ws *WorkerServer) atCapacity() bool {
	return ws.maxConcurrency > 0 && ws.runningTasks.Load() >= int32(ws.maxConcurrency)
}

func (ws *WorkerServer) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /tasks", ws.handleTask)
	mux.HandleFunc("POST /tasks/{id}/cancel", ws.handleCancelTask)
	mux.HandleFunc("POST /tasks/{id}/progress", ws.handleProgress)
	mux.HandleFunc("GET /tasks/{id}/output", ws.handleTaskOutput)
	mux.HandleFunc("POST /tasks/{id}/result", ws.handleResult)
	mux.HandleFunc("GET /health", ws.handleHealth)
	mux.HandleFunc("GET /files/{path...}", ws.handleFile)
}

func (ws *WorkerServer) handleTask(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TaskID     string            `json:"task_id"`
		Tool       string            `json:"tool"`
		Command    []string          `json:"command"`
		Workdir    string            `json:"workdir"`
		RateLimit  int               `json:"rate_limit"`
		InputFiles map[string]string `json:"input_files"`
		ScanDepth  string            `json:"scan_depth"` // "workflow" | "tags" | "both"
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Materialise input files (sent by the dispatcher) onto the worker's
	// filesystem at the absolute paths referenced in the command. This lets
	// tools like `naabu -list /data/.../hosts.txt` work even though the file
	// was created in the server container.
	for path, b64 := range req.InputFiles {
		data, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			log.Printf("[worker] task %s decode input file %s: %v", req.TaskID, path, err)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
			log.Printf("[worker] task %s mkdir input dir %s: %v", req.TaskID, filepath.Dir(path), err)
			continue
		}
		if err := os.WriteFile(path, data, 0640); err != nil {
			log.Printf("[worker] task %s write input file %s: %v", req.TaskID, path, err)
			continue
		}
		log.Printf("[worker] task %s materialised input file: %s (%dB)", req.TaskID, path, len(data))
	}

	if ws.atCapacity() {
		http.Error(w, "worker at capacity", http.StatusServiceUnavailable)
		return
	}
	ws.runningTasks.Add(1)

	// Execute task asynchronously
	// Use background context so task execution survives HTTP connection close.
	go func() {
		defer ws.runningTasks.Add(-1)
		ws.executeTask(context.Background(), req.TaskID, req.Tool, req.Command, req.Workdir, req.RateLimit, req.ScanDepth)
	}()

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
}

func (ws *WorkerServer) executeTask(ctx context.Context, taskID, tool string, command []string, workdir string, rateLimit int, scanDepth string) {
	log.Printf("[worker] executeTask begin: taskID=%s tool=%s workdir=%s command=%v", taskID, tool, workdir, command)

	// Resource governance: block on memory pressure / delay on CPU pressure
	// before starting the subprocess. Failing open if sampling errors.
	if err := ws.governor.Acquire(ctx); err != nil {
		log.Printf("[worker] task %s aborted by governor: %v", taskID, err)
		ws.reportResult(taskID, "failed", nil, fmt.Sprintf("governor: %v", err))
		return
	}

	if workdir == "" {
		workdir = filepath.Join(ws.dataDir, "workdirs", taskID)
		log.Printf("[worker] task %s using default workdir: %s", taskID, workdir)
	}
	if err := os.MkdirAll(workdir, 0750); err != nil {
		log.Printf("[worker] task %s mkdir failed: %v", taskID, err)
		ws.reportResult(taskID, "failed", nil, fmt.Sprintf("create workdir: %v", err))
		return
	}
	log.Printf("[worker] task %s workdir ready: %s", taskID, workdir)

	if len(command) == 0 {
		log.Printf("[worker] task %s empty command", taskID)
		ws.reportResult(taskID, "failed", nil, "empty command")
		return
	}

	// Inject custom nuclei templates if available
	if tool == "nuclei" {
		command = ws.injectCustomNucleiTemplates(command, taskID, scanDepth)
	}

	// Handle shell commands: "sh -c <rest...>" needs the rest joined as a
	// single argument so sh interprets it as a command string.
	if len(command) > 2 && command[0] == "sh" && command[1] == "-c" {
		command = []string{"sh", "-c", strings.Join(command[2:], " ")}
	}

	binary := command[0]
	if _, err := exec.LookPath(binary); err != nil {
		log.Printf("[worker] task %s tool not found: %s", taskID, binary)
		ws.reportResult(taskID, "failed", nil, fmt.Sprintf("tool not found: %s", binary))
		return
	}
	log.Printf("[worker] task %s binary resolved: %s", taskID, binary)

	// Apply rate limit
	if rateLimit > 0 {
		command = appendRateLimitArgs(command, tool, rateLimit)
		log.Printf("[worker] task %s rate limit applied: %d", taskID, rateLimit)
	}

	// Allowlist: reject unknown binaries or args with shell metacharacters.
	if err := ws.allowlist.Validate(binary, command[1:]); err != nil {
		log.Printf("[worker] task %s allowlist rejected command: %v", taskID, err)
		ws.reportResult(taskID, "failed", nil, fmt.Sprintf("allowlist: %v", err))
		return
	}

	cmd := exec.Command(binary, command[1:]...)
	cmd.Dir = workdir
	cmd.Env = os.Environ()
	cmd.WaitDelay = 30 * time.Second // Go 1.20+: force-close unresponsive IO after exit

	stdoutW, err := newTaskOutputWriter(workdir, "stdout")
	if err != nil {
		log.Printf("[worker] task %s stdout writer: %v", taskID, err)
		ws.reportResult(taskID, "failed", nil, fmt.Sprintf("prepare stdout: %v", err))
		return
	}
	defer stdoutW.Close()
	stderrW, err := newTaskOutputWriter(workdir, "stderr")
	if err != nil {
		log.Printf("[worker] task %s stderr writer: %v", taskID, err)
		ws.reportResult(taskID, "failed", nil, fmt.Sprintf("prepare stderr: %v", err))
		return
	}
	defer stderrW.Close()

	cmd.Stdout = stdoutW
	cmd.Stderr = stderrW

	ws.mu.Lock()
	ws.procs[taskID] = cmd
	ws.workdirs[taskID] = workdir
	ws.mu.Unlock()

	log.Printf("[worker] task %s exec: %s %v", taskID, binary, command[1:])

	startTime := time.Now()
	err = cmd.Run()
	elapsed := time.Since(startTime)

	ws.mu.Lock()
	delete(ws.procs, taskID)
	delete(ws.workdirs, taskID)
	ws.mu.Unlock()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
			log.Printf("[worker] task %s exit error: code=%d elapsed=%v", taskID, exitCode, elapsed.Round(time.Second))
		} else {
			exitCode = -1
			log.Printf("[worker] task %s run error: %v elapsed=%v", taskID, err, elapsed.Round(time.Second))
		}
	} else {
		log.Printf("[worker] task %s exited cleanly elapsed=%v", taskID, elapsed.Round(time.Second))
	}

	log.Printf("[worker] task %s stdout=%dB stderr=%dB", taskID, stdoutW.Len(), stderrW.Len())

	// Collect artifacts
	artifacts := []map[string]interface{}{}

	// stdout artifact
	if stdoutW.Len() > 0 {
		artifacts = append(artifacts, map[string]interface{}{
			"type": "stdout",
			"data": stdoutW.Bytes(),
		})
	}

	// stderr artifact
	if stderrW.Len() > 0 {
		artifacts = append(artifacts, map[string]interface{}{
			"type": "stderr",
			"data": stderrW.Bytes(),
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
	errorMsg := ""
	if err != nil && exitCode != 0 {
		status = "failed"
		stderrData := stderrW.Bytes()
		if len(stderrData) > 0 {
			errorMsg = string(stderrData)
			if len(errorMsg) > 500 {
				errorMsg = errorMsg[:500] + "..."
			}
		} else {
			errorMsg = err.Error()
		}
	}

	ws.reportResult(taskID, status, artifacts, errorMsg)
}

func (ws *WorkerServer) reportResult(taskID, status string, artifacts []map[string]interface{}, errorMsg string) {
	result := map[string]interface{}{
		"task_id":   taskID,
		"status":    status,
		"artifacts": artifacts,
		"error":     errorMsg,
	}
	data, _ := json.Marshal(result)
	
	// Write result file (always)
	resultPath := filepath.Join(ws.dataDir, "workdirs", taskID, "_result.json")
	if err := os.MkdirAll(filepath.Dir(resultPath), 0750); err != nil {
		log.Printf("[worker] task %s mkdir for result failed: %v", taskID, err)
	}
	if err := os.WriteFile(resultPath, data, 0640); err != nil {
		log.Printf("[worker] task %s write result failed: %v", taskID, err)
	} else {
		log.Printf("[worker] task %s result written: %s (%dB)", taskID, resultPath, len(data))
	}
	
	// Report to core server if coreURL is set
	if ws.coreURL != "" {
		log.Printf("[worker] task %s reporting to core: %s", taskID, ws.coreURL)
		req, _ := http.NewRequest("POST", fmt.Sprintf("%s/tasks/%s/result", ws.coreURL, taskID), bytes.NewReader(data))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+ws.token)
		resp, err := ws.httpClient.Do(req)
		if err != nil {
			log.Printf("[worker] task %s report result to core failed: %v", taskID, err)
		} else {
			log.Printf("[worker] task %s core reported: %d", taskID, resp.StatusCode)
			resp.Body.Close()
		}
	} else {
		log.Printf("[worker] task %s no coreURL, skipping core report", taskID)
	}
	
	log.Printf("[worker] task %s finished: %s artifacts=%d error=%q", taskID, status, len(artifacts), errorMsg)
}

// injectCustomNucleiTemplates is a no-op. All custom templates now live
// under ~/nuclei-templates/ (nuclei's default search path), so nuclei finds
// them natively without -t or -w injection. The pipeline passes precise
// -w paths for workflow invocations.
func (ws *WorkerServer) injectCustomNucleiTemplates(command []string, taskID string, scanDepth string) []string {
	return command
}

func (ws *WorkerServer) handleProgress(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (ws *WorkerServer) handleTaskOutput(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	stream := r.URL.Query().Get("stream")
	offsetStr := r.URL.Query().Get("offset")

	stream, err := validateOutputStream(stream)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var offset int64
	if offsetStr != "" {
		if _, err := fmt.Sscanf(offsetStr, "%d", &offset); err != nil {
			http.Error(w, "invalid offset", http.StatusBadRequest)
			return
		}
	}

	ws.mu.Lock()
	workdir := ws.workdirs[taskID]
	ws.mu.Unlock()
	if workdir == "" {
		workdir = filepath.Join(ws.dataDir, "workdirs", taskID)
	}

	content, next, atEOF, err := ReadTaskOutput(workdir, stream, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ws.mu.Lock()
	_, running := ws.procs[taskID]
	ws.mu.Unlock()
	done := atEOF && !running

	writeJSON(w, map[string]interface{}{
		"stream":  stream,
		"offset":  next,
		"content": content,
		"done":    done,
	})
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func (ws *WorkerServer) handleResult(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (ws *WorkerServer) handleCancelTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	ws.mu.Lock()
	cmd, ok := ws.procs[taskID]
	ws.mu.Unlock()

	if !ok || cmd == nil || cmd.Process == nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"status": "not_found"})
		return
	}

	// Send SIGTERM, wait briefly, then SIGKILL.
	_ = cmd.Process.Signal(syscall.SIGTERM)

	done := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		_ = cmd.Process.Kill()
		<-done
	}

	ws.mu.Lock()
	delete(ws.procs, taskID)
	ws.mu.Unlock()

	log.Printf("[worker] task %s cancelled by server request", taskID)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "cancelled"})
}

func (ws *WorkerServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	tools := []map[string]string{}
	for _, name := range []string{"subfinder", "httpx", "naabu", "nuclei", "nmap"} {
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




