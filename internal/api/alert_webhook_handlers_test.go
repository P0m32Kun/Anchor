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

// --- handleGetAlertWebhook ---

func TestHandleGetAlertWebhook_NotFound(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/projects/p1/alert-webhook", nil)
	req.SetPathValue("id", "p1")
	w := httptest.NewRecorder()

	server.handleGetAlertWebhook(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["enabled"] != false {
		t.Errorf("enabled = %v, want false", body["enabled"])
	}
}

func TestHandleGetAlertWebhook_MissingID(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/projects//alert-webhook", nil)
	req.SetPathValue("id", "")
	w := httptest.NewRecorder()

	server.handleGetAlertWebhook(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleGetAlertWebhook_Found(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	now := time.Now().UTC()
	webhook := &models.AlertWebhook{
		ID:          util.GenerateID(),
		ProjectID:   p.ID,
		Enabled:     true,
		URL:         "https://hooks.example.com/test",
		Secret:      "mysecret",
		MinSeverity: "high",
		OnNewAsset:  true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := queries.UpsertAlertWebhook(webhook); err != nil {
		t.Fatalf("upsert webhook: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/projects/"+p.ID+"/alert-webhook", nil)
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleGetAlertWebhook(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["enabled"] != true {
		t.Errorf("enabled = %v, want true", body["enabled"])
	}
	if body["url"] != "https://hooks.example.com/test" {
		t.Errorf("url = %v", body["url"])
	}
	if body["has_secret"] != true {
		t.Errorf("has_secret = %v, want true", body["has_secret"])
	}
}

// --- handleUpsertAlertWebhook ---

func TestHandleUpsertAlertWebhook_Create(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)

	enabled := true
	body, _ := json.Marshal(map[string]interface{}{
		"enabled":     enabled,
		"url":         "https://hooks.example.com/test",
		"secret":      "s3cret",
		"min_severity": "high",
	})

	req := httptest.NewRequest(http.MethodPut, "/projects/"+p.ID+"/alert-webhook", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleUpsertAlertWebhook(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result["enabled"] != true {
		t.Errorf("enabled = %v, want true", result["enabled"])
	}
	if result["url"] != "https://hooks.example.com/test" {
		t.Errorf("url = %v", result["url"])
	}
}

func TestHandleUpsertAlertWebhook_MissingID(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	body, _ := json.Marshal(map[string]interface{}{"url": "https://example.com"})
	req := httptest.NewRequest(http.MethodPut, "/projects//alert-webhook", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "")
	w := httptest.NewRecorder()

	server.handleUpsertAlertWebhook(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleUpsertAlertWebhook_InvalidJSON(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPut, "/projects/p1/alert-webhook", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "p1")
	w := httptest.NewRecorder()

	server.handleUpsertAlertWebhook(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleUpsertAlertWebhook_Update(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	now := time.Now().UTC()
	webhook := &models.AlertWebhook{
		ID:          util.GenerateID(),
		ProjectID:   p.ID,
		Enabled:     false,
		URL:         "https://old.example.com",
		MinSeverity: "low",
		OnNewAsset:  true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := queries.UpsertAlertWebhook(webhook); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	onAssetGone := false
	body, _ := json.Marshal(map[string]interface{}{
		"enabled":        true,
		"url":            "https://new.example.com",
		"on_asset_gone":  onAssetGone,
	})

	req := httptest.NewRequest(http.MethodPut, "/projects/"+p.ID+"/alert-webhook", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleUpsertAlertWebhook(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result["enabled"] != true {
		t.Errorf("enabled = %v, want true", result["enabled"])
	}
	if result["url"] != "https://new.example.com" {
		t.Errorf("url = %v", result["url"])
	}
}

// --- handleDeleteAlertWebhook ---

func TestHandleDeleteAlertWebhook_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	now := time.Now().UTC()
	webhook := &models.AlertWebhook{
		ID:        util.GenerateID(),
		ProjectID: p.ID,
		Enabled:   true,
		URL:       "https://hooks.example.com/test",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := queries.UpsertAlertWebhook(webhook); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/projects/"+p.ID+"/alert-webhook", nil)
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleDeleteAlertWebhook(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNoContent)
	}
}

func TestHandleDeleteAlertWebhook_MissingID(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodDelete, "/projects//alert-webhook", nil)
	req.SetPathValue("id", "")
	w := httptest.NewRecorder()

	server.handleDeleteAlertWebhook(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}
