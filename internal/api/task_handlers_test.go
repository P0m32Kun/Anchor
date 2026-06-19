package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

func createTestTask(t *testing.T, queries *db.Queries, projectID string, status models.TaskStatus) *models.ScanTask {
	t.Helper()
	now := time.Now().UTC()
	task := &models.ScanTask{
		ID:        util.GenerateID(),
		ProjectID: projectID,
		Tool:      "nuclei",
		Status:    status,
		CreatedAt: now,
	}
	if err := queries.CreateScanTask(task); err != nil {
		t.Fatalf("create scan task: %v", err)
	}
	return task
}

// --- handleGetTask ---

func TestHandleGetTask_Found(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	task := createTestTask(t, queries, p.ID, models.TaskRunning)

	req := httptest.NewRequest(http.MethodGet, "/scan-tasks/"+task.ID, nil)
	req.SetPathValue("id", task.ID)
	w := httptest.NewRecorder()

	server.handleGetTask(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestHandleGetTask_NotFound(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/scan-tasks/nonexistent", nil)
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()

	server.handleGetTask(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

// --- handleCancelTask ---

func TestHandleCancelTask(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	task := createTestTask(t, queries, p.ID, models.TaskRunning)

	req := httptest.NewRequest(http.MethodPost, "/scan-tasks/"+task.ID+"/cancel", nil)
	req.SetPathValue("id", task.ID)
	w := httptest.NewRecorder()

	server.handleCancelTask(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

// --- handleListArtifacts ---

func TestHandleListArtifacts_Empty(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	task := createTestTask(t, queries, p.ID, models.TaskCompleted)

	req := httptest.NewRequest(http.MethodGet, "/tasks/"+task.ID+"/artifacts", nil)
	req.SetPathValue("id", task.ID)
	w := httptest.NewRecorder()

	server.handleListArtifacts(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

// --- handleGetArtifactContent ---

func TestHandleGetArtifactContent_MissingID(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/artifacts/content", nil)
	w := httptest.NewRecorder()

	server.handleGetArtifactContent(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleGetArtifactContent_NotFound(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/artifacts/content?id=nonexistent", nil)
	w := httptest.NewRecorder()

	server.handleGetArtifactContent(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

// --- handleGetArtifactContentRange ---

func TestHandleGetArtifactContentRange_MissingID(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/tasks/t1/artifacts/a1/content", nil)
	w := httptest.NewRecorder()

	server.handleGetArtifactContentRange(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleGetArtifactContentRange_NotFound(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/tasks/t1/artifacts/a1/content?id=nonexistent", nil)
	w := httptest.NewRecorder()

	server.handleGetArtifactContentRange(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

// --- parseIntParam ---

// --- handleGetArtifactContent with real artifact ---

func TestHandleGetArtifactContent_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)

	// Create a temp file for the artifact
	tmpDir := t.TempDir()
	artifactPath := filepath.Join(tmpDir, "output.txt")
	if err := os.WriteFile(artifactPath, []byte("hello world"), 0644); err != nil {
		t.Fatalf("write artifact: %v", err)
	}

	artifact := &models.RawArtifact{
		ID:              util.GenerateID(),
		ProjectID:       p.ID,
		Type:            models.ArtifactStdout,
		Path:            artifactPath,
		SHA256:          "abc123",
		Size:            11,
		RedactionStatus: "unchecked",
		CreatedAt:       time.Now().UTC(),
	}
	if err := queries.CreateRawArtifact(artifact); err != nil {
		t.Fatalf("create artifact: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/artifacts/content?id="+artifact.ID, nil)
	w := httptest.NewRecorder()

	server.handleGetArtifactContent(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

// --- handleGetArtifactContentRange with real artifact ---

func TestHandleGetArtifactContentRange_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)

	tmpDir := t.TempDir()
	artifactPath := filepath.Join(tmpDir, "output.txt")
	if err := os.WriteFile(artifactPath, []byte("hello world data"), 0644); err != nil {
		t.Fatalf("write artifact: %v", err)
	}

	artifact := &models.RawArtifact{
		ID:              util.GenerateID(),
		ProjectID:       p.ID,
		Type:            models.ArtifactStdout,
		Path:            artifactPath,
		SHA256:          "abc123",
		Size:            16,
		RedactionStatus: "unchecked",
		CreatedAt:       time.Now().UTC(),
	}
	if err := queries.CreateRawArtifact(artifact); err != nil {
		t.Fatalf("create artifact: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/tasks/t1/artifacts/a1/content?id="+artifact.ID, nil)
	w := httptest.NewRecorder()

	server.handleGetArtifactContentRange(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}
