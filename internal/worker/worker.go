package worker

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/scope"
	"github.com/P0m32Kun/Anchor/internal/util"
)

const (
	maxOutputSize = 100 * 1024 * 1024 // 100 MB
	sigkillDelay  = 5 * time.Second
)

// Runner executes CLI tools in isolated workdirs.
type Runner struct {
	queries    *db.Queries
	scopeEng   *scope.Engine
	dataDir    string
	httpClient *http.Client
	procs      map[string]*exec.Cmd
	doneChs    map[string]chan struct{} // closed when process exits
	mu         sync.RWMutex
}

func NewRunner(q *db.Queries, scopeEng *scope.Engine, dataDir string) *Runner {
	return &Runner{
		queries:    q,
		scopeEng:   scopeEng,
		dataDir:    dataDir,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		procs:      make(map[string]*exec.Cmd),
		doneChs:    make(map[string]chan struct{}),
	}
}

// Run executes a tool for the given task.
func (r *Runner) Run(ctx context.Context, taskID string) error {
	task, err := r.queries.GetScanTask(taskID)
	if err != nil {
		return fmt.Errorf("get task: %w", err)
	}
	if task == nil {
		return fmt.Errorf("task not found: %s", taskID)
	}

	now := time.Now().UTC()

	// --- TOCTOU Scope Check ---
	if task.TargetID != nil && *task.TargetID != "" {
		target, err := r.queries.GetTarget(*task.TargetID)
		if err != nil {
			_ = r.queries.UpdateScanTaskStatus(task.ID, models.TaskScopeDenied, nil, &now)
			return fmt.Errorf("get target for scope check: %w", err)
		}
		if target == nil {
			_ = r.queries.UpdateScanTaskStatus(task.ID, models.TaskScopeDenied, nil, &now)
			return fmt.Errorf("target not found: %s", *task.TargetID)
		}

		decision, err := r.scopeEng.ValidateBeforeRun(ctx, task.ProjectID, target, task.ID)
		if err != nil {
			_ = r.queries.UpdateScanTaskStatus(task.ID, models.TaskScopeDenied, nil, &now)
			return fmt.Errorf("scope check failed: %w", err)
		}
		if decision.Decision == models.ScopeDeny {
			_ = r.queries.UpdateScanTaskStatus(task.ID, models.TaskScopeDenied, nil, &now)
			return fmt.Errorf("scope denied: %s", decision.Reason)
		}
	}

	// Update status to running and set started_at.
	if err := r.queries.SetScanTaskRunning(task.ID, now); err != nil {
		return fmt.Errorf("update task running: %w", err)
	}

	// Create workdir.
	workdir := filepath.Join(r.dataDir, "workdirs", task.ProjectID, task.ID)
	if err := os.MkdirAll(workdir, 0750); err != nil {
		_ = r.queries.UpdateScanTaskStatus(task.ID, models.TaskFailed, nil, &now)
		return fmt.Errorf("create workdir: %w", err)
	}

	// Build command.
	args := strings.Fields(task.CommandTemplate)
	if len(args) == 0 {
		_ = r.queries.UpdateScanTaskStatus(task.ID, models.TaskFailed, nil, &now)
		return fmt.Errorf("empty command template")
	}

	// Fetch project to check rate limit.
	project, err := r.queries.GetProject(task.ProjectID)
	if err != nil {
		_ = r.queries.UpdateScanTaskStatus(task.ID, models.TaskFailed, nil, &now)
		return fmt.Errorf("get project for rate limit: %w", err)
	}
	if project == nil {
		_ = r.queries.UpdateScanTaskStatus(task.ID, models.TaskFailed, nil, &now)
		return fmt.Errorf("project not found: %s", task.ProjectID)
	}

	// Try to dispatch to a remote worker first.
	if worker := r.pickOnlineWorker(); worker != nil {
		log.Printf("[runner] dispatching task %s to worker %s (%s)", task.ID, worker.ID, worker.Endpoint)
		if err := r.dispatchToWorker(ctx, task, worker, workdir, project); err != nil {
			log.Printf("[runner] remote dispatch failed: %v, falling back to local", err)
		} else {
			return nil
		}
	}

	binary := args[0]
	if _, err := exec.LookPath(binary); err != nil {
		_ = r.queries.UpdateScanTaskStatus(task.ID, models.TaskFailed, nil, &now)
		return fmt.Errorf("tool not found: %s", binary)
	}

	// Append rate limit arguments if configured.
	cmdArgs := appendRateLimitArgs(args[1:], task.Tool, project.RateLimit)

	cmd := exec.CommandContext(ctx, binary, cmdArgs...)
	cmd.Dir = workdir
	cmd.Env = os.Environ()

	// Capture stdout/stderr with size limit.
	var stdoutBuf, stderrBuf limitedBuffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	// Record and persist invocation.
	inv := &models.ToolInvocation{
		ID:              util.GenerateID(),
		ProjectID:       task.ProjectID,
		TaskID:          task.ID,
		Tool:            task.Tool,
		BinaryPath:      binary,
		CommandRedacted: task.ArgumentsRedacted,
		Workdir:         workdir,
		StartedAt:       time.Now().UTC(),
	}
	if err := r.queries.CreateToolInvocation(inv); err != nil {
		// Log but don't fail the task; ToolInvocation is non-critical for execution.
		_ = err
	}

	// Start process.
	if err := cmd.Start(); err != nil {
		_ = r.queries.UpdateScanTaskStatus(task.ID, models.TaskFailed, nil, &now)
		return fmt.Errorf("start process: %w", err)
	}

	// Track PID for cancellation.
	doneCh := make(chan struct{})
	r.mu.Lock()
	r.procs[task.ID] = cmd
	r.doneChs[task.ID] = doneCh
	r.mu.Unlock()

	// Wait for completion.
	err = cmd.Wait()

	// Remove from tracking.
	r.mu.Lock()
	delete(r.procs, task.ID)
	delete(r.doneChs, task.ID)
	r.mu.Unlock()
	close(doneCh)

	finishedAt := time.Now().UTC()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}

	// Update ToolInvocation with completion info.
	_ = r.queries.UpdateToolInvocation(task.ID, finishedAt, exitCode)

	// Save artifacts.
	if stdoutBuf.Len() > 0 {
		if err := r.saveArtifact(task.ProjectID, task.ID, models.ArtifactStdout, workdir, stdoutBuf.Bytes()); err != nil {
			// Log but don't fail the whole task.
		}
	}
	if stderrBuf.Len() > 0 {
		if err := r.saveArtifact(task.ProjectID, task.ID, models.ArtifactStderr, workdir, stderrBuf.Bytes()); err != nil {
			// Log but don't fail.
		}
	}

	// Determine final status.
	status := models.TaskCompleted
	if exitCode != 0 {
		status = models.TaskFailed
	}
	if stdoutBuf.truncated || stderrBuf.truncated {
		status = models.TaskPartialSuccess
	}

	if err := r.queries.UpdateScanTaskStatus(task.ID, status, &exitCode, &finishedAt); err != nil {
		return fmt.Errorf("update task completed: %w", err)
	}

	return nil
}

