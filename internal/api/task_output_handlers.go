package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/P0m32Kun/Anchor/internal/errors"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/worker"
)

// handleGetTaskOutput returns incremental tool stdout/stderr for a running or completed task.
// Query: stream=stdout|stderr (default stdout), offset=<byte offset>.
func (s *Server) handleGetTaskOutput(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	stream := r.URL.Query().Get("stream")
	offset, _ := strconv.ParseInt(r.URL.Query().Get("offset"), 10, 64)

	streamName, err := validateTaskOutputStream(stream)
	if err != nil {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, err.Error()))
		return
	}

	task, err := s.queries.GetScanTask(taskID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "get task: %v", err))
		return
	}
	if task == nil {
		writeError(w, http.StatusNotFound, errors.New(errors.ErrNotFound, "task not found"))
		return
	}

	workdir := filepath.Join(s.dataDir, "workdirs", task.ProjectID, taskID)
	running := task.Status == models.TaskRunning || task.Status == models.TaskQueued

	// Remote worker: tail from worker HTTP API while the process may still be running.
	if running && task.WorkerID != nil && *task.WorkerID != "" {
		node, err := s.queries.GetWorkerNode(*task.WorkerID)
		if err == nil && node != nil && node.Endpoint != "" {
			if proxied, perr := s.proxyWorkerTaskOutput(node.Endpoint, taskID, streamName, offset); perr == nil {
				writeJSON(w, http.StatusOK, proxied)
				return
			}
		}
	}

	content, next, fileEOF, err := worker.ReadTaskOutput(workdir, streamName, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "read output: %v", err))
		return
	}
	done := fileEOF && !running
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"stream":  streamName,
		"offset":  next,
		"content": content,
		"done":    done,
	})
}

func validateTaskOutputStream(stream string) (string, error) {
	switch stream {
	case "", "stdout":
		return "stdout", nil
	case "stderr":
		return "stderr", nil
	default:
		return "", fmt.Errorf("invalid stream %q", stream)
	}
}

func (s *Server) proxyWorkerTaskOutput(endpoint, taskID, stream string, offset int64) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/tasks/%s/output?stream=%s&offset=%d", strings.TrimRight(endpoint, "/"), taskID, stream, offset)
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("worker output %s: %s", resp.Status, string(body))
	}
	var out map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}
