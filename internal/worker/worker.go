package worker

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"secbench/internal/db"
	"secbench/internal/models"
	"secbench/internal/scope"
	"secbench/internal/util"
)

const (
	maxOutputSize = 100 * 1024 * 1024 // 100 MB
	sigkillDelay  = 5 * time.Second
)

// Runner executes CLI tools in isolated workdirs.
type Runner struct {
	queries  *db.Queries
	scopeEng *scope.Engine
	dataDir  string
	procs    map[string]*exec.Cmd
	doneChs  map[string]chan struct{} // closed when process exits
	mu       sync.RWMutex
}

func NewRunner(q *db.Queries, scopeEng *scope.Engine, dataDir string) *Runner {
	return &Runner{
		queries:  q,
		scopeEng: scopeEng,
		dataDir:  dataDir,
		procs:    make(map[string]*exec.Cmd),
		doneChs:  make(map[string]chan struct{}),
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

	binary := args[0]
	if _, err := exec.LookPath(binary); err != nil {
		_ = r.queries.UpdateScanTaskStatus(task.ID, models.TaskFailed, nil, &now)
		return fmt.Errorf("tool not found: %s", binary)
	}

	cmd := exec.CommandContext(ctx, binary, args[1:]...)
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
func BuildSubfinderCommand(domain string) []string {
	return []string{"subfinder", "-d", domain, "-oJ", "-o", "subfinder_output.jsonl"}
}