func (r *Runner) saveArtifact(projectID, taskID string, artifactType models.ArtifactType, workdir string, data []byte) error {
	filename := fmt.Sprintf("%s_%d.txt", artifactType, time.Now().UnixNano())
	path := filepath.Join(workdir, filename)
	if err := os.WriteFile(path, data, 0640); err != nil {
		return fmt.Errorf("write artifact: %w", err)
	}

	sum := sha256.Sum256(data)
	a := &models.RawArtifact{
		ID:              util.GenerateID(),
		ProjectID:       projectID,
		TaskID:          &taskID,
		Type:            artifactType,
		Path:            path,
		SHA256:          fmt.Sprintf("%x", sum),
		Size:            int64(len(data)),
		RedactionStatus: "unchecked",
		CreatedAt:       time.Now().UTC(),
	}
	return r.queries.CreateRawArtifact(a)
}

// Cancel sends SIGTERM then SIGKILL after delay to a running process.
func (r *Runner) Cancel(taskID string) error {
	r.mu.RLock()
	cmd, ok := r.procs[taskID]
	doneCh := r.doneChs[taskID]
	r.mu.RUnlock()

	if !ok || cmd == nil || cmd.Process == nil {
		// Process not running or already finished; update DB status only.
		now := time.Now().UTC()
		_ = r.queries.UpdateScanTaskStatus(taskID, models.TaskCancelled, nil, &now)
		return nil
	}

	// Send SIGTERM.
	_ = cmd.Process.Signal(syscall.SIGTERM)

	// Wait for graceful shutdown.
	select {
	case <-doneCh:
		// Process exited gracefully.
	case <-time.After(sigkillDelay):
		// Force kill.
		_ = cmd.Process.Kill()
		<-doneCh
	}

	// Update task status.
	now := time.Now().UTC()
	_ = r.queries.UpdateScanTaskStatus(taskID, models.TaskCancelled, nil, &now)
	return nil
}

// limitedBuffer wraps bytes.Buffer with a max size limit.
type limitedBuffer struct {
	buf       bytes.Buffer
	truncated bool
}

func (lb *limitedBuffer) Write(p []byte) (n int, err error) {
	if lb.buf.Len()+len(p) > maxOutputSize {
		lb.truncated = true
		remaining := maxOutputSize - lb.buf.Len()
		if remaining > 0 {
			lb.buf.Write(p[:remaining])
		}
		return len(p), nil
	}
	return lb.buf.Write(p)
}

