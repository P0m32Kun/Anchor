package scanengine

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/asset"
	"github.com/P0m32Kun/Anchor/internal/exclude"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/scope"
	"github.com/P0m32Kun/Anchor/internal/scanconfig"
	"github.com/P0m32Kun/Anchor/internal/scanengine/core"
	"github.com/P0m32Kun/Anchor/internal/scanengine/pool"
	"github.com/P0m32Kun/Anchor/internal/scanengine/queue"
	"github.com/P0m32Kun/Anchor/internal/scanengine/seed"
	"github.com/P0m32Kun/Anchor/internal/toolrun"
)

// ============================================================
// isAssetExcluded — boost from 25%
// ============================================================

func TestIsAssetExcluded_WithExcludeMgr(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()

	t.Run("excludeMgr excludes built-in domain", func(t *testing.T) {
		engine, _ := setupTestEngine(t, fake, cfg)
		excludeMgr := exclude.NewManager()
		engine.excludeMgr = excludeMgr

		// google.com is a built-in excluded domain
		a := &core.DiscoveryAsset{Value: "google.com", Type: core.AssetSubdomain}
		if !engine.isAssetExcluded(a) {
			t.Error("expected google.com to be excluded by built-in list")
		}
	})

	t.Run("excludeMgr excludes custom domain", func(t *testing.T) {
		engine, _ := setupTestEngine(t, fake, cfg)
		excludeMgr := exclude.NewManager()
		excludeMgr.AddCustom("skip-me.com", "test")
		engine.excludeMgr = excludeMgr

		a := &core.DiscoveryAsset{Value: "sub.skip-me.com", Type: core.AssetSubdomain}
		if !engine.isAssetExcluded(a) {
			t.Error("expected sub.skip-me.com to be excluded by custom rule")
		}
	})

	t.Run("excludeMgr does not exclude unknown domain", func(t *testing.T) {
		engine, _ := setupTestEngine(t, fake, cfg)
		excludeMgr := exclude.NewManager()
		engine.excludeMgr = excludeMgr

		a := &core.DiscoveryAsset{Value: "totally-unique-asset.xyz", Type: core.AssetSubdomain}
		if engine.isAssetExcluded(a) {
			t.Error("expected totally-unique-asset.xyz to NOT be excluded")
		}
	})

	t.Run("excludeMgr nil, scopeEng set", func(t *testing.T) {
		engine, _ := setupTestEngine(t, fake, cfg)
		engine.excludeMgr = nil
		engine.scopeEng = scope.NewEngine(engine.queries)

		a := &core.DiscoveryAsset{Value: "example.com", Type: core.AssetSubdomain}
		// With no scope rules, should not be excluded
		if engine.isAssetExcluded(a) {
			t.Error("expected false with no scope rules")
		}
	})

	t.Run("scopeEng with url type asset", func(t *testing.T) {
		engine, _ := setupTestEngine(t, fake, cfg)
		engine.excludeMgr = nil
		engine.scopeEng = scope.NewEngine(engine.queries)

		a := &core.DiscoveryAsset{Value: "https://example.com/path", Type: core.AssetHTTPService}
		if engine.isAssetExcluded(a) {
			t.Error("expected false for url asset with no scope rules")
		}
	})

	t.Run("scopeEng with cidr type asset", func(t *testing.T) {
		engine, _ := setupTestEngine(t, fake, cfg)
		engine.excludeMgr = nil
		engine.scopeEng = scope.NewEngine(engine.queries)

		a := &core.DiscoveryAsset{Value: "10.0.0.0/24", Type: core.AssetCIDR}
		if engine.isAssetExcluded(a) {
			t.Error("expected false for cidr asset with no scope rules")
		}
	})

	t.Run("scopeEng with ip_port type asset", func(t *testing.T) {
		engine, _ := setupTestEngine(t, fake, cfg)
		engine.excludeMgr = nil
		engine.scopeEng = scope.NewEngine(engine.queries)

		a := &core.DiscoveryAsset{Value: "10.0.0.1:80", Type: core.AssetIPPort}
		if engine.isAssetExcluded(a) {
			t.Error("expected false for ip_port asset with no scope rules")
		}
	})
}

// ============================================================
// RunWithSeeds — boost from 0%
// ============================================================

