package db

import (
	"database/sql"
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
)

// --- helpers ---

func setupAssetChangeTest(t *testing.T) (*Queries, string) {
	t.Helper()
	q := New(openTestDB(t))
	now := time.Now().UTC().Truncate(time.Second)
	if err := q.CreateProject(&models.Project{
		ID: "proj-ac-1", Name: "test-ac", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create project: %v", err)
	}
	return q, "proj-ac-1"
}

func seedAssetChange(t *testing.T, q *Queries, id, projectID, changeType, severity string) {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Second)
	if err := q.CreateAssetChange(&models.AssetChange{
		ID: id, ProjectID: projectID, RunID: "run-1",
		AssetID: "asset-1", AssetValue: "192.168.1.1", AssetType: "ip",
		ChangeType: changeType, ChangeSummary: "change detected",
		DetailJSON: "{}", Severity: severity, CreatedAt: now,
	}); err != nil {
		t.Fatalf("seed asset change %s: %v", id, err)
	}
}

// --- tests ---

func TestCreateAssetChange(t *testing.T) {
	q, projID := setupAssetChangeTest(t)

	now := time.Now().UTC().Truncate(time.Second)
	c := &models.AssetChange{
		ID: "ac-1", ProjectID: projID, RunID: "run-1",
		AssetID: "asset-1", AssetValue: "10.0.0.1", AssetType: "ip",
		ChangeType: models.ChangeTypePortNew, ChangeSummary: "new port 443",
		DetailJSON: `{"port":443}`, Severity: "info", CreatedAt: now,
	}
	if err := q.CreateAssetChange(c); err != nil {
		t.Fatalf("CreateAssetChange: %v", err)
	}

	// Verify via listing
	list, err := q.ListAssetChangesByProject(projID, 10, 0)
	if err != nil {
		t.Fatalf("ListAssetChangesByProject: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 change, got %d", len(list))
	}
	got := list[0]
	if got.ID != "ac-1" {
		t.Errorf("ID = %q, want ac-1", got.ID)
	}
	if got.ChangeType != models.ChangeTypePortNew {
		t.Errorf("ChangeType = %q, want %q", got.ChangeType, models.ChangeTypePortNew)
	}
	if got.AssetValue != "10.0.0.1" {
		t.Errorf("AssetValue = %q, want 10.0.0.1", got.AssetValue)
	}
	if got.DetailJSON != `{"port":443}` {
		t.Errorf("DetailJSON = %q, want %q", got.DetailJSON, `{"port":443}`)
	}
}

func TestCreateAssetChange_AutoID(t *testing.T) {
	q, projID := setupAssetChangeTest(t)

	c := &models.AssetChange{
		ProjectID: projID, RunID: "run-1",
		AssetID: "asset-auto", AssetValue: "example.com", AssetType: "domain",
		ChangeType: models.ChangeTypeAssetNew, ChangeSummary: "new asset",
		DetailJSON: "{}", Severity: "info",
	}
	if err := q.CreateAssetChange(c); err != nil {
		t.Fatalf("CreateAssetChange: %v", err)
	}
	if c.ID == "" {
		t.Error("expected auto-generated ID, got empty")
	}
}

func TestListAssetChangesByProject(t *testing.T) {
	q, projID := setupAssetChangeTest(t)

	now := time.Now().UTC().Truncate(time.Second)
	for i := 0; i < 5; i++ {
		c := &models.AssetChange{
			ID: "ac-list-" + string(rune('a'+i)), ProjectID: projID, RunID: "run-1",
			AssetID: "asset-1", AssetValue: "10.0.0.1", AssetType: "ip",
			ChangeType: models.ChangeTypePortNew, ChangeSummary: "change",
			DetailJSON: "{}", Severity: "info", CreatedAt: now.Add(time.Duration(i) * time.Minute),
		}
		if err := q.CreateAssetChange(c); err != nil {
			t.Fatalf("CreateAssetChange %d: %v", i, err)
		}
	}

	// Default limit (<=0 → 50)
	all, err := q.ListAssetChangesByProject(projID, 0, 0)
	if err != nil {
		t.Fatalf("ListAssetChangesByProject(limit=0): %v", err)
	}
	if len(all) != 5 {
		t.Fatalf("expected 5 changes, got %d", len(all))
	}

	// Paginated: limit 2, offset 0
	page1, err := q.ListAssetChangesByProject(projID, 2, 0)
	if err != nil {
		t.Fatalf("page1: %v", err)
	}
	if len(page1) != 2 {
		t.Fatalf("page1: expected 2, got %d", len(page1))
	}

	// Paginated: limit 2, offset 4
	page3, err := q.ListAssetChangesByProject(projID, 2, 4)
	if err != nil {
		t.Fatalf("page3: %v", err)
	}
	if len(page3) != 1 {
		t.Fatalf("page3: expected 1, got %d", len(page3))
	}

	// Ordered by created_at DESC
	if page1[0].CreatedAt.Before(page1[1].CreatedAt) {
		t.Error("expected DESC order: first item should be newer")
	}
}

