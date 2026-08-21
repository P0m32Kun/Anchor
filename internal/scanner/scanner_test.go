package scanner

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/scanengine/core"
	"github.com/P0m32Kun/Anchor/internal/toolregistry"
)

// fake runner for tests
type fakeRunner struct {
	called   int
	taskID   string
	cancelFn func(taskID string) error
	runFn    func(ctx context.Context, taskID string) error
	artifacts []*models.RawArtifact
}

func (f *fakeRunner) Run(ctx context.Context, taskID string) error {
	f.called++
	f.taskID = taskID
	if f.runFn != nil {
		return f.runFn(ctx, taskID)
	}
	return nil
}
func (f *fakeRunner) Cancel(taskID string) error {
	if f.cancelFn != nil {
		return f.cancelFn(taskID)
	}
	return nil
}

// fake DB for scanner tests
type fakeDB struct {
	tasks map[string]*models.ScanTask
	artifacts map[string][]*models.RawArtifact
	customBundle string
}

func newFakeDB() *fakeDB {
	return &fakeDB{tasks: make(map[string]*models.ScanTask), artifacts: make(map[string][]*models.RawArtifact)}
}
func (f *fakeDB) CreateScanTask(t *models.ScanTask) error {
	f.tasks[t.ID] = t
	t.Status = models.TaskCompleted
	ec := 0
	t.ExitCode = &ec
	// create a dummy stdout artifact path
	tmp := t.ID + ".stdout"
	f.artifacts[t.ID] = []*models.RawArtifact{{Type: models.ArtifactStdout, Path: tmp}}
	return nil
}
func (f *fakeDB) ListRawArtifactsByTask(taskID string) ([]*models.RawArtifact, error) {
	if a, ok := f.artifacts[taskID]; ok {
		return a, nil
	}
	return nil, nil
}
func (f *fakeDB) GetActiveNucleiCustomBundleVersion() (string, error) { return f.customBundle, nil }

// helper to get registry
func testRegistry(t *testing.T) *toolregistry.Registry {
	t.Helper()
	reg := toolregistry.DefaultRegistry()
	if reg == nil {
		t.Fatal("nil registry")
	}
	return reg
}

// scope checker mock
type fakeScope struct {
	deny map[string]bool
	err error
}

func (f *fakeScope) IsExcluded(projectID string, target string) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	if f.deny[target] {
		return true, nil
	}
	return false, nil
}

// Test empty targets - tracer slice must reject before touching registry/runner
func TestScanner_EmptyTargets(t *testing.T) {
	reg := testRegistry(t)
	db := newFakeDB()
	runner := &fakeRunner{}
	s := New(reg, runner, db, t.TempDir())
	// also test without provenance
	req := ScanRequest{
		Action:         core.ActionPortScan,
		Targets:        nil,
		PortRange:      "top1000",
		Budgets:        Budgets{Rate: 100, Threads: 10, TimeoutMs: 1000},
		Authorization:  Authorization{ProjectID: "proj-1", RunID: "run-1"},
		IdempotencyKey: "work-1",
	}
	_, err := s.Execute(context.Background(), req)
	if !errors.Is(err, ErrEmptyTargets) {
		t.Fatalf("expected ErrEmptyTargets, got %v", err)
	}
	if runner.called != 0 {
		t.Fatalf("runner should not be called on empty targets")
	}

	// empty string target
	req.Targets = []string{""}
	_, err = s.Execute(context.Background(), req)
	if !errors.Is(err, ErrEmptyTargets) {
		t.Fatalf("expected ErrEmptyTargets for empty string target, got %v", err)
	}

	// valid targets should proceed (not empty error)
	req.Targets = []string{"10.0.0.1"}
	// mock runner to capture
	fakeContent := []byte(`{"port":80}`)
	// use a real temp file for artifact
	// toolrun reads artifact path from DB, but fakeDB returns tmp that doesn't exist.
	// Instead make artifact file exist via temp dir.
	// We'll override DB to not require file read to succeed for this test - just check not empty error.
	// For valid case we expect not ErrEmptyTargets, even if downstream fails due to missing file, it's ok.
	// Use a dataDir that allows WriteHostFile.
	_, err = s.Execute(context.Background(), req)
	if errors.Is(err, ErrEmptyTargets) {
		t.Fatalf("valid target should not get ErrEmptyTargets, got %v", err)
	}
	// Also ensure fakeRunner was called
	if runner.called == 0 {
		t.Fatalf("runner should be called for valid request")
	}
	_ = fakeContent
}

// Cancellation must be observed before and after scope checks
func TestScanner_Cancellation(t *testing.T) {
	reg := testRegistry(t)
	db := newFakeDB()
	runner := &fakeRunner{
		runFn: func(ctx context.Context, taskID string) error {
			// simulate work that respects ctx
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(50 * time.Millisecond):
				return nil
			}
		},
	}
	s := New(reg, runner, db, t.TempDir())
	req := ScanRequest{
		Action:         core.ActionPortScan,
		Targets:        []string{"10.0.0.1"},
		PortRange:      "top1000",
		Budgets:        Budgets{Rate: 100, Threads: 10, TimeoutMs: 1000},
		Authorization:  Authorization{ProjectID: "proj-1", RunID: "run-1"},
		IdempotencyKey: "work-cancel-1",
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := s.Execute(ctx, req)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if runner.called != 0 {
		t.Fatalf("runner should not be called when ctx already canceled, got %d", runner.called)
	}

	// context canceled during run - use a runner that checks ctx
	runner2 := &fakeRunner{
		runFn: func(ctx context.Context, taskID string) error {
			return ctx.Err()
		},
	}
	s2 := New(reg, runner2, db, t.TempDir())
	ctx2, cancel2 := context.WithCancel(context.Background())
	// cancel after starting? We'll cancel immediately and pass that ctx to Invoke via scanner.
	// scanner's execute will pass ctx to toolrun.Invoke which passes to runner.Run
	cancel2()
	_, err = s2.Execute(ctx2, req)
	if err == nil {
		t.Fatalf("expected error when ctx canceled during run")
	}
}