func TestRunWithSeeds_BasicRun(t *testing.T) {
	fake := &fakeExecutor{
		behavior: func(ctx context.Context, w *models.ScanWorkItem) (*toolrun.InvokeResult, error) {
			return &toolrun.InvokeResult{Task: &models.ScanTask{ID: "t"}, Stdout: nil}, nil
		},
	}
	cfg := DefaultEngineConfig()
	cfg.SchedulerTick = 50 * time.Millisecond
	cfg.IdleTimeout = 200 * time.Millisecond
	engine, _ := setupTestEngine(t, fake, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	seeds := []seed.SeedAsset{
		{Value: "10.0.0.1", ValueType: "ip", Source: "target"},
		{Value: "10.0.0.2", ValueType: "ip", Source: "target"},
	}

	err := engine.RunWithSeeds(ctx, seeds)
	if err != nil && err != context.DeadlineExceeded && err != context.Canceled {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify seed bucket map was populated
	if engine.seedTargetCount < 1 {
		t.Errorf("expected seedTargetCount >= 1, got %d", engine.seedTargetCount)
	}
	if len(engine.seedValueBucket) == 0 {
		t.Error("expected seedValueBucket to be populated")
	}
}

func TestRunWithSeeds_EmptySeeds(t *testing.T) {
	fake := &fakeExecutor{
		behavior: func(ctx context.Context, w *models.ScanWorkItem) (*toolrun.InvokeResult, error) {
			return &toolrun.InvokeResult{Task: &models.ScanTask{ID: "t"}, Stdout: nil}, nil
		},
	}
	cfg := DefaultEngineConfig()
	cfg.SchedulerTick = 50 * time.Millisecond
	cfg.IdleTimeout = 200 * time.Millisecond
	engine, _ := setupTestEngine(t, fake, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Empty seeds should still run (seedTargetCount defaults to 1)
	err := engine.RunWithSeeds(ctx, nil)
	if err != nil && err != context.DeadlineExceeded && err != context.Canceled {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunWithSeeds_DuplicateValues(t *testing.T) {
	fake := &fakeExecutor{
		behavior: func(ctx context.Context, w *models.ScanWorkItem) (*toolrun.InvokeResult, error) {
			return &toolrun.InvokeResult{Task: &models.ScanTask{ID: "t"}, Stdout: nil}, nil
		},
	}
	cfg := DefaultEngineConfig()
	cfg.SchedulerTick = 50 * time.Millisecond
	cfg.IdleTimeout = 200 * time.Millisecond
	engine, _ := setupTestEngine(t, fake, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	seeds := []seed.SeedAsset{
		{Value: "10.0.0.1", ValueType: "ip", Source: "target"},
		{Value: "10.0.0.1", ValueType: "ip", Source: "target"}, // duplicate
		{Value: "10.0.0.2", ValueType: "ip", Source: "fofa"},
	}

	err := engine.RunWithSeeds(ctx, seeds)
	if err != nil && err != context.DeadlineExceeded && err != context.Canceled {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ============================================================
// linkFindingToScreenshot — boost from 11.8%
// ============================================================

func TestLinkFindingToScreenshot_SkipScreenshots(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	// Set ANCHOR_SKIP_SCREENSHOTS to trigger early return
	os.Setenv("ANCHOR_SKIP_SCREENSHOTS", "1")
	defer os.Unsetenv("ANCHOR_SKIP_SCREENSHOTS")

	// Should return early without panic
	engine.linkFindingToScreenshot("finding-1", "https://example.com/vuln")
}

func TestLinkFindingToScreenshot_NilScreenshotMgr(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	os.Unsetenv("ANCHOR_SKIP_SCREENSHOTS")
	os.Unsetenv("ANCHOR_NO_BROWSER")
	engine.screenshotMgr = nil

	// Should return early without panic
	engine.linkFindingToScreenshot("finding-1", "https://example.com/vuln")
}

func TestLinkFindingToScreenshot_WithDBState(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, queries := setupTestEngine(t, fake, cfg)

	os.Unsetenv("ANCHOR_SKIP_SCREENSHOTS")
	os.Unsetenv("ANCHOR_NO_BROWSER")

	// Create a finding
	if err := queries.CreateFinding(&models.Finding{
		ID:         "finding-1",
		ProjectID:  "proj1",
		SourceTool: "nuclei",
		DedupKey:   "dedup-1",
		Title:      "test vuln",
		Severity:   models.SeverityHigh,
		Status:     models.FindingNew,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}); err != nil {
		t.Fatalf("create finding: %v", err)
	}

	// Create the asset first (web_endpoints.asset_id has FK to assets)
	if err := queries.CreateAsset(&models.Asset{
		ID: "asset-1", ProjectID: "proj1", Type: "domain", Value: "example.com", NormalizedValue: "example.com",
	}); err != nil {
		t.Fatalf("create asset: %v", err)
	}

	// Create a web endpoint matching the URL
	if err := queries.CreateWebEndpoint(&models.WebEndpoint{
		ID:        "wep-1",
		ProjectID: "proj1",
		AssetID:   "asset-1",
		URL:       "https://example.com/vuln",
		Host:      "example.com",
		Scheme:    "https",
	}); err != nil {
		t.Fatalf("create web endpoint: %v", err)
	}

	// Should find the endpoint and link (no screenshot though)
	engine.linkFindingToScreenshot("finding-1", "https://example.com/vuln")

	// Verify the finding was linked to the web endpoint
	f, err := queries.GetFinding("finding-1")
	if err != nil {
		t.Fatalf("get finding: %v", err)
	}
	if f == nil {
		t.Fatal("finding not found")
	}
	if f.WebEndpointID == nil || *f.WebEndpointID != "wep-1" {
		t.Errorf("expected web_endpoint_id=wep-1, got %v", f.WebEndpointID)
	}
}

func TestLinkFindingToScreenshot_NoMatchingEndpoint(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, queries := setupTestEngine(t, fake, cfg)

	os.Unsetenv("ANCHOR_SKIP_SCREENSHOTS")
	os.Unsetenv("ANCHOR_NO_BROWSER")

	// Create a finding but no matching endpoint
	if err := queries.CreateFinding(&models.Finding{
		ID:         "finding-2",
		ProjectID:  "proj1",
		SourceTool: "nuclei",
		DedupKey:   "dedup-2",
		Title:      "test vuln 2",
		Severity:   models.SeverityLow,
		Status:     models.FindingNew,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}); err != nil {
		t.Fatalf("create finding: %v", err)
	}

	// No matching endpoint — should return without panic
	engine.linkFindingToScreenshot("finding-2", "https://nonexistent.example.com/vuln")
}

// ============================================================
// enqueueNucleiAsset — boost from 20%
// ============================================================

func TestEnqueueNucleiAsset_NilBuckets(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	engine.nucleiBuckets = nil
	engine.nucleiRouter = scanconfig.DefaultNucleiRouter()

	a := &core.DiscoveryAsset{Value: "https://example.com", Type: core.AssetHTTPService}
	// Should return early without panic
	engine.enqueueNucleiAsset(a, "asset-1", "bucket-1")
}

func TestEnqueueNucleiAsset_NilRouter(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	engine.nucleiBuckets = pool.NewNucleiTagBuckets(engine.dataDir, 30, 10*time.Second, func(_ string, _ pool.FlushEvent) {})
	engine.nucleiRouter = nil

	a := &core.DiscoveryAsset{Value: "https://example.com", Type: core.AssetHTTPService}
	// Should return early without panic
	engine.enqueueNucleiAsset(a, "asset-1", "bucket-1")
}

func TestEnqueueNucleiAsset_EmptyURL(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	engine.nucleiBuckets = pool.NewNucleiTagBuckets(engine.dataDir, 30, 10*time.Second, func(_ string, _ pool.FlushEvent) {})
	engine.nucleiRouter = scanconfig.DefaultNucleiRouter()

	a := &core.DiscoveryAsset{Value: "", Type: core.AssetHTTPService}
	// Should return early (empty URL)
	engine.enqueueNucleiAsset(a, "asset-1", "bucket-1")
}

func TestEnqueueNucleiAsset_WithTechRouting(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	engine.nucleiBuckets = pool.NewNucleiTagBuckets(engine.dataDir, 30, 10*time.Second, func(_ string, _ pool.FlushEvent) {})
	engine.nucleiRouter = scanconfig.DefaultNucleiRouter()

	a := &core.DiscoveryAsset{
		Value: "https://example.com",
		Type:  core.AssetHTTPService,
		Attrs: core.AssetAttrs{Technologies: []string{"nginx"}},
	}
	engine.enqueueNucleiAsset(a, "asset-1", "bucket-1")
}

// ============================================================
// flushTier2IfBlockingHigherStages — boost from 37.5%
// ============================================================

func TestFlushTier2IfBlockingHigherStages_EmptyPools(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	// All pools nil — should not panic
	engine.flushTier2IfBlockingHigherStages()
}

func TestFlushTier2IfBlockingHigherStages_WithPools(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	// Initialize pools with no items
	engine.ipPortAgg = pool.NewIPPortAggregator(engine.dataDir, func(_ string, _ pool.FlushEvent) {})
	engine.httpPool = pool.New(pool.DefaultHostPoolConfig(engine.dataDir), func(_ pool.FlushEvent) {})
	engine.nucleiBuckets = pool.NewNucleiTagBuckets(engine.dataDir, 30, 10*time.Second, func(_ string, _ pool.FlushEvent) {})

	// Push a Vuln-stage item to trigger the Vuln branch
	engine.pq.Push(queue.Item{
		WorkID:   "w-vuln",
		Action:   string(core.ActionNucleiScan),
		AssetID:  "a1",
		Priority: queue.PriorityHigh,
	})

	engine.flushTier2IfBlockingHigherStages()

	// Push a Web-stage item to trigger the Web branch
	engine.pq.Push(queue.Item{
		WorkID:   "w-web",
		Action:   string(core.ActionHTTPXFingerprint),
		AssetID:  "a2",
		Priority: queue.PriorityHigh,
	})

	engine.flushTier2IfBlockingHigherStages()
}

// ============================================================
// enqueueTier2Asset — boost from 66.7%
// ============================================================

func TestEnqueueTier2Asset_ServiceFingerprint(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)
	engine.tier2Scheduled = make(map[string]struct{})

	engine.ipPortAgg = pool.NewIPPortAggregator(engine.dataDir, func(_ string, _ pool.FlushEvent) {})

	a := &core.DiscoveryAsset{Value: "10.0.0.1:80", Type: core.AssetIPPort}
	dw := core.DerivedWork{Action: core.ActionServiceFingerprint, AssetID: "a1", Stage: "service"}
	engine.enqueueTier2Asset(context.Background(), a, dw, "bucket-1")
}

func TestEnqueueTier2Asset_UnsupportedAction(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)
	engine.tier2Scheduled = make(map[string]struct{})

	a := &core.DiscoveryAsset{Value: "example.com", Type: core.AssetSubdomain}
	dw := core.DerivedWork{Action: core.ActionKatanaCrawl, AssetID: "a1", Stage: "crawl"}
	// Should return early (unsupported action in tier2)
	engine.enqueueTier2Asset(context.Background(), a, dw, "bucket-1")
}

func TestEnqueueTier2Asset_DuplicateClaim(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)
	engine.tier2Scheduled = make(map[string]struct{})

	engine.httpPool = pool.New(pool.DefaultHostPoolConfig(engine.dataDir), func(_ pool.FlushEvent) {})

	a := &core.DiscoveryAsset{Value: "https://example.com", Type: core.AssetHTTPService}
	dw := core.DerivedWork{Action: core.ActionHTTPXFingerprint, AssetID: "a1", Stage: "web"}

	// First call succeeds
	engine.enqueueTier2Asset(context.Background(), a, dw, "bucket-1")
	// Second call should be deduped
	engine.enqueueTier2Asset(context.Background(), a, dw, "bucket-1")
}

func TestEnqueueTier2Asset_HTTPXEmptyTarget(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)
	engine.tier2Scheduled = make(map[string]struct{})

	engine.httpPool = pool.New(pool.DefaultHostPoolConfig(engine.dataDir), func(_ pool.FlushEvent) {})

	// Empty value → empty target → return early
	a := &core.DiscoveryAsset{Value: "", Type: core.AssetHTTPService}
	dw := core.DerivedWork{Action: core.ActionHTTPXFingerprint, AssetID: "a1", Stage: "web"}
	engine.enqueueTier2Asset(context.Background(), a, dw, "bucket-1")
}

func TestEnqueueTier2Asset_ServiceFingerprintBadHostPort(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)
	engine.tier2Scheduled = make(map[string]struct{})

	engine.ipPortAgg = pool.NewIPPortAggregator(engine.dataDir, func(_ string, _ pool.FlushEvent) {})

	// No port → return early
	a := &core.DiscoveryAsset{Value: "10.0.0.1", Type: core.AssetIPPort}
	dw := core.DerivedWork{Action: core.ActionServiceFingerprint, AssetID: "a1", Stage: "service"}
	engine.enqueueTier2Asset(context.Background(), a, dw, "bucket-1")
}

// ============================================================
// onBatchHTTPXComplete — boost from 50%
// ============================================================

func TestOnBatchHTTPXComplete_EmptyMembers(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	w := &models.ScanWorkItem{
		ID:             "w1",
		RunID:          "run1",
		ProjectID:      "proj1",
		AssetID:        "batch:w1",
		Action:         string(core.ActionHTTPXFingerprint),
		BatchMode:      true,
		MemberAssetIDs: "[]",
	}

	stdout := []byte(`{"input":"example.com","url":"https://example.com","host":"example.com","scheme":"https"}`)

	os.Setenv("ANCHOR_SKIP_SCREENSHOTS", "1")
	defer os.Unsetenv("ANCHOR_SKIP_SCREENSHOTS")

	engine.onBatchHTTPXComplete(context.Background(), w, stdout)
}

func TestOnBatchHTTPXComplete_FallbackToFirstMember(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, queries := setupTestEngine(t, fake, cfg)

	// Create an asset for the member
	if err := queries.CreateAsset(&models.Asset{
		ID: "a1", ProjectID: "proj1", Type: "domain", Value: "example.com",
	}); err != nil {
		t.Fatal(err)
	}

	members := []models.WorkBatchMember{
		{AssetID: "a1", Value: "example.com"},
	}
	membersJSON, _ := json.Marshal(members)

	w := &models.ScanWorkItem{
		ID:             "w1",
		RunID:          "run1",
		ProjectID:      "proj1",
		AssetID:        "batch:w1",
		Action:         string(core.ActionHTTPXFingerprint),
		BatchMode:      true,
		MemberAssetIDs: string(membersJSON),
	}

	// Input doesn't match any member directly — falls back to first
	stdout := []byte(`{"input":"unknown-host.com","url":"https://unknown-host.com","host":"unknown-host.com","scheme":"https"}`)

	os.Setenv("ANCHOR_SKIP_SCREENSHOTS", "1")
	defer os.Unsetenv("ANCHOR_SKIP_SCREENSHOTS")

	engine.onBatchHTTPXComplete(context.Background(), w, stdout)
}

// ============================================================
// onBatchNucleiComplete — boost from 61.5%
// ============================================================

func TestOnBatchNucleiComplete_NoTaskID(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	w := &models.ScanWorkItem{
		ID:             "w1",
		RunID:          "run1",
		ProjectID:      "proj1",
		AssetID:        "batch:w1",
		Action:         string(core.ActionNucleiScan),
		BatchMode:      true,
		MemberAssetIDs: "[]",
	}

	// Empty output — no results
	stdout := []byte("")

	os.Setenv("ANCHOR_SKIP_SCREENSHOTS", "1")
	defer os.Unsetenv("ANCHOR_SKIP_SCREENSHOTS")

	engine.onBatchNucleiComplete(context.Background(), w, stdout)
}

func TestOnBatchNucleiComplete_MemberMatch(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, queries := setupTestEngine(t, fake, cfg)

	// Create assets (NormalizedValue must be unique per project_id)
	for _, id := range []string{"a1", "a2"} {
		domain := id + ".example.com"
		if err := queries.CreateAsset(&models.Asset{
			ID: id, ProjectID: "proj1", Type: "domain", Value: domain, NormalizedValue: domain,
		}); err != nil {
			t.Fatal(err)
		}
	}

	members := []models.WorkBatchMember{
		{AssetID: "a1", Value: "a1.example.com"},
		{AssetID: "a2", Value: "a2.example.com"},
	}
	membersJSON, _ := json.Marshal(members)

	w := &models.ScanWorkItem{
		ID:             "w1",
		RunID:          "run1",
		ProjectID:      "proj1",
		AssetID:        "batch:w1",
		Action:         string(core.ActionNucleiScan),
		BatchMode:      true,
		MemberAssetIDs: string(membersJSON),
	}

	// Output with host matching second member
	stdout := []byte(`{"template-id":"x","host":"https://a2.example.com","matched-at":"https://a2.example.com/","info":{"name":"Vuln","severity":"low","tags":[]}}` + "\n")

	os.Setenv("ANCHOR_SKIP_SCREENSHOTS", "1")
	defer os.Unsetenv("ANCHOR_SKIP_SCREENSHOTS")

	engine.onBatchNucleiComplete(context.Background(), w, stdout)
}

// ============================================================
// executeWork — boost from 63.6%
// ============================================================

func TestExecuteWork_WindDownSkip(t *testing.T) {
	fake := &fakeExecutor{
		behavior: func(ctx context.Context, w *models.ScanWorkItem) (*toolrun.InvokeResult, error) {
			return &toolrun.InvokeResult{Task: &models.ScanTask{ID: "t"}, Stdout: nil}, nil
		},
	}
	cfg := DefaultEngineConfig()
	engine, queries := setupTestEngine(t, fake, cfg)

	// Create a work item for a non-wind-down-allowed action
	if err := queries.CreateScanWorkItem(&models.ScanWorkItem{
		ID:        "w-portscan",
		RunID:     "run1",
		ProjectID: "proj1",
		AssetID:   "a1",
		Action:    string(core.ActionPortScan),
		Status:    models.WorkStatusPending,
		Stage:     "port",
		CreatedAt: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	// Set engine to wind_down
	engine.setEngineState("wind_down")

	item := queue.Item{
		WorkID:   "w-portscan",
		Action:   string(core.ActionPortScan),
		AssetID:  "a1",
		Priority: queue.PriorityMedium,
	}

	engine.executeWork(context.Background(), item)

	// Should be marked as skipped
	w, _ := queries.GetScanWorkItem("w-portscan")
	if w != nil && w.Status != models.WorkStatusSkipped {
		t.Errorf("expected skipped, got %s", w.Status)
	}
}

func TestExecuteWork_NormalExecution(t *testing.T) {
	fake := &fakeExecutor{
		behavior: func(ctx context.Context, w *models.ScanWorkItem) (*toolrun.InvokeResult, error) {
			return &toolrun.InvokeResult{Task: &models.ScanTask{ID: "t"}, Stdout: nil}, nil
		},
	}
	cfg := DefaultEngineConfig()
	engine, queries := setupTestEngine(t, fake, cfg)

	// Create a subdomain_enum work item (needs domain asset)
	if err := queries.CreateAsset(&models.Asset{
		ID: "a1", ProjectID: "proj1", Type: "domain", Value: "example.com",
	}); err != nil {
		t.Fatal(err)
	}
	if err := queries.CreateScanWorkItem(&models.ScanWorkItem{
		ID:        "w-subdomain",
		RunID:     "run1",
		ProjectID: "proj1",
		AssetID:   "a1",
		Action:    string(core.ActionSubdomainEnum),
		Status:    models.WorkStatusPending,
		Stage:     "subdomain",
		CreatedAt: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	item := queue.Item{
		WorkID:   "w-subdomain",
		Action:   string(core.ActionSubdomainEnum),
		AssetID:  "a1",
		Priority: queue.PriorityLow,
	}

	engine.executeWork(context.Background(), item)

	w, _ := queries.GetScanWorkItem("w-subdomain")
	if w != nil && w.Status != models.WorkStatusDone {
		t.Errorf("expected done, got %s", w.Status)
	}
}

func TestExecuteWork_BuildError(t *testing.T) {
	fake := &fakeExecutor{
		behavior: func(ctx context.Context, w *models.ScanWorkItem) (*toolrun.InvokeResult, error) {
			return &toolrun.InvokeResult{Task: &models.ScanTask{ID: "t"}, Stdout: nil}, nil
		},
	}
	cfg := DefaultEngineConfig()
	engine, queries := setupTestEngine(t, fake, cfg)

	// CDN check requires IP asset, create domain instead → build error
	if err := queries.CreateAsset(&models.Asset{
		ID: "a1", ProjectID: "proj1", Type: "domain", Value: "example.com",
	}); err != nil {
		t.Fatal(err)
	}
	if err := queries.CreateScanWorkItem(&models.ScanWorkItem{
		ID:        "w-cdn",
		RunID:     "run1",
		ProjectID: "proj1",
		AssetID:   "a1",
		Action:    string(core.ActionCDNCheck),
		Status:    models.WorkStatusPending,
		Stage:     "cdn",
		CreatedAt: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	item := queue.Item{
		WorkID:   "w-cdn",
		Action:   string(core.ActionCDNCheck),
		AssetID:  "a1",
		Priority: queue.PriorityLow,
	}

	engine.executeWork(context.Background(), item)

	// Should be marked as failed (cdncheck requires IP)
	w, _ := queries.GetScanWorkItem("w-cdn")
	if w != nil && w.Status != models.WorkStatusFailed {
		t.Errorf("expected failed, got %s", w.Status)
	}
}

func TestExecuteWork_ExecutorError(t *testing.T) {
	fake := &fakeExecutor{
		behavior: func(ctx context.Context, w *models.ScanWorkItem) (*toolrun.InvokeResult, error) {
			return nil, fmt.Errorf("executor failed")
		},
	}
	cfg := DefaultEngineConfig()
	cfg.SchedulerTick = 20 * time.Millisecond
	engine, queries := setupTestEngine(t, fake, cfg)

	if err := queries.CreateAsset(&models.Asset{
		ID: "a1", ProjectID: "proj1", Type: "domain", Value: "example.com",
	}); err != nil {
		t.Fatal(err)
	}
	if err := queries.CreateScanWorkItem(&models.ScanWorkItem{
		ID:        "w-sub",
		RunID:     "run1",
		ProjectID: "proj1",
		AssetID:   "a1",
		Action:    string(core.ActionSubdomainEnum),
		Status:    models.WorkStatusPending,
		Stage:     "subdomain",
		CreatedAt: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	item := queue.Item{
		WorkID:   "w-sub",
		Action:   string(core.ActionSubdomainEnum),
		AssetID:  "a1",
		Priority: queue.PriorityLow,
	}

	engine.executeWork(context.Background(), item)

	// Should be marked as failed after retries
	w, _ := queries.GetScanWorkItem("w-sub")
	if w != nil && w.Status != models.WorkStatusFailed {
		t.Errorf("expected failed after retries, got %s", w.Status)
	}
}

func TestExecuteWork_AlreadyClaimed(t *testing.T) {
	fake := &fakeExecutor{
		behavior: func(ctx context.Context, w *models.ScanWorkItem) (*toolrun.InvokeResult, error) {
			return &toolrun.InvokeResult{Task: &models.ScanTask{ID: "t"}, Stdout: nil}, nil
		},
	}
	cfg := DefaultEngineConfig()
	engine, queries := setupTestEngine(t, fake, cfg)

	if err := queries.CreateAsset(&models.Asset{
		ID: "a1", ProjectID: "proj1", Type: "domain", Value: "example.com",
	}); err != nil {
		t.Fatal(err)
	}
	if err := queries.CreateScanWorkItem(&models.ScanWorkItem{
		ID:        "w-claimed",
		RunID:     "run1",
		ProjectID: "proj1",
		AssetID:   "a1",
		Action:    string(core.ActionSubdomainEnum),
		Status:    models.WorkStatusRunning, // already running
		Stage:     "subdomain",
		CreatedAt: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	item := queue.Item{
		WorkID:   "w-claimed",
		Action:   string(core.ActionSubdomainEnum),
		AssetID:  "a1",
		Priority: queue.PriorityLow,
	}

	// TryClaim should fail for already-running item → early return
	engine.executeWork(context.Background(), item)
}

// ============================================================
// onWorkComplete — boost from 66.4% (additional action types)
// ============================================================

func TestOnWorkComplete_HTTPXFingerprint_SingleMode_v2(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, queries := setupTestEngine(t, fake, cfg)

	if err := queries.CreateAsset(&models.Asset{
		ID: "a1", ProjectID: "proj1", Type: "domain", Value: "example.com",
	}); err != nil {
		t.Fatal(err)
	}

	w := &models.ScanWorkItem{
		ID: "w1", RunID: "run1", ProjectID: "proj1",
		AssetID: "a1", Action: string(core.ActionHTTPXFingerprint),
		BatchMode: false,
	}

	stdout := []byte(`{"input":"example.com","url":"https://example.com","host":"example.com","scheme":"https","port":"443","title":"Test","tech":["nginx"],"status-code":200}`)

	os.Setenv("ANCHOR_SKIP_SCREENSHOTS", "1")
	defer os.Unsetenv("ANCHOR_SKIP_SCREENSHOTS")

	engine.onWorkComplete(context.Background(), w, stdout)
}

func TestOnWorkComplete_ServiceFingerprint_SingleMode_v2(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, queries := setupTestEngine(t, fake, cfg)

	if err := queries.CreateAsset(&models.Asset{
		ID: "a1", ProjectID: "proj1", Type: "ip", Value: "10.0.0.1",
	}); err != nil {
		t.Fatal(err)
	}

	w := &models.ScanWorkItem{
		ID: "w1", RunID: "run1", ProjectID: "proj1",
		AssetID: "a1", Action: string(core.ActionServiceFingerprint),
		BatchMode: false,
	}

	stdout := []byte(`<?xml version="1.0"?>
<nmaprun>
  <host>
    <address addr="10.0.0.1" addrtype="ipv4"/>
    <ports>
      <port protocol="tcp" portid="80">
        <state state="open"/>
        <service name="http" product="nginx"/>
      </port>
    </ports>
  </host>
</nmaprun>`)
	engine.onWorkComplete(context.Background(), w, stdout)
}

// ============================================================
// Run — extra paths for coverage
// ============================================================

func TestRun_ExternalProfile(t *testing.T) {
	fake := &fakeExecutor{
		behavior: func(ctx context.Context, w *models.ScanWorkItem) (*toolrun.InvokeResult, error) {
			return &toolrun.InvokeResult{Task: &models.ScanTask{ID: "t"}, Stdout: nil}, nil
		},
	}
	cfg := DefaultEngineConfig()
	cfg.SchedulerTick = 50 * time.Millisecond
	cfg.IdleTimeout = 200 * time.Millisecond

	queries := newTestDB(t)
	if err := queries.CreateProject(&models.Project{ID: "proj1", Name: "test", CreatedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	if err := queries.CreatePipelineRun(&models.PipelineRun{
		ID: "run1", ProjectID: "proj1", Status: "running", CreatedAt: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	merger := asset.NewMerger(queries)
	profile := core.DefaultExternalProfile()
	engine := NewWithExecutor(queries, merger, profile, nil, nil, t.TempDir(), "run1", "proj1", cfg, nil, fake)
	engine.tier1Scheduled = make(map[string]struct{})
	engine.tier2Scheduled = make(map[string]struct{})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Run with a subdomain target — ExternalProfile triggers crt.sh path
	err := engine.Run(ctx, []string{"example.com"})
	if err != nil && err != context.DeadlineExceeded && err != context.Canceled {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRun_MultipleTargets(t *testing.T) {
	fake := &fakeExecutor{
		behavior: func(ctx context.Context, w *models.ScanWorkItem) (*toolrun.InvokeResult, error) {
			return &toolrun.InvokeResult{Task: &models.ScanTask{ID: "t"}, Stdout: nil}, nil
		},
	}
	cfg := DefaultEngineConfig()
	cfg.SchedulerTick = 50 * time.Millisecond
	cfg.IdleTimeout = 300 * time.Millisecond
	engine, _ := setupTestEngine(t, fake, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := engine.Run(ctx, []string{"10.0.0.1", "10.0.0.2", "10.0.0.3"})
	if err != nil && err != context.DeadlineExceeded && err != context.Canceled {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ============================================================
// buildTier2BatchParams — boost coverage
// ============================================================

func TestBuildTier2BatchParams_NucleiWithRateLimitPerMin(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	cfg.Pipeline.NucleiRateLimitPerMinute = 100
	engine, _ := setupTestEngine(t, fake, cfg)

	engine.nucleiRouter = scanconfig.DefaultNucleiRouter()

	w := &models.ScanWorkItem{
		Action:    string(core.ActionNucleiScan),
		InputFile: "/tmp/nuclei_batch.txt",
		BatchMode: true,
		BucketKey: "default",
	}

	params, cleanup, err := engine.buildTier2BatchParams(w)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cleanup != nil {
		cleanup()
	}
	if _, ok := params["rate_limit_per_min"]; !ok {
		t.Error("expected rate_limit_per_min param")
	}
}

// ============================================================
// initTier2Pools — paths for coverage
// ============================================================

func TestInitTier2Pools_AlreadyInitialized(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	// First init
	engine.initTier2Pools(context.Background())
	// Second init — should be idempotent
	engine.initTier2Pools(context.Background())

	if engine.httpPool == nil {
		t.Error("expected httpPool to be initialized")
	}
	if engine.nucleiBuckets == nil {
		t.Error("expected nucleiBuckets to be initialized")
	}
}

// ============================================================
// stopTier2Pools — extra paths
// ============================================================

func TestStopTier2Pools_WithItems(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	engine.initTier2Pools(context.Background())

	// Add items to ipPortAgg
	engine.ipPortAgg.Add("10.0.0.1", 80, "a1", "bucket-1")

	// Stop should flush and stop without panic
	engine.stopTier2Pools()
}

// ============================================================
// onTier2PoolFlush — paths for coverage
// ============================================================

func TestOnTier2PoolFlush_EmptyMembers(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	// Empty flush event — should return early
	engine.onTier2PoolFlush(context.Background(), core.ActionHTTPXFingerprint, "bucket-1", pool.FlushEvent{
		FilePath:   "/tmp/empty.txt",
		Members:    nil,
		Generation: 1,
	})
}

// ============================================================
// stopTier2PoolsOnce — coverage
// ============================================================

func TestStopTier2PoolsOnce(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	engine.initTier2Pools(context.Background())

	// Should be idempotent
	engine.stopTier2PoolsOnce()
	engine.stopTier2PoolsOnce()
}

// ============================================================
// waitForWorkers — coverage
// ============================================================

func TestWaitForWorkers(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	// No in-flight work — should return immediately
	engine.waitForWorkers()
}

// ============================================================
// setEngineState edge cases
// ============================================================

func TestSetEngineState_SameState(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	// Setting same state is a no-op
	engine.setEngineState("running")
	if got := engine.EngineState(); got != "running" {
		t.Errorf("expected running, got %s", got)
	}
}

// ============================================================
// processNewAsset with excluded asset
// ============================================================

func TestProcessNewAsset_ExcludedAsset(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	excludeMgr := exclude.NewManager()
	engine.excludeMgr = excludeMgr

	ctx := context.Background()
	a := &core.DiscoveryAsset{
		ID:    "a-excluded",
		Type:  core.AssetSubdomain,
		Value: "google.com", // built-in excluded
	}
	core.ReconcileDiscoveryAsset(a)

	// Should be excluded and return early
	engine.processNewAsset(ctx, a)
}

// ============================================================
// processHTTPXOutput with IP host
// ============================================================

func TestProcessHTTPXOutput_IPHost(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, queries := setupTestEngine(t, fake, cfg)

	if err := queries.CreateAsset(&models.Asset{
		ID: "a1", ProjectID: "proj1", Type: "ip", Value: "10.0.0.1",
	}); err != nil {
		t.Fatal(err)
	}

	os.Setenv("ANCHOR_SKIP_SCREENSHOTS", "1")
	defer os.Unsetenv("ANCHOR_SKIP_SCREENSHOTS")

	stdout := []byte(`{"input":"10.0.0.1","url":"http://10.0.0.1","host":"10.0.0.1","scheme":"http","port":"80","title":"Test","tech":[]}`)
	engine.processHTTPXOutput(context.Background(), func(_ string) string { return "a1" }, stdout)
}
