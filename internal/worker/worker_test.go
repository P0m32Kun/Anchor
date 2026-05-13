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

func TestLimitedBuffer_Truncation(t *testing.T) {
	var lb limitedBuffer
	// Write exactly maxOutputSize bytes.
	big := bytes.Repeat([]byte("x"), maxOutputSize)
	n, err := lb.Write(big)
	if err != nil {
		t.Fatalf("first write error: %v", err)
	}
	if n != maxOutputSize {
		t.Errorf("first write: got %d, want %d", n, maxOutputSize)
	}
	if lb.truncated {
		t.Error("should not be truncated at exactly maxOutputSize")
	}

	// One more byte should trigger truncation.
	extra := []byte("y")
	n2, err := lb.Write(extra)
	if err != nil {
		t.Fatalf("second write error: %v", err)
	}
	if n2 != 1 {
		t.Errorf("second write: got %d, want 1", n2)
	}
	if !lb.truncated {
		t.Error("should be truncated after exceeding maxOutputSize")
	}
}

func TestLimitedBuffer_MultipleWrites(t *testing.T) {
	var lb limitedBuffer
	for i := 0; i < 10; i++ {
		lb.Write([]byte(strings.Repeat("a", 100)))
	}
	if lb.Len() != 1000 {
		t.Errorf("len = %d, want 1000", lb.Len())
	}
	if lb.truncated {
		t.Error("should not be truncated at 1000 bytes")
	}
}

// --- defaultToolTimeout ---

func TestDefaultToolTimeout(t *testing.T) {
	tests := []struct {
		tool string
		want int // expected minutes
	}{
		{"subfinder", 10},
		{"httpx", 10},
		{"nmap", 10},
		{"naabu", 30},
		{"nuclei", 60},
		{"unknown", 10},
	}
	for _, tt := range tests {
		t.Run(tt.tool, func(t *testing.T) {
			got := defaultToolTimeout(tt.tool)
			if got.Minutes() != float64(tt.want) {
				t.Errorf("defaultToolTimeout(%q) = %v, want %dm", tt.tool, got, tt.want)
			}
		})
	}
}

// --- detectScanStrategy ---

func TestDetectScanStrategy(t *testing.T) {
	tests := []struct {
		name    string
		command []string
		want    string
	}{
		{"no flags", []string{"naabu", "-list", "hosts.txt"}, "default"},
		{"top ports 100", []string{"naabu", "-tp", "100"}, "top100"},
		{"top ports 1000", []string{"naabu", "-tp", "1000"}, "top1000"},
		{"top-ports flag", []string{"naabu", "--top-ports", "100"}, "top100"},
		{"full scan -p-", []string{"naabu", "-p-", "-list", "hosts.txt"}, "full"},
		{"custom port -p", []string{"naabu", "-p", "80,443"}, "default"},
		{"full scan -p full", []string{"naabu", "-p", "full"}, "full"},
		{"empty command", []string{}, "default"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectScanStrategy(tt.command)
			if got != tt.want {
				t.Errorf("detectScanStrategy(%v) = %q, want %q", tt.command, got, tt.want)
			}
		})
	}
}

// --- resolveTimeoutConfig ---

func TestResolveTimeoutConfig_Nuclei(t *testing.T) {
	cmd := []string{"nuclei", "-t", "templates/", "-l", "targets.txt"}
	cfg := resolveTimeoutConfig("nuclei", cmd, "/tmp/workdir")

	if cfg.CPUCheckEnabled {
		t.Error("nuclei should not have CPU check enabled (uses stats heartbeat)")
	}
	if cfg.IdleTimeout.Seconds() != 60 {
		t.Errorf("nuclei idle timeout = %v, want 60s", cfg.IdleTimeout)
	}
}

func TestResolveTimeoutConfig_Nmap(t *testing.T) {
	cmd := []string{"nmap", "-sV", "-iL", "hosts.txt"}
	cfg := resolveTimeoutConfig("nmap", cmd, "/tmp/workdir")

	if cfg.IdleTimeout.Seconds() != 120 {
		t.Errorf("nmap idle timeout = %v, want 120s", cfg.IdleTimeout)
	}
	if !cfg.CPUCheckEnabled {
		t.Error("nmap should have CPU check enabled")
	}
}

func TestResolveTimeoutConfig_Naabu_Full(t *testing.T) {
	cmd := []string{"naabu", "-p-", "-list", "hosts.txt"}
	cfg := resolveTimeoutConfig("naabu", cmd, "/tmp/workdir")

	if cfg.IdleTimeout.Minutes() != 5 {
		t.Errorf("naabu full scan idle timeout = %v, want 5m", cfg.IdleTimeout)
	}
}

func TestResolveTimeoutConfig_Naabu_TopPorts(t *testing.T) {
	cmd := []string{"naabu", "-tp", "100", "-list", "hosts.txt"}
	cfg := resolveTimeoutConfig("naabu", cmd, "/tmp/workdir")

	if cfg.IdleTimeout.Seconds() != 90 {
		t.Errorf("naabu top100 idle timeout = %v, want 90s", cfg.IdleTimeout)
	}
}

func TestResolveTimeoutConfig_Httpx(t *testing.T) {
	cmd := []string{"httpx", "-l", "urls.txt"}
	cfg := resolveTimeoutConfig("httpx", cmd, "/tmp/workdir")

	if cfg.IdleTimeout.Seconds() != 60 {
		t.Errorf("httpx idle timeout = %v, want 60s", cfg.IdleTimeout)
	}
}

