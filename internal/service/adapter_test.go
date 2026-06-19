package service

import (
	"database/sql"
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/models"
	_ "github.com/mattn/go-sqlite3"
)

func openAdapterTestDB(t *testing.T) (*sql.DB, *db.Queries) {
	t.Helper()
	rawDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	rawDB.SetMaxOpenConns(1)
	t.Cleanup(func() { rawDB.Close() })
	if err := db.Migrate(rawDB); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return rawDB, db.New(rawDB)
}

func createTestProject(t *testing.T, q *db.Queries, id, name string) *models.Project {
	t.Helper()
	now := time.Now().UTC()
	p := &models.Project{
		ID: id, Name: name, RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	}
	if err := q.CreateProject(p); err != nil {
		t.Fatalf("create project: %v", err)
	}
	return p
}

// --- projectRepoAdapter ---

func TestProjectRepoAdapter_Create(t *testing.T) {
	_, q := openAdapterTestDB(t)
	adapter := &projectRepoAdapter{queries: q}

	now := time.Now().UTC()
	p := &models.Project{
		ID: "p1", Name: "test", RateLimit: 5,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	}
	if err := adapter.Create(p); err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, err := adapter.Get("p1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != "test" {
		t.Errorf("name = %q, want %q", got.Name, "test")
	}
}

func TestProjectRepoAdapter_Get(t *testing.T) {
	_, q := openAdapterTestDB(t)
	adapter := &projectRepoAdapter{queries: q}

	createTestProject(t, q, "p1", "proj1")

	got, err := adapter.Get("p1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != "p1" {
		t.Errorf("id = %q, want %q", got.ID, "p1")
	}

	// Not found
	missing, err := adapter.Get("nonexistent")
	if err != nil {
		t.Fatalf("Get missing: %v", err)
	}
	if missing != nil {
		t.Errorf("expected nil, got %v", missing)
	}
}

func TestProjectRepoAdapter_List(t *testing.T) {
	_, q := openAdapterTestDB(t)
	adapter := &projectRepoAdapter{queries: q}

	createTestProject(t, q, "p1", "a")
	createTestProject(t, q, "p2", "b")

	list, err := adapter.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("len = %d, want 2", len(list))
	}
}

func TestProjectRepoAdapter_Count(t *testing.T) {
	_, q := openAdapterTestDB(t)
	adapter := &projectRepoAdapter{queries: q}

	createTestProject(t, q, "p1", "a")
	createTestProject(t, q, "p2", "b")

	count, err := adapter.Count()
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}

