package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"secbench/internal/db"
	"secbench/internal/errors"
	"secbench/internal/health"
	"secbench/internal/models"
	"secbench/internal/scope"
	"secbench/internal/util"
	"secbench/internal/worker"
)

// Server holds API dependencies.
type Server struct {
	queries    *db.Queries
	scopeEng   *scope.Engine
	worker     *worker.Runner
	health     *health.Checker
	dataDir    string
	sseClients map[string]chan []byte
}

func NewServer(queries *db.Queries, dataDir string) *Server {
	scopeEng := scope.NewEngine(queries)
	s := &Server{
		queries:    queries,
		scopeEng:   scopeEng,
		worker:     worker.NewRunner(queries, scopeEng, dataDir),
		health:     health.NewChecker(queries),
		dataDir:    dataDir,
		sseClients: make(map[string]chan []byte),
	}
	return s
}

func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /projects", s.handleCreateProject)
	mux.HandleFunc("GET /projects", s.handleListProjects)
	mux.HandleFunc("GET /projects/{id}", s.handleGetProject)
	mux.HandleFunc("POST /projects/{id}/targets", s.handleCreateTarget)
	mux.HandleFunc("GET /projects/{id}/targets", s.handleListTargets)
	mux.HandleFunc("POST /scope-rules", s.handleCreateScopeRule)
	mux.HandleFunc("GET /scope-rules", s.handleListScopeRules)
	mux.HandleFunc("POST /scan-plans", s.handleCreateScanPlan)
	mux.HandleFunc("POST /scan-plans/{id}/approve", s.handleApprovePlan)
	mux.HandleFunc("POST /scan-plans/dry-run", s.handleDryRun)
	mux.HandleFunc("GET /scan-tasks/{id}", s.handleGetTask)
	mux.HandleFunc("POST /scan-tasks/{id}/cancel", s.handleCancelTask)
	mux.HandleFunc("POST /tasks/run", s.handleRunTask)
	mux.HandleFunc("GET /tasks/{id}/artifacts", s.handleListArtifacts)
	mux.HandleFunc("GET /health/tools", s.handleToolHealth)
	mux.HandleFunc("POST /health/check", s.handleHealthCheck)
	mux.HandleFunc("GET /events", s.handleSSE)
}

// --- Error helpers ---

func writeError(w http.ResponseWriter, status int, err *errors.AppError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": err,
	})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// --- CORS middleware ---

func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// --- Projects ---

func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name         string `json:"name"`
		Organization string `json:"organization"`
		Purpose      string `json:"purpose"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "invalid request body").WithDetail(err.Error()))
		return
	}

	p := &models.Project{
		ID:             util.GenerateID(),
		Name:           req.Name,
		Organization:   req.Organization,
		Purpose:        req.Purpose,
		DefaultProfile: "standard",
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	if err := s.queries.CreateProject(p); err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "create project failed: %v", err))
		return
	}

	_ = s.queries.CreateAuditLog(&models.AuditLog{
		ID:           util.GenerateID(),
		ProjectID:    p.ID,
		Actor:        "user",
		Action:       "project.create",
		ResourceType: "project",
		ResourceID:   p.ID,
		Summary:      fmt.Sprintf("Created project %s", p.Name),
		CreatedAt:    time.Now().UTC(),
	})

	writeJSON(w, http.StatusCreated, p)
}

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := s.queries.ListProjects()
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "list projects failed: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, projects)
}

func (s *Server) handleGetProject(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	p, err := s.queries.GetProject(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "get project failed: %v", err))
		return
	}
	if p == nil {
		writeError(w, http.StatusNotFound, errors.New(errors.ErrNotFound, "project not found"))
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// --- Targets ---

func (s *Server) handleCreateTarget(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")

	var req struct {
		Type  string `json:"type"`
		Value string `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "invalid request body").WithDetail(err.Error()))
		return
	}

	t := &models.Target{
		ID:        util.GenerateID(),
		ProjectID: projectID,
		Type:      models.TargetType(req.Type),
		Value:     req.Value,
		Source:    "manual",
		Status:    "active",
		CreatedAt: time.Now().UTC(),
	}
	if err := s.queries.CreateTarget(t); err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "create target failed: %v", err))
		return
	}

	writeJSON(w, http.StatusCreated, t)
}

