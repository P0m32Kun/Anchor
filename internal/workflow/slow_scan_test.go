package workflow

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/scope"
	"github.com/P0m32Kun/Anchor/internal/util"
	"github.com/P0m32Kun/Anchor/internal/worker"
)

// setupSlowScanTest spins up an in-process SQLite DB, project, and pipeline run,
// returning the pieces SlowScanOrchestrator needs plus an emitter wired to a
// spying callback so tests can assert exactly what stage events fired.
//
// The returned *events captures (stage, status, errMsg) for every callback hit,
// in order, under the mutex.
type slowScanFixture struct {
	queries   *db.Queries
	runner    *worker.Runner
	dataDir   string
	projectID string
	runID     string
	emitter   *StageEmitter
	events    *[]stageEvent
	eventsMu  *sync.Mutex
	cleanup   func()
}

type stageEvent struct {
	Stage  StageID
	Status string
	Err    string
}

func setupSlowScanTest(t *testing.T) *slowScanFixture {
	t.Helper()
	dataDir, err := os.MkdirTemp("", "anchor-slowscan-*")
	if err != nil {
		t.Fatalf("temp dir: %v", err)
	}
	sqlDB, err := db.Open(dataDir)
	if err != nil {
		os.RemoveAll(dataDir)
		t.Fatalf("open db: %v", err)
	}
	queries := db.New(sqlDB)
	scopeEng := scope.NewEngine(queries)
	runner := worker.NewRunner(queries, scopeEng, dataDir)

	projectID := util.GenerateID()
	runID := util.GenerateID()
	if err := queries.CreateProject(&models.Project{
		ID:        projectID,
		Name:      "slowscan-test",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("create project: %v", err)
	}
	if err := queries.CreatePipelineRun(&models.PipelineRun{
		ID:        runID,
		ProjectID: projectID,
		Status:    "running",
		StartedAt: time.Now().UTC(),
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("create pipeline_run: %v", err)
	}

	events := []stageEvent{}
	var mu sync.Mutex
	cb := func(_ string, stage StageID, status, errMsg string) {
		mu.Lock()
		defer mu.Unlock()
		events = append(events, stageEvent{Stage: stage, Status: status, Err: errMsg})
	}
	emitter := NewStageEmitter(queries, runID, cb)

	return &slowScanFixture{
		queries:   queries,
		runner:    runner,
		dataDir:   dataDir,
		projectID: projectID,
		runID:     runID,
		emitter:   emitter,
		events:    &events,
		eventsMu:  &mu,
		cleanup: func() {
			sqlDB.Close()
			os.RemoveAll(dataDir)
		},
	}
}

func (f *slowScanFixture) snapshotEvents() []stageEvent {
	f.eventsMu.Lock()
	defer f.eventsMu.Unlock()
	out := make([]stageEvent, len(*f.events))
	copy(out, *f.events)
	return out
}

func (f *slowScanFixture) addWebEndpoint(t *testing.T, host, url string) {
	t.Helper()
	// web_endpoints.asset_id has a NOT NULL FK to assets(id), so we need a
	// parent asset row before we can insert the endpoint.
	asset := &models.Asset{
		ID:              util.GenerateID(),
		ProjectID:       f.projectID,
		Type:            models.AssetTypeURL,
		Value:           url,
		NormalizedValue: url,
		FirstSeen:       time.Now().UTC(),
		LastSeen:        time.Now().UTC(),
	}
	if err := f.queries.CreateAsset(asset); err != nil {
		t.Fatalf("create asset: %v", err)
	}
	ep := &models.WebEndpoint{
		ID:        util.GenerateID(),
		ProjectID: f.projectID,
		AssetID:   asset.ID,
		URL:       url,
		Host:      host,
		CreatedAt: time.Now().UTC(),
	}
	if err := f.queries.CreateWebEndpoint(ep); err != nil {
		t.Fatalf("create web endpoint: %v", err)
	}
}

// ─────────────────────────── finalizeStage ───────────────────────────

func TestFinalizeStage_NoAttempts_Completes(t *testing.T) {
	f := setupSlowScanTest(t)
	defer f.cleanup()

	finalizeStage(f.emitter, StageFfuf, 0, 0, nil)

	events := f.snapshotEvents()
	if len(events) != 1 || events[0].Stage != StageFfuf || events[0].Status != "completed" {
		t.Fatalf("events = %+v, want one completed ffuf event", events)
	}
}

func TestFinalizeStage_AllSucceed_Completes(t *testing.T) {
	f := setupSlowScanTest(t)
	defer f.cleanup()
	f.emitter.Set(StageFfuf)

	finalizeStage(f.emitter, StageFfuf, 3, 0, nil)

	events := f.snapshotEvents()
	if len(events) < 2 || events[len(events)-1].Status != "completed" {
		t.Fatalf("terminal event = %+v, want completed", events)
	}
}

func TestFinalizeStage_AllFail_FailsWithAggregateReason(t *testing.T) {
	f := setupSlowScanTest(t)
	defer f.cleanup()
	f.emitter.Set(StageFfuf)

	finalizeStage(f.emitter, StageFfuf, 3, 3, []string{"a: timeout", "b: 500", "c: dns"})

	events := f.snapshotEvents()
	terminal := events[len(events)-1]
	if terminal.Status != "failed" {
		t.Fatalf("status = %q, want failed", terminal.Status)
	}
	if !strings.Contains(terminal.Err, "3/3 targets failed") {
		t.Fatalf("err = %q, want '3/3 targets failed' prefix", terminal.Err)
	}
	for _, want := range []string{"timeout", "500", "dns"} {
		if !strings.Contains(terminal.Err, want) {
			t.Fatalf("err = %q, missing %q", terminal.Err, want)
		}
	}
}

func TestFinalizeStage_PartialFail_StillCompletes(t *testing.T) {
	// Partial failures don't mark the stage as failed — the user still gets
	// the results from the goroutines that succeeded. The failure detail goes
	// to the log, not the stage.
	f := setupSlowScanTest(t)
	defer f.cleanup()
	f.emitter.Set(StageFfuf)

	finalizeStage(f.emitter, StageFfuf, 3, 1, []string{"b: 500"})

	events := f.snapshotEvents()
	terminal := events[len(events)-1]
	if terminal.Status != "completed" {
		t.Fatalf("status = %q, want completed (partial failure)", terminal.Status)
	}
}

// ─────────────────────────── Run() decision branches ───────────────────────────

func TestRun_NoEndpoints_FfufEnabled_StillSilent(t *testing.T) {
	f := setupSlowScanTest(t)
	defer f.cleanup()

	orch := NewSlowScanOrchestrator(f.queries, f.runner, f.dataDir).
		WithConfig(models.PipelineConfig{
			EnableFfuf:       true,
			FfufDictionaryID: "some-dict-id",
		}).
		WithStageEmitter(f.emitter)

	if err := orch.Run(context.Background(), f.projectID, f.runID); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if got := f.snapshotEvents(); len(got) != 0 {
		t.Fatalf("with zero endpoints, no stage events should fire, got %+v", got)
	}
}

// REGRESSION: this is the Fix 3 backend backstop. If a caller bypasses the
// frontend's disabled-button guard and POSTs {enable_ffuf:true, dict:""} we
// MUST surface the misconfiguration as a failed ffuf stage. Otherwise users
// will keep seeing "ffuf silently skipped" — the exact bug Fix 3 closes.
func TestRun_FfufEnabledNoDict_EmitsFailedStage(t *testing.T) {
	f := setupSlowScanTest(t)
	defer f.cleanup()
	f.addWebEndpoint(t, "example.com", "http://example.com/")

	orch := NewSlowScanOrchestrator(f.queries, f.runner, f.dataDir).
		WithConfig(models.PipelineConfig{
			EnableFfuf:       true,
			FfufDictionaryID: "",
		}).
		WithStageEmitter(f.emitter)

	if err := orch.Run(context.Background(), f.projectID, f.runID); err != nil {
		t.Fatalf("Run: %v", err)
	}

	events := f.snapshotEvents()
	if len(events) != 1 {
		t.Fatalf("events = %+v, want exactly 1 (ffuf failed)", events)
	}
	if events[0].Stage != StageFfuf || events[0].Status != "failed" {
		t.Fatalf("event = %+v, want ffuf failed", events[0])
	}
	if !strings.Contains(events[0].Err, "no dictionary configured") {
		t.Fatalf("err = %q, want 'no dictionary configured' substring", events[0].Err)
	}

	// Reload-after-event: the failed row must be persisted, not just SSE'd.
	row, _ := f.queries.GetPipelineRunStage(f.runID, string(StageFfuf))
	if row == nil || row.Status != models.StageStatusFailed {
		t.Fatalf("ffuf row not persisted as failed: %+v", row)
	}
}

func TestRun_FfufDisabled_NoStageEvents(t *testing.T) {
	f := setupSlowScanTest(t)
	defer f.cleanup()
	f.addWebEndpoint(t, "example.com", "http://example.com/")

	orch := NewSlowScanOrchestrator(f.queries, f.runner, f.dataDir).
		WithConfig(models.PipelineConfig{
			EnableFfuf: false,
		}).
		WithStageEmitter(f.emitter)

	if err := orch.Run(context.Background(), f.projectID, f.runID); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if got := f.snapshotEvents(); len(got) != 0 {
		t.Fatalf("ffuf disabled but stage events fired: %+v", got)
	}
}
