package api

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
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
	queries     *db.Queries
	rawDB       *sql.DB
	scopeEng    *scope.Engine
	worker      *worker.Runner
	health      *health.Checker
	dataDir     string
	sseClients  map[string]map[string]chan []byte
	taskQueue   map[string]chan *models.ScanTask
	taskResults map[string]chan map[string]interface{}
	mu          sync.Mutex
	apiToken    string
}

func generateAPIToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// Fallback: use timestamp + random for safety
		return fmt.Sprintf("%x%x", time.Now().UnixNano(), b)
	}
	return hex.EncodeToString(b)
}

func NewServer(queries *db.Queries, rawDB *sql.DB, dataDir string) *Server {
	scopeEng := scope.NewEngine(queries)

	token := os.Getenv("ANCHOR_API_TOKEN")
	if token == "" {
		token = generateAPIToken()
	}
	log.Printf("[server] API Token: %s  (set ANCHOR_API_TOKEN env to override)", token)

	s := &Server{
		queries:     queries,
		rawDB:       rawDB,
		scopeEng:    scopeEng,
		worker:      worker.NewRunner(queries, scopeEng, dataDir),
		health:      health.NewChecker(queries),
		dataDir:     dataDir,
		sseClients:  make(map[string]map[string]chan []byte),
		taskQueue:   make(map[string]chan *models.ScanTask),
		taskResults: make(map[string]chan map[string]interface{}),
		apiToken:    token,
	}
	// Mark all existing workers as offline on startup (they'll re-register if active)
	s.markAllWorkersOffline()
	// Clean up stale workers every 60s
	go s.cleanupStaleWorkers()
	return s
}

func (s *Server) markAllWorkersOffline() {
	workers, err := s.queries.ListWorkerNodes()
	if err != nil {
		log.Printf("[server] mark all workers offline: list failed: %v", err)
		return
	}
	now := time.Now().UTC()
	for _, w := range workers {
		if w.Status == models.WorkerStatusOffline {
			continue
		}
		if err := s.queries.UpdateWorkerNodeStatus(w.ID, models.WorkerStatusOffline, now); err != nil {
			log.Printf("[server] mark all workers offline: update failed for %s: %v", w.ID, err)
			continue
		}
		s.mu.Lock()
		if ch, ok := s.taskQueue[w.ID]; ok {
			close(ch)
		}
		delete(s.taskQueue, w.ID)
		delete(s.taskResults, w.ID)
		s.mu.Unlock()
		log.Printf("[server] worker %s marked offline on startup", w.ID)
	}
}

func (s *Server) cleanupStaleWorkers() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		workers, err := s.queries.ListWorkerNodes()
		if err != nil {
			log.Printf("[server] cleanup stale workers: list failed: %v", err)
			continue
		}
		now := time.Now().UTC()
		for _, w := range workers {
			if w.Status == models.WorkerStatusOffline {
				// Auto-delete offline workers that have been stale for 7 days
				if w.LastSeen != nil && now.Sub(*w.LastSeen) > 168*time.Hour {
					s.mu.Lock()
					if ch, ok := s.taskQueue[w.ID]; ok {
						close(ch)
					}
					delete(s.taskQueue, w.ID)
					delete(s.taskResults, w.ID)
					s.mu.Unlock()
					if err := s.queries.DeleteWorkerNode(w.ID); err != nil {
						log.Printf("[server] cleanup stale workers: delete worker %s failed: %v", w.ID, err)
						continue
					}
					log.Printf("[server] worker %s deleted (offline for %v)", w.ID, now.Sub(*w.LastSeen))
				}
				continue
			}
			if w.RevokedAt != nil {
				continue
			}
			if w.LastSeen == nil {
				continue
			}
			if now.Sub(*w.LastSeen) > 120*time.Second {
				if err := s.queries.UpdateWorkerNodeStatus(w.ID, models.WorkerStatusOffline, now); err != nil {
					log.Printf("[server] cleanup stale workers: update status failed for %s: %v", w.ID, err)
					continue
				}
				s.mu.Lock()
				if ch, ok := s.taskQueue[w.ID]; ok {
					close(ch)
				}
				delete(s.taskQueue, w.ID)
				delete(s.taskResults, w.ID)
				s.mu.Unlock()
				log.Printf("[server] worker %s marked offline (last seen %v ago)", w.ID, now.Sub(*w.LastSeen))
			}
		}
	}
}

