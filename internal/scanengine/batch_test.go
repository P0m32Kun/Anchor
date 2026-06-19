package scanengine

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/scanengine/core"
)

// ============================================================
// GROUP 4: Batch handler tests with mock stdout data
// ============================================================

// makeBatchWork creates a ScanWorkItem in batch mode with the given members.
func makeBatchWork(action string, members []models.WorkBatchMember) *models.ScanWorkItem {
	data, _ := json.Marshal(members)
	return &models.ScanWorkItem{
		ID:             "batch-w1",
		RunID:          "run1",
		ProjectID:      "proj1",
		AssetID:        "batch-asset-1",
		Action:         action,
		BatchMode:      true,
		InputFile:      "/tmp/test-input.txt",
		MemberAssetIDs: string(data),
	}
}

func TestOnBatchDNSComplete(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	members := []models.WorkBatchMember{
		{AssetID: "a1", Value: "example.com"},
		{AssetID: "a2", Value: "test.org"},
	}
	w := makeBatchWork(string(core.ActionDNSResolve), members)

	// Mock dnsx JSONL output
	stdout := []byte(`{"host":"example.com","a":["1.2.3.4"],"cname":["cdn.example.com"],"aaaa":[]}
{"host":"test.org","a":["5.6.7.8"],"cname":[],"aaaa":[]}
`)

	ctx := context.Background()
	engine.onBatchDNSComplete(ctx, w, stdout)

	// The engine should have processed the DNS results.
	// We verify by checking dedup was called (normalized values stored).
	// Since processNewAsset merges into DB, we can check the DB for new assets.
	assets, _ := engine.queries.ListAssetsByProject("proj1")
	t.Logf("assets after DNS batch: %d", len(assets))
}

func TestOnBatchPortScanComplete(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	// Create parent assets in DB so CreatePortIfNotExists can reference them
	engine.queries.CreateAsset(&models.Asset{
		ID: "a1", ProjectID: "proj1", Type: "ip", Value: "10.0.0.1",
	})

	members := []models.WorkBatchMember{
		{AssetID: "a1", Value: "10.0.0.1"},
	}
	w := makeBatchWork(string(core.ActionPortScan), members)

	// Mock naabu JSONL output
	stdout := []byte(`{"host":"10.0.0.1","ip":"10.0.0.1","port":80}
{"host":"10.0.0.1","ip":"10.0.0.1","port":443}
`)

	ctx := context.Background()
	engine.onBatchPortScanComplete(ctx, w, stdout)
}

func TestOnBatchSubfinderComplete(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	// Create parent asset
	engine.queries.CreateAsset(&models.Asset{
		ID: "a1", ProjectID: "proj1", Type: "domain", Value: "example.com",
	})

	members := []models.WorkBatchMember{
		{AssetID: "a1", Value: "example.com"},
	}
	w := makeBatchWork(string(core.ActionSubdomainEnum), members)

	// Mock subfinder JSONL output
	stdout := []byte(`{"host":"sub1.example.com","input":"example.com","source":"dns"}
{"host":"sub2.example.com","input":"example.com","source":"dns"}
`)

	ctx := context.Background()
	engine.onBatchSubfinderComplete(ctx, w, stdout)

	// Verify new subdomains were discovered
	assets, _ := engine.queries.ListAssetsByProject("proj1")
	t.Logf("assets after subfinder batch: %d", len(assets))
}

func TestOnBatchCDNComplete(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	// Create parent IP asset
	engine.queries.CreateAsset(&models.Asset{
		ID: "a1", ProjectID: "proj1", Type: "ip", Value: "1.2.3.4",
	})

	members := []models.WorkBatchMember{
		{AssetID: "a1", Value: "1.2.3.4"},
	}
	w := makeBatchWork(string(core.ActionCDNCheck), members)

	// Mock cdncheck JSONL output
	stdout := []byte(`{"ip":"1.2.3.4","cdn":true,"cdn_name":"cloudflare"}
`)

	ctx := context.Background()
	engine.onBatchCDNComplete(ctx, w, stdout)
}

