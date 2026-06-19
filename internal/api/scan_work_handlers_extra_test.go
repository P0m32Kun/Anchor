package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/P0m32Kun/Anchor/internal/db"
)

// --- handleGetScanDefaults ---

func TestHandleGetScanDefaults(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/scan/defaults", nil)
	w := httptest.NewRecorder()

	server.handleGetScanDefaults(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result == nil {
		t.Error("expected non-nil result")
	}
}

// --- requireRunInProject ---

func TestRequireRunInProject_NotFound(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	_, ok := server.requireRunInProject(w, req, "proj1", "nonexistent")
	if ok {
		t.Error("expected ok=false")
	}
}

func TestRequireRunInProject_WrongProject(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	run := createTestPipelineRun(t, queries, p.ID, "running")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	_, ok := server.requireRunInProject(w, req, "wrong-project", run.ID)
	if ok {
		t.Error("expected ok=false for wrong project")
	}
}

// --- parseScanDetailPagination ---

func TestParseScanDetailPagination_Default(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	pg := parseScanDetailPagination(req)
	if pg.PageSize != scanDetailDefaultPageSize {
		t.Errorf("page_size = %d, want %d", pg.PageSize, scanDetailDefaultPageSize)
	}
}

func TestParseScanDetailPagination_Custom(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test?page_size=100", nil)
	pg := parseScanDetailPagination(req)
	if pg.PageSize != 100 {
		t.Errorf("page_size = %d, want 100", pg.PageSize)
	}
}

// --- handleListScanRunWorks ---

func TestHandleListScanRunWorks_MissingRunID(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/projects/p1/pipeline/runs//works", nil)
	req.SetPathValue("id", "p1")
	req.SetPathValue("runId", "")
	w := httptest.NewRecorder()

	server.handleListScanRunWorks(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleListScanRunWorks_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	run := createTestPipelineRun(t, queries, p.ID, "completed")

	req := httptest.NewRequest(http.MethodGet, "/projects/"+p.ID+"/pipeline/runs/"+run.ID+"/works", nil)
	req.SetPathValue("id", p.ID)
	req.SetPathValue("runId", run.ID)
	w := httptest.NewRecorder()

	server.handleListScanRunWorks(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

// --- handleListAssetWorks ---

func TestHandleListAssetWorks_MissingAssetID(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/assets//works", nil)
	req.SetPathValue("id", "")
	w := httptest.NewRecorder()

	server.handleListAssetWorks(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleListAssetWorks_MissingRunID(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/assets/a1/works", nil)
	req.SetPathValue("id", "a1")
	w := httptest.NewRecorder()

	server.handleListAssetWorks(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleListAssetWorks_WithProjectFilter(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	run := createTestPipelineRun(t, queries, p.ID, "running")

	req := httptest.NewRequest(http.MethodGet, "/assets/a1/works?run_id="+run.ID+"&project_id="+p.ID, nil)
	req.SetPathValue("id", "a1")
	w := httptest.NewRecorder()

	server.handleListAssetWorks(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

// --- handleListToolCallLogs ---

func TestHandleListToolCallLogs_MissingRunID(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/projects/p1/pipeline/runs//tool-calls", nil)
	req.SetPathValue("id", "p1")
	req.SetPathValue("runId", "")
	w := httptest.NewRecorder()

	server.handleListToolCallLogs(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleListToolCallLogs_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	run := createTestPipelineRun(t, queries, p.ID, "completed")

	req := httptest.NewRequest(http.MethodGet, "/projects/"+p.ID+"/pipeline/runs/"+run.ID+"/tool-calls", nil)
	req.SetPathValue("id", p.ID)
	req.SetPathValue("runId", run.ID)
	w := httptest.NewRecorder()

	server.handleListToolCallLogs(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

// --- handleGetFindingTrace ---

func TestHandleGetFindingTrace_MissingID(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/findings//trace", nil)
	req.SetPathValue("findingId", "")
	w := httptest.NewRecorder()

	server.handleGetFindingTrace(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleGetFindingTrace_NotFound(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/findings/nonexistent/trace", nil)
	req.SetPathValue("findingId", "nonexistent")
	w := httptest.NewRecorder()

	server.handleGetFindingTrace(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

// --- handleGetScanRunMetrics ---

func TestHandleGetScanRunMetrics_MissingRunID(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/projects/p1/pipeline/runs//metrics", nil)
	req.SetPathValue("id", "p1")
	req.SetPathValue("runId", "")
	w := httptest.NewRecorder()

	server.handleGetScanRunMetrics(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleGetScanRunMetrics_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	run := createTestPipelineRun(t, queries, p.ID, "completed")

	req := httptest.NewRequest(http.MethodGet, "/projects/"+p.ID+"/pipeline/runs/"+run.ID+"/metrics", nil)
	req.SetPathValue("id", p.ID)
	req.SetPathValue("runId", run.ID)
	w := httptest.NewRecorder()

	server.handleGetScanRunMetrics(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 200 or 404", resp.StatusCode)
	}
}

// --- isTaskTerminal ---

// --- validateTaskOutputStream ---

func TestValidateTaskOutputStream(t *testing.T) {
	tests := []struct {
		stream string
		want   string
		err    bool
	}{
		{"", "stdout", false},
		{"stdout", "stdout", false},
		{"stderr", "stderr", false},
		{"invalid", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.stream, func(t *testing.T) {
			got, err := validateTaskOutputStream(tt.stream)
			if (err != nil) != tt.err {
				t.Errorf("err = %v, want err=%v", err, tt.err)
			}
			if got != tt.want {
				t.Errorf("got = %q, want %q", got, tt.want)
			}
		})
	}
}

// --- handleListScreenshots ---

// --- handleScreenshotFile ---

func TestHandleScreenshotFile_NotFound(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/screenshots/nonexistent/original", nil)
	req.SetPathValue("id", "nonexistent")
	req.SetPathValue("kind", "original")
	w := httptest.NewRecorder()

	server.handleScreenshotFile(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

// --- handleScreenshotContent ---

func TestHandleScreenshotContent_MissingPath(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/screenshots/content", nil)
	w := httptest.NewRecorder()

	server.handleScreenshotContent(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleScreenshotContent_ForbiddenPath(t *testing.T) {
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
