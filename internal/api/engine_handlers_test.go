package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// --- maskKey ---

func TestMaskKey(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want string
	}{
		{"long key", "abcdefghij", "abcd****"},
		{"short key", "abc", "abc"},
		{"exact 4", "abcd", "abcd"},
		{"5 chars", "abcde", "abcd****"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := maskKey(tt.key)
			if got != tt.want {
				t.Errorf("maskKey(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

// --- handleListEngineCredentials ---

func TestHandleListEngineCredentials_Empty(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/engines/credentials", nil)
	w := httptest.NewRecorder()

	server.handleListEngineCredentials(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var creds []interface{}
	if err := json.NewDecoder(resp.Body).Decode(&creds); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(creds) != 0 {
		t.Errorf("len = %d, want 0", len(creds))
	}
}

func TestHandleListEngineCredentials_WithMaskedKeys(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	cred := util.GenerateID()
	_ = queries.SaveEngineCredential(&models.EngineCredential{
		ID:     cred,
		Engine: "fofa",
		APIKey: "supersecretkey123",
	})

	req := httptest.NewRequest(http.MethodGet, "/engines/credentials", nil)
	w := httptest.NewRecorder()

	server.handleListEngineCredentials(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var creds []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&creds); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(creds) != 1 {
		t.Fatalf("len = %d, want 1", len(creds))
	}
	// API key should be masked
	if creds[0]["api_key"] == "supersecretkey123" {
		t.Error("api_key should be masked")
	}
}

// --- handleSaveEngineCredential ---

func TestHandleSaveEngineCredential_Success(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	body, _ := json.Marshal(map[string]interface{}{
		"engine":  "fofa",
		"api_key": "testkey12345",
	})

	req := httptest.NewRequest(http.MethodPost, "/engines/credentials", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleSaveEngineCredential(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var cred map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&cred); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if cred["engine"] != "fofa" {
		t.Errorf("engine = %v", cred["engine"])
	}
	// API key should be masked in response
	if cred["api_key"] == "testkey12345" {
		t.Error("api_key should be masked in response")
	}
}

func TestHandleSaveEngineCredential_MissingFields(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	body, _ := json.Marshal(map[string]interface{}{
		"engine": "",
		"api_key": "",
	})

	req := httptest.NewRequest(http.MethodPost, "/engines/credentials", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleSaveEngineCredential(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleSaveEngineCredential_InvalidJSON(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/engines/credentials", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleSaveEngineCredential(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

// --- handleDeleteEngineCredential ---

func TestHandleDeleteEngineCredential_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	_ = queries.SaveEngineCredential(&models.EngineCredential{
		ID:     util.GenerateID(),
		Engine: "fofa",
		APIKey: "testkey",
	})

	req := httptest.NewRequest(http.MethodDelete, "/engines/credentials/fofa", nil)
	req.SetPathValue("engine", "fofa")
	w := httptest.NewRecorder()

	server.handleDeleteEngineCredential(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNoContent)
	}
}

func TestHandleDeleteEngineCredential_MissingEngine(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodDelete, "/engines/credentials/", nil)
	req.SetPathValue("engine", "")
	w := httptest.NewRecorder()

	server.handleDeleteEngineCredential(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

// --- handleSearchEngine ---

func TestHandleSearchEngine_MissingParams(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/engines/search", nil)
	w := httptest.NewRecorder()

	server.handleSearchEngine(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleSearchEngine_UnsupportedEngine(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/engines/search?engine=unknown&query=test", nil)
	w := httptest.NewRecorder()

	server.handleSearchEngine(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleSearchEngine_NoCredential(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/engines/search?engine=fofa&query=test", nil)
	w := httptest.NewRecorder()

	server.handleSearchEngine(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	// No credential configured → should return error
	if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want 400 or 500", resp.StatusCode)
	}
}

// --- handleGetEngineQuota ---

func TestHandleGetEngineQuota_MissingEngine(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/engines/quota", nil)
	w := httptest.NewRecorder()

	server.handleGetEngineQuota(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleGetEngineQuota_UnsupportedEngine(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/engines/quota?engine=unknown", nil)
	w := httptest.NewRecorder()

	server.handleGetEngineQuota(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleGetEngineQuota_NoCredential(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/engines/quota?engine=fofa", nil)
	w := httptest.NewRecorder()

	server.handleGetEngineQuota(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want 400 or 500", resp.StatusCode)
	}
}

// --- parseIntQuery ---

func TestParseIntQuery(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		key        string
		defaultVal int
		want       int
	}{
		{"empty", "", "page", 1, 1},
		{"valid", "page=5", "page", 1, 5},
		{"invalid", "page=abc", "page", 1, 1},
		{"zero", "page=0", "page", 1, 1},
		{"negative", "page=-1", "page", 1, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/test"
			if tt.query != "" {
				url = "/test?" + tt.query
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			got := parseIntQuery(req, tt.key, tt.defaultVal)
			if got != tt.want {
				t.Errorf("parseIntQuery() = %d, want %d", got, tt.want)
			}
		})
	}
}
