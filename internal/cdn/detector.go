package cdn

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/toolguard"
)

// Detector detects CDN usage for IPs using cdncheck CLI.
type Detector struct {
	allowlist *toolguard.Allowlist
}

func NewDetector() *Detector {
	return &Detector{allowlist: toolguard.NewAllowlist()}
}

// CheckIP checks if an IP is behind a CDN.
func (d *Detector) CheckIP(ctx context.Context, ip string) (models.CDNResult, error) {
	if err := d.allowlist.Validate("cdncheck", []string{"-i", ip, "-resp"}); err != nil {
		return models.CDNResult{}, fmt.Errorf("cdncheck allowlist: %w", err)
	}
	cmd := exec.CommandContext(ctx, "cdncheck", "-i", ip, "-resp")
	out, err := cmd.Output()
	if err != nil {
		return models.CDNResult{}, fmt.Errorf("cdncheck: %w", err)
	}

	output := strings.TrimSpace(string(out))
	if output == "" {
		return models.CDNResult{IP: ip, IsCDN: false}, nil
	}

	// cdncheck -resp output: "cloudflare" or ""
	return models.CDNResult{
		IP:       ip,
		IsCDN:    true,
		Provider: output,
		Type:     "cdn",
	}, nil
}

// FilterCDNIPs checks a list of IPs and returns non-CDN IPs and CDN results.
func (d *Detector) FilterCDNIPs(ctx context.Context, ips []string) ([]string, []models.CDNResult, error) {
	if len(ips) == 0 {
		return nil, nil, fmt.Errorf("no IPs to classify")
	}

	input := strings.Join(ips, ",")
	if err := d.allowlist.Validate("cdncheck", []string{"-i", input, "-jsonl"}); err != nil {
		return nil, nil, fmt.Errorf("cdncheck allowlist: %w", err)
	}
	cmd := exec.CommandContext(ctx, "cdncheck", "-i", input, "-jsonl")
	out, err := cmd.Output()
	if err != nil {
		return nil, nil, fmt.Errorf("cdncheck: %w", err)
	}

	return ParseJSONLOutput(out, ips)
}
