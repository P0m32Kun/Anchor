package db

import (
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
)

// Regression test for 20260427-null-scan-crash:
// Ensure CreateFinding/GetFinding round-trips correctly when
// optional nullable fields (AssetID, ServiceID, WebEndpointID) are nil.

func createTestProject(t *testing.T, q *Queries) {
	t.Helper()
	now := time.Now().UTC()
	if err := q.CreateProject(&models.Project{
		ID:             "proj-1",
		Name:           "test",
		RateLimit:      10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt:      now,
		UpdatedAt:      now,
	}); err != nil {
		t.Fatalf("create project: %v", err)
	}
}

func TestCreateFinding_AllNullOptionalFields(t *testing.T) {
	q := New(openTestDB(t))
	createTestProject(t, q)
	now := time.Now().UTC()

	f := &models.Finding{
		ID:              "f-null-1",
		ProjectID:       "proj-1",
		AssetID:         nil,
		ServiceID:       nil,
		WebEndpointID:   nil,
		SourceTool:      "nuclei",
		SourceRuleID:    "rule-1",
		DedupKey:        "dedup-null-1",
		Title:           "Test Finding with NULL optionals",
		Severity:        models.SeverityHigh,
		Confidence:      80,
		Priority:        70,
		Status:          models.FindingNew,
		Summary:         "All optional fields are NULL",
		Remediation:     "Fix it",
		RawRequest:      "",
		RawResponse:     "",
		MatchedTemplate: "",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := q.CreateFinding(f); err != nil {
		t.Fatalf("CreateFinding: %v", err)
	}

	got, err := q.GetFinding("f-null-1")
	if err != nil {
		t.Fatalf("GetFinding: %v", err)
	}
	if got == nil {
		t.Fatal("GetFinding returned nil")
	}

	if got.AssetID != nil {
		t.Errorf("AssetID = %v, want nil", *got.AssetID)
	}
	if got.ServiceID != nil {
		t.Errorf("ServiceID = %v, want nil", *got.ServiceID)
	}
	if got.WebEndpointID != nil {
		t.Errorf("WebEndpointID = %v, want nil", *got.WebEndpointID)
	}
	if got.Title != f.Title {
		t.Errorf("Title = %q, want %q", got.Title, f.Title)
	}
	if got.Severity != f.Severity {
		t.Errorf("Severity = %q, want %q", got.Severity, f.Severity)
	}
}

func TestCreateFinding_NullFieldsRoundTrip(t *testing.T) {
	q := New(openTestDB(t))
	createTestProject(t, q)
	now := time.Now().UTC()

	// Create two findings: one with all nil FK fields, one with all nil.
	// Both should round-trip without crashing (the core regression).
	findings := []*models.Finding{
		{
			ID: "f-rt-1", ProjectID: "proj-1", AssetID: nil, ServiceID: nil, WebEndpointID: nil,
			SourceTool: "nuclei", SourceRuleID: "r1", DedupKey: "dk-rt-1", Title: "All nil",
			Severity: models.SeverityHigh, Confidence: 80, Priority: 70, Status: models.FindingNew,
			Summary: "s", Remediation: "r", CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "f-rt-2", ProjectID: "proj-1", AssetID: nil, ServiceID: nil, WebEndpointID: nil,
			SourceTool: "nuclei", SourceRuleID: "r2", DedupKey: "dk-rt-2", Title: "Also nil",
			Severity: models.SeverityMedium, Confidence: 60, Priority: 50, Status: models.FindingConfirmed,
			Summary: "s", Remediation: "r", CreatedAt: now, UpdatedAt: now,
		},
	}

	for _, f := range findings {
		if err := q.CreateFinding(f); err != nil {
			t.Fatalf("CreateFinding %s: %v", f.ID, err)
		}
	}

	// Round-trip via GetFinding.
	for _, orig := range findings {
		got, err := q.GetFinding(orig.ID)
		if err != nil {
			t.Fatalf("GetFinding %s: %v", orig.ID, err)
		}
		if got == nil {
			t.Fatalf("GetFinding %s returned nil", orig.ID)
		}
		if got.AssetID != nil {
			t.Errorf("%s: AssetID = %v, want nil", orig.ID, *got.AssetID)
		}
		if got.ServiceID != nil {
			t.Errorf("%s: ServiceID = %v, want nil", orig.ID, *got.ServiceID)
		}
		if got.WebEndpointID != nil {
			t.Errorf("%s: WebEndpointID = %v, want nil", orig.ID, *got.WebEndpointID)
		}
		if got.Title != orig.Title {
			t.Errorf("%s: Title = %q, want %q", orig.ID, got.Title, orig.Title)
		}
	}
}

func TestGetFindingByDedupKey_NullFields(t *testing.T) {
	q := New(openTestDB(t))
	createTestProject(t, q)
	now := time.Now().UTC()

	f := &models.Finding{
		ID:              "f-dedup-1",
		ProjectID:       "proj-1",
		AssetID:         nil,
		ServiceID:       nil,
		WebEndpointID:   nil,
		SourceTool:      "nuclei",
		SourceRuleID:    "rule-1",
		DedupKey:        "dedup-key-1",
		Title:           "Dedup test",
		Severity:        models.SeverityLow,
		Confidence:      40,
		Priority:        20,
		Status:          models.FindingNew,
		Summary:         "Test",
		Remediation:     "Fix",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := q.CreateFinding(f); err != nil {
		t.Fatalf("CreateFinding: %v", err)
	}

	got, err := q.GetFindingByDedupKey("proj-1", "dedup-key-1")
	if err != nil {
		t.Fatalf("GetFindingByDedupKey: %v", err)
	}
	if got == nil {
		t.Fatal("GetFindingByDedupKey returned nil")
	}
	if got.ID != f.ID {
		t.Errorf("ID = %q, want %q", got.ID, f.ID)
	}
	if got.AssetID != nil {
		t.Errorf("AssetID = %v, want nil", *got.AssetID)
	}
}

func TestListFindingsByProject_NullFields(t *testing.T) {
	q := New(openTestDB(t))
	createTestProject(t, q)
	now := time.Now().UTC()

	// Create 3 findings — all with nil FK fields to avoid FK constraint.
	findings := []*models.Finding{
		{
			ID: "f-list-1", ProjectID: "proj-1", AssetID: nil, ServiceID: nil, WebEndpointID: nil,
			SourceTool: "nuclei", SourceRuleID: "r1", DedupKey: "dk1", Title: "F1",
			Severity: models.SeverityHigh, Confidence: 80, Priority: 70, Status: models.FindingNew,
			Summary: "s", Remediation: "r", CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "f-list-2", ProjectID: "proj-1", AssetID: nil, ServiceID: nil, WebEndpointID: nil,
			SourceTool: "nuclei", SourceRuleID: "r2", DedupKey: "dk2", Title: "F2",
			Severity: models.SeverityMedium, Confidence: 60, Priority: 50, Status: models.FindingConfirmed,
			Summary: "s", Remediation: "r", CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "f-list-3", ProjectID: "proj-1", AssetID: nil, ServiceID: nil, WebEndpointID: nil,
			SourceTool: "nuclei", SourceRuleID: "r3", DedupKey: "dk3", Title: "F3",
			Severity: models.SeverityLow, Confidence: 40, Priority: 20, Status: models.FindingFalsePositive,
			Summary: "s", Remediation: "r", CreatedAt: now, UpdatedAt: now,
		},
	}

	for _, f := range findings {
		if err := q.CreateFinding(f); err != nil {
			t.Fatalf("CreateFinding %s: %v", f.ID, err)
		}
	}

	list, err := q.ListFindingsByProject("proj-1")
	if err != nil {
		t.Fatalf("ListFindingsByProject: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("expected 3 findings, got %d", len(list))
	}

	// Verify null fields survive round-trip.
	for _, got := range list {
		if got.AssetID != nil {
			t.Errorf("%s: AssetID should be nil, got %v", got.ID, *got.AssetID)
		}
		if got.ServiceID != nil {
			t.Errorf("%s: ServiceID should be nil, got %v", got.ID, *got.ServiceID)
		}
		if got.WebEndpointID != nil {
			t.Errorf("%s: WebEndpointID should be nil, got %v", got.ID, *got.WebEndpointID)
		}
	}
}

func TestListFindingsByProjectPaginated_NullFields(t *testing.T) {
	q := New(openTestDB(t))
	createTestProject(t, q)
	now := time.Now().UTC()

	for i := 0; i < 5; i++ {
		f := &models.Finding{
			ID: "f-pag-" + string(rune('a'+i)), ProjectID: "proj-1",
			AssetID: nil, ServiceID: nil, WebEndpointID: nil,
			SourceTool: "nuclei", SourceRuleID: "r", DedupKey: "dk-pag-" + string(rune('a'+i)),
			Title: "F", Severity: models.SeverityInfo, Confidence: 10, Priority: i,
			Status: models.FindingNew, Summary: "s", Remediation: "r",
			CreatedAt: now, UpdatedAt: now,
		}
		if err := q.CreateFinding(f); err != nil {
			t.Fatalf("CreateFinding %s: %v", f.ID, err)
		}
	}

	list, err := q.ListFindingsByProjectPaginated("proj-1", 3, 0)
	if err != nil {
		t.Fatalf("ListFindingsByProjectPaginated: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("expected 3 findings, got %d", len(list))
	}
	for _, got := range list {
		if got.AssetID != nil || got.ServiceID != nil || got.WebEndpointID != nil {
			t.Errorf("%s: expected all nil FK fields", got.ID)
		}
	}
}

func strPtr(s string) *string { return &s }
