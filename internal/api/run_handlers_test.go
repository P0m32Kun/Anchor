package api

import (
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
