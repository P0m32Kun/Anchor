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

// --- handleParseScopeValue ---

func TestHandleParseScopeValue_Domain(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	body, _ := json.Marshal(map[string]interface{}{
		"value": "example.com",
	})

	req := httptest.NewRequest(http.MethodPost, "/scope-rules/parse", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleParseScopeValue(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result ParseScopeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result.Rules) == 0 {
		t.Error("expected at least one parsed rule")
	}
}

func TestHandleParseScopeValue_InvalidJSON(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/scope-rules/parse", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleParseScopeValue(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleParseScopeValue_CIDR(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	body, _ := json.Marshal(map[string]interface{}{
		"value": "192.168.1.0/24",
	})

	req := httptest.NewRequest(http.MethodPost, "/scope-rules/parse", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleParseScopeValue(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

// --- handleDeleteScopeRule ---

func TestHandleDeleteScopeRule_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	now := time.Now().UTC()
	rule := &models.ScopeRule{
		ID:        util.GenerateID(),
		ProjectID: p.ID,
		Action:    models.ScopeActionInclude,
		Type:      models.TargetTypeDomain,
		Value:     "example.com",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := queries.CreateScopeRule(rule); err != nil {
		t.Fatalf("create scope rule: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/scope-rules/"+rule.ID, nil)
	req.SetPathValue("id", rule.ID)
	w := httptest.NewRecorder()

	server.handleDeleteScopeRule(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

// --- handleBatchCreateScopeRules ---

func TestHandleBatchCreateScopeRules_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)

	body, _ := json.Marshal(map[string]interface{}{
		"rules": []map[string]interface{}{
			{"action": "include", "type": "domain", "value": "example.com"},
			{"action": "exclude", "type": "domain", "value": "internal.example.com"},
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/projects/"+p.ID+"/scope-rules/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleBatchCreateScopeRules(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["created"].(float64) != 2 {
		t.Errorf("created = %v, want 2", result["created"])
	}
}

func TestHandleBatchCreateScopeRules_DefaultAction(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)

	body, _ := json.Marshal(map[string]interface{}{
		"rules": []map[string]interface{}{
			{"type": "domain", "value": "example.com"},
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/projects/"+p.ID+"/scope-rules/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleBatchCreateScopeRules(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}
}

func TestHandleBatchCreateScopeRules_InvalidJSON(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/projects/proj1/scope-rules/batch", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "proj1")
	w := httptest.NewRecorder()

	server.handleBatchCreateScopeRules(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}
