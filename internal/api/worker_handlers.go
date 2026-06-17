package api

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/P0m32Kun/Anchor/internal/errors"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

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

func (s *Server) handleRegisterWorker(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name            string `json:"name"`
		Endpoint        string `json:"endpoint"`
		MaxConcurrency  int    `json:"max_concurrency"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "invalid request body"))
		return
	}

	id := util.GenerateID()
	token := generateToken()
	now := time.Now().UTC()

	maxConc := 10
	if req.MaxConcurrency > 0 {
		maxConc = req.MaxConcurrency
	}

	worker := &models.WorkerNode{
		ID:             id,
		Name:           req.Name,
		Endpoint:       req.Endpoint,
		Mode:           models.WorkerModeRemote,
		Status:         models.WorkerStatusOnline,
		TrustLevel:     "standard",
		MaxConcurrency: maxConc,
		LastSeen:       &now,
		CreatedAt:      now,
	}

	if err := s.queries.CreateWorkerNode(worker); err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "create worker: %v", err))
		return
	}

	s.mu.Lock()
	s.taskQueue[id] = make(chan *models.ScanTask, 10)
	s.taskResults[id] = make(chan map[string]interface{}, 10)
	s.mu.Unlock()

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"worker_id": id,
		"token":     token,
	})
}

func (s *Server) handleWorkerHeartbeat(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req struct {
		Status            string   `json:"status"`
		Capabilities      []string `json:"capabilities"`
		TemplateVersions  string   `json:"template_versions"`
		CPUPercent        *float64 `json:"cpu_percent"`
		MemPercent        *float64 `json:"mem_percent"`
		DiskPercent       *float64 `json:"disk_percent"`
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
	if req.TemplateVersions != "" {
		if err := s.queries.UpdateWorkerNodeTemplateVersions(id, status, now, req.TemplateVersions); err != nil {
			writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "update heartbeat: %v", err))
			return
		}
	} else {
		if err := s.queries.UpdateWorkerNodeStatus(id, status, now); err != nil {
			writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "update heartbeat: %v", err))
			return
		}
	}

	// Update resource metrics if provided
	s.updateWorkerMetrics(id, req.CPUPercent, req.MemPercent, req.DiskPercent)

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
// updateWorkerMetrics updates worker resource metrics if provided.
func (s *Server) updateWorkerMetrics(id string, cpu, mem, disk *float64) {
	if cpu == nil && mem == nil && disk == nil {
		return
	}
	now := time.Now().UTC()
	if err := s.queries.UpdateWorkerNodeMetrics(id, cpu, mem, disk, now); err != nil {
		log.Printf("[worker] update metrics for %s: %v", id, err)
	}
}

func (s *Server) handlePollTasks(w http.ResponseWriter, r *http.Request) {
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
	if worker.RevokedAt != nil {
		writeError(w, http.StatusUnauthorized, errors.New(errors.ErrBadRequest, "worker revoked"))
		return
	}

	s.mu.Lock()
	ch, ok := s.taskQueue[id]
	if !ok {
		ch = make(chan *models.ScanTask, 10)
		s.taskQueue[id] = ch
	}
	s.mu.Unlock()

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

	status := models.TaskStatus(req.Status)
	now := time.Now().UTC()
	var exitCode int
	if status == models.TaskFailed {
		exitCode = 1
	}

	// Save worker artifacts BEFORE updating task status to avoid race with
	// dispatchToWorker polling for completion.
	task, _ := s.queries.GetScanTask(taskID)
	for _, art := range req.Artifacts {
		artType, _ := art["type"].(string)
		artName, _ := art["name"].(string)
		var data []byte
		switch v := art["data"].(type) {
		case []byte:
			data = v
		case string:
			decoded, err := base64.StdEncoding.DecodeString(v)
			if err != nil {
				data = []byte(v)
			} else {
				data = decoded
			}
		default:
			continue
		}
		var workdir string
		if task != nil {
			workdir = filepath.Join(s.dataDir, "workdirs", task.ProjectID, taskID)
		} else {
			workdir = filepath.Join(s.dataDir, "workdirs", taskID)
		}
		_ = os.MkdirAll(workdir, 0750)
		filename := artName
		if filename == "" {
			filename = fmt.Sprintf("%s_%d.txt", artType, time.Now().UnixNano())
		}
		path := filepath.Join(workdir, filename)
		if err := os.WriteFile(path, data, 0640); err != nil {
			log.Printf("[server] save artifact %s: %v", path, err)
			continue
		}
		sum := sha256.Sum256(data)
		a := &models.RawArtifact{
			ID:              util.GenerateID(),
			ProjectID:       task.ProjectID,
			TaskID:          &taskID,
			Type:            models.ArtifactType(artType),
			Path:            path,
			SHA256:          fmt.Sprintf("%x", sum),
			Size:            int64(len(data)),
			RedactionStatus: "unchecked",
			CreatedAt:       now,
		}
		if err := s.queries.CreateRawArtifact(a); err != nil {
			log.Printf("[server] create raw artifact: %v", err)
		}
	}

	if err := s.queries.UpdateScanTaskStatus(taskID, status, &exitCode, &now); err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "update task: %v", err))
		return
	}

	if req.Error != "" {
		if err := s.queries.UpdateScanTaskErrorMessage(taskID, req.Error); err != nil {
			log.Printf("[server] update task %s error_message failed: %v", taskID, err)
		}
	}

	if req.Error != "" {
		_ = s.queries.UpdateScanTaskErrorMessage(taskID, req.Error)
	}

	if task != nil && task.RunID != nil {
		go s.checkRunCompletion(*task.RunID)
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleRevokeWorker(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	now := time.Now().UTC()
	if err := s.queries.RevokeWorkerNode(id, now); err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "revoke worker: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

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
