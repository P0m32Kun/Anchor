package cdn

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/P0m32Kun/Anchor/internal/models"
)

// cdncheckJSONL mirrors projectdiscovery/cdncheck internal/runner.Output JSONL fields.
type cdncheckJSONL struct {
	Input     string `json:"input"`
	IP        string `json:"ip"`
	CDN       bool   `json:"cdn"`
	CDNName   string `json:"cdn_name"`
	Cloud     bool   `json:"cloud"`
	CloudName string `json:"cloud_name"`
	WAF       bool   `json:"waf"`
	WAFName   string `json:"waf_name"`
}

func (r cdncheckJSONL) matched() bool {
	return r.CDN || r.Cloud || r.WAF
}

func (r cdncheckJSONL) endpointIP() string {
	if r.IP != "" {
		return r.IP
	}
	return r.Input
}

func (r cdncheckJSONL) providerAndType() (provider, typ string) {
	switch {
	case r.WAF:
		return r.WAFName, "waf"
	case r.Cloud:
		return r.CloudName, "cloud"
	case r.CDN:
		return r.CDNName, "cdn"
	default:
		return "", ""
	}
}

// ParseJSONLOutput parses cdncheck -jsonl stdout.
// cdncheck only emits lines for IPs that matched CDN, cloud, or WAF ranges; empty stdout means no hits.
func ParseJSONLOutput(out []byte, ips []string) ([]string, []models.CDNResult, error) {
	cdnIPSet := make(map[string]bool)
	var cdnResults []models.CDNResult

	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line[0] != '{' {
			continue
		}

		var row cdncheckJSONL
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			return nil, nil, fmt.Errorf("parse cdncheck jsonl: %w", err)
		}
		if !row.matched() {
			continue
		}

		ip := row.endpointIP()
		if ip == "" {
			continue
		}
		provider, typ := row.providerAndType()
		cdnResults = append(cdnResults, models.CDNResult{
			IP:       ip,
			IsCDN:    true,
			Provider: provider,
			Type:     typ,
		})
		cdnIPSet[ip] = true
	}

	var nonCDNIPs []string
	for _, ip := range ips {
		if !cdnIPSet[ip] {
			nonCDNIPs = append(nonCDNIPs, ip)
		}
	}
	return nonCDNIPs, cdnResults, nil
}
