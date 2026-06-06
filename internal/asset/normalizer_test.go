package asset

import (
	"testing"
)

func TestNormalizeDomain(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"Example.COM", "example.com"},
		{"  www.Example.COM  ", "example.com"},
		{"sub.www.example.com", "sub.www.example.com"},
		{"www.example.com", "example.com"},
	}
	for _, c := range cases {
		got := NormalizeDomain(c.in)
		if got != c.want {
			t.Errorf("NormalizeDomain(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestNormalizeURL(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"https://Example.COM/", "https://example.com/"},
		{"https://Example.COM:443/path/", "https://example.com/path"},
		{"http://Example.COM:8080/", "http://example.com:8080/"},
		{"https://Example.COM/path?a=1#frag", "https://example.com/path"},
		{"https://www.Example.COM:443/", "https://example.com/"},
		{"http://www.example.com:80/path", "http://example.com/path"},
	}
	for _, c := range cases {
		got := NormalizeURL(c.in)
		if got != c.want {
			t.Errorf("NormalizeURL(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestNormalizeIP(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"192.168.001.001", "192.168.1.1"},
		{"  10.0.0.1  ", "10.0.0.1"},
		{"::1", "::1"},
		{"0:0:0:0:0:0:0:1", "::1"},
	}
	for _, c := range cases {
		got := NormalizeIP(c.in)
		if got != c.want {
			t.Errorf("NormalizeIP(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestNormalizeCIDR(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"172.30.0.0/24", "172.30.0.0/24"},
		{"192.168.1.0/24", "192.168.1.0/24"},
		{"10.0.0.0/8", "10.0.0.0/8"},
		{"  172.30.0.0/24  ", "172.30.0.0/24"},
		{"invalid", "invalid"},
	}
	for _, c := range cases {
		got := NormalizeCIDR(c.in)
		if got != c.want {
			t.Errorf("NormalizeCIDR(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestInferStorageType_IP(t *testing.T) {
	if got := InferStorageType("172.31.0.13"); got != "ip" {
		t.Fatalf("InferStorageType = %q, want ip", got)
	}
}

func TestNormalize_IPNotAliasedAsDomain(t *testing.T) {
	ipNorm := Normalize("ip", "172.31.0.13")
	domainNorm := Normalize("domain", "172.31.0.13")
	if ipNorm != "172.31.0.13" {
		t.Fatalf("ip normalize = %q", ipNorm)
	}
	if domainNorm != "172.31.0.13" {
		t.Fatalf("domain normalize should coerce to ip, got %q", domainNorm)
	}
}

func TestExtractHostFromURL(t *testing.T) {
	if got := ExtractHostFromURL("https://sub.example.com:8443/path"); got != "sub.example.com" {
		t.Errorf("ExtractHostFromURL = %q, want sub.example.com", got)
	}
}

func TestExtractPortFromURL(t *testing.T) {
	if got := ExtractPortFromURL("https://sub.example.com:8443/path"); got != 8443 {
		t.Errorf("ExtractPortFromURL = %d, want 8443", got)
	}
	if got := ExtractPortFromURL("https://sub.example.com/path"); got != 0 {
		t.Errorf("ExtractPortFromURL = %d, want 0", got)
	}
}
