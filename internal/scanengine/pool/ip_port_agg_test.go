package pool

import (
	"testing"
)

func TestIPPortAggregator_MergesPortsPerIP(t *testing.T) {
	dir := t.TempDir()
	var events []FlushEvent
	agg := NewIPPortAggregator(dir, func(_ string, ev FlushEvent) {
		events = append(events, ev)
	})

	agg.Add("1.2.3.4", 80, "a1", "b1")
	agg.Add("1.2.3.4", 443, "a2", "b1")
	agg.Add("5.6.7.8", 22, "a3", "b2")

	if agg.Len() != 2 {
		t.Fatalf("Len = %d, want 2 IPs", agg.Len())
	}

	agg.FlushAll()
	if len(events) != 2 {
		t.Fatalf("flushes = %d, want 2", len(events))
	}

	// FlushAll iterates a map so order is non-deterministic.
	// Find the event for 5.6.7.8 and the event for 1.2.3.4 by members count.
	var singleIP, multiIP *FlushEvent
	for _, ev := range events {
		if len(ev.Members) == 1 {
			singleIP = &ev
		} else if len(ev.Members) == 2 {
			multiIP = &ev
		}
	}
	if singleIP == nil {
		t.Fatal("expected an event with 1 member for 5.6.7.8")
	}
	if multiIP == nil {
		t.Fatal("expected an event with 2 members for 1.2.3.4")
	}

	ports := SortedPortsFromMembers(singleIP.Members)
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
