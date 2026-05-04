package workflow

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/worker"
)

func TestExtractValues(t *testing.T) {
	tests := []struct {
		name   string
		assets []*models.Asset
		want   []string
	}{
		{
			name:   "empty",
			assets: nil,
			want:   nil,
		},
		{
			name: "single",
			assets: []*models.Asset{
				{Value: "example.com"},
			},
			want: []string{"example.com"},
		},
		{
			name: "multiple unique",
			assets: []*models.Asset{
				{Value: "a.com"},
				{Value: "b.com"},
				{Value: "c.com"},
			},
			want: []string{"a.com", "b.com", "c.com"},
		},
		{
			name: "duplicates removed",
			assets: []*models.Asset{
				{Value: "a.com"},
				{Value: "b.com"},
				{Value: "a.com"},
				{Value: "c.com"},
				{Value: "b.com"},
			},
			want: []string{"a.com", "b.com", "c.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractValues(tt.assets)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("extractValues() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJoinArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"empty", nil, ""},
		{"single", []string{"-a"}, "-a"},
		{"multiple", []string{"-a", "1", "-b", "2"}, "-a 1 -b 2"},
		{"with spaces", []string{"hello world", "foo"}, "hello world foo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := joinArgs(tt.args)
			if got != tt.want {
				t.Errorf("joinArgs(%v) = %q, want %q", tt.args, got, tt.want)
			}
		})
	}
}

func TestParseInt(t *testing.T) {
	tests := []struct {
		name    string
		s       string
		want    int
		wantErr bool
	}{
		{"zero", "0", 0, false},
		{"positive", "42", 42, false},
		{"negative", "-7", -7, false},
		{"large", "123456789", 123456789, false},
		{"invalid empty", "", 0, true},
		{"invalid text", "abc", 0, true},
		{"partial numeric", "12a", 12, false},
		{"invalid leading text", "a12", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseInt(tt.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseInt(%q) error = %v, wantErr %v", tt.s, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseInt(%q) = %d, want %d", tt.s, got, tt.want)
			}
		})
	}
}

func TestWriteHostsFile(t *testing.T) {
	tmpDir := t.TempDir()
	projectID := "proj-1"
	hosts := []string{"192.168.1.1", "192.168.1.2", "example.com"}

	path, err := writeHostsFile(tmpDir, projectID, hosts)
	if err != nil {
		t.Fatalf("writeHostsFile failed: %v", err)
	}

	expectedDir := filepath.Join(tmpDir, "workdirs", projectID, "hostlists")
	if !strings.HasPrefix(path, expectedDir) {
		t.Errorf("path %q does not start with %q", path, expectedDir)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file failed: %v", err)
	}

	want := "192.168.1.1\n192.168.1.2\nexample.com\n"
	if string(content) != want {
		t.Errorf("content = %q, want %q", string(content), want)
	}
}

func TestBuildNaabuArgsWithPortRange(t *testing.T) {
	tests := []struct {
		name      string
		hostFile  string
		portRange string
		want      []string
	}{
		{
			name:      "default empty",
			hostFile:  "hosts.txt",
			portRange: "",
			want:      []string{"naabu", "-json", "-list", "hosts.txt"},
		},
		{
			name:      "top100",
			hostFile:  "hosts.txt",
			portRange: "tp100",
			want:      []string{"naabu", "-json", "-list", "hosts.txt"},
		},
		{
			name:      "top1000",
			hostFile:  "hosts.txt",
			portRange: "tp1000",
			want:      []string{"naabu", "-json", "-list", "hosts.txt", "-tp", "1000"},
		},
		{
			name:      "full",
			hostFile:  "hosts.txt",
			portRange: "full",
			want:      []string{"naabu", "-json", "-list", "hosts.txt", "-tp", "full"},
		},
		{
			name:      "custom range",
			hostFile:  "hosts.txt",
			portRange: "80,443,8080",
			want:      []string{"naabu", "-json", "-list", "hosts.txt", "-p", "80,443,8080"},
		},
		{
			name:      "case insensitive",
			hostFile:  "hosts.txt",
			portRange: "TOP100",
			want:      []string{"naabu", "-json", "-list", "hosts.txt"},
		},
		{
			name:      "high-risk",
			hostFile:  "hosts.txt",
			portRange: "high-risk",
			want:      []string{"naabu", "-json", "-list", "hosts.txt", "-p", worker.HighRiskPorts},
		},
		{
			name:      "high-risk alias hr",
			hostFile:  "hosts.txt",
			portRange: "HR",
			want:      []string{"naabu", "-json", "-list", "hosts.txt", "-p", worker.HighRiskPorts},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildNaabuArgsWithPortRange(tt.hostFile, tt.portRange)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildNaabuArgsWithPortRange() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestComputeDedupKey(t *testing.T) {
	tests := []struct {
		templateID  string
		host        string
		matcherName string
	}{
		{"test-id", "example.com", "matcher1"},
		{"id-2", "192.168.1.1", "status-code"},
		{"", "", ""},
		{"a", "b", "c"},
	}

	// Same inputs should produce same hash.
	for _, tt := range tests {
		got1 := computeDedupKey(tt.templateID, tt.host, tt.matcherName)
		got2 := computeDedupKey(tt.templateID, tt.host, tt.matcherName)
		if got1 != got2 {
			t.Errorf("computeDedupKey(%q, %q, %q) not deterministic: %q vs %q",
				tt.templateID, tt.host, tt.matcherName, got1, got2)
		}
		if len(got1) != 64 { // sha256 hex length
			t.Errorf("computeDedupKey() length = %d, want 64", len(got1))
		}
	}

	// Different inputs should produce different hashes.
	got1 := computeDedupKey("a", "b", "c")
	got2 := computeDedupKey("a", "b", "d")
	if got1 == got2 {
		t.Error("computeDedupKey should produce different hashes for different inputs")
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		maxLen int
		want   string
	}{
		{"empty", "", 10, ""},
		{"shorter", "hello", 10, "hello"},
		{"exact", "hello", 5, "hello"},
		{"longer", "hello world", 5, "hello..."},
		{"zero max", "hello", 0, "..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateString(tt.s, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateString(%q, %d) = %q, want %q", tt.s, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestWriteTargetsFile(t *testing.T) {
	tmpDir := t.TempDir()
	projectID := "proj-1"
	urls := []string{"https://example.com", "https://test.com/path"}

	path, err := writeTargetsFile(tmpDir, projectID, urls)
	if err != nil {
		t.Fatalf("writeTargetsFile failed: %v", err)
	}

	expectedPath := filepath.Join(tmpDir, "workdirs", projectID, "screening", "targets.txt")
	if path != expectedPath {
		t.Errorf("path = %q, want %q", path, expectedPath)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file failed: %v", err)
	}

	want := "https://example.com\nhttps://test.com/path\n"
	if string(content) != want {
		t.Errorf("content = %q, want %q", string(content), want)
	}
}
