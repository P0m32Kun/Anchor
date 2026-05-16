package worker

import (
	"context"
	"errors"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// fakeSampler is a goroutine-safe ResourceSampler for tests. mem/cpu values
// can be mutated mid-test to simulate load draining.
type fakeSampler struct {
	mu      sync.Mutex
	mem     float64
	cpu     float64
	memErr  error
	cpuErr  error
	memHits atomic.Int64
	cpuHits atomic.Int64
}

func (f *fakeSampler) MemoryUsedPercent(ctx context.Context) (float64, error) {
	f.memHits.Add(1)
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.mem, f.memErr
}

func (f *fakeSampler) CPUUsedPercent(ctx context.Context) (float64, error) {
	f.cpuHits.Add(1)
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.cpu, f.cpuErr
}

func (f *fakeSampler) setMem(v float64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.mem = v
}

func (f *fakeSampler) setCPU(v float64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.cpu = v
}

func newTestGovernor(sampler *fakeSampler) *ResourceGovernor {
	cfg := GovernorConfig{
		Enabled:            true,
		MemoryThresholdPct: 85,
		CPUThresholdPct:    80,
		MemoryPollInterval: 10 * time.Millisecond,
		CPUDelay:           20 * time.Millisecond,
	}
	return NewResourceGovernor(cfg, sampler)
}

func TestResourceGovernor_DisabledIsNoop(t *testing.T) {
	sampler := &fakeSampler{mem: 99, cpu: 99}
	cfg := DefaultGovernorConfig()
	cfg.Enabled = false
	g := NewResourceGovernor(cfg, sampler)

	start := time.Now()
	if err := g.Acquire(context.Background()); err != nil {
		t.Fatalf("Acquire returned err with governor disabled: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Millisecond {
		t.Fatalf("Acquire was not instant when disabled: took %v", elapsed)
	}
	if sampler.memHits.Load() != 0 || sampler.cpuHits.Load() != 0 {
		t.Fatalf("sampler was hit while governor disabled: mem=%d cpu=%d", sampler.memHits.Load(), sampler.cpuHits.Load())
	}
}

func TestResourceGovernor_NilReceiverIsNoop(t *testing.T) {
	var g *ResourceGovernor
	if err := g.Acquire(context.Background()); err != nil {
		t.Fatalf("nil governor should be a no-op, got err: %v", err)
	}
}

func TestResourceGovernor_HappyPath(t *testing.T) {
	sampler := &fakeSampler{mem: 50, cpu: 50}
	g := newTestGovernor(sampler)

	start := time.Now()
	if err := g.Acquire(context.Background()); err != nil {
		t.Fatalf("Acquire under threshold returned err: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Millisecond {
		t.Fatalf("Acquire was slow under normal load: %v", elapsed)
	}
}

func TestResourceGovernor_MemoryBlocksUntilDrains(t *testing.T) {
	sampler := &fakeSampler{mem: 95, cpu: 50}
	g := newTestGovernor(sampler)

	// Drain memory after 40ms (4 poll cycles).
	go func() {
		time.Sleep(40 * time.Millisecond)
		sampler.setMem(60)
	}()

	start := time.Now()
	if err := g.Acquire(context.Background()); err != nil {
		t.Fatalf("Acquire returned err after memory drained: %v", err)
	}
	elapsed := time.Since(start)
	if elapsed < 30*time.Millisecond {
		t.Fatalf("Acquire returned too early — expected to block ~40ms, got %v", elapsed)
	}
	if elapsed > 200*time.Millisecond {
		t.Fatalf("Acquire blocked too long: %v", elapsed)
	}
	if sampler.memHits.Load() < 3 {
		t.Fatalf("expected multiple memory samples, got %d", sampler.memHits.Load())
	}
}

func TestResourceGovernor_MemoryCtxCancel(t *testing.T) {
	sampler := &fakeSampler{mem: 95, cpu: 50}
	g := newTestGovernor(sampler)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := g.Acquire(ctx)
	if err == nil {
		t.Fatal("expected ctx cancel error from blocked Acquire")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected DeadlineExceeded, got: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 200*time.Millisecond {
		t.Fatalf("Acquire did not exit promptly on ctx cancel: %v", elapsed)
	}
}

func TestResourceGovernor_CPUDelaysOnce(t *testing.T) {
	sampler := &fakeSampler{mem: 50, cpu: 95}
	g := newTestGovernor(sampler)

	start := time.Now()
	if err := g.Acquire(context.Background()); err != nil {
		t.Fatalf("Acquire returned err with high CPU: %v", err)
	}
	elapsed := time.Since(start)
	if elapsed < 15*time.Millisecond {
		t.Fatalf("Acquire did not apply CPU delay: %v < 20ms", elapsed)
	}
	if elapsed > 100*time.Millisecond {
		t.Fatalf("Acquire delayed longer than CPUDelay: %v", elapsed)
	}
	// One sample is enough; we don't poll for CPU like memory.
	if sampler.cpuHits.Load() != 1 {
		t.Fatalf("expected exactly 1 CPU sample, got %d", sampler.cpuHits.Load())
	}
}

func TestResourceGovernor_CPUUnderThresholdNoDelay(t *testing.T) {
	sampler := &fakeSampler{mem: 50, cpu: 50}
	g := newTestGovernor(sampler)

	start := time.Now()
	if err := g.Acquire(context.Background()); err != nil {
		t.Fatalf("Acquire returned err: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Millisecond {
		t.Fatalf("Acquire incurred delay below CPU threshold: %v", elapsed)
	}
}

func TestResourceGovernor_CPUCtxCancel(t *testing.T) {
	sampler := &fakeSampler{mem: 50, cpu: 95}
	cfg := GovernorConfig{
		Enabled:            true,
		MemoryThresholdPct: 85,
		CPUThresholdPct:    80,
		MemoryPollInterval: 10 * time.Millisecond,
		CPUDelay:           500 * time.Millisecond, // long enough to outlast ctx
	}
	g := NewResourceGovernor(cfg, sampler)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	err := g.Acquire(ctx)
	if err == nil {
		t.Fatal("expected ctx cancel during CPU delay")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected DeadlineExceeded, got: %v", err)
	}
}

func TestResourceGovernor_SamplerFailOpen(t *testing.T) {
	// Memory error → fail open
	sampler := &fakeSampler{mem: 95, cpu: 95, memErr: errors.New("boom")}
	g := newTestGovernor(sampler)
	if err := g.Acquire(context.Background()); err != nil {
		t.Fatalf("expected fail-open when memory sampler errors, got: %v", err)
	}

	// CPU error → fail open (memory passes through)
	sampler2 := &fakeSampler{mem: 50, cpu: 95, cpuErr: errors.New("boom")}
	g2 := newTestGovernor(sampler2)
	if err := g2.Acquire(context.Background()); err != nil {
		t.Fatalf("expected fail-open when cpu sampler errors, got: %v", err)
	}
}

func TestLoadGovernorConfigFromEnv_Defaults(t *testing.T) {
	clearGovernorEnv(t)
	cfg := LoadGovernorConfigFromEnv()
	def := DefaultGovernorConfig()
	if cfg != def {
		t.Fatalf("expected defaults, got %+v vs %+v", cfg, def)
	}
}

func TestLoadGovernorConfigFromEnv_ValidOverrides(t *testing.T) {
	clearGovernorEnv(t)
	t.Setenv("ANCHOR_GOVERNOR_ENABLED", "false")
	t.Setenv("ANCHOR_GOVERNOR_MEM_PCT", "70")
	t.Setenv("ANCHOR_GOVERNOR_CPU_PCT", "60")
	t.Setenv("ANCHOR_GOVERNOR_POLL_MS", "250")
	t.Setenv("ANCHOR_GOVERNOR_CPU_DELAY_MS", "1500")

	cfg := LoadGovernorConfigFromEnv()
	if cfg.Enabled {
		t.Error("Enabled should be false")
	}
	if cfg.MemoryThresholdPct != 70 {
		t.Errorf("MemoryThresholdPct = %v, want 70", cfg.MemoryThresholdPct)
	}
	if cfg.CPUThresholdPct != 60 {
		t.Errorf("CPUThresholdPct = %v, want 60", cfg.CPUThresholdPct)
	}
	if cfg.MemoryPollInterval != 250*time.Millisecond {
		t.Errorf("MemoryPollInterval = %v, want 250ms", cfg.MemoryPollInterval)
	}
	if cfg.CPUDelay != 1500*time.Millisecond {
		t.Errorf("CPUDelay = %v, want 1500ms", cfg.CPUDelay)
	}
}

func TestLoadGovernorConfigFromEnv_InvalidFallsBackToDefault(t *testing.T) {
	clearGovernorEnv(t)
	t.Setenv("ANCHOR_GOVERNOR_MEM_PCT", "not-a-number")
	t.Setenv("ANCHOR_GOVERNOR_CPU_PCT", "150")    // out of range
	t.Setenv("ANCHOR_GOVERNOR_POLL_MS", "-1")     // out of range
	t.Setenv("ANCHOR_GOVERNOR_CPU_DELAY_MS", "x") // not an int

	cfg := LoadGovernorConfigFromEnv()
	def := DefaultGovernorConfig()
	if cfg.MemoryThresholdPct != def.MemoryThresholdPct {
		t.Errorf("MemoryThresholdPct should fall back, got %v", cfg.MemoryThresholdPct)
	}
	if cfg.CPUThresholdPct != def.CPUThresholdPct {
		t.Errorf("CPUThresholdPct should fall back, got %v", cfg.CPUThresholdPct)
	}
	if cfg.MemoryPollInterval != def.MemoryPollInterval {
		t.Errorf("MemoryPollInterval should fall back, got %v", cfg.MemoryPollInterval)
	}
	if cfg.CPUDelay != def.CPUDelay {
		t.Errorf("CPUDelay should fall back, got %v", cfg.CPUDelay)
	}
}

// TestGopsutilSamplerSmoke exercises the real gopsutil-backed sampler against
// the host: memory and CPU readings must come back without error and within
// 0..100. This catches platform breakage in the library/driver layer (e.g.,
// missing /proc on Linux, sysctl access on darwin) that fakes cannot.
func TestGopsutilSamplerSmoke(t *testing.T) {
	s := newGopsutilSampler()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	mem, err := s.MemoryUsedPercent(ctx)
	if err != nil {
		t.Fatalf("real memory sample failed: %v", err)
	}
	if mem < 0 || mem > 100 {
		t.Fatalf("memory percent out of range: %v", mem)
	}

	cpu, err := s.CPUUsedPercent(ctx)
	if err != nil {
		t.Fatalf("real cpu sample failed: %v", err)
	}
	if cpu < 0 || cpu > 100 {
		t.Fatalf("cpu percent out of range: %v", cpu)
	}
	t.Logf("real-system snapshot: mem=%.1f%% cpu=%.1f%%", mem, cpu)
}

// TestRealGovernorHappyPath uses the real sampler and the default thresholds.
// Assumes the test host is under 85% mem / 80% cpu — overwhelmingly true on
// dev machines and CI. If this flakes, raise thresholds via env vars.
func TestRealGovernorHappyPath(t *testing.T) {
	clearGovernorEnv(t)
	cfg := LoadGovernorConfigFromEnv()
	g := NewResourceGovernor(cfg, nil) // nil sampler → real gopsutil

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	start := time.Now()
	if err := g.Acquire(ctx); err != nil {
		t.Fatalf("Acquire under normal load returned err: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 600*time.Millisecond {
		t.Fatalf("Acquire was slow on real sampler under normal load: %v (CPUDelay caps at 500ms; sampling shouldn't add much)", elapsed)
	}
}

// TestRealGovernorMemoryAtImpossibleThresholdBlocks lowers the memory threshold
// to 0.01% so any real reading exceeds it, then verifies Acquire blocks and
// ctx cancellation releases it.
func TestRealGovernorMemoryAtImpossibleThresholdBlocks(t *testing.T) {
	cfg := GovernorConfig{
		Enabled:            true,
		MemoryThresholdPct: 0.01, // unreachable: host memory is always > 0.01%
		CPUThresholdPct:    100,  // never trigger
		MemoryPollInterval: 50 * time.Millisecond,
		CPUDelay:           50 * time.Millisecond,
	}
	g := NewResourceGovernor(cfg, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	err := g.Acquire(ctx)
	if err == nil {
		t.Fatal("expected Acquire to block until ctx cancel with unreachable memory threshold")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected DeadlineExceeded, got: %v", err)
	}
}

// clearGovernorEnv unsets every ANCHOR_GOVERNOR_* variable for the duration of
// the test, restoring originals via t.Cleanup. Using os.Unsetenv directly (not
// t.Setenv with "") because LookupEnv returns true for set-but-empty values.
func clearGovernorEnv(t *testing.T) {
	t.Helper()
	keys := []string{
		"ANCHOR_GOVERNOR_ENABLED",
		"ANCHOR_GOVERNOR_MEM_PCT",
		"ANCHOR_GOVERNOR_CPU_PCT",
		"ANCHOR_GOVERNOR_POLL_MS",
		"ANCHOR_GOVERNOR_CPU_DELAY_MS",
	}
	for _, k := range keys {
		orig, had := os.LookupEnv(k)
		_ = os.Unsetenv(k)
		t.Cleanup(func() {
			if had {
				_ = os.Setenv(k, orig)
			} else {
				_ = os.Unsetenv(k)
			}
		})
	}
}
