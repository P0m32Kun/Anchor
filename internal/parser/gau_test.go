package parser

import (
	"strings"
	"testing"
)

func TestParseGauOutput_Basic(t *testing.T) {
	input := `https://example.com/api/v1?token=abc123&page=1
https://example.com/static/main.js
https://example.com/search?q=test&lang=en
`
	urls, err := ParseGauOutput(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(urls) != 3 {
		t.Fatalf("expected 3 urls, got %d", len(urls))
	}
	if urls[0] != "https://example.com/api/v1?token=abc123&page=1" {
		t.Errorf("urls[0] = %q", urls[0])
	}
	if urls[1] != "https://example.com/static/main.js" {
		t.Errorf("urls[1] = %q", urls[1])
	}
	if urls[2] != "https://example.com/search?q=test&lang=en" {
		t.Errorf("urls[2] = %q", urls[2])
	}
}

func TestParseGauOutput_Empty(t *testing.T) {
	urls, err := ParseGauOutput(strings.NewReader(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(urls) != 0 {
		t.Errorf("expected 0 urls, got %d", len(urls))
	}
}

func TestParseGauOutput_SkipsEmptyLines(t *testing.T) {
	input := "https://a.com/path\n\n\nhttps://b.com/path\n   \nhttps://c.com/path\n"
	urls, err := ParseGauOutput(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(urls) != 3 {
		t.Fatalf("expected 3 urls, got %d: %v", len(urls), urls)
	}
	if urls[0] != "https://a.com/path" {
		t.Errorf("urls[0] = %q", urls[0])
	}
	if urls[1] != "https://b.com/path" {
		t.Errorf("urls[1] = %q", urls[1])
	}
	if urls[2] != "https://c.com/path" {
		t.Errorf("urls[2] = %q", urls[2])
	}
}

func TestParseGauOutput_DuplicateURLs(t *testing.T) {
	input := "https://a.com/x\nhttps://a.com/x\nhttps://b.com/y\n"
	urls, err := ParseGauOutput(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// ParseGauOutput does NOT dedup — it's a plain line parser
	if len(urls) != 3 {
		t.Fatalf("expected 3 urls (no dedup), got %d", len(urls))
	}
}

func TestParseGauOutputBytes(t *testing.T) {
	data := []byte("https://example.com/path?key=value\n")
	urls, err := ParseGauOutputBytes(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(urls) != 1 {
		t.Fatalf("expected 1 url, got %d", len(urls))
	}
	if urls[0] != "https://example.com/path?key=value" {
		t.Errorf("urls[0] = %q", urls[0])
	}
}

func TestParseGauOutputBytes_Empty(t *testing.T) {
	urls, err := ParseGauOutputBytes([]byte(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(urls) != 0 {
		t.Errorf("expected 0 urls, got %d", len(urls))
	}
}
