package resolve

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
)

// Resolver performs DNS resolution for domains.
type Resolver struct {
	servers  []string
	timeout  time.Duration
	parallel int
}

// NewResolver creates a new DNS resolver with default settings.
func NewResolver() *Resolver {
	return &Resolver{
		servers: []string{
			"8.8.8.8:53",
			"1.1.1.1:53",
			"223.5.5.5:53",
			"119.29.29.29:53",
		},
		timeout:  5 * time.Second,
		parallel: 10,
	}
}

// WithTimeout sets the timeout for DNS queries.
func (r *Resolver) WithTimeout(d time.Duration) *Resolver {
	r.timeout = d
	return r
}

// WithParallel sets the concurrency limit.
func (r *Resolver) WithParallel(n int) *Resolver {
	r.parallel = n
	if n < 1 {
		r.parallel = 1
	}
	return r
}

// Resolve resolves a list of domains to DNS records.
func (r *Resolver) Resolve(ctx context.Context, domains []string) ([]models.DNSRecord, error) {
	var mu sync.Mutex
	var records []models.DNSRecord
	var errs []error

	sem := make(chan struct{}, r.parallel)
	var wg sync.WaitGroup

	for _, domain := range domains {
		domain = strings.TrimSpace(domain)
		if domain == "" {
			continue
		}

		wg.Add(1)
		go func(d string) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			record, err := r.resolveOne(ctx, d)
			if err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("resolve %s: %w", d, err))
				mu.Unlock()
				return
			}

			mu.Lock()
			records = append(records, record)
			mu.Unlock()
		}(domain)
	}

	wg.Wait()

	if len(records) == 0 && len(errs) > 0 {
		return nil, errs[0]
	}

	return records, nil
}

func (r *Resolver) resolveOne(ctx context.Context, domain string) (models.DNSRecord, error) {
	record := models.DNSRecord{
		Domain: domain,
		IPs:    []string{},
		CNAMEs: []string{},
	}

	// Use the first available resolver
	resolver := r.servers[0]
	record.Resolver = resolver

	// Create a custom resolver with timeout
	resolverAddr := resolver
	rr := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{Timeout: r.timeout}
			return d.DialContext(ctx, network, resolverAddr)
		},
	}

	// Resolve CNAME chain
	cname, err := rr.LookupCNAME(ctx, domain)
	if err == nil && cname != domain && cname != "" {
		record.CNAMEs = append(record.CNAMEs, strings.TrimSuffix(cname, "."))
	}

	// Resolve A records
	ips, err := rr.LookupHost(ctx, domain)
	if err != nil {
		return record, fmt.Errorf("lookup host: %w", err)
	}

	// Filter only IPv4 addresses
	for _, ip := range ips {
		if net.ParseIP(ip) != nil && net.ParseIP(ip).To4() != nil {
			record.IPs = append(record.IPs, ip)
		}
	}

	if len(record.IPs) == 0 {
		return record, fmt.Errorf("no IPv4 addresses found")
	}

	return record, nil
}

// ExtractAllIPs extracts all unique IPs from DNS records.
func ExtractAllIPs(records []models.DNSRecord) []string {
	seen := make(map[string]bool)
	var ips []string
	for _, r := range records {
		for _, ip := range r.IPs {
			if !seen[ip] {
				seen[ip] = true
				ips = append(ips, ip)
			}
		}
	}
	return ips
}

// ExtractCDNDomains extracts domains that are behind CDN from DNS records.
func ExtractCDNDomains(records []models.DNSRecord, cdnResults []models.CDNResult) []string {
	cdnIPSet := make(map[string]bool)
	for _, c := range cdnResults {
		if c.IsCDN {
			cdnIPSet[c.IP] = true
		}
	}

	seen := make(map[string]bool)
	var domains []string
	for _, r := range records {
		for _, ip := range r.IPs {
			if cdnIPSet[ip] && !seen[r.Domain] {
				seen[r.Domain] = true
				domains = append(domains, r.Domain)
			}
		}
	}
	return domains
}
