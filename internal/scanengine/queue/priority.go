package queue

import (
	"sync"
)

// Priority represents the urgency of a work item.
type Priority int

const (
	PriorityHigh   Priority = 0
	PriorityMedium Priority = 1
	PriorityLow    Priority = 2
)

// Item is a work item in the priority queue.
type Item struct {
	WorkID   string
	Action   string
	AssetID  string
	Priority Priority
}

// Depth returns the queue depth per priority tier.
type Depth struct {
	High   int `json:"high"`
	Medium int `json:"medium"`
	Low    int `json:"low"`
}

// PriorityQueue is a three-tier FIFO queue for work items.
type PriorityQueue struct {
	mu   sync.Mutex
	high []Item
	med  []Item
	low  []Item
}

// New creates a new empty PriorityQueue.
func New() *PriorityQueue {
	return &PriorityQueue{}
}

// Push adds an item to the appropriate tier.
func (q *PriorityQueue) Push(item Item) {
	q.mu.Lock()
	defer q.mu.Unlock()
	switch item.Priority {
	case PriorityHigh:
		q.high = append(q.high, item)
	case PriorityMedium:
		q.med = append(q.med, item)
	case PriorityLow:
		q.low = append(q.low, item)
	}
}

// Pop removes and returns the highest-priority item.
// Returns (_, false) if the queue is empty.
func (q *PriorityQueue) Pop() (Item, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.high) > 0 {
		item := q.high[0]
		q.high = q.high[1:]
		return item, true
	}
	if len(q.med) > 0 {
		item := q.med[0]
		q.med = q.med[1:]
		return item, true
	}
	if len(q.low) > 0 {
		item := q.low[0]
		q.low = q.low[1:]
		return item, true
	}
	return Item{}, false
}

// Len returns the total number of items in the queue.
func (q *PriorityQueue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.high) + len(q.med) + len(q.low)
}

// QueueDepth returns the count per tier.
func (q *PriorityQueue) QueueDepth() Depth {
	q.mu.Lock()
	defer q.mu.Unlock()
	return Depth{
		High:   len(q.high),
		Medium: len(q.med),
		Low:    len(q.low),
	}
}

// IsEmpty returns true if the queue has no items.
func (q *PriorityQueue) IsEmpty() bool {
	return q.Len() == 0
}

// ClassifyAction returns the default priority for a given action.
func ClassifyAction(action string) Priority {
	switch action {
	case "HTTPX_FINGERPRINT", "NUCLEI_SCAN":
		return PriorityHigh
	case "PORT_SCAN", "SERVICE_FINGERPRINT":
		return PriorityMedium
	default:
		return PriorityLow
	}
}
