package parser

import (
	"strings"
	"testing"
)

func TestParseFfufOutput(t *testing.T) {
	input := `{"url":"https://example.com/admin","status":200,"length":1234,"words":56,"lines":20,"input":{"FUZZ":"admin"}}
{"url":"https://example.com/login","status":302,"length":0,"words":0,"lines":0,"input":{"FUZZ":"login"}}`
	results, errs := ParseFfufOutput(strings.NewReader(input))
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	r0 := results[0]
	if r0.URL != "https://example.com/admin" {
		t.Errorf("url: got %q", r0.URL)
	}
	if r0.StatusCode != 200 {
		t.Errorf("status: got %d, want 200", r0.StatusCode)
	}
	if r0.Length != 1234 {
		t.Errorf("length: got %d, want 1234", r0.Length)
	}
	if r0.Words != 56 {
		t.Errorf("words: got %d, want 56", r0.Words)
	}
	if r0.Lines != 20 {
		t.Errorf("lines: got %d, want 20", r0.Lines)
	}
	if r0.Input["FUZZ"] != "admin" {
		t.Errorf("input[fuzz]: got %q", r0.Input["FUZZ"])
	}

	r1 := results[1]
	if r1.StatusCode != 302 {
		t.Errorf("status: got %d, want 302", r1.StatusCode)
	}
}

func TestParseFfufOutput_EmptyInput(t *testing.T) {
	results, errs := ParseFfufOutput(strings.NewReader(""))
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
	if len(errs) != 0 {
		t.Errorf("expected 0 errors, got %d", len(errs))
	}
}

func TestParseFfufOutput_InvalidJSON(t *testing.T) {
	input := `not valid json`
	results, errs := ParseFfufOutput(strings.NewReader(input))
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
	if len(errs) != 1 {
		t.Errorf("expected 1 error, got %d", len(errs))
	}
}

func TestParseFfufOutput_PartialFields(t *testing.T) {
	input := `{"url":"https://example.com/partial","status":200}`
	results, errs := ParseFfufOutput(strings.NewReader(input))
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.URL != "https://example.com/partial" {
		t.Errorf("url: got %q", r.URL)
	}
	if r.StatusCode != 200 {
		t.Errorf("status: got %d", r.StatusCode)
	}
	if r.Length != 0 {
		t.Errorf("length: expected 0 default, got %d", r.Length)
	}
}

func TestParseFfufOutput_EmptyLinesBetween(t *testing.T) {
	input := `{"url":"https://a.com/1","status":200}

{"url":"https://a.com/2","status":404}
`
	results, errs := ParseFfufOutput(strings.NewReader(input))
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}
