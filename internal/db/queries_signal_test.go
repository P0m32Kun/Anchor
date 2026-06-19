package db

import (
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

func TestSignal_CRUD(t *testing.T) {
	q := New(openTestDB(t))
	createTestProject(t, q)
	now := time.Now().UTC()

	s := &models.Signal{
		ID: util.GenerateID(), ProjectID: "proj-1",
		SourceKind: models.SignalSourceKindFinding, SourceID: "f-1",
		Title: "SQL Injection", Severity: models.SignalSeverityHigh,
		Score: 85, ScopeStatus: "in_scope", Status: models.SignalStatusNew,
		Metadata: `{"rule":"sqli-01"}`,
		FirstSeen: now, LastSeen: now, CreatedAt: now, UpdatedAt: now,
	}
	if err := q.CreateSignal(s); err != nil {
		t.Fatalf("CreateSignal: %v", err)
	}

	// ListByProject
	list, err := q.ListSignalsByProject("proj-1")
	if err != nil {
		t.Fatalf("ListSignalsByProject: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("list len = %d, want 1", len(list))
	}
	if list[0].Title != "SQL Injection" {
		t.Errorf("title = %q, want SQL Injection", list[0].Title)
	}

	// UpdateStatus
	if err := q.UpdateSignalStatus(s.ID, models.SignalStatusAcknowledged); err != nil {
		t.Fatalf("UpdateSignalStatus: %v", err)
	}
	list2, _ := q.ListSignalsByProject("proj-1")
	if list2[0].Status != models.SignalStatusAcknowledged {
		t.Errorf("status = %q, want %q", list2[0].Status, models.SignalStatusAcknowledged)
	}
}

func TestCreateSignal_AutoFields(t *testing.T) {
	q := New(openTestDB(t))
	createTestProject(t, q)

	s := &models.Signal{
		ProjectID: "proj-1", SourceKind: models.SignalSourceKindAssetNew,
		SourceID: "asset-1", Title: "New asset", Severity: models.SignalSeverityInfo,
		Score: 10, Status: models.SignalStatusNew,
	}
	if err := q.CreateSignal(s); err != nil {
		t.Fatalf("CreateSignal: %v", err)
	}
	if s.ID == "" {
		t.Error("expected auto-generated ID")
	}
	if s.CreatedAt.IsZero() {
		t.Error("expected auto-set CreatedAt")
	}
	if s.FirstSeen.IsZero() {
		t.Error("expected auto-set FirstSeen")
	}
}

func TestCountSignalsByProject(t *testing.T) {
	q := New(openTestDB(t))
	createTestProject(t, q)
	now := time.Now().UTC()

	signals := []*models.Signal{
		{ID: util.GenerateID(), ProjectID: "proj-1", SourceKind: "finding", SourceID: "f-1", Title: "A", Severity: "high", Score: 80, Status: models.SignalStatusNew, FirstSeen: now, LastSeen: now, CreatedAt: now, UpdatedAt: now},
		{ID: util.GenerateID(), ProjectID: "proj-1", SourceKind: "finding", SourceID: "f-2", Title: "B", Severity: "low", Score: 20, Status: models.SignalStatusResolved, FirstSeen: now, LastSeen: now, CreatedAt: now, UpdatedAt: now},
	}
	for _, s := range signals {
		if err := q.CreateSignal(s); err != nil {
			t.Fatalf("CreateSignal: %v", err)
		}
	}

	// All
	count, err := q.CountSignalsByProject("proj-1", nil)
	if err != nil {
		t.Fatalf("CountSignalsByProject all: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}

	// By status
	newStatus := models.SignalStatusNew
	count2, err := q.CountSignalsByProject("proj-1", &newStatus)
	if err != nil {
		t.Fatalf("CountSignalsByProject new: %v", err)
	}
	if count2 != 1 {
		t.Errorf("count = %d, want 1", count2)
	}
}

func TestGetSignalsBySource(t *testing.T) {
	q := New(openTestDB(t))
	createTestProject(t, q)
	now := time.Now().UTC()

	if err := q.CreateSignal(&models.Signal{
		ID: util.GenerateID(), ProjectID: "proj-1",
		SourceKind: "finding", SourceID: "src-1",
		Title: "Test", Severity: "medium", Score: 50, Status: models.SignalStatusNew,
		FirstSeen: now, LastSeen: now, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("CreateSignal: %v", err)
	}

	list, err := q.GetSignalsBySource("finding", "src-1")
	if err != nil {
		t.Fatalf("GetSignalsBySource: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("list len = %d, want 1", len(list))
	}

	// No match
	list2, err := q.GetSignalsBySource("finding", "nonexistent")
	if err != nil {
		t.Fatalf("GetSignalsBySource nonexistent: %v", err)
	}
	if len(list2) != 0 {
		t.Errorf("list len = %d, want 0", len(list2))
	}
}

func TestUpdateSignalLastSeen(t *testing.T) {
	q := New(openTestDB(t))
	createTestProject(t, q)
	now := time.Now().UTC()

	s := &models.Signal{
		ID: util.GenerateID(), ProjectID: "proj-1",
		SourceKind: "finding", SourceID: "f-1",
		Title: "Test", Severity: "low", Score: 30, Status: models.SignalStatusNew,
		FirstSeen: now, LastSeen: now, CreatedAt: now, UpdatedAt: now,
	}
	if err := q.CreateSignal(s); err != nil {
		t.Fatalf("CreateSignal: %v", err)
	}

	if err := q.UpdateSignalLastSeen(s.ID, 90); err != nil {
		t.Fatalf("UpdateSignalLastSeen: %v", err)
	}

	list, _ := q.ListSignalsByProject("proj-1")
	if list[0].Score != 90 {
		t.Errorf("score = %d, want 90", list[0].Score)
	}
}
