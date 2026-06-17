package parser

import (
	"strings"
	"testing"
)

func TestParseGauOutput_WithQueryParams(t *testing.T) {
	input := `https://example.com/api/v1?token=abc123&page=1
https://example.com/static/main.js
https://example.com/search?q=test&lang=en
https://example.com/api/v1?token=abc123&page=1
`
	results, errs := ParseGauOutput(strings.NewReader(input))

	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results (deduped), got %d", len(results))
	}

	// First URL with query
	r0 := results[0]
	if r0.Host != "example.com" {
		t.Errorf("host = %q, want example.com", r0.Host)
	}
	if r0.Path != "/api/v1" {
		t.Errorf("path = %q, want /api/v1", r0.Path)
	}
	if !r0.HasQuery {
		t.Error("expected HasQuery=true")
	}
	if r0.Params["token"] != "abc123" {
		t.Errorf("params[token] = %q, want abc123", r0.Params["token"])
	}
	if r0.Params["page"] != "1" {
		t.Errorf("params[page] = %q, want 1", r0.Params["page"])
	}

	// Static URL without query
	r1 := results[1]
	if r1.HasQuery {
		t.Error("expected HasQuery=false for static URL")
	}
	if r1.Params != nil {
		t.Error("expected nil params for no-query URL")
	}

	// Search URL with different params
	r2 := results[2]
	if r2.Params["q"] != "test" {
		t.Errorf("params[q] = %q, want test", r2.Params["q"])
	}
	if r2.Params["lang"] != "en" {
		t.Errorf("params[lang] = %q, want en", r2.Params["lang"])
	}
}

func TestParseGauOutput_Empty(t *testing.T) {
	results, errs := ParseGauOutput(strings.NewReader(""))
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestParseGauOutput_InvalidURL(t *testing.T) {
	input := `https://valid.com/path
:not-a-url
https://another.com/path?q=1
`
	results, errs := ParseGauOutput(strings.NewReader(input))

	if len(results) != 2 {
		t.Errorf("expected 2 valid results, got %d", len(results))
	}
	if len(errs) != 1 {
		t.Errorf("expected 1 error, got %d", len(errs))
	}
}

func TestParseGauOutputBytes(t *testing.T) {
	data := []byte("https://example.com/path?key=value\n")
	results, _ := ParseGauOutputBytes(data)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].HasQuery {
		t.Error("expected HasQuery=true")
	}
}
