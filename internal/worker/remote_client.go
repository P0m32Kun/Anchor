package worker

import (
	archive_tar "archive/tar"
	"compress/gzip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// RemoteClient connects a worker node to a core server.
type RemoteClient struct {
	coreURL    string
	workerID   string
	token      string
	endpoint   string
	dataDir    string
	syncer     *BundleSyncer
	httpClient *http.Client
	stopCh     chan struct{}
}

// NewRemoteClient creates a client that registers with the core server.
// apiToken is the global server token required for all API calls.
// dataDir is used for local bundle storage.
func NewRemoteClient(coreURL, endpoint, apiToken, dataDir string) *RemoteClient {
	return &RemoteClient{
		coreURL:    coreURL,
		endpoint:   endpoint,
		token:      apiToken,
		dataDir:    dataDir,
		syncer:     NewBundleSyncer(dataDir, coreURL, apiToken),
		httpClient: &http.Client{Timeout: 30 * time.Second},
		stopCh:     make(chan struct{}),
	}
}

// Register the worker with the core server.
func (c *RemoteClient) Register(name string) error {
	body, _ := json.Marshal(map[string]string{
		"name":     name,
		"endpoint": c.endpoint,
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

// StartBundleSync starts a periodic bundle sync loop. It syncs immediately on
// start, then every interval.
func (c *RemoteClient) StartBundleSync(interval time.Duration) {
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

// syncSources fetches all enabled custom template sources from the server
// and syncs each to its own directory under ~/nuclei-templates/.
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
		Status      string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&sources); err != nil {
		log.Printf("[worker] decode sources: %v", err)
		return
	}

	// Sync each enabled source to its own directory
	for _, src := range sources {
		if !src.Enabled || src.Status != "ready" {
			continue
		}
		if err := c.syncSource(src.ID, src.InstallPath); err != nil {
			log.Printf("[worker] sync source %s (%s): %v", src.ID, src.Name, err)
		}
	}
}

// syncSource downloads a single source's files and extracts them to
// ~/nuclei-templates/{installPath}/ (nuclei's default search path).
func (c *RemoteClient) syncSource(sourceID, installPath string) error {
	if installPath == "" {
		return fmt.Errorf("source %s has no install_path", sourceID)
	}

	// Fetch the source bundle as a tar.gz
	req, _ := http.NewRequest("GET", c.coreURL+"/nuclei/custom/sources/"+sourceID+"/bundle", nil)
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("download bundle: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("bundle not found (source may not be published yet)")
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download bundle: %s", resp.Status)
	}

	// Extract to ~/nuclei-templates/ (archive entries are prefixed with installPath/)
	home, _ := os.UserHomeDir()
	targetDir := filepath.Join(home, "nuclei-templates")

	if err := extractTarGzToDir(resp.Body, targetDir); err != nil {
		return fmt.Errorf("extract: %w", err)
	}

	log.Printf("[worker] synced source %s -> %s", installPath, targetDir)
	return nil
}

func (c *RemoteClient) heartbeat() {
	body, _ := json.Marshal(map[string]interface{}{
		"status":            "idle",
		"capabilities":      []string{"subfinder", "naabu", "httpx", "nuclei"},
		"template_versions": c.syncer.TemplateVersionsJSON(),
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

	// TODO: integrate with actual tool execution
	// For now, simulate execution
	time.Sleep(2 * time.Second)

	// Report result
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

// extractTarGzToDir extracts a .tar.gz stream to targetDir.
func extractTarGzToDir(r io.Reader, targetDir string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("gzip reader: %w", err)
	}
	defer gz.Close()

	tr := archive_tar.NewReader(gz)

	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("mkdir target: %w", err)
	}

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar next: %w", err)
		}

		// Sanitize path
		name := filepath.Clean(hdr.Name)
		if name == ".." || filepath.HasPrefix(name, "../") || filepath.HasPrefix(name, "/") {
			return fmt.Errorf("unsafe tar entry: %s", hdr.Name)
		}

		target := filepath.Join(targetDir, name)

		switch hdr.Typeflag {
		case archive_tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return fmt.Errorf("mkdir %s: %w", name, err)
			}
		case archive_tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return fmt.Errorf("mkdir parent %s: %w", name, err)
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
			if err != nil {
				return fmt.Errorf("create %s: %w", name, err)
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return fmt.Errorf("write %s: %w", name, err)
			}
			f.Close()
		default:
			// Skip non-regular files
			continue
		}
	}
	return nil
}