func (s *Server) handleListTargets(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	targets, err := s.queries.ListTargetsByProject(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "list targets failed: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, targets)
}

// --- Scope Rules ---

func (s *Server) handleCreateScopeRule(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProjectID string `json:"project_id"`
		Action    string `json:"action"`
		Type      string `json:"type"`
		Value     string `json:"value"`
		Reason    string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "invalid request body").WithDetail(err.Error()))
		return
	}

	sr := &models.ScopeRule{
		ID:        util.GenerateID(),
		ProjectID: req.ProjectID,
		Action:    models.ScopeAction(req.Action),
		Type:      models.TargetType(req.Type),
		Value:     req.Value,
		Reason:    req.Reason,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := s.queries.CreateScopeRule(sr); err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "create scope rule failed: %v", err))
		return
	}

	writeJSON(w, http.StatusCreated, sr)
}

func (s *Server) handleListScopeRules(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")
	if projectID == "" {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "missing project_id"))
		return
	}
	rules, err := s.queries.ListScopeRulesByProject(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "list scope rules failed: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, rules)
}

// --- Scan Plans ---

func (s *Server) handleCreateScanPlan(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProjectID    string `json:"project_id"`
		WorkflowType string `json:"workflow_type"`
		Profile      string `json:"profile"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "invalid request body").WithDetail(err.Error()))
		return
	}

	plan := &models.ScanPlan{
		ID:           util.GenerateID(),
		ProjectID:    req.ProjectID,
		WorkflowType: req.WorkflowType,
		Profile:      models.ScanProfile(req.Profile),
		Status:       "draft",
		CreatedBy:    "user",
		CreatedAt:    time.Now().UTC(),
	}
	if err := s.queries.CreateScanPlan(plan); err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "create scan plan failed: %v", err))
		return
	}

	writeJSON(w, http.StatusCreated, plan)
}

func (s *Server) handleApprovePlan(w http.ResponseWriter, r *http.Request) {
	// Placeholder: update plan status.
	writeJSON(w, http.StatusOK, map[string]string{"status": "approved"})
}

func (s *Server) handleDryRun(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")
	if projectID == "" {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "missing project_id"))
		return
	}

	targets, err := s.queries.ListTargetsByProject(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "list targets failed: %v", err))
		return
	}

	var results []map[string]interface{}
	for _, t := range targets {
		decision, err := s.scopeEng.Check(r.Context(), projectID, t)
		if err != nil {
			continue
		}
		results = append(results, map[string]interface{}{
			"target":   t.Value,
			"type":     t.Type,
			"decision": decision.Decision,
			"reason":   decision.Reason,
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"project_id": projectID,
		"results":    results,
		"mode":       "dry-run",
	})
}

// --- Tasks ---

func (s *Server) handleRunTask(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProjectID string `json:"project_id"`
		PlanID    string `json:"plan_id"`
		Tool      string `json:"tool"`
		TargetID  string `json:"target_id"`
		Command   string `json:"command"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "invalid request body").WithDetail(err.Error()))
		return
	}

	// Validate target exists.
	if req.TargetID != "" {
		target, err := s.queries.GetTarget(req.TargetID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "get target failed: %v", err))
			return
		}
		if target == nil {
			writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "target not found"))
			return
		}
	}

	// If no plan_id provided, create a default one to satisfy FK constraint.
	planID := req.PlanID
	if planID == "" {
		plan := &models.ScanPlan{
			ID:           util.GenerateID(),
			ProjectID:    req.ProjectID,
			WorkflowType: "ad-hoc",
			Profile:      models.ProfileStandard,
			Status:       "approved",
			CreatedBy:    "user",
			CreatedAt:    time.Now().UTC(),
		}
		if err := s.queries.CreateScanPlan(plan); err != nil {
			writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "create default plan failed: %v", err))
			return
		}
		planID = plan.ID
	}

	task := &models.ScanTask{
		ID:                util.GenerateID(),
		ProjectID:         req.ProjectID,
		PlanID:            planID,
		TargetID:          nil,
		Tool:              req.Tool,
		CommandTemplate:   req.Command,
		ArgumentsRedacted: redactArgs(req.Command),
		Status:            models.TaskQueued,
		CreatedAt:         time.Now().UTC(),
	}
	if req.TargetID != "" {
		task.TargetID = &req.TargetID
	}
	if err := s.queries.CreateScanTask(task); err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "create scan task failed: %v", err))
		return
	}

	// Determine timeout based on tool type.
	timeout := defaultToolTimeout(task.Tool)

	// Run in background goroutine (MVP Worker).
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		if err := s.worker.Run(ctx, task.ID); err != nil {
			log.Printf("task %s failed: %v", task.ID, err)
			// If context deadline exceeded, mark as failed with timeout indication.
			if ctx.Err() == context.DeadlineExceeded {
				now := time.Now().UTC()
				exitCode := -1
				_ = s.queries.UpdateScanTaskStatus(task.ID, models.TaskFailed, &exitCode, &now)
			}
		}
		// Notify SSE clients.
		s.broadcastSSE(map[string]interface{}{
			"event":   "task_update",
			"task_id": task.ID,
		})
	}()

	writeJSON(w, http.StatusCreated, task)
}

