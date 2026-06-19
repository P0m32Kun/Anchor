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

func createTestFindingTemplate(t *testing.T, queries *db.Queries, sourceTool, matchKey string) *models.FindingTemplate {
	t.Helper()
	now := time.Now().UTC()
	ft := &models.FindingTemplate{
		ID:         util.GenerateID(),
		SourceTool: sourceTool,
		MatchKey:   matchKey,
		MatchKeys:  []string{matchKey},
		Title:      "Test Template",
		Severity:   "high",
		Summary:    "test summary",
		Enabled:    true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := queries.CreateFindingTemplate(ft); err != nil {
		t.Fatalf("create finding template: %v", err)
	}
	return ft
}

// --- handleListFindingTemplates ---

func TestHandleListFindingTemplates_All(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	createTestFindingTemplate(t, queries, "nuclei", "CVE-2024-0001")
	createTestFindingTemplate(t, queries, "httpx", "tech-detect:nginx")

	req := httptest.NewRequest(http.MethodGet, "/finding-templates", nil)
	w := httptest.NewRecorder()

	server.handleListFindingTemplates(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var list []*models.FindingTemplate
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("len = %d, want 2", len(list))
	}
}

func TestHandleListFindingTemplates_FilterBySourceTool(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	createTestFindingTemplate(t, queries, "nuclei", "CVE-2024-0001")
	createTestFindingTemplate(t, queries, "httpx", "tech-detect:nginx")

	req := httptest.NewRequest(http.MethodGet, "/finding-templates?source_tool=nuclei", nil)
	w := httptest.NewRecorder()

	server.handleListFindingTemplates(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var list []*models.FindingTemplate
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("len = %d, want 1", len(list))
	}
	if list[0].SourceTool != "nuclei" {
		t.Errorf("source_tool = %q, want nuclei", list[0].SourceTool)
	}
}

// --- handleCreateFindingTemplate ---

func TestHandleCreateFindingTemplate_Success(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	payload := findingTemplatePayload{
		SourceTool: strPtr("nuclei"),
		MatchKeys:  &[]string{"CVE-2024-1234"},
		Title:      strPtr("XSS Vulnerability"),
		Severity:   strPtr("high"),
		Summary:    strPtr("Cross-site scripting found"),
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/finding-templates", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleCreateFindingTemplate(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	var created models.FindingTemplate
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if created.SourceTool != "nuclei" {
		t.Errorf("source_tool = %q, want nuclei", created.SourceTool)
	}
	if created.Title != "XSS Vulnerability" {
		t.Errorf("title = %q, want XSS Vulnerability", created.Title)
	}
	if !created.Enabled {
		t.Error("expected enabled=true by default")
	}
}

func TestHandleCreateFindingTemplate_MissingSourceTool(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	payload := findingTemplatePayload{
		MatchKeys: &[]string{"CVE-2024-1234"},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/finding-templates", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleCreateFindingTemplate(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleCreateFindingTemplate_MissingMatchKeys(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	payload := findingTemplatePayload{
		SourceTool: strPtr("nuclei"),
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/finding-templates", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleCreateFindingTemplate(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleCreateFindingTemplate_InvalidSeverity(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	payload := findingTemplatePayload{
		SourceTool: strPtr("nuclei"),
		MatchKeys:  &[]string{"CVE-2024-1234"},
		Severity:   strPtr("invalid"),
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/finding-templates", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleCreateFindingTemplate(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleCreateFindingTemplate_InvalidBody(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/finding-templates", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleCreateFindingTemplate(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleCreateFindingTemplate_DuplicateKey(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	createTestFindingTemplate(t, queries, "nuclei", "CVE-2024-0001")

	payload := findingTemplatePayload{
		SourceTool: strPtr("nuclei"),
		MatchKeys:  &[]string{"CVE-2024-0001"},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/finding-templates", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleCreateFindingTemplate(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusConflict)
	}
}

// --- handleGetFindingTemplate ---

func TestHandleGetFindingTemplate_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	ft := createTestFindingTemplate(t, queries, "nuclei", "CVE-2024-0001")

	req := httptest.NewRequest(http.MethodGet, "/finding-templates/"+ft.ID, nil)
	req.SetPathValue("id", ft.ID)
	w := httptest.NewRecorder()

	server.handleGetFindingTemplate(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var got models.FindingTemplate
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.ID != ft.ID {
		t.Errorf("id = %q, want %q", got.ID, ft.ID)
	}
}

func TestHandleGetFindingTemplate_NotFound(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/finding-templates/nonexistent", nil)
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()

	server.handleGetFindingTemplate(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

// --- handlePatchFindingTemplate ---

func TestHandlePatchFindingTemplate_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	ft := createTestFindingTemplate(t, queries, "nuclei", "CVE-2024-0001")

	newTitle := "Updated Title"
	payload := findingTemplatePayload{Title: &newTitle}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPatch, "/finding-templates/"+ft.ID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", ft.ID)
	w := httptest.NewRecorder()

	server.handlePatchFindingTemplate(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var updated models.FindingTemplate
	if err := json.NewDecoder(resp.Body).Decode(&updated); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if updated.Title != "Updated Title" {
		t.Errorf("title = %q, want Updated Title", updated.Title)
	}
}

func TestHandlePatchFindingTemplate_NotFound(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	newTitle := "Updated"
	payload := findingTemplatePayload{Title: &newTitle}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPatch, "/finding-templates/nonexistent", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()

	server.handlePatchFindingTemplate(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestHandlePatchFindingTemplate_InvalidSeverity(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	ft := createTestFindingTemplate(t, queries, "nuclei", "CVE-2024-0001")

	badSev := "invalid"
	payload := findingTemplatePayload{Severity: &badSev}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPatch, "/finding-templates/"+ft.ID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", ft.ID)
	w := httptest.NewRecorder()

	server.handlePatchFindingTemplate(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandlePatchFindingTemplate_EmptySourceTool(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	ft := createTestFindingTemplate(t, queries, "nuclei", "CVE-2024-0001")

	emptyStr := ""
	payload := findingTemplatePayload{SourceTool: &emptyStr}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPatch, "/finding-templates/"+ft.ID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", ft.ID)
	w := httptest.NewRecorder()

	server.handlePatchFindingTemplate(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandlePatchFindingTemplate_EmptyMatchKeys(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	ft := createTestFindingTemplate(t, queries, "nuclei", "CVE-2024-0001")

	emptyKeys := []string{}
	payload := findingTemplatePayload{MatchKeys: &emptyKeys}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPatch, "/finding-templates/"+ft.ID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", ft.ID)
	w := httptest.NewRecorder()

	server.handlePatchFindingTemplate(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandlePatchFindingTemplate_InvalidBody(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	ft := createTestFindingTemplate(t, queries, "nuclei", "CVE-2024-0001")

	req := httptest.NewRequest(http.MethodPatch, "/finding-templates/"+ft.ID, bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", ft.ID)
	w := httptest.NewRecorder()

	server.handlePatchFindingTemplate(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandlePatchFindingTemplate_Disable(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	ft := createTestFindingTemplate(t, queries, "nuclei", "CVE-2024-0001")

	enabled := false
	payload := findingTemplatePayload{Enabled: &enabled}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPatch, "/finding-templates/"+ft.ID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", ft.ID)
	w := httptest.NewRecorder()

	server.handlePatchFindingTemplate(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var updated models.FindingTemplate
	json.NewDecoder(resp.Body).Decode(&updated)
	if updated.Enabled {
		t.Error("expected enabled=false after patch")
	}
}

// --- handleDeleteFindingTemplate ---

func TestHandleDeleteFindingTemplate_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	ft := createTestFindingTemplate(t, queries, "nuclei", "CVE-2024-0001")

	req := httptest.NewRequest(http.MethodDelete, "/finding-templates/"+ft.ID, nil)
	req.SetPathValue("id", ft.ID)
	w := httptest.NewRecorder()

	server.handleDeleteFindingTemplate(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// Verify deleted
	got, _ := queries.GetFindingTemplate(ft.ID)
	if got != nil {
		t.Error("template still exists after delete")
	}
}

func TestHandleDeleteFindingTemplate_NotFound(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodDelete, "/finding-templates/nonexistent", nil)
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()

	server.handleDeleteFindingTemplate(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	// Delete is idempotent - no error for nonexistent
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

// --- handleAcceptFindingTemplateUpstream ---

func TestHandleAcceptFindingTemplateUpstream_NotBuiltin(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	ft := createTestFindingTemplate(t, queries, "nuclei", "CVE-2024-0001")

	req := httptest.NewRequest(http.MethodPost, "/finding-templates/"+ft.ID+"/accept-upstream", nil)
	req.SetPathValue("id", ft.ID)
	w := httptest.NewRecorder()

	server.handleAcceptFindingTemplateUpstream(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleAcceptFindingTemplateUpstream_NotFound(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/finding-templates/nonexistent/accept-upstream", nil)
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()

	server.handleAcceptFindingTemplateUpstream(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestHandleAcceptFindingTemplateUpstream_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	now := time.Now().UTC()

	// Create a builtin template with a payload
	payloadJSON, _ := json.Marshal(db.SeedFindingTemplate{
		SourceTool: "nuclei",
		MatchKey:   "CVE-2024-UPSTREAM",
		MatchKeys:  []string{"CVE-2024-UPSTREAM"},
		Title:      "Upstream Title",
		Severity:   "critical",
		Summary:    "upstream summary",
		Enabled:    boolPtr(true),
	})
	ft := &models.FindingTemplate{
		ID:             util.GenerateID(),
		SourceTool:     "nuclei",
		MatchKey:       "CVE-2024-0001",
		MatchKeys:      []string{"CVE-2024-0001"},
		Title:          "Local Title",
		Severity:       "high",
		Enabled:        true,
		IsBuiltin:      true,
		UserModified:   true,
		BuiltinPayload: string(payloadJSON),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := queries.CreateFindingTemplate(ft); err != nil {
		t.Fatalf("create template: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/finding-templates/"+ft.ID+"/accept-upstream", nil)
	req.SetPathValue("id", ft.ID)
	w := httptest.NewRecorder()

	server.handleAcceptFindingTemplateUpstream(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var updated models.FindingTemplate
	json.NewDecoder(resp.Body).Decode(&updated)
	if updated.Title != "Upstream Title" {
		t.Errorf("title = %q, want Upstream Title", updated.Title)
	}
	if updated.UserModified {
		t.Error("expected user_modified=false after accept upstream")
	}
}

// --- handleExportFindingTemplates ---

func TestHandleExportFindingTemplates_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	createTestFindingTemplate(t, queries, "nuclei", "CVE-2024-0001")
	createTestFindingTemplate(t, queries, "httpx", "tech-detect:nginx")

	req := httptest.NewRequest(http.MethodGet, "/finding-templates/export", nil)
	w := httptest.NewRecorder()

	server.handleExportFindingTemplates(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	ct := resp.Header.Get("Content-Type")
	if ct != "application/json; charset=utf-8" {
		t.Errorf("content-type = %q, want application/json; charset=utf-8", ct)
	}

	cd := resp.Header.Get("Content-Disposition")
	if cd != `attachment; filename="vuln-templates.json"` {
		t.Errorf("content-disposition = %q", cd)
	}

	var seeds []db.SeedFindingTemplate
	if err := json.NewDecoder(resp.Body).Decode(&seeds); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(seeds) != 2 {
		t.Errorf("len = %d, want 2", len(seeds))
	}
}

// --- helpers ---

func strPtr(s string) *string { return &s }
func boolPtr(b bool) *bool    { return &b }
