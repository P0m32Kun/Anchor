package scheduler

import "time"

const (
	BaseConcurrency          = 8
	ConcurrencyPerTarget     = 2
	MaxConcurrency           = 50
	PerBucketConcurrency     = 1
	InitialActiveBuckets     = 3
	ActiveBucketRampInterval = 30 * time.Second
	ActiveBucketRampStep     = 3
)

// Limits describes concurrency caps for a scan run at a point in time.
type Limits struct {
	GlobalMax     int
	PerBucketMax  int
	ActiveBuckets int
}

// ComputeLimits derives dynamic concurrency from seed target count and elapsed run time.
// Global concurrency grows with targets; per-bucket stays low to avoid hammering one company.
// ActiveBuckets ramps over fixed intervals so distinct targets are spread rather than burst on one.
func ComputeLimits(targetCount int, elapsed time.Duration) Limits {
	if targetCount < 1 {
		targetCount = 1
	}

	globalMax := BaseConcurrency + targetCount*ConcurrencyPerTarget
	if globalMax > MaxConcurrency {
		globalMax = MaxConcurrency
	}

	activeBuckets := InitialActiveBuckets + int(elapsed/ActiveBucketRampInterval)*ActiveBucketRampStep
	if activeBuckets > targetCount {
		activeBuckets = targetCount
	}
	if activeBuckets > globalMax {
		activeBuckets = globalMax
	}
	if activeBuckets < 1 {
		activeBuckets = 1
	}

	return Limits{
		GlobalMax:     globalMax,
		PerBucketMax:  PerBucketConcurrency,
		ActiveBuckets: activeBuckets,
	}
}
