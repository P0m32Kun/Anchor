package worker

import (
	"os"
	"testing"
)

func TestLoadMaxConcurrencyFromEnv_Default(t *testing.T) {
	os.Unsetenv("ANCHOR_WORKER_MAX_CONCURRENCY")
	got := LoadMaxConcurrencyFromEnv()
	if got != defaultMaxConcurrency {
		t.Errorf("expected default %d, got %d", defaultMaxConcurrency, got)
	}
}

func TestLoadMaxConcurrencyFromEnv_Valid(t *testing.T) {
	t.Setenv("ANCHOR_WORKER_MAX_CONCURRENCY", "5")
	got := LoadMaxConcurrencyFromEnv()
	if got != 5 {
		t.Errorf("expected 5, got %d", got)
	}
}

func TestLoadMaxConcurrencyFromEnv_Invalid(t *testing.T) {
	t.Setenv("ANCHOR_WORKER_MAX_CONCURRENCY", "abc")
	got := LoadMaxConcurrencyFromEnv()
	if got != defaultMaxConcurrency {
		t.Errorf("expected default %d for invalid input, got %d", defaultMaxConcurrency, got)
	}
}

func TestLoadMaxConcurrencyFromEnv_Zero(t *testing.T) {
	t.Setenv("ANCHOR_WORKER_MAX_CONCURRENCY", "0")
	got := LoadMaxConcurrencyFromEnv()
	if got != defaultMaxConcurrency {
		t.Errorf("expected default %d for zero, got %d", defaultMaxConcurrency, got)
	}
}

func TestLoadMaxConcurrencyFromEnv_Negative(t *testing.T) {
	t.Setenv("ANCHOR_WORKER_MAX_CONCURRENCY", "-1")
	got := LoadMaxConcurrencyFromEnv()
	if got != defaultMaxConcurrency {
		t.Errorf("expected default %d for negative, got %d", defaultMaxConcurrency, got)
	}
}
