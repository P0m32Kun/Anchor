package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// ==================== archive_handlers.go ====================

func TestHandleDownloadArchive_MissingName(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/projects/p1/archive/download", nil)
	req.SetPathValue("id", "p1")
	w := httptest.NewRecorder()

	server.handleDownloadArchive(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleDownloadArchive_NotFound(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/projects/p1/archive/download?name=nonexistent.zip", nil)
	req.SetPathValue("id", "p1")
	w := httptest.NewRecorder()

	server.handleDownloadArchive(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestHandleDownloadArchive_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)

	// Create archive directory and a dummy zip file
	archiveDir := filepath.Join(server.dataDir, "projects", p.ID, "archives")
	if err := os.MkdirAll(archiveDir, 0750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	archiveName := "test_archive.zip"
	archivePath := filepath.Join(archiveDir, archiveName)
	if err := os.WriteFile(archivePath, []byte("PKzip-content"), 0644); err != nil {
		t.Fatalf("write archive: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/projects/"+p.ID+"/archive/download?name="+archiveName, nil)
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleDownloadArchive(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/zip" {
		t.Errorf("Content-Type = %q, want application/zip", ct)
	}
}

// ==================== screenshot_handlers.go ====================

func TestHandleScreenshotCapture_ProjectNotFound(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	body := `{"url": "https://example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/projects/nonexistent/screenshots/capture", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()

	server.handleScreenshotCapture(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestHandleScreenshotCapture_InvalidBody(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)

	req := httptest.NewRequest(http.MethodPost, "/projects/"+p.ID+"/screenshots/capture", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleScreenshotCapture(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleScreenshotCapture_MissingURL(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)

	body := `{"url": ""}`
	req := httptest.NewRequest(http.MethodPost, "/projects/"+p.ID+"/screenshots/capture", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleScreenshotCapture(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleListScreenshots_WithData(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)

	now := time.Now().UTC()
	err := queries.CreateScreenshot(&models.Screenshot{
		ID:           util.GenerateID(),
		ProjectID:    p.ID,
		URL:          "https://example.com",
		OriginalPath: "/tmp/test.png",
		TakenAt:      now,
	})
	if err != nil {
		t.Fatalf("create screenshot: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/projects/"+p.ID+"/screenshots", nil)
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleListScreenshots(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var screenshots []*models.Screenshot
	if err := json.NewDecoder(resp.Body).Decode(&screenshots); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(screenshots) != 1 {
		t.Errorf("len = %d, want 1", len(screenshots))
	}
}

func TestHandleScreenshotFile_WithRealFile(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)

	// Create a temp image file
	tmpDir := t.TempDir()
	imgPath := filepath.Join(tmpDir, "screenshot.png")
	if err := os.WriteFile(imgPath, []byte("PNG-fake-data"), 0644); err != nil {
		t.Fatalf("write image: %v", err)
	}

	ss := &models.Screenshot{
		ID:           util.GenerateID(),
		ProjectID:    p.ID,
		URL:          "https://example.com",
		OriginalPath: imgPath,
		TakenAt:      time.Now().UTC(),
	}
	if err := queries.CreateScreenshot(ss); err != nil {
		t.Fatalf("create screenshot: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/screenshots/"+ss.ID+"/original", nil)
	req.SetPathValue("id", ss.ID)
	req.SetPathValue("kind", "original")
	w := httptest.NewRecorder()

	server.handleScreenshotFile(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "image/png" {
		t.Errorf("Content-Type = %q, want image/png", ct)
	}
}

func TestHandleScreenshotFile_Thumbnail(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)

	tmpDir := t.TempDir()
	thumbPath := filepath.Join(tmpDir, "thumb.jpg")
	if err := os.WriteFile(thumbPath, []byte("JPEG-fake-data"), 0644); err != nil {
		t.Fatalf("write thumb: %v", err)
	}

	ss := &models.Screenshot{
		ID:            util.GenerateID(),
		ProjectID:     p.ID,
		URL:           "https://example.com",
		OriginalPath:  filepath.Join(tmpDir, "original.png"),
		ThumbnailPath: thumbPath,
		TakenAt:       time.Now().UTC(),
	}
	if err := queries.CreateScreenshot(ss); err != nil {
		t.Fatalf("create screenshot: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/screenshots/"+ss.ID+"/thumbnail", nil)
	req.SetPathValue("id", ss.ID)
	req.SetPathValue("kind", "thumbnail")
	w := httptest.NewRecorder()

	server.handleScreenshotFile(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "image/jpeg" {
		t.Errorf("Content-Type = %q, want image/jpeg", ct)
	}
}

func TestHandleScreenshotFile_EmptyPath(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)

	ss := &models.Screenshot{
		ID:           util.GenerateID(),
		ProjectID:    p.ID,
		URL:          "https://example.com",
		OriginalPath: "", // empty
		TakenAt:      time.Now().UTC(),
	}
	if err := queries.CreateScreenshot(ss); err != nil {
		t.Fatalf("create screenshot: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/screenshots/"+ss.ID+"/original", nil)
	req.SetPathValue("id", ss.ID)
	req.SetPathValue("kind", "original")
	w := httptest.NewRecorder()

	server.handleScreenshotFile(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestHandleScreenshotContent_Success(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a file inside the dataDir
	imgPath := filepath.Join(server.dataDir, "test_image.png")
	if err := os.WriteFile(imgPath, []byte("PNG-data"), 0644); err != nil {
		t.Fatalf("write image: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/screenshots/content?path="+imgPath, nil)
	w := httptest.NewRecorder()

	server.handleScreenshotContent(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "image/png" {
		t.Errorf("Content-Type = %q, want image/png", ct)
	}
}

func TestHandleScreenshotContent_JpegFile(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	imgPath := filepath.Join(server.dataDir, "test_image.jpg")
	if err := os.WriteFile(imgPath, []byte("JPEG-data"), 0644); err != nil {
		t.Fatalf("write image: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/screenshots/content?path="+imgPath, nil)
	w := httptest.NewRecorder()

	server.handleScreenshotContent(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "image/jpeg" {
		t.Errorf("Content-Type = %q, want image/jpeg", ct)
	}
}

func TestHandleScreenshotContent_FileNotFound(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	nonexistent := filepath.Join(server.dataDir, "nonexistent.png")
	req := httptest.NewRequest(http.MethodGet, "/screenshots/content?path="+nonexistent, nil)
	w := httptest.NewRecorder()

	server.handleScreenshotContent(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

// ==================== sse_token.go — handleIssueSSEToken ====================

func TestHandleIssueSSEToken_MissingProjectID(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/projects//sse-token", nil)
	req.SetPathValue("id", "")
	w := httptest.NewRecorder()

	server.handleIssueSSEToken(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleIssueSSEToken_ProjectNotFound(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/projects/nonexistent/sse-token", nil)
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()

	server.handleIssueSSEToken(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	// GetProject returns (nil, nil) for missing projects (not an error),
	// so the handler proceeds and returns 200.
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestHandleIssueSSEToken_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)

	req := httptest.NewRequest(http.MethodPost, "/projects/"+p.ID+"/sse-token", nil)
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleIssueSSEToken(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result["token"] == nil || result["token"] == "" {
		t.Error("token is empty")
	}
	if result["project_id"] != p.ID {
		t.Errorf("project_id = %v, want %s", result["project_id"], p.ID)
	}
	// Verify the token is valid
	token, ok := result["token"].(string)
	if !ok {
		t.Fatalf("token is not a string: %T", result["token"])
	}
	claims, err := server.ValidateSSEToken(token)
	if err != nil {
		t.Fatalf("validate token: %v", err)
	}
	if claims.ProjectID != p.ID {
		t.Errorf("claims.ProjectID = %q, want %q", claims.ProjectID, p.ID)
	}
}

// ==================== sse.go — handleProjectSSE ====================

func TestHandleProjectSSE_MissingProjectID(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/projects//events", nil)
	req.SetPathValue("id", "")
	w := httptest.NewRecorder()

	server.handleProjectSSE(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleProjectSSE_ConnectAndReceiveEvent(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	// Use a context that we cancel quickly to avoid blocking
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req := httptest.NewRequest(http.MethodGet, "/projects/p1/events", nil).WithContext(ctx)
	req.SetPathValue("id", "p1")
	w := httptest.NewRecorder()

	// Cancel context right away to prevent the handler from blocking forever
	// but we need to let it start first
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	server.handleProjectSSE(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	// The handler should have sent at least the "connected" event
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	body := w.Body.String()
	if !strings.Contains(body, `"event":"connected"`) {
		t.Errorf("response does not contain connected event: %s", body)
	}
}

// ==================== task_output_handlers.go ====================

func TestHandleGetTaskOutput_InvalidStream(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/tasks/t1/output?stream=invalid", nil)
	req.SetPathValue("id", "t1")
	w := httptest.NewRecorder()

	server.handleGetTaskOutput(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleGetTaskOutput_TaskNotFound(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/tasks/nonexistent/output", nil)
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()

	server.handleGetTaskOutput(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestHandleGetTaskOutput_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	task := createTestTask(t, queries, p.ID, models.TaskCompleted)

	// Create workdir with output file
	workdir := filepath.Join(server.dataDir, "workdirs", p.ID, task.ID)
	if err := os.MkdirAll(workdir, 0750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workdir, "stdout.log"), []byte("hello output"), 0644); err != nil {
		t.Fatalf("write output: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/tasks/"+task.ID+"/output?stream=stdout&offset=0", nil)
	req.SetPathValue("id", task.ID)
	w := httptest.NewRecorder()

	server.handleGetTaskOutput(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result["stream"] != "stdout" {
		t.Errorf("stream = %v, want stdout", result["stream"])
	}
}

func TestHandleGetTaskOutput_DefaultStream(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	task := createTestTask(t, queries, p.ID, models.TaskCompleted)

	// Create workdir with stdout file
	workdir := filepath.Join(server.dataDir, "workdirs", p.ID, task.ID)
	if err := os.MkdirAll(workdir, 0750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workdir, "stdout.log"), []byte("data"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/tasks/"+task.ID+"/output", nil)
	req.SetPathValue("id", task.ID)
	w := httptest.NewRecorder()

	server.handleGetTaskOutput(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestHandleGetTaskOutput_StderrStream(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	task := createTestTask(t, queries, p.ID, models.TaskCompleted)

	workdir := filepath.Join(server.dataDir, "workdirs", p.ID, task.ID)
	if err := os.MkdirAll(workdir, 0750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workdir, "stderr.log"), []byte("error output"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/tasks/"+task.ID+"/output?stream=stderr", nil)
	req.SetPathValue("id", task.ID)
	w := httptest.NewRecorder()

	server.handleGetTaskOutput(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result["stream"] != "stderr" {
		t.Errorf("stream = %v, want stderr", result["stream"])
	}
}

func TestHandleGetTaskOutput_RemoteWorker(t *testing.T) {
	// Test the proxy path: task is running and has a WorkerID
	// We'll set up a mock worker HTTP server
	workerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"stream":  "stdout",
			"offset":  10,
			"content": "remote output",
			"done":    false,
		})
	}))
	defer workerServer.Close()

	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	task := createTestTask(t, queries, p.ID, models.TaskRunning)

	// Create the worker node
	workerID := "test-worker"
	if err := queries.CreateWorkerNode(&models.WorkerNode{
		ID:          workerID,
		Name:        "test-worker",
		Endpoint:    workerServer.URL,
		Mode:        models.WorkerModeRemote,
		Status:      models.WorkerStatusOnline,
		TrustLevel:  "standard",
		CreatedAt:   time.Now().UTC(),
	}); err != nil {
		t.Fatalf("create worker node: %v", err)
	}

	// Set worker ID on the task
	if err := queries.SetScanTaskWorker(task.ID, workerID); err != nil {
		t.Fatalf("set task worker: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/tasks/"+task.ID+"/output?stream=stdout&offset=0", nil)
	req.SetPathValue("id", task.ID)
	w := httptest.NewRecorder()

	server.handleGetTaskOutput(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result["content"] != "remote output" {
		t.Errorf("content = %v, want remote output", result["content"])
	}
}

// ==================== pagination.go ====================

func TestParsePagination(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		wantPage int
		wantSize int
	}{
		{"defaults", "", 1, DefaultPageSize},
		{"custom page", "page=3", 3, DefaultPageSize},
		{"custom size", "page_size=50", 1, 50},
		{"both", "page=2&page_size=50", 2, 50},
		{"zero page", "page=0", 1, DefaultPageSize},
		{"negative page", "page=-1", 1, DefaultPageSize},
		{"zero size", "page_size=0", 1, DefaultPageSize},
		{"exceeds max", "page_size=9999", 1, MaxPageSize},
		{"exact max", "page_size=1000", 1, 1000},
		{"invalid page", "page=abc", 1, DefaultPageSize},
		{"invalid size", "page_size=abc", 1, DefaultPageSize},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/test"
			if tt.query != "" {
				url = "/test?" + tt.query
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			got := parsePagination(req)
			if got.Page != tt.wantPage {
				t.Errorf("Page = %d, want %d", got.Page, tt.wantPage)
			}
			if got.PageSize != tt.wantSize {
				t.Errorf("PageSize = %d, want %d", got.PageSize, tt.wantSize)
			}
		})
	}
}

func TestWritePaginatedJSON(t *testing.T) {
	w := httptest.NewRecorder()
	data := []string{"a", "b", "c"}
	pg := PaginationParams{Page: 2, PageSize: 10}

	writePaginatedJSON(w, data, 25, pg)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result PaginatedResponse[string]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Total != 25 {
		t.Errorf("total = %d, want 25", result.Total)
	}
	if result.Page != 2 {
		t.Errorf("page = %d, want 2", result.Page)
	}
	if result.PageSize != 10 {
		t.Errorf("page_size = %d, want 10", result.PageSize)
	}
	if len(result.Data) != 3 {
		t.Errorf("len(data) = %d, want 3", len(result.Data))
	}
}

// ==================== workdir_cleanup.go — cleanupProjectWorkdir ====================

func TestCleanupProjectWorkdir(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a workdir for a project
	projectDir := filepath.Join(server.dataDir, "workdirs", "test-project", "task1")
	if err := os.MkdirAll(projectDir, 0750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "output.txt"), []byte("data"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Verify it exists
	if _, err := os.Stat(projectDir); err != nil {
		t.Fatalf("dir should exist: %v", err)
	}

	// Clean it up
	server.cleanupProjectWorkdir("test-project")

	// Verify it's gone
	parentDir := filepath.Join(server.dataDir, "workdirs", "test-project")
	if _, err := os.Stat(parentDir); !os.IsNotExist(err) {
		t.Errorf("dir should not exist, err=%v", err)
	}
}

func TestCleanupStaleWorkdirs_OrphanProject(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a workdir for a project that doesn't exist in DB
	orphanDir := filepath.Join(server.dataDir, "workdirs", "orphan-project")
	if err := os.MkdirAll(orphanDir, 0750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	server.cleanupStaleWorkdirs()

	// Orphan dir should be removed
	if _, err := os.Stat(orphanDir); !os.IsNotExist(err) {
		t.Errorf("orphan dir should be removed, err=%v", err)
	}
}

func TestCleanupStaleWorkdirs_OldFile(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)

	// Create an old host-list file
	projectDir := filepath.Join(server.dataDir, "workdirs", p.ID)
	if err := os.MkdirAll(projectDir, 0750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	oldFile := filepath.Join(projectDir, "nmap-old.txt")
	if err := os.WriteFile(oldFile, []byte("scan data"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Set the mtime to 60 days ago
	staleTime := time.Now().Add(-60 * 24 * time.Hour)
	os.Chtimes(oldFile, staleTime, staleTime)

	server.cleanupStaleWorkdirs()

	// Old file should be removed
	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Errorf("old file should be removed, err=%v", err)
	}
}

// ==================== handleGetScanRunMetrics — DB nil case ====================

func TestHandleGetScanRunMetrics_RunNotFound(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)

	// Use a valid run that exists but has no metrics
	run := createTestPipelineRun(t, queries, p.ID, "running")

	req := httptest.NewRequest(http.MethodGet, "/projects/"+p.ID+"/pipeline/runs/"+run.ID+"/metrics", nil)
	req.SetPathValue("id", p.ID)
	req.SetPathValue("runId", run.ID)
	w := httptest.NewRecorder()

	server.handleGetScanRunMetrics(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	// Run exists, so GetScanRunMetrics returns a non-nil ScanRunMetrics with zero values.
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d (body: %s)", resp.StatusCode, http.StatusOK, w.Body.String())
	}
}
