package scope

import (
	"encoding/csv"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"

	"github.com/P0m32Kun/Anchor/internal/models"
)

// ImportTarget is a parsed target entry from TXT or CSV import.
type ImportTarget struct {
	Type  models.TargetType
	Value string
}

// DeniedTarget represents a target that was denied during import with its reason.
type DeniedTarget struct {
	Value  string `json:"value"`
	Reason string `json:"reason"`
}

// ImportResult holds statistics from a batch import.
type ImportResult struct {
	Imported      int              `json:"imported"`
	Duplicates    int              `json:"duplicates"`
	Denied        int              `json:"denied"`
	Errors        int              `json:"errors"`
	Targets       []*models.Target `json:"targets"`
	DeniedTargets []DeniedTarget   `json:"denied_targets"`
}

// maxImportTargets is the maximum number of targets allowed in a single import.
const maxImportTargets = 10000

// ParseTXT parses a TXT import where each non-empty line is:
//
//	<type>:<value>  or  <value>
//
// Supported types: domain, url, ip, cidr. Default is domain.
// Lines may contain comma-separated values and IP hyphen ranges
// (e.g. "ip:192.168.0.1-10").
func ParseTXT(r io.Reader) ([]ImportTarget, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(data), "\n")
	var targets []ImportTarget
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parsed := parseLine(line)
		if len(targets)+len(parsed) > maxImportTargets {
			return nil, fmt.Errorf("too many targets: maximum %d allowed", maxImportTargets)
		}
		targets = append(targets, parsed...)
	}
	return targets, nil
}

// ParseCSV parses a CSV import. The first line is the header.
// Columns: type,value or just value. Default type is domain.
// Values may contain comma-separated entries and IP hyphen ranges
// within each cell.
func ParseCSV(r io.Reader) ([]ImportTarget, error) {
	reader := csv.NewReader(r)
	reader.TrimLeadingSpace = true

	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	var targets []ImportTarget
	hasHeader := false
	for i, record := range records {
		// Detect header: first row with "type"/"value" column names.
		if i == 0 && len(record) >= 1 {
			first := strings.ToLower(strings.TrimSpace(record[0]))
			if first == "type" || first == "target_type" || first == "value" || first == "target" {
				hasHeader = true
				continue
			}
		}

		var line string
		switch len(record) {
		case 0:
			continue
		case 1:
			line = strings.TrimSpace(record[0])
		default:
			line = strings.TrimSpace(record[0]) + ":" + strings.TrimSpace(record[1])
		}

		if line == "" {
			continue
		}
		parsed := parseLine(line)
		if len(targets)+len(parsed) > maxImportTargets {
			return nil, fmt.Errorf("too many targets: maximum %d allowed", maxImportTargets)
		}
		targets = append(targets, parsed...)
	}

	_ = hasHeader
	return targets, nil
}

// parseLine parses a single "<type>:<value>" or "<value>" line.
// It returns a slice because a single line may expand into multiple
// targets via comma separation or IP hyphen ranges.
func parseLine(line string) []ImportTarget {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}

	var explicitType models.TargetType
	hasExplicitType := false
	content := line

	// Check for type prefix: domain:example.com
	if idx := strings.Index(line, ":"); idx > 0 {
		typ := strings.ToLower(strings.TrimSpace(line[:idx]))
		val := strings.TrimSpace(line[idx+1:])
		if val != "" {
			switch typ {
			case "domain":
				explicitType = models.TargetTypeDomain
				hasExplicitType = true
				content = val
			case "url":
				explicitType = models.TargetTypeURL
				hasExplicitType = true
				content = val
			case "ip":
				explicitType = models.TargetTypeIP
				hasExplicitType = true
				content = val
			case "cidr":
				explicitType = models.TargetTypeCIDR
				hasExplicitType = true
				content = val
			case "company":
				explicitType = models.TargetTypeCompany
				hasExplicitType = true
				content = val
			default:
				// Unrecognized type, treat whole line as domain.
				hasExplicitType = false
				content = line
			}
		} else {
			// Empty value after colon (e.g. "domain:")
			hasExplicitType = false
			content = line
		}
	}

	// Expand comma-separated values.
	parts := expandCommas(content)
	var targets []ImportTarget

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		typ := explicitType
		if !hasExplicitType {
			typ = DetectType(part)
		}

		// If type is ip, try expanding hyphen ranges.
		if typ == models.TargetTypeIP {
			ips, err := expandIPRange(part)
			if err == nil {
				for _, ip := range ips {
					targets = append(targets, ImportTarget{Type: models.TargetTypeIP, Value: ip})
				}
				continue
			}
		}

		targets = append(targets, ImportTarget{Type: typ, Value: part})
	}

	return targets
}

