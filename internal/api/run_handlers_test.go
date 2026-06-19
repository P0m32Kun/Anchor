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

// --- handleListRuns ---

func TestHandleListRuns_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	now := time.Now().UTC()

	for i := 0; i < 3; i++ {
		queries.CreateRun(&models.Run{
			ID: util.GenerateID(), ProjectID: p.ID,
			Name: "run-" + string(rune('a'+i)), Status: models.RunCompleted,
			CreatedAt: now,
		})
	}

	req := httptest.NewRequest(http.MethodGet, "/projects/"+p.ID+"/runs", nil)
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleListRuns(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result PaginatedResponse[*models.Run]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Total != 3 {
		t.Errorf("total = %d, want 3", result.Total)
	}
}

// --- handleGetRun ---

func TestHandleGetRun_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	now := time.Now().UTC()

	run := &models.Run{
		ID: util.GenerateID(), ProjectID: p.ID,
		Name: "test-run", Status: models.RunRunning,
		StartedAt: &now, CreatedAt: now,
	}
	queries.CreateRun(run)

	req := httptest.NewRequest(http.MethodGet, "/runs/"+run.ID, nil)
	req.SetPathValue("id", run.ID)
	w := httptest.NewRecorder()

	server.handleGetRun(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result models.Run
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.ID != run.ID {
		t.Errorf("id = %q, want %q", result.ID, run.ID)
	}
	if result.Name != "test-run" {
		t.Errorf("name = %q, want test-run", result.Name)
	}
}

