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
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/models"
)

// Dispatcher handles dispatching tasks to remote workers.
type Dispatcher struct {
	queries    *db.Queries
	httpClient *http.Client

	inflightMu sync.Mutex
	inflight   map[string]int // tasks dispatched but not yet finished (server-side)
	pickSeq    uint64         // round-robin tie-break among equal load
}

// NewDispatcher creates a new Dispatcher.
func NewDispatcher(queries *db.Queries) *Dispatcher {
	return &Dispatcher{
		queries:    queries,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		inflight:   make(map[string]int),
	}
}

// PickOnlineWorker returns the least-loaded eligible online worker.
func (d *Dispatcher) PickOnlineWorker() *models.WorkerNode {
	return d.PickOnlineWorkerExcluding(nil)
}

// PickOnlineWorkerExcluding selects the least-loaded eligible worker not in
// excludeIDs, increments its in-flight counter, and returns it. Call
// ReleaseWorker when dispatch finishes (success or failure).
func (d *Dispatcher) PickOnlineWorkerExcluding(excludeIDs []string) *models.WorkerNode {
	w := d.pickLeastLoaded(excludeIDs)
	if w == nil {
		return nil
	}
	d.trackInflight(w.ID, +1)
	return w
}

// ReleaseWorker decrements the in-flight counter for a worker after dispatch
// completes or is abandoned.
func (d *Dispatcher) ReleaseWorker(workerID string) {
	d.trackInflight(workerID, -1)
}

func (d *Dispatcher) trackInflight(workerID string, delta int) {
	if workerID == "" {
		return
	}
	d.inflightMu.Lock()
	defer d.inflightMu.Unlock()
	d.inflight[workerID] += delta
	if d.inflight[workerID] <= 0 {
		delete(d.inflight, workerID)
	}
}

func (d *Dispatcher) currentInflight(workerID string) int {
	d.inflightMu.Lock()
	defer d.inflightMu.Unlock()
	return d.inflight[workerID]
}

func (d *Dispatcher) isEligible(w *models.WorkerNode, exclude map[string]struct{}) bool {
	if w == nil || w.Endpoint == "" || w.RevokedAt != nil {
		return false
	}
	if _, skip := exclude[w.ID]; skip {
		return false
	}
	switch w.Status {
	case models.WorkerStatusOnline, models.WorkerStatusBusy:
		return true
	default:
		return false
	}
}

func workerAtCapacity(w *models.WorkerNode, load int) bool {
	if w.MaxConcurrency <= 0 {
		return false
	}
	return load >= w.MaxConcurrency
}

// pickLeastLoaded chooses the online worker with the fewest running + in-flight
// tasks. Ties rotate round-robin so new workers with zero load receive work
// immediately alongside existing idle peers.
func (d *Dispatcher) pickLeastLoaded(excludeIDs []string) *models.WorkerNode {
	workers, err := d.queries.ListWorkerNodes()
	if err != nil || len(workers) == 0 {
		return nil
	}
	exclude := make(map[string]struct{}, len(excludeIDs))
	for _, id := range excludeIDs {
		exclude[id] = struct{}{}
	}

	running, err := d.queries.CountRunningScanTasksPerWorker()
	if err != nil {
		running = map[string]int{}
	}

	var candidates []*models.WorkerNode
	minLoad := -1
	for _, w := range workers {
		if !d.isEligible(w, exclude) {
			continue
		}
		load := running[w.ID] + d.currentInflight(w.ID)
		if workerAtCapacity(w, load) {
			continue
		}
		if minLoad < 0 || load < minLoad {
			minLoad = load
			candidates = []*models.WorkerNode{w}
		} else if load == minLoad {
			candidates = append(candidates, w)
		}
	}
	if len(candidates) == 0 {
		return nil
	}
	if len(candidates) == 1 {
		return candidates[0]
	}

	d.inflightMu.Lock()
	idx := d.pickSeq % uint64(len(candidates))
	d.pickSeq++
	d.inflightMu.Unlock()
	return candidates[idx]
}

// MarkWorkerOffline marks a worker as offline (used when dispatch fails to a worker
// that registered as online but has become unreachable).
func (d *Dispatcher) MarkWorkerOffline(workerID string) error {
	return d.queries.UpdateWorkerNodeStatus(workerID, models.WorkerStatusOffline, time.Now().UTC())
}

// DispatchToWorker sends a task to a remote worker and polls until completion.
// Retries up to maxDispatchAttempts on unreachable-worker failures (network
// errors, worker going offline mid-task) so the runner can try another worker.
func (d *Dispatcher) DispatchToWorker(ctx context.Context, task *models.ScanTask, worker *models.WorkerNode, workdir string, project *models.Project) error {
	const maxDispatchAttempts = 3
	for attempt := 1; attempt <= maxDispatchAttempts; attempt++ {
		err := d.dispatchOnce(ctx, task, worker, workdir, project)
		if err == nil {
			return nil
		}
		if attempt >= maxDispatchAttempts {
			return err
		}
		if !isUnreachableError(err) {
			return err
		}
		log.Printf("[dispatcher] task %s worker unreachable on attempt %d, retrying (max %d)", task.ID, attempt, maxDispatchAttempts)
		if resetErr := d.queries.ResetScanTaskForRetry(task.ID); resetErr != nil {
			log.Printf("[dispatcher] failed to reset task %s for retry: %v", task.ID, resetErr)
			return err
		}
	}
	return fmt.Errorf("task %s failed after %d attempts", task.ID, maxDispatchAttempts)
}

