package toolregistry

import (
	"encoding/json"
	"embed"
	"io/fs"
	"slices"
	"testing"
)

//go:embed testdata/golden/*.json
var goldenFS embed.FS

type goldenFile struct {
	Name         string                 `json:"name"`
	ToolID       string                 `json:"tool_id"`
	Params       map[string]interface{} `json:"params"`
	IgnoreTokens []string               `json:"ignore_tokens,omitempty"`
	WantArgv     []string               `json:"want_argv"`
}

func TestGolden_RenderMatchesFixtures(t *testing.T) {
	reg := DefaultRegistry()

	entries, err := fs.Glob(goldenFS, "testdata/golden/*.json")
	if err != nil {
		t.Fatalf("glob golden fixtures: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("no golden fixtures found")
	}

	for _, path := range entries {
		path := path
		t.Run(path, func(t *testing.T) {
			data, err := goldenFS.ReadFile(path)
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}
			var gf goldenFile
			if err := json.Unmarshal(data, &gf); err != nil {
				t.Fatalf("unmarshal fixture: %v", err)
			}

			got, err := reg.Render(gf.ToolID, RenderParams(gf.Params))
			if err != nil {
				t.Fatalf("Render(%q) error: %v", gf.ToolID, err)
			}

			gotSet := ArgvSetMinus(got, gf.IgnoreTokens)
			if !slices.Equal(gotSet, gf.WantArgv) {
				t.Errorf("Render(%q) argv set mismatch:\ngot:  %v\nwant: %v", gf.ToolID, gotSet, gf.WantArgv)
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
