package parser

import (
	"strings"
	"testing"
)

func TestParseDNSx(t *testing.T) {
	input := `{"host":"example.com","a":["93.184.216.34"],"aaaa":["2606:2800:220:1:248:1893:25c8:1946"],"cname":[],"mx":[],"ns":[],"txt":[],"ttl":3600}
{"host":"sub.example.com","a":["10.0.0.1"],"cname":["example.com"],"ttl":300}`
	results, errs := ParseDNSx(strings.NewReader(input))
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	r0 := results[0]
	if r0.Host != "example.com" {
		t.Errorf("host: got %q, want example.com", r0.Host)
	}
	if len(r0.A) != 1 || r0.A[0] != "93.184.216.34" {
		t.Errorf("A records: got %v", r0.A)
	}
	if len(r0.AAAA) != 1 || r0.AAAA[0] != "2606:2800:220:1:248:1893:25c8:1946" {
		t.Errorf("AAAA records: got %v", r0.AAAA)
	}
	if r0.TTL != 3600 {
		t.Errorf("TTL: got %d, want 3600", r0.TTL)
	}

	r1 := results[1]
	if r1.Host != "sub.example.com" {
		t.Errorf("host: got %q", r1.Host)
	}
	if len(r1.CNAME) != 1 || r1.CNAME[0] != "example.com" {
		t.Errorf("CNAME: got %v", r1.CNAME)
	}
}

func TestParseDNSx_EmptyInput(t *testing.T) {
	results, errs := ParseDNSx(strings.NewReader(""))
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
	if len(errs) != 0 {
		t.Errorf("expected 0 errors, got %d", len(errs))
	}
}

func TestParseDNSx_InvalidJSON(t *testing.T) {
	input := `not valid json`
	results, errs := ParseDNSx(strings.NewReader(input))
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
	if len(errs) != 1 {
		t.Errorf("expected 1 error, got %d", len(errs))
	}
}

func TestParseDNSx_MissingHost(t *testing.T) {
	input := `{"a":["1.2.3.4"],"ttl":60}`
	results, errs := ParseDNSx(strings.NewReader(input))
	if len(results) != 0 {
		t.Errorf("expected 0 results (missing host), got %d", len(results))
	}
	if len(errs) != 1 {
		t.Errorf("expected 1 error, got %d", len(errs))
	}
}

func TestParseDNSxOutput(t *testing.T) {
	input := `{"host":"a.com","a":["1.1.1.1"],"ttl":60}
{"host":"b.com","a":["2.2.2.2"],"ttl":120}`
	m := ParseDNSxOutput(strings.NewReader(input))
	if len(m) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(m))
	}
	if _, ok := m["a.com"]; !ok {
		t.Error("missing a.com in output map")
	}
	if _, ok := m["b.com"]; !ok {
		t.Error("missing b.com in output map")
	}
}

func TestExtractDNSxIPs(t *testing.T) {
	rec := DNSxResult{
		Host: "test.com",
		A:    []string{"1.2.3.4", "5.6.7.8"},
		AAAA: []string{"::1", "1.2.3.4"}, // 1.2.3.4 is dup with A
	}
	ips := ExtractDNSxIPs(rec)
	if len(ips) != 3 {
		t.Fatalf("expected 3 unique IPs, got %d: %v", len(ips), ips)
	}
	expected := map[string]bool{"1.2.3.4": true, "5.6.7.8": true, "::1": true}
	for _, ip := range ips {
		if !expected[ip] {
			t.Errorf("unexpected IP %q", ip)
		}
	}
}

func TestExtractDNSxIPs_Empty(t *testing.T) {
	rec := DNSxResult{Host: "empty.com"}
	ips := ExtractDNSxIPs(rec)
	if len(ips) != 0 {
		t.Errorf("expected 0 IPs, got %d", len(ips))
	}
}

func TestExtractDNSxIPs_SkipEmptyStrings(t *testing.T) {
	rec := DNSxResult{
		A:    []string{"", "1.2.3.4", ""},
		AAAA: []string{"", "::1"},
	}
	ips := ExtractDNSxIPs(rec)
	if len(ips) != 2 {
		t.Errorf("expected 2 IPs (skip empty), got %d: %v", len(ips), ips)
	}
}

func TestExtractDNSxCNAMEs(t *testing.T) {
	rec := DNSxResult{
		CNAME: []string{"alias.example.com", "other.example.com"},
	}
	cnames := ExtractDNSxCNAMEs(rec)
	if len(cnames) != 2 {
		t.Fatalf("expected 2 CNAMEs, got %d", len(cnames))
	}
	if cnames[0] != "alias.example.com" {
		t.Errorf("cname[0] = %q", cnames[0])
	}
}

func TestExtractDNSxCNAMEs_Empty(t *testing.T) {
	rec := DNSxResult{CNAME: nil}
	cnames := ExtractDNSxCNAMEs(rec)
	if len(cnames) != 0 {
		t.Errorf("expected 0 CNAMEs, got %d", len(cnames))
	}
}

func TestExtractDNSxCNAMEs_SkipEmptyStrings(t *testing.T) {
	rec := DNSxResult{CNAME: []string{"", "real.example.com", ""}}
	cnames := ExtractDNSxCNAMEs(rec)
	if len(cnames) != 1 {
		t.Errorf("expected 1 CNAME, got %d: %v", len(cnames), cnames)
	}
}

func TestJoinDNSxIPs(t *testing.T) {
	rec := DNSxResult{
		A:    []string{"1.1.1.1"},
		AAAA: []string{"::1"},
	}
	joined := JoinDNSxIPs(rec)
	if joined != "1.1.1.1,::1" {
		t.Errorf("JoinDNSxIPs = %q", joined)
	}
}

func TestJoinDNSxIPs_Empty(t *testing.T) {
	rec := DNSxResult{Host: "empty.com"}
	joined := JoinDNSxIPs(rec)
	if joined != "" {
		t.Errorf("expected empty string, got %q", joined)
	}
}
