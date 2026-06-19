package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/P0m32Kun/Anchor/internal/db"
)

// --- handleGetWatchConfig ---

func TestHandleGetWatchConfig_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+p.ID+"/watch-config", nil)
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleGetWatchConfig(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result["id"] != p.ID {
		t.Errorf("id = %v, want %q", result["id"], p.ID)
	}
}

func TestHandleGetWatchConfig_NotFound(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/projects/nonexistent/watch-config", nil)
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()

	server.handleGetWatchConfig(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

// --- handleUpdateWatchConfig ---

func TestHandleUpdateWatchConfig_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)

	body, _ := json.Marshal(map[string]interface{}{
		"watch_enabled":        true,
		"watch_interval_hours": 12,
		"watch_passive_only":   true,
	})

	req := httptest.NewRequest(http.MethodPatch, "/projects/"+p.ID+"/watch-config", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleUpdateWatchConfig(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result["watch_enabled"] != true {
		t.Errorf("watch_enabled = %v, want true", result["watch_enabled"])
	}
	if result["watch_passive_only"] != true {
		t.Errorf("watch_passive_only = %v, want true", result["watch_passive_only"])
	}
}

func TestHandleUpdateWatchConfig_DefaultInterval(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)

	// Watch interval hours = 0 should default to 24
	body, _ := json.Marshal(map[string]interface{}{
		"watch_enabled":        true,
		"watch_interval_hours": 0,
	})

	req := httptest.NewRequest(http.MethodPatch, "/projects/"+p.ID+"/watch-config", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleUpdateWatchConfig(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["watch_interval_hours"].(float64) != 24 {
		t.Errorf("watch_interval_hours = %v, want 24 (default)", result["watch_interval_hours"])
	}
}

func TestHandleUpdateWatchConfig_InvalidBody(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)

	req := httptest.NewRequest(http.MethodPatch, "/projects/"+p.ID+"/watch-config", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleUpdateWatchConfig(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

// --- handleDeleteTarget ---

func TestHandleDeleteTarget_NotFound(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	_ = createTestProject(t, queries)

	req := httptest.NewRequest(http.MethodDelete, "/projects/x/targets/nonexistent", nil)
	req.SetPathValue("id", "x")
	req.SetPathValue("targetId", "nonexistent")
	w := httptest.NewRecorder()

	server.handleDeleteTarget(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	// Handler returns 200 even for nonexistent targets (delete is idempotent).
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

// --- handleListWebEndpointsByProject ---

func TestHandleListWebEndpointsByProject_Empty(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+p.ID+"/web-endpoints", nil)
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleListWebEndpointsByProject(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result PaginatedResponse[interface{}]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Total != 0 {
		t.Errorf("total = %d, want 0", result.Total)
	}
}

// --- handleGetAssetLineage ---

func TestHandleGetAssetLineage_MissingID(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/assets//lineage", nil)
	req.SetPathValue("id", "")
	w := httptest.NewRecorder()

	server.handleGetAssetLineage(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleGetAssetLineage_NotFound(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/assets/nonexistent/lineage", nil)
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()

	server.handleGetAssetLineage(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}
