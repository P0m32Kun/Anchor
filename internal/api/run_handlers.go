package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/P0m32Kun/Anchor/internal/errors"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
	"github.com/P0m32Kun/Anchor/internal/workflows"
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

	// 2. Mark run as running
	now := time.Now().UTC()
	if err := s.queries.UpdateRunStatus(run.ID, models.RunRunning, &now, nil); err != nil {
		log.Printf("[run] update status: %v", err)
		return
	}

	// 3. Delegate to AssetDiscoveryWorkflow which handles:
	//    - querying project targets and classifying by type
	//    - building correct tool commands (subfinder -d, naabu -list, etc.)
	//    - creating hostlist files for batch tools
	//    - parsing tool output and creating assets/findings
	wf := workflows.NewAssetDiscoveryWorkflow(s.queries, s.worker, s.scopeEng, s.dataDir).WithRunID(run.ID)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	result, err := wf.Run(ctx, run.ProjectID)
	if err != nil {
		log.Printf("[run] workflow failed for run %s: %v", run.ID, err)
		s.updateRunStatus(run.ID, models.RunFailed, nil)
		return
	}

	log.Printf("[run] %s completed: domains=%d ips=%d ports=%d web=%d",
		run.ID, result.DomainsFound, result.IPsFound, result.PortsFound, result.WebEndpointsFound)

	// 4. Check final task states and update run status
	s.checkRunCompletion(run.ID)
}

// dispatchTasksToWorkers assigns tasks to available workers.
func (s *Server) dispatchTasksToWorkers(tasks []*models.ScanTask) {
	for _, task := range tasks {
		if s.enqueueToWorker(task) {
			continue
		}

		// No worker available, log and retry later
		log.Printf("[run] no worker available for task %s", task.ID)
	}
}

func (s *Server) enqueueToWorker(task *models.ScanTask) bool {
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
	page := parsePagination(r)
	total, err := s.queries.CountRunsByProject(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "count runs: %v", err))
		return
	}
	runs, err := s.queries.ListRunsByProjectPaginated(projectID, page.PageSize, page.Offset())
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "list runs: %v", err))
		return
	}
	writePaginatedJSON(w, runs, total, page)
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

func (s *Server) handleCancelRun(w http.ResponseWriter, r *http.Request) {
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

	if run.Status == models.RunCompleted || run.Status == models.RunFailed || run.Status == models.RunCancelled {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "run already finished"))
		return
	}

	tasks, err := s.queries.ListScanTasksByRun(runID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "list tasks: %v", err))
		return
	}

	now := time.Now().UTC()
	for _, task := range tasks {
		if task.Status == models.TaskCompleted || task.Status == models.TaskFailed || task.Status == models.TaskCancelled {
			continue
		}
		_ = s.queries.UpdateScanTaskStatus(task.ID, models.TaskCancelled, nil, &now)
		_ = s.worker.Cancel(task.ID)
	}

	s.updateRunStatus(runID, models.RunCancelled, nil)

	// Notify SSE clients.
	s.broadcastProjectSSE(run.ProjectID, map[string]interface{}{
		"event":   "task_update",
		"run_id":  runID,
	})

	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}
