package queue

import (
	"sync"
	"testing"
)

func TestPushPop_BasicFIFO(t *testing.T) {
	q := New()
	q.Push(Item{WorkID: "a", Priority: PriorityLow})
	q.Push(Item{WorkID: "b", Priority: PriorityLow})
	item, ok := q.Pop()
	if !ok || item.WorkID != "a" {
		t.Fatalf("expected a, got %v", item)
	}
	item, ok = q.Pop()
	if !ok || item.WorkID != "b" {
		t.Fatalf("expected b, got %v", item)
	}
	if _, ok := q.Pop(); ok {
		t.Fatal("expected empty queue")
	}
}

func TestPushPop_PriorityOrder(t *testing.T) {
	q := New()
	q.Push(Item{WorkID: "low", Priority: PriorityLow})
	q.Push(Item{WorkID: "high", Priority: PriorityHigh})
	q.Push(Item{WorkID: "med", Priority: PriorityMedium})

	item, _ := q.Pop()
	if item.WorkID != "high" {
		t.Fatalf("expected high, got %s", item.WorkID)
	}
	item, _ = q.Pop()
	if item.WorkID != "med" {
		t.Fatalf("expected med, got %s", item.WorkID)
	}
	item, _ = q.Pop()
	if item.WorkID != "low" {
		t.Fatalf("expected low, got %s", item.WorkID)
	}
}

func TestPushPop_SamePriorityFIFO(t *testing.T) {
	q := New()
	q.Push(Item{WorkID: "h1", Priority: PriorityHigh})
	q.Push(Item{WorkID: "h2", Priority: PriorityHigh})
	q.Push(Item{WorkID: "h3", Priority: PriorityHigh})

	for _, want := range []string{"h1", "h2", "h3"} {
		item, ok := q.Pop()
		if !ok || item.WorkID != want {
			t.Fatalf("expected %s, got %v (ok=%v)", want, item, ok)
		}
	}
}

func TestDedup(t *testing.T) {
	q := New()
	q.Push(Item{WorkID: "dup", Priority: PriorityHigh})
	q.Push(Item{WorkID: "dup", Priority: PriorityHigh}) // duplicate
	q.Push(Item{WorkID: "other", Priority: PriorityHigh})

	if q.Len() != 2 {
		t.Fatalf("expected len 2, got %d", q.Len())
	}
	item, _ := q.Pop()
	if item.WorkID != "dup" {
		t.Fatalf("expected dup, got %s", item.WorkID)
	}
	item, _ = q.Pop()
	if item.WorkID != "other" {
		t.Fatalf("expected other, got %s", item.WorkID)
	}
	if _, ok := q.Pop(); ok {
		t.Fatal("expected empty")
	}
}

func TestDedup_AfterPopAllowsRepush(t *testing.T) {
	q := New()
	q.Push(Item{WorkID: "x", Priority: PriorityLow})
	q.Pop() // removes from seen
	q.Push(Item{WorkID: "x", Priority: PriorityLow}) // should succeed
	if q.Len() != 1 {
		t.Fatalf("expected len 1 after repush, got %d", q.Len())
	}
}

func TestIsEmpty(t *testing.T) {
	q := New()
	if !q.IsEmpty() {
		t.Fatal("new queue should be empty")
	}
	q.Push(Item{WorkID: "a", Priority: PriorityLow})
	if q.IsEmpty() {
		t.Fatal("queue should not be empty after push")
	}
	q.Pop()
	if !q.IsEmpty() {
		t.Fatal("queue should be empty after pop")
	}
}

func TestQueueDepth(t *testing.T) {
	q := New()
	q.Push(Item{WorkID: "h1", Priority: PriorityHigh})
	q.Push(Item{WorkID: "h2", Priority: PriorityHigh})
	q.Push(Item{WorkID: "m1", Priority: PriorityMedium})
	q.Push(Item{WorkID: "l1", Priority: PriorityLow})

	d := q.QueueDepth()
	if d.High != 2 || d.Medium != 1 || d.Low != 1 {
		t.Fatalf("unexpected depth: %+v", d)
	}
}

func TestConcurrentPushPop(t *testing.T) {
	q := New()
	const n = 100
	var wg sync.WaitGroup

	// Push n items from n goroutines
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(id int) {
			defer wg.Done()
			q.Push(Item{WorkID: string(rune('a' + id%26)) + string(rune('0'+id/26)), Priority: Priority(id % 3)})
		}(i)
	}
	wg.Wait()

	// Pop all items, count them
	popped := 0
	for {
		_, ok := q.Pop()
		if !ok {
			break
		}
		popped++
	}
	if popped == 0 {
		t.Fatal("expected some items to be popped")
	}
	t.Logf("popped %d unique items", popped)
}

func BenchmarkPushPop(b *testing.B) {
	for i := 0; i < b.N; i++ {
		q := New()
		for j := 0; j < 1000; j++ {
			q.Push(Item{WorkID: string(rune(j)), Priority: Priority(j % 3)})
		}
		for j := 0; j < 1000; j++ {
			q.Pop()
		}
	}
}

func TestClassifyAction(t *testing.T) {
	tests := []struct {
		action string
		want   Priority
	}{
		{"HTTPX_FINGERPRINT", PriorityHigh},
		{"NUCLEI_SCAN", PriorityHigh},
		{"PORT_SCAN", PriorityMedium},
		{"SERVICE_FINGERPRINT", PriorityMedium},
		{"SUBDOMAIN_ENUM", PriorityLow},
		{"KATANA_CRAWL", PriorityLow},
		{"FFUF_BRUTE", PriorityLow},
	}
	for _, tt := range tests {
		if got := ClassifyAction(tt.action); got != tt.want {
			t.Errorf("ClassifyAction(%s) = %d, want %d", tt.action, got, tt.want)
		}
	}
}