func (s *Server) Register(mux *http.ServeMux) {
	// Public routes (no token required)
	mux.HandleFunc("GET /health", s.handleHealth)

	// Protected routes — wrapped with TokenAuthMiddleware
	auth := s.TokenAuthMiddleware
	mux.Handle("POST /projects", auth(http.HandlerFunc(s.handleCreateProject)))
	mux.Handle("GET /projects", auth(http.HandlerFunc(s.handleListProjects)))
	mux.Handle("GET /projects/{id}", auth(http.HandlerFunc(s.handleGetProject)))
	mux.Handle("DELETE /projects/{id}", auth(http.HandlerFunc(s.handleDeleteProject)))
	mux.Handle("POST /projects/{id}/targets", auth(http.HandlerFunc(s.handleCreateTarget)))
	mux.Handle("POST /projects/{id}/targets/import", auth(http.HandlerFunc(s.handleImportTargets)))
	mux.Handle("GET /projects/{id}/targets", auth(http.HandlerFunc(s.handleListTargets)))
	mux.Handle("POST /projects/{id}/runs", auth(http.HandlerFunc(s.handleCreateRun)))
	mux.Handle("GET /projects/{id}/runs", auth(http.HandlerFunc(s.handleListRuns)))
	mux.Handle("GET /runs/{id}", auth(http.HandlerFunc(s.handleGetRun)))
	mux.Handle("GET /runs/{id}/tasks", auth(http.HandlerFunc(s.handleGetRunTasks)))
	mux.Handle("POST /runs/{id}/cancel", auth(http.HandlerFunc(s.handleCancelRun)))
	mux.Handle("POST /projects/{id}/workflows/asset-discovery", auth(http.HandlerFunc(s.handleStartAssetDiscovery)))
	mux.Handle("GET /projects/{id}/assets", auth(http.HandlerFunc(s.handleListAssetsFiltered)))
	mux.Handle("GET /projects/{id}/web-endpoints", auth(http.HandlerFunc(s.handleListWebEndpointsByProject)))
	mux.Handle("GET /assets/{id}/ports", auth(http.HandlerFunc(s.handleListPorts)))
	mux.Handle("GET /assets/{id}/services", auth(http.HandlerFunc(s.handleListServices)))
	mux.Handle("POST /projects/{id}/workflows/web-screening", auth(http.HandlerFunc(s.handleStartWebScreening)))
	mux.Handle("GET /projects/{id}/findings", auth(http.HandlerFunc(s.handleListFindings)))
	mux.Handle("GET /findings/{id}", auth(http.HandlerFunc(s.handleGetFinding)))
	mux.Handle("PATCH /findings/{id}/status", auth(http.HandlerFunc(s.handlePatchFindingStatus)))
	mux.Handle("POST /findings/{id}/evidence", auth(http.HandlerFunc(s.handleAddEvidence)))
	mux.Handle("POST /findings/{id}/retest", auth(http.HandlerFunc(s.handleRetestFinding)))
	mux.Handle("GET /findings/{id}/retests", auth(http.HandlerFunc(s.handleListRetests)))
	mux.Handle("PATCH /findings/batch-status", auth(http.HandlerFunc(s.handleBatchUpdateFindingStatus)))
	mux.Handle("GET /findings/{id}/curl", auth(http.HandlerFunc(s.handleGetFindingCurl)))
	mux.Handle("POST /scope-rules", auth(http.HandlerFunc(s.handleCreateScopeRule)))
	mux.Handle("GET /scope-rules", auth(http.HandlerFunc(s.handleListScopeRules)))
	mux.Handle("POST /projects/{id}/scope-rules/batch", auth(http.HandlerFunc(s.handleBatchCreateScopeRules)))
	mux.Handle("POST /scan-plans", auth(http.HandlerFunc(s.handleCreateScanPlan)))
	mux.Handle("POST /scan-plans/{id}/approve", auth(http.HandlerFunc(s.handleApprovePlan)))
	mux.Handle("POST /scan-plans/dry-run", auth(http.HandlerFunc(s.handleDryRun)))
	mux.Handle("GET /scan-tasks/{id}", auth(http.HandlerFunc(s.handleGetTask)))
	mux.Handle("POST /scan-tasks/{id}/cancel", auth(http.HandlerFunc(s.handleCancelTask)))
	mux.Handle("POST /tasks/run", auth(http.HandlerFunc(s.handleRunTask)))
	mux.Handle("GET /tasks/{id}/artifacts", auth(http.HandlerFunc(s.handleListArtifacts)))
	mux.Handle("GET /health/tools", auth(http.HandlerFunc(s.handleToolHealth)))
	mux.Handle("POST /health/check", auth(http.HandlerFunc(s.handleHealthCheck)))
	mux.Handle("GET /projects/{id}/events", auth(http.HandlerFunc(s.handleProjectSSE)))
	mux.Handle("GET /projects/{id}/reports/export.md", auth(http.HandlerFunc(s.handleExportReportMD)))
	mux.Handle("GET /projects/{id}/reports/export.json", auth(http.HandlerFunc(s.handleExportReportJSON)))
	mux.Handle("POST /projects/{id}/archive", auth(http.HandlerFunc(s.handleCreateArchive)))
	mux.Handle("GET /projects/{id}/archive/download", auth(http.HandlerFunc(s.handleDownloadArchive)))
	mux.Handle("GET /tool-templates", auth(http.HandlerFunc(s.handleListToolTemplates)))
	mux.Handle("GET /tool-templates/{id}", auth(http.HandlerFunc(s.handleGetToolTemplate)))
	mux.Handle("GET /workers", auth(http.HandlerFunc(s.handleListWorkers)))
	// Worker APIs (also protected by global token)
	mux.Handle("POST /workers/register", auth(http.HandlerFunc(s.handleRegisterWorker)))
	mux.Handle("POST /workers/{id}/heartbeat", auth(http.HandlerFunc(s.handleWorkerHeartbeat)))
	mux.Handle("GET /workers/{id}/tasks/poll", auth(http.HandlerFunc(s.handlePollTasks)))
	mux.Handle("POST /tasks/{id}/result", auth(http.HandlerFunc(s.handleTaskResult)))
	mux.Handle("POST /workers/{id}/revoke", auth(http.HandlerFunc(s.handleRevokeWorker)))
	mux.Handle("DELETE /workers/{id}", auth(http.HandlerFunc(s.handleDeleteWorker)))
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
		if origin := r.Header.Get("Origin"); origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "86400")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// TokenAuthMiddleware verifies the Bearer token for all protected routes.
