package parser

import (
	"strings"
	"testing"
)

func TestParseKatanaJSONL(t *testing.T) {
	out := `{"request":{"method":"GET","endpoint":"https://example.com/a"}}
{"request":{"endpoint":"https://cdn.example.com/app.js"}}
`
	urls, errs := ParseKatanaJSONL(strings.NewReader(out))
	if len(errs) != 0 {
		t.Fatalf("errs: %v", errs)
	}
	if len(urls) != 2 {
		t.Fatalf("urls: %v", urls)
	}
}

func TestParseKatanaJSONL_Dedup(t *testing.T) {
	out := `{"request":{"endpoint":"https://example.com/a"}}
{"request":{"endpoint":"https://example.com/a"}}
`
	urls, _ := ParseKatanaJSONL(strings.NewReader(out))
	if len(urls) != 1 {
		t.Fatalf("want 1 url, got %v", urls)
	}
}
