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

func createTestProject(t *testing.T, queries *db.Queries) *models.Project {
	t.Helper()
	p := &models.Project{
		ID: util.GenerateID(), Name: "Test", DefaultProfile: "standard",
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	if err := queries.CreateProject(p); err != nil {
		t.Fatalf("create project: %v", err)
	}
	return p
}

func createTestFinding(t *testing.T, queries *db.Queries, projectID string, status models.FindingStatus) *models.Finding {
	t.Helper()
	f := &models.Finding{
		ID: util.GenerateID(), ProjectID: projectID,
		SourceTool: "nuclei", SourceRuleID: "r1", DedupKey: util.GenerateID(),
		Title: "Test Finding", Severity: models.SeverityHigh,
		Confidence: 80, Priority: 70, Status: status,
		Summary: "test", Remediation: "fix",
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	if err := queries.CreateFinding(f); err != nil {
		t.Fatalf("create finding: %v", err)
	}
	return f
}

// --- handleListFindings ---

func TestHandleListFindings_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	createTestFinding(t, queries, p.ID, models.FindingConfirmed)
	createTestFinding(t, queries, p.ID, models.FindingNew)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+p.ID+"/findings", nil)
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleListFindings(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result PaginatedResponse[*models.Finding]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result.Data) != 2 {
		t.Errorf("len = %d, want 2", len(result.Data))
	}
	if result.Total != 2 {
		t.Errorf("total = %d, want 2", result.Total)
	}
}

func TestHandleListFindings_FilterByStatus(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	createTestFinding(t, queries, p.ID, models.FindingConfirmed)
	createTestFinding(t, queries, p.ID, models.FindingNew)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+p.ID+"/findings?status=confirmed", nil)
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleListFindings(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	var result PaginatedResponse[*models.Finding]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result.Data) != 1 {
		t.Errorf("len = %d, want 1", len(result.Data))
	}
	if result.Data[0].Status != models.FindingConfirmed {
		t.Errorf("status = %q, want confirmed", result.Data[0].Status)
	}
}

// --- handleGetFinding ---

func TestHandleGetFinding_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	f := createTestFinding(t, queries, p.ID, models.FindingConfirmed)

	req := httptest.NewRequest(http.MethodGet, "/findings/"+f.ID, nil)
	req.SetPathValue("id", f.ID)
	w := httptest.NewRecorder()

	server.handleGetFinding(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result struct {
		Finding  *models.Finding   `json:"finding"`
		Evidence []*models.Evidence `json:"evidence"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Finding == nil {
		t.Fatal("finding is nil")
	}
	if result.Finding.ID != f.ID {
		t.Errorf("id = %q, want %q", result.Finding.ID, f.ID)
	}
	if result.Finding.Title != f.Title {
		t.Errorf("title = %q, want %q", result.Finding.Title, f.Title)
	}
}

func TestHandleGetFinding_NotFound(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/findings/nonexistent", nil)
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()

	server.handleGetFinding(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

// --- handlePatchFindingStatus ---

func TestHandlePatchFindingStatus_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	f := createTestFinding(t, queries, p.ID, models.FindingNew)

	body, _ := json.Marshal(map[string]string{"status": "confirmed"})
	req := httptest.NewRequest(http.MethodPatch, "/findings/"+f.ID+"/status", bytes.NewReader(body))
	req.SetPathValue("id", f.ID)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handlePatchFindingStatus(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// Verify DB was updated.
	updated, _ := db.New(rawDB).GetFinding(f.ID)
	if updated.Status != models.FindingConfirmed {
		t.Errorf("db status = %q, want confirmed", updated.Status)
	}
}

func TestHandlePatchFindingStatus_InvalidStatus(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	f := createTestFinding(t, queries, p.ID, models.FindingNew)

	body, _ := json.Marshal(map[string]string{"status": "bogus"})
	req := httptest.NewRequest(http.MethodPatch, "/findings/"+f.ID+"/status", bytes.NewReader(body))
	req.SetPathValue("id", f.ID)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handlePatchFindingStatus(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandlePatchFindingStatus_NotFound(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	body, _ := json.Marshal(map[string]string{"status": "confirmed"})
	req := httptest.NewRequest(http.MethodPatch, "/findings/nonexistent/status", bytes.NewReader(body))
	req.SetPathValue("id", "nonexistent")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handlePatchFindingStatus(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

// --- handleAddEvidence ---

func TestHandleAddEvidence_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	f := createTestFinding(t, queries, p.ID, models.FindingConfirmed)

	body, _ := json.Marshal(map[string]string{
		"type":    "note",
		"excerpt": "manual verification done",
	})
	req := httptest.NewRequest(http.MethodPost, "/findings/"+f.ID+"/evidence", bytes.NewReader(body))
	req.SetPathValue("id", f.ID)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleAddEvidence(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	// Verify evidence was created.
	evList, _ := queries.ListEvidenceByFinding(f.ID)
	if len(evList) != 1 {
		t.Fatalf("evidence count = %d, want 1", len(evList))
	}
	if evList[0].Type != models.EvidenceNote {
		t.Errorf("type = %q, want note", evList[0].Type)
	}
}

func TestHandleAddEvidence_MissingType(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	f := createTestFinding(t, queries, p.ID, models.FindingConfirmed)

	body, _ := json.Marshal(map[string]string{"excerpt": "test"})
	req := httptest.NewRequest(http.MethodPost, "/findings/"+f.ID+"/evidence", bytes.NewReader(body))
	req.SetPathValue("id", f.ID)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleAddEvidence(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}
