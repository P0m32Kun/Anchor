package db

import (
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
)

func TestBatchInsertFindings(t *testing.T) {
	rawDB := openTestDB(t)
	q := New(rawDB)
	createTestProject(t, q)
	now := time.Now().UTC()

	findings := []*models.Finding{
		{
			ID: "f-batch-1", ProjectID: "proj-1", AssetID: nil, ServiceID: nil, WebEndpointID: nil,
			SourceTool: "nuclei", SourceRuleID: "r1", DedupKey: "dk-batch-1", Title: "F1",
			Severity: models.SeverityHigh, Confidence: 80, Priority: 70, Status: models.FindingNew,
			Summary: "s", Remediation: "r", CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "f-batch-2", ProjectID: "proj-1", AssetID: nil, ServiceID: nil, WebEndpointID: nil,
			SourceTool: "nuclei", SourceRuleID: "r2", DedupKey: "dk-batch-2", Title: "F2",
			Severity: models.SeverityMedium, Confidence: 60, Priority: 50, Status: models.FindingConfirmed,
			Summary: "s", Remediation: "r", CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "f-batch-3", ProjectID: "proj-1", AssetID: nil, ServiceID: nil, WebEndpointID: nil,
			SourceTool: "nuclei", SourceRuleID: "r3", DedupKey: "dk-batch-3", Title: "F3",
			Severity: models.SeverityLow, Confidence: 40, Priority: 20, Status: models.FindingFalsePositive,
			Summary: "s", Remediation: "r", CreatedAt: now, UpdatedAt: now,
		},
	}

	if err := q.BatchInsertFindings(findings); err != nil {
		t.Fatalf("BatchInsertFindings: %v", err)
	}

	// Verify all 3 findings exist.
	for _, orig := range findings {
		got, err := q.GetFinding(orig.ID)
		if err != nil {
			t.Fatalf("GetFinding %s: %v", orig.ID, err)
		}
		if got == nil {
			t.Fatalf("GetFinding %s returned nil", orig.ID)
		}
		if got.Title != orig.Title {
			t.Errorf("%s: Title = %q, want %q", orig.ID, got.Title, orig.Title)
		}
		if got.DedupKey != orig.DedupKey {
			t.Errorf("%s: DedupKey = %q, want %q", orig.ID, got.DedupKey, orig.DedupKey)
		}
	}
}

// Regression test: BatchInsert must handle nullable FK fields correctly.
func TestBatchInsertFindings_NullFields(t *testing.T) {
	rawDB := openTestDB(t)
	q := New(rawDB)
	createTestProject(t, q)
	now := time.Now().UTC()

	findings := []*models.Finding{
		{
			ID: "f-b-null-1", ProjectID: "proj-1", AssetID: nil, ServiceID: nil, WebEndpointID: nil, RunID: nil,
			SourceTool: "nuclei", SourceRuleID: "r1", DedupKey: "dk-b-null-1", Title: "Null FKs",
			Severity: models.SeverityHigh, Confidence: 80, Priority: 70, Status: models.FindingNew,
			Summary: "s", Remediation: "r", CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "f-b-null-2", ProjectID: "proj-1", AssetID: nil, ServiceID: nil, WebEndpointID: nil, RunID: nil,
			SourceTool: "nuclei", SourceRuleID: "r2", DedupKey: "dk-b-null-2", Title: "Also null",
			Severity: models.SeverityMedium, Confidence: 60, Priority: 50, Status: models.FindingConfirmed,
			Summary: "s", Remediation: "r", CreatedAt: now, UpdatedAt: now,
		},
	}

	if err := q.BatchInsertFindings(findings); err != nil {
		t.Fatalf("BatchInsertFindings: %v", err)
	}

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
		if got.RunID != nil {
			t.Errorf("%s: RunID = %v, want nil", orig.ID, *got.RunID)
		}
	}
}

func TestBatchInsertFindings_Empty(t *testing.T) {
	rawDB := openTestDB(t)
	q := New(rawDB)
	if err := q.BatchInsertFindings(nil); err != nil {
		t.Fatalf("BatchInsertFindings(nil): %v", err)
	}
	if err := q.BatchInsertFindings([]*models.Finding{}); err != nil {
		t.Fatalf("BatchInsertFindings(empty): %v", err)
	}
}

func TestFindingBuffer_AddAndFlush(t *testing.T) {
	rawDB := openTestDB(t)
	q := New(rawDB)
	createTestProject(t, q)
	now := time.Now().UTC()

	buf := NewFindingBuffer(q, 10, 5*time.Second)

	f := &models.Finding{
		ID: "f-buf-1", ProjectID: "proj-1", AssetID: nil, ServiceID: nil, WebEndpointID: nil,
		SourceTool: "nuclei", SourceRuleID: "r1", DedupKey: "dk-buf-1", Title: "Buffered",
		Severity: models.SeverityHigh, Confidence: 80, Priority: 70, Status: models.FindingNew,
		Summary: "s", Remediation: "r", CreatedAt: now, UpdatedAt: now,
	}

	buf.Add(f)

	// Before flush, finding should not exist in DB.
	if got, _ := q.GetFinding(f.ID); got != nil {
		t.Fatal("finding should not exist before flush")
	}

	// Flush manually.
	if err := buf.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	// After flush, finding should exist.
	got, err := q.GetFinding(f.ID)
	if err != nil {
		t.Fatalf("GetFinding after flush: %v", err)
	}
	if got == nil {
		t.Fatal("finding should exist after flush")
	}
	if got.Title != f.Title {
		t.Errorf("Title = %q, want %q", got.Title, f.Title)
	}

	// Close should be a no-op after manual flush.
	if err := buf.Close(); err != nil {
		t.Fatalf("Close after flush: %v", err)
	}
}