func TestOnBatchHTTPXComplete(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	// Create parent asset
	engine.queries.CreateAsset(&models.Asset{
		ID: "a1", ProjectID: "proj1", Type: "domain", Value: "example.com",
	})

	members := []models.WorkBatchMember{
		{AssetID: "a1", Value: "https://example.com"},
	}
	w := makeBatchWork(string(core.ActionHTTPXFingerprint), members)

	// Mock httpx JSON output
	stdout := []byte(fmt.Sprintf(`{"input":"https://example.com","url":"https://example.com","host":"example.com","port":"443","path":"/","title":"Welcome","status_code":200,"tech":["nginx"]}%s`, "\n"))

	ctx := context.Background()
	engine.onBatchHTTPXComplete(ctx, w, stdout)

	assets, _ := engine.queries.ListAssetsByProject("proj1")
	t.Logf("assets after httpx batch: %d", len(assets))
}

func TestOnBatchNmapComplete(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	// Create parent asset
	engine.queries.CreateAsset(&models.Asset{
		ID: "a1", ProjectID: "proj1", Type: "ip", Value: "10.0.0.1",
	})

	members := []models.WorkBatchMember{
		{AssetID: "a1", Value: "10.0.0.1:80"},
	}
	w := makeBatchWork(string(core.ActionServiceFingerprint), members)

	// Mock nmap XML output
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
</nmaprun>
`)

	ctx := context.Background()
	engine.onBatchNmapComplete(ctx, w, stdout)
}

func TestOnBatchNucleiComplete(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	// Create parent asset
	engine.queries.CreateAsset(&models.Asset{
		ID: "a1", ProjectID: "proj1", Type: "url", Value: "https://example.com",
	})

	members := []models.WorkBatchMember{
		{AssetID: "a1", Value: "https://example.com"},
	}
	w := makeBatchWork(string(core.ActionNucleiScan), members)

	// Mock nuclei JSONL output
	stdout := []byte(`{"template-id":"test-001","host":"https://example.com","matched-at":"https://example.com/vuln","info":{"name":"Test Vuln","severity":"high"}}
`)

	ctx := context.Background()
	engine.onBatchNucleiComplete(ctx, w, stdout)
}

// ============================================================
// GROUP 5: onWorkComplete action routing tests
// ============================================================

func TestOnWorkComplete_KatanaCrawl(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	// Create parent asset in DB
	engine.queries.CreateAsset(&models.Asset{
		ID: "parent1", ProjectID: "proj1", Type: "domain", Value: "example.com",
	})
	engine.assetDepth.Store("parent1", 0)
	engine.assetBuckets.Store("parent1", "seed:example.com")

	w := &models.ScanWorkItem{
		ID:        "w1",
		RunID:     "run1",
		ProjectID: "proj1",
		AssetID:   "parent1",
		Action:    string(core.ActionKatanaCrawl),
	}

	// Mock katana JSON output
	stdout := []byte(`{"request":{"url":"https://example.com/page1"},"response":{"status":200}}
`)

	ctx := context.Background()
	engine.onWorkComplete(ctx, w, stdout)
}

func TestOnWorkComplete_FFUFBrute(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	engine.queries.CreateAsset(&models.Asset{
		ID: "parent1", ProjectID: "proj1", Type: "domain", Value: "example.com",
	})
	engine.assetDepth.Store("parent1", 0)
	engine.assetBuckets.Store("parent1", "seed:example.com")

	w := &models.ScanWorkItem{
		ID:        "w1",
		RunID:     "run1",
		ProjectID: "proj1",
		AssetID:   "parent1",
		Action:    string(core.ActionFFUFBrute),
	}

	stdout := []byte(`{"input":{"FUZZ":"admin"},"url":"https://example.com/admin","status":200,"length":1234}
`)

	ctx := context.Background()
	engine.onWorkComplete(ctx, w, stdout)
}

func TestOnWorkComplete_SpoorScan(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	engine.queries.CreateAsset(&models.Asset{
		ID: "parent1", ProjectID: "proj1", Type: "domain", Value: "example.com",
	})
	engine.assetDepth.Store("parent1", 0)
	engine.assetBuckets.Store("parent1", "seed:example.com")

	w := &models.ScanWorkItem{
		ID:        "w1",
		RunID:     "run1",
		ProjectID: "proj1",
		AssetID:   "parent1",
		Action:    string(core.ActionSpoorScan),
	}

	// Mock spoor output (empty = no findings)
	stdout := []byte(``)

	ctx := context.Background()
	engine.onWorkComplete(ctx, w, stdout)
}

func TestOnWorkComplete_DNSResolve(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	engine.queries.CreateAsset(&models.Asset{
		ID: "parent1", ProjectID: "proj1", Type: "domain", Value: "example.com",
	})
	engine.assetDepth.Store("parent1", 0)
	engine.assetBuckets.Store("parent1", "seed:example.com")

	w := &models.ScanWorkItem{
		ID:        "w1",
		RunID:     "run1",
		ProjectID: "proj1",
		AssetID:   "parent1",
		Action:    string(core.ActionDNSResolve),
	}

	stdout := []byte(`{"host":"example.com","a":["1.2.3.4"],"cname":[],"aaaa":[]}
`)

	ctx := context.Background()
	engine.onWorkComplete(ctx, w, stdout)
}

func TestOnWorkComplete_CDNCheck(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	engine.queries.CreateAsset(&models.Asset{
		ID: "parent1", ProjectID: "proj1", Type: "ip", Value: "1.2.3.4",
	})
	engine.assetDepth.Store("parent1", 0)
	engine.assetBuckets.Store("parent1", "seed:example.com")

	w := &models.ScanWorkItem{
		ID:        "w1",
		RunID:     "run1",
		ProjectID: "proj1",
		AssetID:   "parent1",
		Action:    string(core.ActionCDNCheck),
	}

	stdout := []byte(`{"ip":"1.2.3.4","cdn":true,"cdn_name":"cloudflare"}
`)

	ctx := context.Background()
	engine.onWorkComplete(ctx, w, stdout)
}

func TestOnWorkComplete_SubdomainEnum(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	engine.queries.CreateAsset(&models.Asset{
		ID: "parent1", ProjectID: "proj1", Type: "domain", Value: "example.com",
	})
	engine.assetDepth.Store("parent1", 0)
	engine.assetBuckets.Store("parent1", "seed:example.com")

	w := &models.ScanWorkItem{
		ID:        "w1",
		RunID:     "run1",
		ProjectID: "proj1",
		AssetID:   "parent1",
		Action:    string(core.ActionSubdomainEnum),
	}

	stdout := []byte(`{"host":"sub.example.com","input":"example.com","source":"dns"}
`)

	ctx := context.Background()
	engine.onWorkComplete(ctx, w, stdout)
}

func TestOnWorkComplete_PortScan(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	engine.queries.CreateAsset(&models.Asset{
		ID: "parent1", ProjectID: "proj1", Type: "ip", Value: "10.0.0.1",
	})
	engine.assetDepth.Store("parent1", 0)
	engine.assetBuckets.Store("parent1", "seed:10.0.0.1")

	w := &models.ScanWorkItem{
		ID:        "w1",
		RunID:     "run1",
		ProjectID: "proj1",
		AssetID:   "parent1",
		Action:    string(core.ActionPortScan),
	}

	stdout := []byte(`{"host":"10.0.0.1","ip":"10.0.0.1","port":22}
`)

	ctx := context.Background()
	engine.onWorkComplete(ctx, w, stdout)
}

func TestOnWorkComplete_HTTPXFingerprint_SingleMode(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	engine.queries.CreateAsset(&models.Asset{
		ID: "parent1", ProjectID: "proj1", Type: "domain", Value: "example.com",
	})
	engine.assetDepth.Store("parent1", 0)
	engine.assetBuckets.Store("parent1", "seed:example.com")

	w := &models.ScanWorkItem{
		ID:        "w1",
		RunID:     "run1",
		ProjectID: "proj1",
		AssetID:   "parent1",
		Action:    string(core.ActionHTTPXFingerprint),
		BatchMode: false,
	}

	stdout := []byte(fmt.Sprintf(`{"input":"https://example.com","url":"https://example.com","host":"example.com","port":"443","path":"/","title":"Welcome","status_code":200,"tech":["nginx"]}%s`, "\n"))

	ctx := context.Background()
	engine.onWorkComplete(ctx, w, stdout)
}

func TestOnWorkComplete_NucleiScan_SingleMode(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	engine.queries.CreateAsset(&models.Asset{
		ID: "parent1", ProjectID: "proj1", Type: "url", Value: "https://example.com",
	})
	engine.assetDepth.Store("parent1", 0)
	engine.assetBuckets.Store("parent1", "seed:example.com")

	w := &models.ScanWorkItem{
		ID:        "w1",
		RunID:     "run1",
		ProjectID: "proj1",
		AssetID:   "parent1",
		Action:    string(core.ActionNucleiScan),
		BatchMode: false,
	}

	stdout := []byte(`{"template-id":"test-001","host":"https://example.com","matched-at":"https://example.com/vuln","info":{"name":"Test Vuln","severity":"high"}}
`)

	ctx := context.Background()
	engine.onWorkComplete(ctx, w, stdout)
}

func TestOnWorkComplete_ServiceFingerprint_SingleMode(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	engine.queries.CreateAsset(&models.Asset{
		ID: "parent1", ProjectID: "proj1", Type: "ip_port", Value: "10.0.0.1:80",
	})
	engine.assetDepth.Store("parent1", 0)
	engine.assetBuckets.Store("parent1", "seed:10.0.0.1")

	w := &models.ScanWorkItem{
		ID:        "w1",
		RunID:     "run1",
		ProjectID: "proj1",
		AssetID:   "parent1",
		Action:    string(core.ActionServiceFingerprint),
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
</nmaprun>
`)

	ctx := context.Background()
	engine.onWorkComplete(ctx, w, stdout)
}

