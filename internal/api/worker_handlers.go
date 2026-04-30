package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"github.com/P0m32Kun/Anchor/internal/errors"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// Remote worker task queue
func (s *Server) initTaskQueue() {
	s.taskQueue = make(map[string]chan *models.ScanTask)
	s.taskResults = make(map[string]chan map[string]interface{})
}

func (s *Server) enqueueTask(workerID string, task *models.ScanTask) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ch, ok := s.taskQueue[workerID]; ok {
		select {
		case ch <- task:
		default:
		}
	}
}

func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// POST /workers/register
func (s *Server) handleRegisterWorker(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name     string `json:"name"`
		Endpoint string `json:"endpoint"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "invalid request body"))
		return
	}

	id := util.GenerateID()
	token := generateToken()
	now := time.Now().UTC()

	worker := &models.WorkerNode{
		ID:       id,
		Name:     req.Name,
		Endpoint: req.Endpoint,
		Mode:       models.WorkerModeRemote,
		Status:     models.WorkerStatusOnline,
		TrustLevel: "standard",
		LastSeen:   &now,
		CreatedAt:  now,
	}

	if err := s.queries.CreateWorkerNode(worker); err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "create worker: %v", err))
		return
	}

	// Initialize task channel for this worker
	s.mu.Lock()
	s.taskQueue[id] = make(chan *models.ScanTask, 10)
	s.taskResults[id] = make(chan map[string]interface{}, 10)
	s.mu.Unlock()

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"worker_id": id,
		"token":     token,
	})
}

// POST /workers/{id}/heartbeat
func (s *Server) handleWorkerHeartbeat(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req struct {
		Status       string   `json:"status"`
		Capabilities []string `json:"capabilities"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "invalid body"))
		return
	}

	status := models.WorkerStatus(req.Status)
	if req.Status == "idle" {
		status = models.WorkerStatusOnline
	}

	now := time.Now().UTC()
	if err := s.queries.UpdateWorkerNodeStatus(id, status, now); err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "update heartbeat: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// GET /workers/{id}/tasks/poll
func (s *Server) handlePollTasks(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	// Check if worker exists and is not revoked
	worker, err := s.queries.GetWorkerNode(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "get worker: %v", err))
		return
	}
	if worker == nil {
		writeError(w, http.StatusNotFound, errors.New(errors.ErrNotFound, "worker not found"))
		return
	}
	if worker.RevokedAt != nil {
		writeError(w, http.StatusUnauthorized, errors.New(errors.ErrBadRequest, "worker revoked"))
		return
	}

	// Get or create task channel
	s.mu.Lock()
	ch, ok := s.taskQueue[id]
	if !ok {
		ch = make(chan *models.ScanTask, 10)
		s.taskQueue[id] = ch
	}
	s.mu.Unlock()

	// Try to get a task with timeout
	select {
	case task := <-ch:
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"task_id": task.ID,
			"type":    "scan",
			"tool":    task.Tool,
			"payload": map[string]interface{}{
				"command_template": task.CommandTemplate,
				"project_id":       task.ProjectID,
			},
		})
	case <-time.After(5 * time.Second):
		w.WriteHeader(http.StatusNoContent)
	}
}

// POST /tasks/{id}/result
func (s *Server) handleTaskResult(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	var req struct {
		Status    string                   `json:"status"`
		Artifacts []map[string]interface{} `json:"artifacts"`
		Error     string                   `json:"error"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "invalid body"))
		return
	}

	// Update task status in database
	status := models.TaskStatus(req.Status)
	now := time.Now().UTC()
	var exitCode int
	if status == models.TaskFailed {
		exitCode = 1
	}

	if err := s.queries.UpdateScanTaskStatus(taskID, status, &exitCode, &now); err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "update task: %v", err))
		return
	}

	// Check if associated run is complete
	task, err := s.queries.GetScanTask(taskID)
	if err == nil && task != nil && task.RunID != nil {
		go s.checkRunCompletion(*task.RunID)
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// POST /workers/{id}/revoke
func (s *Server) handleRevokeWorker(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	now := time.Now().UTC()
	if err := s.queries.RevokeWorkerNode(id, now); err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "revoke worker: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

// DELETE /workers/{id}
func (s *Server) handleDeleteWorker(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	worker, err := s.queries.GetWorkerNode(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "get worker: %v", err))
		return
	}
	if worker == nil {
		writeError(w, http.StatusNotFound, errors.New(errors.ErrNotFound, "worker not found"))
		return
	}
	if worker.Status != models.WorkerStatusOffline {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "只能删除离线的 Worker"))
		return
	}

	s.mu.Lock()
	if ch, ok := s.taskQueue[id]; ok {
		close(ch)
	}
	delete(s.taskQueue, id)
	delete(s.taskResults, id)
	s.mu.Unlock()

	if err := s.queries.DeleteWorkerNode(id); err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "delete worker: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"status": "deleted"})
}
