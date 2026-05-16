package worker

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

// GovernorConfig configures static-threshold resource governance.
// Units match upstream tooling (percent stays percent, ms stays ms) per
// project convention: no unit conversion inside the code.
type GovernorConfig struct {
	// Enabled toggles the entire governor; when false Acquire is a no-op.
	Enabled bool
	// MemoryThresholdPct is the system memory usage percent (0-100) above
	// which Acquire blocks until usage drops below the threshold.
	MemoryThresholdPct float64
	// CPUThresholdPct is the system CPU usage percent (0-100) above which
	// Acquire sleeps CPUDelay before returning (rate-halving semantics).
	CPUThresholdPct float64
	// MemoryPollInterval is how often Acquire re-samples memory while waiting.
	MemoryPollInterval time.Duration
	// CPUDelay is the fixed sleep applied per Acquire call when CPU is over
	// threshold. New tasks are pushed back by this duration each.
	CPUDelay time.Duration
}

// ResourceSampler reads system-level memory and CPU utilisation. The
// interface lets tests inject deterministic values without spinning up real
// gopsutil sampling.
type ResourceSampler interface {
	MemoryUsedPercent(ctx context.Context) (float64, error)
	CPUUsedPercent(ctx context.Context) (float64, error)
}

// ResourceGovernor gates new task execution based on system memory and CPU
// thresholds. Memory over threshold blocks; CPU over threshold delays.
type ResourceGovernor struct {
	cfg     GovernorConfig
	sampler ResourceSampler
	logger  *log.Logger
}

// DefaultGovernorConfig returns the baseline thresholds (mem 85%, cpu 80%,
// poll 1s, cpu-delay 500ms). Enabled defaults to true.
func DefaultGovernorConfig() GovernorConfig {
	return GovernorConfig{
		Enabled:            true,
		MemoryThresholdPct: 85,
		CPUThresholdPct:    80,
		MemoryPollInterval: 1 * time.Second,
		CPUDelay:           500 * time.Millisecond,
	}
}

// LoadGovernorConfigFromEnv reads thresholds from ANCHOR_GOVERNOR_* environment
// variables, falling back to DefaultGovernorConfig. Invalid values are logged
// and the default is used in their place.
func LoadGovernorConfigFromEnv() GovernorConfig {
	cfg := DefaultGovernorConfig()
	if v, ok := os.LookupEnv("ANCHOR_GOVERNOR_ENABLED"); ok {
		b, err := strconv.ParseBool(v)
		if err == nil {
			cfg.Enabled = b
		} else {
			log.Printf("[governor] invalid ANCHOR_GOVERNOR_ENABLED=%q, using default %v", v, cfg.Enabled)
		}
	}
	if v, ok := os.LookupEnv("ANCHOR_GOVERNOR_MEM_PCT"); ok {
		f, err := strconv.ParseFloat(v, 64)
		if err == nil && f > 0 && f <= 100 {
			cfg.MemoryThresholdPct = f
		} else {
			log.Printf("[governor] invalid ANCHOR_GOVERNOR_MEM_PCT=%q, using default %v", v, cfg.MemoryThresholdPct)
		}
	}
	if v, ok := os.LookupEnv("ANCHOR_GOVERNOR_CPU_PCT"); ok {
		f, err := strconv.ParseFloat(v, 64)
		if err == nil && f > 0 && f <= 100 {
			cfg.CPUThresholdPct = f
		} else {
			log.Printf("[governor] invalid ANCHOR_GOVERNOR_CPU_PCT=%q, using default %v", v, cfg.CPUThresholdPct)
		}
	}
	if v, ok := os.LookupEnv("ANCHOR_GOVERNOR_POLL_MS"); ok {
		n, err := strconv.Atoi(v)
		if err == nil && n > 0 {
			cfg.MemoryPollInterval = time.Duration(n) * time.Millisecond
		} else {
			log.Printf("[governor] invalid ANCHOR_GOVERNOR_POLL_MS=%q, using default %v", v, cfg.MemoryPollInterval)
		}
	}
	if v, ok := os.LookupEnv("ANCHOR_GOVERNOR_CPU_DELAY_MS"); ok {
		n, err := strconv.Atoi(v)
		if err == nil && n >= 0 {
			cfg.CPUDelay = time.Duration(n) * time.Millisecond
		} else {
			log.Printf("[governor] invalid ANCHOR_GOVERNOR_CPU_DELAY_MS=%q, using default %v", v, cfg.CPUDelay)
		}
	}
	return cfg
}

