package toolregistry

import (
	"slices"
	"testing"

	"github.com/P0m32Kun/Anchor/internal/worker"
)

// goldenTest asserts that registry.Render(id, params) produces argv equivalent
// to the legacy worker.Build* function (order-independent set comparison).
type goldenTest struct {
	toolID string
	params RenderParams
	wantFn func() []string
	// ignoreTokens are argv tokens excluded from comparison (e.g. -stats -si 30).
	ignoreTokens []string
}

func TestGolden_RenderMatchesBuildFunctions(t *testing.T) {
	reg := DefaultRegistry()

	tests := []goldenTest{
		{
			toolID: "subfinder",
			params: RenderParams{
				"domain":       "example.com",
				"rate_limit":   50,
				"threads":      10,
				"timeout":      30,
				"mode":         "passive",
			},
			wantFn: func() []string {
				return worker.BuildSubfinderCommand("example.com", 50, 10, 30, "passive", "")
			},
		},
		{
			toolID: "subfinder",
			params: RenderParams{
				"domain":     "test.org",
				"rate_limit": 100,
				"threads":    5,
				"mode":       "active",
			},
			wantFn: func() []string {
				return worker.BuildSubfinderCommand("test.org", 100, 5, 0, "active", "")
			},
		},
		{
			toolID: "dnsx",
			params: RenderParams{
				"host_file":  "/tmp/hosts.txt",
				"rate_limit": 100,
				"threads":    50,
				"timeout":    5,
			},
			wantFn: func() []string {
				return worker.BuildDNSxCommand("/tmp/hosts.txt", nil, 100, 50, 5)
			},
		},
		{
			toolID: "cdncheck",
			params: RenderParams{
				"ips": "1.1.1.1,8.8.8.8",
			},
			wantFn: func() []string {
				return worker.BuildCDNCheckCommand("1.1.1.1,8.8.8.8")
			},
		},
		{
			toolID: "nmap_alive",
			params: RenderParams{
				"host_file": "/tmp/hosts.txt",
			},
			wantFn: func() []string {
				return worker.BuildNmapAliveCommand("/tmp/hosts.txt")
			},
		},
		{
			toolID: "nmap_service",
			params: RenderParams{
				"host_file":    "/tmp/hosts.txt",
				"ports":        []string{"80", "443"},
				"host_timeout": 180,
			},
			wantFn: func() []string {
				return worker.BuildNmapServiceScanCommand("/tmp/hosts.txt", []int{80, 443}, 180)
			},
		},
		{
			toolID: "naabu",
			params: RenderParams{
				"host_file":  "/tmp/hosts.txt",
				"port_range": "top1000",
				"rate":       1000,
				"threads":    100,
			},
			wantFn: func() []string {
				return worker.BuildNaabuCommand("/tmp/hosts.txt", "top1000", 1000, 100, 0)
			},
		},
		{
			toolID: "naabu",
			params: RenderParams{
				"host_file":  "/tmp/hosts.txt",
				"port_range": "high-risk",
				"rate":       300,
				"threads":    50,
			},
			wantFn: func() []string {
				return worker.BuildNaabuCommand("/tmp/hosts.txt", "high-risk", 300, 50, 0)
			},
		},
		{
			toolID: "httpx",
			params: RenderParams{
				"host_file":  "/tmp/hosts.txt",
				"rate_limit": 150,
				"threads":    50,
			},
			wantFn: func() []string {
				return worker.BuildHttpxCommand("/tmp/hosts.txt", 150, 50, "")
			},
		},
		{
			toolID: "katana",
			params: RenderParams{
				"list_file":  "/tmp/urls.txt",
				"depth":      2,
				"rate_limit": 10,
				"timeout":    10,
			},
			wantFn: func() []string {
				return worker.BuildKatanaCommand("/tmp/urls.txt", 2, 10, 10)
			},
		},
		{
			toolID: "ffuf",
			params: RenderParams{
				"target":     "http://example.com/FUZZ",
				"wordlist":   "/tmp/dict.txt",
				"rate_limit": 6,
				"timeout":    30,
			},
			wantFn: func() []string {
				return worker.BuildFfufCommand("http://example.com/FUZZ", "/tmp/dict.txt", 6, 30)
			},
		},
		{
			toolID: "nuclei",
			params: RenderParams{
				"target_file": "/tmp/targets.txt",
				"profile":     "deep",
				"rate_limit":  20,
				"concurrency": 5,
				"scan_depth":  "tags",
				"tags":        []string{"cve", "misconfig"},
			},
			wantFn: func() []string {
				return worker.BuildNucleiCommand("/tmp/targets.txt", "deep", 20, 0, 5, []string{"cve", "misconfig"}, "tags", "", "")
			},
			ignoreTokens: []string{"-stats", "-si", "30"},
		},
		{
			toolID: "nuclei",
			params: RenderParams{
				"target_file":  "/tmp/targets.txt",
				"profile":      "standard",
				"scan_depth":   "workflow",
				"workflow_dir": "/root/workflows",
			},
			wantFn: func() []string {
				return worker.BuildNucleiCommand("/tmp/targets.txt", "standard", 0, 0, 0, nil, "workflow", "/root/workflows", "")
			},
			ignoreTokens: []string{"-stats", "-si", "30"},
		},
		{
			toolID: "nuclei",
			params: RenderParams{
				"target_file":  "/tmp/targets.txt",
				"profile":      "light",
				"rate_limit":   150,
				"concurrency":  10,
				"scan_depth":   "both",
				"workflow_dir": "/root/workflows",
				"tags":         []string{"rce"},
			},
			wantFn: func() []string {
				return worker.BuildNucleiCommand("/tmp/targets.txt", "light", 150, 0, 10, []string{"rce"}, "both", "/root/workflows", "")
			},
			ignoreTokens: []string{"-stats", "-si", "30"},
		},
		{
			toolID: "nuclei",
			params: RenderParams{
				"target_file":   "/tmp/targets.txt",
				"profile":       "deep",
				"template_path": "/root/custom-workflows/ssh.yaml",
			},
			wantFn: func() []string {
				return worker.BuildNucleiCommand("/tmp/targets.txt", "deep", 0, 0, 0, nil, "tags", "", "/root/custom-workflows/ssh.yaml")
			},
			ignoreTokens: []string{"-stats", "-si", "30"},
		},
		{
			toolID: "gau",
			params: RenderParams{
				"domain": "example.com",
			},
			wantFn: func() []string {
				return []string{"gau", "example.com"}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.toolID, func(t *testing.T) {
			got, err := reg.Render(tt.toolID, tt.params)
			if err != nil {
				t.Fatalf("Render(%q) error: %v", tt.toolID, err)
			}
			want := tt.wantFn()

			gotSet := ArgvSetMinus(got, tt.ignoreTokens)
			wantSet := ArgvSetMinus(want, tt.ignoreTokens)

			if !slices.Equal(gotSet, wantSet) {
				t.Errorf("Render(%q) argv set mismatch:\ngot:  %v\nwant: %v", tt.toolID, gotSet, wantSet)
			}
		})
	}
}

func TestBinaries(t *testing.T) {
	reg := DefaultRegistry()
	bins := reg.Binaries()
	expected := []string{
		"subfinder", "dnsx", "cdncheck", "nmap", "naabu",
		"httpx", "katana", "ffuf", "nuclei", "gau",
	}
	for _, b := range expected {
		if !slices.Contains(bins, b) {
			t.Errorf("Binaries() missing %q", b)
		}
	}
}

func TestList(t *testing.T) {
	reg := DefaultRegistry()
	ids := reg.List()
	expected := []string{
		"subfinder", "dnsx", "cdncheck", "nmap_alive", "nmap_service",
		"naabu", "httpx", "katana", "ffuf", "nuclei", "gau",
	}
	for _, id := range expected {
		if !slices.Contains(ids, id) {
			t.Errorf("List() missing %q", id)
		}
	}
}

func TestGet(t *testing.T) {
	reg := DefaultRegistry()
	if def := reg.Get("naabu"); def == nil {
		t.Fatal("Get('naabu') returned nil")
	} else if def.Binary != "naabu" {
		t.Errorf("naabu binary = %q, want 'naabu'", def.Binary)
	}
	if def := reg.Get("nonexistent"); def != nil {
		t.Errorf("Get('nonexistent') = %v, want nil", def)
	}
}

func TestDuplicateToolID(t *testing.T) {
	_, err := Load([]*ToolDef{
		{ID: "test", Binary: "tool1"},
		{ID: "test", Binary: "tool2"},
	})
	if err == nil {
		t.Fatal("expected error for duplicate tool id")
	}
}

func TestEmptyToolID(t *testing.T) {
	_, err := Load([]*ToolDef{
		{ID: "", Binary: "tool"},
	})
	if err == nil {
		t.Fatal("expected error for empty tool id")
	}
}

func TestValidate(t *testing.T) {
	reg := DefaultRegistry()
	warnings, errors := reg.ValidateAll()
	for _, err := range errors {
		t.Errorf("validation error: %v", err)
	}
	if len(warnings) > 0 {
		t.Logf("validation warnings: %v", warnings)
	}
}
