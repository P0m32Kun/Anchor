package scope

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/models"
	_ "github.com/mattn/go-sqlite3"
)

// --- helpers ---

func openTestDB(t *testing.T) (*sql.DB, error) {
	t.Helper()
	rawDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, err
	}
	rawDB.SetMaxOpenConns(1)
	t.Cleanup(func() { rawDB.Close() })
	if err := db.Migrate(rawDB); err != nil {
		return nil, err
	}
	return rawDB, nil
}

func setupEngine(t *testing.T) (*Engine, *db.Queries, string) {
	t.Helper()
	rawDB, err := openTestDB(t)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	q := db.New(rawDB)
	eng := NewEngine(q)

	projID := "proj-test"
	now := time.Now().UTC()
	if err := q.CreateProject(&models.Project{
		ID:             projID,
		Name:           "test",
		RateLimit:      10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt:      now,
		UpdatedAt:      now,
	}); err != nil {
		t.Fatalf("create project: %v", err)
	}
	return eng, q, projID
}

func addRule(t *testing.T, q *db.Queries, projectID string, action models.ScopeAction, ruleType models.TargetType, value string) string {
	t.Helper()
	id := "rule-" + value + "-" + string(action)
	now := time.Now().UTC()
	if err := q.CreateScopeRule(&models.ScopeRule{
		ID:        id,
		ProjectID: projectID,
		Action:    action,
		Type:      ruleType,
		Value:     value,
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create scope rule: %v", err)
	}
	return id
}

func assertAllow(t *testing.T, d *models.ScopeDecision, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.Decision != models.ScopeAllow {
		t.Fatalf("expected allow, got %s (reason: %s)", d.Decision, d.Reason)
	}
}

func assertDeny(t *testing.T, d *models.ScopeDecision, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.Decision != models.ScopeDeny {
		t.Fatalf("expected deny, got %s (reason: %s)", d.Decision, d.Reason)
	}
}

// --- Check: domain ---

func TestEngine_Check_Domain_Allow(t *testing.T) {
	eng, q, pid := setupEngine(t)
	addRule(t, q, pid, models.ScopeActionInclude, models.TargetTypeDomain, "example.com")

	target := &models.Target{Type: models.TargetTypeDomain, Value: "example.com"}
	d, err := eng.Check(context.Background(), pid, target)
	assertAllow(t, d, err)
}

func TestEngine_Check_Domain_Subdomain_Allow(t *testing.T) {
	eng, q, pid := setupEngine(t)
	addRule(t, q, pid, models.ScopeActionInclude, models.TargetTypeDomain, "example.com")

	target := &models.Target{Type: models.TargetTypeDomain, Value: "sub.example.com"}
	d, err := eng.Check(context.Background(), pid, target)
	assertAllow(t, d, err)
}

func TestEngine_Check_Domain_ExcludeOverInclude(t *testing.T) {
	eng, q, pid := setupEngine(t)
	addRule(t, q, pid, models.ScopeActionInclude, models.TargetTypeDomain, "example.com")
	addRule(t, q, pid, models.ScopeActionExclude, models.TargetTypeDomain, "admin.example.com")

	target := &models.Target{Type: models.TargetTypeDomain, Value: "admin.example.com"}
	d, err := eng.Check(context.Background(), pid, target)
	assertDeny(t, d, err)
	if d.MatchedRuleID == nil || *d.MatchedRuleID != "rule-admin.example.com-exclude" {
		t.Fatalf("expected exclude rule matched, got %v", d.MatchedRuleID)
	}
}

func TestEngine_Check_Domain_NoRules_Deny(t *testing.T) {
	eng, _, pid := setupEngine(t)

	target := &models.Target{Type: models.TargetTypeDomain, Value: "unknown.com"}
	d, err := eng.Check(context.Background(), pid, target)
	assertDeny(t, d, err)
}

// --- Check: IP ---

func TestEngine_Check_IP_Allow(t *testing.T) {
	eng, q, pid := setupEngine(t)
	addRule(t, q, pid, models.ScopeActionInclude, models.TargetTypeIP, "10.0.0.1")

	target := &models.Target{Type: models.TargetTypeIP, Value: "10.0.0.1"}
	d, err := eng.Check(context.Background(), pid, target)
	assertAllow(t, d, err)
}

func TestEngine_Check_IP_Deny(t *testing.T) {
	eng, q, pid := setupEngine(t)
	addRule(t, q, pid, models.ScopeActionInclude, models.TargetTypeIP, "10.0.0.1")

	target := &models.Target{Type: models.TargetTypeIP, Value: "10.0.0.2"}
	d, err := eng.Check(context.Background(), pid, target)
	assertDeny(t, d, err)
}

// --- Check: CIDR ---

func TestEngine_Check_CIDR_Allow(t *testing.T) {
	eng, q, pid := setupEngine(t)
	addRule(t, q, pid, models.ScopeActionInclude, models.TargetTypeCIDR, "10.0.0.0/24")

	target := &models.Target{Type: models.TargetTypeIP, Value: "10.0.0.42"}
	d, err := eng.Check(context.Background(), pid, target)
	assertAllow(t, d, err)
}

func TestEngine_Check_CIDR_Deny_OutOfRange(t *testing.T) {
	eng, q, pid := setupEngine(t)
	addRule(t, q, pid, models.ScopeActionInclude, models.TargetTypeCIDR, "10.0.0.0/24")

	target := &models.Target{Type: models.TargetTypeIP, Value: "10.0.1.1"}
	d, err := eng.Check(context.Background(), pid, target)
	assertDeny(t, d, err)
}

// --- Check: URL ---

func TestEngine_Check_URL_DomainRule_Allow(t *testing.T) {
	eng, q, pid := setupEngine(t)
	addRule(t, q, pid, models.ScopeActionInclude, models.TargetTypeDomain, "example.com")

	target := &models.Target{Type: models.TargetTypeURL, Value: "https://example.com/path"}
	d, err := eng.Check(context.Background(), pid, target)
	assertAllow(t, d, err)
}

func TestEngine_Check_URL_PrefixRule_Allow(t *testing.T) {
	eng, q, pid := setupEngine(t)
	addRule(t, q, pid, models.ScopeActionInclude, models.TargetTypeURL, "https://example.com/api")

	target := &models.Target{Type: models.TargetTypeURL, Value: "https://example.com/api/v1"}
	d, err := eng.Check(context.Background(), pid, target)
	assertAllow(t, d, err)
}

func TestEngine_Check_URL_IPRule_Allow(t *testing.T) {
	eng, q, pid := setupEngine(t)
	addRule(t, q, pid, models.ScopeActionInclude, models.TargetTypeIP, "10.0.0.1")

	target := &models.Target{Type: models.TargetTypeURL, Value: "http://10.0.0.1:8080/api"}
	d, err := eng.Check(context.Background(), pid, target)
	assertAllow(t, d, err)
}

func TestEngine_Check_URL_CIDRRule_Allow(t *testing.T) {
	eng, q, pid := setupEngine(t)
	addRule(t, q, pid, models.ScopeActionInclude, models.TargetTypeCIDR, "192.168.1.0/24")

	target := &models.Target{Type: models.TargetTypeURL, Value: "http://192.168.1.50/secret"}
	d, err := eng.Check(context.Background(), pid, target)
	assertAllow(t, d, err)
}

// --- Check: rate limit ---

func TestEngine_Check_NegativeRateLimit_Deny(t *testing.T) {
	rawDB, err := openTestDB(t)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	q := db.New(rawDB)
	eng := NewEngine(q)

	projID := "proj-rl"
	now := time.Now().UTC()
	if err := q.CreateProject(&models.Project{
		ID:             projID,
		Name:           "test-rl",
		RateLimit:      -1,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt:      now,
		UpdatedAt:      now,
	}); err != nil {
		t.Fatalf("create project: %v", err)
	}

	target := &models.Target{Type: models.TargetTypeDomain, Value: "example.com"}
	d, err := eng.Check(context.Background(), projID, target)
	assertDeny(t, d, err)
	if d.Reason == "" {
		t.Fatal("expected reason about invalid rate limit")
	}
}

// --- CheckIP ---

func TestEngine_CheckIP_Allow(t *testing.T) {
	eng, q, pid := setupEngine(t)
	addRule(t, q, pid, models.ScopeActionInclude, models.TargetTypeIP, "10.0.0.1")

	d, err := eng.CheckIP(context.Background(), pid, "10.0.0.1")
	assertAllow(t, d, err)
}

func TestEngine_CheckIP_Deny(t *testing.T) {
	eng, _, pid := setupEngine(t)

	d, err := eng.CheckIP(context.Background(), pid, "10.0.0.1")
	assertDeny(t, d, err)
}

// --- ValidateBeforeRun ---

func TestEngine_ValidateBeforeRun_FreshCheck(t *testing.T) {
	eng, q, pid := setupEngine(t)
	addRule(t, q, pid, models.ScopeActionInclude, models.TargetTypeDomain, "example.com")

	target := &models.Target{Type: models.TargetTypeDomain, Value: "example.com"}
	d, err := eng.ValidateBeforeRun(context.Background(), pid, target, "task-1")
	assertAllow(t, d, err)
}

func TestEngine_ValidateBeforeRun_ReuseDecision(t *testing.T) {
	eng, q, pid := setupEngine(t)
	addRule(t, q, pid, models.ScopeActionInclude, models.TargetTypeDomain, "example.com")

	ctx := context.Background()
	target := &models.Target{Type: models.TargetTypeDomain, Value: "example.com"}

	// First check creates a decision.
	d1, err := eng.Check(ctx, pid, target)
	assertAllow(t, d1, err)

	// ValidateBeforeRun should reuse the cached decision.
	d2, err := eng.ValidateBeforeRun(ctx, pid, target, "task-2")
	assertAllow(t, d2, err)
	if d2.TaskID == nil || *d2.TaskID != "task-2" {
		t.Fatalf("expected task-2 attached, got %v", d2.TaskID)
	}
}

func TestEngine_ValidateBeforeRun_RulesChanged_ReCheck(t *testing.T) {
	eng, q, pid := setupEngine(t)
	addRule(t, q, pid, models.ScopeActionInclude, models.TargetTypeDomain, "example.com")

	ctx := context.Background()
	target := &models.Target{Type: models.TargetTypeDomain, Value: "example.com"}

	// First check creates a decision.
	d1, err := eng.Check(ctx, pid, target)
	assertAllow(t, d1, err)

	// Add exclude rule AFTER the decision (newer UpdatedAt).
	time.Sleep(10 * time.Millisecond)
	addRule(t, q, pid, models.ScopeActionExclude, models.TargetTypeDomain, "example.com")

	// ValidateBeforeRun should detect rule change and re-check → deny.
	d2, err := eng.ValidateBeforeRun(ctx, pid, target, "task-2")
	assertDeny(t, d2, err)
}

func TestEngine_ValidateBeforeRun_NegativeRateLimit_ForceReCheck(t *testing.T) {
	rawDB, err := openTestDB(t)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	q := db.New(rawDB)
	eng := NewEngine(q)

	projID := "proj-vbr"
	now := time.Now().UTC()
	if err := q.CreateProject(&models.Project{
		ID:             projID,
		Name:           "test-vbr",
		RateLimit:      -1,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt:      now,
		UpdatedAt:      now,
	}); err != nil {
		t.Fatalf("create project: %v", err)
	}

	target := &models.Target{Type: models.TargetTypeDomain, Value: "example.com"}
	d, err := eng.ValidateBeforeRun(context.Background(), projID, target, "task-1")
	assertDeny(t, d, err)
}

func TestEngine_ValidateBeforeRun_NoCachedDecision_FreshCheck(t *testing.T) {
	eng, q, pid := setupEngine(t)
	addRule(t, q, pid, models.ScopeActionInclude, models.TargetTypeDomain, "example.com")

	target := &models.Target{Type: models.TargetTypeDomain, Value: "example.com"}
	d, err := eng.ValidateBeforeRun(context.Background(), pid, target, "task-1")
	assertAllow(t, d, err)
}

// --- matchURL edge cases ---

func TestEngine_Check_URL_WrongDomain_Deny(t *testing.T) {
	eng, q, pid := setupEngine(t)
	addRule(t, q, pid, models.ScopeActionInclude, models.TargetTypeDomain, "example.com")

	target := &models.Target{Type: models.TargetTypeURL, Value: "https://evil.com/phish"}
	d, err := eng.Check(context.Background(), pid, target)
	assertDeny(t, d, err)
}

func TestEngine_Check_URL_WrongIP_Deny(t *testing.T) {
	eng, q, pid := setupEngine(t)
	addRule(t, q, pid, models.ScopeActionInclude, models.TargetTypeIP, "10.0.0.1")

	target := &models.Target{Type: models.TargetTypeURL, Value: "http://10.0.0.2:8080/api"}
	d, err := eng.Check(context.Background(), pid, target)
	assertDeny(t, d, err)
}
