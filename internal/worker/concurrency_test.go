package worker

import "testing"

func TestLoadMaxConcurrencyFromEnv(t *testing.T) {
	t.Setenv("ANCHOR_WORKER_MAX_CONCURRENCY", "")
	if got := LoadMaxConcurrencyFromEnv(); got != 10 {
		t.Fatalf("default = %d, want 10", got)
	}
	t.Setenv("ANCHOR_WORKER_MAX_CONCURRENCY", "5")
	if got := LoadMaxConcurrencyFromEnv(); got != 5 {
		t.Fatalf("explicit = %d, want 5", got)
	}
	t.Setenv("ANCHOR_WORKER_MAX_CONCURRENCY", "0")
	if got := LoadMaxConcurrencyFromEnv(); got != 10 {
		t.Fatalf("invalid = %d, want fallback 10", got)
	}
}
