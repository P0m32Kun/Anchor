package db

import (
	"database/sql"
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
	_ "github.com/mattn/go-sqlite3"
)

// openTestDB returns a fresh in-memory SQLite DB with all migrations applied.
// MaxOpenConns=1 keeps the in-memory database addressable across calls.
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	rawDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	rawDB.SetMaxOpenConns(1)
	t.Cleanup(func() { rawDB.Close() })
	if err := Migrate(rawDB); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return rawDB
}

func TestNucleiCustomSourceCRUD_RoundTrip(t *testing.T) {
	q := New(openTestDB(t))

	uri := "https://github.com/example/templates.git"
	branch := "main"
	now := time.Now().UTC().Truncate(time.Second)

	src := &models.NucleiCustomSource{
		ID:            "src-1",
		Name:          "demo",
		InstallPath:   "demo",
		Type:          models.NucleiCustomSourceTypeGit,
		URI:           &uri,
		Branch:        &branch,
		Enabled:       true,
		RoutingPolicy: "manual",
		Status:        models.NucleiCustomSourceStatusDraft,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := q.CreateNucleiCustomSource(src); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := q.GetNucleiCustomSource("src-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got == nil {
		t.Fatal("get returned nil for existing row")
	}
	if got.ID != src.ID || got.Name != src.Name {
		t.Errorf("identity mismatch: got %+v", got)
	}
	if got.Type != models.NucleiCustomSourceTypeGit {
		t.Errorf("type: want git, got %q", got.Type)
	}
	if got.URI == nil || *got.URI != uri {
		t.Errorf("uri: want %q, got %v", uri, got.URI)
	}
	if got.Branch == nil || *got.Branch != branch {
		t.Errorf("branch: want %q, got %v", branch, got.Branch)
	}
	if !got.Enabled {
		t.Error("enabled: want true")
	}
	if got.RoutingPolicy != "manual" {
		t.Errorf("routing_policy: want manual, got %q", got.RoutingPolicy)
	}
	if got.Status != models.NucleiCustomSourceStatusDraft {
		t.Errorf("status: want draft, got %q", got.Status)
	}
	if got.LastSyncAt != nil || got.LastValidateAt != nil || got.LastError != nil {
		t.Errorf("nullable fields should remain nil; got %+v", got)
	}

	syncAt := now.Add(time.Minute)
	validateAt := now.Add(2 * time.Minute)
	errMsg := "boom"
	got.Enabled = false
	got.Status = models.NucleiCustomSourceStatusReady
	got.LastSyncAt = &syncAt
	got.LastValidateAt = &validateAt
	got.LastError = &errMsg
	got.UpdatedAt = now.Add(3 * time.Minute)
	if err := q.UpdateNucleiCustomSource(got); err != nil {
		t.Fatalf("update: %v", err)
	}

	updated, err := q.GetNucleiCustomSource("src-1")
	if err != nil {
		t.Fatalf("get after update: %v", err)
	}
	if updated.Enabled {
		t.Error("enabled: want false after update")
	}
	if updated.Status != models.NucleiCustomSourceStatusReady {
		t.Errorf("status: want ready, got %q", updated.Status)
	}
	if updated.LastSyncAt == nil || !updated.LastSyncAt.Equal(syncAt) {
		t.Errorf("last_sync_at: want %v, got %v", syncAt, updated.LastSyncAt)
	}
	if updated.LastValidateAt == nil || !updated.LastValidateAt.Equal(validateAt) {
		t.Errorf("last_validate_at: want %v, got %v", validateAt, updated.LastValidateAt)
	}
	if updated.LastError == nil || *updated.LastError != errMsg {
		t.Errorf("last_error: want %q, got %v", errMsg, updated.LastError)
	}

	list, err := q.ListNucleiCustomSources()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("list len: want 1, got %d", len(list))
	}
	if list[0].ID != "src-1" {
		t.Errorf("list[0].ID: want src-1, got %q", list[0].ID)
	}

	if err := q.DeleteNucleiCustomSource("src-1"); err != nil {
		t.Fatalf("delete: %v", err)
	}

	gone, err := q.GetNucleiCustomSource("src-1")
	if err != nil {
		t.Fatalf("get after delete: %v", err)
	}
	if gone != nil {
		t.Errorf("get after delete: want nil, got %+v", gone)
	}

	emptyList, err := q.ListNucleiCustomSources()
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if len(emptyList) != 0 {
		t.Errorf("list after delete: want 0, got %d", len(emptyList))
	}
}

func TestGetNucleiCustomSource_NotFoundReturnsNilNil(t *testing.T) {
	q := New(openTestDB(t))
	got, err := q.GetNucleiCustomSource("missing")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got != nil {
		t.Errorf("want nil, got %+v", got)
	}
}

