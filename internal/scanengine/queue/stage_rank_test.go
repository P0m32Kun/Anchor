package queue_test

import (
	"testing"

	"github.com/P0m32Kun/Anchor/internal/scanengine/queue"
)

func TestActionToStageRank(t *testing.T) {
	tests := []struct {
		action string
		want   queue.StageRank
	}{
		{"PASSIVE_SEARCH", queue.StageDiscovery},
		{"PASSIVE_CERT", queue.StageDiscovery},
		{"PASSIVE_URL", queue.StageDiscovery},
		{"SUBDOMAIN_ENUM", queue.StageSubdomain},
		{"DNS_RESOLVE", queue.StageResolve},
		{"CDN_CHECK", queue.StageCDN},
		{"PORT_SCAN", queue.StagePort},
		{"SERVICE_FINGERPRINT", queue.StageService},
		{"HTTPX_FINGERPRINT", queue.StageWeb},
		{"KATANA_CRAWL", queue.StageCrawl},
		{"SPOOR_SCAN", queue.StageCrawl},
		{"FFUF_BRUTE", queue.StageBrute},
		{"NUCLEI_SCAN", queue.StageVuln},
		{"UNKNOWN_ACTION", queue.StageVuln}, // default
	}
	for _, tt := range tests {
		if got := queue.ActionToStageRank(tt.action); got != tt.want {
			t.Errorf("ActionToStageRank(%s) = %d, want %d", tt.action, got, tt.want)
		}
	}
}

func TestPopFairStaged_HigherStageBlocksLower(t *testing.T) {
	q := queue.New()

	// Push items from different stages
	q.Push(queue.Item{WorkID: "dns1", Action: "DNS_RESOLVE", BucketKey: "target:1", Priority: queue.PriorityLow})
	q.Push(queue.Item{WorkID: "httpx1", Action: "HTTPX_FINGERPRINT", BucketKey: "target:1", Priority: queue.PriorityHigh})
	q.Push(queue.Item{WorkID: "nuclei1", Action: "NUCLEI_SCAN", BucketKey: "target:1", Priority: queue.PriorityHigh})

	// Should pop DNS_RESOLVE first (stage 30) because it has higher priority than HTTPX (70) and NUCLEI (100)
	item, ok := q.PopFairStaged(10, 10, map[string]int{})
	if !ok {
		t.Fatal("expected item")
	}
	if item.WorkID != "dns1" {
		t.Fatalf("expected dns1 (DNS_RESOLVE stage 30), got %s (action=%s)", item.WorkID, item.Action)
	}

	// Now only HTTPX and NUCLEI remain. HTTPX (stage 70) should come before NUCLEI (stage 100)
	item, ok = q.PopFairStaged(10, 10, map[string]int{})
	if !ok {
		t.Fatal("expected item")
	}
	if item.WorkID != "httpx1" {
		t.Fatalf("expected httpx1 (HTTPX_FINGERPRINT stage 70), got %s (action=%s)", item.WorkID, item.Action)
	}

	// Finally NUCLEI
	item, ok = q.PopFairStaged(10, 10, map[string]int{})
	if !ok {
		t.Fatal("expected item")
	}
	if item.WorkID != "nuclei1" {
		t.Fatalf("expected nuclei1 (NUCLEI_SCAN stage 100), got %s (action=%s)", item.WorkID, item.Action)
	}

	// Queue should be empty
	_, ok = q.PopFairStaged(10, 10, map[string]int{})
	if ok {
		t.Fatal("expected empty queue")
	}
}

func TestPopFairStaged_SameStageFairScheduling(t *testing.T) {
	q := queue.New()

	// Push multiple items in the same stage (DNS_RESOLVE)
	q.Push(queue.Item{WorkID: "dns-a1", Action: "DNS_RESOLVE", BucketKey: "target:A", Priority: queue.PriorityLow})
	q.Push(queue.Item{WorkID: "dns-a2", Action: "DNS_RESOLVE", BucketKey: "target:A", Priority: queue.PriorityLow})
	q.Push(queue.Item{WorkID: "dns-b1", Action: "DNS_RESOLVE", BucketKey: "target:B", Priority: queue.PriorityLow})

	// With target:A having 1 in-flight, should prefer target:B
	inflight := map[string]int{"target:A": 1}
	item, ok := q.PopFairStaged(2, 10, inflight)
	if !ok {
		t.Fatal("expected item")
	}
	if item.BucketKey != "target:B" {
		t.Fatalf("expected target:B (lower in-flight), got %s", item.BucketKey)
	}
}

func TestPopFairStaged_BucketCapRespected(t *testing.T) {
	q := queue.New()

	q.Push(queue.Item{WorkID: "dns1", Action: "DNS_RESOLVE", BucketKey: "target:1", Priority: queue.PriorityLow})
	q.Push(queue.Item{WorkID: "dns2", Action: "DNS_RESOLVE", BucketKey: "target:2", Priority: queue.PriorityLow})

	// target:1 is at max capacity
	inflight := map[string]int{"target:1": 2}
	item, ok := q.PopFairStaged(2, 10, inflight)
	if !ok {
		t.Fatal("expected item")
	}
	if item.BucketKey != "target:2" {
		t.Fatalf("expected target:2 (target:1 at cap), got %s", item.BucketKey)
	}
}