// dispatchOnce sends a single dispatch attempt to a worker and polls for completion.
func (d *Dispatcher) dispatchOnce(ctx context.Context, task *models.ScanTask, worker *models.WorkerNode, workdir string, project *models.Project) error {
	args := strings.Fields(task.CommandTemplate)
	inputFiles := collectInputFiles(args)

	// Extract scanDepth from project config for nuclei workflow injection control.
	scanDepth := ""
	if task.Tool == "nuclei" && project.PipelineConfig != nil && *project.PipelineConfig != "" {
		var cfg models.PipelineConfig
		if err := json.Unmarshal([]byte(*project.PipelineConfig), &cfg); err == nil {
			scanDepth = cfg.NucleiScanDepth
		}
	}

	reqBody, _ := json.Marshal(map[string]interface{}{
		"task_id":     task.ID,
		"tool":        task.Tool,
		"command":     args,
		"workdir":     workdir,
		"rate_limit":  project.RateLimit,
		"input_files": inputFiles,
		"scan_depth":  scanDepth,
	})

	resp, err := d.httpClient.Post(worker.Endpoint+"/tasks", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("post task to worker: %w", err)
	}
	resp.Body.Close()
	if resp.StatusCode == http.StatusServiceUnavailable {
		return fmt.Errorf("worker at capacity: %s", resp.Status)
	}
	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("worker rejected task: %s", resp.Status)
	}
	if err := d.queries.SetScanTaskWorker(task.ID, worker.ID); err != nil {
		log.Printf("[dispatcher] set worker_id for task %s: %v", task.ID, err)
	}

	// Poll for task completion. The server trusts the worker's heartbeat
	// mechanism: as long as the worker keeps reporting health (the API-server
	// watchdog marks it offline after ~120s of missed heartbeats), we keep
	// polling. No server-side timeout — we trust the tool's own -timeout
	// parameters to terminate the process when appropriate.
	const (
		pollInterval        = 2 * time.Second
		workerCheckInterval = 30 * time.Second
	)
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	workerCheck := time.NewTicker(workerCheckInterval)
	defer workerCheck.Stop()
	startTime := time.Now()
	log.Printf("[dispatcher] polling task %s on worker %s (heartbeat-aware, no server-side timeout)", task.ID, worker.ID)

	for {
		select {
		case <-ticker.C:
			t, err := d.queries.GetScanTask(task.ID)
			if err != nil || t == nil {
				continue
			}
			if t.Status == models.TaskCompleted {
				return nil
			}
			if t.Status == models.TaskFailed || t.Status == models.TaskScopeDenied {
				return fmt.Errorf("task %s finished with status: %s", task.ID, t.Status)
			}
			if t.Status == models.TaskCancelled {
				d.cancelRemoteTask(worker.Endpoint, task.ID)
				return fmt.Errorf("task %s cancelled", task.ID)
			}
		case <-workerCheck.C:
			// Trust the heartbeat watchdog: if the API server's background
			// goroutine has already marked this worker offline (heartbeat lost
			// > 120s), the task is effectively orphaned — give up and let the
			// runner retry on another worker.
			w, err := d.queries.GetWorkerNode(worker.ID)
			if err != nil || w == nil {
				continue
			}
			if w.Status == models.WorkerStatusOffline {
				// Include "worker unreachable" so isUnreachableError matches.
				return fmt.Errorf("task %s: worker %s heartbeat lost (worker unreachable) after %v", task.ID, worker.ID, time.Since(startTime).Round(time.Second))
			}
		case <-ctx.Done():
			d.cancelRemoteTask(worker.Endpoint, task.ID)
			return fmt.Errorf("task %s cancelled", task.ID)
		}
	}
}

// cancelRemoteTask sends a cancel request to a remote worker for the given task.
// Best-effort: failures are logged but not returned, since the server-side
// status update is the authoritative cancellation signal.
func (d *Dispatcher) cancelRemoteTask(workerEndpoint, taskID string) {
	req, err := http.NewRequest("POST", workerEndpoint+"/tasks/"+taskID+"/cancel", nil)
	if err != nil {
		log.Printf("[dispatcher] build cancel request for task %s: %v", taskID, err)
		return
	}
	resp, err := d.httpClient.Do(req)
	if err != nil {
		log.Printf("[dispatcher] cancel task %s on worker %s: %v", taskID, workerEndpoint, err)
		return
	}
	resp.Body.Close()
	log.Printf("[dispatcher] cancelled task %s on worker %s (status %d)", taskID, workerEndpoint, resp.StatusCode)
}

// collectInputFiles inspects command arguments for absolute file paths that
// exist on the server's filesystem. Each existing file is read and returned in
// a map keyed by absolute path with base64-encoded contents. The worker
// recreates these files at the same absolute path before executing the tool,
// so commands referencing input lists (e.g. naabu -list /data/.../hosts.txt)
// work transparently across the dispatch boundary.
func collectInputFiles(args []string) map[string]string {
	files := make(map[string]string)
	for _, a := range args {
		if !filepath.IsAbs(a) {
			continue
		}
		info, err := os.Stat(a)
		if err != nil || info.IsDir() {
			continue
		}
		// Cap individual file size at 32 MB to avoid bloating task payloads.
		if info.Size() > 32*1024*1024 {
			continue
		}
		data, err := os.ReadFile(a)
		if err != nil {
			continue
		}
		files[a] = base64.StdEncoding.EncodeToString(data)
	}
	return files
}
