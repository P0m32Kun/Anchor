package db

import (
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// --- Target helpers ---

func setupTargetTest(t *testing.T) (*Queries, string) {
	t.Helper()
	q := New(openTestDB(t))
	now := time.Now().UTC()
	if err := q.CreateProject(&models.Project{
		ID: "proj-tgt", Name: "target-test", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create project: %v", err)
	}
	return q, "proj-tgt"
}

func seedTarget(t *testing.T, q *Queries, id, projectID, value string) {
	t.Helper()
	now := time.Now().UTC()
	if err := q.CreateTarget(&models.Target{
		ID: id, ProjectID: projectID, Type: models.TargetTypeDomain,
		Value: value, Source: "manual", Status: "active", CreatedAt: now,
	}); err != nil {
		t.Fatalf("seed target %s: %v", id, err)
	}
}

// --- 1. BulkCreateTargets ---

func TestBulkCreateTargets(t *testing.T) {
	q, projID := setupTargetTest(t)
	now := time.Now().UTC()

	targets := []*models.Target{
		{ID: "tgt-1", ProjectID: projID, Type: models.TargetTypeDomain, Value: "a.example.com", Source: "manual", Status: "active", CreatedAt: now},
		{ID: "tgt-2", ProjectID: projID, Type: models.TargetTypeDomain, Value: "b.example.com", Source: "manual", Status: "active", CreatedAt: now},
		{ID: "tgt-3", ProjectID: projID, Type: models.TargetTypeIP, Value: "1.2.3.4", Source: "import", Status: "active", CreatedAt: now},
	}

	if err := q.BulkCreateTargets(targets); err != nil {
		t.Fatalf("BulkCreateTargets: %v", err)
	}

	list, err := q.ListTargetsByProject(projID)
	if err != nil {
		t.Fatalf("ListTargetsByProject: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("expected 3 targets, got %d", len(list))
	}
}

func TestBulkCreateTargets_Empty(t *testing.T) {
	q, _ := setupTargetTest(t)

	if err := q.BulkCreateTargets(nil); err != nil {
		t.Fatalf("BulkCreateTargets(nil): %v", err)
	}
}

// --- 2. TargetExistsByValue ---

func TestTargetExistsByValue(t *testing.T) {
	q, projID := setupTargetTest(t)
	seedTarget(t, q, "tgt-1", projID, "exists.example.com")

	exists, err := q.TargetExistsByValue(projID, "exists.example.com")
	if err != nil {
		t.Fatalf("TargetExistsByValue: %v", err)
	}
	if !exists {
		t.Error("expected target to exist")
	}

	exists, err = q.TargetExistsByValue(projID, "nope.example.com")
	if err != nil {
		t.Fatalf("TargetExistsByValue nope: %v", err)
	}
	if exists {
		t.Error("expected target not to exist")
	}
}

// --- 3. ListTargetsByProject ---

func TestListTargetsByProject(t *testing.T) {
	q, projID := setupTargetTest(t)
	seedTarget(t, q, "tgt-1", projID, "a.example.com")
	seedTarget(t, q, "tgt-2", projID, "b.example.com")

	list, err := q.ListTargetsByProject(projID)
	if err != nil {
		t.Fatalf("ListTargetsByProject: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 targets, got %d", len(list))
	}

	// Empty project
	list2, err := q.ListTargetsByProject("nonexistent")
	if err != nil {
		t.Fatalf("ListTargetsByProject nonexistent: %v", err)
	}
	if len(list2) != 0 {
		t.Errorf("expected 0, got %d", len(list2))
	}
}

// --- 4. CountTargetsByProject ---

func TestCountTargetsByProject(t *testing.T) {
	q, projID := setupTargetTest(t)

	count, err := q.CountTargetsByProject(projID)
	if err != nil {
		t.Fatalf("CountTargetsByProject empty: %v", err)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}

	seedTarget(t, q, "tgt-1", projID, "a.example.com")
	seedTarget(t, q, "tgt-2", projID, "b.example.com")

	count, err = q.CountTargetsByProject(projID)
	if err != nil {
		t.Fatalf("CountTargetsByProject: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}

// --- 5. ListTargetsByProjectPaginated ---

func TestListTargetsByProjectPaginated(t *testing.T) {
	q, projID := setupTargetTest(t)
	for i := 0; i < 5; i++ {
		seedTarget(t, q, util.GenerateID(), projID, string(rune('a'+i))+".example.com")
	}

	page1, err := q.ListTargetsByProjectPaginated(projID, 2, 0)
	if err != nil {
		t.Fatalf("page1: %v", err)
	}
	if len(page1) != 2 {
		t.Fatalf("page1: expected 2, got %d", len(page1))
	}

	page2, err := q.ListTargetsByProjectPaginated(projID, 2, 2)
	if err != nil {
		t.Fatalf("page2: %v", err)
	}
	if len(page2) != 2 {
		t.Fatalf("page2: expected 2, got %d", len(page2))
	}

	page3, err := q.ListTargetsByProjectPaginated(projID, 2, 4)
	if err != nil {
		t.Fatalf("page3: %v", err)
	}
	if len(page3) != 1 {
		t.Fatalf("page3: expected 1, got %d", len(page3))
	}
}

// --- 6. DeleteTarget ---

func TestDeleteTarget(t *testing.T) {
	q, projID := setupTargetTest(t)
	seedTarget(t, q, "tgt-del", projID, "delete-me.example.com")

	if err := q.DeleteTarget("tgt-del"); err != nil {
		t.Fatalf("DeleteTarget: %v", err)
	}

	got, err := q.GetTarget("tgt-del")
	if err != nil {
		t.Fatalf("GetTarget: %v", err)
	}
	if got != nil {
		t.Error("expected nil after delete")
	}
}

// --- 7. CreateIPDiscoveryResult / List ---

func TestIPDiscoveryResult_CRUD(t *testing.T) {
	q, projID := setupTargetTest(t)
	seedTarget(t, q, "tgt-1", projID, "example.com")
	now := time.Now().UTC()

	hostname := "host1.example.com"
	results := []*models.IPDiscoveryResult{
		{ID: "ip-1", ProjectID: projID, TargetID: "tgt-1", IP: "1.2.3.4", Hostname: &hostname, Source: "naabu", Alive: true, CreatedAt: now},
		{ID: "ip-2", ProjectID: projID, TargetID: "tgt-1", IP: "5.6.7.8", Source: "nmap", Alive: false, CreatedAt: now},
	}
	for _, r := range results {
		if err := q.CreateIPDiscoveryResult(r); err != nil {
			t.Fatalf("CreateIPDiscoveryResult %s: %v", r.ID, err)
		}
	}

	// List by project
	byProj, err := q.ListIPDiscoveryResultsByProject(projID)
	if err != nil {
		t.Fatalf("ListIPDiscoveryResultsByProject: %v", err)
	}
	if len(byProj) != 2 {
		t.Fatalf("expected 2, got %d", len(byProj))
	}

	// List by target
	byTgt, err := q.ListIPDiscoveryResultsByTarget("tgt-1")
	if err != nil {
		t.Fatalf("ListIPDiscoveryResultsByTarget: %v", err)
	}
	if len(byTgt) != 2 {
		t.Fatalf("expected 2, got %d", len(byTgt))
	}

	// Verify fields
	for _, r := range byProj {
		if r.ID == "ip-1" {
			if r.Hostname == nil || *r.Hostname != hostname {
				t.Errorf("hostname = %v, want %q", r.Hostname, hostname)
			}
			if !r.Alive {
				t.Error("ip-1 should be alive")
			}
		}
	}

	// Empty target
	empty, err := q.ListIPDiscoveryResultsByTarget("nonexistent")
	if err != nil {
		t.Fatalf("ListIPDiscoveryResultsByTarget empty: %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("expected 0, got %d", len(empty))
	}
}

// --- 8. ScopeRule CRUD ---

func TestScopeRule_CRUD(t *testing.T) {
	q, projID := setupTargetTest(t)
	now := time.Now().UTC()

	rules := []*models.ScopeRule{
		{ID: "sr-1", ProjectID: projID, Action: models.ScopeActionInclude, Type: models.TargetTypeDomain, Value: "*.example.com", Reason: "in scope", CreatedAt: now, UpdatedAt: now},
		{ID: "sr-2", ProjectID: projID, Action: models.ScopeActionExclude, Type: models.TargetTypeDomain, Value: "*.internal.example.com", Reason: "out of scope", CreatedAt: now, UpdatedAt: now},
	}
	for _, r := range rules {
		if err := q.CreateScopeRule(r); err != nil {
			t.Fatalf("CreateScopeRule %s: %v", r.ID, err)
		}
	}

	// List
	list, err := q.ListScopeRulesByProject(projID)
	if err != nil {
		t.Fatalf("ListScopeRulesByProject: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2, got %d", len(list))
	}

	// Count
	count, err := q.CountScopeRulesByProject(projID)
	if err != nil {
		t.Fatalf("CountScopeRulesByProject: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}

	// Paginated
	page, err := q.ListScopeRulesByProjectPaginated(projID, 1, 0)
	if err != nil {
		t.Fatalf("ListScopeRulesByProjectPaginated: %v", err)
	}
	if len(page) != 1 {
		t.Fatalf("paginated: expected 1, got %d", len(page))
	}

	// GetMaxScopeRuleUpdatedAt
	maxTime, err := q.GetMaxScopeRuleUpdatedAt(projID)
	if err != nil {
		t.Fatalf("GetMaxScopeRuleUpdatedAt: %v", err)
	}
	if maxTime.IsZero() {
		t.Error("expected non-zero max updated_at")
	}

	// Delete
	if err := q.DeleteScopeRule("sr-1"); err != nil {
		t.Fatalf("DeleteScopeRule: %v", err)
	}
	count2, _ := q.CountScopeRulesByProject(projID)
	if count2 != 1 {
		t.Errorf("count after delete = %d, want 1", count2)
	}

	// GetMaxScopeRuleUpdatedAt empty project
	emptyTime, err := q.GetMaxScopeRuleUpdatedAt("nonexistent")
	if err != nil {
		t.Fatalf("GetMaxScopeRuleUpdatedAt empty: %v", err)
	}
	if !emptyTime.IsZero() {
		t.Error("expected zero time for empty project")
	}
}

// --- 9. ScopeDecision CRUD ---

func TestScopeDecision_CRUD(t *testing.T) {
	q, projID := setupTargetTest(t)
	now := time.Now().UTC()

	// Create scope rule and scan_task for FK references
	ruleID := "sr-1"
	q.CreateScopeRule(&models.ScopeRule{
		ID: ruleID, ProjectID: projID, Action: models.ScopeActionInclude,
		Type: models.TargetTypeDomain, Value: "*.example.com",
		Reason: "in scope", CreatedAt: now, UpdatedAt: now,
	})
	q.CreateScanTask(&models.ScanTask{
		ID: "task-sd-1", ProjectID: projID, Tool: "scope_check",
		Status: models.TaskCreated, CreatedAt: now,
	})

	taskID := "task-sd-1"
	d := &models.ScopeDecision{
		ID: "sd-1", ProjectID: projID, TargetValue: "api.example.com",
		TaskID: &taskID, Decision: models.ScopeAllow, MatchedRuleID: &ruleID,
		Reason: "matches include rule", CreatedAt: now,
	}
	if err := q.CreateScopeDecision(d); err != nil {
		t.Fatalf("CreateScopeDecision: %v", err)
	}

	// Get latest
	got, err := q.GetLatestScopeDecision(projID, "api.example.com")
	if err != nil {
		t.Fatalf("GetLatestScopeDecision: %v", err)
	}
	if got == nil {
		t.Fatal("expected decision, got nil")
	}
	if got.Decision != models.ScopeAllow {
		t.Errorf("decision = %q, want %q", got.Decision, models.ScopeAllow)
	}
	if got.TaskID == nil || *got.TaskID != taskID {
		t.Errorf("task_id = %v, want %q", got.TaskID, taskID)
	}

	// Not found
	got2, err := q.GetLatestScopeDecision(projID, "nonexistent.example.com")
	if err != nil {
		t.Fatalf("GetLatestScopeDecision not found: %v", err)
	}
	if got2 != nil {
		t.Error("expected nil for nonexistent target")
	}

	// List by plan (project)
	list, err := q.ListScopeDecisionsByPlan(projID)
	if err != nil {
		t.Fatalf("ListScopeDecisionsByPlan: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1, got %d", len(list))
	}
}
