package worker

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/models"
)

// Dispatcher handles dispatching tasks to remote workers.
type Dispatcher struct {
	queries    *db.Queries
	httpClient *http.Client
}

// NewDispatcher creates a new Dispatcher.
func NewDispatcher(queries *db.Queries) *Dispatcher {
	return &Dispatcher{
		queries:    queries,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// PickOnlineWorker returns the most recently seen online remote worker, or nil if none.
func (d *Dispatcher) PickOnlineWorker() *models.WorkerNode {
	return d.PickOnlineWorkerExcluding(nil)
}

// PickOnlineWorkerExcluding returns the most recently seen online remote worker
// whose ID is not in excludeIDs. Returns nil if no eligible worker exists.
func (d *Dispatcher) PickOnlineWorkerExcluding(excludeIDs []string) *models.WorkerNode {
	workers, err := d.queries.ListWorkerNodes()
	if err != nil {
		return nil
	}
	exclude := make(map[string]struct{}, len(excludeIDs))
	for _, id := range excludeIDs {
		exclude[id] = struct{}{}
	}
	var latest *models.WorkerNode
	for _, w := range workers {
		if _, skip := exclude[w.ID]; skip {
			continue
		}
		if w.Status == models.WorkerStatusOnline && w.Endpoint != "" && w.RevokedAt == nil {
			if latest == nil || (w.LastSeen != nil && latest.LastSeen != nil && w.LastSeen.After(*latest.LastSeen)) {
				latest = w
			}
		}
	}
	return latest
}

// MarkWorkerOffline marks a worker as offline (used when dispatch fails to a worker
// that registered as online but has become unreachable).
func (d *Dispatcher) MarkWorkerOffline(workerID string) error {
	return d.queries.UpdateWorkerNodeStatus(workerID, models.WorkerStatusOffline, time.Now().UTC())
}

// DispatchToWorker sends a task to a remote worker and polls until completion.
func (d *Dispatcher) DispatchToWorker(ctx context.Context, task *models.ScanTask, worker *models.WorkerNode, workdir string, project *models.Project) error {
	args := strings.Fields(task.CommandTemplate)
	inputFiles := collectInputFiles(args)
	reqBody, _ := json.Marshal(map[string]interface{}{
		"task_id":     task.ID,
		"tool":        task.Tool,
		"command":     args,
		"workdir":     workdir,
		"rate_limit":  project.RateLimit,
		"input_files": inputFiles,
	})

	resp, err := d.httpClient.Post(worker.Endpoint+"/tasks", "application/json", bytes.NewReader(reqBody))
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
			t, err := d.queries.GetScanTask(task.ID)
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
