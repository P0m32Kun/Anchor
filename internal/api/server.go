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
	"github.com/P0m32Kun/Anchor/internal/dictionary"
	"github.com/P0m32Kun/Anchor/internal/health"
	"github.com/P0m32Kun/Anchor/internal/httpxfp"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/nuclei/custom"
	"github.com/P0m32Kun/Anchor/internal/scope"
	"github.com/P0m32Kun/Anchor/internal/service"
	"github.com/P0m32Kun/Anchor/internal/worker"
)

// Server holds API dependencies.
type Server struct {
	queries         *db.Queries
	rawDB           *sql.DB
	scopeEng        *scope.Engine
	worker          *worker.Runner
	health          *health.Checker
	dataDir         string
	sseClients      map[string]map[string]chan []byte
	taskQueue       map[string]chan *models.ScanTask
	taskResults     map[string]chan map[string]interface{}
	mu              sync.Mutex
	apiToken        string
	projectSvc      service.ProjectService
	targetSvc       service.TargetService
	findingSvc      service.FindingService
	nucleiCustomMgr *custom.Manager
	dictMgr         *dictionary.Manager
	httpxFpMgr      *httpxfp.Manager
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
	s.nucleiCustomMgr = custom.NewManager(queries, rawDB, dataDir, custom.ExecCloner{})
	if err := s.nucleiCustomMgr.EnsureLayout(); err != nil {
		log.Printf("[server] nuclei custom layout init: %v (continuing)", err)
	}
	s.dictMgr = dictionary.NewManager(queries, dataDir)
	if err := s.dictMgr.EnsureLayout(); err != nil {
		log.Printf("[server] dictionary layout init: %v (continuing)", err)
	}
	builtinDictRoot := os.Getenv("ANCHOR_BUILTIN_DICT_ROOT")
	if builtinDictRoot == "" {
		builtinDictRoot = "/opt/dict"
	}
	if err := s.dictMgr.SeedBuiltin(builtinDictRoot); err != nil {
		log.Printf("[server] dictionary builtin seed: %v (continuing)", err)
	}
	s.httpxFpMgr = httpxfp.NewManager(queries, dataDir)
	if err := s.httpxFpMgr.EnsureLayout(); err != nil {
		log.Printf("[server] httpx fingerprint layout init: %v (continuing)", err)
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
	mux.Handle("GET /dashboard/stats", auth(http.HandlerFunc(s.handleGetDashboardStats)))
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
	mux.Handle("GET /projects/{id}/service-ports", auth(http.HandlerFunc(s.handleListServicePorts)))
	mux.Handle("GET /assets/{id}/ports", auth(http.HandlerFunc(s.handleListPorts)))
	mux.Handle("GET /assets/{id}/services", auth(http.HandlerFunc(s.handleListServices)))
	mux.Handle("POST /projects/{id}/workflows/web-screening", auth(http.HandlerFunc(s.handleStartWebScreening)))
	mux.Handle("POST /projects/{id}/pipeline/run", auth(http.HandlerFunc(s.handleRunPipeline)))
	mux.Handle("GET /projects/{id}/pipeline/runs", auth(http.HandlerFunc(s.handleListPipelineRuns)))
	mux.Handle("GET /projects/{id}/pipeline/runs/{runId}", auth(http.HandlerFunc(s.handleGetPipelineRun)))
	mux.Handle("GET /projects/{id}/pipeline/runs/{runId}/stages", auth(http.HandlerFunc(s.handleGetPipelineRunStages)))
	mux.Handle("POST /projects/{id}/pipeline/runs/{runId}/cancel", auth(http.HandlerFunc(s.handleCancelPipelineRun)))
	mux.Handle("GET /projects/{id}/pipeline/config", auth(http.HandlerFunc(s.handleGetPipelineConfig)))
	mux.Handle("POST /projects/{id}/pipeline/config", auth(http.HandlerFunc(s.handleUpdatePipelineConfig)))
	// Unified scan
	mux.Handle("POST /projects/{id}/scan", auth(http.HandlerFunc(s.handleCreateScan)))
	mux.Handle("GET /projects/{id}/scan/runs", auth(http.HandlerFunc(s.handleListScanRuns)))
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
	// Report engine
	mux.Handle("POST /runs/{runId}/report", auth(http.HandlerFunc(s.handleCreateReport)))
	mux.Handle("GET /reports/{reportId}", auth(http.HandlerFunc(s.handleGetReport)))
	mux.Handle("GET /reports/{reportId}/download", auth(http.HandlerFunc(s.handleDownloadReport)))
	mux.Handle("DELETE /reports/{reportId}", auth(http.HandlerFunc(s.handleDeleteReport)))
	mux.Handle("GET /reports", auth(http.HandlerFunc(s.handleListReports)))
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
	// Engine search
	mux.Handle("GET /engines/credentials", auth(http.HandlerFunc(s.handleListEngineCredentials)))
	mux.Handle("POST /engines/credentials", auth(http.HandlerFunc(s.handleSaveEngineCredential)))
	mux.Handle("DELETE /engines/credentials/{engine}", auth(http.HandlerFunc(s.handleDeleteEngineCredential)))
	mux.Handle("GET /engines/search", auth(http.HandlerFunc(s.handleSearchEngine)))
		mux.Handle("GET /engines/quota", auth(http.HandlerFunc(s.handleGetEngineQuota)))
	// Nuclei custom template sources
	mux.Handle("GET /nuclei/custom/sources", auth(http.HandlerFunc(s.handleListNucleiCustomSources)))
	mux.Handle("POST /nuclei/custom/sources/git", auth(http.HandlerFunc(s.handleCreateNucleiCustomGitSource)))
	mux.Handle("POST /nuclei/custom/sources/upload", auth(http.HandlerFunc(s.handleCreateNucleiCustomUploadSource)))
	mux.Handle("POST /nuclei/custom/sources/{id}/refresh", auth(http.HandlerFunc(s.handleRefreshNucleiCustomSource)))
	mux.Handle("PATCH /nuclei/custom/sources/{id}", auth(http.HandlerFunc(s.handlePatchNucleiCustomSource)))
	mux.Handle("DELETE /nuclei/custom/sources/{id}", auth(http.HandlerFunc(s.handleDeleteNucleiCustomSource)))
	mux.Handle("GET /nuclei/custom/sources/{id}/files", auth(http.HandlerFunc(s.handleListNucleiCustomFiles)))
	mux.Handle("GET /nuclei/custom/sources/{id}/files/{path...}", auth(http.HandlerFunc(s.handleReadNucleiCustomFile)))
	mux.Handle("PUT /nuclei/custom/sources/{id}/files/{path...}", auth(http.HandlerFunc(s.handleWriteNucleiCustomFile)))
	mux.Handle("DELETE /nuclei/custom/sources/{id}/files/{path...}", auth(http.HandlerFunc(s.handleDeleteNucleiCustomFile)))
	// Phase 2: Validation & Publishing
	mux.Handle("POST /nuclei/custom/sources/{id}/validate", auth(http.HandlerFunc(s.handleValidateNucleiCustomSource)))
	mux.Handle("POST /nuclei/custom/validate", auth(http.HandlerFunc(s.handleValidateAllNucleiCustom)))
	mux.Handle("POST /nuclei/custom/publish", auth(http.HandlerFunc(s.handlePublishNucleiCustom)))
	mux.Handle("GET /nuclei/custom/manifest", auth(http.HandlerFunc(s.handleGetNucleiCustomManifest)))
	mux.Handle("GET /nuclei/custom/bundles/{version}", auth(http.HandlerFunc(s.handleDownloadNucleiCustomBundle)))
	// Dictionaries
	mux.Handle("GET /dictionaries", auth(http.HandlerFunc(s.handleListDictionaries)))
	mux.Handle("POST /dictionaries", auth(http.HandlerFunc(s.handleCreateDictionary)))
	mux.Handle("GET /dictionaries/{id}", auth(http.HandlerFunc(s.handleGetDictionary)))
	mux.Handle("PATCH /dictionaries/{id}", auth(http.HandlerFunc(s.handlePatchDictionary)))
	mux.Handle("DELETE /dictionaries/{id}", auth(http.HandlerFunc(s.handleDeleteDictionary)))
	mux.Handle("GET /dictionaries/{id}/content", auth(http.HandlerFunc(s.handleReadDictionaryContent)))
	mux.Handle("PUT /dictionaries/{id}/content", auth(http.HandlerFunc(s.handleWriteDictionaryContent)))
	// HTTPX fingerprints
	mux.Handle("GET /httpx/fingerprints", auth(http.HandlerFunc(s.handleListHttpxFingerprints)))
	mux.Handle("POST /httpx/fingerprints", auth(http.HandlerFunc(s.handleCreateHttpxFingerprint)))
	mux.Handle("GET /httpx/fingerprints/{id}", auth(http.HandlerFunc(s.handleGetHttpxFingerprint)))
	mux.Handle("PATCH /httpx/fingerprints/{id}", auth(http.HandlerFunc(s.handlePatchHttpxFingerprint)))
	mux.Handle("DELETE /httpx/fingerprints/{id}", auth(http.HandlerFunc(s.handleDeleteHttpxFingerprint)))
	mux.Handle("GET /httpx/fingerprints/{id}/content", auth(http.HandlerFunc(s.handleReadHttpxFingerprintContent)))
	mux.Handle("PUT /httpx/fingerprints/{id}/content", auth(http.HandlerFunc(s.handleWriteHttpxFingerprintContent)))
	// Slow scan tasks
	mux.Handle("GET /projects/{id}/slow-scans", auth(http.HandlerFunc(s.handleListSlowScans)))
	mux.Handle("GET /slow-scans/{id}", auth(http.HandlerFunc(s.handleGetSlowScan)))
	mux.Handle("POST /slow-scans/{id}/cancel", auth(http.HandlerFunc(s.handleCancelSlowScan)))
	// Finding templates (vulnerability knowledge base)
	mux.Handle("GET /finding-templates", auth(http.HandlerFunc(s.handleListFindingTemplates)))
	mux.Handle("POST /finding-templates", auth(http.HandlerFunc(s.handleCreateFindingTemplate)))
	mux.Handle("GET /finding-templates/export", auth(http.HandlerFunc(s.handleExportFindingTemplates)))
	mux.Handle("GET /finding-templates/{id}", auth(http.HandlerFunc(s.handleGetFindingTemplate)))
	mux.Handle("PATCH /finding-templates/{id}", auth(http.HandlerFunc(s.handlePatchFindingTemplate)))
	mux.Handle("DELETE /finding-templates/{id}", auth(http.HandlerFunc(s.handleDeleteFindingTemplate)))
	mux.Handle("POST /finding-templates/{id}/accept-upstream", auth(http.HandlerFunc(s.handleAcceptFindingTemplateUpstream)))
}