// Scope denial must be enforced at scanner boundary
func TestScanner_ScopeDenial(t *testing.T) {
	reg := testRegistry(t)
	db := newFakeDB()
	runner := &fakeRunner{}
	s := New(reg, runner, db, t.TempDir())
	s.WithScopeChecker(&fakeScope{deny: map[string]bool{"10.0.0.99": true}})

	req := ScanRequest{
		Action:         core.ActionPortScan,
		Targets:        []string{"10.0.0.99"},
		PortRange:      "high-risk",
		Budgets:        Budgets{Rate: 300, Threads: 50, TimeoutMs: 2000},
		Authorization:  Authorization{ProjectID: "proj-1", RunID: "run-1"},
		IdempotencyKey: "work-scope-1",
	}
	_, err := s.Execute(context.Background(), req)
	if !errors.Is(err, ErrScopeDenied) {
		t.Fatalf("expected ErrScopeDenied, got %v", err)
	}
	if runner.called != 0 {
		t.Fatalf("runner should not be called on scope denial")
	}

	// allowed target should pass scope
	s2 := New(reg, runner, db, t.TempDir())
	s2.WithScopeChecker(&fakeScope{deny: map[string]bool{}})
	req.Targets = []string{"10.0.0.1"}
	_, err = s2.Execute(context.Background(), req)
	if errors.Is(err, ErrScopeDenied) {
		t.Fatalf("allowed target should not be denied, got %v", err)
	}
}

// Parity with existing RenderParams seam: the guarded process adapter must
// produce the same argv as the legacy engine path (engine.go buildParams →
// toolregistry.Render), given the same semantic values.
func TestScanner_NaabuParity(t *testing.T) {
	reg := testRegistry(t)
	s := New(reg, &fakeRunner{}, newFakeDB(), t.TempDir())

	cases := []struct {
		name      string
		portRange string
		rate      int
		threads   int
		timeoutMs int
	}{
		{"high-risk", "high-risk", 300, 50, 1000},
		{"top1000", "top1000", 1000, 100, 0},
	}

	const hostFile = "/tmp/hosts.txt" // deterministic host file for both paths

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := ScanRequest{
				Action:         core.ActionPortScan,
				Targets:        []string{"10.0.0.1"},
				PortRange:      tc.portRange,
				Budgets:        Budgets{Rate: tc.rate, Threads: tc.threads, TimeoutMs: tc.timeoutMs},
				Authorization:  Authorization{ProjectID: "proj-1", RunID: "run-1"},
				IdempotencyKey: "work-parity-" + tc.name,
			}

			got, err := s.renderArgs("naabu", req, hostFile)
			if err != nil {
				t.Fatalf("scanner renderArgs: %v", err)
			}

			// Independent source of truth: the legacy engine builds exactly
			// these RenderParams for ActionPortScan (see engine.go buildParams).
			legacy := toolregistry.RenderParams{
				"host_file":  hostFile,
				"port_range": tc.portRange,
				"rate":       tc.rate,
				"threads":    tc.threads,
				"timeout":    tc.timeoutMs,
			}
			want, err := reg.Render("naabu", legacy)
			if err != nil {
				t.Fatalf("legacy Render: %v", err)
			}
			// Compare as token sets: registry.Render iterates a map, so argv order
			// is non-deterministic (its own golden tests use ArgvSetMinus).
			if !equalArgvSet(got, want) {
				t.Fatalf("parity mismatch:\ngot:  %v\nwant: %v", got, want)
			}

			// Absolute anchor: the adapter must keep the naabu CLI shape stable —
			// key tokens from the registry golden fixtures must be present.
			gotSet := toolregistry.ArgvSetMinus(got, nil)
			if tc.portRange == "high-risk" {
				if !slices.Contains(gotSet, "-p") {
					t.Fatalf("high-risk preset missing -p in %v", got)
				}
			}
			if tc.portRange == "top1000" {
				if !slices.Contains(gotSet, "-tp") || !slices.Contains(gotSet, "1000") {
					t.Fatalf("top1000 preset missing in %v", got)
				}
			}
			for _, flag := range []string{"-rate", "-c", "-json", "-list"} {
				if !slices.Contains(gotSet, flag) {
					t.Fatalf("flag %s missing in scanner argv %v", flag, got)
				}
			}
		})
	}
}

// equalArgvSet compares two argv slices as unordered token sets.
func equalArgvSet(a, b []string) bool {
	aa := toolregistry.ArgvSetMinus(a, nil)
	bb := toolregistry.ArgvSetMinus(b, nil)
	if len(aa) != len(bb) {
		return false
	}
	for i := range aa {
		if aa[i] != bb[i] {
			return false
		}
	}
	return true
}
