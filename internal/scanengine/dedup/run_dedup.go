package dedup

import (
	"sync"
)

// RunDedup provides run-level deduplication for discovered assets.
// It tracks normalized values to prevent the same asset from being
// processed multiple times within a single pipeline run.
type RunDedup struct {
	mu    sync.Mutex
	seen  map[string]bool // normalized_value -> seen
}

// New creates a new RunDedup.
func New() *RunDedup {
	return &RunDedup{
		seen: make(map[string]bool),
	}
}

// IsNew returns true if this normalized value has not been seen before
// in this run. If it is new, it is automatically marked as seen.
func (d *RunDedup) IsNew(normalizedValue string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.seen[normalizedValue] {
		return false
	}
	d.seen[normalizedValue] = true
	return true
}

// MarkSeen explicitly marks a normalized value as seen without checking.
func (d *RunDedup) MarkSeen(normalizedValue string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.seen[normalizedValue] = true
}

// Count returns the number of unique assets seen.
func (d *RunDedup) Count() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.seen)
}
