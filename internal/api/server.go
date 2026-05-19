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
//
// 字段按消费面分四类(查询请参考 internal/api/README.md):
//   - 广字段(被多数 handler 使用,改动会牵连大面积):queries / dataDir
//   - 业务子系统字段(几个 handler 共用):scopeEng / worker
//   - 任务分发与 SSE 子系统(run/worker/report/sse 四个文件共享):
//     sseClients / taskQueue / taskResults / mu
//   - 单一消费者字段(只服务一个 handler 文件,改动 blast radius 极小):
//     rawDB / health / apiToken / projectSvc / targetSvc / findingSvc /
//     nucleiCustomMgr / dictMgr / httpxFpMgr
//
// 改任何字段前,先在 README.md 找消费者列表,再决定动手范围。
type Server struct {
	// queries: 持久化层。绝大多数 handler 都直接使用,改签名前慎重。
	// 消费者: archive / asset / dashboard / engine / finding_template /
	//   pipeline / report / retest / run / scope / slow_scan / task /
	//   worker / workdir_cleanup / workflow / handlers
	queries *db.Queries

	// rawDB: 原始 sql.DB,仅在需要事务/低层 API 时使用。
	// 消费者: retest_handlers.go
	rawDB *sql.DB

	// scopeEng: 范围(scope)校验引擎。
	// 消费者: pipeline / run / scope / workflow
	scopeEng *scope.Engine

	// worker: 本地 worker.Runner(注意区别于 worker 包内的 RemoteAgent)。
	// 消费者: pipeline / run / slow_scan / task / workflow
	worker *worker.Runner

	// health: 工具可用性探测器。
	// 消费者: handlers.go(/health/tools, /health/check)
	health *health.Checker

	// dataDir: 数据/产物目录根路径。
	// 消费者: archive / handlers / pipeline / report / run /
	//   worker / workdir_cleanup / workflow
	dataDir string

	// 任务分发与 SSE 子系统 ↓↓↓ 这 4 个字段绑在一起,要改一起改。
	// 受 mu 保护,跨 goroutine 读写。

	// sseClients: project_id -> client_id -> event chan。
	// 消费者: report_handlers.go / sse.go
	sseClients map[string]map[string]chan []byte

	// taskQueue: worker_id -> 待派发任务 chan。
	// 消费者: run_handlers.go / worker_handlers.go
	taskQueue map[string]chan *models.ScanTask

	// taskResults: worker_id -> 结果回报 chan。
	// 消费者: worker_handlers.go
	taskResults map[string]chan map[string]interface{}

	// mu: 保护 sseClients / taskQueue / taskResults 的并发访问。
	// 消费者: run / report / sse / worker
	mu sync.Mutex

	// 任务分发与 SSE 子系统 ↑↑↑

	// apiToken: 启动时生成或从 ANCHOR_API_TOKEN 读取。
	// 消费者: middleware.go(TokenAuthMiddleware)
	apiToken string

	// 业务服务层(每个只服务对应一个 handler 文件)↓

	// projectSvc 消费者: project_handlers.go
	projectSvc service.ProjectService
	// targetSvc 消费者: target_handlers.go
	targetSvc service.TargetService
	// findingSvc 消费者: finding_handlers.go
	findingSvc service.FindingService
	// nucleiCustomMgr 消费者: nuclei_custom_handlers.go
	nucleiCustomMgr *custom.Manager
	// dictMgr 消费者: dictionary_handlers.go
	dictMgr *dictionary.Manager
	// httpxFpMgr 消费者: httpx_fingerprint_handlers.go
	httpxFpMgr *httpxfp.Manager
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
	go s.startWorkdirCleanup()
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
	mux.Handle("DELETE /projects/{id}/targets/{targetId}", auth(http.HandlerFunc(s.handleDeleteTarget)))
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
	mux.Handle("DELETE /scope-rules/{id}", auth(http.HandlerFunc(s.handleDeleteScopeRule)))
	mux.Handle("POST /scope-rules/parse", auth(http.HandlerFunc(s.handleParseScopeValue)))
	mux.Handle("POST /projects/{id}/scope-rules/batch", auth(http.HandlerFunc(s.handleBatchCreateScopeRules)))
	mux.Handle("POST /scan-plans", auth(http.HandlerFunc(s.handleCreateScanPlan)))
	mux.Handle("POST /scan-plans/{id}/approve", auth(http.HandlerFunc(s.handleApprovePlan)))
	mux.Handle("POST /scan-plans/dry-run", auth(http.HandlerFunc(s.handleDryRun)))
	mux.Handle("GET /scan-tasks/{id}", auth(http.HandlerFunc(s.handleGetTask)))
	mux.Handle("POST /scan-tasks/{id}/cancel", auth(http.HandlerFunc(s.handleCancelTask)))
	mux.Handle("POST /tasks/run", auth(http.HandlerFunc(s.handleRunTask)))
	mux.Handle("GET /tasks/{id}/artifacts", auth(http.HandlerFunc(s.handleListArtifacts)))
	mux.Handle("GET /artifacts/content", auth(http.HandlerFunc(s.handleGetArtifactContent)))
	mux.Handle("GET /health/tools", auth(http.HandlerFunc(s.handleToolHealth)))
	mux.Handle("POST /health/check", auth(http.HandlerFunc(s.handleHealthCheck)))
	mux.Handle("GET /projects/{id}/events", auth(http.HandlerFunc(s.handleProjectSSE)))
	mux.Handle("GET /projects/{id}/reports/export.md", auth(http.HandlerFunc(s.handleExportReportMD)))
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
		mux.Handle("GET /nuclei/custom/sources/{id}/bundle", auth(http.HandlerFunc(s.handleDownloadNucleiCustomSourceBundle)))
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
