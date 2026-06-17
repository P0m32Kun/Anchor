package worker

import (
	"os"
	"strconv"
)

const defaultMaxConcurrency = 10

// LoadMaxConcurrencyFromEnv reads ANCHOR_WORKER_MAX_CONCURRENCY (default 10).
// Values <= 0 fall back to the default.
func LoadMaxConcurrencyFromEnv() int {
	raw := os.Getenv("ANCHOR_WORKER_MAX_CONCURRENCY")
	if raw == "" {
		return defaultMaxConcurrency
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return defaultMaxConcurrency
	}
	return n
}