func TestResolveTimeoutConfig_Default(t *testing.T) {
	cmd := []string{"custom-tool", "-arg"}
	cfg := resolveTimeoutConfig("custom-tool", cmd, "/tmp/workdir")

	if cfg.StartupTimeout.Seconds() != 30 {
		t.Errorf("default startup timeout = %v, want 30s", cfg.StartupTimeout)
	}
	if !cfg.CPUCheckEnabled {
		t.Error("default should have CPU check enabled")
	}
}

// --- saveArtifact ---

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
	workdir := filepath.Join(dataDir, "workdirs", "proj-1", "task-1")
	if err := os.MkdirAll(workdir, 0750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	data := []byte("hello stdout output")
	if err := r.saveArtifact("proj-1", "task-1", models.ArtifactStdout, workdir, data); err != nil {
		t.Fatalf("saveArtifact: %v", err)
	}

	// Verify file was written.
	files, _ := os.ReadDir(workdir)
	if len(files) != 1 {
		t.Fatalf("expected 1 file in workdir, got %d", len(files))
	}
	if !strings.Contains(files[0].Name(), string(models.ArtifactStdout)) {
		t.Errorf("filename should contain artifact type, got %q", files[0].Name())
	}

	// Verify DB record.
	arts, err := q.ListRawArtifactsByTask("task-1")
	if err != nil {
		t.Fatalf("ListRawArtifactsByTask: %v", err)
	}
	if len(arts) != 1 {
		t.Fatalf("expected 1 artifact in DB, got %d", len(arts))
	}
	a := arts[0]
	if a.Type != models.ArtifactStdout {
		t.Errorf("type = %q, want %q", a.Type, models.ArtifactStdout)
	}
	if a.Size != int64(len(data)) {
		t.Errorf("size = %d, want %d", a.Size, len(data))
	}
	sum := sha256.Sum256(data)
	expectedSHA := fmt.Sprintf("%x", sum)
	if a.SHA256 != expectedSHA {
		t.Errorf("sha256 = %q, want %q", a.SHA256, expectedSHA)
	}
	if a.RedactionStatus != "unchecked" {
		t.Errorf("redaction_status = %q, want %q", a.RedactionStatus, "unchecked")
	}
}

func TestSaveArtifact_Stderr(t *testing.T) {
	r, q, dataDir := setupRunner(t)
	workdir := filepath.Join(dataDir, "workdirs", "proj-1", "task-1")
	if err := os.MkdirAll(workdir, 0750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	data := []byte("error output")
	if err := r.saveArtifact("proj-1", "task-1", models.ArtifactStderr, workdir, data); err != nil {
		t.Fatalf("saveArtifact: %v", err)
	}

	arts, err := q.ListRawArtifactsByTask("task-1")
	if err != nil {
		t.Fatalf("ListRawArtifactsByTask: %v", err)
	}
	if len(arts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(arts))
	}
	if arts[0].Type != models.ArtifactStderr {
		t.Errorf("type = %q, want %q", arts[0].Type, models.ArtifactStderr)
	}
}

func TestSaveArtifact_JSONL(t *testing.T) {
	r, q, dataDir := setupRunner(t)
	workdir := filepath.Join(dataDir, "workdirs", "proj-1", "task-1")
	if err := os.MkdirAll(workdir, 0750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	data := []byte(`{"host":"example.com","port":80}`)
	if err := r.saveArtifact("proj-1", "task-1", models.ArtifactJSONL, workdir, data); err != nil {
		t.Fatalf("saveArtifact: %v", err)
	}

	arts, err := q.ListRawArtifactsByTask("task-1")
	if err != nil {
		t.Fatalf("ListRawArtifactsByTask: %v", err)
	}
	if len(arts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(arts))
	}
	if arts[0].Type != models.ArtifactJSONL {
		t.Errorf("type = %q, want %q", arts[0].Type, models.ArtifactJSONL)
	}
}

func TestSaveArtifact_EmptyData(t *testing.T) {
	r, _, dataDir := setupRunner(t)
	workdir := filepath.Join(dataDir, "workdirs", "proj-1", "task-1")
	if err := os.MkdirAll(workdir, 0750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if err := r.saveArtifact("proj-1", "task-1", models.ArtifactStdout, workdir, []byte{}); err != nil {
		t.Fatalf("saveArtifact empty: %v", err)
	}
}

func TestSaveArtifact_MultipleArtifacts(t *testing.T) {
	r, q, dataDir := setupRunner(t)
	workdir := filepath.Join(dataDir, "workdirs", "proj-1", "task-1")
	if err := os.MkdirAll(workdir, 0750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	types := []models.ArtifactType{
		models.ArtifactStdout,
		models.ArtifactStderr,
		models.ArtifactJSONL,
	}
	for _, at := range types {
		if err := r.saveArtifact("proj-1", "task-1", at, workdir, []byte("data")); err != nil {
			t.Fatalf("saveArtifact %s: %v", at, err)
		}
	}

	arts, err := q.ListRawArtifactsByTask("task-1")
	if err != nil {
		t.Fatalf("ListRawArtifactsByTask: %v", err)
	}
	if len(arts) != 3 {
		t.Fatalf("expected 3 artifacts, got %d", len(arts))
	}
}
