package queue_test

import (
	"testing"

	"github.com/P0m32Kun/Anchor/internal/scanengine/queue"
)

func TestPriorityQueue_PopFair_SpreadsBuckets(t *testing.T) {
	q := queue.New()
	q.Push(queue.Item{WorkID: "a1", BucketKey: "target:1", Priority: queue.PriorityLow})
	q.Push(queue.Item{WorkID: "a2", BucketKey: "target:1", Priority: queue.PriorityLow})
	q.Push(queue.Item{WorkID: "b1", BucketKey: "target:2", Priority: queue.PriorityLow})

	inflight := map[string]int{"target:1": 1}
	item, ok := q.PopFair(1, 3, inflight)
	if !ok {
		t.Fatal("expected item")
	}
	if item.BucketKey != "target:2" {
		t.Fatalf("BucketKey = %q, want target:2", item.BucketKey)
	}
}

func TestPriorityQueue_PopFair_RespectsActiveBucketCap(t *testing.T) {
	q := queue.New()
	q.Push(queue.Item{WorkID: "c1", BucketKey: "target:3", Priority: queue.PriorityLow})

	inflight := map[string]int{"target:1": 1, "target:2": 1}
	_, ok := q.PopFair(1, 2, inflight)
	if ok {
		t.Fatal("expected no item when active bucket cap reached")
	}
}

func TestPriorityQueue_PopFair_PrefersHigherPriority(t *testing.T) {
	q := queue.New()
	q.Push(queue.Item{WorkID: "low", BucketKey: "target:1", Priority: queue.PriorityLow})
	q.Push(queue.Item{WorkID: "high", BucketKey: "target:2", Priority: queue.PriorityHigh})

	item, ok := q.PopFair(1, 3, map[string]int{})
	if !ok {
		t.Fatal("expected item")
	}
	if item.WorkID != "high" {
		t.Fatalf("WorkID = %q, want high", item.WorkID)
	}
}
