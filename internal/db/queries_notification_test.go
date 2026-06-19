package db

import (
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
)

// --- Notification Channel tests ---

func setupNotificationTest(t *testing.T) (*Queries, string) {
	t.Helper()
	q := New(openTestDB(t))
	now := time.Now().UTC()
	if err := q.CreateProject(&models.Project{
		ID: "proj-notif", Name: "notif-test", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create project: %v", err)
	}
	return q, "proj-notif"
}

func TestNotificationChannel_CRUD(t *testing.T) {
	q, projID := setupNotificationTest(t)
	now := time.Now().UTC()

	ch := &models.NotificationChannel{
		ID: "ch-1", ProjectID: projID, Name: "slack-alerts",
		ChannelType: "slack", URL: "https://hooks.slack.com/test",
		Enabled: true, CreatedAt: now, UpdatedAt: now,
	}
	if err := q.CreateNotificationChannel(ch); err != nil {
		t.Fatalf("CreateNotificationChannel: %v", err)
	}

	// Get by ID
	got, err := q.GetNotificationChannelByID("ch-1", projID)
	if err != nil {
		t.Fatalf("GetNotificationChannelByID: %v", err)
	}
	if got == nil {
		t.Fatal("expected channel, got nil")
	}
	if got.Name != "slack-alerts" {
		t.Errorf("name = %q, want slack-alerts", got.Name)
	}
	if got.ChannelType != "slack" {
		t.Errorf("channel_type = %q, want slack", got.ChannelType)
	}
	if !got.Enabled {
		t.Error("expected enabled=true")
	}

	// List
	list, err := q.ListNotificationChannelsByProject(projID)
	if err != nil {
		t.Fatalf("ListNotificationChannelsByProject: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1, got %d", len(list))
	}

	// Update
	ch.Name = "slack-updated"
	ch.URL = "https://hooks.slack.com/new"
	ch.Enabled = false
	if err := q.UpdateNotificationChannel(ch); err != nil {
		t.Fatalf("UpdateNotificationChannel: %v", err)
	}
	got2, _ := q.GetNotificationChannelByID("ch-1", projID)
	if got2.Name != "slack-updated" {
		t.Errorf("name after update = %q, want slack-updated", got2.Name)
	}
	if got2.Enabled {
		t.Error("expected enabled=false after update")
	}

	// Delete
	if err := q.DeleteNotificationChannel("ch-1", projID); err != nil {
		t.Fatalf("DeleteNotificationChannel: %v", err)
	}
	got3, _ := q.GetNotificationChannelByID("ch-1", projID)
	if got3 != nil {
		t.Error("expected nil after delete")
	}
}

func TestNotificationChannel_AutoID(t *testing.T) {
	q, projID := setupNotificationTest(t)
	now := time.Now().UTC()

	ch := &models.NotificationChannel{
		ProjectID: projID, Name: "auto-id",
		ChannelType: "webhook", URL: "https://example.com/hook",
		Enabled: true, CreatedAt: now, UpdatedAt: now,
	}
	if err := q.CreateNotificationChannel(ch); err != nil {
		t.Fatalf("CreateNotificationChannel: %v", err)
	}
	if ch.ID == "" {
		t.Error("expected auto-generated ID")
	}
}

func TestNotificationChannel_ListEnabled(t *testing.T) {
	q, projID := setupNotificationTest(t)
	now := time.Now().UTC()

	ch1 := &models.NotificationChannel{
		ID: "ch-enabled", ProjectID: projID, Name: "enabled",
		ChannelType: "slack", URL: "https://hooks.slack.com/1",
		Enabled: true, CreatedAt: now, UpdatedAt: now,
	}
	ch2 := &models.NotificationChannel{
		ID: "ch-disabled", ProjectID: projID, Name: "disabled",
		ChannelType: "slack", URL: "https://hooks.slack.com/2",
		Enabled: false, CreatedAt: now, UpdatedAt: now,
	}
	q.CreateNotificationChannel(ch1)
	q.CreateNotificationChannel(ch2)

	enabled, err := q.ListEnabledNotificationChannelsByProject(projID)
	if err != nil {
		t.Fatalf("ListEnabledNotificationChannelsByProject: %v", err)
	}
	if len(enabled) != 1 {
		t.Fatalf("expected 1 enabled, got %d", len(enabled))
	}
	if enabled[0].ID != "ch-enabled" {
		t.Errorf("enabled channel ID = %q, want ch-enabled", enabled[0].ID)
	}
}

func TestNotificationChannel_NotFound(t *testing.T) {
	q, projID := setupNotificationTest(t)

	got, err := q.GetNotificationChannelByID("nonexistent", projID)
	if err == nil {
		t.Fatal("expected error for nonexistent channel")
	}
	if got != nil {
		t.Error("expected nil for nonexistent channel")
	}
}
