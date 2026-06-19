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

func createTestSignal(t *testing.T, queries *db.Queries, projectID, status, severity string) *models.Signal {
	t.Helper()
	now := time.Now().UTC()
	sig := &models.Signal{
		ID:          util.GenerateID(),
		ProjectID:   projectID,
		SourceKind:  models.SignalSourceKindFinding,
		SourceID:    util.GenerateID(),
		Title:       "Test Signal",
		Severity:    severity,
		Score:       80,
		ScopeStatus: "in_scope",
		Status:      status,
		Metadata:    "{}",
		FirstSeen:   now,
		LastSeen:    now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := queries.CreateSignal(sig); err != nil {
		t.Fatalf("create signal: %v", err)
	}
	return sig
}

// --- handleListSignals ---

func TestHandleListSignals_Empty(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/projects/p1/signals", nil)
	req.SetPathValue("id", "p1")
	w := httptest.NewRecorder()

	server.handleListSignals(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var signals []*models.Signal
	if err := json.NewDecoder(resp.Body).Decode(&signals); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(signals) != 0 {
		t.Errorf("len = %d, want 0", len(signals))
	}
}

func TestHandleListSignals_WithFilters(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	createTestSignal(t, queries, p.ID, models.SignalStatusNew, "high")
	createTestSignal(t, queries, p.ID, models.SignalStatusAcknowledged, "low")

	req := httptest.NewRequest(http.MethodGet, "/projects/"+p.ID+"/signals?status=new&severity=high", nil)
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleListSignals(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var signals []*models.Signal
	json.NewDecoder(resp.Body).Decode(&signals)
	if len(signals) != 1 {
		t.Errorf("len = %d, want 1", len(signals))
	}
}

func TestHandleListSignals_MissingProjectID(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/projects//signals", nil)
	req.SetPathValue("id", "")
	w := httptest.NewRecorder()

	server.handleListSignals(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleListSignals_SourceKindFilter(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	createTestSignal(t, queries, p.ID, models.SignalStatusNew, "high")

	req := httptest.NewRequest(http.MethodGet, "/projects/"+p.ID+"/signals?source_kind=finding", nil)
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleListSignals(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

// --- handleUpdateSignalStatus ---

func TestHandleUpdateSignalStatus_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	sig := createTestSignal(t, queries, p.ID, models.SignalStatusNew, "high")

	body, _ := json.Marshal(map[string]interface{}{
		"status": models.SignalStatusAcknowledged,
	})

	req := httptest.NewRequest(http.MethodPatch, "/signals/"+sig.ID+"/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", sig.ID)
	w := httptest.NewRecorder()

	server.handleUpdateSignalStatus(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNoContent)
	}
}

func TestHandleUpdateSignalStatus_InvalidStatus(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	body, _ := json.Marshal(map[string]interface{}{
		"status": "invalid",
	})

	req := httptest.NewRequest(http.MethodPatch, "/signals/s1/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "s1")
	w := httptest.NewRecorder()

	server.handleUpdateSignalStatus(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleUpdateSignalStatus_MissingID(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	body, _ := json.Marshal(map[string]interface{}{"status": "new"})
	req := httptest.NewRequest(http.MethodPatch, "/signals//status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "")
	w := httptest.NewRecorder()

	server.handleUpdateSignalStatus(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleUpdateSignalStatus_InvalidJSON(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPatch, "/signals/s1/status", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "s1")
	w := httptest.NewRecorder()

	server.handleUpdateSignalStatus(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

// --- handleBatchUpdateSignalStatus ---

func TestHandleBatchUpdateSignalStatus_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	sig1 := createTestSignal(t, queries, p.ID, models.SignalStatusNew, "high")
	sig2 := createTestSignal(t, queries, p.ID, models.SignalStatusNew, "low")

	body, _ := json.Marshal(map[string]interface{}{
		"ids":    []string{sig1.ID, sig2.ID},
		"status": models.SignalStatusResolved,
	})

	req := httptest.NewRequest(http.MethodPatch, "/signals/batch-status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleBatchUpdateSignalStatus(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result map[string]int
	json.NewDecoder(resp.Body).Decode(&result)
	if result["updated"] != 2 {
		t.Errorf("updated = %d, want 2", result["updated"])
	}
}

func TestHandleBatchUpdateSignalStatus_EmptyIDs(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	body, _ := json.Marshal(map[string]interface{}{
		"ids":    []string{},
		"status": "new",
	})

	req := httptest.NewRequest(http.MethodPatch, "/signals/batch-status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleBatchUpdateSignalStatus(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleBatchUpdateSignalStatus_InvalidStatus(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	body, _ := json.Marshal(map[string]interface{}{
		"ids":    []string{"s1"},
		"status": "invalid",
	})

	req := httptest.NewRequest(http.MethodPatch, "/signals/batch-status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleBatchUpdateSignalStatus(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleBatchUpdateSignalStatus_InvalidJSON(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPatch, "/signals/batch-status", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleBatchUpdateSignalStatus(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

// --- handleSignalStats ---

func TestHandleSignalStats_Empty(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/projects/p1/signals/stats", nil)
	req.SetPathValue("id", "p1")
	w := httptest.NewRecorder()

	server.handleSignalStats(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var stats map[string]int
	json.NewDecoder(resp.Body).Decode(&stats)
	if stats["total"] != 0 {
		t.Errorf("total = %d, want 0", stats["total"])
	}
}

func TestHandleSignalStats_WithData(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	createTestSignal(t, queries, p.ID, models.SignalStatusNew, "high")
	createTestSignal(t, queries, p.ID, models.SignalStatusAcknowledged, "low")
	createTestSignal(t, queries, p.ID, models.SignalStatusResolved, "medium")

	req := httptest.NewRequest(http.MethodGet, "/projects/"+p.ID+"/signals/stats", nil)
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleSignalStats(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var stats map[string]int
	json.NewDecoder(resp.Body).Decode(&stats)
	if stats["total"] != 3 {
		t.Errorf("total = %d, want 3", stats["total"])
	}
	if stats["new"] != 1 {
		t.Errorf("new = %d, want 1", stats["new"])
	}
	if stats["acknowledged"] != 1 {
		t.Errorf("acknowledged = %d, want 1", stats["acknowledged"])
	}
	if stats["resolved"] != 1 {
		t.Errorf("resolved = %d, want 1", stats["resolved"])
	}
}

func TestHandleSignalStats_MissingProjectID(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/projects//signals/stats", nil)
	req.SetPathValue("id", "")
	w := httptest.NewRecorder()

	server.handleSignalStats(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

// --- handleSignalCount ---

func TestHandleSignalCount_Empty(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/projects/p1/signals/count", nil)
	req.SetPathValue("id", "p1")
	w := httptest.NewRecorder()

	server.handleSignalCount(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result map[string]int
	json.NewDecoder(resp.Body).Decode(&result)
	if result["count"] != 0 {
		t.Errorf("count = %d, want 0", result["count"])
	}
}

func TestHandleSignalCount_WithStatusFilter(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	createTestSignal(t, queries, p.ID, models.SignalStatusNew, "high")
	createTestSignal(t, queries, p.ID, models.SignalStatusAcknowledged, "low")

	req := httptest.NewRequest(http.MethodGet, "/projects/"+p.ID+"/signals/count?status=new", nil)
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleSignalCount(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result map[string]int
	json.NewDecoder(resp.Body).Decode(&result)
	if result["count"] != 1 {
		t.Errorf("count = %d, want 1", result["count"])
	}
}

func TestHandleSignalCount_MissingProjectID(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/projects//signals/count", nil)
	req.SetPathValue("id", "")
	w := httptest.NewRecorder()

	server.handleSignalCount(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}
