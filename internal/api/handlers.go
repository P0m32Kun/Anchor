package api

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/errors"
	"github.com/P0m32Kun/Anchor/internal/health"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/report"
	"github.com/P0m32Kun/Anchor/internal/scope"
	"github.com/P0m32Kun/Anchor/internal/util"
	"github.com/P0m32Kun/Anchor/internal/worker"
	"github.com/P0m32Kun/Anchor/internal/workflow"
)

// Server holds API dependencies.
type Server struct {
	queries          *db.Queries
	rawDB            *sql.DB
	scopeEng         *scope.Engine
	worker           *worker.Runner
	health           *health.Checker
	dataDir          string
	sseClients       map[string]chan []byte
	workerEndpoint   string
	workerHTTPClient *http.Client
	workerProc       *os.Process
}

func NewServer(queries *db.Queries, rawDB *sql.DB, dataDir string) *Server {
	scopeEng := scope.NewEngine(queries)
	s := &Server{
		queries:    queries,
		rawDB:      rawDB,
		scopeEng:   scopeEng,
		worker:     worker.NewRunner(queries, scopeEng, dataDir),
		health:     health.NewChecker(queries),
		dataDir:    dataDir,
		sseClients: make(map[string]chan []byte),
		workerHTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
	// Auto-start local worker
	go func() {
		time.Sleep(1 * time.Second)
		if err := s.spawnLocalWorker(); err != nil {
			log.Printf("[server] auto-start local worker failed: %v", err)
		}
	}()
	// Monitor worker and auto-restart
	go s.monitorWorker()
	return s
}

func (s *Server) monitorWorker() {
	restartCount := 0
	for {
		time.Sleep(5 * time.Second)
		if s.workerProc == nil {
			if restartCount >= 3 {
				log.Printf("[server] worker restart limit reached, giving up")
				return
			}
			restartCount++
			log.Printf("[server] worker not running, restarting (attempt %d/3)", restartCount)
			if err := s.spawnLocalWorker(); err != nil {
				log.Printf("[server] worker restart failed: %v", err)
			}
		} else {
			restartCount = 0
		}
	}
}

func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /projects", s.handleCreateProject)
	mux.HandleFunc("GET /projects", s.handleListProjects)
	mux.HandleFunc("GET /projects/{id}", s.handleGetProject)
	mux.HandleFunc("POST /projects/{id}/targets", s.handleCreateTarget)
	mux.HandleFunc("POST /projects/{id}/targets/import", s.handleImportTargets)
	mux.HandleFunc("GET /projects/{id}/targets", s.handleListTargets)
	mux.HandleFunc("POST /projects/{id}/workflows/asset-discovery", s.handleStartAssetDiscovery)
	mux.HandleFunc("GET /projects/{id}/assets", s.handleListAssets)
	mux.HandleFunc("GET /projects/{id}/web-endpoints", s.handleListWebEndpointsByProject)
	mux.HandleFunc("GET /assets/{id}/ports", s.handleListPorts)
	mux.HandleFunc("GET /assets/{id}/services", s.handleListServices)
	mux.HandleFunc("POST /projects/{id}/workflows/web-screening", s.handleStartWebScreening)
	mux.HandleFunc("GET /projects/{id}/findings", s.handleListFindings)
	mux.HandleFunc("GET /findings/{id}", s.handleGetFinding)
	mux.HandleFunc("PATCH /findings/{id}/status", s.handlePatchFindingStatus)
	mux.HandleFunc("POST /findings/{id}/evidence", s.handleAddEvidence)
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
	mux.HandleFunc("GET /projects/{id}/reports/export.md", s.handleExportReportMD)
	mux.HandleFunc("GET /projects/{id}/reports/export.json", s.handleExportReportJSON)
	mux.HandleFunc("GET /tool-templates", s.handleListToolTemplates)
	mux.HandleFunc("GET /tool-templates/{id}", s.handleGetToolTemplate)
	mux.HandleFunc("GET /workers", s.handleListWorkers)
	mux.HandleFunc("POST /workers/local/start", s.handleStartLocalWorker)
	mux.HandleFunc("POST /workers/local/stop", s.handleStopLocalWorker)
	mux.HandleFunc("GET /workers/health", s.handleWorkerHealth)
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
		Name         string     `json:"name"`
		Organization string     `json:"organization"`
		Purpose      string     `json:"purpose"`
		StartTime    *time.Time `json:"start_time"`
		EndTime      *time.Time `json:"end_time"`
		RateLimit    int        `json:"rate_limit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "invalid request body").WithDetail(err.Error()))
		return
	}

	// Validate rate_limit.
	if req.RateLimit < 0 {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "rate_limit must be >= 0"))
		return
	}

	p := &models.Project{
		ID:             util.GenerateID(),
		Name:           req.Name,
		Organization:   req.Organization,
		Purpose:        req.Purpose,
		StartTime:      req.StartTime,
		EndTime:        req.EndTime,
		RateLimit:      req.RateLimit,
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

func (s *Server) handleImportTargets(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")

	// Verify project exists.
	project, err := s.queries.GetProject(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "get project failed: %v", err))
		return
	}
	if project == nil {
		writeError(w, http.StatusNotFound, errors.New(errors.ErrNotFound, "project not found"))
		return
	}

	// Parse multipart form (max 32 MB).
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "failed to parse multipart form").WithDetail(err.Error()))
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "missing file field").WithDetail(err.Error()))
		return
	}
	defer file.Close()

	// Determine parser based on file extension.
	var parsed []scope.ImportTarget
	name := strings.ToLower(header.Filename)
	if strings.HasSuffix(name, ".csv") {
		parsed, err = scope.ParseCSV(file)
	} else {
		parsed, err = scope.ParseTXT(file)
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrParse, "failed to parse file").WithDetail(err.Error()))
		return
	}

	if len(parsed) == 0 {
		writeJSON(w, http.StatusOK, scope.ImportResult{})
		return
	}

	// Deduplicate: track seen values within this import batch.
	seen := make(map[string]bool)
	var uniqueTargets []scope.ImportTarget
	for _, t := range parsed {
		if t.Value == "" {
			continue
		}
		if seen[t.Value] {
			continue
		}
		seen[t.Value] = true
		uniqueTargets = append(uniqueTargets, t)
	}

	// Scope-check each target and build result.
	now := time.Now().UTC()
	result := scope.ImportResult{}

	// Collect targets to create (use transaction for bulk insert).
	var toInsert []*models.Target

	for _, pt := range uniqueTargets {
		// Check for duplicate in DB.
		exists, dbErr := s.queries.TargetExistsByValue(projectID, pt.Value)
		if dbErr != nil {
			result.Errors++
			continue
		}
		if exists {
			result.Duplicates++
			continue
		}

		t := &models.Target{
			ID:        util.GenerateID(),
			ProjectID: projectID,
			Type:      pt.Type,
			Value:     pt.Value,
			Source:    "import",
			Status:    "active",
			CreatedAt: now,
		}

		// Run scope check.
		decision, chkErr := s.scopeEng.Check(r.Context(), projectID, t)
		if chkErr != nil {
			result.Errors++
			continue
		}

		if decision.Decision == models.ScopeDeny {
			result.Denied++
			result.DeniedTargets = append(result.DeniedTargets, scope.DeniedTarget{Value: t.Value, Reason: decision.Reason})
			continue
		}

		toInsert = append(toInsert, t)
		result.Targets = append(result.Targets, t)
		result.Imported++
	}

	// Bulk insert within a transaction.
	if len(toInsert) > 0 {
		txErr := db.WithTx(s.rawDB, func(tx *db.Queries) error {
			return tx.BulkCreateTargets(toInsert)
		})
		if txErr != nil {
			writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "bulk insert failed: %v", txErr))
			return
		}
	}

	writeJSON(w, http.StatusOK, result)
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

	project, err := s.queries.GetProject(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "get project failed: %v", err))
		return
	}
	if project == nil {
		writeError(w, http.StatusNotFound, errors.New(errors.ErrNotFound, "project not found"))
		return
	}

	targets, err := s.queries.ListTargetsByProject(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "list targets failed: %v", err))
		return
	}

	// Time window validation.
	timeWindowOK := true
	timeWindowReason := ""
	if denyReason := checkTimeWindowAPI(project); denyReason != "" {
		timeWindowOK = false
		timeWindowReason = denyReason
	}

	var results []map[string]interface{}
	allowCount := 0
	for _, t := range targets {
		decision, err := s.scopeEng.Check(r.Context(), projectID, t)
		if err != nil {
			results = append(results, map[string]interface{}{
				"target":   t.Value,
				"type":     t.Type,
				"decision": "error",
				"reason":   err.Error(),
			})
			continue
		}
		if decision.Decision == models.ScopeAllow {
			allowCount++
		}
		results = append(results, map[string]interface{}{
			"target":   t.Value,
			"type":     t.Type,
			"decision": decision.Decision,
			"reason":   decision.Reason,
		})
	}

	// Estimate execution time based on target count and profile.
	estimatedSeconds := estimateExecutionTime(len(targets), project.DefaultProfile)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"project_id":        projectID,
		"results":           results,
		"mode":              "dry-run",
		"time_window_valid":    timeWindowOK,
		"time_window_reason": timeWindowReason,
		"rate_limit":        project.RateLimit,
		"target_count":      len(targets),
		"allow_count":       allowCount,
		"estimated_seconds": estimatedSeconds,
	})
}

// estimateExecutionTime returns a rough estimate in seconds based on target count and profile.
func estimateExecutionTime(targetCount int, profile string) int {
	if targetCount == 0 {
		return 0
	}
	// Per-target base: light=30s, standard=60s, deep=120s
	var perTarget int
	switch profile {
	case "light":
		perTarget = 30
	case "deep":
		perTarget = 120
	default:
		perTarget = 60
	}
	return targetCount * perTarget
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

	// Time window check (TOCTOU protection: user might have changed window after scope check).
	project, err := s.queries.GetProject(req.ProjectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "get project failed: %v", err))
		return
	}
	if project == nil {
		writeError(w, http.StatusNotFound, errors.New(errors.ErrNotFound, "project not found"))
		return
	}
	if denyReason := checkTimeWindowAPI(project); denyReason != "" {
		writeError(w, http.StatusForbidden, errors.New(errors.ErrScopeDenied, denyReason))
		return
	}
	if project.RateLimit < 0 {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "rate_limit must be >= 0"))
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

func (s *Server) spawnLocalWorker() error {
	if s.workerProc != nil {
		return nil // already running
	}

	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable: %w", err)
	}

	cmd := exec.Command(execPath, "--worker", "--core-url", "http://localhost:17421")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start worker: %w", err)
	}

	s.workerProc = cmd.Process

	// Parse WORKER_READY line
	scanner := bufio.NewScanner(stdout)
	if scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "WORKER_READY ") {
			s.workerEndpoint = strings.TrimPrefix(line, "WORKER_READY ")
			log.Printf("[server] local worker ready at %s", s.workerEndpoint)
		}
	}

	go func() {
		cmd.Wait()
		log.Printf("[server] local worker exited")
		s.workerProc = nil
		s.workerEndpoint = ""
	}()

	return nil
}

func (s *Server) stopLocalWorker() error {
	if s.workerProc == nil {
		return nil
	}
	if err := s.workerProc.Signal(os.Interrupt); err != nil {
		return err
	}
	time.Sleep(2 * time.Second)
	if s.workerProc != nil {
		s.workerProc.Kill()
	}
	s.workerProc = nil
	s.workerEndpoint = ""
	return nil
}

func (s *Server) handleListWorkers(w http.ResponseWriter, r *http.Request) {
	workers := []map[string]interface{}{}
	status := "stopped"
	if s.workerProc != nil {
		status = "running"
	}
	workers = append(workers, map[string]interface{}{
		"id":       "local",
		"name":     "本地 Worker",
		"mode":     "local",
		"status":   status,
		"endpoint": s.workerEndpoint,
	})
	writeJSON(w, http.StatusOK, workers)
}

func (s *Server) handleStartLocalWorker(w http.ResponseWriter, r *http.Request) {
	if err := s.spawnLocalWorker(); err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "spawn worker: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "starting"})
}

func (s *Server) handleStopLocalWorker(w http.ResponseWriter, r *http.Request) {
	if err := s.stopLocalWorker(); err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "stop worker: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "stopped"})
}

func (s *Server) handleWorkerHealth(w http.ResponseWriter, r *http.Request) {
	if s.workerEndpoint == "" {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status": "no_worker",
			"tools":  []map[string]string{},
		})
		return
	}

	resp, err := s.workerHTTPClient.Get(s.workerEndpoint + "/health")
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status": "unreachable",
			"error":  err.Error(),
		})
		return
	}
	defer resp.Body.Close()

	var health map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status": "error",
			"error":  err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, health)
}

func (s *Server) handleListToolTemplates(w http.ResponseWriter, r *http.Request) {
	templates, err := s.queries.ListToolTemplates()
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.New(errors.ErrInternal, err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, templates)
}

func (s *Server) handleGetToolTemplate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	template, err := s.queries.GetToolTemplate(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.New(errors.ErrInternal, err.Error()))
		return
	}
	if template == nil {
		writeError(w, http.StatusNotFound, errors.New(errors.ErrNotFound, "tool-template not found: "+id))
		return
	}
	writeJSON(w, http.StatusOK, template)
}

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

// checkTimeWindowAPI returns a deny reason if now is outside the project's
// configured time window. Returns empty string if within window or unconfigured.
func checkTimeWindowAPI(project *models.Project) string {
	now := time.Now()
	if project.StartTime != nil && now.Before(*project.StartTime) {
		return "不在测试时间窗口内（未到开始时间）"
	}
	if project.EndTime != nil && now.After(*project.EndTime) {
		return "不在测试时间窗口内（已过结束时间）"
	}
	return ""
}

// --- Asset Discovery Workflow ---

func (s *Server) handleStartAssetDiscovery(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	project, err := s.queries.GetProject(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "get project failed: %v", err))
		return
	}
	if project == nil {
		writeError(w, http.StatusNotFound, errors.New(errors.ErrNotFound, "project not found"))
		return
	}
	if denyReason := checkTimeWindowAPI(project); denyReason != "" {
		writeError(w, http.StatusForbidden, errors.New(errors.ErrScopeDenied, denyReason))
		return
	}

	wf := workflow.NewAssetDiscoveryWorkflow(s.queries, s.worker, s.scopeEng, s.dataDir)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		result, err := wf.Run(ctx, projectID)
		if err != nil {
			log.Printf("asset discovery workflow failed for project %s: %v", projectID, err)
		}
		s.broadcastSSE(map[string]interface{}{
			"event":      "asset_discovery_complete",
			"project_id": projectID,
			"result":     result,
		})
	}()

	writeJSON(w, http.StatusAccepted, map[string]string{"status": "started"})
}

// --- Assets ---

func (s *Server) handleListAssets(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	assets, err := s.queries.ListAssetsByProject(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "list assets failed: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, assets)
}

func (s *Server) handleListWebEndpointsByProject(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	endpoints, err := s.queries.ListWebEndpointsByProject(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "list web endpoints failed: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, endpoints)
}

func (s *Server) handleListPorts(w http.ResponseWriter, r *http.Request) {
	assetID := r.PathValue("id")
	ports, err := s.queries.ListPortsByAsset(assetID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "list ports failed: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, ports)
}

func (s *Server) handleListServices(w http.ResponseWriter, r *http.Request) {
	assetID := r.PathValue("id")
	services, err := s.queries.ListServicesByAsset(assetID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "list services failed: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, services)
}

// --- Web Screening Workflow ---

func (s *Server) handleStartWebScreening(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	project, err := s.queries.GetProject(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "get project failed: %v", err))
		return
	}
	if project == nil {
		writeError(w, http.StatusNotFound, errors.New(errors.ErrNotFound, "project not found"))
		return
	}
	if denyReason := checkTimeWindowAPI(project); denyReason != "" {
		writeError(w, http.StatusForbidden, errors.New(errors.ErrScopeDenied, denyReason))
		return
	}

	wf := workflow.NewWebScreeningWorkflow(s.queries, s.worker, s.scopeEng, s.dataDir)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		result, err := wf.Run(ctx, projectID)
		if err != nil {
			log.Printf("web screening workflow failed for project %s: %v", projectID, err)
		}
		s.broadcastSSE(map[string]interface{}{
			"event":      "web_screening_complete",
			"project_id": projectID,
			"result":     result,
		})
	}()

	writeJSON(w, http.StatusAccepted, map[string]string{"status": "started"})
}

// --- Findings ---

func (s *Server) handleListFindings(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	status := r.URL.Query().Get("status")
	var findings []*models.Finding
	var err error
	if status != "" {
		findings, err = s.queries.ListFindingsByStatus(projectID, models.FindingStatus(status))
	} else {
		findings, err = s.queries.ListFindingsByProject(projectID)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "list findings failed: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, findings)
}

func (s *Server) handleGetFinding(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	finding, err := s.queries.GetFinding(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "get finding failed: %v", err))
		return
	}
	if finding == nil {
		writeError(w, http.StatusNotFound, errors.New(errors.ErrNotFound, "finding not found"))
		return
	}
	evidence, err := s.queries.ListEvidenceByFinding(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "list evidence failed: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"finding":  finding,
		"evidence": evidence,
	})
}

func (s *Server) handlePatchFindingStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	finding, err := s.queries.GetFinding(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "get finding failed: %v", err))
		return
	}
	if finding == nil {
		writeError(w, http.StatusNotFound, errors.New(errors.ErrNotFound, "finding not found"))
		return
	}
	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "invalid request body").WithDetail(err.Error()))
		return
	}
	validStatuses := map[string]bool{
		"confirmed":      true,
		"false_positive": true,
		"accepted_risk":  true,
		"ignored":        true,
		"pending_review": true,
	}
	if !validStatuses[req.Status] {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "invalid status"))
		return
	}
	now := time.Now().UTC()
	if err := s.queries.UpdateFindingStatus(id, models.FindingStatus(req.Status), now); err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "update finding status failed: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": req.Status})
}

func (s *Server) handleAddEvidence(w http.ResponseWriter, r *http.Request) {
	findingID := r.PathValue("id")
	var req struct {
		Type    string `json:"type"`
		Excerpt string `json:"excerpt"`
		CreatedBy string `json:"created_by"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "invalid request body").WithDetail(err.Error()))
		return
	}
	if req.Type == "" || req.Excerpt == "" {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "type and excerpt are required"))
		return
	}
	validTypes := map[string]bool{
		"note":       true,
		"screenshot": true,
		"file":       true,
	}
	if !validTypes[req.Type] {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "invalid evidence type"))
		return
	}
	ev := &models.Evidence{
		ID:        util.GenerateID(),
		FindingID: findingID,
		Type:      models.EvidenceType(req.Type),
		Excerpt:   req.Excerpt,
		CreatedBy: req.CreatedBy,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.queries.CreateEvidence(ev); err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "create evidence failed: %v", err))
		return
	}
	writeJSON(w, http.StatusCreated, ev)
}

// handleExportReportMD generates a Markdown report for a project.
func (s *Server) handleExportReportMD(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")

	project, err := s.queries.GetProject(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "get project failed: %v", err))
		return
	}
	if project == nil {
		writeError(w, http.StatusNotFound, errors.New(errors.ErrNotFound, "project not found"))
		return
	}

	data, err := report.Aggregate(r.Context(), s.queries, project)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "report aggregation failed: %v", err))
		return
	}

	md := report.GenerateMarkdown(data)
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=report_%s.md", projectID))
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(md))
}

// handleExportReportJSON generates a JSON export for a project.
func (s *Server) handleExportReportJSON(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")

	project, err := s.queries.GetProject(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "get project failed: %v", err))
		return
	}
	if project == nil {
		writeError(w, http.StatusNotFound, errors.New(errors.ErrNotFound, "project not found"))
		return
	}

	data, err := report.Aggregate(r.Context(), s.queries, project)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "report aggregation failed: %v", err))
		return
	}

	jsonData, err := report.GenerateJSON(data)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "json generation failed: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=report_%s.json", projectID))
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)
}