// NewResourceGovernor wires the config with a sampler. Pass nil sampler to use
// the default gopsutil-backed implementation.
func NewResourceGovernor(cfg GovernorConfig, sampler ResourceSampler) *ResourceGovernor {
	if sampler == nil {
		sampler = newGopsutilSampler()
	}
	return &ResourceGovernor{
		cfg:     cfg,
		sampler: sampler,
		logger:  log.Default(),
	}
}

// Acquire is called before starting a new task. Memory above threshold blocks
// via polling until usage drops; CPU above threshold sleeps CPUDelay once.
// Returns ctx.Err() if the context is cancelled while waiting.
func (g *ResourceGovernor) Acquire(ctx context.Context) error {
	if g == nil || !g.cfg.Enabled {
		return nil
	}
	if err := g.waitForMemory(ctx); err != nil {
		return err
	}
	return g.applyCPUDelay(ctx)
}

// waitForMemory polls memory usage and blocks while over threshold.
func (g *ResourceGovernor) waitForMemory(ctx context.Context) error {
	loggedOnce := false
	for {
		used, err := g.sampler.MemoryUsedPercent(ctx)
		if err != nil {
			// Fail open: if sampling fails, do not block tasks indefinitely.
			g.logger.Printf("[governor] memory sample failed (failing open): %v", err)
			return nil
		}
		if used < g.cfg.MemoryThresholdPct {
			if loggedOnce {
				g.logger.Printf("[governor] memory back under threshold (%.1f%% < %.1f%%), releasing task", used, g.cfg.MemoryThresholdPct)
			}
			return nil
		}
		if !loggedOnce {
			g.logger.Printf("[governor] memory over threshold (%.1f%% >= %.1f%%), queueing task", used, g.cfg.MemoryThresholdPct)
			loggedOnce = true
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("governor: %w", ctx.Err())
		case <-time.After(g.cfg.MemoryPollInterval):
		}
	}
}

// applyCPUDelay sleeps CPUDelay once when CPU is over threshold.
func (g *ResourceGovernor) applyCPUDelay(ctx context.Context) error {
	if g.cfg.CPUDelay <= 0 {
		return nil
	}
	used, err := g.sampler.CPUUsedPercent(ctx)
	if err != nil {
		g.logger.Printf("[governor] cpu sample failed (failing open): %v", err)
		return nil
	}
	if used < g.cfg.CPUThresholdPct {
		return nil
	}
	g.logger.Printf("[governor] cpu over threshold (%.1f%% >= %.1f%%), delaying task by %v", used, g.cfg.CPUThresholdPct, g.cfg.CPUDelay)
	select {
	case <-ctx.Done():
		return fmt.Errorf("governor: %w", ctx.Err())
	case <-time.After(g.cfg.CPUDelay):
		return nil
	}
}

// ErrSamplerUnavailable indicates the underlying gopsutil call failed and the
// caller should fall back to default behaviour.
var ErrSamplerUnavailable = errors.New("sampler unavailable")

// gopsutilSampler is the default ResourceSampler backed by shirou/gopsutil.
// CPU sampling uses interval=0 (delta since last call). A background prime
// call seeds the baseline so the first Acquire returns meaningful data.
type gopsutilSampler struct {
	primeOnce sync.Once
}

func newGopsutilSampler() *gopsutilSampler {
	s := &gopsutilSampler{}
	// Fire-and-forget prime so the first non-zero CPU reading is ready
	// shortly after process start. Acquire still works before prime
	// completes — it just reads 0% CPU during that brief window.
	go func() {
		_, _ = cpu.Percent(500*time.Millisecond, false)
	}()
	return s
}

func (s *gopsutilSampler) MemoryUsedPercent(ctx context.Context) (float64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	v, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrSamplerUnavailable, err)
	}
	return v.UsedPercent, nil
}

func (s *gopsutilSampler) CPUUsedPercent(ctx context.Context) (float64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	// Interval 0 = delta since last call. The prime goroutine in
	// newGopsutilSampler seeds the baseline.
	pct, err := cpu.PercentWithContext(ctx, 0, false)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrSamplerUnavailable, err)
	}
	if len(pct) == 0 {
		return 0, nil
	}
	return pct[0], nil
}
