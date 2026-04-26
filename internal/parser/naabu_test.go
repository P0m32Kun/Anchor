package parser

import (
	"strings"
	"testing"
)

func TestParseNaabuJSONL(t *testing.T) {
	input := `{"host":"sub.example.com","port":443,"ip":"93.184.216.34"}
{"host":"sub.example.com","port":80,"ip":"93.184.216.34"}
invalid json
{"host":"sub.example.com","ip":"93.184.216.34"}
{"port":443,"ip":"93.184.216.34"}
`

	results, errs := ParseNaabu(strings.NewReader(input))

	// ip-only entries are valid (ip assets with ports)
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if results[0].Host != "sub.example.com" || results[0].Port != 443 {
		t.Errorf("unexpected result[0]: %+v", results[0])
	}
	if results[0].IP != "93.184.216.34" {
		t.Errorf("expected ip 93.184.216.34, got %s", results[0].IP)
	}
	if results[1].Port != 80 {
		t.Errorf("expected port 80, got %d", results[1].Port)
	}
	if results[2].IP != "93.184.216.34" || results[2].Port != 443 {
		t.Errorf("unexpected result[2]: %+v", results[2])
	}

	if len(errs) != 2 {
		t.Fatalf("expected 2 parse errors, got %d", len(errs))
	}
}

func TestParseNaabuCSV(t *testing.T) {
	input := `sub.example.com,93.184.216.34,443
sub.example.com,93.184.216.34,80
sub.example.com,93.184.216.34
,93.184.216.34,443
sub.example.com,93.184.216.34,abc
`

	results, errs := ParseNaabu(strings.NewReader(input))

	// ip-only entries are valid
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if results[0].Host != "sub.example.com" || results[0].Port != 443 {
		t.Errorf("unexpected result[0]: %+v", results[0])
	}
	if results[1].Port != 80 {
		t.Errorf("expected port 80, got %d", results[1].Port)
	}
	if results[2].IP != "93.184.216.34" || results[2].Port != 443 {
		t.Errorf("unexpected result[2]: %+v", results[2])
	}

	if len(errs) != 2 {
		t.Fatalf("expected 2 parse errors, got %d", len(errs))
	}
}

func TestParseNaabuEmpty(t *testing.T) {
	results, errs := ParseNaabu(strings.NewReader(""))
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
	if len(errs) != 0 {
		t.Errorf("expected 0 errors, got %d", len(errs))
	}
}

func TestParseNaabuPortString(t *testing.T) {
	input := `{"host":"a.com","port":"8080","ip":"1.2.3.4"}`
	results, errs := ParseNaabu(strings.NewReader(input))
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Port != 8080 {
		t.Errorf("expected port 8080, got %d", results[0].Port)
	}
	if len(errs) != 0 {
		t.Errorf("expected 0 errors, got %d", len(errs))
	}
}
