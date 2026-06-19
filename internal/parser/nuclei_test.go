package parser

import (
	"strings"
	"testing"
)

func TestParseNuclei(t *testing.T) {
	input := `{"template-id":"cve-2021-44228","template-path":"cves/2021/CVE-2021-44228.yaml","info":{"name":"Apache Log4j2 Remote Code Execution","severity":"critical","tags":["cve","log4j","rce"]},"host":"https://target.com","matched-at":"https://target.com/api","matcher-name":"log4j-jndi","extracted-results":["jndi:ldap://xxx"],"request":"GET /api HTTP/1.1...","response":"HTTP/1.1 200 OK...","timestamp":"2026-04-26T10:00:00Z"}`

	results, errs := ParseNuclei(strings.NewReader(input))
	if len(errs) > 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	r := results[0]
	if r.TemplateID != "cve-2021-44228" {
		t.Errorf("template-id: got %q, want cve-2021-44228", r.TemplateID)
	}
	if r.Name != "Apache Log4j2 Remote Code Execution" {
		t.Errorf("name: got %q, want 'Apache Log4j2 Remote Code Execution'", r.Name)
	}
	if r.Severity != "critical" {
		t.Errorf("severity: got %q, want critical", r.Severity)
	}
	if r.Host != "https://target.com" {
		t.Errorf("host: got %q", r.Host)
	}
	if r.MatcherName != "log4j-jndi" {
		t.Errorf("matcher-name: got %q", r.MatcherName)
	}
	if len(r.ExtractedResults) != 1 || r.ExtractedResults[0] != "jndi:ldap://xxx" {
		t.Errorf("extracted-results: got %v", r.ExtractedResults)
	}
}

func TestParseNucleiInvalidJSON(t *testing.T) {
	input := `this is not json`
	results, errs := ParseNuclei(strings.NewReader(input))
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
	if len(errs) != 1 {
		t.Errorf("expected 1 parse error, got %d", len(errs))
	}
}

func TestParseNucleiMultipleLines(t *testing.T) {
	input := `{"template-id":"a","info":{"name":"A","severity":"high"},"host":"https://a.com"}
{"template-id":"b","info":{"name":"B","severity":"medium"},"host":"https://b.com"}`

	results, errs := ParseNuclei(strings.NewReader(input))
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].TemplateID != "a" || results[1].TemplateID != "b" {
		t.Errorf("unexpected results")
	}
}

func TestParseNuclei_EmptyInput(t *testing.T) {
	results, errs := ParseNuclei(strings.NewReader(""))
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
	if len(errs) != 0 {
		t.Errorf("expected 0 errors, got %d", len(errs))
	}
}

func TestParseNuclei_EmptyLinesBetweenRecords(t *testing.T) {
	input := `{"template-id":"a","info":{"name":"A","severity":"high"},"host":"https://a.com"}


{"template-id":"b","info":{"name":"B","severity":"medium"},"host":"https://b.com"}
`
	results, errs := ParseNuclei(strings.NewReader(input))
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].TemplateID != "a" || results[1].TemplateID != "b" {
		t.Errorf("unexpected template IDs: %q, %q", results[0].TemplateID, results[1].TemplateID)
	}
}

func TestParseNuclei_MissingInfo(t *testing.T) {
	input := `{"template-id":"test-1","host":"https://example.com","matched-at":"https://example.com/api"}`
	results, errs := ParseNuclei(strings.NewReader(input))
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.TemplateID != "test-1" {
		t.Errorf("template-id: got %q", r.TemplateID)
	}
	if r.Name != "" {
		t.Errorf("expected empty name when info missing, got %q", r.Name)
	}
	if r.Severity != "" {
		t.Errorf("expected empty severity when info missing, got %q", r.Severity)
	}
	if r.Host != "https://example.com" {
		t.Errorf("host: got %q", r.Host)
	}
}

func TestParseNuclei_PartialFields(t *testing.T) {
	input := `{"template-id":"minimal","info":{"name":"Minimal Check","severity":"info"}}`
	results, errs := ParseNuclei(strings.NewReader(input))
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.TemplateID != "minimal" {
		t.Errorf("template-id: got %q", r.TemplateID)
	}
	if r.Name != "Minimal Check" {
		t.Errorf("name: got %q", r.Name)
	}
	if r.Severity != "info" {
		t.Errorf("severity: got %q", r.Severity)
	}
	if r.Host != "" {
		t.Errorf("expected empty host, got %q", r.Host)
	}
	if r.MatchedAt != "" {
		t.Errorf("expected empty matched-at, got %q", r.MatchedAt)
	}
	if r.MatcherName != "" {
		t.Errorf("expected empty matcher-name, got %q", r.MatcherName)
	}
}
