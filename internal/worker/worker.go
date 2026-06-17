package worker

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
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
	"github.com/P0m32Kun/Anchor/internal/toolguard"
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
	dispatcher *Dispatcher
	governor   *ResourceGovernor
	allowlist  *toolguard.Allowlist
	procs      map[string]*exec.Cmd
	doneChs    map[string]chan struct{} // closed when process exits
	mu         sync.RWMutex
}

func NewRunner(q *db.Queries, scopeEng *scope.Engine, dataDir string) *Runner {
	return &Runner{
		queries:    q,
		scopeEng:   scopeEng,
		dataDir:    dataDir,
		dispatcher: NewDispatcher(q),
		governor:   NewResourceGovernor(LoadGovernorConfigFromEnv(), nil),
		allowlist:  toolguard.NewAllowlist(),
		procs:      make(map[string]*exec.Cmd),
		doneChs:    make(map[string]chan struct{}),
	}
}

// SetGovernor swaps the resource governor (used by tests and bootstrapping
// when a custom sampler or disabled governor is desired).
func (r *Runner) SetGovernor(g *ResourceGovernor) {
	r.governor = g
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

	// Resource governance: gate new task execution on memory/CPU thresholds.
	// Acquire may block (memory) or sleep (CPU); ctx cancellation aborts.
	if err := r.governor.Acquire(ctx); err != nil {
		return fmt.Errorf("resource governor blocked task %s: %w", taskID, err)
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

	// Handle shell commands: "sh -c <rest...>" needs the rest joined as a
	// single argument so sh interprets it as a command string.
	if args[0] == "sh" && len(args) > 2 && args[1] == "-c" {
		args = []string{"sh", "-c", strings.Join(args[2:], " ")}
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

	// Try to dispatch to a remote worker first. On dispatch failure (worker
	// unreachable), mark the worker offline and try the next online worker.
	var triedWorkerIDs []string
	for {
		worker := r.dispatcher.PickOnlineWorkerExcluding(triedWorkerIDs)
		if worker == nil {
			break
		}
		log.Printf("[runner] dispatching task %s to worker %s (%s)", task.ID, worker.ID, worker.Endpoint)
			err := r.dispatcher.DispatchToWorker(ctx, task, worker, workdir, project)
			r.dispatcher.ReleaseWorker(worker.ID)
			if err != nil {
				log.Printf("[runner] remote dispatch to %s failed: %v", worker.ID, err)
				if isAtCapacityError(err) {
					triedWorkerIDs = append(triedWorkerIDs, worker.ID)
					continue
				}
				if isUnreachableError(err) {
					if markErr := r.dispatcher.MarkWorkerOffline(worker.ID); markErr != nil {
						log.Printf("[runner] mark worker %s offline failed: %v", worker.ID, markErr)
					} else {
						log.Printf("[runner] worker %s marked offline (unreachable)", worker.ID)
					}
					triedWorkerIDs = append(triedWorkerIDs, worker.ID)
					continue
				}
				// Task-level failure (worker reachable but task failed/cancelled/timed out).
				// Do not silently fall back to local — propagate the failure.
				return err
			}
			return nil
	}

	// All online workers exhausted (none registered or all marked offline).
	// Fall back to local server-side execution. Log explicitly so the operator
	// understands this is a fallback path, not the normal one — in a worker
	// deployment the server image usually has no tools installed.
	binary := args[0]
	if len(triedWorkerIDs) > 0 {
		log.Printf("[runner] task %s: no remaining online workers (tried %d: %v); attempting local server fallback for tool %q", task.ID, len(triedWorkerIDs), triedWorkerIDs, binary)
	} else {
		log.Printf("[runner] task %s: no online worker registered; attempting local server fallback for tool %q", task.ID, binary)
	}
	if _, err := exec.LookPath(binary); err != nil {
		_ = r.queries.UpdateScanTaskStatus(task.ID, models.TaskFailed, nil, &now)
		if len(triedWorkerIDs) > 0 {
			return fmt.Errorf("task %s failed: tool %q unavailable — no online worker (tried %v) and server image has no local fallback", task.ID, binary, triedWorkerIDs)
		}
		return fmt.Errorf("task %s failed: tool %q unavailable — no online worker registered and server image has no local fallback", task.ID, binary)
	}
	log.Printf("[runner] task %s: executing locally on server (fallback path) — tool %q", task.ID, binary)

	// Inject custom nuclei templates if available (local execution fallback)
	if task.Tool == "nuclei" {
		scanDepth := ""
		if project.PipelineConfig != nil && *project.PipelineConfig != "" {
			var cfg models.PipelineConfig
			if err := json.Unmarshal([]byte(*project.PipelineConfig), &cfg); err == nil {
				scanDepth = cfg.NucleiScanDepth
			}
		}
		args = r.injectCustomNucleiTemplates(args, scanDepth)
	}

	// Append rate limit arguments if configured.
	cmdArgs := appendRateLimitArgs(args[1:], task.Tool, project.RateLimit)

	// Allowlist: reject unknown binaries or args with shell metacharacters.
	if err := r.allowlist.Validate(binary, cmdArgs); err != nil {
		_ = r.queries.UpdateScanTaskStatus(task.ID, models.TaskFailed, nil, &now)
		return fmt.Errorf("task %s failed: allowlist rejected command: %w", taskID, err)
	}

	cmd := exec.CommandContext(ctx, binary, cmdArgs...)
	cmd.Dir = workdir
	cmd.Env = os.Environ()

	stdoutW, err := newTaskOutputWriter(workdir, "stdout")
	if err != nil {
		_ = r.queries.UpdateScanTaskStatus(task.ID, models.TaskFailed, nil, &now)
		return fmt.Errorf("prepare stdout: %w", err)
	}
	defer stdoutW.Close()
	stderrW, err := newTaskOutputWriter(workdir, "stderr")
	if err != nil {
		_ = r.queries.UpdateScanTaskStatus(task.ID, models.TaskFailed, nil, &now)
		return fmt.Errorf("prepare stderr: %w", err)
	}
	defer stderrW.Close()
	cmd.Stdout = stdoutW
	cmd.Stderr = stderrW

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
	if stdoutW.Len() > 0 {
		if err := r.saveArtifact(task.ProjectID, task.ID, models.ArtifactStdout, workdir, stdoutW.Bytes()); err != nil {
			// Log but don't fail the whole task.
		}
	}
	if stderrW.Len() > 0 {
		if err := r.saveArtifact(task.ProjectID, task.ID, models.ArtifactStderr, workdir, stderrW.Bytes()); err != nil {
			// Log but don't fail.
		}
	}

	// Determine final status.
	status := models.TaskCompleted
	if exitCode != 0 {
		status = models.TaskFailed
	}
	if stdoutW.Truncated() || stderrW.Truncated() {
		status = models.TaskPartialSuccess
	}

	if err := r.queries.UpdateScanTaskStatus(task.ID, status, &exitCode, &finishedAt); err != nil {
		return fmt.Errorf("update task completed: %w", err)
	}

	if status == models.TaskFailed {
		stderr := string(stderrW.Bytes())
		if len(stderr) > 500 {
			stderr = stderr[:500] + "..."
		}
		_ = r.queries.UpdateScanTaskErrorMessage(task.ID, stderr)
		return fmt.Errorf("command exited with code %d", exitCode)
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

// injectCustomNucleiTemplates is a no-op. All custom templates now live
// under ~/nuclei-templates/ (nuclei's default search path), so nuclei finds
// them natively without -t or -w injection. The pipeline passes precise
// -w paths for workflow invocations.
func (r *Runner) injectCustomNucleiTemplates(command []string, scanDepth string) []string {
	return command
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

// isUnreachableError reports whether the dispatch error indicates the worker
// is unreachable (connection refused, DNS failure, network I/O timeout, or an
// explicit heartbeat-lost signal from the dispatcher) rather than a task-level
// failure.
//
// Note: the bare token "timeout" is intentionally NOT matched here. The
// dispatcher uses message text such as "exceeded N server-side poll deadline"
// for a server-side give-up where the worker may still be executing — that
// case must NOT mark the worker offline. Heartbeat-loss errors include the
// explicit "worker unreachable" signature and are matched precisely.
func isUnreachableError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	for _, signature := range []string{
		"post task to worker",
		"connection refused",
		"no such host",
		"i/o timeout",
		"network is unreachable",
		"dial tcp",
		"worker unreachable",
	} {
		if strings.Contains(msg, signature) {
			return true
		}
	}
	return false
}

func isAtCapacityError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "at capacity")
}