func TestPopFairStaged_MixedStagesAndBuckets(t *testing.T) {
	q := queue.New()

	// Mix of stages and buckets
	q.Push(queue.Item{WorkID: "httpx-a1", Action: "HTTPX_FINGERPRINT", BucketKey: "target:A", Priority: queue.PriorityHigh})
	q.Push(queue.Item{WorkID: "httpx-b1", Action: "HTTPX_FINGERPRINT", BucketKey: "target:B", Priority: queue.PriorityHigh})
	q.Push(queue.Item{WorkID: "dns-a1", Action: "DNS_RESOLVE", BucketKey: "target:A", Priority: queue.PriorityLow})
	q.Push(queue.Item{WorkID: "dns-b1", Action: "DNS_RESOLVE", BucketKey: "target:B", Priority: queue.PriorityLow})

	// Should pop DNS items first (stage 30), with fair scheduling between A and B
	inflight := map[string]int{}
	item1, ok := q.PopFairStaged(1, 10, inflight)
	if !ok {
		t.Fatal("expected item")
	}
	if item1.Action != "DNS_RESOLVE" {
		t.Fatalf("expected DNS_RESOLVE (stage 30), got %s", item1.Action)
	}

	// Mark the popped bucket as in-flight
	inflight[item1.BucketKey] = 1

	// Should pop the other DNS item
	item2, ok := q.PopFairStaged(1, 10, inflight)
	if !ok {
		t.Fatal("expected item")
	}
	if item2.Action != "DNS_RESOLVE" {
		t.Fatalf("expected DNS_RESOLVE (stage 30), got %s", item2.Action)
	}
	if item2.BucketKey == item1.BucketKey {
		t.Fatalf("expected different bucket, got same: %s", item2.BucketKey)
	}

	// Now DNS is done, should pop HTTPX items
	inflight = map[string]int{}
	item3, ok := q.PopFairStaged(1, 10, inflight)
	if !ok {
		t.Fatal("expected item")
	}
	if item3.Action != "HTTPX_FINGERPRINT" {
		t.Fatalf("expected HTTPX_FINGERPRINT (stage 70), got %s", item3.Action)
	}
}

func TestPopFairStaged_EmptyQueue(t *testing.T) {
	q := queue.New()

	_, ok := q.PopFairStaged(10, 10, map[string]int{})
	if ok {
		t.Fatal("expected false for empty queue")
	}
}

func TestPopFairStaged_AllStagesPopulated(t *testing.T) {
	q := queue.New()

	// Push items from all stages
	actions := []string{
		"PASSIVE_SEARCH", "SUBDOMAIN_ENUM", "DNS_RESOLVE", "CDN_CHECK",
		"PORT_SCAN", "SERVICE_FINGERPRINT", "HTTPX_FINGERPRINT",
		"KATANA_CRAWL", "FFUF_BRUTE", "NUCLEI_SCAN",
	}
	for i, action := range actions {
		q.Push(queue.Item{
			WorkID:    action,
			Action:    action,
			BucketKey: "target:1",
			Priority:  queue.PriorityLow,
		})
		_ = i
	}

	// Should pop in stage order
	expectedOrder := []string{
		"PASSIVE_SEARCH",  // stage 10
		"SUBDOMAIN_ENUM",  // stage 20
		"DNS_RESOLVE",     // stage 30
		"CDN_CHECK",       // stage 40
		"PORT_SCAN",       // stage 50
		"SERVICE_FINGERPRINT", // stage 60
		"HTTPX_FINGERPRINT",   // stage 70
		"KATANA_CRAWL",        // stage 80
		"FFUF_BRUTE",          // stage 90
		"NUCLEI_SCAN",         // stage 100
	}

	for _, expected := range expectedOrder {
		item, ok := q.PopFairStaged(10, 10, map[string]int{})
		if !ok {
			t.Fatalf("expected item for %s", expected)
		}
		if item.Action != expected {
			t.Fatalf("expected %s, got %s", expected, item.Action)
		}
	}

	// Queue should be empty
	_, ok := q.PopFairStaged(10, 10, map[string]int{})
	if ok {
		t.Fatal("expected empty queue")
	}
}

func TestStageDepth(t *testing.T) {
	q := queue.New()

	q.Push(queue.Item{WorkID: "dns1", Action: "DNS_RESOLVE", Priority: queue.PriorityLow})
	q.Push(queue.Item{WorkID: "dns2", Action: "DNS_RESOLVE", Priority: queue.PriorityLow})
	q.Push(queue.Item{WorkID: "httpx1", Action: "HTTPX_FINGERPRINT", Priority: queue.PriorityHigh})
	q.Push(queue.Item{WorkID: "nuclei1", Action: "NUCLEI_SCAN", Priority: queue.PriorityHigh})

	depth := q.StageDepth()
	if depth[queue.StageResolve] != 2 {
		t.Fatalf("expected 2 DNS_RESOLVE items, got %d", depth[queue.StageResolve])
	}
	if depth[queue.StageWeb] != 1 {
		t.Fatalf("expected 1 HTTPX_FINGERPRINT item, got %d", depth[queue.StageWeb])
	}
	if depth[queue.StageVuln] != 1 {
		t.Fatalf("expected 1 NUCLEI_SCAN item, got %d", depth[queue.StageVuln])
	}
}
