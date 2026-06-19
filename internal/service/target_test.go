package service

import (
	"context"
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/scope"
	_ "github.com/mattn/go-sqlite3"
)

func setupTargetService(t *testing.T) (TargetService, *db.Queries, string) {
	t.Helper()
	rawDB := openServiceTestDB(t)
	q := db.New(rawDB)
	scopeEng := scope.NewEngine(q)
	svc := NewTargetService(q, rawDB, scopeEng)

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

func addScopeRule(t *testing.T, q *db.Queries, projectID string) {
	t.Helper()
	now := time.Now().UTC()
	if err := q.CreateScopeRule(&models.ScopeRule{
		ID: "sr1", ProjectID: projectID, Action: models.ScopeAction("include"),
		Type: models.TargetTypeDomain, Value: "*.example.com",
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("create scope rule: %v", err)
	}
}

// --- NewTargetService ---

func TestNewTargetService(t *testing.T) {
	svc, _, _ := setupTargetService(t)
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

// --- Create ---

func TestTargetService_Create_WithScopeRule(t *testing.T) {
	svc, q, pid := setupTargetService(t)
	ctx := context.Background()

	addScopeRule(t, q, pid)

	resp, err := svc.Create(ctx, pid, CreateTargetRequest{
		Type: "domain", Value: "test.example.com",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if resp.Target == nil {
		t.Fatal("expected target in response")
	}
	if resp.Target.Value != "test.example.com" {
		t.Errorf("value = %q, want %q", resp.Target.Value, "test.example.com")
	}
	if resp.NeedsScopeConfirmation {
		t.Error("expected NeedsScopeConfirmation=false")
	}
}

func TestTargetService_Create_AutoType(t *testing.T) {
	svc, q, pid := setupTargetService(t)
	ctx := context.Background()

	addScopeRule(t, q, pid)

	resp, err := svc.Create(ctx, pid, CreateTargetRequest{
		Type: "auto", Value: "example.com",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if resp.Target == nil {
		t.Fatal("expected target")
	}
	// domain should be auto-detected
	if resp.Target.Type != models.TargetTypeDomain {
		t.Errorf("type = %q, want domain", resp.Target.Type)
	}
}

func TestTargetService_Create_NoScopeRules_SuggestsConfirmation(t *testing.T) {
	svc, _, pid := setupTargetService(t)
	ctx := context.Background()

	resp, err := svc.Create(ctx, pid, CreateTargetRequest{
		Type: "domain", Value: "example.com",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if !resp.NeedsScopeConfirmation {
		t.Error("expected NeedsScopeConfirmation=true")
	}
	if resp.SuggestedRule == nil {
		t.Fatal("expected suggested rule")
	}
	if resp.SuggestedRule.Action != "include" {
		t.Errorf("action = %q, want %q", resp.SuggestedRule.Action, "include")
	}
	if resp.SuggestedRule.Value != "example.com" {
		t.Errorf("value = %q, want %q", resp.SuggestedRule.Value, "example.com")
	}
}

func TestTargetService_Create_NoScopeRules_IP_SuggestsCIDR(t *testing.T) {
	svc, _, pid := setupTargetService(t)
	ctx := context.Background()

	resp, err := svc.Create(ctx, pid, CreateTargetRequest{
		Type: "ip", Value: "192.168.1.1",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if !resp.NeedsScopeConfirmation {
		t.Error("expected NeedsScopeConfirmation=true")
	}
	if resp.SuggestedRule == nil {
		t.Fatal("expected suggested rule")
	}
	if resp.SuggestedRule.Type != string(models.TargetTypeCIDR) {
		t.Errorf("type = %q, want cidr", resp.SuggestedRule.Type)
	}
	if resp.SuggestedRule.Value != "192.168.1.1/32" {
		t.Errorf("value = %q, want %q", resp.SuggestedRule.Value, "192.168.1.1/32")
	}
}

// --- List ---

func TestTargetService_List(t *testing.T) {
	svc, q, pid := setupTargetService(t)
	ctx := context.Background()

	now := time.Now().UTC()
	for i, v := range []string{"a.com", "b.com"} {
		q.CreateTarget(&models.Target{
			ID: "t" + string(rune('1'+i)), ProjectID: pid, Type: models.TargetTypeDomain,
			Value: v, Source: "manual", Status: "active", CreatedAt: now,
		})
	}

	list, err := svc.List(ctx, pid)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("len = %d, want 2", len(list))
	}
}

// --- ListPaginated ---

func TestTargetService_ListPaginated(t *testing.T) {
	svc, q, pid := setupTargetService(t)
	ctx := context.Background()

	now := time.Now().UTC()
	for i := 0; i < 5; i++ {
		q.CreateTarget(&models.Target{
			ID: "t" + string(rune('1'+i)), ProjectID: pid, Type: models.TargetTypeDomain,
			Value: string(rune('a'+i)) + ".com", Source: "manual", Status: "active", CreatedAt: now,
		})
	}

	result, err := svc.ListPaginated(ctx, pid, PaginationParams{Page: 1, PageSize: 2})
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

	// Page 2
	result, err = svc.ListPaginated(ctx, pid, PaginationParams{Page: 2, PageSize: 2})
	if err != nil {
		t.Fatalf("ListPaginated page 2: %v", err)
	}
	if len(result.Data) != 2 {
		t.Errorf("data len = %d, want 2", len(result.Data))
	}
}

// --- Import ---

func TestTargetService_Import_WithScopeRules(t *testing.T) {
	svc, q, pid := setupTargetService(t)
	ctx := context.Background()

	addScopeRule(t, q, pid)

	result, err := svc.Import(ctx, pid, []ImportTarget{
		{Type: models.TargetTypeDomain, Value: "a.example.com"},
		{Type: models.TargetTypeDomain, Value: "b.example.com"},
	})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if result.Imported != 2 {
		t.Errorf("imported = %d, want 2", result.Imported)
	}
	if len(result.Targets) != 2 {
		t.Errorf("targets len = %d, want 2", len(result.Targets))
	}
}

func TestTargetService_Import_NoScopeRules_SuggestsConfirmation(t *testing.T) {
	svc, _, pid := setupTargetService(t)
	ctx := context.Background()

	result, err := svc.Import(ctx, pid, []ImportTarget{
		{Type: models.TargetTypeDomain, Value: "a.com"},
	})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if !result.NeedsScopeConfirmation {
		t.Error("expected NeedsScopeConfirmation=true")
	}
	if len(result.SuggestedRules) != 1 {
		t.Errorf("suggested rules len = %d, want 1", len(result.SuggestedRules))
	}
}

func TestTargetService_Import_Duplicates(t *testing.T) {
	svc, q, pid := setupTargetService(t)
	ctx := context.Background()

	addScopeRule(t, q, pid)

	// First import
	svc.Import(ctx, pid, []ImportTarget{
		{Type: models.TargetTypeDomain, Value: "dup.com"},
	})

	// Second import with same value
	result, err := svc.Import(ctx, pid, []ImportTarget{
		{Type: models.TargetTypeDomain, Value: "dup.com"},
	})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if result.Duplicates != 1 {
		t.Errorf("duplicates = %d, want 1", result.Duplicates)
	}
}

func TestTargetService_Import_EmptyValue(t *testing.T) {
	svc, q, pid := setupTargetService(t)
	ctx := context.Background()

	addScopeRule(t, q, pid)

	result, err := svc.Import(ctx, pid, []ImportTarget{
		{Type: models.TargetTypeDomain, Value: ""},
	})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if result.Errors != 1 {
		t.Errorf("errors = %d, want 1", result.Errors)
	}
}

// --- Delete ---

func TestTargetService_Delete(t *testing.T) {
	svc, q, pid := setupTargetService(t)
	ctx := context.Background()

	now := time.Now().UTC()
	q.CreateTarget(&models.Target{
		ID: "t1", ProjectID: pid, Type: models.TargetTypeDomain,
		Value: "del.com", Source: "manual", Status: "active", CreatedAt: now,
	})

	if err := svc.Delete(ctx, "t1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Verify deleted
	list, _ := q.ListTargetsByProject(pid)
	if len(list) != 0 {
		t.Errorf("expected 0 targets after delete, got %d", len(list))
	}
}

func TestTargetService_Delete_NotFound(t *testing.T) {
	svc, _, _ := setupTargetService(t)
	ctx := context.Background()

	// SQLite DELETE is a no-op for missing rows; no error expected.
	if err := svc.Delete(ctx, "nonexistent"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- buildSuggestedScopeRules ---

func TestBuildSuggestedScopeRules(t *testing.T) {
	t.Run("domain targets", func(t *testing.T) {
		targets := []ImportTarget{
			{Type: models.TargetTypeDomain, Value: "a.com"},
			{Type: models.TargetTypeDomain, Value: "b.com"},
		}
		rules := buildSuggestedScopeRules(targets)
		if len(rules) != 2 {
			t.Errorf("len = %d, want 2", len(rules))
		}
		for _, r := range rules {
			if r.Action != "include" {
				t.Errorf("action = %q, want include", r.Action)
			}
			if r.Type != string(models.TargetTypeDomain) {
				t.Errorf("type = %q, want domain", r.Type)
			}
		}
	})

	t.Run("ip target suggests cidr", func(t *testing.T) {
		targets := []ImportTarget{
			{Type: models.TargetTypeIP, Value: "10.0.0.1"},
		}
		rules := buildSuggestedScopeRules(targets)
		if len(rules) != 1 {
			t.Fatalf("len = %d, want 1", len(rules))
		}
		if rules[0].Type != string(models.TargetTypeCIDR) {
			t.Errorf("type = %q, want cidr", rules[0].Type)
		}
		if rules[0].Value != "10.0.0.1/32" {
			t.Errorf("value = %q, want %q", rules[0].Value, "10.0.0.1/32")
		}
	})

	t.Run("company targets skipped", func(t *testing.T) {
		targets := []ImportTarget{
			{Type: models.TargetTypeCompany, Value: "Acme Corp"},
		}
		rules := buildSuggestedScopeRules(targets)
		if len(rules) != 0 {
			t.Errorf("len = %d, want 0 (company skipped)", len(rules))
		}
	})

	t.Run("dedup by type+value", func(t *testing.T) {
		targets := []ImportTarget{
			{Type: models.TargetTypeDomain, Value: "dup.com"},
			{Type: models.TargetTypeDomain, Value: "dup.com"},
		}
		rules := buildSuggestedScopeRules(targets)
		if len(rules) != 1 {
			t.Errorf("len = %d, want 1 (deduped)", len(rules))
		}
	})

	t.Run("empty list", func(t *testing.T) {
		rules := buildSuggestedScopeRules(nil)
		if len(rules) != 0 {
			t.Errorf("len = %d, want 0", len(rules))
		}
	})
}
