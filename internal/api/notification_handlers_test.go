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

func createTestNotificationChannel(t *testing.T, queries *db.Queries, projectID string) *models.NotificationChannel {
	t.Helper()
	now := time.Now().UTC()
	ch := &models.NotificationChannel{
		ID:          util.GenerateID(),
		ProjectID:   projectID,
		Name:        "test-channel",
		ChannelType: models.NotificationChannelTypeWebhook,
		URL:         "https://hooks.example.com/test",
		Enabled:     true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := queries.CreateNotificationChannel(ch); err != nil {
		t.Fatalf("create notification channel: %v", err)
	}
	return ch
}

// --- handleCreateNotificationChannel ---

func TestHandleCreateNotificationChannel_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)

	body, _ := json.Marshal(map[string]interface{}{
		"name":         "Slack Alert",
		"channel_type": "webhook",
		"url":          "https://hooks.slack.com/test",
		"enabled":      true,
	})

	req := httptest.NewRequest(http.MethodPost, "/projects/"+p.ID+"/notifications", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleCreateNotificationChannel(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	var ch models.NotificationChannel
	if err := json.NewDecoder(resp.Body).Decode(&ch); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if ch.Name != "Slack Alert" {
		t.Errorf("name = %q, want Slack Alert", ch.Name)
	}
	if ch.URL != "https://hooks.slack.com/test" {
		t.Errorf("url = %q", ch.URL)
	}
}

func TestHandleCreateNotificationChannel_Defaults(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)

	body, _ := json.Marshal(map[string]interface{}{
		"url": "https://hooks.example.com/test",
	})

	req := httptest.NewRequest(http.MethodPost, "/projects/"+p.ID+"/notifications", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleCreateNotificationChannel(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	var ch models.NotificationChannel
	if err := json.NewDecoder(resp.Body).Decode(&ch); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if ch.Name != "Webhook" {
		t.Errorf("name = %q, want Webhook", ch.Name)
	}
	if ch.ChannelType != models.NotificationChannelTypeWebhook {
		t.Errorf("channel_type = %q, want %q", ch.ChannelType, models.NotificationChannelTypeWebhook)
	}
	if !ch.Enabled {
		t.Error("enabled should default to true")
	}
}

func TestHandleCreateNotificationChannel_MissingURL(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	body, _ := json.Marshal(map[string]interface{}{
		"name": "test",
	})

	req := httptest.NewRequest(http.MethodPost, "/projects/p1/notifications", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "p1")
	w := httptest.NewRecorder()

	server.handleCreateNotificationChannel(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleCreateNotificationChannel_MissingProjectID(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	body, _ := json.Marshal(map[string]interface{}{"url": "https://example.com"})
	req := httptest.NewRequest(http.MethodPost, "/projects//notifications", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "")
	w := httptest.NewRecorder()

	server.handleCreateNotificationChannel(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleCreateNotificationChannel_InvalidJSON(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/projects/p1/notifications", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "p1")
	w := httptest.NewRecorder()

	server.handleCreateNotificationChannel(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleCreateNotificationChannel_Disabled(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)

	enabled := false
	body, _ := json.Marshal(map[string]interface{}{
		"url":     "https://hooks.example.com",
		"enabled": enabled,
	})

	req := httptest.NewRequest(http.MethodPost, "/projects/"+p.ID+"/notifications", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleCreateNotificationChannel(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	var ch models.NotificationChannel
	json.NewDecoder(resp.Body).Decode(&ch)
	if ch.Enabled {
		t.Error("enabled should be false")
	}
}

// --- handleListNotificationChannels ---

func TestHandleListNotificationChannels_Empty(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/projects/p1/notifications", nil)
	req.SetPathValue("id", "p1")
	w := httptest.NewRecorder()

	server.handleListNotificationChannels(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var channels []*models.NotificationChannel
	if err := json.NewDecoder(resp.Body).Decode(&channels); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(channels) != 0 {
		t.Errorf("len = %d, want 0", len(channels))
	}
}

func TestHandleListNotificationChannels_WithData(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	createTestNotificationChannel(t, queries, p.ID)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+p.ID+"/notifications", nil)
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleListNotificationChannels(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var channels []*models.NotificationChannel
	if err := json.NewDecoder(resp.Body).Decode(&channels); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(channels) != 1 {
		t.Errorf("len = %d, want 1", len(channels))
	}
}

// --- handleUpdateNotificationChannel ---

func TestHandleUpdateNotificationChannel_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	ch := createTestNotificationChannel(t, queries, p.ID)

	newName := "Updated Name"
	newURL := "https://updated.example.com"
	enabled := false
	body, _ := json.Marshal(map[string]interface{}{
		"name":    newName,
		"url":     newURL,
		"enabled": enabled,
	})

	req := httptest.NewRequest(http.MethodPatch, "/projects/"+p.ID+"/notifications/"+ch.ID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", p.ID)
	req.SetPathValue("channelId", ch.ID)
	w := httptest.NewRecorder()

	server.handleUpdateNotificationChannel(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var updated models.NotificationChannel
	if err := json.NewDecoder(resp.Body).Decode(&updated); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if updated.Name != newName {
		t.Errorf("name = %q, want %q", updated.Name, newName)
	}
	if updated.URL != newURL {
		t.Errorf("url = %q, want %q", updated.URL, newURL)
	}
	if updated.Enabled {
		t.Error("enabled should be false")
	}
}

func TestHandleUpdateNotificationChannel_NotFound(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	body, _ := json.Marshal(map[string]interface{}{"name": "x"})
	req := httptest.NewRequest(http.MethodPatch, "/projects/p1/notifications/nonexistent", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "p1")
	req.SetPathValue("channelId", "nonexistent")
	w := httptest.NewRecorder()

	server.handleUpdateNotificationChannel(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestHandleUpdateNotificationChannel_MissingIDs(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	body, _ := json.Marshal(map[string]interface{}{"name": "x"})
	req := httptest.NewRequest(http.MethodPatch, "/projects//notifications/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "")
	req.SetPathValue("channelId", "")
	w := httptest.NewRecorder()

	server.handleUpdateNotificationChannel(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleUpdateNotificationChannel_InvalidJSON(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	ch := createTestNotificationChannel(t, queries, p.ID)

	req := httptest.NewRequest(http.MethodPatch, "/projects/"+p.ID+"/notifications/"+ch.ID, bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", p.ID)
	req.SetPathValue("channelId", ch.ID)
	w := httptest.NewRecorder()

	server.handleUpdateNotificationChannel(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

// --- handleDeleteNotificationChannel ---

func TestHandleDeleteNotificationChannel_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	ch := createTestNotificationChannel(t, queries, p.ID)

	req := httptest.NewRequest(http.MethodDelete, "/projects/"+p.ID+"/notifications/"+ch.ID, nil)
	req.SetPathValue("id", p.ID)
	req.SetPathValue("channelId", ch.ID)
	w := httptest.NewRecorder()

	server.handleDeleteNotificationChannel(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNoContent)
	}
}

func TestHandleDeleteNotificationChannel_MissingIDs(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodDelete, "/projects//notifications/", nil)
	req.SetPathValue("id", "")
	req.SetPathValue("channelId", "")
	w := httptest.NewRecorder()

	server.handleDeleteNotificationChannel(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}
