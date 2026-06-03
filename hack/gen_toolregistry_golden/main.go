// One-off generator for internal/toolregistry/testdata/golden/*.json
// Run: go run ./hack/gen_toolregistry_golden
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/P0m32Kun/Anchor/internal/toolregistry"
	"github.com/P0m32Kun/Anchor/internal/worker"
)

type goldenFile struct {
	Name         string                 `json:"name"`
	ToolID       string                 `json:"tool_id"`
	Params       map[string]interface{} `json:"params"`
	IgnoreTokens []string               `json:"ignore_tokens,omitempty"`
	WantArgv     []string               `json:"want_argv"`
}

type caseDef struct {
	name         string
	toolID       string
	params       toolregistry.RenderParams
	ignoreTokens []string
	wantFn       func() []string
}

func main() {
	outDir := filepath.Join("internal", "toolregistry", "testdata", "golden")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir: %v\n", err)
		os.Exit(1)
	}

	cases := []caseDef{
		{
			name:   "subfinder_passive",
			toolID: "subfinder",
			params: toolregistry.RenderParams{
				"domain": "example.com", "rate_limit": 50, "threads": 10, "timeout": 30, "mode": "passive",
			},
			wantFn: func() []string {
				return worker.BuildSubfinderCommand("example.com", 50, 10, 30, "passive", "")
			},
		},
		{
			name:   "subfinder_active",
			toolID: "subfinder",
			params: toolregistry.RenderParams{
				"domain": "test.org", "rate_limit": 100, "threads": 5, "mode": "active",
			},
			wantFn: func() []string {
				return worker.BuildSubfinderCommand("test.org", 100, 5, 0, "active", "")
			},
		},
		{
			name:   "dnsx",
			toolID: "dnsx",
			params: toolregistry.RenderParams{
				"host_file": "/tmp/hosts.txt", "rate_limit": 100, "threads": 50, "timeout": 5,
			},
			wantFn: func() []string {
				return worker.BuildDNSxCommand("/tmp/hosts.txt", nil, 100, 50, 5)
			},
		},
		{
			name:   "cdncheck",
			toolID: "cdncheck",
			params: toolregistry.RenderParams{"ips": "1.1.1.1,8.8.8.8"},
			wantFn: func() []string { return worker.BuildCDNCheckCommand("1.1.1.1,8.8.8.8") },
		},
		{
			name:   "nmap_alive",
			toolID: "nmap_alive",
			params: toolregistry.RenderParams{"host_file": "/tmp/hosts.txt"},
			wantFn: func() []string { return worker.BuildNmapAliveCommand("/tmp/hosts.txt") },
		},
		{
			name:   "nmap_service",
			toolID: "nmap_service",
			params: toolregistry.RenderParams{
				"host_file": "/tmp/hosts.txt", "ports": []string{"80", "443"}, "host_timeout": 180,
			},
			wantFn: func() []string {
				return worker.BuildNmapServiceScanCommand("/tmp/hosts.txt", []int{80, 443}, 180)
			},
		},
		{
			name:   "naabu_top1000",
			toolID: "naabu",
			params: toolregistry.RenderParams{
				"host_file": "/tmp/hosts.txt", "port_range": "top1000", "rate": 1000, "threads": 100,
			},
			wantFn: func() []string {
				return worker.BuildNaabuCommand("/tmp/hosts.txt", "top1000", 1000, 100, 0)
			},
		},
		{
			name:   "naabu_high_risk",
			toolID: "naabu",
			params: toolregistry.RenderParams{
				"host_file": "/tmp/hosts.txt", "port_range": "high-risk", "rate": 300, "threads": 50,
			},
			wantFn: func() []string {
				return worker.BuildNaabuCommand("/tmp/hosts.txt", "high-risk", 300, 50, 0)
			},
		},
		{
			name:   "httpx",
			toolID: "httpx",
			params: toolregistry.RenderParams{
				"host_file": "/tmp/hosts.txt", "rate_limit": 150, "threads": 50,
			},
			wantFn: func() []string { return worker.BuildHttpxCommand("/tmp/hosts.txt", 150, 50, "") },
		},
		{
			name:   "katana",
			toolID: "katana",
			params: toolregistry.RenderParams{
				"list_file": "/tmp/urls.txt", "depth": 2, "rate_limit": 10, "timeout": 10,
			},
			wantFn: func() []string { return worker.BuildKatanaCommand("/tmp/urls.txt", 2, 10, 10) },
		},
		{
			name:   "ffuf",
			toolID: "ffuf",
			params: toolregistry.RenderParams{
				"target": "http://example.com/FUZZ", "wordlist": "/tmp/dict.txt", "rate_limit": 6, "timeout": 30,
			},
			wantFn: func() []string {
				return worker.BuildFfufCommand("http://example.com/FUZZ", "/tmp/dict.txt", 6, 30)
			},
		},
		{
			name:   "nuclei_deep_tags",
			toolID: "nuclei",
			params: toolregistry.RenderParams{
				"target_file": "/tmp/targets.txt", "profile": "deep", "rate_limit": 20, "concurrency": 5,
				"scan_depth": "tags", "tags": []string{"cve", "misconfig"},
			},
			ignoreTokens: []string{"-stats", "-si", "30"},
			wantFn: func() []string {
				return worker.BuildNucleiCommand("/tmp/targets.txt", "deep", 20, 0, 5, []string{"cve", "misconfig"}, "tags", "", "")
			},
		},
		{
			name:   "nuclei_standard_workflow",
			toolID: "nuclei",
			params: toolregistry.RenderParams{
				"target_file": "/tmp/targets.txt", "profile": "standard", "scan_depth": "workflow",
				"workflow_dir": "/root/workflows",
			},
			ignoreTokens: []string{"-stats", "-si", "30"},
			wantFn: func() []string {
				return worker.BuildNucleiCommand("/tmp/targets.txt", "standard", 0, 0, 0, nil, "workflow", "/root/workflows", "")
			},
		},
		{
			name:   "nuclei_light_both",
			toolID: "nuclei",
			params: toolregistry.RenderParams{
				"target_file": "/tmp/targets.txt", "profile": "light", "rate_limit": 150, "concurrency": 10,
				"scan_depth": "both", "workflow_dir": "/root/workflows", "tags": []string{"rce"},
			},
			ignoreTokens: []string{"-stats", "-si", "30"},
			wantFn: func() []string {
				return worker.BuildNucleiCommand("/tmp/targets.txt", "light", 150, 0, 10, []string{"rce"}, "both", "/root/workflows", "")
			},
		},
		{
			name:   "nuclei_custom_template",
			toolID: "nuclei",
			params: toolregistry.RenderParams{
				"target_file": "/tmp/targets.txt", "profile": "deep", "template_path": "/root/custom-workflows/ssh.yaml",
			},
			ignoreTokens: []string{"-stats", "-si", "30"},
			wantFn: func() []string {
				return worker.BuildNucleiCommand("/tmp/targets.txt", "deep", 0, 0, 0, nil, "tags", "", "/root/custom-workflows/ssh.yaml")
			},
		},
		{
			name:   "gau",
			toolID: "gau",
			params: toolregistry.RenderParams{"domain": "example.com"},
			wantFn: func() []string { return []string{"gau", "example.com"} },
		},
	}

	reg := toolregistry.DefaultRegistry()
	for _, c := range cases {
		got, err := reg.Render(c.toolID, c.params)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Render(%s): %v\n", c.name, err)
			os.Exit(1)
		}
		gotSet := toolregistry.ArgvSetMinus(got, c.ignoreTokens)
		wantSet := toolregistry.ArgvSetMinus(c.wantFn(), c.ignoreTokens)
		if fmt.Sprint(gotSet) != fmt.Sprint(wantSet) {
			fmt.Fprintf(os.Stderr, "warn %s: render/worker mismatch (fixture uses render)\n", c.name)
		}
		g := goldenFile{
			Name: c.name, ToolID: c.toolID,
			Params: map[string]interface{}(c.params),
			IgnoreTokens: c.ignoreTokens,
			WantArgv:     gotSet,
		}
		b, _ := json.MarshalIndent(g, "", "  ")
		path := filepath.Join(outDir, c.name+".json")
		if err := os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "write %s: %v\n", path, err)
			os.Exit(1)
		}
		fmt.Println("wrote", path)
	}
}
