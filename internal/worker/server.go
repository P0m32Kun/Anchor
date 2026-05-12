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
	"time"
)

// WorkerServer runs in worker mode, receiving tasks via HTTP and executing them.
type WorkerServer struct {
	dataDir    string
	coreURL    string
	token      string
	httpClient *http.Client
	procs      map[string]*exec.Cmd
	mu         sync.Mutex
}

func NewWorkerServer(dataDir string, coreURL string, token string) *WorkerServer {
	return &WorkerServer{
		dataDir:    dataDir,
		coreURL:    coreURL,
		token:      token,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		procs:      make(map[string]*exec.Cmd),
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
		TaskID     string            `json:"task_id"`
		Tool       string            `json:"tool"`
		Command    []string          `json:"command"`
		Workdir    string            `json:"workdir"`
		RateLimit  int               `json:"rate_limit"`
		InputFiles map[string]string `json:"input_files"`
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

	// Execute task asynchronously
	// Use background context so task execution survives HTTP connection close.
	go ws.executeTask(context.Background(), req.TaskID, req.Tool, req.Command, req.Workdir, req.RateLimit)

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
}

func (ws *WorkerServer) executeTask(ctx context.Context, taskID, tool string, command []string, workdir string, rateLimit int) {
	log.Printf("[worker] executeTask begin: taskID=%s tool=%s workdir=%s command=%v", taskID, tool, workdir, command)

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
		command = ws.injectCustomNucleiTemplates(command, taskID)
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

	ctx, cancel := context.WithTimeout(ctx, defaultToolTimeout(tool))
	defer cancel()

	cmd := exec.CommandContext(ctx, binary, command[1:]...)
	cmd.Dir = workdir
	cmd.Env = os.Environ()

	stdout := &idleWatchedWriter{}
	stderr := &idleWatchedWriter{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	ws.mu.Lock()
	ws.procs[taskID] = cmd
	ws.mu.Unlock()

	log.Printf("[worker] task %s exec: %s %v", taskID, binary, command[1:])

	startTime := time.Now()
	stopWatchdog := make(chan struct{})
	idleKilled := make(chan struct{}, 1)
	idleThreshold := idleOutputTimeout(tool)
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-stopWatchdog:
				return
			case <-ticker.C:
				last := stdout.Last()
				if l2 := stderr.Last(); l2.After(last) {
					last = l2
				}
				if last.IsZero() {
					last = startTime
				}
				since := time.Since(last)
				if since > idleThreshold {
					log.Printf("[worker] task %s idle for %v (threshold %v), killing process", taskID, since.Round(time.Second), idleThreshold)
					select {
					case idleKilled <- struct{}{}:
					default:
					}
					cancel()
					return
				}
			}
		}
	}()

	err := cmd.Run()
	close(stopWatchdog)

	wasIdleKilled := false
	select {
	case <-idleKilled:
		wasIdleKilled = true
	default:
	}

	stdoutBuf := stdout.Bytes()
	stderrBuf := stderr.Bytes()

	ws.mu.Lock()
	delete(ws.procs, taskID)
	ws.mu.Unlock()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
			log.Printf("[worker] task %s exit error: code=%d", taskID, exitCode)
		} else {
			exitCode = -1
			log.Printf("[worker] task %s run error: %v", taskID, err)
		}
	} else {
		log.Printf("[worker] task %s exited cleanly", taskID)
	}

	log.Printf("[worker] task %s stdout=%dB stderr=%dB", taskID, len(stdoutBuf), len(stderrBuf))

	// Collect artifacts
	artifacts := []map[string]interface{}{}

	// stdout artifact
	if len(stdoutBuf) > 0 {
		artifacts = append(artifacts, map[string]interface{}{
			"type": "stdout",
			"data": stdoutBuf,
		})
	}

	// stderr artifact
	if len(stderrBuf) > 0 {
		artifacts = append(artifacts, map[string]interface{}{
			"type": "stderr",
			"data": stderrBuf,
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

	// Write stdout/stderr to workdir so Server can parse tool output.
	if len(stdoutBuf) > 0 {
		os.WriteFile(filepath.Join(workdir, "stdout.txt"), stdoutBuf, 0640)
	}
	if len(stderrBuf) > 0 {
		os.WriteFile(filepath.Join(workdir, "stderr.txt"), stderrBuf, 0640)
	}

	status := "completed"
	errorMsg := ""
	if wasIdleKilled {
		status = "failed"
		errorMsg = fmt.Sprintf("idle-timeout: no output for %v, process killed", idleThreshold)
	} else if err != nil && exitCode != 0 {
		status = "failed"
		if stderrData, readErr := os.ReadFile(filepath.Join(workdir, "stderr.txt")); readErr == nil && len(stderrData) > 0 {
			errorMsg = string(stderrData)
			if len(errorMsg) > 500 {
				errorMsg = errorMsg[:500] + "..."
			}
		} else if err != nil {
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

// injectCustomNucleiTemplates checks for a local custom bundle and appends
// -t (templates) and -w (workflows) flags to the nuclei command if the
// directories exist and are not already present in the command.
func (ws *WorkerServer) injectCustomNucleiTemplates(command []string, taskID string) []string {
	syncer := NewBundleSyncer(ws.dataDir, "", "")
	templatesDir := syncer.TemplatesDir()
	workflowsDir := syncer.WorkflowsDir()

	// Check if custom template paths are already in the command
	hasTemplatesFlag := false
	hasWorkflowsFlag := false
	for _, arg := range command {
		if arg == "-t" {
			hasTemplatesFlag = true
		}
		if arg == "-w" {
			hasWorkflowsFlag = true
		}
	}

	if !hasTemplatesFlag {
		if info, err := os.Stat(templatesDir); err == nil && info.IsDir() {
			command = append(command, "-t", templatesDir)
			log.Printf("[worker] task %s injected custom templates: %s", taskID, templatesDir)
		}
	}

	if !hasWorkflowsFlag {
		if info, err := os.Stat(workflowsDir); err == nil && info.IsDir() {
			command = append(command, "-w", workflowsDir)
			log.Printf("[worker] task %s injected custom workflows: %s", taskID, workflowsDir)
		}
	}

	return command
}

func (ws *WorkerServer) handleProgress(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (ws *WorkerServer) handleResult(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
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
	case "nmap":
		return 10 * time.Minute
	default:
		return 10 * time.Minute
	}
}
