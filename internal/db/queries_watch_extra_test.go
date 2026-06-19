package db

import (
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
)

func createWatchTestProject(t *testing.T, q *Queries, id, name string) {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Second)
	p := &models.Project{
		ID:             id,
		Name:           name,
		Organization:   "test-org",
		Purpose:        "testing",
		RateLimit:      100,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := q.CreateProject(p); err != nil {
		t.Fatalf("create project %s: %v", id, err)
	}
}

func TestGetWatchProject_RoundTrip(t *testing.T) {
	q := New(openTestDB(t))
	createWatchTestProject(t, q, "proj-1", "alpha")

	got, err := q.GetWatchProject("proj-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got == nil {
		t.Fatal("get returned nil for existing project")
	}
	if got.ID != "proj-1" {
		t.Errorf("id: want proj-1, got %q", got.ID)
	}
	if got.Name != "alpha" {
		t.Errorf("name: want alpha, got %q", got.Name)
	}
	if got.WatchEnabled {
		t.Error("watch_enabled: want false by default")
	}
	if got.WatchIntervalHours != 24 {
		t.Errorf("watch_interval_hours: want 24 (default), got %d", got.WatchIntervalHours)
	}
	if !got.WatchPassiveOnly {
		t.Error("watch_passive_only: want true by default")
	}
	if got.WatchLastTickAt != nil {
		t.Errorf("watch_last_tick_at: want nil, got %v", got.WatchLastTickAt)
	}
}

func TestGetWatchProject_NotFoundReturnsNil(t *testing.T) {
	q := New(openTestDB(t))

	got, err := q.GetWatchProject("nonexistent")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got != nil {
		t.Errorf("want nil, got %+v", got)
	}
}

func TestUpdateProjectWatchConfig(t *testing.T) {
	q := New(openTestDB(t))
	createWatchTestProject(t, q, "proj-cfg", "beta")

	if err := q.UpdateProjectWatchConfig("proj-cfg", true, 6, true); err != nil {
		t.Fatalf("update config: %v", err)
	}

	got, err := q.GetWatchProject("proj-cfg")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got == nil {
		t.Fatal("nil")
	}
	if !got.WatchEnabled {
		t.Error("watch_enabled: want true")
	}
	if got.WatchIntervalHours != 6 {
		t.Errorf("watch_interval_hours: want 6, got %d", got.WatchIntervalHours)
	}
	if !got.WatchPassiveOnly {
		t.Error("watch_passive_only: want true")
	}

	// Update again to disable
	if err := q.UpdateProjectWatchConfig("proj-cfg", false, 12, false); err != nil {
		t.Fatalf("update config 2: %v", err)
	}
	got2, err := q.GetWatchProject("proj-cfg")
	if err != nil {
		t.Fatalf("get 2: %v", err)
	}
	if got2.WatchEnabled {
		t.Error("watch_enabled: want false after second update")
	}
	if got2.WatchIntervalHours != 12 {
		t.Errorf("watch_interval_hours: want 12, got %d", got2.WatchIntervalHours)
	}
	if got2.WatchPassiveOnly {
		t.Error("watch_passive_only: want false after second update")
	}
}

func TestUpdateProjectWatchTick(t *testing.T) {
	q := New(openTestDB(t))
	createWatchTestProject(t, q, "proj-tick", "gamma")

	tickAt := time.Now().UTC().Truncate(time.Second)
	if err := q.UpdateProjectWatchTick("proj-tick", tickAt); err != nil {
		t.Fatalf("update tick: %v", err)
	}

	got, err := q.GetWatchProject("proj-tick")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got == nil {
		t.Fatal("nil")
	}
	if got.WatchLastTickAt == nil || !got.WatchLastTickAt.Equal(tickAt) {
		t.Errorf("watch_last_tick_at: want %v, got %v", tickAt, got.WatchLastTickAt)
	}
}

func TestListWatchEnabledProjects(t *testing.T) {
	q := New(openTestDB(t))

	createWatchTestProject(t, q, "proj-a", "alpha")
	createWatchTestProject(t, q, "proj-b", "beta")
	createWatchTestProject(t, q, "proj-c", "gamma")

	// Enable watch on proj-a and proj-c
	if err := q.UpdateProjectWatchConfig("proj-a", true, 4, false); err != nil {
		t.Fatalf("enable proj-a: %v", err)
	}
	if err := q.UpdateProjectWatchConfig("proj-c", true, 8, true); err != nil {
		t.Fatalf("enable proj-c: %v", err)
	}

	list, err := q.ListWatchEnabledProjects()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("len: want 2, got %d", len(list))
	}
	// Ordered by id
	if list[0].ID != "proj-a" {
		t.Errorf("position 0: want proj-a, got %q", list[0].ID)
	}
	if list[1].ID != "proj-c" {
		t.Errorf("position 1: want proj-c, got %q", list[1].ID)
	}
	if !list[0].WatchEnabled {
		t.Error("proj-a watch_enabled: want true")
	}
	if list[0].WatchIntervalHours != 4 {
		t.Errorf("proj-a watch_interval_hours: want 4, got %d", list[0].WatchIntervalHours)
	}
	if !list[1].WatchPassiveOnly {
		t.Error("proj-c watch_passive_only: want true")
	}
}

func TestListWatchEnabledProjects_Empty(t *testing.T) {
	q := New(openTestDB(t))

	createWatchTestProject(t, q, "proj-x", "no-watch")

	list, err := q.ListWatchEnabledProjects()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("want 0, got %d", len(list))
	}
}
