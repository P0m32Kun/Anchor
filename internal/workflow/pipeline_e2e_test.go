//go:build e2e
// +build e2e

package workflow

import (
	"context"
	"net"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/scope"
	"github.com/P0m32Kun/Anchor/internal/util"
	"github.com/P0m32Kun/Anchor/internal/worker"
)

func setupE2E(t *testing.T) (*db.Queries, *worker.Runner, *scope.Engine, string, func()) {
	t.Helper()

	dataDir, err := os.MkdirTemp("", "anchor-e2e-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}

	sqlDB, err := db.Open(dataDir)
	if err != nil {
		os.RemoveAll(dataDir)
		t.Fatalf("open db: %v", err)
	}

	queries := db.New(sqlDB)
	scopeEng := scope.NewEngine(queries)
	runner := worker.NewRunner(queries, scopeEng, dataDir)

	cleanup := func() {
		sqlDB.Close()
		os.RemoveAll(dataDir)
	}

	return queries, runner, scopeEng, dataDir, cleanup
}

func TestPipelineE2E(t *testing.T) {
	// Skip if nerva is not installed
	if _, err := exec.LookPath("nerva"); err != nil {
		t.Skip("nerva not installed")
	}

	// Skip if rangefield is not running (check nginx on port 18080)
	conn, err := net.DialTimeout("tcp", "127.0.0.1:18080", 2*time.Second)
	if err != nil {
		t.Skip("靶场 nginx 未就绪:", err)
	}
	conn.Close()

	queries, runner, scopeEng, dataDir, cleanup := setupE2E(t)
	defer cleanup()

	// b. Create test project
	project := &models.Project{
		ID:        util.GenerateID(),
		Name:      "E2E Test Project",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := queries.CreateProject(project); err != nil {
		t.Fatalf("create project: %v", err)
	}

	// c. Create IP target
	target := &models.Target{
		ID:        util.GenerateID(),
		ProjectID: project.ID,
		Type:      models.TargetTypeIP,
		Value:     "127.0.0.1",
		Source:    "e2e-test",
		Status:    "active",
		CreatedAt: time.Now().UTC(),
	}
	if err := queries.CreateTarget(target); err != nil {
		t.Fatalf("create target: %v", err)
	}

	// Create PipelineRun manually (same as API handler does)
	runID := util.GenerateID()
	now := time.Now().UTC()
	if err := queries.CreatePipelineRun(&models.PipelineRun{
		ID:        runID,
		ProjectID: project.ID,
		Status:    "running",
		StartedAt: now,
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("create pipeline run: %v", err)
	}

	// d. Run Pipeline
	pipeline := NewPipeline(queries, runner, scopeEng, dataDir).
		WithConfig(models.PipelineConfig{
			EnableCDNFilter:     false, // local IP, no CDN filter needed
			EnableNerva:         true,
			EnableNuclei:        true,
			PortScanConcurrency: 50,
			NervaConcurrency:    50,
			NucleiRateLimit:     100,
		}).
		WithRunID(runID)

	start := time.Now()
	err = pipeline.Run(context.Background(), project.ID)
	elapsed := time.Since(start)

	// e. Verify assertions

	// 1. Pipeline should complete without fatal error
	// (some tool errors are logged but not returned as fatal)
	if err != nil {
		t.Logf("Pipeline completed with error (may be non-fatal): %v", err)
	}

	// 2. PipelineRun should be created and have correct status
	runs, err := queries.ListPipelineRunsByProject(project.ID)
	if err != nil {
		t.Fatalf("list pipeline runs: %v", err)
	}
	if len(runs) < 1 {
		t.Fatalf("expected at least 1 pipeline run, got %d", len(runs))
	}

	run := runs[0]
	if run.Status != "completed" && run.Status != "failed" {
		t.Fatalf("unexpected pipeline run status: %s", run.Status)
	}
	if run.Stage == "" {
		t.Errorf("expected pipeline run stage to be non-empty")
	}

	// 3. ServiceFingerprint should have data
	fps, err := queries.ListServiceFingerprintsByProject(project.ID)
	if err != nil {
		t.Fatalf("list service fingerprints: %v", err)
	}
	if len(fps) == 0 {
		t.Fatalf("expected at least 1 service fingerprint, got %d", len(fps))
	}

	webCount := 0
	nonWebCount := 0
	for _, fp := range fps {
		if fp.IsWeb {
			webCount++
		} else {
			nonWebCount++
		}
	}

	if webCount < 1 {
		t.Errorf("expected at least 1 web fingerprint, got %d", webCount)
	}
	if nonWebCount < 1 {
		t.Errorf("expected at least 1 non-web fingerprint, got %d", nonWebCount)
	}

	// Report results
	t.Logf("=== E2E Test Results ===")
	t.Logf("Execution time: %s", elapsed)
	t.Logf("PipelineRun status: %s", run.Status)
	t.Logf("PipelineRun stage: %s", run.Stage)
	t.Logf("Total fingerprints: %d", len(fps))
	t.Logf("Web fingerprints: %d", webCount)
	t.Logf("Non-web fingerprints: %d", nonWebCount)
	for _, fp := range fps {
		t.Logf("  - %s:%d (%s) is_web=%v service=%s", fp.IP, fp.Port, fp.Protocol, fp.IsWeb, fp.Service)
	}
}