func TestHandleGetRun_NotFound(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/runs/nonexistent", nil)
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()

	server.handleGetRun(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

// --- handleGetRunTasks ---

func TestHandleGetRunTasks_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	now := time.Now().UTC()

	run := &models.Run{
		ID: util.GenerateID(), ProjectID: p.ID,
		Name: "test-run", Status: models.RunRunning, CreatedAt: now,
	}
	queries.CreateRun(run)

	runID := run.ID
	queries.CreateScanTask(&models.ScanTask{
		ID: util.GenerateID(), ProjectID: p.ID, RunID: &runID,
		Tool: "nuclei", Status: models.TaskRunning, CreatedAt: now,
	})
	queries.CreateScanTask(&models.ScanTask{
		ID: util.GenerateID(), ProjectID: p.ID, RunID: &runID,
		Tool: "httpx", Status: models.TaskQueued, CreatedAt: now,
	})

	req := httptest.NewRequest(http.MethodGet, "/runs/"+run.ID+"/tasks", nil)
	req.SetPathValue("id", run.ID)
	w := httptest.NewRecorder()

	server.handleGetRunTasks(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result []*models.ScanTask
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("len = %d, want 2", len(result))
	}
}

// --- handleCancelRun ---

func TestHandleCancelRun_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	now := time.Now().UTC()

	run := &models.Run{
		ID: util.GenerateID(), ProjectID: p.ID,
		Name: "test-run", Status: models.RunRunning, CreatedAt: now,
	}
	queries.CreateRun(run)

	req := httptest.NewRequest(http.MethodPost, "/runs/"+run.ID+"/cancel", nil)
	req.SetPathValue("id", run.ID)
	w := httptest.NewRecorder()

	server.handleCancelRun(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// Verify run was cancelled
	updated, _ := queries.GetRun(run.ID)
	if updated.Status != models.RunCancelled {
		t.Errorf("status = %q, want cancelled", updated.Status)
	}
}

func TestHandleCancelRun_AlreadyFinished(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	now := time.Now().UTC()

	run := &models.Run{
		ID: util.GenerateID(), ProjectID: p.ID,
		Name: "test-run", Status: models.RunCompleted, CreatedAt: now,
	}
	queries.CreateRun(run)

	req := httptest.NewRequest(http.MethodPost, "/runs/"+run.ID+"/cancel", nil)
	req.SetPathValue("id", run.ID)
	w := httptest.NewRecorder()

	server.handleCancelRun(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleCancelRun_NotFound(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/runs/nonexistent/cancel", nil)
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()

	server.handleCancelRun(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

// --- handleCreateRun ---

func TestHandleCreateRun_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)

	body, _ := json.Marshal(map[string]interface{}{
		"name": "test-run",
	})

	req := httptest.NewRequest(http.MethodPost, "/projects/"+p.ID+"/runs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleCreateRun(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	var run models.Run
	if err := json.NewDecoder(resp.Body).Decode(&run); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if run.Name != "test-run" {
		t.Errorf("name = %q, want test-run", run.Name)
	}
	if run.ProjectID != p.ID {
		t.Errorf("project_id = %q, want %q", run.ProjectID, p.ID)
	}
	if run.Status != models.RunPending {
		t.Errorf("status = %q, want pending", run.Status)
	}
}

func TestHandleCreateRun_DefaultName(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)

	body, _ := json.Marshal(map[string]interface{}{})

	req := httptest.NewRequest(http.MethodPost, "/projects/"+p.ID+"/runs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleCreateRun(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	var run models.Run
	json.NewDecoder(resp.Body).Decode(&run)
	if run.Name != "未命名扫描" {
		t.Errorf("name = %q, want 未命名扫描", run.Name)
	}
}

func TestHandleCreateRun_WithToolTemplate(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)

	// Create a tool_template row so the FK constraint on tool_template_id is satisfied.
	if _, err := rawDB.Exec(`INSERT INTO tool_templates (id, name, profile_type, tools_json) VALUES (?, ?, ?, ?)`,
		"tpl-123", "test-tpl", "external", "[]"); err != nil {
		t.Fatalf("insert tool_template: %v", err)
	}

	body, _ := json.Marshal(map[string]interface{}{
		"name":             "template-run",
		"tool_template_id": "tpl-123",
	})

	req := httptest.NewRequest(http.MethodPost, "/projects/"+p.ID+"/runs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleCreateRun(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	var run models.Run
	json.NewDecoder(resp.Body).Decode(&run)
	if run.ToolTemplateID == nil || *run.ToolTemplateID != "tpl-123" {
		t.Errorf("tool_template_id = %v, want tpl-123", run.ToolTemplateID)
	}
}

func TestHandleCreateRun_InvalidBody(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)

	req := httptest.NewRequest(http.MethodPost, "/projects/"+p.ID+"/runs", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleCreateRun(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

// --- checkRunCompletion ---

func TestCheckRunCompletion_AllCompleted(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	now := time.Now().UTC()

	run := &models.Run{
		ID: util.GenerateID(), ProjectID: p.ID,
		Name: "test-run", Status: models.RunRunning, CreatedAt: now,
	}
	queries.CreateRun(run)

	runID := run.ID
	queries.CreateScanTask(&models.ScanTask{
		ID: util.GenerateID(), ProjectID: p.ID, RunID: &runID,
		Tool: "nuclei", Status: models.TaskCompleted, CreatedAt: now,
	})
	queries.CreateScanTask(&models.ScanTask{
		ID: util.GenerateID(), ProjectID: p.ID, RunID: &runID,
		Tool: "httpx", Status: models.TaskCompleted, CreatedAt: now,
	})

	server.checkRunCompletion(run.ID)

	updated, _ := queries.GetRun(run.ID)
	if updated.Status != models.RunCompleted {
		t.Errorf("status = %q, want completed", updated.Status)
	}
}

func TestCheckRunCompletion_HasFailed(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	now := time.Now().UTC()

	run := &models.Run{
		ID: util.GenerateID(), ProjectID: p.ID,
		Name: "test-run", Status: models.RunRunning, CreatedAt: now,
	}
	queries.CreateRun(run)

	runID := run.ID
	queries.CreateScanTask(&models.ScanTask{
		ID: util.GenerateID(), ProjectID: p.ID, RunID: &runID,
		Tool: "nuclei", Status: models.TaskCompleted, CreatedAt: now,
	})
	queries.CreateScanTask(&models.ScanTask{
		ID: util.GenerateID(), ProjectID: p.ID, RunID: &runID,
		Tool: "httpx", Status: models.TaskFailed, CreatedAt: now,
	})

	server.checkRunCompletion(run.ID)

	updated, _ := queries.GetRun(run.ID)
	if updated.Status != models.RunFailed {
		t.Errorf("status = %q, want failed", updated.Status)
	}
}

func TestCheckRunCompletion_NotAllDone(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	now := time.Now().UTC()

	run := &models.Run{
		ID: util.GenerateID(), ProjectID: p.ID,
		Name: "test-run", Status: models.RunRunning, CreatedAt: now,
	}
	queries.CreateRun(run)

	runID := run.ID
	queries.CreateScanTask(&models.ScanTask{
		ID: util.GenerateID(), ProjectID: p.ID, RunID: &runID,
		Tool: "nuclei", Status: models.TaskCompleted, CreatedAt: now,
	})
	queries.CreateScanTask(&models.ScanTask{
		ID: util.GenerateID(), ProjectID: p.ID, RunID: &runID,
		Tool: "httpx", Status: models.TaskRunning, CreatedAt: now,
	})

	server.checkRunCompletion(run.ID)

	updated, _ := queries.GetRun(run.ID)
	if updated.Status != models.RunRunning {
		t.Errorf("status = %q, want running (not all done)", updated.Status)
	}
}

func TestCheckRunCompletion_NoTasks(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	now := time.Now().UTC()

	run := &models.Run{
		ID: util.GenerateID(), ProjectID: p.ID,
		Name: "test-run", Status: models.RunRunning, CreatedAt: now,
	}
	queries.CreateRun(run)

	// No tasks — allDone=true, hasFailed=false → completed
	server.checkRunCompletion(run.ID)

	updated, _ := queries.GetRun(run.ID)
	if updated.Status != models.RunCompleted {
		t.Errorf("status = %q, want completed (empty task list)", updated.Status)
	}
}
