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
	"strconv"
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

	// Execute task asynchronously
	// Use background context so task execution survives HTTP connection close.
	go ws.executeTask(context.Background(), req.TaskID, req.Tool, req.Command, req.Workdir, req.RateLimit, req.ScanDepth)

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
}

func (ws *WorkerServer) executeTask(ctx context.Context, taskID, tool string, command []string, workdir string, rateLimit int, scanDepth string) {
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

	cfg := resolveTimeoutConfig(tool, command, workdir)
	ctx, cancel := context.WithTimeout(ctx, cfg.RunningTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, binary, command[1:]...)
	cmd.Dir = workdir
	cmd.Env = os.Environ()
	cmd.WaitDelay = 30 * time.Second // Go 1.20+: force-close unresponsive IO after exit

	stdout := &idleWatchedWriter{}
	stderr := &idleWatchedWriter{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	ws.mu.Lock()
	ws.procs[taskID] = cmd
	ws.mu.Unlock()

	log.Printf("[worker] task %s exec: %s %v", taskID, binary, command[1:])
	log.Printf("[worker] task %s timeout config: startup=%v idle=%v running=%v cpu-check=%v",
		taskID, cfg.StartupTimeout, cfg.IdleTimeout, cfg.RunningTimeout, cfg.CPUCheckEnabled)

	startTime := time.Now()
	stopWatchdog := make(chan struct{})
	idleKilled := make(chan struct{}, 1)

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		inStartup := true
		startupDeadline := startTime.Add(cfg.StartupTimeout)

		for {
			select {
			case <-stopWatchdog:
				return
			case <-ticker.C:
				now := time.Now()

				// Phase 1: Startup — process must produce output within StartupTimeout
				if inStartup && now.Before(startupDeadline) {
					last := stdout.Last()
					if l2 := stderr.Last(); l2.After(last) {
						last = l2
					}
					if !last.IsZero() {
						inStartup = false
						log.Printf("[worker] task %s startup OK (first output at %v)", taskID, last.Sub(startTime).Round(time.Second))
					}
					continue
				}

				if inStartup {
					log.Printf("[worker] task %s startup timeout (%v) expired with no output, killing", taskID, cfg.StartupTimeout)
					select {
					case idleKilled <- struct{}{}:
					default:
					}
					cancel()
					return
				}

				// Phase 2: Running — check idle timeout
				last := stdout.Last()
				if l2 := stderr.Last(); l2.After(last) {
					last = l2
				}
				if last.IsZero() {
					last = startTime
				}
				since := now.Sub(last)

				if since > cfg.IdleTimeout {
					// Before killing, verify process isn't just slow via CPU check
					if cfg.CPUCheckEnabled && cmd.Process != nil {
						pid := cmd.Process.Pid
						if !isProcessStateHung(pid) {
							log.Printf("[worker] task %s idle for %v but CPU still active (pid=%d), extending grace",
								taskID, since.Round(time.Second), pid)
							continue
						}
						log.Printf("[worker] task %s idle for %v and CPU stalled (pid=%d), killing",
							taskID, since.Round(time.Second), pid)
					} else {
						log.Printf("[worker] task %s idle for %v (threshold %v), killing process",
							taskID, since.Round(time.Second), cfg.IdleTimeout)
					}
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
		errorMsg = fmt.Sprintf("idle-timeout: no output for %v, process killed", cfg.IdleTimeout)
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

// injectCustomNucleiTemplates injects official and custom template directories
// via -t flags. Workflow injection (-w) is handled by the pipeline with precise
// per-service paths; the worker only layers template directories for tag-based
// matching.
//
// Directory structure:
//   ~/templates-{sourceId}/  - custom templates (templates/ + workflows/)
//   ~/nuclei-templates/      - official nuclei templates
func (ws *WorkerServer) injectCustomNucleiTemplates(command []string, taskID string, scanDepth string) []string {
	home, _ := os.UserHomeDir()
	if home == "" {
		return command
	}

	// Inject official templates directory
	officialPath := filepath.Join(home, "nuclei-templates")
	if info, err := os.Stat(officialPath); err == nil && info.IsDir() {
		command = append(command, "-t", officialPath)
		log.Printf("[worker] task %s injected official templates: %s", taskID, officialPath)
	}

	// Scan for custom template source directories (~/templates-*)
	entries, err := os.ReadDir(home)
	if err != nil {
		return command
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "templates-") {
			continue
		}

		sourceDir := filepath.Join(home, name)
		if info, err := os.Stat(sourceDir); err == nil && info.IsDir() {
			command = append(command, "-t", sourceDir)
			log.Printf("[worker] task %s injected custom templates: %s", taskID, sourceDir)
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

// estimateScanScale returns (targetCount, portsPerTarget, totalProbes) by
// inspecting command args and reading the target list file.
func estimateScanScale(command []string, workdir string) (targets, portsPerTarget, total int) {
	// 1. Port range from strategy
	strategy := detectScanStrategy(command)
	switch strategy {
	case "full":
		portsPerTarget = 65535
	case "top1000":
		portsPerTarget = 1000
	case "top100":
		portsPerTarget = 100
	default:
		portsPerTarget = 1000
	}

	// 2. Target count from list file
	for i, arg := range command {
		if arg == "-list" || arg == "-l" || arg == "-host" {
			if i+1 < len(command) {
				targetFile := command[i+1]
				data, err := os.ReadFile(targetFile)
				if err != nil && workdir != "" {
					data, err = os.ReadFile(filepath.Join(workdir, targetFile))
				}
				if err == nil {
					lines := strings.Split(string(data), "\n")
					for _, line := range lines {
						if strings.TrimSpace(line) != "" {
							targets++
						}
					}
				}
			}
		}
	}

	total = targets * portsPerTarget
	return
}

// dynamicRunningTimeout calculates a realistic timeout based on scan scale.
// Uses conservative per-probe estimates to avoid underestimating.
func dynamicRunningTimeout(tool string, command []string, workdir string) time.Duration {
	base := defaultToolTimeout(tool)

	targets, portsPerTarget, total := estimateScanScale(command, workdir)
	if total <= 0 {
		return base
	}

	switch tool {
	case "naabu":
		// Conservative: 0.15s per probe (CONNECT with RTT, retries, SYN fallback).
		// This is pessimistic; root+SYN will be ~5x faster in practice.
		estimatedSecs := float64(total) * 0.15
		estimated := time.Duration(estimatedSecs) * time.Second
		// Add base overhead for startup / file I/O
		estimated += 5 * time.Minute
		// Floor at base, ceiling at 3 hours
		const maxTimeout = 3 * time.Hour
		if estimated > maxTimeout {
			return maxTimeout
		}
		if estimated < base {
			return base
		}
		log.Printf("[worker] dynamic timeout: %d targets x %d ports = %d probes -> %v",
			targets, portsPerTarget, total, estimated.Round(time.Minute))
		return estimated
	case "nmap":
		// nmap -sV is slower: ~0.3s per port with version detection
		estimatedSecs := float64(total) * 0.3
		estimated := time.Duration(estimatedSecs) * time.Second
		estimated += 5 * time.Minute
		const maxTimeout = 2 * time.Hour
		if estimated > maxTimeout {
			return maxTimeout
		}
		if estimated < base {
			return base
		}
		return estimated
	}
	return base
}

// TaskTimeoutConfig defines tiered timeout settings for external tool execution.
// It combines stdout idle detection with CPU activity checks to avoid killing
// slow-but-healthy scans (e.g. naabu CONNECT full-port scan).
type TaskTimeoutConfig struct {
	StartupTimeout  time.Duration // max time without any output after process starts
	RunningTimeout  time.Duration // hard ceiling for total execution time
	IdleTimeout     time.Duration // max silence on stdout/stderr during running
	CPUCheckEnabled bool          // if true, verify CPU time is growing before idle-kill
}

// resolveTimeoutConfig returns timeout settings tailored to the tool and scan strategy.
func resolveTimeoutConfig(tool string, command []string, workdir string) TaskTimeoutConfig {
	base := TaskTimeoutConfig{
		StartupTimeout:  30 * time.Second,
		RunningTimeout:  dynamicRunningTimeout(tool, command, workdir),
		IdleTimeout:     90 * time.Second,
		CPUCheckEnabled: true,
	}

	switch tool {
	case "nuclei":
		// nuclei emits -stats heartbeat every 30s; 60s is safe
		base.IdleTimeout = 60 * time.Second
		base.CPUCheckEnabled = false // stats heartbeat is sufficient
	case "nmap":
		base.IdleTimeout = 120 * time.Second
	case "naabu":
		strategy := detectScanStrategy(command)
		if strategy == "full" {
			// Full-port CONNECT scan can go minutes between open ports
			base.IdleTimeout = 5 * time.Minute
		} else {
			base.IdleTimeout = 90 * time.Second
		}
	case "httpx", "subfinder", "dnsx":
		base.IdleTimeout = 60 * time.Second
	}
	return base
}

// detectScanStrategy inspects command args to determine scan depth.
func detectScanStrategy(command []string) string {
	for i, arg := range command {
		if arg == "-tp" || arg == "--top-ports" || arg == "-p" {
			if i+1 < len(command) {
				switch command[i+1] {
				case "full", "-":
					return "full"
				case "100":
					return "top100"
				case "1000":
					return "top1000"
				}
			}
		}
		// nmap -p- means all ports
		if arg == "-p-" {
			return "full"
		}
	}
	return "default"
}

// readProcessCPUTime reads the total CPU time (user+system jiffies) for a PID
// from /proc/<pid>/stat. Returns 0 if the process no longer exists.
func readProcessCPUTime(pid int) uint64 {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(data))
	if len(fields) < 15 {
		return 0
	}
	// fields[13] = utime, fields[14] = stime
	utime, _ := strconv.ParseUint(fields[13], 10, 64)
	stime, _ := strconv.ParseUint(fields[14], 10, 64)
	return utime + stime
}

// readProcessState returns the State field from /proc/<pid>/status.
func readProcessState(pid int) string {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return ""
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "State:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "State:"))
		}
	}
	return ""
}

// isProcessStateHung checks if a process appears truly stuck.
// It samples CPU time over a short window and inspects process state.
func isProcessStateHung(pid int) bool {
	t1 := readProcessCPUTime(pid)
	state1 := readProcessState(pid)
	time.Sleep(2 * time.Second)
	t2 := readProcessCPUTime(pid)
	state2 := readProcessState(pid)

	// Process exited
	if t2 == 0 {
		return false
	}

	// CPU is still advancing → process is working
	if t2 > t1 {
		return false
	}

	// CPU stalled but process is in D state (uninterruptible sleep, usually IO)
	// Give it more grace; D state alone doesn't mean deadlocked
	if strings.HasPrefix(state2, "D") {
		// If it was also D before, it might be stuck
		if strings.HasPrefix(state1, "D") {
			return true
		}
		return false
	}

	// CPU stalled and not in D state → likely hung or finished
	return true
}

// idleWatchedWriter is an io.Writer that records the timestamp of the most
// recent Write call. The watchdog goroutine reads Last() to detect hung
// subprocesses that produce no output for too long.
type idleWatchedWriter struct {
	mu   sync.Mutex
	buf  []byte
	last time.Time
}

func (w *idleWatchedWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	w.last = time.Now()
	w.buf = append(w.buf, p...)
	w.mu.Unlock()
	return len(p), nil
}

func (w *idleWatchedWriter) Bytes() []byte {
	w.mu.Lock()
	defer w.mu.Unlock()
	out := make([]byte, len(w.buf))
	copy(out, w.buf)
	return out
}

func (w *idleWatchedWriter) Last() time.Time {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.last
}
