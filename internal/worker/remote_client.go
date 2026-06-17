package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"syscall"
	"time"

	"github.com/P0m32Kun/Anchor/internal/builtin"
)

// RemoteClient connects a worker node to a core server.
type RemoteClient struct {
	coreURL    string
	workerID   string
	token      string
	endpoint   string
	dataDir    string
	runner     *Runner
	httpClient *http.Client
	stopCh     chan struct{}
}

// NewRemoteClient creates a client that registers with the core server.
// apiToken is the global server token required for all API calls.
// dataDir is used for local nuclei template paths.
// runner is the local task runner for executing tools.
func NewRemoteClient(coreURL, endpoint, apiToken, dataDir string, runner *Runner) *RemoteClient {
	return &RemoteClient{
		coreURL:    coreURL,
		endpoint:   endpoint,
		token:      apiToken,
		dataDir:    dataDir,
		runner:     runner,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		stopCh:     make(chan struct{}),
	}
}

// Register the worker with the core server.
func (c *RemoteClient) Register(name string) error {
	body, _ := json.Marshal(map[string]interface{}{
		"name":            name,
		"endpoint":        c.endpoint,
		"max_concurrency": LoadMaxConcurrencyFromEnv(),
	})
	req, _ := http.NewRequest("POST", c.coreURL+"/workers/register", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("register request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("register failed: unauthorized (invalid API token)")
	}
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("register failed: %s", resp.Status)
	}

	var result struct {
		WorkerID string `json:"worker_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode register response: %w", err)
	}

	c.workerID = result.WorkerID
	log.Printf("[worker] registered as %s", c.workerID)
	// Immediately heartbeat so server marks us online without waiting for ticker.
	c.heartbeat()
	return nil
}

// StartHeartbeat sends periodic heartbeats to the core server.
func (c *RemoteClient) StartHeartbeat(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				c.heartbeat()
			case <-c.stopCh:
				return
			}
		}
	}()
}

// StartSourceSync periodically syncs enabled nuclei template sources from the server.
func (c *RemoteClient) StartSourceSync(interval time.Duration) {
	// Sync immediately on start
	go func() {
		c.syncSources()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				c.syncSources()
			case <-c.stopCh:
				return
			}
		}
	}()
}

// syncSources fetches nuclei template sources and applies RBKD builtin symlink.
func (c *RemoteClient) syncSources() {
	// Fetch all enabled sources from the server
	req, _ := http.NewRequest("GET", c.coreURL+"/nuclei/custom/sources", nil)
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("[worker] fetch sources error: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[worker] fetch sources status: %s", resp.Status)
		return
	}

	var sources []struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		InstallPath string `json:"install_path"`
		Enabled     bool   `json:"enabled"`
		Builtin     bool   `json:"builtin"`
		Status      string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&sources); err != nil {
		log.Printf("[worker] decode sources: %v", err)
		return
	}

	for _, src := range sources {
		if !src.Builtin {
			continue
		}
		if err := builtin.ApplyRBKDNucleiSymlink(src.Enabled); err != nil {
			log.Printf("[worker] rbkd symlink: %v", err)
		}
	}
}

func (c *RemoteClient) heartbeat() {
	// Collect system resource metrics
	cpu, mem, disk := collectSystemMetrics()

	body, _ := json.Marshal(map[string]interface{}{
		"status":            "idle",
		"capabilities":      []string{"subfinder", "naabu", "httpx", "nuclei"},
		"template_versions": "{}",
		"cpu_percent":       cpu,
		"mem_percent":       mem,
		"disk_percent":      disk,
	})
	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/workers/%s/heartbeat", c.coreURL, c.workerID), bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("[worker] heartbeat error: %v", err)
		return
	}
	resp.Body.Close()
}

// StartPolling long-polls for tasks from the core server.
func (c *RemoteClient) StartPolling() {
	go func() {
		for {
			select {
			case <-c.stopCh:
				return
			default:
			}

			req, _ := http.NewRequest("GET", fmt.Sprintf("%s/workers/%s/tasks/poll", c.coreURL, c.workerID), nil)
			req.Header.Set("Authorization", "Bearer "+c.token)
			resp, err := c.httpClient.Do(req)
			if err != nil {
				log.Printf("[worker] poll error: %v", err)
				time.Sleep(5 * time.Second)
				continue
			}

			if resp.StatusCode == http.StatusNoContent {
				resp.Body.Close()
				continue // no task, immediately poll again
			}

			if resp.StatusCode == http.StatusUnauthorized {
				log.Printf("[worker] poll unauthorized (revoked?), stopping")
				resp.Body.Close()
				return
			}
			if resp.StatusCode == http.StatusNotFound {
				log.Printf("[worker] poll 404, re-registering...")
				resp.Body.Close()
				if err := c.Register("remote-worker"); err != nil {
					log.Printf("[worker] re-register failed: %v", err)
					// If re-register succeeds, heartbeat is sent inside Register.
				}
				continue
			}

			if resp.StatusCode == http.StatusOK {
				var task struct {
					TaskID  string                 `json:"task_id"`
					Type    string                 `json:"type"`
					Tool    string                 `json:"tool"`
					Payload map[string]interface{} `json:"payload"`
				}
				if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
					log.Printf("[worker] decode task: %v", err)
					resp.Body.Close()
					continue
				}
				resp.Body.Close()
				c.executeTask(task.TaskID, task.Tool, task.Payload)
				continue
			}

			resp.Body.Close()
			time.Sleep(1 * time.Second)
		}
	}()
}

func (c *RemoteClient) executeTask(taskID, tool string, payload map[string]interface{}) {
	log.Printf("[worker] executing task %s (tool=%s)", taskID, tool)

	// Execute the task using the local runner
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	if err := c.runner.Run(ctx, taskID); err != nil {
		log.Printf("[worker] task %s execution error: %v", taskID, err)

		// Report failure
		resultBody, _ := json.Marshal(map[string]interface{}{
			"status": "failed",
			"error":  err.Error(),
		})
		req, _ := http.NewRequest("POST", fmt.Sprintf("%s/tasks/%s/result", c.coreURL, taskID), bytes.NewReader(resultBody))
		req.Header.Set("Authorization", "Bearer "+c.token)
		req.Header.Set("Content-Type", "application/json")
		resp, err := c.httpClient.Do(req)
		if err != nil {
			log.Printf("[worker] report result error: %v", err)
			return
		}
		resp.Body.Close()
		return
	}

	// Report success
	resultBody, _ := json.Marshal(map[string]interface{}{
		"status":    "completed",
		"artifacts": []map[string]interface{}{},
	})
	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/tasks/%s/result", c.coreURL, taskID), bytes.NewReader(resultBody))
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("[worker] report result error: %v", err)
		return
	}
	resp.Body.Close()
	log.Printf("[worker] task %s completed", taskID)
}

// Stop gracefully shuts down the remote client.
func (c *RemoteClient) Stop() {
	close(c.stopCh)
}

// collectSystemMetrics gathers CPU, memory, and disk usage.
// Returns (cpuPercent, memPercent, diskPercent).
func collectSystemMetrics() (cpu, mem, disk *float64) {
	// CPU usage approximation: use NumCPU as a placeholder
	// (real CPU usage requires sampling over time)
	numCPU := float64(runtime.NumCPU())
	if numCPU > 0 {
		v := 100.0 // placeholder: assume busy when reporting
		cpu = &v
	}

	// Memory stats from runtime
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	memUsed := float64(m.Alloc)
	memTotal := float64(m.Sys)
	if memTotal > 0 {
		v := (memUsed / memTotal) * 100
		mem = &v
	}

	// Disk stats from syscall
	var stat syscall.Statfs_t
	if err := syscall.Statfs("/data", &stat); err == nil {
		total := float64(stat.Blocks) * float64(stat.Bsize)
		free := float64(stat.Bavail) * float64(stat.Bsize)
		used := total - free
		if total > 0 {
			v := (used / total) * 100
			disk = &v
		}
	}

	return cpu, mem, disk
}
