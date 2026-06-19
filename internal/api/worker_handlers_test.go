package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// --- helpers ---

func createTestWorker(t *testing.T, queries *db.Queries, status models.WorkerStatus) *models.WorkerNode {
	t.Helper()
	now := time.Now().UTC()
	w := &models.WorkerNode{
		ID:             util.GenerateID(),
		Name:           "test-worker",
		Endpoint:       "http://localhost:9999",
		Mode:           models.WorkerModeRemote,
		Status:         status,
		TrustLevel:     "standard",
		MaxConcurrency: 10,
		LastSeen:       &now,
		CreatedAt:      now,
	}
	if err := queries.CreateWorkerNode(w); err != nil {
		t.Fatalf("create worker: %v", err)
	}
	return w
}

// --- handleRegisterWorker ---

func TestHandleRegisterWorker_Success(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	body, _ := json.Marshal(map[string]interface{}{
		"name":            "worker-1",
		"endpoint":        "http://10.0.0.1:8080",
		"max_concurrency": 5,
	})

	req := httptest.NewRequest(http.MethodPost, "/workers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleRegisterWorker(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result["worker_id"] == nil || result["worker_id"] == "" {
		t.Error("worker_id is empty")
	}
	if result["token"] == nil || result["token"] == "" {
		t.Error("token is empty")
	}
}

