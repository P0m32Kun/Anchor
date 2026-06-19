package resolve

import (
	"context"
	"strings"
	"testing"
	"time"

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

func TestExtractAllIPs_MultipleRecordsNoOverlap(t *testing.T) {
	records := []models.DNSRecord{
		{Domain: "a.com", IPs: []string{"1.1.1.1"}},
		{Domain: "b.com", IPs: []string{"2.2.2.2"}},
		{Domain: "c.com", IPs: []string{"3.3.3.3"}},
	}
	ips := ExtractAllIPs(records)
	if len(ips) != 3 {
		t.Errorf("len = %d, want 3", len(ips))
	}
}

func TestExtractAllIPs_EmptyIPsSliceInRecord(t *testing.T) {
	records := []models.DNSRecord{
		{Domain: "a.com", IPs: nil},
		{Domain: "b.com", IPs: []string{"1.1.1.1"}},
	}
	ips := ExtractAllIPs(records)
	if len(ips) != 1 {
		t.Errorf("len = %d, want 1", len(ips))
	}
}

func TestExtractAllIPs_AllSameIP(t *testing.T) {
	records := []models.DNSRecord{
		{Domain: "a.com", IPs: []string{"1.1.1.1"}},
		{Domain: "b.com", IPs: []string{"1.1.1.1"}},
		{Domain: "c.com", IPs: []string{"1.1.1.1"}},
	}
	ips := ExtractAllIPs(records)
	if len(ips) != 1 {
		t.Errorf("len = %d, want 1 (all same IP)", len(ips))
	}
}

func TestExtractAllIPs_EmptyRecordsSlice(t *testing.T) {
	ips := ExtractAllIPs([]models.DNSRecord{})
	if len(ips) != 0 {
		t.Errorf("len = %d, want 0", len(ips))
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

func TestExtractCDNDomains_CDNIPNotInRecords(t *testing.T) {
	records := []models.DNSRecord{
		{Domain: "a.com", IPs: []string{"1.1.1.1"}},
	}
	cdnResults := []models.CDNResult{
		{IP: "9.9.9.9", IsCDN: true}, // CDN IP not in any record
	}

	domains := ExtractCDNDomains(records, cdnResults)
	if len(domains) != 0 {
		t.Errorf("len = %d, want 0 (CDN IP not in records)", len(domains))
	}
}

func TestExtractCDNDomains_EmptyCDNResults(t *testing.T) {
	records := []models.DNSRecord{
		{Domain: "a.com", IPs: []string{"1.1.1.1"}},
	}
	domains := ExtractCDNDomains(records, []models.CDNResult{})
	if len(domains) != 0 {
		t.Errorf("len = %d, want 0", len(domains))
	}
}

func TestExtractCDNDomains_EmptyRecords(t *testing.T) {
	cdnResults := []models.CDNResult{
		{IP: "1.1.1.1", IsCDN: true},
	}
	domains := ExtractCDNDomains([]models.DNSRecord{}, cdnResults)
	if len(domains) != 0 {
		t.Errorf("len = %d, want 0", len(domains))
	}
}

func TestExtractCDNDomains_MixedCDNAndNonCDN(t *testing.T) {
	records := []models.DNSRecord{
		{Domain: "a.com", IPs: []string{"1.1.1.1"}},
		{Domain: "b.com", IPs: []string{"2.2.2.2"}},
		{Domain: "c.com", IPs: []string{"3.3.3.3"}},
	}
	cdnResults := []models.CDNResult{
		{IP: "1.1.1.1", IsCDN: true, Provider: "cloudflare"},
		{IP: "2.2.2.2", IsCDN: false},
		{IP: "3.3.3.3", IsCDN: true, Provider: "akamai"},
	}

	domains := ExtractCDNDomains(records, cdnResults)
	if len(domains) != 2 {
		t.Fatalf("len = %d, want 2", len(domains))
	}
	// Order follows records iteration order
	if domains[0] != "a.com" || domains[1] != "c.com" {
		t.Errorf("got %v, want [a.com c.com]", domains)
	}
}

func TestExtractCDNDomains_PartialCDNOverlap(t *testing.T) {
	// Domain has 2 IPs, only 1 is CDN
	records := []models.DNSRecord{
		{Domain: "a.com", IPs: []string{"1.1.1.1", "2.2.2.2"}},
	}
	cdnResults := []models.CDNResult{
		{IP: "1.1.1.1", IsCDN: true},
		{IP: "2.2.2.2", IsCDN: false},
	}

	domains := ExtractCDNDomains(records, cdnResults)
	if len(domains) != 1 {
		t.Errorf("len = %d, want 1 (partial overlap still counts)", len(domains))
	}
}

// --- NewResolver ---

func TestNewResolver_Defaults(t *testing.T) {
	r := NewResolver()
	if r == nil {
		t.Fatal("NewResolver returned nil")
	}

	if len(r.servers) != 4 {
		t.Errorf("servers count = %d, want 4", len(r.servers))
	}

	wantServers := []string{"8.8.8.8:53", "1.1.1.1:53", "223.5.5.5:53", "119.29.29.29:53"}
	for i, want := range wantServers {
		if r.servers[i] != want {
			t.Errorf("servers[%d] = %q, want %q", i, r.servers[i], want)
		}
	}

	if r.timeout != 5*time.Second {
		t.Errorf("timeout = %v, want 5s", r.timeout)
	}

	if r.parallel != 10 {
		t.Errorf("parallel = %d, want 10", r.parallel)
	}
}

// --- WithTimeout ---

func TestWithTimeout_SetsValue(t *testing.T) {
	r := NewResolver()
	r.WithTimeout(3 * time.Second)
	if r.timeout != 3*time.Second {
		t.Errorf("timeout = %v, want 3s", r.timeout)
	}
}

func TestWithTimeout_ReturnsSelf(t *testing.T) {
	r := NewResolver()
	got := r.WithTimeout(1 * time.Second)
	if got != r {
		t.Errorf("WithTimeout did not return same pointer")
	}
}

func TestWithTimeout_ZeroValue(t *testing.T) {
	r := NewResolver()
	r.WithTimeout(0)
	if r.timeout != 0 {
		t.Errorf("timeout = %v, want 0", r.timeout)
	}
}

func TestWithTimeout_Chaining(t *testing.T) {
	r := NewResolver()
	result := r.WithTimeout(2 * time.Second).WithParallel(3)
	if result != r {
		t.Error("chaining should return same resolver")
	}
	if r.timeout != 2*time.Second {
		t.Errorf("timeout = %v, want 2s", r.timeout)
	}
	if r.parallel != 3 {
		t.Errorf("parallel = %d, want 3", r.parallel)
	}
}

// --- WithParallel ---

func TestWithParallel_SetsValue(t *testing.T) {
	r := NewResolver()
	r.WithParallel(5)
	if r.parallel != 5 {
		t.Errorf("parallel = %d, want 5", r.parallel)
	}
}

func TestWithParallel_MinOne(t *testing.T) {
	r := NewResolver()
	r.WithParallel(0)
	if r.parallel != 1 {
		t.Errorf("parallel = %d, want 1 (min clamp)", r.parallel)
	}

	r.WithParallel(-5)
	if r.parallel != 1 {
		t.Errorf("parallel = %d, want 1 (negative clamp)", r.parallel)
	}
}

func TestWithParallel_ReturnsSelf(t *testing.T) {
	r := NewResolver()
	got := r.WithParallel(3)
	if got != r {
		t.Errorf("WithParallel did not return same pointer")
	}
}

func TestWithParallel_NegativeOne(t *testing.T) {
	r := NewResolver()
	r.WithParallel(-1)
	if r.parallel != 1 {
		t.Errorf("parallel = %d, want 1 (clamp from -1)", r.parallel)
	}
}

func TestWithParallel_LargeValue(t *testing.T) {
	r := NewResolver()
	r.WithParallel(1000)
	if r.parallel != 1000 {
		t.Errorf("parallel = %d, want 1000", r.parallel)
	}
}

// --- Resolve ---

func TestResolve_EmptyDomains(t *testing.T) {
	r := NewResolver()
	ctx := context.Background()

	records, err := r.Resolve(ctx, nil)
	if err != nil {
		t.Errorf("unexpected error for nil domains: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("got %d records, want 0", len(records))
	}

	records, err = r.Resolve(ctx, []string{})
	if err != nil {
		t.Errorf("unexpected error for empty domains: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("got %d records, want 0", len(records))
	}
}

func TestResolve_BlankDomainsSkipped(t *testing.T) {
	r := NewResolver()
	ctx := context.Background()

	records, err := r.Resolve(ctx, []string{"", "  ", "\t"})
	if err != nil {
		t.Errorf("unexpected error for blank domains: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("got %d records, want 0 (blanks should be skipped)", len(records))
	}
}

func TestResolve_ValidDomain(t *testing.T) {
	r := NewResolver().WithTimeout(10 * time.Second)
	ctx := context.Background()

	records, err := r.Resolve(ctx, []string{"example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("got %d records, want 1", len(records))
	}

	rec := records[0]
	if rec.Domain != "example.com" {
		t.Errorf("domain = %q, want example.com", rec.Domain)
	}
	if len(rec.IPs) == 0 {
		t.Error("expected at least one IP for example.com")
	}
	for _, ip := range rec.IPs {
		if ip == "" {
			t.Error("empty IP in result")
		}
	}
}

func TestResolve_InvalidDomain(t *testing.T) {
	r := NewResolver().WithTimeout(5 * time.Second)
	ctx := context.Background()

	records, err := r.Resolve(ctx, []string{"this-domain-definitely-does-not-exist-abc123.invalid"})
	if err == nil {
		t.Error("expected error for invalid domain, got nil")
	}
	if len(records) != 0 {
		t.Errorf("got %d records, want 0 for invalid domain", len(records))
	}
}

func TestResolve_ContextCancellation(t *testing.T) {
	r := NewResolver().WithTimeout(10 * time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	records, err := r.Resolve(ctx, []string{"example.com"})
	if err == nil {
		t.Error("expected error for cancelled context, got nil")
	}
	if len(records) != 0 {
		t.Errorf("got %d records, want 0 for cancelled context", len(records))
	}
}

func TestResolve_MixedValidAndInvalid(t *testing.T) {
	r := NewResolver().WithTimeout(10 * time.Second)
	ctx := context.Background()

	// One valid, one invalid — should return the valid record and not error
	// (since len(records) > 0, the error is suppressed per the code logic)
	records, err := r.Resolve(ctx, []string{
		"example.com",
		"this-domain-definitely-does-not-exist-abc123.invalid",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 1 {
		t.Errorf("got %d records, want 1 (only valid domain)", len(records))
	}
}

func TestResolve_ParallelClamped(t *testing.T) {
	// Verify that WithParallel(0) clamps to 1 and still works
	r := NewResolver().WithParallel(0).WithTimeout(10 * time.Second)
	ctx := context.Background()

	records, err := r.Resolve(ctx, []string{"example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 1 {
		t.Errorf("got %d records, want 1", len(records))
	}
}

func TestResolve_TimeoutApplied(t *testing.T) {
	// Very short timeout — should still work for a fast DNS or fail gracefully
	r := NewResolver().WithTimeout(1 * time.Millisecond)
	ctx := context.Background()

	// We don't assert a specific outcome — DNS may or may not respond in 1ms.
	// Just verify no panic and the function returns.
	_, _ = r.Resolve(ctx, []string{"example.com"})
}

func TestResolve_MultipleValidDomains(t *testing.T) {
	r := NewResolver().WithTimeout(10 * time.Second).WithParallel(2)
	ctx := context.Background()

	records, err := r.Resolve(ctx, []string{"example.com", "google.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) < 1 {
		t.Errorf("got %d records, want >= 1", len(records))
	}

	domains := make(map[string]bool)
	for _, rec := range records {
		domains[rec.Domain] = true
	}
	if !domains["example.com"] && !domains["google.com"] {
		t.Error("neither example.com nor google.com in results")
	}
}

func TestResolve_AllInvalidDomains(t *testing.T) {
	r := NewResolver().WithTimeout(3 * time.Second)
	ctx := context.Background()

	records, err := r.Resolve(ctx, []string{
		"invalid1-doesnotexist.invalid",
		"invalid2-doesnotexist.invalid",
	})
	if err == nil {
		t.Error("expected error when all domains fail")
	}
	if len(records) != 0 {
		t.Errorf("got %d records, want 0", len(records))
	}
}

func TestResolve_MixOfBlanksAndValid(t *testing.T) {
	r := NewResolver().WithTimeout(10 * time.Second)
	ctx := context.Background()

	records, err := r.Resolve(ctx, []string{"", "example.com", "  ", "\t"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 1 {
		t.Errorf("got %d records, want 1 (only example.com)", len(records))
	}
}

func TestResolve_MixOfBlanksAndInvalid(t *testing.T) {
	r := NewResolver().WithTimeout(3 * time.Second)
	ctx := context.Background()

	// All blanks are skipped, invalid domain fails — should get error
	records, err := r.Resolve(ctx, []string{"", "invalid-domain-abc.invalid", "  "})
	if err == nil {
		t.Error("expected error (blanks skipped, invalid domain fails)")
	}
	if len(records) != 0 {
		t.Errorf("got %d records, want 0", len(records))
	}
}

func TestResolve_ContextTimeout(t *testing.T) {
	r := NewResolver().WithTimeout(10 * time.Second)

	// Very short context timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// May or may not resolve depending on timing, just ensure no panic
	_, _ = r.Resolve(ctx, []string{"example.com"})
}

func TestResolve_ConcurrentAccess(t *testing.T) {
	// Verify no race conditions with concurrent Resolve calls
	r := NewResolver().WithTimeout(10 * time.Second).WithParallel(5)
	ctx := context.Background()

	done := make(chan struct{}, 3)
	for i := 0; i < 3; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			_, _ = r.Resolve(ctx, []string{"example.com"})
		}()
	}

	for i := 0; i < 3; i++ {
		select {
		case <-done:
		case <-time.After(15 * time.Second):
			t.Fatal("concurrent Resolve timed out")
		}
	}
}

// --- resolveOne (via Resolve, since it's unexported) ---

func TestResolveOne_CoverageViaResolve(t *testing.T) {
	// This test exercises resolveOne through Resolve with a single domain.
	// It verifies the record structure (IPs, CNAMEs, Resolver fields).
	r := NewResolver().WithTimeout(10 * time.Second)
	ctx := context.Background()

	records, err := r.Resolve(ctx, []string{"example.com"})
	if err != nil {
		t.Fatalf("resolveOne error: %v", err)
	}
	if len(records) == 0 {
		t.Fatal("no records returned")
	}

	rec := records[0]
	if rec.Domain != "example.com" {
		t.Errorf("domain = %q, want example.com", rec.Domain)
	}
	if rec.Resolver != "8.8.8.8:53" {
		t.Errorf("resolver = %q, want 8.8.8.8:53", rec.Resolver)
	}
	if len(rec.IPs) == 0 {
		t.Error("expected at least one IP")
	}
	// CNAMEs may or may not be present; just verify the slice is not nil
	if rec.CNAMEs == nil {
		t.Error("CNAMEs should be initialized (not nil)")
	}
}

func TestResolveOne_CNAMEChain(t *testing.T) {
	// www.github.com typically has a CNAME chain
	r := NewResolver().WithTimeout(10 * time.Second)
	ctx := context.Background()

	records, err := r.Resolve(ctx, []string{"www.github.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) == 0 {
		t.Fatal("no records returned")
	}

	rec := records[0]
	if len(rec.IPs) == 0 {
		t.Error("expected at least one IP for www.github.com")
	}
	// www.github.com typically has CNAMEs
	if len(rec.CNAMEs) == 0 {
		t.Log("warning: www.github.com returned no CNAMEs (may vary by DNS)")
	}
	// Verify CNAMEs don't have trailing dots
	for _, cname := range rec.CNAMEs {
		if strings.HasSuffix(cname, ".") {
			t.Errorf("CNAME %q has trailing dot", cname)
		}
	}
}

func TestResolveOne_IPv4Only(t *testing.T) {
	// Verify that only IPv4 addresses are returned
	r := NewResolver().WithTimeout(10 * time.Second)
	ctx := context.Background()

	records, err := r.Resolve(ctx, []string{"example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) == 0 {
		t.Fatal("no records returned")
	}

	for _, ip := range records[0].IPs {
		if strings.Contains(ip, ":") {
			t.Errorf("found IPv6 address %q in results (should be IPv4 only)", ip)
		}
	}
}

func TestResolveOne_NonexistentDomainError(t *testing.T) {
	// resolveOne should return error for domains that can't be resolved
	r := NewResolver().WithTimeout(3 * time.Second)
	ctx := context.Background()

	records, err := r.Resolve(ctx, []string{"nx-domain-xyz.invalid"})
	if err == nil {
		t.Error("expected error for nonexistent domain")
	}
	if len(records) != 0 {
		t.Errorf("got %d records, want 0", len(records))
	}
}

func TestResolveOne_RecordFieldsInitialized(t *testing.T) {
	// Verify that the DNSRecord fields are properly initialized
	r := NewResolver().WithTimeout(10 * time.Second)
	ctx := context.Background()

	records, err := r.Resolve(ctx, []string{"google.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) == 0 {
		t.Fatal("no records returned")
	}

	rec := records[0]
	if rec.Domain == "" {
		t.Error("domain should not be empty")
	}
	if rec.Resolver == "" {
		t.Error("resolver should not be empty")
	}
	if rec.IPs == nil {
		t.Error("IPs should be initialized (not nil)")
	}
	if rec.CNAMEs == nil {
		t.Error("CNAMEs should be initialized (not nil)")
	}
}

func TestResolveOne_MultipleDomainsDifferentResults(t *testing.T) {
	// Resolve two domains and verify each has its own record
	r := NewResolver().WithTimeout(10 * time.Second).WithParallel(2)
	ctx := context.Background()

	records, err := r.Resolve(ctx, []string{"example.com", "google.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	domainToRecord := make(map[string]models.DNSRecord)
	for _, rec := range records {
		domainToRecord[rec.Domain] = rec
	}

	if rec, ok := domainToRecord["example.com"]; ok {
		if len(rec.IPs) == 0 {
			t.Error("example.com should have IPs")
		}
	}
	if rec, ok := domainToRecord["google.com"]; ok {
		if len(rec.IPs) == 0 {
			t.Error("google.com should have IPs")
		}
	}
}

func TestResolve_ErrorAggregation(t *testing.T) {
	// All domains fail — error should be the first one (errs[0])
	r := NewResolver().WithTimeout(3 * time.Second)
	ctx := context.Background()

	_, err := r.Resolve(ctx, []string{
		"bad1.invalid",
		"bad2.invalid",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	// Error should contain "resolve" prefix
	if !strings.Contains(err.Error(), "resolve") {
		t.Errorf("error %q should contain 'resolve'", err.Error())
	}
}
