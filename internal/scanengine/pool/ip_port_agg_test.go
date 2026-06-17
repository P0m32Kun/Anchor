package pool

import (
	"testing"
)

func TestIPPortAggregator_MergesPortsPerIP(t *testing.T) {
	dir := t.TempDir()
	var flushes int
	var last FlushEvent
	agg := NewIPPortAggregator(dir, func(_ string, ev FlushEvent) {
		flushes++
		last = ev
	})

	agg.Add("1.2.3.4", 80, "a1", "b1")
	agg.Add("1.2.3.4", 443, "a2", "b1")
	agg.Add("5.6.7.8", 22, "a3", "b2")

	if agg.Len() != 2 {
		t.Fatalf("Len = %d, want 2 IPs", agg.Len())
	}

	agg.FlushAll()
	if flushes != 2 {
		t.Fatalf("flushes = %d, want 2", flushes)
	}
	if len(last.Members) != 1 {
		t.Fatalf("last batch members = %d", len(last.Members))
	}
	ports := SortedPortsFromMembers(last.Members)
	if len(ports) != 1 || ports[0] != 22 {
		t.Fatalf("ports = %v", ports)
	}
}

func TestSortedPortsFromMembers(t *testing.T) {
	ports := SortedPortsFromMembers([]Member{
		{Value: "1.2.3.4:443"},
		{Value: "1.2.3.4:80"},
		{Value: "1.2.3.4:443"},
	})
	if len(ports) != 2 || ports[0] != 80 || ports[1] != 443 {
		t.Fatalf("ports = %v", ports)
	}
}
