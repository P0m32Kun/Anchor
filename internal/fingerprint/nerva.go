package fingerprint

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/P0m32Kun/Anchor/internal/models"
)

// NervaResult represents a single service fingerprint from nerva.
type NervaResult struct {
	Host      string                 `json:"host"`
	IP        string                 `json:"ip"`
	Port      int                    `json:"port"`
	Protocol  string                 `json:"protocol"`  // http, mysql, redis, ssh, etc.
	Transport string                 `json:"transport"` // tcp | udp | sctp
	Metadata  map[string]interface{} `json:"metadata"`  // service-specific metadata
}

// IsWebService checks if the detected service is a web service.
func IsWebService(result NervaResult) bool {
	return result.Protocol == "http" || result.Protocol == "https"
}

// BuildNervaCommand builds the nerva CLI command for fingerprinting.
func BuildNervaCommand(targets []string) []string {
	// nerva -t host1:port1,host2:port2 --json
	return []string{
		"nerva",
		"--json",
		"-t", strings.Join(targets, ","),
	}
}

// RunNerva runs nerva against a list of targets and returns the results.
func RunNerva(ctx context.Context, targets []string) ([]NervaResult, error) {
	if len(targets) == 0 {
		return nil, nil
	}

	cmd := exec.CommandContext(ctx, "nerva", "--json", "-t", strings.Join(targets, ","))
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("nerva failed: %w, stderr: %s", err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("nerva failed: %w", err)
	}

	return ParseNervaOutput(string(out)), nil
}

// ParseNervaOutput parses nerva JSONL output.
func ParseNervaOutput(output string) []NervaResult {
	var results []NervaResult
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var result NervaResult
		if err := json.Unmarshal([]byte(line), &result); err != nil {
			continue
		}
		results = append(results, result)
	}
	return results
}

// ConvertToServiceFingerprints converts nerva results to ServiceFingerprint models.
func ConvertToServiceFingerprints(projectID string, results []NervaResult) []models.ServiceFingerprint {
	var fps []models.ServiceFingerprint
	for _, r := range results {
		fp := models.ServiceFingerprint{
			IP:       r.IP,
			Port:     r.Port,
			Protocol: r.Transport,
			IsWeb:    IsWebService(r),
			Service:  r.Protocol,
			Metadata: r.Metadata,
			Source:   "nerva",
		}
		if fp.IP == "" {
			fp.IP = r.Host
		}
		fps = append(fps, fp)
	}
	return fps
}

// SplitByServiceType splits nerva results into web and non-web services.
func SplitByServiceType(results []NervaResult) (web []NervaResult, nonWeb []NervaResult) {
	for _, r := range results {
		if IsWebService(r) {
			web = append(web, r)
		} else {
			nonWeb = append(nonWeb, r)
		}
	}
	return
}
