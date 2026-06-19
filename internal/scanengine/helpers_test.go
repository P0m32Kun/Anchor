package scanengine

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/scanengine/core"
	"github.com/P0m32Kun/Anchor/internal/scanengine/scheduler"
)

// ============================================================
// GROUP 1: Pure helper functions (no DB needed)
// ============================================================

func TestIsWindDownAllowed(t *testing.T) {
	tests := []struct {
		action string
		want   bool
	}{
		{string(core.ActionNucleiScan), true},
		{string(core.ActionHTTPXFingerprint), true},
		{string(core.ActionPortScan), false},
		{string(core.ActionDNSResolve), false},
		{string(core.ActionCDNCheck), false},
		{string(core.ActionSubdomainEnum), false},
		{string(core.ActionKatanaCrawl), false},
		{string(core.ActionFFUFBrute), false},
		{string(core.ActionSpoorScan), false},
		{string(core.ActionServiceFingerprint), false},
		{"UNKNOWN_ACTION", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := isWindDownAllowed(tt.action); got != tt.want {
			t.Errorf("isWindDownAllowed(%q) = %v, want %v", tt.action, got, tt.want)
		}
	}
}

func TestAssetTypeToString(t *testing.T) {
	tests := []struct {
		input core.AssetType
		want  string
	}{
		{core.AssetSubdomain, "domain"},
		{core.AssetIP, "ip"},
		{core.AssetCIDR, "cidr"},
		{core.AssetIPPort, "ip"},
		{core.AssetHTTPService, "url"},
		{core.AssetHTTPPath, "url"},
		{core.AssetJSURL, ""},
		{"UNKNOWN", ""},
		{"", ""},
	}
	for _, tt := range tests {
		if got := assetTypeToString(tt.input); got != tt.want {
			t.Errorf("assetTypeToString(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestAssetToScopeTarget(t *testing.T) {
	tests := []struct {
		name      string
		input     *core.DiscoveryAsset
		wantType  models.TargetType
		wantValue string
		wantNil   bool
	}{
		{
			name:    "nil input",
			input:   nil,
			wantNil: true,
		},
		{
			name:      "subdomain",
			input:     &core.DiscoveryAsset{Type: core.AssetSubdomain, Value: "example.com"},
			wantType:  models.TargetTypeDomain,
			wantValue: "example.com",
		},
		{
			name:      "ip",
			input:     &core.DiscoveryAsset{Type: core.AssetIP, Value: "10.0.0.1"},
			wantType:  models.TargetTypeIP,
			wantValue: "10.0.0.1",
		},
		{
			name:      "cidr",
			input:     &core.DiscoveryAsset{Type: core.AssetCIDR, Value: "10.0.0.0/24"},
			wantType:  models.TargetTypeCIDR,
			wantValue: "10.0.0.0/24",
		},
		{
			name:      "ip_port with ip",
			input:     &core.DiscoveryAsset{Type: core.AssetIPPort, Value: "10.0.0.1:80"},
			wantType:  models.TargetTypeIP,
			wantValue: "10.0.0.1",
		},
		{
			name:      "ip_port with domain",
			input:     &core.DiscoveryAsset{Type: core.AssetIPPort, Value: "example.com:443"},
			wantType:  models.TargetTypeDomain,
			wantValue: "example.com",
		},
		{
			name:      "ip_port without port (fallback)",
			input:     &core.DiscoveryAsset{Type: core.AssetIPPort, Value: "10.0.0.1"},
			wantType:  models.TargetTypeIP,
			wantValue: "10.0.0.1",
		},
		{
			name:      "http_service",
			input:     &core.DiscoveryAsset{Type: core.AssetHTTPService, Value: "https://example.com"},
			wantType:  models.TargetTypeURL,
			wantValue: "https://example.com",
		},
		{
			name:      "http_path",
			input:     &core.DiscoveryAsset{Type: core.AssetHTTPPath, Value: "https://example.com/api"},
			wantType:  models.TargetTypeURL,
			wantValue: "https://example.com/api",
		},
		{
			name:      "unknown type defaults to domain",
			input:     &core.DiscoveryAsset{Type: core.AssetJSURL, Value: "https://cdn.example.com/app.js"},
			wantType:  models.TargetTypeDomain,
			wantValue: "https://cdn.example.com/app.js",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := assetToScopeTarget(tt.input)
			if tt.wantNil {
				if got != nil {
					t.Fatalf("expected nil, got %+v", got)
				}
				return
			}
			if got == nil {
				t.Fatal("expected non-nil, got nil")
			}
			if got.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", got.Type, tt.wantType)
			}
			if got.Value != tt.wantValue {
				t.Errorf("Value = %q, want %q", got.Value, tt.wantValue)
			}
		})
	}
}

func TestCandidateURLs(t *testing.T) {
	t.Run("plain url without query", func(t *testing.T) {
		urls := candidateURLs("https://example.com/path")
		if len(urls) < 1 {
			t.Fatal("expected at least 1 URL")
		}
		// Should contain the original
		found := false
		for _, u := range urls {
			if u == "https://example.com/path" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected original URL in results: %v", urls)
		}
	})

	t.Run("url with query string", func(t *testing.T) {
		urls := candidateURLs("https://example.com/path?q=1&b=2")
		// Should have original + stripped query + scheme+host + alt scheme
		if len(urls) < 3 {
			t.Errorf("expected at least 3 candidates, got %d: %v", len(urls), urls)
		}
	})

	t.Run("url with fragment", func(t *testing.T) {
		urls := candidateURLs("https://example.com/path#section")
		if len(urls) < 2 {
			t.Errorf("expected at least 2 candidates, got %d: %v", len(urls), urls)
		}
	})

	t.Run("url with query and fragment", func(t *testing.T) {
		urls := candidateURLs("https://example.com/path?q=1#frag")
		if len(urls) < 3 {
			t.Errorf("expected at least 3 candidates, got %d: %v", len(urls), urls)
		}
	})

	t.Run("http scheme generates https alternate", func(t *testing.T) {
		urls := candidateURLs("http://example.com")
		foundHTTPS := false
		for _, u := range urls {
			if u == "https://example.com" {
				foundHTTPS = true
			}
		}
		if !foundHTTPS {
			t.Errorf("expected https alternate for http URL: %v", urls)
		}
	})

	t.Run("https scheme generates http alternate", func(t *testing.T) {
		urls := candidateURLs("https://example.com")
		foundHTTP := false
		for _, u := range urls {
			if u == "http://example.com" {
				foundHTTP = true
			}
		}
		if !foundHTTP {
			t.Errorf("expected http alternate for https URL: %v", urls)
		}
	})

	t.Run("non-url string", func(t *testing.T) {
		urls := candidateURLs("just-a-string")
		// Should return at least the original
		if len(urls) < 1 {
			t.Fatal("expected at least 1 URL")
		}
		if urls[0] != "just-a-string" {
			t.Errorf("expected original string, got %q", urls[0])
		}
	})
}

func TestSkipScreenshots(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	// Save and restore env vars
	for _, key := range []string{"ANCHOR_SKIP_SCREENSHOTS", "ANCHOR_NO_BROWSER"} {
		old, had := os.LookupEnv(key)
		if had {
			t.Cleanup(func() { os.Setenv(key, old) })
		} else {
			t.Cleanup(func() { os.Unsetenv(key) })
		}
	}

	t.Run("no env vars", func(t *testing.T) {
		os.Unsetenv("ANCHOR_SKIP_SCREENSHOTS")
		os.Unsetenv("ANCHOR_NO_BROWSER")
		if engine.skipScreenshots() {
			t.Error("expected false when no env vars set")
		}
	})

	t.Run("ANCHOR_SKIP_SCREENSHOTS=1", func(t *testing.T) {
		os.Setenv("ANCHOR_SKIP_SCREENSHOTS", "1")
		os.Unsetenv("ANCHOR_NO_BROWSER")
		if !engine.skipScreenshots() {
			t.Error("expected true when ANCHOR_SKIP_SCREENSHOTS=1")
		}
		os.Unsetenv("ANCHOR_SKIP_SCREENSHOTS")
	})

	t.Run("ANCHOR_NO_BROWSER=1", func(t *testing.T) {
		os.Unsetenv("ANCHOR_SKIP_SCREENSHOTS")
		os.Setenv("ANCHOR_NO_BROWSER", "1")
		if !engine.skipScreenshots() {
			t.Error("expected true when ANCHOR_NO_BROWSER=1")
		}
		os.Unsetenv("ANCHOR_NO_BROWSER")
	})

	t.Run("both set", func(t *testing.T) {
		os.Setenv("ANCHOR_SKIP_SCREENSHOTS", "1")
		os.Setenv("ANCHOR_NO_BROWSER", "1")
		if !engine.skipScreenshots() {
			t.Error("expected true when both env vars set")
		}
		os.Unsetenv("ANCHOR_SKIP_SCREENSHOTS")
		os.Unsetenv("ANCHOR_NO_BROWSER")
	})
}

// ============================================================
// GROUP 2: Engine method helpers (need ScanEngine instance)
// ============================================================

func TestSetOnNewAsset(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	if engine.onNewAsset != nil {
		t.Fatal("expected onNewAsset to be nil initially")
	}

	called := false
	engine.SetOnNewAsset(func(assetID, value, assetType string) {
		called = true
	})

	if engine.onNewAsset == nil {
		t.Fatal("expected onNewAsset to be set")
	}
	// Call it to verify it works
	engine.onNewAsset("id", "val", "type")
	if !called {
		t.Error("callback was not invoked")
	}
}

func TestCancel(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	// Before Run, cancel should be nil (no panic)
	engine.Cancel()

	// After setting cancel via a derived context
	ctx, cancel := context.WithCancel(context.Background())
	engine.cancel = cancel
	engine.Cancel()

	<-ctx.Done() // should be done
}

func TestGlobalConcurrencyLimit(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	limits := scheduler.ComputeLimits(1, 0)

	t.Run("SchedulerConcurrency overrides", func(t *testing.T) {
		engine.config.SchedulerConcurrency = 42
		engine.config.BatchSize = 0
		if got := engine.globalConcurrencyLimit(limits); got != 42 {
			t.Errorf("got %d, want 42", got)
		}
	})

	t.Run("BatchSize fallback", func(t *testing.T) {
		engine.config.SchedulerConcurrency = 0
		engine.config.BatchSize = 10
		if got := engine.globalConcurrencyLimit(limits); got != 10 {
			t.Errorf("got %d, want 10", got)
		}
	})

	t.Run("limits.GlobalMax fallback", func(t *testing.T) {
		engine.config.SchedulerConcurrency = 0
		engine.config.BatchSize = 0
		got := engine.globalConcurrencyLimit(limits)
		if got != limits.GlobalMax {
			t.Errorf("got %d, want %d", got, limits.GlobalMax)
		}
	})
}

func TestResolveBucketKey(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	// Setup seed bucket
	engine.seedValueBucket = map[string]string{
		"example.com": "seed:example.com",
	}

	t.Run("nil asset", func(t *testing.T) {
		if got := engine.resolveBucketKey(nil); got != "asset:unknown" {
			t.Errorf("got %q, want %q", got, "asset:unknown")
		}
	})

	t.Run("seed asset with known value", func(t *testing.T) {
		a := &core.DiscoveryAsset{SourceTool: "seed", Value: "example.com"}
		if got := engine.resolveBucketKey(a); got != "seed:example.com" {
			t.Errorf("got %q, want %q", got, "seed:example.com")
		}
	})

	t.Run("seed asset with unknown value", func(t *testing.T) {
		a := &core.DiscoveryAsset{SourceTool: "seed", Value: "unknown.com"}
		if got := engine.resolveBucketKey(a); got != "seed:unknown.com" {
			t.Errorf("got %q, want %q", got, "seed:unknown.com")
		}
	})

	t.Run("seed asset with empty value", func(t *testing.T) {
		a := &core.DiscoveryAsset{SourceTool: "seed", Value: ""}
		// Empty value → falls through to ID check
		got := engine.resolveBucketKey(a)
		if got == "" {
			t.Error("expected non-empty bucket key")
		}
	})

	t.Run("child asset with parent bucket", func(t *testing.T) {
		engine.assetBuckets.Store("parent1", "seed:example.com")
		a := &core.DiscoveryAsset{ParentID: "parent1", SourceTool: "subfinder"}
		if got := engine.resolveBucketKey(a); got != "seed:example.com" {
			t.Errorf("got %q, want %q", got, "seed:example.com")
		}
	})

	t.Run("asset with ID fallback", func(t *testing.T) {
		a := &core.DiscoveryAsset{ID: "asset-123", SourceTool: "naabu"}
		if got := engine.resolveBucketKey(a); got != "asset:asset-123" {
			t.Errorf("got %q, want %q", got, "asset:asset-123")
		}
	})

	t.Run("unknown asset", func(t *testing.T) {
		a := &core.DiscoveryAsset{SourceTool: "unknown"}
		if got := engine.resolveBucketKey(a); got != "asset:unknown" {
			t.Errorf("got %q, want %q", got, "asset:unknown")
		}
	})
}

func TestBucketForAssetID(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	t.Run("stored value", func(t *testing.T) {
		engine.assetBuckets.Store("a1", "seed:example.com")
		if got := engine.bucketForAssetID("a1"); got != "seed:example.com" {
			t.Errorf("got %q, want %q", got, "seed:example.com")
		}
	})

	t.Run("fallback", func(t *testing.T) {
		if got := engine.bucketForAssetID("a999"); got != "asset:a999" {
			t.Errorf("got %q, want %q", got, "asset:a999")
		}
	})
}

func TestIncDecBucketInflight(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	t.Run("increment", func(t *testing.T) {
		engine.incBucketInflight("b1")
		engine.incBucketInflight("b1")
		engine.bucketInflightMu.Lock()
		if engine.bucketInflight["b1"] != 2 {
			t.Errorf("expected 2, got %d", engine.bucketInflight["b1"])
		}
		engine.bucketInflightMu.Unlock()
	})

	t.Run("decrement", func(t *testing.T) {
		engine.decBucketInflight("b1")
		engine.bucketInflightMu.Lock()
		if engine.bucketInflight["b1"] != 1 {
			t.Errorf("expected 1, got %d", engine.bucketInflight["b1"])
		}
		engine.bucketInflightMu.Unlock()
	})

	t.Run("decrement to zero removes key", func(t *testing.T) {
		engine.decBucketInflight("b1")
		engine.bucketInflightMu.Lock()
		if _, ok := engine.bucketInflight["b1"]; ok {
			t.Error("expected key to be deleted when count reaches 0")
		}
		engine.bucketInflightMu.Unlock()
	})

	t.Run("decrement below zero is no-op", func(t *testing.T) {
		engine.decBucketInflight("b-nonexistent")
		engine.bucketInflightMu.Lock()
		if _, ok := engine.bucketInflight["b-nonexistent"]; ok {
			t.Error("expected no key for nonexistent bucket")
		}
		engine.bucketInflightMu.Unlock()
	})

	t.Run("empty key defaults", func(t *testing.T) {
		engine.incBucketInflight("")
		engine.bucketInflightMu.Lock()
		if engine.bucketInflight["default"] != 1 {
			t.Errorf("expected default=1, got %d", engine.bucketInflight["default"])
		}
		engine.bucketInflightMu.Unlock()
		engine.decBucketInflight("")
	})
}

func TestIsAssetExcluded(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()

	t.Run("nil excludeMgr and nil scopeEng", func(t *testing.T) {
		engine, _ := setupTestEngine(t, fake, cfg)
		engine.excludeMgr = nil
		engine.scopeEng = nil
		a := &core.DiscoveryAsset{Value: "evil.com"}
		if engine.isAssetExcluded(a) {
			t.Error("expected false when both managers are nil")
		}
	})

	t.Run("nil excludeMgr, nil scopeEng returns false", func(t *testing.T) {
		engine, _ := setupTestEngine(t, fake, cfg)
		a := &core.DiscoveryAsset{Value: "example.com", Type: core.AssetSubdomain}
		if engine.isAssetExcluded(a) {
			t.Error("expected false")
		}
	})
}

func TestPrepareChildAsset(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	t.Run("empty parent ID", func(t *testing.T) {
		a := &core.DiscoveryAsset{ID: "child1", DiscoveryDepth: 0}
		engine.prepareChildAsset(a, "")
		if a.DiscoveryDepth != 0 {
			t.Errorf("expected depth 0, got %d", a.DiscoveryDepth)
		}
	})

	t.Run("parent with stored depth", func(t *testing.T) {
		engine.assetDepth.Store("parent1", 3)
		engine.assetBuckets.Store("parent1", "seed:example.com")
		a := &core.DiscoveryAsset{ID: "child1", DiscoveryDepth: 0}
		engine.prepareChildAsset(a, "parent1")
		if a.DiscoveryDepth != 4 {
			t.Errorf("expected depth 4, got %d", a.DiscoveryDepth)
		}
		// Verify bucket was stored for child
		v, ok := engine.assetBuckets.Load("child1")
		if !ok || v.(string) != "seed:example.com" {
			t.Errorf("expected bucket seed:example.com, got %v", v)
		}
	})

	t.Run("parent with no stored depth defaults to increment", func(t *testing.T) {
		a := &core.DiscoveryAsset{ID: "child2", DiscoveryDepth: 5}
		engine.prepareChildAsset(a, "nonexistent-parent")
		if a.DiscoveryDepth != 6 {
			t.Errorf("expected depth 6, got %d", a.DiscoveryDepth)
		}
	})
}

func TestRecordAssetRelation(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, queries := setupTestEngine(t, fake, cfg)

	// Create an asset first
	assetID := "asset-rel-1"
	if err := queries.CreateAsset(&models.Asset{
		ID:        assetID,
		ProjectID: "proj1",
		Type:      "domain",
		Value:     "example.com",
	}); err != nil {
		t.Fatalf("create asset: %v", err)
	}

	t.Run("with parent ID", func(t *testing.T) {
		a := &core.DiscoveryAsset{
			ParentID:   "parent-asset-1",
			SourceTool: "subfinder",
		}
		engine.recordAssetRelation(a, assetID)
		// Verify relation was created (no error = success)
	})

	t.Run("with explicit lineage fields", func(t *testing.T) {
		a := &core.DiscoveryAsset{
			LineageSourceType:   "target",
			LineageSourceID:     "target-1",
			LineageRelationType: "discovered_from",
			SourceTool:          "manual",
		}
		engine.recordAssetRelation(a, assetID)
	})

	t.Run("missing source fields skips", func(t *testing.T) {
		a := &core.DiscoveryAsset{SourceTool: "test"}
		// Should not panic, just skip
		engine.recordAssetRelation(a, "")
	})
}

func TestIsTier1Action(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	tests := []struct {
		action core.TaskAction
		want   bool
	}{
		{core.ActionSubdomainEnum, true},
		{core.ActionDNSResolve, true},
		{core.ActionCDNCheck, true},
		{core.ActionPortScan, true},
		{core.ActionHTTPXFingerprint, false},
		{core.ActionNucleiScan, false},
		{core.ActionServiceFingerprint, false},
		{core.ActionKatanaCrawl, false},
		{core.ActionFFUFBrute, false},
		{core.ActionSpoorScan, false},
	}
	for _, tt := range tests {
		if got := engine.isTier1Action(tt.action); got != tt.want {
			t.Errorf("isTier1Action(%q) = %v, want %v", tt.action, got, tt.want)
		}
	}
}

func TestIsTier2PooledAction(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	tests := []struct {
		action core.TaskAction
		want   bool
	}{
		{core.ActionHTTPXFingerprint, true},
		{core.ActionServiceFingerprint, true},
		{core.ActionNucleiScan, true},
		{core.ActionSubdomainEnum, false},
		{core.ActionDNSResolve, false},
		{core.ActionCDNCheck, false},
		{core.ActionPortScan, false},
		{core.ActionKatanaCrawl, false},
		{core.ActionFFUFBrute, false},
	}
	for _, tt := range tests {
		if got := engine.isTier2PooledAction(tt.action); got != tt.want {
			t.Errorf("isTier2PooledAction(%q) = %v, want %v", tt.action, got, tt.want)
		}
	}
}

func TestEngineState(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	if got := engine.EngineState(); got != "running" {
		t.Errorf("initial state = %q, want %q", got, "running")
	}
}

func TestSetEngineState(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	engine.setEngineState("wind_down")
	if got := engine.EngineState(); got != "wind_down" {
		t.Errorf("state = %q, want %q", got, "wind_down")
	}

	// Same state, no-op
	engine.setEngineState("wind_down")
	if got := engine.EngineState(); got != "wind_down" {
		t.Errorf("state = %q, want %q", got, "wind_down")
	}

	engine.setEngineState("stopped")
	if got := engine.EngineState(); got != "stopped" {
		t.Errorf("state = %q, want %q", got, "stopped")
	}
}

func TestPQIsEmptyOrOnlyDiscovery(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	// Empty queue
	if !engine.pqIsEmptyOrOnlyDiscovery() {
		t.Error("expected true for empty queue")
	}
}

func TestPQHasStageOrHigher(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	// Empty queue
	if engine.pqHasStageOrHigher(0) {
		t.Error("expected false for empty queue")
	}
}

// ============================================================
// GROUP 3: Batch helper functions
// ============================================================

func TestParseBatchMembers(t *testing.T) {
	t.Run("nil work item", func(t *testing.T) {
		if got := parseBatchMembers(nil); got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("empty MemberAssetIDs", func(t *testing.T) {
		w := &models.ScanWorkItem{ID: "w1", MemberAssetIDs: ""}
		if got := parseBatchMembers(w); got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		w := &models.ScanWorkItem{ID: "w1", MemberAssetIDs: "not-json"}
		if got := parseBatchMembers(w); got != nil {
			t.Errorf("expected nil for invalid JSON, got %v", got)
		}
	})

	t.Run("valid JSON", func(t *testing.T) {
		members := []models.WorkBatchMember{
			{AssetID: "a1", Value: "example.com"},
			{AssetID: "a2", Value: "test.com"},
		}
		data, _ := json.Marshal(members)
		w := &models.ScanWorkItem{ID: "w1", MemberAssetIDs: string(data)}
		got := parseBatchMembers(w)
		if len(got) != 2 {
			t.Fatalf("expected 2 members, got %d", len(got))
		}
		if got[0].AssetID != "a1" || got[0].Value != "example.com" {
			t.Errorf("member[0] = %+v", got[0])
		}
		if got[1].AssetID != "a2" || got[1].Value != "test.com" {
			t.Errorf("member[1] = %+v", got[1])
		}
	})
}

func TestBatchMemberByValue(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		got := batchMemberByValue(nil)
		if len(got) != 0 {
			t.Errorf("expected empty, got %d", len(got))
		}
	})

	t.Run("with entries", func(t *testing.T) {
		members := []models.WorkBatchMember{
			{AssetID: "a1", Value: "Example.com"},
			{AssetID: "a2", Value: "TEST.com"},
		}
		got := batchMemberByValue(members)
		if len(got) != 2 {
			t.Fatalf("expected 2, got %d", len(got))
		}
		if m, ok := got["example.com"]; !ok || m.AssetID != "a1" {
			t.Errorf("missing lowercase key example.com")
		}
		if m, ok := got["test.com"]; !ok || m.AssetID != "a2" {
			t.Errorf("missing lowercase key test.com")
		}
	})

	t.Run("empty value skipped", func(t *testing.T) {
		members := []models.WorkBatchMember{
			{AssetID: "a1", Value: ""},
			{AssetID: "a2", Value: "ok.com"},
		}
		got := batchMemberByValue(members)
		if len(got) != 1 {
			t.Errorf("expected 1, got %d", len(got))
		}
	})
}

func TestBatchMemberByIP(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		got := batchMemberByIP(nil)
		if len(got) != 0 {
			t.Errorf("expected empty, got %d", len(got))
		}
	})

	t.Run("with IPs", func(t *testing.T) {
		members := []models.WorkBatchMember{
			{AssetID: "a1", Value: "10.0.0.1"},
			{AssetID: "a2", Value: "10.0.0.2:80"},
		}
		got := batchMemberByIP(members)
		if len(got) != 2 {
			t.Fatalf("expected 2, got %d", len(got))
		}
		if m, ok := got["10.0.0.1"]; !ok || m.AssetID != "a1" {
			t.Errorf("missing 10.0.0.1")
		}
		if m, ok := got["10.0.0.2"]; !ok || m.AssetID != "a2" {
			t.Errorf("missing 10.0.0.2 (from 10.0.0.2:80)")
		}
	})
}

