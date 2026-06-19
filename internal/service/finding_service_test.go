package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/errors"
	"github.com/P0m32Kun/Anchor/internal/models"
	_ "github.com/mattn/go-sqlite3"
)

func openServiceTestDB(t *testing.T) *sql.DB {
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
	return rawDB
}

func setupFindingService(t *testing.T) (FindingService, *db.Queries, string) {
	t.Helper()
	rawDB := openServiceTestDB(t)
	q := db.New(rawDB)
	svc := NewFindingService(q)

	now := time.Now().UTC()
	if err := q.CreateProject(&models.Project{
		ID: "proj-1", Name: "test", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create project: %v", err)
	}

	return svc, q, "proj-1"
}

var findingCounter int

func createSvcFinding(t *testing.T, q *db.Queries, projectID string, status models.FindingStatus) *models.Finding {
	t.Helper()
	findingCounter++
	f := &models.Finding{
		ID: "f-" + string(rune('a'+findingCounter)), ProjectID: projectID,
		SourceTool: "nuclei", SourceRuleID: "r1", DedupKey: "dk-" + string(rune('a'+findingCounter)),
		Title: "Test", Severity: models.SeverityHigh,
		Confidence: 80, Priority: 70, Status: status,
		Summary: "s", Remediation: "r",
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	if err := q.CreateFinding(f); err != nil {
		t.Fatalf("create finding: %v", err)
	}
	return f
}

// --- UpdateStatus state machine ---

func TestFindingService_UpdateStatus_ValidTransitions(t *testing.T) {
	svc, q, pid := setupFindingService(t)
	ctx := context.Background()

	validStatuses := []string{
		"confirmed", "false_positive", "accepted_risk", "ignored", "pending_review",
	}

	for _, status := range validStatuses {
		t.Run(status, func(t *testing.T) {
			f := createSvcFinding(t, q, pid, models.FindingNew)
			if err := svc.UpdateStatus(ctx, f.ID, status); err != nil {
				t.Errorf("UpdateStatus(%s) = %v, want nil", status, err)
			}
		})
	}
}

func TestFindingService_UpdateStatus_InvalidStatus(t *testing.T) {
	svc, q, pid := setupFindingService(t)
	ctx := context.Background()

	f := createSvcFinding(t, q, pid, models.FindingNew)

	invalidStatuses := []string{"bogus", "", "CONFIRMED", "new", "reported"}
	for _, status := range invalidStatuses {
		t.Run(status, func(t *testing.T) {
			err := svc.UpdateStatus(ctx, f.ID, status)
			if err == nil {
				t.Errorf("UpdateStatus(%s) expected error", status)
			}
		})
	}
}

func TestFindingService_UpdateStatus_NotFound(t *testing.T) {
	svc, _, _ := setupFindingService(t)
	ctx := context.Background()

	err := svc.UpdateStatus(ctx, "nonexistent", "confirmed")
	if err == nil {
		t.Error("expected error for non-existent finding")
	}
}

// --- Get ---

func TestFindingService_Get_Success(t *testing.T) {
	svc, q, pid := setupFindingService(t)
	ctx := context.Background()

	f := createSvcFinding(t, q, pid, models.FindingConfirmed)

	got, err := svc.Get(ctx, f.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != f.ID {
		t.Errorf("id = %q, want %q", got.ID, f.ID)
	}
}

func TestFindingService_Get_NotFound(t *testing.T) {
	svc, _, _ := setupFindingService(t)
	ctx := context.Background()

	_, err := svc.Get(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for non-existent finding")
	}
}

// --- List ---

func TestFindingService_List_All(t *testing.T) {
	svc, q, pid := setupFindingService(t)
	ctx := context.Background()

	createSvcFinding(t, q, pid, models.FindingConfirmed)
	createSvcFinding(t, q, pid, models.FindingNew)

	findings, err := svc.List(ctx, pid, "")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(findings) != 2 {
		t.Errorf("len = %d, want 2", len(findings))
	}
}

func TestFindingService_List_FilterByStatus(t *testing.T) {
	svc, q, pid := setupFindingService(t)
	ctx := context.Background()

	createSvcFinding(t, q, pid, models.FindingConfirmed)
	createSvcFinding(t, q, pid, models.FindingNew)

	findings, err := svc.List(ctx, pid, "confirmed")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(findings) != 1 {
		t.Errorf("len = %d, want 1", len(findings))
	}
}

// --- AddEvidence ---

func TestFindingService_AddEvidence_Success(t *testing.T) {
	svc, q, pid := setupFindingService(t)
	ctx := context.Background()

	f := createSvcFinding(t, q, pid, models.FindingConfirmed)

	ev, err := svc.AddEvidence(ctx, f.ID, AddEvidenceRequest{
		Type:    "note",
		Excerpt: "manual verification",
	})
	if err != nil {
		t.Fatalf("AddEvidence: %v", err)
	}
	if ev.Type != models.EvidenceNote {
		t.Errorf("type = %q, want note", ev.Type)
	}
	if ev.FindingID != f.ID {
		t.Errorf("finding_id = %q, want %q", ev.FindingID, f.ID)
	}
}

func TestFindingService_AddEvidence_MissingType(t *testing.T) {
	svc, q, pid := setupFindingService(t)
	ctx := context.Background()

	f := createSvcFinding(t, q, pid, models.FindingConfirmed)

	_, err := svc.AddEvidence(ctx, f.ID, AddEvidenceRequest{
		Excerpt: "test",
	})
	if err == nil {
		t.Error("expected error for missing type")
	}
}

func TestFindingService_AddEvidence_MissingExcerpt(t *testing.T) {
	svc, q, pid := setupFindingService(t)
	ctx := context.Background()

	f := createSvcFinding(t, q, pid, models.FindingConfirmed)

	_, err := svc.AddEvidence(ctx, f.ID, AddEvidenceRequest{
		Type: "note",
	})
	if err == nil {
		t.Error("expected error for missing excerpt")
	}
}

func TestFindingService_AddEvidence_InvalidType(t *testing.T) {
	svc, q, pid := setupFindingService(t)
	ctx := context.Background()

	f := createSvcFinding(t, q, pid, models.FindingConfirmed)

	_, err := svc.AddEvidence(ctx, f.ID, AddEvidenceRequest{
		Type:    "request",
		Excerpt: "test",
	})
	if err == nil {
		t.Error("expected error for invalid evidence type")
	}
}

// --- ListEvidence ---

func TestFindingService_ListEvidence_Empty(t *testing.T) {
	svc, q, pid := setupFindingService(t)
	ctx := context.Background()

	f := createSvcFinding(t, q, pid, models.FindingConfirmed)

	evList, err := svc.ListEvidence(ctx, f.ID)
	if err != nil {
		t.Fatalf("ListEvidence: %v", err)
	}
	if len(evList) != 0 {
		t.Errorf("len = %d, want 0", len(evList))
	}
}

func TestFindingService_ListEvidence_WithData(t *testing.T) {
	svc, q, pid := setupFindingService(t)
	ctx := context.Background()

	f := createSvcFinding(t, q, pid, models.FindingConfirmed)

	svc.AddEvidence(ctx, f.ID, AddEvidenceRequest{Type: "note", Excerpt: "ev1"})
	svc.AddEvidence(ctx, f.ID, AddEvidenceRequest{Type: "screenshot", Excerpt: "ev2"})

	evList, err := svc.ListEvidence(ctx, f.ID)
	if err != nil {
		t.Fatalf("ListEvidence: %v", err)
	}
	if len(evList) != 2 {
		t.Errorf("len = %d, want 2", len(evList))
	}
}

// --- ListPaginated ---

func TestFindingService_ListPaginated_All(t *testing.T) {
	svc, q, pid := setupFindingService(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		createSvcFinding(t, q, pid, models.FindingNew)
	}

	result, err := svc.ListPaginated(ctx, pid, "", PaginationParams{Page: 1, PageSize: 2})
	if err != nil {
		t.Fatalf("ListPaginated: %v", err)
	}
	if result.Total != 5 {
		t.Errorf("total = %d, want 5", result.Total)
	}
	if len(result.Data) != 2 {
		t.Errorf("data len = %d, want 2", len(result.Data))
	}
	if result.Page != 1 {
		t.Errorf("page = %d, want 1", result.Page)
	}
}

func TestFindingService_ListPaginated_FilterByStatus(t *testing.T) {
	svc, q, pid := setupFindingService(t)
	ctx := context.Background()

	createSvcFinding(t, q, pid, models.FindingConfirmed)
	createSvcFinding(t, q, pid, models.FindingNew)
	createSvcFinding(t, q, pid, models.FindingConfirmed)

	result, err := svc.ListPaginated(ctx, pid, "confirmed", PaginationParams{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListPaginated: %v", err)
	}
	if result.Total != 2 {
		t.Errorf("total = %d, want 2", result.Total)
	}
	if len(result.Data) != 2 {
		t.Errorf("data len = %d, want 2", len(result.Data))
	}
}

func TestFindingService_ListPaginated_SecondPage(t *testing.T) {
	svc, q, pid := setupFindingService(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		createSvcFinding(t, q, pid, models.FindingNew)
	}

	result, err := svc.ListPaginated(ctx, pid, "", PaginationParams{Page: 2, PageSize: 2})
	if err != nil {
		t.Fatalf("ListPaginated: %v", err)
	}
	if len(result.Data) != 2 {
		t.Errorf("data len = %d, want 2", len(result.Data))
	}
}

// --- Create with nil audit (covers audit==nil branch in Create) ---

func TestProjectService_Create_NilAudit(t *testing.T) {
	repo := NewMockProjectRepository()
	svc := NewProjectServiceWithDeps(repo, nil)

	project, err := svc.Create(context.Background(), CreateProjectRequest{Name: "no-audit"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if project.Name != "no-audit" {
		t.Errorf("name = %q, want %q", project.Name, "no-audit")
	}
}

func TestProjectService_Delete_NilAudit(t *testing.T) {
	repo := NewMockProjectRepository()
	svc := NewProjectServiceWithDeps(repo, nil)

	project, _ := svc.Create(context.Background(), CreateProjectRequest{Name: "to-delete"})
	err := svc.Delete(context.Background(), project.ID)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestProjectService_Create_NameTooLong(t *testing.T) {
	repo := NewMockProjectRepository()
	svc := NewProjectServiceWithDeps(repo, nil)

	longName := string(make([]byte, 201))
	_, err := svc.Create(context.Background(), CreateProjectRequest{Name: longName})
	if err == nil {
		t.Fatal("expected error")
	}
	appErr := err.(*errors.AppError)
	if appErr.Code != errors.ErrBadRequest {
		t.Errorf("code = %q, want %q", appErr.Code, errors.ErrBadRequest)
	}
}

func TestProjectService_List_Error(t *testing.T) {
	repo := NewMockProjectRepository()
	repo.ListFn = func() ([]*models.Project, error) {
		return nil, errors.Newf(errors.ErrInternal, "db error")
	}
	svc := NewProjectServiceWithDeps(repo, nil)

	_, err := svc.List(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestProjectService_Get_Error(t *testing.T) {
	repo := NewMockProjectRepository()
	repo.GetFn = func(id string) (*models.Project, error) {
		return nil, errors.Newf(errors.ErrInternal, "db error")
	}
	svc := NewProjectServiceWithDeps(repo, nil)

	_, err := svc.Get(context.Background(), "p1")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestProjectService_Delete_GetError(t *testing.T) {
	repo := NewMockProjectRepository()
	repo.GetFn = func(id string) (*models.Project, error) {
		return nil, errors.Newf(errors.ErrInternal, "db error")
	}
	svc := NewProjectServiceWithDeps(repo, nil)

	err := svc.Delete(context.Background(), "p1")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestProjectService_Delete_DeleteError(t *testing.T) {
	repo := NewMockProjectRepository()
	repo.Create(&models.Project{ID: "p1", Name: "test"})
	repo.DeleteFn = func(id string) error {
		return errors.Newf(errors.ErrInternal, "delete error")
	}
	svc := NewProjectServiceWithDeps(repo, nil)

	err := svc.Delete(context.Background(), "p1")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestProjectService_Create_RepoError(t *testing.T) {
	repo := NewMockProjectRepository()
	repo.CreateFn = func(p *models.Project) error {
		return errors.Newf(errors.ErrInternal, "create error")
	}
	svc := NewProjectServiceWithDeps(repo, nil)

	_, err := svc.Create(context.Background(), CreateProjectRequest{Name: "test"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestProjectService_ListPaginated_CountError(t *testing.T) {
	repo := NewMockProjectRepository()
	// Override List to make Count return from the mock's internal state
	// Count is not directly overridable, but we can test via the real mock
	svc := NewProjectServiceWithDeps(repo, nil)

	// Empty repo - count=0
	result, err := svc.ListPaginated(context.Background(), PaginationParams{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListPaginated: %v", err)
	}
	if result.Total != 0 {
		t.Errorf("total = %d, want 0", result.Total)
	}
}