func TestNucleiCustomSource_NullableFieldsRoundTrip(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC().Truncate(time.Second)

	src := &models.NucleiCustomSource{
		ID:            "upload-1",
		Name:          "uploaded",
		InstallPath:   "uploaded",
		Type:          models.NucleiCustomSourceTypeUpload,
		Enabled:       true,
		RoutingPolicy: "manual",
		Status:        models.NucleiCustomSourceStatusReady,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := q.CreateNucleiCustomSource(src); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := q.GetNucleiCustomSource("upload-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got == nil {
		t.Fatal("got nil")
	}
	if got.URI != nil || got.Branch != nil ||
		got.LastSyncAt != nil || got.LastValidateAt != nil || got.LastError != nil {
		t.Errorf("expected all nullable fields nil, got %+v", got)
	}
}

func TestListNucleiCustomSources_OrderByCreatedAtDesc(t *testing.T) {
	q := New(openTestDB(t))
	base := time.Now().UTC().Truncate(time.Second)

	for i, id := range []string{"src-old", "src-mid", "src-new"} {
		src := &models.NucleiCustomSource{
			ID:            id,
			Name:          id,
			InstallPath:   id,
			Type:          models.NucleiCustomSourceTypeFile,
			Enabled:       true,
			RoutingPolicy: "manual",
			Status:        models.NucleiCustomSourceStatusDraft,
			CreatedAt:     base.Add(time.Duration(i) * time.Minute),
			UpdatedAt:     base.Add(time.Duration(i) * time.Minute),
		}
		if err := q.CreateNucleiCustomSource(src); err != nil {
			t.Fatalf("create %s: %v", id, err)
		}
	}

	list, err := q.ListNucleiCustomSources()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("len: want 3, got %d", len(list))
	}
	wantOrder := []string{"src-new", "src-mid", "src-old"}
	for i, want := range wantOrder {
		if list[i].ID != want {
			t.Errorf("position %d: want %q, got %q", i, want, list[i].ID)
		}
	}
}

func TestNucleiCustomBundleCRUD_RoundTrip(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC().Truncate(time.Second)
	activatedAt := now.Add(time.Minute)

	bundle := &models.NucleiCustomBundle{
		Version:      "v1",
		ManifestJSON: `{"sources":[]}`,
		ArchivePath:  "/tmp/bundles/v1.tar.gz",
		Status:       "active",
		CreatedAt:    now,
		ActivatedAt:  &activatedAt,
	}
	if err := q.CreateNucleiCustomBundle(bundle); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := q.GetNucleiCustomBundle("v1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got == nil {
		t.Fatal("get nil")
	}
	if got.Version != "v1" || got.Status != "active" {
		t.Errorf("identity: %+v", got)
	}
	if got.ActivatedAt == nil || !got.ActivatedAt.Equal(activatedAt) {
		t.Errorf("activated_at: want %v, got %v", activatedAt, got.ActivatedAt)
	}

	bundle2 := &models.NucleiCustomBundle{
		Version:      "v2",
		ManifestJSON: "{}",
		ArchivePath:  "/tmp/bundles/v2.tar.gz",
		Status:       "draft",
		CreatedAt:    now.Add(time.Hour),
	}
	if err := q.CreateNucleiCustomBundle(bundle2); err != nil {
		t.Fatalf("create v2: %v", err)
	}

	list, err := q.ListNucleiCustomBundles()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("len: want 2, got %d", len(list))
	}
	if list[0].Version != "v2" {
		t.Errorf("first: want v2, got %q", list[0].Version)
	}
	if list[1].ActivatedAt == nil {
		t.Error("v1 ActivatedAt should round-trip non-nil")
	}

	newActivated := now.Add(2 * time.Hour)
	if err := q.SetNucleiCustomBundleStatus("v2", "active", &newActivated); err != nil {
		t.Fatalf("set status: %v", err)
	}
	updated, err := q.GetNucleiCustomBundle("v2")
	if err != nil {
		t.Fatalf("get after set: %v", err)
	}
	if updated.Status != "active" {
		t.Errorf("status: want active, got %q", updated.Status)
	}
	if updated.ActivatedAt == nil || !updated.ActivatedAt.Equal(newActivated) {
		t.Errorf("activated_at after set: want %v, got %v", newActivated, updated.ActivatedAt)
	}
}

func TestGetNucleiCustomBundle_NotFoundReturnsNilNil(t *testing.T) {
	q := New(openTestDB(t))
	got, err := q.GetNucleiCustomBundle("nope")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got != nil {
		t.Errorf("want nil, got %+v", got)
	}
}

func TestMigrationV10_AddsScanTaskBundleVersionColumn(t *testing.T) {
	rawDB := openTestDB(t)
	rows, err := rawDB.Query(`SELECT name FROM pragma_table_info('scan_tasks')`)
	if err != nil {
		t.Fatalf("pragma_table_info: %v", err)
	}
	defer rows.Close()
	found := false
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if name == "nuclei_custom_bundle_version" {
			found = true
			break
		}
	}
	if !found {
		t.Error("scan_tasks.nuclei_custom_bundle_version column missing after migration v10")
	}
}
