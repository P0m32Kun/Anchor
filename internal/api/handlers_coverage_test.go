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

// --- handleHealth ---

func TestHandleHealth(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	server.handleHealth(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("status = %q, want ok", body["status"])
	}
}

// --- handleToolHealth ---

func TestHandleToolHealth_Empty(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/health/tools", nil)
	w := httptest.NewRecorder()

	server.handleToolHealth(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestHandleToolHealth_WithData(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	now := time.Now().UTC()
	err := queries.UpsertToolHealth(&models.ToolHealth{
		ID:               util.GenerateID(),
		Tool:             "nuclei",
		BinaryPath:       "/usr/bin/nuclei",
		Version:          "3.0.0",
		WorkdirWritable:  true,
		NetworkAvailable: true,
		DNSAvailable:     true,
		LastCheckAt:      now,
	})
	if err != nil {
		t.Fatalf("upsert tool health: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/health/tools", nil)
	w := httptest.NewRecorder()

	server.handleToolHealth(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var tools []models.ToolHealth
	if err := json.NewDecoder(resp.Body).Decode(&tools); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(tools) != 1 {
		t.Fatalf("len(tools) = %d, want 1", len(tools))
	}
	if tools[0].Tool != "nuclei" {
		t.Errorf("tool = %q, want nuclei", tools[0].Tool)
	}
}

// --- handleHealthCheck ---

func TestHandleHealthCheck(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/health/check", nil)
	w := httptest.NewRecorder()

	server.handleHealthCheck(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

// --- handleListWorkers ---

func TestHandleListWorkers_Empty(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/workers", nil)
	w := httptest.NewRecorder()

	server.handleListWorkers(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var workers []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&workers); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(workers) != 0 {
		t.Errorf("len(workers) = %d, want 0", len(workers))
	}
}

func TestHandleListWorkers_WithData(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	createTestWorker(t, queries, models.WorkerStatusOnline)

	req := httptest.NewRequest(http.MethodGet, "/workers", nil)
	w := httptest.NewRecorder()

	server.handleListWorkers(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var workers []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&workers); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(workers) != 1 {
		t.Errorf("len(workers) = %d, want 1", len(workers))
	}
}

// --- handleListToolTemplates ---

func TestHandleListToolTemplates(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/tool-templates", nil)
	w := httptest.NewRecorder()

	server.handleListToolTemplates(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

// --- handleGetToolTemplate ---

func TestHandleGetToolTemplate_NotFound(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/tool-templates/nonexistent", nil)
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()

	server.handleGetToolTemplate(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

// --- defaultToolTimeout ---

func TestDefaultToolTimeout(t *testing.T) {
	tests := []struct {
		tool string
		want time.Duration
	}{
		{"subfinder", 300 * time.Second},
		{"httpx", 300 * time.Second},
		{"naabu", 600 * time.Second},
		{"nuclei", 1800 * time.Second},
		{"nmap", 600 * time.Second},
		{"unknown", 300 * time.Second},
		{"", 300 * time.Second},
	}
	for _, tt := range tests {
		t.Run(tt.tool, func(t *testing.T) {
			got := defaultToolTimeout(tt.tool)
			if got != tt.want {
				t.Errorf("defaultToolTimeout(%q) = %v, want %v", tt.tool, got, tt.want)
			}
		})
	}
}

// --- redactArgs ---

func TestRedactArgs(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
		want string
	}{
		{"no secrets", "nuclei -t template.yaml", "nuclei -t template.yaml"},
		{"with api_key", "tool --api_key=secret123", "tool [REDACTED]"},
		{"with token", "tool --token=abc123", "tool [REDACTED]"},
		{"with secret", "tool --secret=mysecret", "tool [REDACTED]"},
		{"with password", "tool --password=pass123", "tool [REDACTED]"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := redactArgs(tt.cmd)
			if got != tt.want {
				t.Errorf("redactArgs(%q) = %q, want %q", tt.cmd, got, tt.want)
			}
		})
	}
}

// --- handleToolHealth error path ---

func TestHandleToolHealth_Error(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	rawDB.Close()

	req := httptest.NewRequest(http.MethodGet, "/health/tools", nil)
	w := httptest.NewRecorder()

	server.handleToolHealth(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusInternalServerError)
	}
}

// --- handleListToolTemplates error path ---

func TestHandleListToolTemplates_Error(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	rawDB.Close()

	req := httptest.NewRequest(http.MethodGet, "/tool-templates", nil)
	w := httptest.NewRecorder()

	server.handleListToolTemplates(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusInternalServerError)
	}
}

// --- handleGetToolTemplate error path ---

func TestHandleGetToolTemplate_DBError(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	rawDB.Close()

	req := httptest.NewRequest(http.MethodGet, "/tool-templates/some-id", nil)
	req.SetPathValue("id", "some-id")
	w := httptest.NewRecorder()

	server.handleGetToolTemplate(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusInternalServerError)
	}
}

