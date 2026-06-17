package pool

import (
	"testing"

	"github.com/P0m32Kun/Anchor/internal/scanengine/core"
)

func TestProbeTargetFromAsset(t *testing.T) {
	tests := []struct {
		value string
		typ   core.AssetType
		want  string
	}{
		{"Example.COM", core.AssetSubdomain, "example.com"},
		{"10.0.0.1", core.AssetIP, "10.0.0.1"},
		{"10.0.0.1:443", core.AssetIPPort, "10.0.0.1:443"},
		{"https://a.example.com", core.AssetHTTPService, "https://a.example.com"},
	}
	for _, tt := range tests {
		if got := ProbeTargetFromAsset(tt.value, tt.typ); got != tt.want {
			t.Errorf("ProbeTargetFromAsset(%q, %v) = %q, want %q", tt.value, tt.typ, got, tt.want)
		}
	}
}

func TestParseHostPort(t *testing.T) {
	host, port := ParseHostPort("1.2.3.4:8080")
	if host != "1.2.3.4" || port != 8080 {
		t.Fatalf("got %s:%d", host, port)
	}
}