func (s *Server) TokenAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for health check and CORS preflight
		if r.Method == "OPTIONS" || r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}

		auth := r.Header.Get("Authorization")
		const prefix = "Bearer "
		if !strings.HasPrefix(auth, prefix) || auth[len(prefix):] != s.apiToken {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]string{"message": "Unauthorized: invalid or missing token"},
			})
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

	// Validate name.
	if strings.TrimSpace(req.Name) == "" {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "name is required"))
		return
	}
	if len(req.Name) > 200 {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "name too long (max 200)"))
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

func (s *Server) handleDeleteProject(w http.ResponseWriter, r *http.Request) {
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
	if err := s.queries.DeleteProject(id); err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "delete project failed: %v", err))
		return
	}

	_ = s.queries.CreateAuditLog(&models.AuditLog{
		ID:           util.GenerateID(),
		ProjectID:    id,
		Actor:        "user",
		Action:       "project.delete",
		ResourceType: "project",
		ResourceID:   id,
		Summary:      fmt.Sprintf("Deleted project %s", p.Name),
		CreatedAt:    time.Now().UTC(),
	})

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
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

	// Auto-detect target type when frontend sends "auto"
	resolvedType := req.Type
	if resolvedType == "auto" {
		resolvedType = string(scope.DetectType(req.Value))
	}

	// 检查项目是否已有 scope 规则
	rules, err := s.queries.ListScopeRulesByProject(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "list scope rules failed: %v", err))
		return
	}

	// 无 scope 规则 → 提示用户确认是否自动加入授权范围
	if len(rules) == 0 {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"needs_scope_confirmation": true,
			"message":                  "当前项目未设置授权范围，是否将此目标自动加入授权范围？",
			"suggested_rule": map[string]string{
				"action": "include",
				"type":   resolvedType,
				"value":  req.Value,
			},
		})
		return
	}

	t := &models.Target{
		ID:        util.GenerateID(),
		ProjectID: projectID,
		Type:      models.TargetType(resolvedType),
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

	// Check if project has any scope rules.
	rules, err := s.queries.ListScopeRulesByProject(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "list scope rules failed: %v", err))
		return
	}

	// 无 scope 规则 → 提示用户确认是否自动加入授权范围
	if len(rules) == 0 {
		var suggested []map[string]string
		seenRule := make(map[string]bool)
		for _, pt := range uniqueTargets {
			key := string(pt.Type) + ":" + pt.Value
			if seenRule[key] {
				continue
			}
			seenRule[key] = true
			suggested = append(suggested, map[string]string{
				"action": "include",
				"type":   string(pt.Type),
				"value":  pt.Value,
			})
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"needs_scope_confirmation": true,
			"message":                  "当前项目未设置授权范围，是否将导入的目标自动加入授权范围？",
			"suggested_rules":          suggested,
		})
		return
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

