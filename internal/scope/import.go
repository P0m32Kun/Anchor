package scope

import (
	"encoding/csv"
	"io"
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

// ParseTXT parses a TXT import where each non-empty line is:
//
//	<type>:<value>  or  <value>
//
// Supported types: domain, url, ip, cidr. Default is domain.
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
		t := parseLine(line)
		targets = append(targets, t)
	}
	return targets, nil
}

// ParseCSV parses a CSV import. The first line is the header.
// Columns: type,value or just value. Default type is domain.
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

		var t ImportTarget
		switch len(record) {
		case 0:
			continue
		case 1:
			t = parseLine(strings.TrimSpace(record[0]))
		default:
			t = parseLine(strings.TrimSpace(record[0]) + ":" + strings.TrimSpace(record[1]))
		}

		if t.Value == "" {
			continue
		}
		targets = append(targets, t)
	}

	_ = hasHeader
	return targets, nil
}

// parseLine parses a single "<type>:<value>" or "<value>" line.
func parseLine(line string) ImportTarget {
	line = strings.TrimSpace(line)
	if line == "" {
		return ImportTarget{Type: models.TargetTypeDomain}
	}

	// Check for type prefix: domain:example.com
	if idx := strings.Index(line, ":"); idx > 0 {
		typ := strings.ToLower(strings.TrimSpace(line[:idx]))
		val := strings.TrimSpace(line[idx+1:])
		if val == "" {
			return ImportTarget{Type: models.TargetTypeDomain, Value: line}
		}
		switch typ {
		case "domain":
			return ImportTarget{Type: models.TargetTypeDomain, Value: val}
		case "url":
			return ImportTarget{Type: models.TargetTypeURL, Value: val}
		case "ip":
			return ImportTarget{Type: models.TargetTypeIP, Value: val}
		case "cidr":
			return ImportTarget{Type: models.TargetTypeCIDR, Value: val}
		default:
			// Unrecognized type, treat whole line as domain.
			return ImportTarget{Type: models.TargetTypeDomain, Value: line}
		}
	}

	// No type prefix, default to domain.
	return ImportTarget{Type: models.TargetTypeDomain, Value: line}
}

// isValidTargetType checks if the target type is one of the allowed values.
func isValidTargetType(t models.TargetType) bool {
	switch t {
	case models.TargetTypeDomain, models.TargetTypeURL, models.TargetTypeIP, models.TargetTypeCIDR:
		return true
	}
	return false
}