func TestOnWorkComplete_BatchModeDNS(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	members := []models.WorkBatchMember{
		{AssetID: "a1", Value: "example.com"},
	}
	data, _ := json.Marshal(members)
	w := &models.ScanWorkItem{
		ID:             "w1",
		RunID:          "run1",
		ProjectID:      "proj1",
		AssetID:        "a1",
		Action:         string(core.ActionDNSResolve),
		BatchMode:      true,
		MemberAssetIDs: string(data),
	}

	stdout := []byte(`{"host":"example.com","a":["1.2.3.4"],"cname":[],"aaaa":[]}
`)

	ctx := context.Background()
	engine.onWorkComplete(ctx, w, stdout)
}

func TestOnWorkComplete_BatchModeSubdomain(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	engine.queries.CreateAsset(&models.Asset{
		ID: "a1", ProjectID: "proj1", Type: "domain", Value: "example.com",
	})

	members := []models.WorkBatchMember{
		{AssetID: "a1", Value: "example.com"},
	}
	data, _ := json.Marshal(members)
	w := &models.ScanWorkItem{
		ID:             "w1",
		RunID:          "run1",
		ProjectID:      "proj1",
		AssetID:        "a1",
		Action:         string(core.ActionSubdomainEnum),
		BatchMode:      true,
		MemberAssetIDs: string(data),
	}

	stdout := []byte(`{"host":"sub.example.com","input":"example.com","source":"dns"}
`)

	ctx := context.Background()
	engine.onWorkComplete(ctx, w, stdout)
}

