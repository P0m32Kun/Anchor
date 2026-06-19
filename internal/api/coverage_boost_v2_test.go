package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// ==================== workdir_cleanup.go — isTaskTerminal ====================

func TestIsTaskTerminal(t *testing.T) {
	tests := []struct {
		status models.TaskStatus
		want   bool
	}{
		{models.TaskRunning, false},
		{models.TaskCreated, false},
		{models.TaskQueued, false},
		{models.TaskCompleted, true},
		{models.TaskFailed, true},
		{models.TaskCancelled, true},
		{models.TaskPartialSuccess, true},
		{models.TaskScopeDenied, true},
		{models.TaskStatus("unknown"), false},
	}
	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			got := isTaskTerminal(tt.status)
			if got != tt.want {
				t.Errorf("isTaskTerminal(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

// ==================== task_handlers.go — parseIntParam ====================

func TestParseIntParam_Coverage(t *testing.T) {
	tests := []struct {
		name       string
		s          string
		defaultVal int
		want       int
	}{
		{"empty", "", 0, 0},
		{"valid", "42", 0, 42},
		{"default for empty", "", 10, 10},
		{"with letters", "abc", 5, 5},
		{"mixed", "12abc", 5, 5},
		{"zero", "0", 10, 0},
		{"large", "999999", 0, 999999},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseIntParam(tt.s, tt.defaultVal)
			if got != tt.want {
				t.Errorf("parseIntParam(%q, %d) = %d, want %d", tt.s, tt.defaultVal, got, tt.want)
			}
		})
	}
}

// ==================== worker_handlers.go — initTaskQueue / enqueueTask ====================

func TestInitTaskQueue_Coverage(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	server.mu.Lock()
	defer server.mu.Unlock()
	if server.taskQueue == nil {
		t.Error("taskQueue should be initialized")
	}
	if server.taskResults == nil {
		t.Error("taskResults should be initialized")
	}
}

func TestEnqueueTask_Coverage(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	workerID := "test-worker-enq"
	server.mu.Lock()
	server.taskQueue[workerID] = make(chan *models.ScanTask, 10)
	server.mu.Unlock()

	task := &models.ScanTask{
		ID:        util.GenerateID(),
		ProjectID: "p1",
		Tool:      "nuclei",
	}

	server.enqueueTask(workerID, task)

	server.mu.Lock()
	ch := server.taskQueue[workerID]
	server.mu.Unlock()

	select {
	case received := <-ch:
		if received.ID != task.ID {
			t.Errorf("task ID = %s, want %s", received.ID, task.ID)
		}
	default:
		t.Error("expected task in queue")
	}
}

func TestEnqueueTask_NonexistentWorker_Coverage(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	task := &models.ScanTask{ID: util.GenerateID()}
	server.enqueueTask("nonexistent", task)
}

func TestEnqueueTask_FullChannel_Coverage(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	workerID := "test-worker-full-cov"
	server.mu.Lock()
	server.taskQueue[workerID] = make(chan *models.ScanTask, 1)
	server.mu.Unlock()

	task1 := &models.ScanTask{ID: "task-fc-1"}
	task2 := &models.ScanTask{ID: "task-fc-2"}

	server.enqueueTask(workerID, task1)
	server.enqueueTask(workerID, task2)

	server.mu.Lock()
	ch := server.taskQueue[workerID]
	server.mu.Unlock()

	count := 0
	for {
		select {
		case <-ch:
			count++
		default:
			goto done
		}
	}
done:
	if count != 1 {
		t.Errorf("expected 1 task in queue, got %d", count)
	}
}

// ==================== archive_handlers.go — handleCreateArchive ====================

func TestHandleCreateArchive_ProjectNotFound(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/projects/nonexistent/archive", nil)
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()

	server.handleCreateArchive(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestHandleCreateArchive_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)

	req := httptest.NewRequest(http.MethodPost, "/projects/"+p.ID+"/archive", nil)
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleCreateArchive(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", resp.StatusCode, http.StatusCreated, w.Body.String())
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result["archive_name"] == "" {
		t.Error("expected non-empty archive_name")
	}
}

// ==================== pipeline_handlers.go — handleUpdatePipelineConfig ====================

func TestHandleUpdatePipelineConfig_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)

	body := `{"enable_subfinder":true,"enable_naabu":false,"nuclei_concurrency":5}`
	req := httptest.NewRequest(http.MethodPut, "/projects/"+p.ID+"/pipeline/config", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleUpdatePipelineConfig(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", resp.StatusCode, http.StatusOK, w.Body.String())
	}
}

func TestHandleUpdatePipelineConfig_MissingProjectID(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	body := `{}`
	req := httptest.NewRequest(http.MethodPut, "/projects//pipeline/config", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "")
	w := httptest.NewRecorder()

	server.handleUpdatePipelineConfig(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

// ==================== scope_handlers.go — handleDeleteScopeRule ====================

func TestHandleDeleteScopeRule_Coverage(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)

	rule := &models.ScopeRule{
		ID:        util.GenerateID(),
		ProjectID: p.ID,
		Action:    models.ScopeActionInclude,
		Type:      models.TargetTypeDomain,
		Value:     "example.com",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := queries.CreateScopeRule(rule); err != nil {
		t.Fatalf("create scope rule: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/scope-rules/"+rule.ID, nil)
	req.SetPathValue("id", rule.ID)
	w := httptest.NewRecorder()

	server.handleDeleteScopeRule(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}


// ==================== notification_handlers.go — handleDeleteNotificationChannel ====================

func TestHandleDeleteNotificationChannel_MissingProjectID_Coverage(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodDelete, "/projects//notifications/c1", nil)
	req.SetPathValue("id", "")
	req.SetPathValue("channelId", "c1")
	w := httptest.NewRecorder()

	server.handleDeleteNotificationChannel(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleDeleteNotificationChannel_MissingChannelID_Coverage(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodDelete, "/projects/p1/notifications/", nil)
	req.SetPathValue("id", "p1")
	req.SetPathValue("channelId", "")
	w := httptest.NewRecorder()

	server.handleDeleteNotificationChannel(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

// ==================== engine_handlers.go — handleSearchEngine / handleGetEngineQuota with credentials ====================

func TestHandleSearchEngine_FofaWithCredential(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	now := time.Now().UTC()
	if err := queries.SaveEngineCredential(&models.EngineCredential{
		ID: util.GenerateID(), Engine: "fofa", APIKey: "test-key-1234",
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("save credential: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/engines/search?engine=fofa&query=test&page=1&size=10", nil)
	w := httptest.NewRecorder()

	server.handleSearchEngine(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusBadRequest {
		t.Error("should not be 400 when credential exists")
	}
}

func TestHandleSearchEngine_HunterWithCredential(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	now := time.Now().UTC()
	if err := queries.SaveEngineCredential(&models.EngineCredential{
		ID: util.GenerateID(), Engine: "hunter", APIKey: "test-key-1234",
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("save credential: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/engines/search?engine=hunter&query=test", nil)
	w := httptest.NewRecorder()

	server.handleSearchEngine(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusBadRequest {
		t.Error("should not be 400 when credential exists")
	}
}

func TestHandleSearchEngine_QuakeWithCredential(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	now := time.Now().UTC()
	if err := queries.SaveEngineCredential(&models.EngineCredential{
		ID: util.GenerateID(), Engine: "quake", APIKey: "test-key-1234",
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("save credential: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/engines/search?engine=quake&query=test", nil)
	w := httptest.NewRecorder()

	server.handleSearchEngine(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusBadRequest {
		t.Error("should not be 400 when credential exists")
	}
}

func TestHandleGetEngineQuota_HunterWithCredential(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	now := time.Now().UTC()
	if err := queries.SaveEngineCredential(&models.EngineCredential{
		ID: util.GenerateID(), Engine: "hunter", APIKey: "test-key-1234",
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("save credential: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/engines/quota?engine=hunter", nil)
	w := httptest.NewRecorder()

	server.handleGetEngineQuota(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusBadRequest {
		t.Error("should not be 400 when credential exists")
	}
}

func TestHandleGetEngineQuota_QuakeWithCredential(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	now := time.Now().UTC()
	if err := queries.SaveEngineCredential(&models.EngineCredential{
		ID: util.GenerateID(), Engine: "quake", APIKey: "test-key-1234",
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("save credential: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/engines/quota?engine=quake", nil)
	w := httptest.NewRecorder()

	server.handleGetEngineQuota(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusBadRequest {
		t.Error("should not be 400 when credential exists")
	}
}

// ==================== worker_handlers.go — handleWorkerHeartbeat / handlePollTasks / handleDeleteWorker ====================

func TestHandleWorkerHeartbeat_NoMetrics_Coverage(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	worker := createTestWorker(t, queries, models.WorkerStatusOnline)

	body, _ := json.Marshal(map[string]interface{}{
		"status": "busy",
	})

	req := httptest.NewRequest(http.MethodPost, "/workers/"+worker.ID+"/heartbeat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", worker.ID)
	w := httptest.NewRecorder()

	server.handleWorkerHeartbeat(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestHandlePollTasks_WithTask(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	worker := createTestWorker(t, queries, models.WorkerStatusOnline)

	server.mu.Lock()
	ch, ok := server.taskQueue[worker.ID]
	if !ok {
		ch = make(chan *models.ScanTask, 10)
		server.taskQueue[worker.ID] = ch
	}
	ch <- &models.ScanTask{
		ID:              util.GenerateID(),
		ProjectID:       "p1",
		Tool:            "nuclei",
		CommandTemplate: "nuclei -t test.yaml",
	}
	server.mu.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/workers/"+worker.ID+"/tasks/poll", nil)
	req.SetPathValue("id", worker.ID)
	w := httptest.NewRecorder()

	server.handlePollTasks(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", resp.StatusCode, http.StatusOK, w.Body.String())
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["task_id"] == nil {
		t.Error("expected task_id in response")
	}
}

func TestHandleDeleteWorker_NotFound(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodDelete, "/workers/nonexistent", nil)
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()

	server.handleDeleteWorker(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

// ==================== worker_handlers.go — handleTaskResult artifact paths ====================

func TestHandleTaskResult_ArtifactStringData(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	task := createTestTask(t, queries, p.ID, models.TaskRunning)

	body := `{
		"status": "completed",
		"artifacts": [
			{
				"type": "stdout",
				"name": "output.txt",
				"data": "plain string data not base64"
			}
		]
	}`
	req := httptest.NewRequest(http.MethodPost, "/tasks/"+task.ID+"/result", strings.NewReader(body))
	req.SetPathValue("id", task.ID)
	w := httptest.NewRecorder()

	server.handleTaskResult(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", resp.StatusCode, http.StatusOK, w.Body.String())
	}
}

func TestHandleTaskResult_ArtifactNoName(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	task := createTestTask(t, queries, p.ID, models.TaskRunning)

	body := `{
		"status": "completed",
		"artifacts": [
			{
				"type": "stdout",
				"data": "c2NhbiByZXN1bHQ="
			}
		]
	}`
	req := httptest.NewRequest(http.MethodPost, "/tasks/"+task.ID+"/result", strings.NewReader(body))
	req.SetPathValue("id", task.ID)
	w := httptest.NewRecorder()

	server.handleTaskResult(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

// ==================== exclude_handlers.go — handleResetExcludedDomains ====================

func TestHandleResetExcludedDomains_WithCustomDomains(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	createTestExcludedDomain(t, queries, "custom-reset-1.example.com", false)
	createTestExcludedDomain(t, queries, "custom-reset-2.example.com", false)

	req := httptest.NewRequest(http.MethodPost, "/excluded-domains/reset", nil)
	w := httptest.NewRecorder()

	server.handleResetExcludedDomains(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	domains, _ := queries.ListExcludedDomains()
	for _, d := range domains {
		if !d.Builtin {
			t.Errorf("custom domain %q still exists after reset", d.Domain)
		}
	}
}






func TestHandleScreenshotContent_PathTraversal(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/screenshots/content?path=/etc/passwd", nil)
	w := httptest.NewRecorder()

	server.handleScreenshotContent(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusForbidden)
	}
}





// ==================== pipeline_handlers.go — additional paths ====================

func TestHandleScanDiff_MissingIDs(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/projects//scan/diff?before=run1&after=run2", nil)
	req.SetPathValue("id", "")
	w := httptest.NewRecorder()

	server.handleScanDiff(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		t.Error("expected non-200 for missing project ID")
	}
}

func TestHandleGetPipelineConfig_MissingProjectID(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/projects//pipeline/config", nil)
	req.SetPathValue("id", "")
	w := httptest.NewRecorder()

	server.handleGetPipelineConfig(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		t.Error("expected non-200 for missing project ID")
	}
}

func TestHandleListPipelineRuns_MissingProjectID(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/projects//pipeline-runs", nil)
	req.SetPathValue("id", "")
	w := httptest.NewRecorder()

	server.handleListPipelineRuns(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		t.Error("expected non-200 for missing project ID")
	}
}

func TestHandleGetPipelineRunStages_MissingParams(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/projects//pipeline/runs//stages", nil)
	req.SetPathValue("id", "")
	req.SetPathValue("runId", "")
	w := httptest.NewRecorder()

	server.handleGetPipelineRunStages(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleGetPipelineRunStages_RunNotFound(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+p.ID+"/pipeline/runs/nonexistent/stages", nil)
	req.SetPathValue("id", p.ID)
	req.SetPathValue("runId", "nonexistent")
	w := httptest.NewRecorder()

	server.handleGetPipelineRunStages(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

// ==================== handlers.go — additional paths ====================

func TestHandleListWorkers_MultipleWorkers(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	createTestWorker(t, queries, models.WorkerStatusOffline)
	createTestWorker(t, queries, models.WorkerStatusOnline)
	createTestWorker(t, queries, models.WorkerStatusBusy)

	req := httptest.NewRequest(http.MethodGet, "/workers", nil)
	w := httptest.NewRecorder()

	server.handleListWorkers(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var workers []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&workers); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(workers) != 3 {
		t.Errorf("len(workers) = %d, want 3", len(workers))
	}
}

func TestHandleGetToolTemplate_Found(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	templates, _ := queries.ListToolTemplates()
	if len(templates) == 0 {
		t.Skip("no tool templates in DB")
	}

	req := httptest.NewRequest(http.MethodGet, "/tool-templates/"+templates[0].ID, nil)
	req.SetPathValue("id", templates[0].ID)
	w := httptest.NewRecorder()

	server.handleGetToolTemplate(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

// ==================== httpx_fingerprint_handlers.go — writeHttpxFpMgrError ====================

// errNotFound creates a plain error whose message contains "not found".
type notFoundError struct{ msg string }

func (e *notFoundError) Error() string { return e.msg }
func errNotFound(msg string) error     { return &notFoundError{msg} }

func TestWriteHttpxFpMgrError_NotFoundString(t *testing.T) {
	w := httptest.NewRecorder()
	writeHttpxFpMgrError(w, errNotFound("fingerprint xyz not found"), "test")

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestWriteHttpxFpMgrError_Generic(t *testing.T) {
	w := httptest.NewRecorder()
	writeHttpxFpMgrError(w, errGeneric, "test action")

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusInternalServerError)
	}
}

// ==================== dictionary_handlers.go — nil manager paths ====================

func TestHandleDeleteDictionary_NilManager(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	orig := server.dictMgr
	server.dictMgr = nil
	defer func() { server.dictMgr = orig }()

	req := httptest.NewRequest(http.MethodDelete, "/dictionaries/d1", nil)
	req.SetPathValue("id", "d1")
	w := httptest.NewRecorder()

	server.handleDeleteDictionary(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusServiceUnavailable)
	}
}

func TestHandleWriteDictionaryContent_NilManager(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	orig := server.dictMgr
	server.dictMgr = nil
	defer func() { server.dictMgr = orig }()

	req := httptest.NewRequest(http.MethodPut, "/dictionaries/d1/content", strings.NewReader("word1\nword2"))
	req.SetPathValue("id", "d1")
	w := httptest.NewRecorder()

	server.handleWriteDictionaryContent(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusServiceUnavailable)
	}
}

func TestHandleReadDictionaryContent_NilManager(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	orig := server.dictMgr
	server.dictMgr = nil
	defer func() { server.dictMgr = orig }()

	req := httptest.NewRequest(http.MethodGet, "/dictionaries/d1/content", nil)
	req.SetPathValue("id", "d1")
	w := httptest.NewRecorder()

	server.handleReadDictionaryContent(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusServiceUnavailable)
	}
}

// ==================== httpx_fingerprint_handlers.go — nil manager paths ====================

func TestHandlePatchHttpxFingerprint_NilManager(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	orig := server.httpxFpMgr
	server.httpxFpMgr = nil
	defer func() { server.httpxFpMgr = orig }()

	body := `{"name": "test"}`
	req := httptest.NewRequest(http.MethodPatch, "/httpx-fingerprints/fp1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "fp1")
	w := httptest.NewRecorder()

	server.handlePatchHttpxFingerprint(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusServiceUnavailable)
	}
}

func TestHandlePatchHttpxFingerprintEnabled_NilManager(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	orig := server.httpxFpMgr
	server.httpxFpMgr = nil
	defer func() { server.httpxFpMgr = orig }()

	body := `{"enabled": false}`
	req := httptest.NewRequest(http.MethodPatch, "/httpx-fingerprints/fp1/enabled", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "fp1")
	w := httptest.NewRecorder()

	server.handlePatchHttpxFingerprintEnabled(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusServiceUnavailable)
	}
}

func TestHandleDeleteHttpxFingerprint_NilManager(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	orig := server.httpxFpMgr
	server.httpxFpMgr = nil
	defer func() { server.httpxFpMgr = orig }()

	req := httptest.NewRequest(http.MethodDelete, "/httpx-fingerprints/fp1", nil)
	req.SetPathValue("id", "fp1")
	w := httptest.NewRecorder()

	server.handleDeleteHttpxFingerprint(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusServiceUnavailable)
	}
}

// ==================== scope_handlers.go — additional paths ====================

func TestHandleParseScopeValue_Success(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	body := `{"value": "example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/scope/parse", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleParseScopeValue(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestHandleParseScopeValue_InvalidBody(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/scope/parse", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleParseScopeValue(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}


func TestHandleBatchCreateScopeRules_InvalidBody(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)

	req := httptest.NewRequest(http.MethodPost, "/projects/"+p.ID+"/scope-rules/batch", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleBatchCreateScopeRules(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleCreateScopeRule_InvalidBody(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/scope-rules", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleCreateScopeRule(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

// ==================== target_handlers.go — additional paths ====================

func TestHandleCreateTarget_InvalidBody(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)

	req := httptest.NewRequest(http.MethodPost, "/projects/"+p.ID+"/targets", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleCreateTarget(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}


func TestHandleImportTargets_InvalidMultipart(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)

	req := httptest.NewRequest(http.MethodPost, "/projects/"+p.ID+"/targets/import", strings.NewReader("not multipart"))
	req.Header.Set("Content-Type", "multipart/form-data")
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleImportTargets(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}




// ==================== workdir_cleanup.go — cleanupProjectWorkdir ====================

func TestCleanupProjectWorkdir_Coverage(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	server.cleanupProjectWorkdir("nonexistent-project")
}
