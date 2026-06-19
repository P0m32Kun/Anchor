package db

import (
	"database/sql"
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
)

// --- isRetryableDBError ---

func TestIsRetryableDBError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"locked", sql.ErrNoRows, false},
		{"database is locked", &mockErr{"database is locked"}, true},
		{"busy", &mockErr{"busy processing"}, true},
		{"other", &mockErr{"constraint failed"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRetryableDBError(tt.err); got != tt.want {
				t.Errorf("isRetryableDBError() = %v, want %v", got, tt.want)
			}
		})
	}
}

type mockErr struct{ msg string }

func (e *mockErr) Error() string { return e.msg }

// --- BatchInsertFindings with tx path ---

func TestBatchInsertFindings_TxPath(t *testing.T) {
	q, projID, runID := setupFindingTestData(t)
	now := time.Now().UTC().Truncate(time.Second)

	findings := []*models.Finding{
		{
			ID: "f-batch-1", ProjectID: projID, RunID: &runID,
			SourceTool: "nuclei", SourceRuleID: "CVE-2024-0001", DedupKey: "f-batch-1",
			Title: "Batch 1", Severity: models.SeverityHigh, Confidence: 80, Priority: 100,
			Status: models.FindingNew, Summary: "test", CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "f-batch-2", ProjectID: projID, RunID: &runID,
			SourceTool: "nuclei", SourceRuleID: "CVE-2024-0002", DedupKey: "f-batch-2",
			Title: "Batch 2", Severity: models.SeverityMedium, Confidence: 70, Priority: 80,
			Status: models.FindingNew, Summary: "test", CreatedAt: now, UpdatedAt: now,
		},
	}

	if err := q.BatchInsertFindings(findings); err != nil {
		t.Fatalf("BatchInsertFindings: %v", err)
	}

	// Verify both were inserted
	got1, _ := q.GetFinding("f-batch-1")
	if got1 == nil {
		t.Fatal("f-batch-1 not found")
	}
	got2, _ := q.GetFinding("f-batch-2")
	if got2 == nil {
		t.Fatal("f-batch-2 not found")
	}
}

func TestBatchInsertFindings_EmptySlice(t *testing.T) {
	q := New(openTestDB(t))

	if err := q.BatchInsertFindings(nil); err != nil {
		t.Fatalf("BatchInsertFindings(nil): %v", err)
	}
}

func TestBatchInsertFindings_TxFallback(t *testing.T) {
	rawDB := openTestDB(t)
	now := time.Now().UTC().Truncate(time.Second)

	// Create project and run in this DB
	q := New(rawDB)
	if err := q.CreateProject(&models.Project{
		ID: "proj-tx", Name: "tx-test", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create project: %v", err)
	}
	if err := q.CreateRun(&models.Run{
		ID: "run-tx", ProjectID: "proj-tx", Name: "tx-run",
		Status: models.RunRunning, CreatedAt: now,
	}); err != nil {
		t.Fatalf("create run: %v", err)
	}

	// Use WithTx to create a Queries backed by *sql.Tx (not *sql.DB)
	err := WithTx(rawDB, func(tq *Queries) error {
		f := &models.Finding{
			ID: "f-tx-1", ProjectID: "proj-tx", RunID: strPtr("run-tx"),
			SourceTool: "nuclei", SourceRuleID: "CVE-2024-0001", DedupKey: "f-tx-1",
			Title: "TX Path", Severity: models.SeverityLow, Confidence: 50, Priority: 30,
			Status: models.FindingNew, Summary: "test", CreatedAt: now, UpdatedAt: now,
		}
		return tq.batchInsertOnce([]*models.Finding{f})
	})
	if err != nil {
		t.Fatalf("batchInsertOnce via tx: %v", err)
	}
}

// --- GetDictionaryHitStats with artifact ---

func TestGetDictionaryHitStats_WithArtifact(t *testing.T) {
	q, projID, runID := setupFindingTestData(t)
	now := time.Now().UTC().Truncate(time.Second)

	// Create an ffuf scan task
	q.CreateScanTask(&models.ScanTask{
		ID: "task-ffuf-2", ProjectID: projID, PlanID: "plan-1",
		RunID: &runID, Tool: "ffuf", CommandTemplate: "ffuf -w wordlist",
		Status: models.TaskCompleted, CreatedAt: now,
	})

	// Create a stdout artifact for this task
	taskPtr := "task-ffuf-2"
	q.CreateRawArtifact(&models.RawArtifact{
		ID: "art-ffuf", ProjectID: projID, TaskID: &taskPtr,
		Type: "stdout", Path: "/tmp/ffuf-output.txt",
		CreatedAt: now,
	})

	stats, err := q.GetDictionaryHitStats(runID)
	if err != nil {
		t.Fatalf("GetDictionaryHitStats: %v", err)
	}
	if len(stats) != 1 {
		t.Fatalf("expected 1 stat, got %d", len(stats))
	}
	if stats[0].TaskID != "task-ffuf-2" {
		t.Errorf("task_id = %q, want task-ffuf-2", stats[0].TaskID)
	}
}