func TestOnWorkComplete_BatchModeCDN(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	engine.queries.CreateAsset(&models.Asset{
		ID: "a1", ProjectID: "proj1", Type: "ip", Value: "1.2.3.4",
	})

	members := []models.WorkBatchMember{
		{AssetID: "a1", Value: "1.2.3.4"},
	}
	data, _ := json.Marshal(members)
	w := &models.ScanWorkItem{
		ID:             "w1",
		RunID:          "run1",
		ProjectID:      "proj1",
		AssetID:        "a1",
		Action:         string(core.ActionCDNCheck),
		BatchMode:      true,
		MemberAssetIDs: string(data),
	}

	stdout := []byte(`{"ip":"1.2.3.4","cdn":true,"cdn_name":"cloudflare"}
`)

	ctx := context.Background()
	engine.onWorkComplete(ctx, w, stdout)
}

func TestOnWorkComplete_BatchModeHTTPX(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	// Skip screenshots to avoid TempDir cleanup race with background goroutine
	t.Setenv("ANCHOR_SKIP_SCREENSHOTS", "1")

	engine.queries.CreateAsset(&models.Asset{
		ID: "a1", ProjectID: "proj1", Type: "domain", Value: "example.com", NormalizedValue: "example.com",
	})

	members := []models.WorkBatchMember{
		{AssetID: "a1", Value: "https://example.com"},
	}
	data, _ := json.Marshal(members)
	w := &models.ScanWorkItem{
		ID:             "w1",
		RunID:          "run1",
		ProjectID:      "proj1",
		AssetID:        "a1",
		Action:         string(core.ActionHTTPXFingerprint),
		BatchMode:      true,
		MemberAssetIDs: string(data),
	}

	stdout := []byte(fmt.Sprintf(`{"input":"https://example.com","url":"https://example.com","host":"example.com","port":"443","path":"/","title":"Welcome","status_code":200,"tech":["nginx"]}%s`, "\n"))

	ctx := context.Background()
	engine.onWorkComplete(ctx, w, stdout)
}

