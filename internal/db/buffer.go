package db

import (
	"log"
	"sync"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
)

// FindingBuffer batches finding inserts to reduce database round-trips.
// Flush triggers on capacity or timeout.
// Deduplication happens in-memory by dedup_key and INSERT OR IGNORE handles
// cross-boundary duplicates at the DB level.
type FindingBuffer struct {
	queries       *Queries
	capacity      int
	flushInterval time.Duration

	mu     sync.Mutex
	buf    []*models.Finding
	timer  *time.Timer
	closed bool
}

// NewFindingBuffer creates a buffer. capacity is max items before auto-flush.
// flushInterval is max time before auto-flush.
func NewFindingBuffer(queries *Queries, capacity int, flushInterval time.Duration) *FindingBuffer {
	return &FindingBuffer{
		queries:       queries,
		capacity:      capacity,
		flushInterval: flushInterval,
	}
}

// Add inserts a finding into the buffer. If the buffer is full, it flushes
// immediately. If this is the first item, a timer is started.
func (b *FindingBuffer) Add(f *models.Finding) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		// Buffer closed — fall back to direct insert.
		if err := b.queries.CreateFinding(f); err != nil {
			log.Printf("[finding_buffer] direct insert after close: %v", err)
		}
		return
	}

	b.buf = append(b.buf, f)

	// Start timer on first item.
	if len(b.buf) == 1 {
		b.timer = time.AfterFunc(b.flushInterval, b.flushTimerCallback)
	}

	// Auto-flush on capacity.
	if len(b.buf) >= b.capacity {
		b.flushLocked()
	}
}

// Flush forces an immediate flush. Safe to call concurrently.
func (b *FindingBuffer) Flush() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.flushLocked()
}

// Close stops the timer and flushes remaining items.
// Safe to call multiple times; subsequent calls are no-ops.
func (b *FindingBuffer) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil
	}
	b.closed = true

	if b.timer != nil {
		b.timer.Stop()
		b.timer = nil
	}
	return b.flushLocked()
}

// flushTimerCallback is invoked by time.AfterFunc.
// If the flush fails, findings remain in the buffer and the timer is
// restarted so a retry will occur on the next interval.
func (b *FindingBuffer) flushTimerCallback() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	if err := b.flushLocked(); err != nil {
		// Restart timer so we retry on the next interval.
		if !b.closed && len(b.buf) > 0 {
			b.timer = time.AfterFunc(b.flushInterval, b.flushTimerCallback)
		}
	}
}

// flushLocked performs the actual DB write. Caller must hold b.mu.
// On failure, findings are retained in the buffer so they can be retried.
func (b *FindingBuffer) flushLocked() error {
	if len(b.buf) == 0 {
		return nil
	}

	if b.timer != nil {
		b.timer.Stop()
		b.timer = nil
	}

	findings := b.buf

	// In-memory dedup by dedup_key (keep first occurrence).
	seen := make(map[string]bool, len(findings))
	deduped := make([]*models.Finding, 0, len(findings))
	for _, f := range findings {
		if seen[f.DedupKey] {
			continue
		}
		seen[f.DedupKey] = true
		deduped = append(deduped, f)
	}

	if err := b.queries.BatchInsertFindings(deduped); err != nil {
		log.Printf("[finding_buffer] batch insert %d findings: %v", len(deduped), err)
		return err
	}

	b.buf = nil
	return nil
}
