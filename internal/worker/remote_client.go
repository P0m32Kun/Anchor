package worker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// RemoteClient connects a worker node to a core server.
type RemoteClient struct {
	coreURL    string
	workerID   string
	token      string
	endpoint   string
	httpClient *http.Client
	stopCh     chan struct{}
}

// NewRemoteClient creates a client that registers with the core server.
// apiToken is the global server token required for all API calls.
func NewRemoteClient(coreURL, endpoint, apiToken string) *RemoteClient {
	return &RemoteClient{
		coreURL:    coreURL,
		endpoint:   endpoint,
		token:      apiToken,
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

func (c *RemoteClient) heartbeat() {
	body, _ := json.Marshal(map[string]interface{}{
		"status":       "idle",
		"capabilities": []string{"subfinder", "naabu", "httpx", "nuclei"},
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