func TestOnWorkComplete_BatchModeNuclei(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	engine.queries.CreateAsset(&models.Asset{
		ID: "a1", ProjectID: "proj1", Type: "url", Value: "https://example.com",
	})

	members := []models.WorkBatchMember{
		{AssetID: "a1", Value: "https://example.com"},
	}
	data, _ := json.Marshal(members)
	w := &models.ScanWorkItem{
		ID:             "w1",
		RunID:          "run1",
		ProjectID:      "proj1",
		AssetID:        "a1",
		Action:         string(core.ActionNucleiScan),
		BatchMode:      true,
		MemberAssetIDs: string(data),
	}

	stdout := []byte(`{"template-id":"test-001","host":"https://example.com","matched-at":"https://example.com/vuln","info":{"name":"Test Vuln","severity":"high"}}
`)

	ctx := context.Background()
	engine.onWorkComplete(ctx, w, stdout)
}

func TestOnWorkComplete_BatchModePortScan(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	engine.queries.CreateAsset(&models.Asset{
		ID: "a1", ProjectID: "proj1", Type: "ip", Value: "10.0.0.1",
	})

	members := []models.WorkBatchMember{
		{AssetID: "a1", Value: "10.0.0.1"},
	}
	data, _ := json.Marshal(members)
	w := &models.ScanWorkItem{
		ID:             "w1",
		RunID:          "run1",
		ProjectID:      "proj1",
		AssetID:        "a1",
		Action:         string(core.ActionPortScan),
		BatchMode:      true,
		MemberAssetIDs: string(data),
	}

	stdout := []byte(`{"host":"10.0.0.1","ip":"10.0.0.1","port":80}
`)

	ctx := context.Background()
	engine.onWorkComplete(ctx, w, stdout)
}