func TestHandleRegisterWorker_DefaultConcurrency(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	body, _ := json.Marshal(map[string]interface{}{
		"name":     "worker-1",
		"endpoint": "http://10.0.0.1:8080",
	})

	req := httptest.NewRequest(http.MethodPost, "/workers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleRegisterWorker(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}
}

func TestHandleRegisterWorker_InvalidBody(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/workers", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleRegisterWorker(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

// --- handleWorkerHeartbeat ---

func TestHandleWorkerHeartbeat_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	worker := createTestWorker(t, queries, models.WorkerStatusOnline)

	body, _ := json.Marshal(map[string]interface{}{
		"status": "online",
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

func TestHandleWorkerHeartbeat_WithMetrics(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	worker := createTestWorker(t, queries, models.WorkerStatusOnline)

	cpu := 45.5
	mem := 60.2
	disk := 30.0
	body, _ := json.Marshal(map[string]interface{}{
		"status":      "busy",
		"cpu_percent": cpu,
		"mem_percent": mem,
		"disk_percent": disk,
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

func TestHandleWorkerHeartbeat_WithTemplateVersions(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	worker := createTestWorker(t, queries, models.WorkerStatusOnline)

	body, _ := json.Marshal(map[string]interface{}{
		"status":            "online",
		"template_versions": `{"nuclei":"v3.0"}`,
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

func TestHandleWorkerHeartbeat_IdleStatus(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	worker := createTestWorker(t, queries, models.WorkerStatusBusy)

	body, _ := json.Marshal(map[string]interface{}{
		"status": "idle",
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

func TestHandleWorkerHeartbeat_InvalidBody(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	worker := createTestWorker(t, queries, models.WorkerStatusOnline)

	req := httptest.NewRequest(http.MethodPost, "/workers/"+worker.ID+"/heartbeat", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", worker.ID)
	w := httptest.NewRecorder()

	server.handleWorkerHeartbeat(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

// --- handlePollTasks ---

func TestHandlePollTasks_WorkerNotFound(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/workers/nonexistent/tasks/poll", nil)
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()

	server.handlePollTasks(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestHandlePollTasks_RevokedWorker(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	worker := createTestWorker(t, queries, models.WorkerStatusOnline)

	// Revoke the worker
	now := time.Now().UTC()
	queries.RevokeWorkerNode(worker.ID, now)

	req := httptest.NewRequest(http.MethodGet, "/workers/"+worker.ID+"/tasks/poll", nil)
	req.SetPathValue("id", worker.ID)
	w := httptest.NewRecorder()

	server.handlePollTasks(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestHandlePollTasks_TimeoutNoContent(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	worker := createTestWorker(t, queries, models.WorkerStatusOnline)

	req := httptest.NewRequest(http.MethodGet, "/workers/"+worker.ID+"/tasks/poll", nil)
	req.SetPathValue("id", worker.ID)
	w := httptest.NewRecorder()

	server.handlePollTasks(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	// No tasks queued, should timeout with 204
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNoContent)
	}
}

// --- handleRevokeWorker ---

func TestHandleRevokeWorker_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	worker := createTestWorker(t, queries, models.WorkerStatusOnline)

	req := httptest.NewRequest(http.MethodPost, "/workers/"+worker.ID+"/revoke", nil)
	req.SetPathValue("id", worker.ID)
	w := httptest.NewRecorder()

	server.handleRevokeWorker(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// Verify worker was revoked
	updated, _ := queries.GetWorkerNode(worker.ID)
	if updated.RevokedAt == nil {
		t.Error("expected revoked_at to be set")
	}
}

// --- handleDeleteWorker ---

func TestHandleDeleteWorker_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	worker := createTestWorker(t, queries, models.WorkerStatusOffline)

	req := httptest.NewRequest(http.MethodDelete, "/workers/"+worker.ID, nil)
	req.SetPathValue("id", worker.ID)
	w := httptest.NewRecorder()

	server.handleDeleteWorker(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// Verify deleted
	got, _ := queries.GetWorkerNode(worker.ID)
	if got != nil {
		t.Error("worker still exists after delete")
	}
}

func TestHandleDeleteWorker_OnlyOffline(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	worker := createTestWorker(t, queries, models.WorkerStatusOnline)

	req := httptest.NewRequest(http.MethodDelete, "/workers/"+worker.ID, nil)
	req.SetPathValue("id", worker.ID)
	w := httptest.NewRecorder()

	server.handleDeleteWorker(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

// --- handleTaskResult ---

func TestHandleTaskResult_InvalidBody(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/tasks/task1/result", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "task1")
	w := httptest.NewRecorder()

	server.handleTaskResult(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleTaskResult_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	now := time.Now().UTC()

	task := &models.ScanTask{
		ID:        util.GenerateID(),
		ProjectID: p.ID,
		Tool:      "nuclei",
		Status:    models.TaskRunning,
		CreatedAt: now,
	}
	queries.CreateScanTask(task)

	body, _ := json.Marshal(map[string]interface{}{
		"status": "completed",
		"error":  "",
	})

	req := httptest.NewRequest(http.MethodPost, "/tasks/"+task.ID+"/result", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", task.ID)
	w := httptest.NewRecorder()

	server.handleTaskResult(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestHandleTaskResult_WithError(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	now := time.Now().UTC()

	task := &models.ScanTask{
		ID:        util.GenerateID(),
		ProjectID: p.ID,
		Tool:      "nuclei",
		Status:    models.TaskRunning,
		CreatedAt: now,
	}
	queries.CreateScanTask(task)

	body, _ := json.Marshal(map[string]interface{}{
		"status": "failed",
		"error":  "tool crashed",
	})

	req := httptest.NewRequest(http.MethodPost, "/tasks/"+task.ID+"/result", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", task.ID)
	w := httptest.NewRecorder()

	server.handleTaskResult(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

// --- handleRevokeWorker DB error ---

func TestHandleRevokeWorker_DBError(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	rawDB.Close()

	req := httptest.NewRequest(http.MethodPost, "/workers/some-id/revoke", nil)
	req.SetPathValue("id", "some-id")
	w := httptest.NewRecorder()

	server.handleRevokeWorker(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusInternalServerError)
	}
}

// --- handleRegisterWorker DB error ---

func TestHandleRegisterWorker_DBError(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	rawDB.Close()

	body, _ := json.Marshal(map[string]interface{}{
		"name":     "worker-1",
		"endpoint": "http://10.0.0.1:8080",
	})

	req := httptest.NewRequest(http.MethodPost, "/workers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleRegisterWorker(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusInternalServerError)
	}
}

// --- handleWorkerHeartbeat DB errors ---

func TestHandleWorkerHeartbeat_UpdateStatusError(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	worker := createTestWorker(t, queries, models.WorkerStatusOnline)

	rawDB.Close()

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

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusInternalServerError)
	}
}

func TestHandleWorkerHeartbeat_UpdateTemplateVersionsError(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	worker := createTestWorker(t, queries, models.WorkerStatusOnline)

	rawDB.Close()

	body, _ := json.Marshal(map[string]interface{}{
		"status":            "online",
		"template_versions": `{"nuclei":"v3.0"}`,
	})

	req := httptest.NewRequest(http.MethodPost, "/workers/"+worker.ID+"/heartbeat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", worker.ID)
	w := httptest.NewRecorder()

	server.handleWorkerHeartbeat(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusInternalServerError)
	}
}

// --- updateWorkerMetrics ---

func TestUpdateWorkerMetrics_AllNil(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	server.updateWorkerMetrics("some-id", nil, nil, nil)
}

func TestUpdateWorkerMetrics_WithValues(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	worker := createTestWorker(t, queries, models.WorkerStatusOnline)

	cpu := 50.0
	mem := 75.0
	disk := 25.0

	server.updateWorkerMetrics(worker.ID, &cpu, &mem, &disk)

	// GetWorkerNode does not SELECT metric columns; use ListWorkerNodes
	// (which includes cpu_percent, mem_percent, disk_percent) to verify.
	list, err := queries.ListWorkerNodes()
	if err != nil {
		t.Fatalf("list workers: %v", err)
	}
	var updated *models.WorkerNode
	for _, w := range list {
		if w.ID == worker.ID {
			updated = w
			break
		}
	}
	if updated == nil {
		t.Fatal("worker not found")
	}
	if updated.CPUPercent == nil || *updated.CPUPercent != cpu {
		t.Errorf("cpu = %v, want %v", updated.CPUPercent, cpu)
	}
	if updated.MemPercent == nil || *updated.MemPercent != mem {
		t.Errorf("mem = %v, want %v", updated.MemPercent, mem)
	}
	if updated.DiskPercent == nil || *updated.DiskPercent != disk {
		t.Errorf("disk = %v, want %v", updated.DiskPercent, disk)
	}
}

// --- handleDeleteWorker DB error on Get ---

func TestHandleDeleteWorker_DBErrorOnGet(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	rawDB.Close()

	req := httptest.NewRequest(http.MethodDelete, "/workers/some-id", nil)
	req.SetPathValue("id", "some-id")
	w := httptest.NewRecorder()

	server.handleDeleteWorker(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusInternalServerError)
	}
}

// --- handlePollTasks DB error ---

func TestHandlePollTasks_DBError(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	rawDB.Close()

	req := httptest.NewRequest(http.MethodGet, "/workers/some-id/tasks/poll", nil)
	req.SetPathValue("id", "some-id")
	w := httptest.NewRecorder()

	server.handlePollTasks(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusInternalServerError)
	}
}

