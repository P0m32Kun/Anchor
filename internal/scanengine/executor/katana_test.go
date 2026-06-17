package executor

import (
	"testing"

	"github.com/P0m32Kun/Anchor/internal/scanengine/core"
)

func TestParseKatanaOutput_JSURLClassification(t *testing.T) {
	stdout := []byte(`{"url":"https://example.com/app.js"}
{"url":"https://example.com/style.css"}
{"url":"https://example.com/api/v1?token=abc"}
{"url":"https://cdn.example.com/bundle.min.js?v=123"}
`)

	assets, err := ParseKatanaOutput(stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(assets) != 4 {
		t.Fatalf("expected 4 assets, got %d", len(assets))
	}

	// app.js → JS_URL
	if assets[0].Type != core.AssetJSURL {
		t.Errorf("asset[0] type = %q, want JS_URL", assets[0].Type)
	}

	// style.css → HTTP_PATH
	if assets[1].Type != core.AssetHTTPPath {
		t.Errorf("asset[1] type = %q, want HTTP_PATH", assets[1].Type)
	}

	// api/v1 → HTTP_PATH
	if assets[2].Type != core.AssetHTTPPath {
		t.Errorf("asset[2] type = %q, want HTTP_PATH", assets[2].Type)
	}

	// bundle.min.js?v=123 → JS_URL (query stripped)
	if assets[3].Type != core.AssetJSURL {
		t.Errorf("asset[3] type = %q, want JS_URL", assets[3].Type)
	}
}

func TestIsJSURL(t *testing.T) {
	tests := []struct {
		url    string
		expect bool
	}{
		{"https://example.com/app.js", true},
		{"https://example.com/bundle.min.js", true},
		{"https://example.com/app.js?v=123", true},
		{"https://example.com/style.css", false},
		{"https://example.com/api/v1", false},
		{"https://example.com/file.js#section", true},
	}

	for _, tt := range tests {
		got := isJSURL(tt.url)
		if got != tt.expect {
			t.Errorf("isJSURL(%q) = %v, want %v", tt.url, got, tt.expect)
		}
	}
}