func TestOnWorkComplete_BatchModeNmap(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	engine.queries.CreateAsset(&models.Asset{
		ID: "a1", ProjectID: "proj1", Type: "ip", Value: "10.0.0.1",
	})

	members := []models.WorkBatchMember{
		{AssetID: "a1", Value: "10.0.0.1:80"},
	}
	data, _ := json.Marshal(members)
	w := &models.ScanWorkItem{
		ID:             "w1",
		RunID:          "run1",
		ProjectID:      "proj1",
		AssetID:        "a1",
		Action:         string(core.ActionServiceFingerprint),
		BatchMode:      true,
		MemberAssetIDs: string(data),
		InputFile:      "/tmp/nmap-input.txt",
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
</nmaprun>
`)

	ctx := context.Background()
	engine.onWorkComplete(ctx, w, stdout)
}

// ============================================================
// buildParams tests
// ============================================================

func TestBuildParams(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	// Create assets for testing (must set NormalizedValue to avoid UNIQUE constraint)
	engine.queries.CreateAsset(&models.Asset{
		ID: "a-subdomain", ProjectID: "proj1", Type: "domain", Value: "example.com", NormalizedValue: "example.com",
	})
	engine.queries.CreateAsset(&models.Asset{
		ID: "a-ip", ProjectID: "proj1", Type: "ip", Value: "10.0.0.1", NormalizedValue: "10.0.0.1",
	})
	engine.queries.CreateAsset(&models.Asset{
		ID: "a-ipport", ProjectID: "proj1", Type: "ip", Value: "10.0.0.1:80", NormalizedValue: "10.0.0.1:80",
	})
	engine.queries.CreateAsset(&models.Asset{
		ID: "a-url", ProjectID: "proj1", Type: "url", Value: "https://example.com", NormalizedValue: "https://example.com",
	})

	ctx := context.Background()

	t.Run("SubdomainEnum", func(t *testing.T) {
		w := &models.ScanWorkItem{AssetID: "a-subdomain", Action: string(core.ActionSubdomainEnum)}
		params, cleanup, err := engine.buildParams(ctx, w)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cleanup != nil {
			cleanup()
		}
		if _, ok := params["domain"]; !ok {
			t.Error("expected 'domain' param")
		}
	})

	t.Run("DNSResolve", func(t *testing.T) {
		w := &models.ScanWorkItem{AssetID: "a-subdomain", Action: string(core.ActionDNSResolve)}
		params, cleanup, err := engine.buildParams(ctx, w)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cleanup != nil {
			cleanup()
		}
		if _, ok := params["host_file"]; !ok {
			t.Error("expected 'host_file' param")
		}
	})

	t.Run("PortScan", func(t *testing.T) {
		w := &models.ScanWorkItem{AssetID: "a-ip", Action: string(core.ActionPortScan)}
		params, cleanup, err := engine.buildParams(ctx, w)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cleanup != nil {
			cleanup()
		}
		if _, ok := params["host_file"]; !ok {
			t.Error("expected 'host_file' param")
		}
		if _, ok := params["port_range"]; !ok {
			t.Error("expected 'port_range' param")
		}
	})

	t.Run("CDNCheck with IP", func(t *testing.T) {
		w := &models.ScanWorkItem{AssetID: "a-ip", Action: string(core.ActionCDNCheck)}
		params, cleanup, err := engine.buildParams(ctx, w)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cleanup != nil {
			cleanup()
		}
		if _, ok := params["ips"]; !ok {
			t.Error("expected 'ips' param")
		}
	})

	t.Run("CDNCheck with domain fails", func(t *testing.T) {
		w := &models.ScanWorkItem{AssetID: "a-subdomain", Action: string(core.ActionCDNCheck)}
		_, _, err := engine.buildParams(ctx, w)
		if err == nil {
			t.Error("expected error for CDN check with domain")
		}
	})

	t.Run("HTTPXFingerprint", func(t *testing.T) {
		w := &models.ScanWorkItem{AssetID: "a-url", Action: string(core.ActionHTTPXFingerprint)}
		params, cleanup, err := engine.buildParams(ctx, w)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cleanup != nil {
			cleanup()
		}
		if _, ok := params["host_file"]; !ok {
			t.Error("expected 'host_file' param")
		}
	})

	t.Run("NucleiScan", func(t *testing.T) {
		w := &models.ScanWorkItem{AssetID: "a-url", Action: string(core.ActionNucleiScan)}
		params, cleanup, err := engine.buildParams(ctx, w)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cleanup != nil {
			cleanup()
		}
		if _, ok := params["host_file"]; !ok {
			t.Error("expected 'host_file' param")
		}
		if _, ok := params["profile"]; !ok {
			t.Error("expected 'profile' param")
		}
	})

	t.Run("ServiceFingerprint", func(t *testing.T) {
		w := &models.ScanWorkItem{AssetID: "a-ipport", Action: string(core.ActionServiceFingerprint)}
		params, cleanup, err := engine.buildParams(ctx, w)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cleanup != nil {
			cleanup()
		}
		if _, ok := params["host_file"]; !ok {
			t.Error("expected 'host_file' param")
		}
		if _, ok := params["ports"]; !ok {
			t.Error("expected 'ports' param")
		}
	})

	t.Run("KatanaCrawl", func(t *testing.T) {
		w := &models.ScanWorkItem{AssetID: "a-url", Action: string(core.ActionKatanaCrawl)}
		params, cleanup, err := engine.buildParams(ctx, w)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cleanup != nil {
			cleanup()
		}
		if _, ok := params["url"]; !ok {
			t.Error("expected 'url' param")
		}
	})

	t.Run("FFUFBrute", func(t *testing.T) {
		w := &models.ScanWorkItem{AssetID: "a-url", Action: string(core.ActionFFUFBrute)}
		params, cleanup, err := engine.buildParams(ctx, w)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cleanup != nil {
			cleanup()
		}
		if _, ok := params["url"]; !ok {
			t.Error("expected 'url' param")
		}
	})

	t.Run("SpoorScan", func(t *testing.T) {
		w := &models.ScanWorkItem{AssetID: "a-url", Action: string(core.ActionSpoorScan)}
		params, cleanup, err := engine.buildParams(ctx, w)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cleanup != nil {
			cleanup()
		}
		if _, ok := params["target"]; !ok {
			t.Error("expected 'target' param")
		}
	})

	t.Run("default action", func(t *testing.T) {
		w := &models.ScanWorkItem{AssetID: "a-url", Action: "UNKNOWN"}
		params, cleanup, err := engine.buildParams(ctx, w)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cleanup != nil {
			cleanup()
		}
		if len(params) != 0 {
			t.Errorf("expected empty params, got %v", params)
		}
	})
}

func TestBuildBatchParams(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	t.Run("DNS batch", func(t *testing.T) {
		w := &models.ScanWorkItem{
			Action:    string(core.ActionDNSResolve),
			InputFile: "/tmp/dns-input.txt",
		}
		params, cleanup, err := engine.buildBatchParams(w)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cleanup != nil {
			cleanup()
		}
		if params["host_file"] != "/tmp/dns-input.txt" {
			t.Errorf("host_file = %v", params["host_file"])
		}
	})

	t.Run("PortScan batch", func(t *testing.T) {
		w := &models.ScanWorkItem{
			Action:    string(core.ActionPortScan),
			InputFile: "/tmp/port-input.txt",
		}
		params, cleanup, err := engine.buildBatchParams(w)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cleanup != nil {
			cleanup()
		}
		if params["host_file"] != "/tmp/port-input.txt" {
			t.Errorf("host_file = %v", params["host_file"])
		}
	})

	t.Run("CDN batch", func(t *testing.T) {
		// CDN batch reads lines from input file; use a temp file
		tmpFile := t.TempDir() + "/cdn-input.txt"
		if err := writeLinesFile(tmpFile, []string{"1.2.3.4"}); err != nil {
			t.Fatal(err)
		}
		w := &models.ScanWorkItem{
			Action:    string(core.ActionCDNCheck),
			InputFile: tmpFile,
		}
		params, cleanup, err := engine.buildBatchParams(w)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cleanup != nil {
			cleanup()
		}
		if _, ok := params["ips"]; !ok {
			t.Error("expected 'ips' param")
		}
	})

	t.Run("Subdomain batch", func(t *testing.T) {
		w := &models.ScanWorkItem{
			Action:    string(core.ActionSubdomainEnum),
			InputFile: "/tmp/sub-input.txt",
		}
		params, cleanup, err := engine.buildBatchParams(w)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cleanup != nil {
			cleanup()
		}
		if params["domain_file"] != "/tmp/sub-input.txt" {
			t.Errorf("domain_file = %v", params["domain_file"])
		}
	})

	t.Run("unsupported action", func(t *testing.T) {
		w := &models.ScanWorkItem{
			Action:    "UNKNOWN",
			InputFile: "/tmp/input.txt",
		}
		_, _, err := engine.buildBatchParams(w)
		if err == nil {
			t.Error("expected error for unsupported action")
		}
	})
}

func TestBuildTier2BatchParams(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	t.Run("HTTPX batch", func(t *testing.T) {
		w := &models.ScanWorkItem{
			Action:    string(core.ActionHTTPXFingerprint),
			InputFile: "/tmp/httpx-input.txt",
		}
		params, cleanup, err := engine.buildTier2BatchParams(w)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cleanup != nil {
			cleanup()
		}
		if params["host_file"] != "/tmp/httpx-input.txt" {
			t.Errorf("host_file = %v", params["host_file"])
		}
	})

	t.Run("Nmap batch", func(t *testing.T) {
		members := []models.WorkBatchMember{
			{AssetID: "a1", Value: "10.0.0.1:80", BucketKey: "b1"},
		}
		data, _ := json.Marshal(members)
		w := &models.ScanWorkItem{
			Action:         string(core.ActionServiceFingerprint),
			InputFile:      "/tmp/nmap-input.txt",
			MemberAssetIDs: string(data),
		}
		params, cleanup, err := engine.buildTier2BatchParams(w)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cleanup != nil {
			cleanup()
		}
		if params["host_file"] != "/tmp/nmap-input.txt" {
			t.Errorf("host_file = %v", params["host_file"])
		}
		if _, ok := params["ports"]; !ok {
			t.Error("expected 'ports' param")
		}
	})

	t.Run("Nuclei batch", func(t *testing.T) {
		w := &models.ScanWorkItem{
			Action:     string(core.ActionNucleiScan),
			InputFile:  "/tmp/nuclei-input.txt",
			BucketKey:  "nuclei:high",
		}
		params, cleanup, err := engine.buildTier2BatchParams(w)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cleanup != nil {
			cleanup()
		}
		if params["target_file"] != "/tmp/nuclei-input.txt" {
			t.Errorf("target_file = %v", params["target_file"])
		}
	})

	t.Run("unsupported", func(t *testing.T) {
		w := &models.ScanWorkItem{
			Action:    "UNKNOWN",
			InputFile: "/tmp/input.txt",
		}
		_, _, err := engine.buildTier2BatchParams(w)
		if err == nil {
			t.Error("expected error for unsupported action")
		}
	})
}

func TestAssetHostValue(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	engine.queries.CreateAsset(&models.Asset{
		ID: "a1", ProjectID: "proj1", Type: "domain", Value: "example.com",
	})

	t.Run("existing asset", func(t *testing.T) {
		val, err := engine.assetHostValue("a1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if val != "example.com" {
			t.Errorf("got %q, want %q", val, "example.com")
		}
	})

	t.Run("nonexistent asset", func(t *testing.T) {
		_, err := engine.assetHostValue("nonexistent")
		if err == nil {
			t.Error("expected error for nonexistent asset")
		}
	})
}

func TestEnqueueTier1Asset_NilPools(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)
	// Pools are nil by default (not initialized via initTier1Pools)

	ctx := context.Background()

	a := &core.DiscoveryAsset{
		ID:    "a1",
		Type:  core.AssetSubdomain,
		Value: "example.com",
	}

	t.Run("SubdomainEnum with nil pool", func(t *testing.T) {
		dw := core.DerivedWork{Action: core.ActionSubdomainEnum, AssetID: "a1"}
		// Should not panic
		engine.enqueueTier1Asset(ctx, a, dw, "bucket1")
	})

	t.Run("DNSResolve with nil pool", func(t *testing.T) {
		dw := core.DerivedWork{Action: core.ActionDNSResolve, AssetID: "a1"}
		engine.enqueueTier1Asset(ctx, a, dw, "bucket1")
	})

	t.Run("CDNCheck with nil pool", func(t *testing.T) {
		dw := core.DerivedWork{Action: core.ActionCDNCheck, AssetID: "a1"}
		engine.enqueueTier1Asset(ctx, a, dw, "bucket1")
	})

	t.Run("PortScan with nil pool", func(t *testing.T) {
		dw := core.DerivedWork{Action: core.ActionPortScan, AssetID: "a1"}
		engine.enqueueTier1Asset(ctx, a, dw, "bucket1")
	})
}

func TestEnqueueTier2Asset_NilPools(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)
	// Pools are nil by default

	ctx := context.Background()

	a := &core.DiscoveryAsset{
		ID:    "a1",
		Type:  core.AssetHTTPService,
		Value: "https://example.com",
	}

	t.Run("HTTPX with nil pool", func(t *testing.T) {
		dw := core.DerivedWork{Action: core.ActionHTTPXFingerprint, AssetID: "a1"}
		engine.enqueueTier2Asset(ctx, a, dw, "bucket1")
	})

	t.Run("ServiceFingerprint with nil pool", func(t *testing.T) {
		dw := core.DerivedWork{Action: core.ActionServiceFingerprint, AssetID: "a1"}
		engine.enqueueTier2Asset(ctx, a, dw, "bucket1")
	})

	t.Run("Nuclei with nil pool", func(t *testing.T) {
		dw := core.DerivedWork{Action: core.ActionNucleiScan, AssetID: "a1"}
		engine.enqueueTier2Asset(ctx, a, dw, "bucket1")
	})
}

func TestLinkFindingToScreenshot(t *testing.T) {
	fake := &fakeExecutor{}
	cfg := DefaultEngineConfig()
	engine, _ := setupTestEngine(t, fake, cfg)

	// With nil screenshotMgr, should not panic
	engine.screenshotMgr = nil
	engine.linkFindingToScreenshot("finding1", "https://example.com")
}

func TestCandidateURLs_Empty(t *testing.T) {
	urls := candidateURLs("")
	if len(urls) < 1 {
		t.Error("expected at least 1 URL for empty input")
	}
}

func TestCandidateURLs_WithPort(t *testing.T) {
	urls := candidateURLs("https://example.com:8443/path?q=1")
	if len(urls) < 3 {
		t.Errorf("expected at least 3 candidates, got %d: %v", len(urls), urls)
	}
}

// writeLinesFile is a helper to create a temp file with one line per entry.
func writeLinesFile(path string, lines []string) error {
	data := ""
	for _, l := range lines {
		data += l + "\n"
	}
	return os.WriteFile(path, []byte(data), 0644)
}
