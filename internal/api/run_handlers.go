package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"github.com/P0m32Kun/Anchor/internal/errors"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// POST /projects/{id}/runs
func (s *Server) handleCreateRun(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	var req struct {
		ToolTemplateID string `json:"tool_template_id"`
		Name           string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "invalid body"))
		return
	}

	if req.Name == "" {
		req.Name = "未命名扫描"
	}

	run := &models.Run{
		ID:        util.GenerateID(),
		ProjectID: projectID,
		Name:      req.Name,
		Status:    models.RunPending,
		CreatedAt: time.Now().UTC(),
	}
	if req.ToolTemplateID != "" {
		run.ToolTemplateID = &req.ToolTemplateID
	}

	if err := s.queries.CreateRun(run); err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "create run: %v", err))
		return
	}

	// Auto-start: create scan tasks and dispatch to workers
	if run.ToolTemplateID != nil {
		go s.dispatchRun(run)
	}

	writeJSON(w, http.StatusCreated, run)
}

// dispatchRun creates scan tasks from tool template and dispatches them to workers.
func (s *Server) dispatchRun(run *models.Run) {
	// 1. Get tool template
	template, err := s.queries.GetToolTemplate(*run.ToolTemplateID)
	if err != nil {
		log.Printf("[run] get template: %v", err)
		s.updateRunStatus(run.ID, models.RunFailed, nil)
		return
	}
	if template == nil {
		log.Printf("[run] template not found: %s", *run.ToolTemplateID)
		s.updateRunStatus(run.ID, models.RunFailed, nil)
		return
	}

	// 2. Parse tools JSON
	var tools []models.TemplateTool
	if err := json.Unmarshal([]byte(template.ToolsJSON), &tools); err != nil {
		log.Printf("[run] parse tools_json: %v", err)
		s.updateRunStatus(run.ID, models.RunFailed, nil)
		return
	}

	// 3. Update run to running
	now := time.Now().UTC()
	if err := s.queries.UpdateRunStatus(run.ID, models.RunRunning, &now, nil); err != nil {
		log.Printf("[run] update status: %v", err)
		return
	}

	// 4. Create scan tasks for each tool
	var tasks []*models.ScanTask
	for _, tool := range tools {
		if !tool.Enabled {
			continue
		}
		task := &models.ScanTask{
			ID:              util.GenerateID(),
			ProjectID:       run.ProjectID,
			RunID:           &run.ID,
			Tool:            tool.Tool,
			CommandTemplate: fmt.Sprintf("%s -d {{target}}", tool.Tool),
			Status:          models.TaskCreated,
			CreatedAt:       time.Now().UTC(),
		}
		if err := s.queries.CreateScanTask(task); err != nil {
			log.Printf("[run] create scan task: %v", err)
			continue
		}
		tasks = append(tasks, task)
	}

	if len(tasks) == 0 {
		log.Printf("[run] no tasks created")
		s.updateRunStatus(run.ID, models.RunFailed, nil)
		return
	}

	// 5. Dispatch tasks to available workers
	s.dispatchTasksToWorkers(tasks)
}

// dispatchTasksToWorkers assigns tasks to available workers (local or remote).
func (s *Server) dispatchTasksToWorkers(tasks []*models.ScanTask) {
	for _, task := range tasks {
		// Try local worker first
		if s.workerProc != nil && s.workerEndpoint != "" {
			if s.enqueueToLocalWorker(task) {
				continue
			}
		}

		// Try remote workers
		if s.enqueueToRemoteWorker(task) {
			continue
		}

		// No worker available, log and retry later
		log.Printf("[run] no worker available for task %s", task.ID)
	}
}

func (s *Server) enqueueToLocalWorker(task *models.ScanTask) bool {
	// Local worker expects: task_id, tool, command, workdir, rate_limit
	payload := map[string]interface{}{
		"task_id": task.ID,
		"tool":    task.Tool,
		"command": []string{task.Tool, "-d", "{{target}}"},
		"workdir": filepath.Join(s.dataDir, "workdirs", task.ID),
	}
	b, _ := json.Marshal(payload)
	resp, err := s.workerHTTPClient.Post(
		s.workerEndpoint+"/tasks",
		"application/json",
		bytes.NewBuffer(b),
	)
	if err != nil {
		return false
	}
	resp.Body.Close()
	if resp.StatusCode == http.StatusAccepted || resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
		return true
	}
	return false
}

func (s *Server) enqueueToRemoteWorker(task *models.ScanTask) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Find an available remote worker (online status, not revoked)
	workers, err := s.queries.ListWorkerNodes()
	if err != nil {
		return false
	}

	for _, w := range workers {
		if w.Status != models.WorkerStatusOnline || w.RevokedAt != nil {
			continue
		}
		if ch, ok := s.taskQueue[w.ID]; ok {
			select {
			case ch <- task:
				log.Printf("[run] dispatched task %s to worker %s", task.ID, w.ID)
				return true
			default:
				// channel full, try next worker
			}
		}
	}
	return false
}

func (s *Server) updateRunStatus(runID string, status models.RunStatus, startedAt *time.Time) {
	now := time.Now().UTC()
	var finishedAt *time.Time
	if status == models.RunCompleted || status == models.RunFailed || status == models.RunCancelled {
		finishedAt = &now
	}
	if err := s.queries.UpdateRunStatus(runID, status, startedAt, finishedAt); err != nil {
		log.Printf("[run] update status: %v", err)
	}
}

// checkRunCompletion checks if all tasks for a run are done and updates run status.
func (s *Server) checkRunCompletion(runID string) {
	tasks, err := s.queries.ListScanTasksByRun(runID)
	if err != nil {
		log.Printf("[run] check completion error: %v", err)
		return
	}

	allDone := true
	hasFailed := false
	for _, t := range tasks {
		if t.Status != models.TaskCompleted && t.Status != models.TaskFailed && t.Status != models.TaskCancelled {
			allDone = false
			break
		}
		if t.Status == models.TaskFailed {
			hasFailed = true
		}
	}

	if allDone {
		status := models.RunCompleted
		if hasFailed {
			status = models.RunFailed
		}
		s.updateRunStatus(runID, status, nil)
		log.Printf("[run] %s completed with status %s", runID, status)
	}
}

// GET /projects/{id}/runs
func (s *Server) handleListRuns(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	runs, err := s.queries.ListRunsByProject(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "list runs: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, runs)
}

// GET /runs/{id}
func (s *Server) handleGetRun(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	run, err := s.queries.GetRun(runID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "get run: %v", err))
		return
	}
	if run == nil {
		writeError(w, http.StatusNotFound, errors.New(errors.ErrNotFound, "run not found"))
		return
	}
	writeJSON(w, http.StatusOK, run)
}

// GET /runs/{id}/tasks
func (s *Server) handleGetRunTasks(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	tasks, err := s.queries.ListScanTasksByRun(runID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "list tasks: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, tasks)
}
