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

// --- handleRetestFinding ---

func TestHandleRetestFinding_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	finding := createTestFinding(t, queries, p.ID, models.FindingPendingReview)

	req := httptest.NewRequest(http.MethodPost, "/findings/"+finding.ID+"/retest", nil)
	req.SetPathValue("id", finding.ID)
	w := httptest.NewRecorder()

	server.handleRetestFinding(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	var retest models.RetestRun
	if err := json.NewDecoder(resp.Body).Decode(&retest); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if retest.FindingID != finding.ID {
		t.Errorf("finding_id = %q, want %q", retest.FindingID, finding.ID)
	}
	if retest.Result != models.RetestInconclusive {
		t.Errorf("result = %q, want %q", retest.Result, models.RetestInconclusive)
	}
}

func TestHandleRetestFinding_NotFound(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/findings/nonexistent/retest", nil)
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()

	server.handleRetestFinding(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

// --- handleListRetests ---

func TestHandleListRetests_WithData(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	finding := createTestFinding(t, queries, p.ID, models.FindingPendingReview)

	// Create a retest run
	now := time.Now().UTC()
	retest := &models.RetestRun{
		ID:        util.GenerateID(),
		FindingID: finding.ID,
		Result:    models.RetestInconclusive,
		CreatedAt: now,
	}
	if err := queries.CreateRetestRun(retest); err != nil {
		t.Fatalf("create retest: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/findings/"+finding.ID+"/retests", nil)
	req.SetPathValue("id", finding.ID)
	w := httptest.NewRecorder()

	server.handleListRetests(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var retests []*models.RetestRun
	json.NewDecoder(resp.Body).Decode(&retests)
	if len(retests) != 1 {
		t.Errorf("len = %d, want 1", len(retests))
	}
}

// --- handleBatchUpdateFindingStatus ---

func TestHandleBatchUpdateFindingStatus_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	f1 := createTestFinding(t, queries, p.ID, models.FindingPendingReview)
	f2 := createTestFinding(t, queries, p.ID, models.FindingPendingReview)

	body, _ := json.Marshal(map[string]interface{}{
		"ids":    []string{f1.ID, f2.ID},
		"status": "confirmed",
	})

	req := httptest.NewRequest(http.MethodPatch, "/findings/batch-status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleBatchUpdateFindingStatus(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["updated"].(float64) != 2 {
		t.Errorf("updated = %v, want 2", result["updated"])
	}
}

func TestHandleBatchUpdateFindingStatus_EmptyIDs(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	body, _ := json.Marshal(map[string]interface{}{
		"ids":    []string{},
		"status": "confirmed",
	})

	req := httptest.NewRequest(http.MethodPatch, "/findings/batch-status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleBatchUpdateFindingStatus(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleBatchUpdateFindingStatus_InvalidStatus(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	body, _ := json.Marshal(map[string]interface{}{
		"ids":    []string{"f1"},
		"status": "invalid",
	})

	req := httptest.NewRequest(http.MethodPatch, "/findings/batch-status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleBatchUpdateFindingStatus(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleBatchUpdateFindingStatus_InvalidJSON(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPatch, "/findings/batch-status", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleBatchUpdateFindingStatus(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleBatchUpdateFindingStatus_TooManyIDs(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	ids := make([]string, 1001)
	for i := range ids {
		ids[i] = "id"
	}
	body, _ := json.Marshal(map[string]interface{}{
		"ids":    ids,
		"status": "confirmed",
	})

	req := httptest.NewRequest(http.MethodPatch, "/findings/batch-status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleBatchUpdateFindingStatus(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

// --- handleGetFindingCurl ---

func TestHandleGetFindingCurl_NotFound(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/findings/nonexistent/curl", nil)
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()

	server.handleGetFindingCurl(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestHandleGetFindingCurl_WithRawRequest(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	finding := createTestFinding(t, queries, p.ID, models.FindingPendingReview)

	// Update finding with raw request
	now := time.Now().UTC()
	queries.UpdateFindingStatus(finding.ID, models.FindingPendingReview, now)

	req := httptest.NewRequest(http.MethodGet, "/findings/"+finding.ID+"/curl", nil)
	req.SetPathValue("id", finding.ID)
	w := httptest.NewRecorder()

	server.handleGetFindingCurl(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}
