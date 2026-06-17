package scheduler

import (
	"context"
	"net"
	"strings"
	"sync"
	"time"
)

// Phase represents a scan phase for per-host mutual exclusion.
type Phase int

const (
	PhaseDiscovery Phase = iota // PASSIVE_*, SUBDOMAIN_ENUM
	PhaseResolve                // DNS_RESOLVE, CDN_CHECK
	PhasePort                   // PORT_SCAN, SERVICE_FINGERPRINT
	PhaseWeb                    // HTTPX_FINGERPRINT
	PhaseCrawl                  // KATANA_CRAWL, SPOOR_SCAN
	PhaseBrute                  // FFUF_BRUTE
	PhaseVuln                   // NUCLEI_SCAN
)

// ActionPhase maps an action string to its Phase.
func ActionPhase(action string) Phase {
	switch action {
	case "PASSIVE_SEARCH", "PASSIVE_CERT", "PASSIVE_URL", "SUBDOMAIN_ENUM":
		return PhaseDiscovery
	case "DNS_RESOLVE", "CDN_CHECK":
		return PhaseResolve
	case "PORT_SCAN", "SERVICE_FINGERPRINT":
		return PhasePort
	case "HTTPX_FINGERPRINT":
		return PhaseWeb
	case "KATANA_CRAWL", "SPOOR_SCAN":
		return PhaseCrawl
	case "FFUF_BRUTE":
		return PhaseBrute
	case "NUCLEI_SCAN":
		return PhaseVuln
	default:
		return PhaseVuln
	}
}

// PhaseConflicts returns true if two phases should not run concurrently on the same host.
// PhaseResolve and PhasePort conflict because both probe the host simultaneously.
// PhaseWeb, PhaseCrawl, PhaseBrute, and PhaseVuln conflict with PhasePort and PhaseResolve
// because port scanning + HTTP probing at the same time can trigger WAF/IPS.
func PhaseConflicts(a, b Phase) bool {
	if a == b {
		return true // same phase always conflicts
	}
	// Group 1: discovery is always safe to parallelize
	if a == PhaseDiscovery || b == PhaseDiscovery {
		return false
	}
	// Group 2: resolve + port conflict
	if (a == PhaseResolve || a == PhasePort) && (b == PhaseResolve || b == PhasePort) {
		return true
	}
	// Group 3: web/crawl/brute/vuln conflict with port/resolve
	if (a >= PhaseWeb) && (b == PhasePort || b == PhaseResolve) {
		return true
	}
	if (b >= PhaseWeb) && (a == PhasePort || a == PhaseResolve) {
		return true
	}
	return false
}

// ipEntry tracks waiting goroutines for cleanup.
type ipEntry struct {
	sem     chan struct{}
	waiting int
}

// IPThrottler enforces per-IP concurrency limits.
// When a host resolves to the same IP, the throttler ensures that IP
// is not hammered by multiple concurrent tool invocations.
type IPThrottler struct {
	mu      sync.Mutex
	entries map[string]*ipEntry
}

// NewIPThrottler creates a new IPThrottler.
func NewIPThrottler() *IPThrottler {
	return &IPThrottler{
		entries: make(map[string]*ipEntry),
	}
}

// Acquire blocks until the IP has a free slot (max 1 concurrent per IP).
// Returns the resolved IP and any error. If resolution fails, the original
// host is used as the key (best-effort).
func (t *IPThrottler) Acquire(ctx context.Context, host string) (string, error) {
	ip := ResolveHostToIP(host)

	t.mu.Lock()
	e, ok := t.entries[ip]
	if !ok {
		e = &ipEntry{sem: make(chan struct{}, 1)}
		t.entries[ip] = e
	}
	e.waiting++
	t.mu.Unlock()

	select {
	case e.sem <- struct{}{}:
		return ip, nil
	case <-ctx.Done():
		t.mu.Lock()
		e.waiting--
		if e.waiting == 0 && len(e.sem) == 0 {
			delete(t.entries, ip)
		}
		t.mu.Unlock()
		return ip, ctx.Err()
	}
}

// Release frees the slot for the given IP.
func (t *IPThrottler) Release(ip string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	e, ok := t.entries[ip]
	if !ok {
		return
	}
	<-e.sem
	e.waiting--
	if e.waiting == 0 && len(e.sem) == 0 {
		delete(t.entries, ip)
	}
}

// ResolveHostToIP resolves a hostname to its first IP address.
// If the input is already an IP, it is returned as-is.
// If resolution fails, the original host is returned (best-effort).
func ResolveHostToIP(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return host
	}
	// Strip port if present
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	if ip := net.ParseIP(host); ip != nil {
		return host
	}
	ips, err := net.LookupIP(host)
	if err != nil || len(ips) == 0 {
		return host
	}
	return ips[0].String()
}

// PhaseGate enforces per-host phase mutual exclusion.
// It ensures that conflicting phases (e.g., port scan + httpx) do not
// run concurrently on the same host.
type PhaseGate struct {
	mu     sync.Mutex
	active map[string]Phase // key = "host:phase"
}

// NewPhaseGate creates a new PhaseGate.
func NewPhaseGate() *PhaseGate {
	return &PhaseGate{
		active: make(map[string]Phase),
	}
}

// TryAcquire attempts to acquire a phase slot for a host.
// Returns true if acquired, false if a conflicting phase is already active.
func (g *PhaseGate) TryAcquire(host string, phase Phase) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Normalize host for comparison
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	hostNorm := strings.ToLower(host)

	// Check for conflicts with active phases on the SAME host only
	for key, activePhase := range g.active {
		if !strings.HasPrefix(key, hostNorm+":") {
			continue
		}
		if PhaseConflicts(activePhase, phase) {
			return false
		}
	}

	g.active[hostPhaseKey(host, phase)] = phase
	return true
}

// Release frees a phase slot for a host.
func (g *PhaseGate) Release(host string, phase Phase) {
	g.mu.Lock()
	defer g.mu.Unlock()
	key := hostPhaseKey(host, phase)
	delete(g.active, key)
}

// hostPhaseKey creates a unique key for a host+phase combination.
func hostPhaseKey(host string, phase Phase) string {
	// Normalize host: strip port, lowercase
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	return strings.ToLower(host) + ":" + string(rune('0'+phase))
}

// JitterDuration returns a random duration between 100ms and 500ms.
// This spreads out task starts to avoid thundering herd effects.
func JitterDuration() time.Duration {
	// Simple deterministic jitter using time.Now().UnixNano()
	// Range: 100ms + (0..400ms)
	n := time.Now().UnixNano()
	jitter := 100 + (n % 400)
	return time.Duration(jitter) * time.Millisecond
}
