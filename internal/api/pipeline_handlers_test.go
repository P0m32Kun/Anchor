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

func createTestPipelineRun(t *testing.T, queries *db.Queries, projectID, status string) *models.PipelineRun {
	t.Helper()
	now := time.Now().UTC()
	r := &models.PipelineRun{
		ID:        util.GenerateID(),
		ProjectID: projectID,
		Mode:      "external",
		Status:    status,
		StartedAt: now,
		CreatedAt: now,
	}
	if err := queries.CreatePipelineRun(r); err != nil {
		t.Fatalf("create pipeline run: %v", err)
	}
	return r
}

// --- handleListPipelineRuns ---

func TestHandleListPipelineRuns_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	createTestPipelineRun(t, queries, p.ID, "running")
	createTestPipelineRun(t, queries, p.ID, "completed")

	req := httptest.NewRequest(http.MethodGet, "/projects/"+p.ID+"/pipeline-runs", nil)
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleListPipelineRuns(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	runs, ok := result["runs"].([]interface{})
	if !ok {
		t.Fatalf("runs is not an array: %T", result["runs"])
	}
	if len(runs) != 2 {
		t.Errorf("len(runs) = %d, want 2", len(runs))
	}
}

func TestHandleListPipelineRuns_EmptyProjectID(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/projects//pipeline-runs", nil)
	w := httptest.NewRecorder()

	server.handleListPipelineRuns(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

// --- handleGetPipelineRun ---

func TestHandleGetPipelineRun_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	run := createTestPipelineRun(t, queries, p.ID, "running")

	req := httptest.NewRequest(http.MethodGet, "/projects/"+p.ID+"/pipeline-runs/"+run.ID, nil)
	req.SetPathValue("id", p.ID)
	req.SetPathValue("runId", run.ID)
	w := httptest.NewRecorder()

	server.handleGetPipelineRun(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var got models.PipelineRun
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.ID != run.ID {
		t.Errorf("id = %q, want %q", got.ID, run.ID)
	}
}

func TestHandleGetPipelineRun_NotFound(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+p.ID+"/pipeline-runs/nonexistent", nil)
	req.SetPathValue("id", p.ID)
	req.SetPathValue("runId", "nonexistent")
	w := httptest.NewRecorder()

	server.handleGetPipelineRun(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestHandleGetPipelineRun_MismatchedProject(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p1 := createTestProject(t, queries)
	p2 := createTestProject(t, queries)
	run := createTestPipelineRun(t, queries, p1.ID, "running")

	req := httptest.NewRequest(http.MethodGet, "/projects/"+p2.ID+"/pipeline-runs/"+run.ID, nil)
	req.SetPathValue("id", p2.ID)
	req.SetPathValue("runId", run.ID)
	w := httptest.NewRecorder()

	server.handleGetPipelineRun(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestHandleGetPipelineRun_MissingParams(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/projects//pipeline-runs/", nil)
	req.SetPathValue("id", "")
	req.SetPathValue("runId", "")
	w := httptest.NewRecorder()

	server.handleGetPipelineRun(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

// --- handleGetRunSummary ---

func TestHandleGetRunSummary_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	run := createTestPipelineRun(t, queries, p.ID, "completed")

	// Create a stage with work items
	total := 10
	done := 10
	err := queries.CreatePipelineRunStage(&models.PipelineRunStage{
		ID:        util.GenerateID(),
		RunID:     run.ID,
		Stage:     "subfinder",
		Status:    models.StageStatusCompleted,
		WorkTotal: &total,
		WorkDone:  &done,
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("create stage: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/projects/"+p.ID+"/pipeline-runs/"+run.ID+"/summary", nil)
	req.SetPathValue("id", p.ID)
	req.SetPathValue("runId", run.ID)
	w := httptest.NewRecorder()

	server.handleGetRunSummary(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := json.MarshalIndent(map[string]string{"body": w.Body.String()}, "", "  ")
		t.Fatalf("status = %d, want %d; body=%s", resp.StatusCode, http.StatusOK, body)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result["run_id"] != run.ID {
		t.Errorf("run_id = %v, want %s", result["run_id"], run.ID)
	}
	if result["complete"] != true {
		t.Errorf("complete = %v, want true", result["complete"])
	}
}

func TestHandleGetRunSummary_NotFound(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+p.ID+"/pipeline-runs/nonexistent/summary", nil)
	req.SetPathValue("id", p.ID)
	req.SetPathValue("runId", "nonexistent")
	w := httptest.NewRecorder()

	server.handleGetRunSummary(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

// --- handleGetPipelineConfig ---

func TestHandleGetPipelineConfig_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+p.ID+"/pipeline-config", nil)
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleGetPipelineConfig(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var cfg models.PipelineConfig
	if err := json.NewDecoder(resp.Body).Decode(&cfg); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// Should return default config
	if cfg.PortRange == "" {
		t.Log("PortRange is empty (may be default)")
	}
}

func TestHandleGetPipelineConfig_ProjectNotFound(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/projects/nonexistent/pipeline-config", nil)
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()

	server.handleGetPipelineConfig(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

// --- handleUpdatePipelineConfig ---

func TestHandleUpdatePipelineConfig_InvalidBody(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)

	req := httptest.NewRequest(http.MethodPut, "/projects/"+p.ID+"/pipeline-config", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleUpdatePipelineConfig(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

// --- handleGetPipelineRunStages ---

func TestHandleGetPipelineRunStages_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	run := createTestPipelineRun(t, queries, p.ID, "running")

	total := 5
	done := 3
	err := queries.CreatePipelineRunStage(&models.PipelineRunStage{
		ID:        util.GenerateID(),
		RunID:     run.ID,
		Stage:     "httpx",
		Status:    models.StageStatusRunning,
		WorkTotal: &total,
		WorkDone:  &done,
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("create stage: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/projects/"+p.ID+"/pipeline-runs/"+run.ID+"/stages", nil)
	req.SetPathValue("id", p.ID)
	req.SetPathValue("runId", run.ID)
	w := httptest.NewRecorder()

	server.handleGetPipelineRunStages(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	stages, ok := result["stages"].([]interface{})
	if !ok {
		t.Fatalf("stages is not an array: %T", result["stages"])
	}
	if len(stages) != 1 {
		t.Errorf("len(stages) = %d, want 1", len(stages))
	}
}

func TestHandleGetPipelineRunStages_NotFound(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+p.ID+"/pipeline-runs/nonexistent/stages", nil)
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

func TestHandleGetPipelineRunStages_MismatchedProject(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p1 := createTestProject(t, queries)
	p2 := createTestProject(t, queries)
	run := createTestPipelineRun(t, queries, p1.ID, "running")

	req := httptest.NewRequest(http.MethodGet, "/projects/"+p2.ID+"/pipeline-runs/"+run.ID+"/stages", nil)
	req.SetPathValue("id", p2.ID)
	req.SetPathValue("runId", run.ID)
	w := httptest.NewRecorder()

	server.handleGetPipelineRunStages(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

// --- handleListScanRuns ---

func TestHandleListScanRuns_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	createTestPipelineRun(t, queries, p.ID, "running")
	createTestPipelineRun(t, queries, p.ID, "completed")

	req := httptest.NewRequest(http.MethodGet, "/projects/"+p.ID+"/scan-runs", nil)
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleListScanRuns(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result PaginatedResponse[*models.PipelineRun]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Total != 2 {
		t.Errorf("total = %d, want 2", result.Total)
	}
}

func TestHandleListScanRuns_EmptyProjectID(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/projects//scan-runs", nil)
	req.SetPathValue("id", "")
	w := httptest.NewRecorder()

	server.handleListScanRuns(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

// --- handleCancelPipelineRun ---

func TestHandleCancelPipelineRun_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	run := createTestPipelineRun(t, queries, p.ID, "running")

	req := httptest.NewRequest(http.MethodPost, "/projects/"+p.ID+"/pipeline-runs/"+run.ID+"/cancel", nil)
	req.SetPathValue("id", p.ID)
	req.SetPathValue("runId", run.ID)
	w := httptest.NewRecorder()

	server.handleCancelPipelineRun(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// Verify run was cancelled
	updated, _ := queries.GetPipelineRun(run.ID)
	if updated.Status != "cancelled" {
		t.Errorf("status = %q, want cancelled", updated.Status)
	}
}

func TestHandleCancelPipelineRun_AlreadyCompleted(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	run := createTestPipelineRun(t, queries, p.ID, "completed")

	req := httptest.NewRequest(http.MethodPost, "/projects/"+p.ID+"/pipeline-runs/"+run.ID+"/cancel", nil)
	req.SetPathValue("id", p.ID)
	req.SetPathValue("runId", run.ID)
	w := httptest.NewRecorder()

	server.handleCancelPipelineRun(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleCancelPipelineRun_NotFound(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)

	req := httptest.NewRequest(http.MethodPost, "/projects/"+p.ID+"/pipeline-runs/nonexistent/cancel", nil)
	req.SetPathValue("id", p.ID)
	req.SetPathValue("runId", "nonexistent")
	w := httptest.NewRecorder()

	server.handleCancelPipelineRun(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestHandleCancelPipelineRun_MismatchedProject(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p1 := createTestProject(t, queries)
	p2 := createTestProject(t, queries)
	run := createTestPipelineRun(t, queries, p1.ID, "running")

	req := httptest.NewRequest(http.MethodPost, "/projects/"+p2.ID+"/pipeline-runs/"+run.ID+"/cancel", nil)
	req.SetPathValue("id", p2.ID)
	req.SetPathValue("runId", run.ID)
	w := httptest.NewRecorder()

	server.handleCancelPipelineRun(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

// --- handleCreateScan ---

func TestHandleCreateScan_InvalidBody(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)

	req := httptest.NewRequest(http.MethodPost, "/projects/"+p.ID+"/scan", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleCreateScan(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleCreateScan_ProjectNotFound(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	body, _ := json.Marshal(map[string]interface{}{"mode": "external"})
	req := httptest.NewRequest(http.MethodPost, "/projects/nonexistent/scan", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()

	server.handleCreateScan(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestHandleCreateScan_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)

	body, _ := json.Marshal(map[string]interface{}{"mode": "external"})
	req := httptest.NewRequest(http.MethodPost, "/projects/"+p.ID+"/scan", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleCreateScan(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusAccepted)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result["run_id"] == "" {
		t.Error("run_id is empty")
	}
	if result["status"] != "accepted" {
		t.Errorf("status = %q, want accepted", result["status"])
	}
}

// --- handleScanDiff ---

func TestHandleScanDiff_MissingParams(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/projects/p1/scan/diff", nil)
	req.SetPathValue("id", "p1")
	w := httptest.NewRecorder()

	server.handleScanDiff(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleScanDiff_SameRun(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/projects/p1/scan/diff?base=r1&target=r1", nil)
	req.SetPathValue("id", "p1")
	w := httptest.NewRecorder()

	server.handleScanDiff(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleScanDiff_BaseRunNotFound(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+p.ID+"/scan/diff?base=nonexistent&target=also-nonexistent", nil)
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleScanDiff(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestHandleScanDiff_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	baseRun := createTestPipelineRun(t, queries, p.ID, "completed")
	targetRun := createTestPipelineRun(t, queries, p.ID, "completed")

	req := httptest.NewRequest(http.MethodGet, "/projects/"+p.ID+"/scan/diff?base="+baseRun.ID+"&target="+targetRun.ID, nil)
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleScanDiff(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result models.ScanDiffResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.BaseRunID != baseRun.ID {
		t.Errorf("base_run_id = %q, want %q", result.BaseRunID, baseRun.ID)
	}
	if result.TargetRunID != targetRun.ID {
		t.Errorf("target_run_id = %q, want %q", result.TargetRunID, targetRun.ID)
	}
}