func TestBatchMemberByHostPort(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		got := batchMemberByHostPort(nil)
		if len(got) != 0 {
			t.Errorf("expected empty, got %d", len(got))
		}
	})

	t.Run("valid entries", func(t *testing.T) {
		members := []models.WorkBatchMember{
			{AssetID: "a1", Value: "10.0.0.1:80"},
			{AssetID: "a2", Value: "10.0.0.2:443"},
		}
		got := batchMemberByHostPort(members)
		if len(got) != 2 {
			t.Fatalf("expected 2, got %d", len(got))
		}
		if m, ok := got["10.0.0.1:80"]; !ok || m.AssetID != "a1" {
			t.Errorf("missing 10.0.0.1:80")
		}
	})

	t.Run("invalid entry skipped", func(t *testing.T) {
		members := []models.WorkBatchMember{
			{AssetID: "a1", Value: "no-port-here"},
		}
		got := batchMemberByHostPort(members)
		if len(got) != 0 {
			t.Errorf("expected 0 for invalid hostport, got %d", len(got))
		}
	})
}

func TestParentAssetForSubdomain(t *testing.T) {
	members := []models.WorkBatchMember{
		{AssetID: "a1", Value: "example.com"},
		{AssetID: "a2", Value: "test.org"},
	}

	t.Run("exact match", func(t *testing.T) {
		if got := parentAssetForSubdomain("example.com", members); got != "a1" {
			t.Errorf("got %q, want a1", got)
		}
	})

	t.Run("suffix match", func(t *testing.T) {
		if got := parentAssetForSubdomain("sub.example.com", members); got != "a1" {
			t.Errorf("got %q, want a1", got)
		}
	})

	t.Run("case insensitive", func(t *testing.T) {
		if got := parentAssetForSubdomain("SUB.EXAMPLE.COM", members); got != "a1" {
			t.Errorf("got %q, want a1", got)
		}
	})

	t.Run("no match falls back to first", func(t *testing.T) {
		if got := parentAssetForSubdomain("other.net", members); got != "a1" {
			t.Errorf("got %q, want a1 (fallback)", got)
		}
	})

	t.Run("empty members", func(t *testing.T) {
		if got := parentAssetForSubdomain("example.com", nil); got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})
}