func (lb *limitedBuffer) Len() int      { return lb.buf.Len() }
func (lb *limitedBuffer) Bytes() []byte { return lb.buf.Bytes() }

// BuildSubfinderCommand builds a Subfinder command for the given domain.
// Output goes to stdout as JSONL so the worker can capture it as an artifact.
func BuildSubfinderCommand(domain string) []string {
	return []string{"subfinder", "-d", domain, "-oJ"}
}

// BuildHttpxCommand builds an httpx command that reads hosts from a file.
// hostFile should contain one host per line.
// Output goes to stdout as JSONL so the worker can capture it as an artifact.
func BuildHttpxCommand(hostFile string) []string {
	return []string{"httpx", "-json", "-l", hostFile, "-follow-redirects"}
}

// BuildNaabuCommand builds a Naabu command that reads hosts from a file.
// hostFile should contain one host per line.
// Output goes to stdout as JSONL so the worker can capture it as an artifact.
func BuildNaabuCommand(hostFile string) []string {
	return []string{"naabu", "-json", "-list", hostFile}
}

// BuildNucleiCommand builds a Nuclei command.
// If tags is non-empty, adds -tags flag. Otherwise runs without tag filter.
func BuildNucleiCommand(targetFile, profile string, rateLimit int, tags []string) []string {
	args := []string{"nuclei", "-jsonl", "-l", targetFile}

	switch profile {
	case "light":
		args = append(args, "-severity", "critical,high", "-timeout", "3")
	case "standard", "":
		args = append(args, "-severity", "critical,high,medium", "-timeout", "5")
	case "deep":
		args = append(args, "-severity", "critical,high,medium,low,info", "-timeout", "10")
	}

	if len(tags) > 0 {
		args = append(args, "-tags", strings.Join(tags, ","))
	}

	if rateLimit > 0 {
		args = append(args, "-rl", fmt.Sprintf("%d", rateLimit))
	}

	return args
}

// appendRateLimitArgs appends tool-specific rate limit flags to the argument list.
// Only adds flags when rate > 0 and the tool supports it.
func appendRateLimitArgs(args []string, tool string, rate int) []string {
	if rate <= 0 {
		return args
	}
	switch strings.ToLower(tool) {
	case "naabu":
		return append(args, "-rate", fmt.Sprintf("%d", rate))
	case "nuclei":
		return append(args, "-rl", fmt.Sprintf("%d", rate))
	case "httpx":
		return append(args, "-rate-limit", fmt.Sprintf("%d", rate))
	default:
		// Subfinder and others don't support rate limiting; skip.
		return args
	}
}

// pickOnlineWorker returns the most recently seen online remote worker, or nil if none.
func (r *Runner) pickOnlineWorker() *models.WorkerNode {
	workers, err := r.queries.ListWorkerNodes()
	if err != nil {
		return nil
	}
	var latest *models.WorkerNode
	for _, w := range workers {
		if w.Status == models.WorkerStatusOnline && w.Endpoint != "" && w.RevokedAt == nil {
			if latest == nil || (w.LastSeen != nil && latest.LastSeen != nil && w.LastSeen.After(*latest.LastSeen)) {
				latest = w
			}
		}
	}
	return latest
}

// dispatchToWorker sends a task to a remote worker and polls until completion.
func (r *Runner) dispatchToWorker(ctx context.Context, task *models.ScanTask, worker *models.WorkerNode, workdir string, project *models.Project) error {
	args := strings.Fields(task.CommandTemplate)
	reqBody, _ := json.Marshal(map[string]interface{}{
		"task_id":    task.ID,
		"tool":       task.Tool,
		"command":    args,
		"workdir":    workdir,
		"rate_limit": project.RateLimit,
	})

	resp, err := r.httpClient.Post(worker.Endpoint+"/tasks", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("post task to worker: %w", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("worker rejected task: %s", resp.Status)
	}

	// Poll for task completion (up to 10 minutes).
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	timeout := time.After(10 * time.Minute)

	for {
		select {
		case <-ticker.C:
			t, err := r.queries.GetScanTask(task.ID)
			if err != nil || t == nil {
				continue
			}
			if t.Status == models.TaskCompleted {
				return nil
			}
			if t.Status == models.TaskFailed || t.Status == models.TaskScopeDenied || t.Status == models.TaskCancelled {
				return fmt.Errorf("task %s finished with status: %s", task.ID, t.Status)
			}
		case <-timeout:
			return fmt.Errorf("task %s timeout waiting for worker", task.ID)
		case <-ctx.Done():
			return fmt.Errorf("task %s cancelled", task.ID)
		}
	}
}