func TestListAssetChangesByAsset(t *testing.T) {
	q, projID := setupAssetChangeTest(t)

	now := time.Now().UTC().Truncate(time.Second)
	changes := []*models.AssetChange{
		{ID: "ac-a1", ProjectID: projID, RunID: "run-1", AssetID: "asset-X", AssetValue: "10.0.0.1", AssetType: "ip", ChangeType: models.ChangeTypePortNew, ChangeSummary: "a", DetailJSON: "{}", Severity: "info", CreatedAt: now},
		{ID: "ac-a2", ProjectID: projID, RunID: "run-1", AssetID: "asset-X", AssetValue: "10.0.0.1", AssetType: "ip", ChangeType: models.ChangeTypePortGone, ChangeSummary: "b", DetailJSON: "{}", Severity: "info", CreatedAt: now.Add(time.Second)},
		{ID: "ac-a3", ProjectID: projID, RunID: "run-1", AssetID: "asset-Y", AssetValue: "10.0.0.2", AssetType: "ip", ChangeType: models.ChangeTypePortNew, ChangeSummary: "c", DetailJSON: "{}", Severity: "info", CreatedAt: now},
	}
	for _, c := range changes {
		if err := q.CreateAssetChange(c); err != nil {
			t.Fatalf("CreateAssetChange %s: %v", c.ID, err)
		}
	}

	list, err := q.ListAssetChangesByAsset("asset-X")
	if err != nil {
		t.Fatalf("ListAssetChangesByAsset: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 changes for asset-X, got %d", len(list))
	}
	for _, c := range list {
		if c.AssetID != "asset-X" {
			t.Errorf("asset_id = %q, want asset-X", c.AssetID)
		}
	}

	// Non-existent asset
	empty, err := q.ListAssetChangesByAsset("nonexistent")
	if err != nil {
		t.Fatalf("ListAssetChangesByAsset(nonexistent): %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("expected 0 changes, got %d", len(empty))
	}
}

func TestGetAlertWebhook(t *testing.T) {
	q, projID := setupAssetChangeTest(t)

	// Not found → returns sql.ErrNoRows
	got, err := q.GetAlertWebhook(projID)
	if err != sql.ErrNoRows {
		t.Fatalf("GetAlertWebhook(not found): err = %v, want sql.ErrNoRows", err)
	}
	if got != nil {
		t.Errorf("expected nil webhook, got %+v", got)
	}

	// Insert then get
	now := time.Now().UTC().Truncate(time.Second)
	q.UpsertAlertWebhook(&models.AlertWebhook{
		ID: "wh-1", ProjectID: projID, Enabled: true,
		URL: "https://hooks.example.com/alert", Secret: "s3cret",
		MinSeverity: "high", OnNewAsset: true, OnAssetGone: true,
		OnPortChange: false, OnServiceChange: false, OnCertExpiry: true,
		CreatedAt: now, UpdatedAt: now,
	})

	got, err = q.GetAlertWebhook(projID)
	if err != nil {
		t.Fatalf("GetAlertWebhook: %v", err)
	}
	if got == nil {
		t.Fatal("GetAlertWebhook returned nil")
	}
	if got.ID != "wh-1" {
		t.Errorf("ID = %q, want wh-1", got.ID)
	}
	if got.URL != "https://hooks.example.com/alert" {
		t.Errorf("URL = %q, want hooks URL", got.URL)
	}
	if !got.Enabled {
		t.Error("expected Enabled = true")
	}
	if !got.OnNewAsset {
		t.Error("expected OnNewAsset = true")
	}
	if got.OnPortChange {
		t.Error("expected OnPortChange = false")
	}
	if got.MinSeverity != "high" {
		t.Errorf("MinSeverity = %q, want high", got.MinSeverity)
	}
}

func TestUpsertAlertWebhook_CreateThenUpdate(t *testing.T) {
	q, projID := setupAssetChangeTest(t)

	// First upsert → create
	w := &models.AlertWebhook{
		ID: "wh-up-1", ProjectID: projID, Enabled: true,
		URL: "https://original.example.com", Secret: "old",
		MinSeverity: "info", OnNewAsset: true, OnAssetGone: false,
		OnPortChange: true, OnServiceChange: false, OnCertExpiry: false,
	}
	if err := q.UpsertAlertWebhook(w); err != nil {
		t.Fatalf("UpsertAlertWebhook(create): %v", err)
	}

	got, err := q.GetAlertWebhook(projID)
	if err != nil {
		t.Fatalf("GetAlertWebhook: %v", err)
	}
	if got.URL != "https://original.example.com" {
		t.Errorf("URL = %q, want original", got.URL)
	}
	createdAt := got.CreatedAt

	// Second upsert → update (same project_id triggers ON CONFLICT)
	w2 := &models.AlertWebhook{
		ID: "wh-up-1", ProjectID: projID, Enabled: false,
		URL: "https://updated.example.com", Secret: "new",
		MinSeverity: "critical", OnNewAsset: false, OnAssetGone: true,
		OnPortChange: false, OnServiceChange: true, OnCertExpiry: true,
	}
	if err := q.UpsertAlertWebhook(w2); err != nil {
		t.Fatalf("UpsertAlertWebhook(update): %v", err)
	}

	got2, err := q.GetAlertWebhook(projID)
	if err != nil {
		t.Fatalf("GetAlertWebhook after update: %v", err)
	}
	if got2.URL != "https://updated.example.com" {
		t.Errorf("URL = %q, want updated URL", got2.URL)
	}
	if got2.Enabled {
		t.Error("expected Enabled = false after update")
	}
	if got2.MinSeverity != "critical" {
		t.Errorf("MinSeverity = %q, want critical", got2.MinSeverity)
	}
	if !got2.OnAssetGone {
		t.Error("expected OnAssetGone = true after update")
	}
	if !got2.OnServiceChange {
		t.Error("expected OnServiceChange = true after update")
	}
	// CreatedAt should be preserved from first insert
	if !got2.CreatedAt.Equal(createdAt) {
		t.Errorf("CreatedAt changed: %v → %v", createdAt, got2.CreatedAt)
	}
}

func TestDeleteAlertWebhook(t *testing.T) {
	q, projID := setupAssetChangeTest(t)

	// Insert
	now := time.Now().UTC().Truncate(time.Second)
	q.UpsertAlertWebhook(&models.AlertWebhook{
		ID: "wh-del", ProjectID: projID, Enabled: true,
		URL: "https://delete.me", Secret: "s",
		MinSeverity: "info", OnNewAsset: true, OnAssetGone: true,
		OnPortChange: true, OnServiceChange: true, OnCertExpiry: true,
		CreatedAt: now, UpdatedAt: now,
	})

	// Verify exists
	got, err := q.GetAlertWebhook(projID)
	if err != nil {
		t.Fatalf("GetAlertWebhook before delete: %v", err)
	}
	if got == nil {
		t.Fatal("webhook should exist before delete")
	}

	// Delete
	if err := q.DeleteAlertWebhook(projID); err != nil {
		t.Fatalf("DeleteAlertWebhook: %v", err)
	}

	// Verify gone
	got2, err := q.GetAlertWebhook(projID)
	if err != sql.ErrNoRows {
		t.Fatalf("GetAlertWebhook after delete: err = %v, want sql.ErrNoRows", err)
	}
	if got2 != nil {
		t.Errorf("expected nil after delete, got %+v", got2)
	}

	// Delete non-existent is not an error
	if err := q.DeleteAlertWebhook("nonexistent-project"); err != nil {
		t.Fatalf("DeleteAlertWebhook(nonexistent): %v", err)
	}
}