func TestProjectRepoAdapter_ListPaginated(t *testing.T) {
	_, q := openAdapterTestDB(t)
	adapter := &projectRepoAdapter{queries: q}

	createTestProject(t, q, "p1", "a")
	createTestProject(t, q, "p2", "b")
	createTestProject(t, q, "p3", "c")

	list, err := adapter.ListPaginated(2, 0)
	if err != nil {
		t.Fatalf("ListPaginated: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("len = %d, want 2", len(list))
	}

	list, err = adapter.ListPaginated(2, 2)
	if err != nil {
		t.Fatalf("ListPaginated offset: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("len = %d, want 1", len(list))
	}
}

func TestProjectRepoAdapter_Delete(t *testing.T) {
	_, q := openAdapterTestDB(t)
	adapter := &projectRepoAdapter{queries: q}

	createTestProject(t, q, "p1", "test")

	if err := adapter.Delete("p1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	got, _ := adapter.Get("p1")
	if got != nil {
		t.Error("expected nil after delete")
	}
}

// --- targetRepoAdapter ---

func TestTargetRepoAdapter_Create(t *testing.T) {
	rawDB, q := openAdapterTestDB(t)
	adapter := &targetRepoAdapter{queries: q, rawDB: rawDB}

	createTestProject(t, q, "p1", "proj")

	now := time.Now().UTC()
	tgt := &models.Target{
		ID: "t1", ProjectID: "p1", Type: models.TargetTypeDomain,
		Value: "example.com", Source: "manual", Status: "active", CreatedAt: now,
	}
	if err := adapter.Create(tgt); err != nil {
		t.Fatalf("Create: %v", err)
	}

	list, _ := adapter.ListByProject("p1")
	if len(list) != 1 {
		t.Errorf("len = %d, want 1", len(list))
	}
}

func TestTargetRepoAdapter_ListByProject(t *testing.T) {
	rawDB, q := openAdapterTestDB(t)
	adapter := &targetRepoAdapter{queries: q, rawDB: rawDB}

	createTestProject(t, q, "p1", "proj")

	now := time.Now().UTC()
	for i, v := range []string{"a.com", "b.com"} {
		q.CreateTarget(&models.Target{
			ID: "t" + string(rune('1'+i)), ProjectID: "p1", Type: models.TargetTypeDomain,
			Value: v, Source: "manual", Status: "active", CreatedAt: now,
		})
	}

	list, err := adapter.ListByProject("p1")
	if err != nil {
		t.Fatalf("ListByProject: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("len = %d, want 2", len(list))
	}
}

func TestTargetRepoAdapter_CountByProject(t *testing.T) {
	rawDB, q := openAdapterTestDB(t)
	adapter := &targetRepoAdapter{queries: q, rawDB: rawDB}

	createTestProject(t, q, "p1", "proj")

	now := time.Now().UTC()
	q.CreateTarget(&models.Target{
		ID: "t1", ProjectID: "p1", Type: models.TargetTypeDomain,
		Value: "a.com", Source: "manual", Status: "active", CreatedAt: now,
	})

	count, err := adapter.CountByProject("p1")
	if err != nil {
		t.Fatalf("CountByProject: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
}

func TestTargetRepoAdapter_ListByProjectPaginated(t *testing.T) {
	rawDB, q := openAdapterTestDB(t)
	adapter := &targetRepoAdapter{queries: q, rawDB: rawDB}

	createTestProject(t, q, "p1", "proj")

	now := time.Now().UTC()
	for i := 0; i < 5; i++ {
		q.CreateTarget(&models.Target{
			ID: "t" + string(rune('1'+i)), ProjectID: "p1", Type: models.TargetTypeDomain,
			Value: string(rune('a'+i)) + ".com", Source: "manual", Status: "active", CreatedAt: now,
		})
	}

	list, err := adapter.ListByProjectPaginated("p1", 2, 0)
	if err != nil {
		t.Fatalf("ListByProjectPaginated: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("len = %d, want 2", len(list))
	}
}

func TestTargetRepoAdapter_ExistsByValue(t *testing.T) {
	rawDB, q := openAdapterTestDB(t)
	adapter := &targetRepoAdapter{queries: q, rawDB: rawDB}

	createTestProject(t, q, "p1", "proj")

	now := time.Now().UTC()
	q.CreateTarget(&models.Target{
		ID: "t1", ProjectID: "p1", Type: models.TargetTypeDomain,
		Value: "exists.com", Source: "manual", Status: "active", CreatedAt: now,
	})

	exists, err := adapter.ExistsByValue("p1", "exists.com")
	if err != nil {
		t.Fatalf("ExistsByValue: %v", err)
	}
	if !exists {
		t.Error("expected exists=true")
	}

	exists, err = adapter.ExistsByValue("p1", "nope.com")
	if err != nil {
		t.Fatalf("ExistsByValue: %v", err)
	}
	if exists {
		t.Error("expected exists=false")
	}
}

func TestTargetRepoAdapter_BulkCreate(t *testing.T) {
	rawDB, q := openAdapterTestDB(t)
	adapter := &targetRepoAdapter{queries: q, rawDB: rawDB}

	createTestProject(t, q, "p1", "proj")

	now := time.Now().UTC()
	targets := []*models.Target{
		{ID: "t1", ProjectID: "p1", Type: models.TargetTypeDomain, Value: "a.com", Source: "import", Status: "active", CreatedAt: now},
		{ID: "t2", ProjectID: "p1", Type: models.TargetTypeDomain, Value: "b.com", Source: "import", Status: "active", CreatedAt: now},
	}
	if err := adapter.BulkCreate(targets); err != nil {
		t.Fatalf("BulkCreate: %v", err)
	}

	list, _ := adapter.ListByProject("p1")
	if len(list) != 2 {
		t.Errorf("len = %d, want 2", len(list))
	}
}

// --- findingRepoAdapter ---

func createTestFinding(t *testing.T, q *db.Queries, projectID, id string) {
	t.Helper()
	now := time.Now().UTC()
	q.CreateFinding(&models.Finding{
		ID: id, ProjectID: projectID,
		SourceTool: "nuclei", SourceRuleID: "r1", DedupKey: "dk-" + id,
		Title: "Test", Severity: models.SeverityHigh,
		Confidence: 80, Priority: 70, Status: models.FindingNew,
		Summary: "s", Remediation: "r",
		CreatedAt: now, UpdatedAt: now,
	})
}

func TestFindingRepoAdapter_Get(t *testing.T) {
	_, q := openAdapterTestDB(t)
	adapter := &findingRepoAdapter{queries: q}

	createTestProject(t, q, "p1", "proj")
	createTestFinding(t, q, "p1", "f1")

	got, err := adapter.Get("f1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != "f1" {
		t.Errorf("id = %q, want %q", got.ID, "f1")
	}

	missing, err := adapter.Get("nonexistent")
	if err != nil {
		t.Fatalf("Get missing: %v", err)
	}
	if missing != nil {
		t.Error("expected nil")
	}
}

func TestFindingRepoAdapter_ListByProject(t *testing.T) {
	_, q := openAdapterTestDB(t)
	adapter := &findingRepoAdapter{queries: q}

	createTestProject(t, q, "p1", "proj")
	createTestFinding(t, q, "p1", "f1")
	createTestFinding(t, q, "p1", "f2")

	list, err := adapter.ListByProject("p1")
	if err != nil {
		t.Fatalf("ListByProject: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("len = %d, want 2", len(list))
	}
}

func TestFindingRepoAdapter_ListByStatus(t *testing.T) {
	_, q := openAdapterTestDB(t)
	adapter := &findingRepoAdapter{queries: q}

	createTestProject(t, q, "p1", "proj")
	createTestFinding(t, q, "p1", "f1")

	list, err := adapter.ListByStatus("p1", models.FindingNew)
	if err != nil {
		t.Fatalf("ListByStatus: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("len = %d, want 1", len(list))
	}
}

func TestFindingRepoAdapter_CountByProject(t *testing.T) {
	_, q := openAdapterTestDB(t)
	adapter := &findingRepoAdapter{queries: q}

	createTestProject(t, q, "p1", "proj")
	createTestFinding(t, q, "p1", "f1")
	createTestFinding(t, q, "p1", "f2")

	count, err := adapter.CountByProject("p1", "")
	if err != nil {
		t.Fatalf("CountByProject: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}

	count, err = adapter.CountByProject("p1", models.FindingNew)
	if err != nil {
		t.Fatalf("CountByProject status: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}

func TestFindingRepoAdapter_ListByProjectPaginated(t *testing.T) {
	_, q := openAdapterTestDB(t)
	adapter := &findingRepoAdapter{queries: q}

	createTestProject(t, q, "p1", "proj")
	for i := 0; i < 5; i++ {
		createTestFinding(t, q, "p1", "f"+string(rune('1'+i)))
	}

	list, err := adapter.ListByProjectPaginated("p1", 2, 0)
	if err != nil {
		t.Fatalf("ListByProjectPaginated: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("len = %d, want 2", len(list))
	}
}

func TestFindingRepoAdapter_ListByStatusPaginated(t *testing.T) {
	_, q := openAdapterTestDB(t)
	adapter := &findingRepoAdapter{queries: q}

	createTestProject(t, q, "p1", "proj")
	for i := 0; i < 5; i++ {
		createTestFinding(t, q, "p1", "f"+string(rune('1'+i)))
	}

	list, err := adapter.ListByStatusPaginated("p1", models.FindingNew, 3, 0)
	if err != nil {
		t.Fatalf("ListByStatusPaginated: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("len = %d, want 3", len(list))
	}
}

func TestFindingRepoAdapter_UpdateStatus(t *testing.T) {
	_, q := openAdapterTestDB(t)
	adapter := &findingRepoAdapter{queries: q}

	createTestProject(t, q, "p1", "proj")
	createTestFinding(t, q, "p1", "f1")

	if err := adapter.UpdateStatus("f1", models.FindingConfirmed); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}
	got, _ := adapter.Get("f1")
	if got.Status != models.FindingConfirmed {
		t.Errorf("status = %q, want %q", got.Status, models.FindingConfirmed)
	}
}

func TestFindingRepoAdapter_CreateEvidence(t *testing.T) {
	_, q := openAdapterTestDB(t)
	adapter := &findingRepoAdapter{queries: q}

	createTestProject(t, q, "p1", "proj")
	createTestFinding(t, q, "p1", "f1")

	now := time.Now().UTC()
	ev := &models.Evidence{
		ID: "e1", FindingID: "f1", Type: models.EvidenceNote,
		Excerpt: "test", CreatedAt: now,
	}
	if err := adapter.CreateEvidence(ev); err != nil {
		t.Fatalf("CreateEvidence: %v", err)
	}

	list, err := adapter.ListEvidenceByFinding("f1")
	if err != nil {
		t.Fatalf("ListEvidenceByFinding: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("len = %d, want 1", len(list))
	}
}

// --- auditLogAdapter ---

func TestAuditLogAdapter_Create(t *testing.T) {
	_, q := openAdapterTestDB(t)
	adapter := &auditLogAdapter{queries: q}

	createTestProject(t, q, "p1", "proj")

	now := time.Now().UTC()
	log := &models.AuditLog{
		ID: "a1", ProjectID: "p1", Actor: "user",
		Action: "test", ResourceType: "project", ResourceID: "p1",
		Summary: "test log", CreatedAt: now,
	}
	if err := adapter.Create(log); err != nil {
		t.Fatalf("Create: %v", err)
	}
}

// --- NewProjectService / NewFindingService convenience constructors ---

func TestNewProjectService(t *testing.T) {
	_, q := openAdapterTestDB(t)
	svc := NewProjectService(q)
	if svc == nil {
		t.Fatal("expected non-nil")
	}

	// Verify it works end-to-end
	proj, err := svc.Create(nil, CreateProjectRequest{Name: "test"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if proj.Name != "test" {
		t.Errorf("name = %q, want %q", proj.Name, "test")
	}
}

func TestNewFindingService(t *testing.T) {
	_, q := openAdapterTestDB(t)
	svc := NewFindingService(q)
	if svc == nil {
		t.Fatal("expected non-nil")
	}

	// Create a project and finding to verify the service works
	createTestProject(t, q, "p1", "proj")
	createTestFinding(t, q, "p1", "f1")

	findings, err := svc.List(nil, "p1", "")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(findings) != 1 {
		t.Errorf("len = %d, want 1", len(findings))
	}
}
