// Package pool implements generic batching pools for Tier-1 scan actions.
package pool

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Member is one row in a batch input file with scheduling metadata.
type Member struct {
	Value     string // host / IP / domain written to the input file
	AssetID   string
	BucketKey string
}

// FlushEvent is emitted when a pool flushes a batch.
type FlushEvent struct {
	Members    []Member
	FilePath   string
	Generation int
}

// Config controls pool behavior.
type Config struct {
	BatchSize    int
	FlushTimeout time.Duration
	DataDir      string
	FilePrefix   string
	Label        string // log label
}

// DefaultHostPoolConfig returns defaults for DNS host pooling.
func DefaultHostPoolConfig(dataDir string) Config {
	return Config{
		BatchSize:    100,
		FlushTimeout: 10 * time.Second,
		DataDir:      dataDir,
		FilePrefix:   "host_batch",
		Label:        "hostpool",
	}
}

// DefaultIPPoolConfig returns defaults for CDN / port IP pooling.
func DefaultIPPoolConfig(dataDir, prefix, label string, batchSize int) Config {
	if batchSize < 1 {
		batchSize = 50
	}
	return Config{
		BatchSize:    batchSize,
		FlushTimeout: 10 * time.Second,
		DataDir:      dataDir,
		FilePrefix:   prefix,
		Label:        label,
	}
}

// Pool accumulates members and flushes them in batches.
type Pool struct {
	mu         sync.Mutex
	config     Config
	members    []Member
	seen       map[string]bool
	oldestAt   time.Time
	generation int
	onFlush    func(FlushEvent)
	stopCh     chan struct{}
	stopped    chan struct{}
}

// New creates a Pool. Call Start() to begin timeout flushing.
func New(config Config, onFlush func(FlushEvent)) *Pool {
	return &Pool{
		config:  config,
		seen:    make(map[string]bool),
		onFlush: onFlush,
		stopCh:  make(chan struct{}),
		stopped: make(chan struct{}),
	}
}

// Start begins the background timeout checker.
func (p *Pool) Start() {
	go p.loop()
}

// Stop halts the loop, performs a final flush, and waits for exit.
func (p *Pool) Stop() {
	close(p.stopCh)
	<-p.stopped
}

// Len returns the current pool size.
func (p *Pool) Len() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.members)
}

// Add enqueues a member. Returns false if the value is a duplicate in the current batch window.
// Triggers a synchronous flush when BatchSize is reached.
func (p *Pool) Add(m Member) bool {
	key := normalizeKey(m.Value)
	if key == "" {
		return false
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.seen[key] {
		return false
	}
	p.seen[key] = true

	if len(p.members) == 0 {
		p.oldestAt = time.Now()
	}
	p.members = append(p.members, m)

	if len(p.members) >= p.config.BatchSize {
		p.flushLocked()
	}
	return true
}

func (p *Pool) loop() {
	defer close(p.stopped)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-p.stopCh:
			p.mu.Lock()
			if len(p.members) > 0 {
				p.flushLocked()
			}
			p.mu.Unlock()
			return
		case <-ticker.C:
			p.mu.Lock()
			if len(p.members) > 0 && time.Since(p.oldestAt) >= p.config.FlushTimeout {
				p.flushLocked()
			}
			p.mu.Unlock()
		}
	}
}

func (p *Pool) flushLocked() {
	p.generation++
	gen := p.generation
	members := p.members
	p.members = nil
	p.seen = make(map[string]bool)
	p.oldestAt = time.Time{}

	path := filepath.Join(p.config.DataDir, fmt.Sprintf("%s_%d.txt", p.config.FilePrefix, gen))
	if err := writeLinesFile(path, members); err != nil {
		log.Printf("[%s] write batch file gen %d: %v", p.config.Label, gen, err)
		return
	}

	log.Printf("[%s] flush gen %d: %d members → %s", p.config.Label, gen, len(members), path)
	if p.onFlush != nil {
		p.onFlush(FlushEvent{
			Members:    members,
			FilePath:   path,
			Generation: gen,
		})
	}
}

func writeLinesFile(path string, members []Member) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, m := range members {
		if _, err := fmt.Fprintln(f, m.Value); err != nil {
			return err
		}
	}
	return nil
}

func normalizeKey(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}

func (p *Pool) FlushNow() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.members) > 0 {
		p.flushLocked()
	}
}

// ReadLines reads non-empty lines from a batch input file.
func ReadLines(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var lines []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines, nil
}
