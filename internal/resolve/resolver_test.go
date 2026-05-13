package resolve

import (
	"testing"

	"github.com/P0m32Kun/Anchor/internal/models"
)

// --- ExtractAllIPs ---

func TestExtractAllIPs_Dedup(t *testing.T) {
	records := []models.DNSRecord{
		{Domain: "a.com", IPs: []string{"1.1.1.1", "2.2.2.2"}},
		{Domain: "b.com", IPs: []string{"2.2.2.2", "3.3.3.3"}},
	}

	ips := ExtractAllIPs(records)
	if len(ips) != 3 {
		t.Errorf("len = %d, want 3 (dedup)", len(ips))
	}

	seen := map[string]bool{}
	for _, ip := range ips {
		seen[ip] = true
	}
	for _, want := range []string{"1.1.1.1", "2.2.2.2", "3.3.3.3"} {
		if !seen[want] {
			t.Errorf("missing %q", want)
		}
	}
}

func TestExtractAllIPs_Empty(t *testing.T) {
	ips := ExtractAllIPs(nil)
	if ips != nil {
		t.Errorf("expected nil for nil input, got %v", ips)
	}
}

func TestExtractAllIPs_NoIPs(t *testing.T) {
	records := []models.DNSRecord{
		{Domain: "a.com", IPs: []string{}},
	}
	ips := ExtractAllIPs(records)
	if len(ips) != 0 {
		t.Errorf("len = %d, want 0", len(ips))
	}
}

func TestExtractAllIPs_SingleRecord(t *testing.T) {
	records := []models.DNSRecord{
		{Domain: "a.com", IPs: []string{"10.0.0.1"}},
	}
	ips := ExtractAllIPs(records)
	if len(ips) != 1 || ips[0] != "10.0.0.1" {
		t.Errorf("got %v, want [10.0.0.1]", ips)
	}
}

// --- ExtractCDNDomains ---

func TestExtractCDNDomains_Basic(t *testing.T) {
	records := []models.DNSRecord{
		{Domain: "a.com", IPs: []string{"1.1.1.1", "2.2.2.2"}},
		{Domain: "b.com", IPs: []string{"3.3.3.3"}},
	}
	cdnResults := []models.CDNResult{
		{IP: "1.1.1.1", IsCDN: true, Provider: "cloudflare"},
		{IP: "3.3.3.3", IsCDN: false},
	}

	domains := ExtractCDNDomains(records, cdnResults)
	if len(domains) != 1 || domains[0] != "a.com" {
		t.Errorf("got %v, want [a.com]", domains)
	}
}

func TestExtractCDNDomains_MultipleCDN(t *testing.T) {
	records := []models.DNSRecord{
		{Domain: "a.com", IPs: []string{"1.1.1.1"}},
		{Domain: "b.com", IPs: []string{"2.2.2.2"}},
	}
	cdnResults := []models.CDNResult{
		{IP: "1.1.1.1", IsCDN: true},
		{IP: "2.2.2.2", IsCDN: true},
	}

	domains := ExtractCDNDomains(records, cdnResults)
	if len(domains) != 2 {
		t.Errorf("len = %d, want 2", len(domains))
	}
}

func TestExtractCDNDomains_NoCDN(t *testing.T) {
	records := []models.DNSRecord{
		{Domain: "a.com", IPs: []string{"1.1.1.1"}},
	}
	cdnResults := []models.CDNResult{
		{IP: "1.1.1.1", IsCDN: false},
	}

	domains := ExtractCDNDomains(records, cdnResults)
	if len(domains) != 0 {
		t.Errorf("len = %d, want 0", len(domains))
	}
}

func TestExtractCDNDomains_EmptyInputs(t *testing.T) {
	domains := ExtractCDNDomains(nil, nil)
	if domains != nil {
		t.Errorf("expected nil, got %v", domains)
	}
}

func TestExtractCDNDomains_DedupDomains(t *testing.T) {
	// Same domain with two IPs, both behind CDN
	records := []models.DNSRecord{
		{Domain: "a.com", IPs: []string{"1.1.1.1", "2.2.2.2"}},
	}
	cdnResults := []models.CDNResult{
		{IP: "1.1.1.1", IsCDN: true},
		{IP: "2.2.2.2", IsCDN: true},
	}

	domains := ExtractCDNDomains(records, cdnResults)
	if len(domains) != 1 {
		t.Errorf("len = %d, want 1 (dedup)", len(domains))
	}
}
