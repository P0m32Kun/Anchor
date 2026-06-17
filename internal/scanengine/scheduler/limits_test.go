package scheduler

import (
	"testing"
	"time"
)

func TestComputeLimits_ScalesWithTargets(t *testing.T) {
	lim := ComputeLimits(1, 0)
	if lim.GlobalMax != 10 {
		t.Fatalf("GlobalMax = %d, want 10", lim.GlobalMax)
	}
	if lim.ActiveBuckets != 1 {
		t.Fatalf("ActiveBuckets = %d, want 1", lim.ActiveBuckets)
	}

	lim = ComputeLimits(20, 0)
	if lim.GlobalMax != 48 {
		t.Fatalf("GlobalMax = %d, want 48", lim.GlobalMax)
	}
	if lim.ActiveBuckets != InitialActiveBuckets {
		t.Fatalf("ActiveBuckets = %d, want %d", lim.ActiveBuckets, InitialActiveBuckets)
	}
}

func TestComputeLimits_CapsGlobalMax(t *testing.T) {
	lim := ComputeLimits(100, 0)
	if lim.GlobalMax != MaxConcurrency {
		t.Fatalf("GlobalMax = %d, want %d", lim.GlobalMax, MaxConcurrency)
	}
}

func TestComputeLimits_RampsActiveBuckets(t *testing.T) {
	lim := ComputeLimits(20, ActiveBucketRampInterval)
	if lim.ActiveBuckets != InitialActiveBuckets+ActiveBucketRampStep {
		t.Fatalf("ActiveBuckets = %d, want %d", lim.ActiveBuckets, InitialActiveBuckets+ActiveBucketRampStep)
	}

	lim = ComputeLimits(20, 10*ActiveBucketRampInterval)
	if lim.ActiveBuckets != 20 {
		t.Fatalf("ActiveBuckets = %d, want 20 (target count cap)", lim.ActiveBuckets)
	}
}

func TestComputeLimits_PerBucketStaysLow(t *testing.T) {
	lim := ComputeLimits(50, time.Hour)
	if lim.PerBucketMax != PerBucketConcurrency {
		t.Fatalf("PerBucketMax = %d, want %d", lim.PerBucketMax, PerBucketConcurrency)
	}
}
