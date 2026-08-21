package scanconfig

import (
	"strings"

	"github.com/P0m32Kun/Anchor/internal/errors"
	"github.com/P0m32Kun/Anchor/internal/models"
)

// ErrInternalModeRemoved is the error code returned to clients that still attempt
// to start a scan in the retired "internal" network mode. It signals a permanent
// compatibility exit, not a transient failure.
const ErrInternalModeRemoved errors.ErrorCode = "INTERNAL_MODE_REMOVED"

// InternalModeError is the documented compatibility error for internal-mode scans.
func InternalModeError() *errors.AppError {
	return errors.New(ErrInternalModeRemoved,
		"the dedicated internal scan mode has been retired. Use the internet scan mode; see docs/current/product.md#product-boundaries. To migrate a saved internal configuration, call with ?migrate=external.")
}

// IsInternalMode reports whether a submitted scan mode still targets the retired
// internal network profile.
func IsInternalMode(mode string) bool {
	return strings.EqualFold(strings.TrimSpace(mode), "internal")
}

// IsLegacyInternalConfig reports whether a persisted PipelineConfig still carries
// the retired internal-mode shape: subfinder and CDN filtering disabled together
// with an aggressive port sweep (high naabu rate or the "high-risk"/full port
// range). An ordinary internet config that happens to toggle a single tool off is
// not misclassified.
func IsLegacyInternalConfig(cfg models.PipelineConfig) bool {
	if cfg.EnableSubfinder || cfg.EnableCDNFilter {
		return false
	}
	highRate := cfg.NaabuRate >= 500
	highRange := cfg.PortRange == "high-risk" || cfg.PortRange == "full"
	return highRate || highRange
}

// MigrateInternalConfigToExternal explicitly widens a saved internal-shaped config
// to the internet baseline so a scan can run without silently expanding scope.
// It is only invoked when the caller passed ?migrate=external.
func MigrateInternalConfigToExternal(cfg models.PipelineConfig) models.PipelineConfig {
	base := models.DefaultExternalPipelineConfig()
	if cfg.PortRange == "high-risk" || cfg.PortRange == "" {
		cfg.PortRange = base.PortRange
	}
	cfg.EnableSubfinder = true
	cfg.EnableDNSx = true
	cfg.EnableCDNFilter = true
	cfg.EnableNmapService = true
	cfg.EnableHttpx = true
	cfg.EnableNuclei = true
	cfg.EnablePassiveSearch = true
	cfg.SkipPortscanOnCDNHost = true
	cfg.NucleiRequireFingerprint = true
	if cfg.NaabuRate > base.NaabuRate {
		cfg.NaabuRate = base.NaabuRate
	}
	return cfg
}
