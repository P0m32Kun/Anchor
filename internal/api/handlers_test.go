package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

func setupTestServer(t *testing.T) (*Server, *sql.DB, func()) {
	t.Helper()

	rawDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	if err := db.Migrate(rawDB); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	queries := db.New(rawDB)
	dataDir := t.TempDir()
	server := NewServer(queries, rawDB, dataDir)

	cleanup := func() {
		rawDB.Close()
	}

	return server, rawDB, cleanup
}

func TestCreateProject(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	reqBody := map[string]interface{}{
		"name":         "Test Project",
		"organization": "Test Org",
		"purpose":      "testing",
		"rate_limit":   10,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/projects", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleCreateProject(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	var project models.Project
	if err := json.NewDecoder(resp.Body).Decode(&project); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if project.Name != "Test Project" {
		t.Errorf("name = %q, want %q", project.Name, "Test Project")
	}
	if project.Organization != "Test Org" {
		t.Errorf("organization = %q, want %q", project.Organization, "Test Org")
	}
	if project.DefaultProfile != "standard" {
		t.Errorf("default_profile = %q, want standard", project.DefaultProfile)
	}
}

func TestListProjects(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	now := time.Now().UTC()

	for i := 0; i < 3; i++ {
		p := &models.Project{
			ID:             util.GenerateID(),
			Name:           string(rune('A' + i)),
			DefaultProfile: "standard",
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		if err := queries.CreateProject(p); err != nil {
			t.Fatalf("create project: %v", err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/projects", nil)
	w := httptest.NewRecorder()

	server.handleListProjects(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result PaginatedResponse[*models.Project]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(result.Data) != 3 {
		t.Errorf("len(projects) = %d, want 3", len(result.Data))
	}
	if result.Total != 3 {
		t.Errorf("total = %d, want 3", result.Total)
	}
}

func TestGetProject(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		server, rawDB, cleanup := setupTestServer(t)
		defer cleanup()

		queries := db.New(rawDB)
		p := &models.Project{
			ID:             util.GenerateID(),
			Name:           "Find Me",
			DefaultProfile: "standard",
			CreatedAt:      time.Now().UTC(),
			UpdatedAt:      time.Now().UTC(),
		}
		if err := queries.CreateProject(p); err != nil {
			t.Fatalf("create project: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/projects/"+p.ID, nil)
		req.SetPathValue("id", p.ID)
		w := httptest.NewRecorder()

		server.handleGetProject(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var got models.Project
		if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if got.ID != p.ID {
			t.Errorf("id = %q, want %q", got.ID, p.ID)
		}
	})

	t.Run("not found", func(t *testing.T) {
		server, _, cleanup := setupTestServer(t)
		defer cleanup()

		req := httptest.NewRequest(http.MethodGet, "/projects/nonexistent", nil)
		req.SetPathValue("id", "nonexistent")
		w := httptest.NewRecorder()

		server.handleGetProject(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
		}
	})
}

func TestDeleteProject(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := &models.Project{
		ID:             util.GenerateID(),
		Name:           "To Delete",
		DefaultProfile: "standard",
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	if err := queries.CreateProject(p); err != nil {
		t.Fatalf("create project: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/projects/"+p.ID, nil)
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleDeleteProject(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "deleted") {
		t.Errorf("response does not contain 'deleted': %s", string(body))
	}

	// Verify deletion.
	deleted, err := queries.GetProject(p.ID)
	if err != nil {
		t.Fatalf("get project after delete: %v", err)
	}
	if deleted != nil {
		t.Error("project still exists after deletion")
	}
}

func TestImportTargets(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)

	// Create a project.
	p := &models.Project{
		ID:             util.GenerateID(),
		Name:           "Import Test",
		DefaultProfile: "standard",
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	if err := queries.CreateProject(p); err != nil {
		t.Fatalf("create project: %v", err)
	}

	// Create a scope rule so import proceeds without confirmation.
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

	// Build multipart request with a TXT file.
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", "targets.txt")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	_, _ = part.Write([]byte("example.com\ntest.example.com\n"))
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/projects/"+p.ID+"/targets/import", &buf)
	req.SetPathValue("id", p.ID)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	server.handleImportTargets(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, body = %s", resp.StatusCode, string(body))
	}

	var result struct {
		Imported int `json:"imported"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.Imported != 2 {
		t.Errorf("imported = %d, want 2", result.Imported)
	}
}
