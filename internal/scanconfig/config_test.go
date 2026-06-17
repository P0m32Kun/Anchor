package scanconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadAppliesSidecarFiles(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "bundled")
	if err := os.MkdirAll(src, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(src, "scan.config.yaml"), []byte(`
paths:
  high_risk_ports_file: custom-ports.txt
  junk_keywords_file: custom-junk.txt
passive:
  result_limit: 42
  fofa_queries:
    - 'org="{{company}}"'
presets:
  external:
    port_range: top1000
`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "custom-ports.txt"), []byte("8080,8443\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "custom-junk.txt"), []byte("testjunk word_boundary\n"), 0644); err != nil {
		t.Fatal(err)
	}

	EnsureConfigFiles(src, dir)
	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.HighRiskPorts != "8080,8443" {
		t.Fatalf("HighRiskPorts = %q", cfg.HighRiskPorts)
	}
	if cfg.Passive.ResultLimit != 42 {
		t.Fatalf("Passive.ResultLimit = %d", cfg.Passive.ResultLimit)
	}
	if len(cfg.JunkKeywords) != 1 || cfg.JunkKeywords[0].Keyword != "testjunk" || !cfg.JunkKeywords[0].WordBoundary {
		t.Fatalf("JunkKeywords = %+v", cfg.JunkKeywords)
	}

	preset := cfg.Preset("external")
	if preset.PortRange != "top1000" {
		t.Fatalf("external preset PortRange = %q", preset.PortRange)
	}
}

func TestDefaultsAPIIncludesPresets(t *testing.T) {
	dir := t.TempDir()
	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	resp := cfg.DefaultsAPI()
	if resp.HighRiskPorts == "" {
		t.Fatal("expected compiled high_risk_ports")
	}
	if _, ok := resp.Presets["external"]; !ok {
		t.Fatal("missing external preset")
	}
	if _, ok := resp.Presets["internal"]; !ok {
		t.Fatal("missing internal preset")
	}
}

func TestReadJunkKeywordsFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "junk.txt")
	content := "# comment\n博彩\nporn word_boundary\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	kws, err := readJunkKeywordsFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(kws) != 2 {
		t.Fatalf("got %d keywords", len(kws))
	}
	if kws[1].Keyword != "porn" || !kws[1].WordBoundary {
		t.Fatalf("second keyword = %+v", kws[1])
	}
}

func TestReadPortsFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ports.txt")
	if err := os.WriteFile(path, []byte("80, 443\n8080\n"), 0644); err != nil {
		t.Fatal(err)
	}
	ports, err := readPortsFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(ports, "80") || !strings.Contains(ports, "8080") {
		t.Fatalf("ports = %q", ports)
	}
}

func TestPresetValuesMatchDesignDoc(t *testing.T) {
	dir := t.TempDir()
	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// external_low: naabu 100/20, nuclei 5/3/20rpm
	low := cfg.Preset("external_low")
	if low.NaabuRate != 100 {
		t.Errorf("external_low.NaabuRate = %d, want 100", low.NaabuRate)
	}
	if low.NaabuThreads != 20 {
		t.Errorf("external_low.NaabuThreads = %d, want 20", low.NaabuThreads)
	}
	if low.NucleiRateLimit != 5 {
		t.Errorf("external_low.NucleiRateLimit = %d, want 5", low.NucleiRateLimit)
	}
	if low.NucleiConcurrency != 3 {
		t.Errorf("external_low.NucleiConcurrency = %d, want 3", low.NucleiConcurrency)
	}
	if low.NucleiRateLimitPerMinute != 20 {
		t.Errorf("external_low.NucleiRateLimitPerMinute = %d, want 20", low.NucleiRateLimitPerMinute)
	}
	if low.EnableFfuf {
		t.Error("external_low.EnableFfuf should be false (design: off)")
	}

	// external_standard: naabu 150/30, nuclei 10/5/30rpm
	std := cfg.Preset("external_standard")
	if std.NaabuRate != 150 {
		t.Errorf("external_standard.NaabuRate = %d, want 150", std.NaabuRate)
	}
	if std.NaabuThreads != 30 {
		t.Errorf("external_standard.NaabuThreads = %d, want 30", std.NaabuThreads)
	}
	if std.NucleiRateLimit != 10 {
		t.Errorf("external_standard.NucleiRateLimit = %d, want 10", std.NucleiRateLimit)
	}
	if std.NucleiConcurrency != 5 {
		t.Errorf("external_standard.NucleiConcurrency = %d, want 5", std.NucleiConcurrency)
	}
	if std.NucleiRateLimitPerMinute != 30 {
		t.Errorf("external_standard.NucleiRateLimitPerMinute = %d, want 30", std.NucleiRateLimitPerMinute)
	}

	// watch: naabu 150/30, nuclei 10/5/30rpm
	watch := cfg.Preset("watch")
	if watch.NaabuRate != 150 {
		t.Errorf("watch.NaabuRate = %d, want 150", watch.NaabuRate)
	}
	if watch.NaabuThreads != 30 {
		t.Errorf("watch.NaabuThreads = %d, want 30", watch.NaabuThreads)
	}
	if watch.NucleiRateLimit != 10 {
		t.Errorf("watch.NucleiRateLimit = %d, want 10", watch.NucleiRateLimit)
	}
}