func (s *Server) handleGetTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	task, err := s.queries.GetScanTask(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "get task failed: %v", err))
		return
	}
	if task == nil {
		writeError(w, http.StatusNotFound, errors.New(errors.ErrNotFound, "task not found"))
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (s *Server) handleCancelTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	now := time.Now().UTC()
	_ = s.queries.UpdateScanTaskStatus(id, models.TaskCancelled, nil, &now)
	_ = s.worker.Cancel(id)
	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

func (s *Server) handleListArtifacts(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	artifacts, err := s.queries.ListRawArtifactsByTask(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "list artifacts failed: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, artifacts)
}

// --- Health ---

func (s *Server) handleToolHealth(w http.ResponseWriter, r *http.Request) {
	h, err := s.queries.ListToolHealth()
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "list tool health failed: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, h)
}

func (s *Server) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	if err := s.health.CheckAll(r.Context(), s.dataDir); err != nil {
		log.Printf("health check error: %v", err)
	}
	s.handleToolHealth(w, r)
}

// --- SSE ---

func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	clientID := util.GenerateID()
	ch := make(chan []byte, 10)
	s.sseClients[clientID] = ch
	defer delete(s.sseClients, clientID)

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, errors.New(errors.ErrInternal, "SSE not supported"))
		return
	}

	// Send initial connection event.
	fmt.Fprintf(w, "data: %s\n\n", `{"event":"connected"}`)
	flusher.Flush()

	for {
		select {
		case msg := <-ch:
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func (s *Server) broadcastSSE(data map[string]interface{}) {
	b, _ := json.Marshal(data)
	for _, ch := range s.sseClients {
		select {
		case ch <- b:
		default:
		}
	}
}

// --- Helpers ---

func defaultToolTimeout(tool string) time.Duration {
	switch strings.ToLower(tool) {
	case "subfinder":
		return 300 * time.Second
	case "httpx":
		return 300 * time.Second
	case "naabu":
		return 600 * time.Second
	case "nuclei":
		return 1800 * time.Second
	case "nmap":
		return 600 * time.Second
	default:
		return 300 * time.Second
	}
}

func redactArgs(cmd string) string {
	// Simple redaction: replace anything that looks like a key/token.
	parts := strings.Fields(cmd)
	for i, p := range parts {
		if strings.Contains(strings.ToLower(p), "key") ||
			strings.Contains(strings.ToLower(p), "token") ||
			strings.Contains(strings.ToLower(p), "secret") ||
			strings.Contains(strings.ToLower(p), "password") {
			parts[i] = "[REDACTED]"
		}
	}
	return strings.Join(parts, " ")
}
