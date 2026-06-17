// Package domainpool implements a batching pool for subdomain enumeration.
//
// Instead of launching one subfinder process per domain, domains accumulate
// in a pool and are flushed as a batch (subfinder -dL file) when either:
//   - the pool reaches BatchSize, or
//   - the oldest domain has waited longer than FlushTimeout.
//
// Each flush produces a single "generation" work item that covers N domains.
// New domains discovered mid-scan are added to the pool for the next flush.
package domainpool

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// PooledDomain represents a subdomain waiting to be batched.
type PooledDomain struct {
	Value      string // e.g. "sub.example.com"
	AssetID    string // merged asset ID
	ParentID   string
	Depth      int
	DiscoveredAt time.Time
}

// FlushEvent is emitted when the pool flushes a batch.
type FlushEvent struct {
	Domains  []PooledDomain
	FilePath string // temp file with one domain per line
	Generation int  // monotonic flush counter
}

// Config controls pool behavior.
type Config struct {
	BatchSize    int           // flush when pool reaches this size (default 50)
	FlushTimeout time.Duration // flush if oldest domain waited this long (default 10s)
	DataDir      string        // directory for temp domain files
}

// DefaultConfig returns sensible defaults.
func DefaultConfig(dataDir string) Config {
	return Config{
		BatchSize:    50,
		FlushTimeout: 10 * time.Second,
		DataDir:      dataDir,
	}
}

// Pool accumulates subdomains and flushes them in batches.
type Pool struct {
	mu          sync.Mutex
	config      Config
	domains     []PooledDomain
	seen        map[string]bool // dedup within pool
	oldestAt    time.Time       // when the first domain in current batch was added
	generation  int
	onFlush     func(FlushEvent)
	stopCh      chan struct{}
	stopped     chan struct{}
}

// New creates a Pool. Call Start() to begin the timeout ticker.
func New(config Config, onFlush func(FlushEvent)) *Pool {
	return &Pool{
		config:  config,
		seen:    make(map[string]bool),
		onFlush: onFlush,
		stopCh:  make(chan struct{}),
		stopped: make(chan struct{}),
	}
}

// Start begins the background timeout checker. Call Stop() to shut down.
func (p *Pool) Start() {
	go p.loop()
}

// Stop halts the background loop and waits for it to exit.
func (p *Pool) Stop() {
	close(p.stopCh)
	<-p.stopped
}

// Len returns the current pool size.
func (p *Pool) Len() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.domains)
}

// Add enqueues a domain. Returns true if accepted (not a duplicate).
// If the pool reaches BatchSize, a flush is triggered synchronously.
func (p *Pool) Add(d PooledDomain) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.seen[d.Value] {
		return false
	}
	p.seen[d.Value] = true

	if len(p.domains) == 0 {
		p.oldestAt = time.Now()
	}
	p.domains = append(p.domains, d)

	if len(p.domains) >= p.config.BatchSize {
		p.flushLocked()
	}
	return true
}

// loop runs the timeout checker.
func (p *Pool) loop() {
	defer close(p.stopped)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-p.stopCh:
			// Final flush on shutdown
			p.mu.Lock()
			if len(p.domains) > 0 {
				p.flushLocked()
			}
			p.mu.Unlock()
			return
		case <-ticker.C:
			p.mu.Lock()
			if len(p.domains) > 0 && time.Since(p.oldestAt) >= p.config.FlushTimeout {
				p.flushLocked()
			}
			p.mu.Unlock()
		}
	}
}

// flushLocked flushes the current batch. Caller must hold p.mu.
func (p *Pool) flushLocked() {
	p.generation++
	gen := p.generation
	domains := p.domains
	p.domains = nil
	// Reset seen for next generation (new domains may reappear from different parents)
	p.seen = make(map[string]bool)
	p.oldestAt = time.Time{}

	// Write domain file
	path := filepath.Join(p.config.DataDir, fmt.Sprintf("domain_batch_%d.txt", gen))
	if err := writeDomainFile(path, domains); err != nil {
		log.Printf("[domainpool] write domain file gen %d: %v", gen, err)
		return
	}

	log.Printf("[domainpool] flush gen %d: %d domains → %s", gen, len(domains), path)

	if p.onFlush != nil {
		p.onFlush(FlushEvent{
			Domains:    domains,
			FilePath:   path,
			Generation: gen,
		})
	}
}

// FlushNow forces a synchronous flush when the pool has pending domains.
func (p *Pool) FlushNow() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.domains) > 0 {
		p.flushLocked()
	}
}

func writeDomainFile(path string, domains []PooledDomain) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, d := range domains {
		fmt.Fprintln(f, d.Value)
	}
	return nil
}
