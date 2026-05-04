package api

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/health"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/scope"
	"github.com/P0m32Kun/Anchor/internal/service"
	"github.com/P0m32Kun/Anchor/internal/worker"
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
	projectSvc  service.ProjectService
	targetSvc   service.TargetService
	findingSvc  service.FindingService
}

func generateAPIToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
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
		projectSvc:  service.NewProjectService(queries),
		targetSvc:   service.NewTargetService(queries, rawDB, scopeEng),
		findingSvc:  service.NewFindingService(queries),
	}
	s.markAllWorkersOffline()
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
	mux.HandleFunc("GET /health", s.handleHealth)

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
	mux.Handle("POST /projects/{id}/pipeline/run", auth(http.HandlerFunc(s.handleRunPipeline)))
	mux.Handle("GET /projects/{id}/pipeline/runs", auth(http.HandlerFunc(s.handleListPipelineRuns)))
	mux.Handle("GET /projects/{id}/pipeline/runs/{runId}", auth(http.HandlerFunc(s.handleGetPipelineRun)))
	mux.Handle("GET /projects/{id}/pipeline/config", auth(http.HandlerFunc(s.handleGetPipelineConfig)))
	mux.Handle("POST /projects/{id}/pipeline/config", auth(http.HandlerFunc(s.handleUpdatePipelineConfig)))
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
	mux.Handle("POST /workers/register", auth(http.HandlerFunc(s.handleRegisterWorker)))
	mux.Handle("POST /workers/{id}/heartbeat", auth(http.HandlerFunc(s.handleWorkerHeartbeat)))
	mux.Handle("GET /workers/{id}/tasks/poll", auth(http.HandlerFunc(s.handlePollTasks)))
	mux.Handle("POST /tasks/{id}/result", auth(http.HandlerFunc(s.handleTaskResult)))
	mux.Handle("POST /workers/{id}/revoke", auth(http.HandlerFunc(s.handleRevokeWorker)))
	mux.Handle("DELETE /workers/{id}", auth(http.HandlerFunc(s.handleDeleteWorker)))
}