func TestFindingBuffer_FlushOnCapacity(t *testing.T) {
	rawDB := openTestDB(t)
	q := New(rawDB)
	createTestProject(t, q)
	now := time.Now().UTC()

	buf := NewFindingBuffer(q, 3, 5*time.Second)

	for i := 0; i < 3; i++ {
		f := &models.Finding{
			ID: "f-cap-" + string(rune('a'+i)), ProjectID: "proj-1", AssetID: nil, ServiceID: nil, WebEndpointID: nil,
			SourceTool: "nuclei", SourceRuleID: "r", DedupKey: "dk-cap-" + string(rune('a'+i)), Title: "Cap",
			Severity: models.SeverityInfo, Confidence: 10, Priority: i, Status: models.FindingNew,
			Summary: "s", Remediation: "r", CreatedAt: now, UpdatedAt: now,
		}
		buf.Add(f)
	}

	// Capacity is 3, so the 3rd Add should trigger auto-flush.
	// Verify all 3 exist.
	for i := 0; i < 3; i++ {
		id := "f-cap-" + string(rune('a' + i))
		got, err := q.GetFinding(id)
		if err != nil {
			t.Fatalf("GetFinding %s: %v", id, err)
		}
		if got == nil {
			t.Fatalf("finding %s should exist after capacity flush", id)
		}
	}
}

func TestFindingBuffer_Dedup(t *testing.T) {
	rawDB := openTestDB(t)
	q := New(rawDB)
	createTestProject(t, q)
	now := time.Now().UTC()

	buf := NewFindingBuffer(q, 10, 5*time.Second)

	// Add two findings with the same dedup_key.
	f1 := &models.Finding{
		ID: "f-dedup-1", ProjectID: "proj-1", AssetID: nil, ServiceID: nil, WebEndpointID: nil,
		SourceTool: "nuclei", SourceRuleID: "r1", DedupKey: "same-dedup", Title: "First",
		Severity: models.SeverityHigh, Confidence: 80, Priority: 70, Status: models.FindingNew,
		Summary: "s", Remediation: "r", CreatedAt: now, UpdatedAt: now,
	}
	f2 := &models.Finding{
		ID: "f-dedup-2", ProjectID: "proj-1", AssetID: nil, ServiceID: nil, WebEndpointID: nil,
		SourceTool: "nuclei", SourceRuleID: "r1", DedupKey: "same-dedup", Title: "Second",
		Severity: models.SeverityHigh, Confidence: 80, Priority: 70, Status: models.FindingNew,
		Summary: "s", Remediation: "r", CreatedAt: now, UpdatedAt: now,
	}

	buf.Add(f1)
	buf.Add(f2)
	if err := buf.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	// Only the first should be inserted (buffer dedup keeps first occurrence).
	got1, _ := q.GetFinding(f1.ID)
	if got1 == nil {
		t.Fatal("first finding should exist")
	}
	got2, _ := q.GetFinding(f2.ID)
	if got2 != nil {
		t.Fatal("second finding with same dedup_key should be dropped")
	}
}

func TestFindingBuffer_CloseFlushes(t *testing.T) {
	rawDB := openTestDB(t)
	q := New(rawDB)
	createTestProject(t, q)
	now := time.Now().UTC()

	buf := NewFindingBuffer(q, 100, 5*time.Second)
	f := &models.Finding{
		ID: "f-close-1", ProjectID: "proj-1", AssetID: nil, ServiceID: nil, WebEndpointID: nil,
		SourceTool: "nuclei", SourceRuleID: "r1", DedupKey: "dk-close-1", Title: "Close",
		Severity: models.SeverityHigh, Confidence: 80, Priority: 70, Status: models.FindingNew,
		Summary: "s", Remediation: "r", CreatedAt: now, UpdatedAt: now,
	}
	buf.Add(f)

	// Close should flush remaining items.
	if err := buf.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	got, _ := q.GetFinding(f.ID)
	if got == nil {
		t.Fatal("finding should exist after Close flush")
	}

	// Subsequent Add after Close should fall back to direct insert.
	f2 := &models.Finding{
		ID: "f-close-2", ProjectID: "proj-1", AssetID: nil, ServiceID: nil, WebEndpointID: nil,
		SourceTool: "nuclei", SourceRuleID: "r2", DedupKey: "dk-close-2", Title: "AfterClose",
		Severity: models.SeverityHigh, Confidence: 80, Priority: 70, Status: models.FindingNew,
		Summary: "s", Remediation: "r", CreatedAt: now, UpdatedAt: now,
	}
	buf.Add(f2)
	got2, _ := q.GetFinding(f2.ID)
	if got2 == nil {
		t.Fatal("finding after close should still be inserted via direct path")
	}
}
