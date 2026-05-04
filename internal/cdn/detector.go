package cdn

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"

	"github.com/P0m32Kun/Anchor/internal/models"
)

// Detector detects CDN usage for IPs using cdncheck CLI.
type Detector struct{}

// NewDetector creates a new CDN detector.
func NewDetector() *Detector {
	return &Detector{}
}

// CheckIP checks if an IP is behind a CDN.
func (d *Detector) CheckIP(ctx context.Context, ip string) (models.CDNResult, error) {
	cmd := exec.CommandContext(ctx, "cdncheck", "-i", ip, "-resp")
	out, err := cmd.Output()
	if err != nil {
		// If cdncheck fails, assume not CDN
		return models.CDNResult{IP: ip, IsCDN: false}, nil
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
		return nil, nil, nil
	}

	// Use cdncheck with JSON output for batch processing
	input := strings.Join(ips, ",")
	cmd := exec.CommandContext(ctx, "cdncheck", "-i", input, "-jsonl")
	out, err := cmd.Output()
	if err != nil {
		// If cdncheck fails, return all IPs as non-CDN
		return ips, nil, nil
	}

	var cdnResults []models.CDNResult
	cdnIPSet := make(map[string]bool)

	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var result struct {
			IP       string `json:"ip"`
			CDN      bool   `json:"cdn"`
			Provider string `json:"provider"`
			Type     string `json:"type"`
		}
		if err := json.Unmarshal([]byte(line), &result); err != nil {
			continue
		}

		if result.CDN {
			cdnResults = append(cdnResults, models.CDNResult{
				IP:       result.IP,
				IsCDN:    true,
				Provider: result.Provider,
				Type:     result.Type,
			})
			cdnIPSet[result.IP] = true
		}
	}

	var nonCDNIPs []string
	for _, ip := range ips {
		if !cdnIPSet[ip] {
			nonCDNIPs = append(nonCDNIPs, ip)
		}
	}

	return nonCDNIPs, cdnResults, nil
}
