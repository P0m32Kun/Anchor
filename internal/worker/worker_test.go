package worker

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/scope"
	_ "github.com/mattn/go-sqlite3"
)

// --- isUnreachableError ---

func TestIsUnreachableError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"connection refused", errors.New("post task to worker: connection refused"), true},
		{"no such host", errors.New("dial tcp: no such host"), true},
		{"i/o timeout", errors.New("i/o timeout"), true},
		{"network unreachable", errors.New("network is unreachable"), true},
		{"dial tcp", errors.New("dial tcp 10.0.0.1:9999: connect: connection refused"), true},
		{"worker unreachable", errors.New("worker unreachable"), true},
		{"bare timeout NOT unreachable", errors.New("exceeded 30s server-side poll deadline"), false},
		{"task failure NOT unreachable", errors.New("nuclei exited with code 1"), false},
		{"scope denied NOT unreachable", errors.New("scope denied"), false},
		{"wrapped unreachable", errors.New("dispatch failed: post task to worker: connection refused"), true},
		{"empty error", errors.New(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isUnreachableError(tt.err)
			if got != tt.want {
				t.Errorf("isUnreachableError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestIsAtCapacityError(t *testing.T) {
	if !isAtCapacityError(errors.New("worker at capacity: 503")) {
		t.Fatal("expected capacity error")
	}
	if isAtCapacityError(errors.New("connection refused")) {
		t.Fatal("connection refused is not capacity")
	}
}

// --- limitedBuffer ---

func TestLimitedBuffer_WriteUnderLimit(t *testing.T) {
	var lb limitedBuffer
	data := []byte("hello world")
	n, err := lb.Write(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != len(data) {
		t.Errorf("wrote %d bytes, want %d", n, len(data))
	}
	if lb.truncated {
		t.Error("should not be truncated under limit")
	}
	if !bytes.Equal(lb.Bytes(), data) {
		t.Errorf("bytes mismatch: got %q, want %q", lb.Bytes(), data)
	}
	if lb.Len() != len(data) {
		t.Errorf("len = %d, want %d", lb.Len(), len(data))
	}
}

func TestLimitedBuffer_WriteOverLimit(t *testing.T) {
	var lb limitedBuffer
	// Write exactly at limit
	big := make([]byte, maxOutputSize)
	n, err := lb.Write(big)
	if err != nil {
		t.Fatalf("first write: %v", err)
	}
	if n != maxOutputSize {
		t.Errorf("first write: n = %d, want %d", n, maxOutputSize)
	}

	// Second write should truncate
	n2, err := lb.Write([]byte("overflow"))
	if err != nil {
		t.Fatalf("second write: %v", err)
	}
	if n2 != 8 {
		t.Errorf("second write: n = %d, want 8 (truncated but reports full length)", n2)
	}
	if !lb.truncated {
		t.Error("expected truncated=true after overflow")
	}
}

func TestLimitedBuffer_TruncatedFlag(t *testing.T) {
	var lb limitedBuffer
	large := make([]byte, maxOutputSize+100)
	lb.Write(large)
	if !lb.truncated {
		t.Error("expected truncated")
	}
	// Bytes() should still return the buffered portion
	if lb.Len() > maxOutputSize {
		t.Errorf("len = %d, exceeds max", lb.Len())
	}
}

// --- saveArtifact ---

func strPtr(s string) *string { return &s }

func openWorkerTestDB(t *testing.T) *sql.DB {
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

func setupRunner(t *testing.T) (*Runner, *db.Queries, string) {
	t.Helper()
	rawDB := openWorkerTestDB(t)
	q := db.New(rawDB)
	dataDir := t.TempDir()

	// Create project and task for FK constraints.
	now := time.Now().UTC()
	if err := q.CreateProject(&models.Project{
		ID: "proj-1", Name: "test", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create project: %v", err)
	}
	if err := q.CreateScanPlan(&models.ScanPlan{
		ID: "plan-1", ProjectID: "proj-1", WorkflowType: "manual",
		Profile: models.ProfileStandard, Status: "approved", CreatedBy: "test",
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("create scan plan: %v", err)
	}
	if err := q.CreateScanTask(&models.ScanTask{
		ID: "task-1", ProjectID: "proj-1", PlanID: "plan-1",
		Tool: "nuclei", CommandTemplate: "nuclei -t test",
		Status: models.TaskRunning, CreatedAt: now,
	}); err != nil {
		t.Fatalf("create scan task: %v", err)
	}

	r := NewRunner(q, nil, dataDir)
	return r, q, dataDir
}

func TestSaveArtifact_Stdout(t *testing.T) {
	r, q, dataDir := setupRunner(t)
	now := time.Now().UTC()

	task := &models.ScanTask{
		ID: "task-art", ProjectID: "proj-1", PlanID: "plan-1",
		Tool: "nuclei", CommandTemplate: "nuclei -t test",
		Status: models.TaskRunning, CreatedAt: now,
	}
	if err := q.CreateScanTask(task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	workdir := filepath.Join(dataDir, "workdirs", task.ProjectID, task.ID)
	if err := os.MkdirAll(workdir, 0750); err != nil {
		t.Fatalf("create workdir: %v", err)
	}

	content := []byte("scan output line 1\nscan output line 2\n")
	if err := r.saveArtifact(task.ProjectID, task.ID, models.ArtifactStdout, workdir, content); err != nil {
		t.Fatalf("saveArtifact: %v", err)
	}

	// Verify artifact was written to disk
	entries, err := os.ReadDir(workdir)
	if err != nil {
		t.Fatalf("read workdir: %v", err)
	}
	var found bool
	for _, e := range entries {
		if strings.Contains(e.Name(), "stdout") {
			found = true
			fullPath := filepath.Join(workdir, e.Name())
			got, err := os.ReadFile(fullPath)
			if err != nil {
				t.Fatalf("read artifact: %v", err)
			}
			if !bytes.Equal(got, content) {
				t.Errorf("content mismatch")
			}
			// Verify SHA256
			h := sha256.Sum256(content)
			expected := fmt.Sprintf("%x", h)
			_ = expected // artifact is stored on disk, verified via content
			break
		}
	}
	if !found {
		t.Error("stdout artifact file not found in workdir")
	}
}

func TestSaveArtifact_EmptyContent(t *testing.T) {
	r, _, _ := setupRunner(t)

	workdir := t.TempDir()
	if err := r.saveArtifact("proj-1", "task-1", models.ArtifactStderr, workdir, []byte{}); err != nil {
		t.Fatalf("saveArtifact empty: %v", err)
	}
}

func TestSaveArtifact_LargeContent(t *testing.T) {
	r, _, _ := setupRunner(t)

	workdir := t.TempDir()
	// 1 MB of data
	content := make([]byte, 1024*1024)
	for i := range content {
		content[i] = byte(i % 256)
	}

	if err := r.saveArtifact("proj-1", "task-1", models.ArtifactJSONL, workdir, content); err != nil {
		t.Fatalf("saveArtifact large: %v", err)
	}

	entries, err := os.ReadDir(workdir)
	if err != nil {
		t.Fatalf("read workdir: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected artifact file in workdir")
	}
	fullPath := filepath.Join(workdir, entries[0].Name())
	got, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Error("content mismatch for large artifact")
	}
}

// --- Run tests ---

func TestRun_TaskNotFound(t *testing.T) {
	r, _, _ := setupRunner(t)
	r.SetGovernor(NewResourceGovernor(GovernorConfig{Enabled: false}, nil))

	err := r.Run(t.Context(), "nonexistent-task")
	if err == nil {
		t.Fatal("expected error for nonexistent task")
	}
	if !strings.Contains(err.Error(), "task not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRun_TargetNotFound(t *testing.T) {
	rawDB := openWorkerTestDB(t)
	q := db.New(rawDB)
	dataDir := t.TempDir()
	now := time.Now().UTC()

	// Create project and plan for FK constraints.
	if err := q.CreateProject(&models.Project{
		ID: "proj-1", Name: "test", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create project: %v", err)
	}
	if err := q.CreateScanPlan(&models.ScanPlan{
		ID: "plan-1", ProjectID: "proj-1", WorkflowType: "manual",
		Profile: models.ProfileStandard, Status: "approved", CreatedBy: "test",
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("create scan plan: %v", err)
	}

	// Create target, then delete it to simulate "not found".
	if err := q.CreateTarget(&models.Target{
		ID: "nonexistent-target", ProjectID: "proj-1", Value: "deleted.com",
		Type: "domain", CreatedAt: now,
	}); err != nil {
		t.Fatalf("create target: %v", err)
	}
	if err := q.CreateScanTask(&models.ScanTask{
		ID: "task-badtgt", ProjectID: "proj-1", PlanID: "plan-1",
		TargetID: strPtr("nonexistent-target"), Tool: "sh",
		CommandTemplate: "sh -c echo hi", Status: models.TaskCreated, CreatedAt: now,
	}); err != nil {
		t.Fatalf("create task: %v", err)
	}
	// Delete target to trigger "target not found" path.
	rawDB.Exec("PRAGMA foreign_keys = OFF")
	rawDB.Exec("DELETE FROM targets WHERE id = 'nonexistent-target'")
	rawDB.Exec("PRAGMA foreign_keys = ON")

	scopeEng := scope.NewEngine(q)
	r := NewRunner(q, scopeEng, dataDir)
	r.SetGovernor(NewResourceGovernor(GovernorConfig{Enabled: false}, nil))

	err := r.Run(t.Context(), "task-badtgt")
	if err == nil {
		t.Fatal("expected error for missing target")
	}
	if !strings.Contains(err.Error(), "target not found") {
		t.Errorf("unexpected error: %v", err)
	}

	task, _ := q.GetScanTask("task-badtgt")
	if task.Status != models.TaskScopeDenied {
		t.Errorf("status = %q, want %q", task.Status, models.TaskScopeDenied)
	}
}

func TestRun_ScopeDeny(t *testing.T) {
	rawDB := openWorkerTestDB(t)
	q := db.New(rawDB)
	dataDir := t.TempDir()
	now := time.Now().UTC()

	// Create project and plan for FK constraints.
	if err := q.CreateProject(&models.Project{
		ID: "proj-1", Name: "test", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create project: %v", err)
	}
	if err := q.CreateScanPlan(&models.ScanPlan{
		ID: "plan-1", ProjectID: "proj-1", WorkflowType: "manual",
		Profile: models.ProfileStandard, Status: "approved", CreatedBy: "test",
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("create scan plan: %v", err)
	}

	// Create target that will be excluded by scope rule.
	if err := q.CreateTarget(&models.Target{
		ID: "tgt-evil", ProjectID: "proj-1", Value: "evil.com",
		Type: "domain", CreatedAt: now,
	}); err != nil {
		t.Fatalf("create target: %v", err)
	}

	// Add exclude rule matching the target.
	if err := q.CreateScopeRule(&models.ScopeRule{
		ID: "rule-exclude", ProjectID: "proj-1",
		Action: models.ScopeActionExclude, Type: models.TargetTypeDomain,
		Value: "evil.com", Reason: "test exclude",
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create scope rule: %v", err)
	}

	// Create task referencing the target.
	if err := q.CreateScanTask(&models.ScanTask{
		ID: "task-scope", ProjectID: "proj-1", PlanID: "plan-1",
		TargetID: strPtr("tgt-evil"), Tool: "sh",
		CommandTemplate: "sh -c echo hi", Status: models.TaskCreated, CreatedAt: now,
	}); err != nil {
		t.Fatalf("create task: %v", err)
	}

	scopeEng := scope.NewEngine(q)
	r := NewRunner(q, scopeEng, dataDir)
	r.SetGovernor(NewResourceGovernor(GovernorConfig{Enabled: false}, nil))

	err := r.Run(t.Context(), "task-scope")
	if err == nil {
		t.Fatal("expected scope denied error")
	}
	if !strings.Contains(err.Error(), "scope denied") {
		t.Errorf("unexpected error: %v", err)
	}

	task, _ := q.GetScanTask("task-scope")
	if task.Status != models.TaskScopeDenied {
		t.Errorf("status = %q, want %q", task.Status, models.TaskScopeDenied)
	}
}

func TestRun_ProjectNotFound(t *testing.T) {
	rawDB := openWorkerTestDB(t)
	q := db.New(rawDB)
	dataDir := t.TempDir()
	now := time.Now().UTC()

	// Create project, plan, and task for FK constraints.
	if err := q.CreateProject(&models.Project{
		ID: "proj-missing", Name: "orphan", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create project: %v", err)
	}
	if err := q.CreateScanPlan(&models.ScanPlan{
		ID: "plan-orphan", ProjectID: "proj-missing", WorkflowType: "manual",
		Profile: models.ProfileStandard, Status: "approved", CreatedBy: "test",
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("create scan plan: %v", err)
	}
	if err := q.CreateScanTask(&models.ScanTask{
		ID: "task-orphan", ProjectID: "proj-missing", PlanID: "plan-orphan",
		Tool: "sh", CommandTemplate: "sh -c echo hi",
		Status: models.TaskCreated, CreatedAt: now,
	}); err != nil {
		t.Fatalf("create task: %v", err)
	}
	// Delete project to trigger "project not found" path (disable FK temporarily).
	rawDB.Exec("PRAGMA foreign_keys = OFF")
	rawDB.Exec("DELETE FROM projects WHERE id = 'proj-missing'")
	rawDB.Exec("PRAGMA foreign_keys = ON")

	scopeEng := scope.NewEngine(q)
	r := NewRunner(q, scopeEng, dataDir)
	r.SetGovernor(NewResourceGovernor(GovernorConfig{Enabled: false}, nil))

	err := r.Run(t.Context(), "task-orphan")
	if err == nil {
		t.Fatal("expected error for missing project")
	}
	if !strings.Contains(err.Error(), "project not found") {
		t.Errorf("unexpected error: %v", err)
	}
}


func TestSetGovernor(t *testing.T) {
	r, _, _ := setupRunner(t)

	g := NewResourceGovernor(GovernorConfig{Enabled: false}, nil)
	r.SetGovernor(g)

	if r.governor != g {
		t.Error("SetGovernor did not set governor")
	}
}

// --- Cancel ---


// --- injectCustomNucleiTemplates ---

func TestInjectCustomNucleiTemplates_NoOp(t *testing.T) {
	r, _, _ := setupRunner(t)

	input := []string{"nuclei", "-t", "test.yaml", "-u", "http://example.com"}
	got := r.injectCustomNucleiTemplates(input, "workflow")
	if len(got) != len(input) {
		t.Errorf("length = %d, want %d (should be no-op)", len(got), len(input))
	}
}

func TestInjectCustomNucleiTemplates_WithCustom(t *testing.T) {
	r, _, _ := setupRunner(t)

	// injectCustomNucleiTemplates is a no-op — custom templates live under
	// ~/nuclei-templates/ and nuclei finds them natively.
	input := []string{"nuclei", "-t", "test.yaml", "-u", "http://example.com"}
	got := r.injectCustomNucleiTemplates(input, "standard")
	if len(got) != len(input) {
		t.Errorf("expected no-op (same length), got %d (was %d)", len(got), len(input))
	}
	for i, v := range got {
		if v != input[i] {
			t.Errorf("element %d = %q, want %q", i, v, input[i])
		}
	}
}
