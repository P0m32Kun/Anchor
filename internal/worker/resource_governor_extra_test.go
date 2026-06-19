package worker

import (
	"context"
	"testing"
	"time"
)

// --- gopsutilSampler edge cases ---

func TestGopsutilSampler_contextCancelled_Memory(t *testing.T) {
	s := newGopsutilSampler()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := s.MemoryUsedPercent(ctx)
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}

func TestGopsutilSampler_contextCancelled_CPU(t *testing.T) {
	s := newGopsutilSampler()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := s.CPUUsedPercent(ctx)
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}

func TestGopsutilSampler_MemoryUsedPercent(t *testing.T) {
	s := newGopsutilSampler()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	v, err := s.MemoryUsedPercent(ctx)
	if err != nil {
		t.Fatalf("MemoryUsedPercent: %v", err)
	}
	if v < 0 || v > 100 {
		t.Errorf("memory = %f, want 0..100", v)
	}
}

func TestGopsutilSampler_CPUUsedPercent(t *testing.T) {
	s := newGopsutilSampler()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// First call may return 0% due to baseline seeding.
	v, err := s.CPUUsedPercent(ctx)
	if err != nil {
		t.Fatalf("CPUUsedPercent: %v", err)
	}
	if v < 0 || v > 100 {
		t.Errorf("cpu = %f, want 0..100", v)
	}
}

// --- LoadGovernorConfigFromEnv edge cases ---

func TestLoadGovernorConfigFromEnv_enabledEmpty(t *testing.T) {
	clearGovernorEnv(t)
	t.Setenv("ANCHOR_GOVERNOR_ENABLED", "")

	cfg := LoadGovernorConfigFromEnv()
	// Empty string is not a valid bool, so it falls back to default.
	def := DefaultGovernorConfig()
	if cfg.Enabled != def.Enabled {
		t.Errorf("Enabled = %v, want %v (default for empty string)", cfg.Enabled, def.Enabled)
	}
}

func TestLoadGovernorConfigFromEnv_cpuDelayZero(t *testing.T) {
	clearGovernorEnv(t)
	t.Setenv("ANCHOR_GOVERNOR_CPU_DELAY_MS", "0")

	cfg := LoadGovernorConfigFromEnv()
	if cfg.CPUDelay != 0 {
		t.Errorf("CPUDelay = %v, want 0", cfg.CPUDelay)
	}
}

func TestDefaultGovernorConfig(t *testing.T) {
	cfg := DefaultGovernorConfig()
	if !cfg.Enabled {
		t.Error("Enabled should be true by default")
	}
	if cfg.MemoryThresholdPct != 85 {
		t.Errorf("MemoryThresholdPct = %v, want 85", cfg.MemoryThresholdPct)
	}
	if cfg.CPUThresholdPct != 80 {
		t.Errorf("CPUThresholdPct = %v, want 80", cfg.CPUThresholdPct)
	}
	if cfg.MemoryPollInterval != 1*time.Second {
		t.Errorf("MemoryPollInterval = %v, want 1s", cfg.MemoryPollInterval)
	}
	if cfg.CPUDelay != 500*time.Millisecond {
		t.Errorf("CPUDelay = %v, want 500ms", cfg.CPUDelay)
	}
}

// --- Acquire with CPUDelay = 0 ---

func TestResourceGovernor_CPUDelayZero(t *testing.T) {
	sampler := &fakeSampler{mem: 50, cpu: 95}
	cfg := GovernorConfig{
		Enabled:            true,
		MemoryThresholdPct: 85,
		CPUThresholdPct:    80,
		MemoryPollInterval: 10 * time.Millisecond,
		CPUDelay:           0,
	}
	g := NewResourceGovernor(cfg, sampler)

	start := time.Now()
	if err := g.Acquire(context.Background()); err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Millisecond {
		t.Errorf("Acquire should be instant with CPUDelay=0, took %v", elapsed)
	}
}

// --- Acquire with memory error fail-open ---

func TestResourceGovernor_MemoryErrorFailOpen(t *testing.T) {
	sampler := &fakeSampler{mem: 95, cpu: 50, memErr: context.DeadlineExceeded}
	cfg := GovernorConfig{
		Enabled:            true,
		MemoryThresholdPct: 85,
		CPUThresholdPct:    80,
		MemoryPollInterval: 10 * time.Millisecond,
		CPUDelay:           20 * time.Millisecond,
	}
	g := NewResourceGovernor(cfg, sampler)

	if err := g.Acquire(context.Background()); err != nil {
		t.Fatalf("expected fail-open, got: %v", err)
	}
}

// --- NewResourceGovernor with nil sampler ---

func TestNewResourceGovernor_nilSampler(t *testing.T) {
	cfg := GovernorConfig{Enabled: false}
	g := NewResourceGovernor(cfg, nil)
	if g == nil {
		t.Fatal("expected non-nil governor")
	}
	if g.sampler == nil {
		t.Error("expected non-nil sampler (gopsutil fallback)")
	}
}
