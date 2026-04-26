package parser

import (
	"strings"
	"testing"
)

func TestParseSubfinder(t *testing.T) {
	input := `{"host":"sub.example.com","input":"example.com","source":"anubis"}
{"host":"sub2.example.com","input":"example.com","source":"virustotal"}
invalid json line
{"input":"example.com"}
{"host":"","input":"example.com"}
`

	results, errs := ParseSubfinder(strings.NewReader(input))

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Host != "sub.example.com" {
		t.Errorf("expected host sub.example.com, got %s", results[0].Host)
	}
	if results[0].Source != "anubis" {
		t.Errorf("expected source anubis, got %s", results[0].Source)
	}
	if results[1].Host != "sub2.example.com" {
		t.Errorf("expected host sub2.example.com, got %s", results[1].Host)
	}

	if len(errs) != 3 {
		t.Fatalf("expected 3 parse errors, got %d", len(errs))
	}
}

func TestParseSubfinderEmpty(t *testing.T) {
	results, errs := ParseSubfinder(strings.NewReader(""))
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
	if len(errs) != 0 {
		t.Errorf("expected 0 errors, got %d", len(errs))
	}
}

func TestParseSubfinderExtraFields(t *testing.T) {
	input := `{"host":"a.example.com","input":"example.com","source":"crtsh","extra":"ignored"}`
	results, errs := ParseSubfinder(strings.NewReader(input))
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if len(errs) != 0 {
		t.Errorf("expected 0 errors, got %d", len(errs))
	}
}
