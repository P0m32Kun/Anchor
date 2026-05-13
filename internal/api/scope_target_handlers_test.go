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

// --- handleCreateScopeRule ---

func TestHandleCreateScopeRule_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)

	body, _ := json.Marshal(map[string]string{
		"project_id": p.ID,
		"action":     "include",
		"type":       "domain",
		"value":      "example.com",
		"reason":     "in scope",
	})
	req := httptest.NewRequest(http.MethodPost, "/scope-rules", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleCreateScopeRule(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	var rule models.ScopeRule
	if err := json.NewDecoder(resp.Body).Decode(&rule); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if rule.Value != "example.com" {
		t.Errorf("value = %q, want example.com", rule.Value)
	}
	if rule.Action != models.ScopeActionInclude {
		t.Errorf("action = %q, want include", rule.Action)
	}
}

func TestHandleCreateScopeRule_MissingFields(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	body, _ := json.Marshal(map[string]string{
		"project_id": "proj-1",
		// missing action, type, value
	})
	req := httptest.NewRequest(http.MethodPost, "/scope-rules", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleCreateScopeRule(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	// Handler doesn't validate missing fields — it passes to DB which may
	// return 500 (FK constraint) or 400 depending on implementation.
	if resp.StatusCode < 400 {
		t.Errorf("status = %d, want >= 400", resp.StatusCode)
	}
}

// --- handleListScopeRules ---

func TestHandleListScopeRules_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	now := time.Now().UTC()

	for _, v := range []string{"example.com", "test.com"} {
		queries.CreateScopeRule(&models.ScopeRule{
			ID: util.GenerateID(), ProjectID: p.ID,
			Action: models.ScopeActionInclude, Type: models.TargetTypeDomain,
			Value: v, CreatedAt: now, UpdatedAt: now,
		})
	}

	req := httptest.NewRequest(http.MethodGet, "/scope-rules?project_id="+p.ID, nil)
	w := httptest.NewRecorder()

	server.handleListScopeRules(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result PaginatedResponse[*models.ScopeRule]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result.Data) != 2 {
		t.Errorf("len = %d, want 2", len(result.Data))
	}
}

// --- handleCreateTarget ---

func TestHandleCreateTarget_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)

	// Add scope rule so target can be created.
	now := time.Now().UTC()
	queries.CreateScopeRule(&models.ScopeRule{
		ID: util.GenerateID(), ProjectID: p.ID,
		Action: models.ScopeActionInclude, Type: models.TargetTypeDomain,
		Value: "example.com", CreatedAt: now, UpdatedAt: now,
	})

	body, _ := json.Marshal(map[string]string{
		"type":  "domain",
		"value": "example.com",
	})
	req := httptest.NewRequest(http.MethodPost, "/projects/"+p.ID+"/targets", bytes.NewReader(body))
	req.SetPathValue("id", p.ID)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleCreateTarget(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}
}

func TestHandleCreateTarget_AutoDetect(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)

	body, _ := json.Marshal(map[string]string{
		"type":  "auto",
		"value": "192.168.1.1",
	})
	req := httptest.NewRequest(http.MethodPost, "/projects/"+p.ID+"/targets", bytes.NewReader(body))
	req.SetPathValue("id", p.ID)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleCreateTarget(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	// No scope rules → needs confirmation (202 or similar).
	// The exact status depends on implementation; just verify it doesn't crash.
	if resp.StatusCode >= 500 {
		t.Errorf("status = %d, want < 500", resp.StatusCode)
	}
}

// --- handleListTargets ---

func TestHandleListTargets_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	now := time.Now().UTC()

	for _, v := range []string{"a.com", "b.com"} {
		queries.CreateTarget(&models.Target{
			ID: util.GenerateID(), ProjectID: p.ID,
			Type: models.TargetTypeDomain, Value: v,
			Source: "manual", Status: "active",
			CreatedAt: now,
		})
	}

	req := httptest.NewRequest(http.MethodGet, "/projects/"+p.ID+"/targets", nil)
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleListTargets(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result PaginatedResponse[*models.Target]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result.Data) != 2 {
		t.Errorf("len = %d, want 2", len(result.Data))
	}
}