func TestClaimTier1(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)
	engine.tier1Scheduled = make(map[string]struct{})

	t.Run("first claim succeeds", func(t *testing.T) {
		if !engine.claimTier1(core.ActionDNSResolve, "a1") {
			t.Error("expected first claim to succeed")
		}
	})

	t.Run("duplicate claim fails", func(t *testing.T) {
		if engine.claimTier1(core.ActionDNSResolve, "a1") {
			t.Error("expected duplicate claim to fail")
		}
	})

	t.Run("different action succeeds", func(t *testing.T) {
		if !engine.claimTier1(core.ActionPortScan, "a1") {
			t.Error("expected different action to succeed")
		}
	})

	t.Run("different asset succeeds", func(t *testing.T) {
		if !engine.claimTier1(core.ActionDNSResolve, "a2") {
			t.Error("expected different asset to succeed")
		}
	})
}

func TestClaimTier2(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)
	engine.tier2Scheduled = make(map[string]struct{})

	t.Run("first claim succeeds", func(t *testing.T) {
		if !engine.claimTier2(core.ActionHTTPXFingerprint, "a1") {
			t.Error("expected first claim to succeed")
		}
	})

	t.Run("duplicate claim fails", func(t *testing.T) {
		if engine.claimTier2(core.ActionHTTPXFingerprint, "a1") {
			t.Error("expected duplicate claim to fail")
		}
	})
}