// handleBatchCreateScopeRules 批量创建 scope include 规则（用于用户确认自动加入授权范围）
func (s *Server) handleBatchCreateScopeRules(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")

	var req struct {
		Rules []struct {
			Action string `json:"action"`
			Type   string `json:"type"`
			Value  string `json:"value"`
			Reason string `json:"reason"`
		} `json:"rules"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "invalid request body").WithDetail(err.Error()))
		return
	}

	now := time.Now().UTC()
	var created []*models.ScopeRule
	for _, ruleReq := range req.Rules {
		if ruleReq.Action == "" {
			ruleReq.Action = "include"
		}
		sr := &models.ScopeRule{
			ID:        util.GenerateID(),
			ProjectID: projectID,
			Action:    models.ScopeAction(ruleReq.Action),
			Type:      models.TargetType(ruleReq.Type),
			Value:     ruleReq.Value,
			Reason:    ruleReq.Reason,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := s.queries.CreateScopeRule(sr); err != nil {
			writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "create scope rule failed: %v", err))
			return
		}
		created = append(created, sr)
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"created": len(created),
		"rules":   created,
	})
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
		s.broadcastProjectSSE(task.ProjectID, map[string]interface{}{
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

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	if err := s.health.CheckAll(r.Context(), s.dataDir); err != nil {
		log.Printf("health check error: %v", err)
	}
	s.handleToolHealth(w, r)
}

// --- SSE ---

const maxSSEClientsPerProject = 100

func (s *Server) handleProjectSSE(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	if projectID == "" {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "missing project id"))
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	clientID := util.GenerateID()
	ch := make(chan []byte, 10)

	s.mu.Lock()
	if len(s.sseClients[projectID]) >= maxSSEClientsPerProject {
		s.mu.Unlock()
		writeError(w, http.StatusServiceUnavailable, errors.New(errors.ErrInternal, "too many SSE connections"))
		return
	}
	if s.sseClients[projectID] == nil {
		s.sseClients[projectID] = make(map[string]chan []byte)
	}
	s.sseClients[projectID][clientID] = ch
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.sseClients[projectID], clientID)
		if len(s.sseClients[projectID]) == 0 {
			delete(s.sseClients, projectID)
		}
		s.mu.Unlock()
	}()

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, errors.New(errors.ErrInternal, "SSE not supported"))
		return
	}

	// Send initial connection event.
	fmt.Fprintf(w, "data: %s\n\n", `{"event":"connected"}`)
	flusher.Flush()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case msg := <-ch:
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		case <-ticker.C:
			fmt.Fprintf(w, "data: %s\n\n", `{"event":"ping"}`)
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func (s *Server) broadcastProjectSSE(projectID string, data map[string]interface{}) {
	b, _ := json.Marshal(data)
	s.mu.Lock()
	clients, ok := s.sseClients[projectID]
	s.mu.Unlock()
	if !ok {
		return
	}
	for _, ch := range clients {
		select {
		case ch <- b:
		default:
		}
	}
}

// --- Helpers ---

func (s *Server) handleListWorkers(w http.ResponseWriter, r *http.Request) {
	workers := []map[string]interface{}{}

	dbWorkers, err := s.queries.ListWorkerNodes()
	if err == nil {
		for _, w := range dbWorkers {
			workers = append(workers, map[string]interface{}{
				"id":       w.ID,
				"name":     w.Name,
				"mode":     w.Mode,
				"status":   w.Status,
				"endpoint": w.Endpoint,
			})
		}
	}

	writeJSON(w, http.StatusOK, workers)
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
		s.broadcastProjectSSE(projectID, map[string]interface{}{
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
		s.broadcastProjectSSE(projectID, map[string]interface{}{
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