// expandCommas splits a line by commas and trims whitespace,
// filtering out empty segments.
func expandCommas(line string) []string {
	parts := strings.Split(line, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// DetectType automatically infers the target type from a raw value.
//
// Rules:
//   - http:// or https:// prefix  → url
//   - Contains "/" and valid CIDR → cidr
//   - Valid IPv4 or IPv4 range    → ip
//   - Looks like a company name   → company
//   - Otherwise                   → domain
func DetectType(val string) models.TargetType {
	if strings.HasPrefix(val, "http://") || strings.HasPrefix(val, "https://") {
		return models.TargetTypeURL
	}
	if strings.Contains(val, "/") {
		if _, _, err := net.ParseCIDR(val); err == nil {
			return models.TargetTypeCIDR
		}
	}
	if looksLikeIPv4(val) {
		return models.TargetTypeIP
	}
	if isLikelyCompanyName(val) {
		return models.TargetTypeCompany
	}
	return models.TargetTypeDomain
}

// isLikelyCompanyName checks if a value looks like a company name rather than a domain.
// Heuristics: contains spaces, Chinese characters, or common company suffixes.
func isLikelyCompanyName(val string) bool {
	val = strings.TrimSpace(val)
	if val == "" {
		return false
	}

	// Contains spaces → likely a company name
	if strings.Contains(val, " ") {
		return true
	}

	// Contains Chinese characters → likely a Chinese company name
	for _, r := range val {
		if r >= 0x4e00 && r <= 0x9fff {
			return true
		}
	}

	// Common company suffixes
	suffixes := []string{
		" inc", " corp", " ltd", " llc", " limited", " corporation",
		" company", " group", " holdings", " technologies", " solutions",
		"科技", "技术", "有限公司", "集团", "股份", "公司",
	}
	lower := strings.ToLower(val)
	for _, s := range suffixes {
		if strings.HasSuffix(lower, s) {
			return true
		}
	}

	return false
}

// looksLikeIPv4 reports whether val is a valid IPv4 address or an
// IPv4 hyphen range (e.g. "192.168.0.1-10").
func looksLikeIPv4(val string) bool {
	// Standard IPv4 address.
	if ip := net.ParseIP(val); ip != nil && ip.To4() != nil {
		return true
	}

	// Hyphen range in the last octet.
	lastDot := strings.LastIndex(val, ".")
	if lastDot == -1 {
		return false
	}
	if !strings.Contains(val[lastDot+1:], "-") {
		return false
	}
	prefix := val[:lastDot]
	return strings.Count(prefix, ".") == 2
}

// expandIPRange expands an IPv4 hyphen range such as "192.168.0.1-10"
// into a slice of individual IP strings.
//
// Constraints:
//   - Only the last octet may vary.
//   - Maximum 256 addresses per range.
//   - Each octet must be within 0-255.
func expandIPRange(s string) ([]string, error) {
	lastDot := strings.LastIndex(s, ".")
	if lastDot == -1 {
		return nil, fmt.Errorf("invalid IP range format: %s", s)
	}

	prefix := s[:lastDot]
	suffix := s[lastDot+1:]

	hyphenIdx := strings.Index(suffix, "-")
	if hyphenIdx == -1 {
		// Not a range — validate as a plain IPv4 address.
		if ip := net.ParseIP(s); ip == nil || ip.To4() == nil {
			return nil, fmt.Errorf("invalid IPv4: %s", s)
		}
		return []string{s}, nil
	}

	startStr := suffix[:hyphenIdx]
	endStr := suffix[hyphenIdx+1:]

	start, err := strconv.Atoi(startStr)
	if err != nil {
		return nil, fmt.Errorf("invalid range start %q: %w", startStr, err)
	}
	end, err := strconv.Atoi(endStr)
	if err != nil {
		return nil, fmt.Errorf("invalid range end %q: %w", endStr, err)
	}

	if start > end {
		return nil, fmt.Errorf("invalid range: start %d > end %d", start, end)
	}
	if start < 0 || end > 255 {
		return nil, fmt.Errorf("range %d-%d out of bounds (0-255)", start, end)
	}
	count := end - start + 1
	if count > 256 {
		return nil, fmt.Errorf("range too large: %d IPs (max 256)", count)
	}

	// Validate prefix by checking the first IP in the range.
	firstIP := fmt.Sprintf("%s.%d", prefix, start)
	if ip := net.ParseIP(firstIP); ip == nil || ip.To4() == nil {
		return nil, fmt.Errorf("invalid IP prefix: %s", prefix)
	}

	results := make([]string, 0, count)
	for i := start; i <= end; i++ {
		results = append(results, fmt.Sprintf("%s.%d", prefix, i))
	}
	return results, nil
}

// isValidTargetType checks if the target type is one of the allowed values.
func isValidTargetType(t models.TargetType) bool {
	switch t {
	case models.TargetTypeDomain, models.TargetTypeURL, models.TargetTypeIP, models.TargetTypeCIDR:
		return true
	}
	return false
}
