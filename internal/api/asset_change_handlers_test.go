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

func createTestAssetChange(t *testing.T, queries *db.Queries, projectID, assetID string) *models.AssetChange {
	t.Helper()
	change := &models.AssetChange{
		ID:            util.GenerateID(),
		ProjectID:     projectID,
		RunID:         util.GenerateID(),
		AssetID:       assetID,
		AssetValue:    "example.com",
		AssetType:     "domain",
		ChangeType:    models.ChangeTypeAssetNew,
		ChangeSummary: "new asset discovered",
		DetailJSON:    "{}",
		Severity:      "info",
		CreatedAt:     time.Now().UTC(),
	}
	if err := queries.CreateAssetChange(change); err != nil {
		t.Fatalf("create asset change: %v", err)
	}
	return change
}

// --- handleListAssetChanges ---

func TestHandleListAssetChanges_Empty(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/projects/p1/asset-changes", nil)
	req.SetPathValue("id", "p1")
	w := httptest.NewRecorder()

	server.handleListAssetChanges(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var changes []*models.AssetChange
	if err := json.NewDecoder(resp.Body).Decode(&changes); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(changes) != 0 {
		t.Errorf("len = %d, want 0", len(changes))
	}
}

func TestHandleListAssetChanges_WithData(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	createTestAssetChange(t, queries, p.ID, "asset1")
	createTestAssetChange(t, queries, p.ID, "asset2")

	req := httptest.NewRequest(http.MethodGet, "/projects/"+p.ID+"/asset-changes", nil)
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleListAssetChanges(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var changes []*models.AssetChange
	if err := json.NewDecoder(resp.Body).Decode(&changes); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(changes) != 2 {
		t.Errorf("len = %d, want 2", len(changes))
	}
}

func TestHandleListAssetChanges_WithPagination(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	createTestAssetChange(t, queries, p.ID, "asset1")
	createTestAssetChange(t, queries, p.ID, "asset2")

	req := httptest.NewRequest(http.MethodGet, "/projects/"+p.ID+"/asset-changes?limit=1&offset=0", nil)
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleListAssetChanges(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var changes []*models.AssetChange
	json.NewDecoder(resp.Body).Decode(&changes)
	if len(changes) != 1 {
		t.Errorf("len = %d, want 1", len(changes))
	}
}

func TestHandleListAssetChanges_MissingProjectID(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/projects//asset-changes", nil)
	req.SetPathValue("id", "")
	w := httptest.NewRecorder()

	server.handleListAssetChanges(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

// --- handleAssetChangeTimeline ---

func TestHandleAssetChangeTimeline_Empty(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/assets/asset1/changes", nil)
	req.SetPathValue("id", "asset1")
	w := httptest.NewRecorder()

	server.handleAssetChangeTimeline(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var changes []*models.AssetChange
	if err := json.NewDecoder(resp.Body).Decode(&changes); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(changes) != 0 {
		t.Errorf("len = %d, want 0", len(changes))
	}
}

func TestHandleAssetChangeTimeline_WithData(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	createTestAssetChange(t, queries, p.ID, "asset1")

	req := httptest.NewRequest(http.MethodGet, "/assets/asset1/changes", nil)
	req.SetPathValue("id", "asset1")
	w := httptest.NewRecorder()

	server.handleAssetChangeTimeline(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var changes []*models.AssetChange
	json.NewDecoder(resp.Body).Decode(&changes)
	if len(changes) != 1 {
		t.Errorf("len = %d, want 1", len(changes))
	}
}

func TestHandleAssetChangeTimeline_MissingAssetID(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/assets//changes", nil)
	req.SetPathValue("id", "")
	w := httptest.NewRecorder()

	server.handleAssetChangeTimeline(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}
