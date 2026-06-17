package scheduler

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestIPThrottler_SingleConcurrent(t *testing.T) {
	th := NewIPThrottler()
	ctx := context.Background()

	ip, err := th.Acquire(ctx, "127.0.0.1")
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	if ip != "127.0.0.1" {
		t.Fatalf("expected IP 127.0.0.1, got %s", ip)
	}

	// Second acquire on same IP should block
	ctx2, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err = th.Acquire(ctx2, "127.0.0.1")
	if err != context.DeadlineExceeded {
		t.Fatalf("expected DeadlineExceeded, got %v", err)
	}

	// Release and re-acquire should work
	th.Release("127.0.0.1")
	_, err = th.Acquire(ctx, "127.0.0.1")
	if err != nil {
		t.Fatalf("Acquire after Release: %v", err)
	}
	th.Release("127.0.0.1")
}

func TestIPThrottler_DifferentIPs(t *testing.T) {
	th := NewIPThrottler()
	ctx := context.Background()

	_, err := th.Acquire(ctx, "127.0.0.1")
	if err != nil {
		t.Fatalf("Acquire 127.0.0.1: %v", err)
	}
	defer th.Release("127.0.0.1")

	_, err = th.Acquire(ctx, "127.0.0.2")
	if err != nil {
		t.Fatalf("Acquire 127.0.0.2: %v", err)
	}
	th.Release("127.0.0.2")
}

func TestIPThrottler_ConcurrentSafety(t *testing.T) {
	th := NewIPThrottler()
	var maxConcurrent int64
	var current int64

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx := context.Background()
			_, err := th.Acquire(ctx, "10.0.0.1")
			if err != nil {
				return
			}
			cur := atomic.AddInt64(&current, 1)
			for {
				old := atomic.LoadInt64(&maxConcurrent)
				if cur <= old || atomic.CompareAndSwapInt64(&maxConcurrent, old, cur) {
					break
				}
			}
			time.Sleep(10 * time.Millisecond)
			atomic.AddInt64(&current, -1)
			th.Release("10.0.0.1")
		}()
	}
	wg.Wait()

	if got := atomic.LoadInt64(&maxConcurrent); got > 1 {
		t.Errorf("max concurrent per IP = %d, want ≤ 1", got)
	}
}

func TestIPThrottler_CancelReleasesSlot(t *testing.T) {
	th := NewIPThrottler()
	ctx := context.Background()

	_, err := th.Acquire(ctx, "10.0.0.1")
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}

	// Cancel should unblock
	ctx2, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := th.Acquire(ctx2, "10.0.0.1")
		done <- err
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != context.Canceled {
			t.Fatalf("expected Canceled, got %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Acquire did not return after cancel")
	}

	th.Release("10.0.0.1")
}

func TestPhaseGate_BasicConflict(t *testing.T) {
	g := NewPhaseGate()

	if !g.TryAcquire("example.com", PhasePort) {
		t.Fatal("expected first acquire to succeed")
	}

	// Same phase should conflict
	if g.TryAcquire("example.com", PhasePort) {
		t.Fatal("expected conflict for same phase")
	}

	// Different phase that conflicts
	if g.TryAcquire("example.com", PhaseResolve) {
		t.Fatal("expected conflict for resolve vs port")
	}

	// Discovery should not conflict
	if !g.TryAcquire("example.com", PhaseDiscovery) {
		t.Fatal("expected discovery to not conflict with port")
	}

	g.Release("example.com", PhasePort)
	g.Release("example.com", PhaseDiscovery)
}

func TestPhaseGate_WebPortConflict(t *testing.T) {
	g := NewPhaseGate()

	if !g.TryAcquire("example.com", PhasePort) {
		t.Fatal("expected first acquire to succeed")
	}

	// Web phase should conflict with port
	if g.TryAcquire("example.com", PhaseWeb) {
		t.Fatal("expected conflict for web vs port")
	}

	// Vuln phase should conflict with port
	if g.TryAcquire("example.com", PhaseVuln) {
		t.Fatal("expected conflict for vuln vs port")
	}

	g.Release("example.com", PhasePort)
}

func TestPhaseGate_DifferentHosts(t *testing.T) {
	g := NewPhaseGate()

	if !g.TryAcquire("host1.com", PhasePort) {
		t.Fatal("expected first acquire to succeed")
	}

	// Different host should not conflict
	if !g.TryAcquire("host2.com", PhasePort) {
		t.Fatal("expected different host to not conflict")
	}

	g.Release("host1.com", PhasePort)
	g.Release("host2.com", PhasePort)
}

func TestPhaseConflicts(t *testing.T) {
	tests := []struct {
		a, b   Phase
		expect bool
	}{
		{PhaseDiscovery, PhaseDiscovery, true},
		{PhaseDiscovery, PhasePort, false},
		{PhasePort, PhasePort, true},
		{PhasePort, PhaseResolve, true},
		{PhaseResolve, PhaseResolve, true},
		{PhaseWeb, PhasePort, true},
		{PhaseWeb, PhaseVuln, false},
		{PhaseWeb, PhaseCrawl, false},
		{PhaseBrute, PhaseResolve, true},
		{PhaseVuln, PhaseDiscovery, false},
	}
	for _, tt := range tests {
		got := PhaseConflicts(tt.a, tt.b)
		if got != tt.expect {
			t.Errorf("PhaseConflicts(%d, %d) = %v, want %v", tt.a, tt.b, got, tt.expect)
		}
	}
}

func TestResolveHostToIP_IP(t *testing.T) {
	ip := ResolveHostToIP("192.168.1.1")
	if ip != "192.168.1.1" {
		t.Fatalf("expected 192.168.1.1, got %s", ip)
	}
}

func TestResolveHostToIP_WithPort(t *testing.T) {
	ip := ResolveHostToIP("192.168.1.1:8080")
	if ip != "192.168.1.1" {
		t.Fatalf("expected 192.168.1.1, got %s", ip)
	}
}

func TestJitterDuration_Range(t *testing.T) {
	for i := 0; i < 100; i++ {
		d := JitterDuration()
		if d < 100*time.Millisecond || d > 500*time.Millisecond {
			t.Fatalf("jitter %v out of range [100ms, 500ms]", d)
		}
	}
}
