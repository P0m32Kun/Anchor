package executor

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/scanengine/core"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// ParseSpoorOutput parses Spoor JSONL stdout into:
//   - endpoints: DiscoveryAsset (AssetHTTPPath) for kind=endpoint findings
//   - findings: models.Finding for kind=secret findings
//   - kind=path findings are logged only, not returned
func ParseSpoorOutput(stdout []byte, runID, projectID string) ([]*core.DiscoveryAsset, []*models.Finding, error) {
	var endpoints []*core.DiscoveryAsset
	var findings []*models.Finding
	seen := make(map[string]bool)

	lines := strings.Split(strings.TrimSpace(string(stdout)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var entry spoorFinding
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			log.Printf("[scanengine] spoor: skip malformed line: %v", err)
			continue
		}

		switch entry.Kind {
		case "endpoint":
			if entry.Value == "" {
				continue
			}
			if seen[entry.Value] {
				continue
			}
			seen[entry.Value] = true
			endpoints = append(endpoints, &core.DiscoveryAsset{
				ID:              util.GenerateID(),
				Type:            core.AssetHTTPPath,
				Value:           entry.Value,
				NormalizedValue: entry.Value,
				SourceTool:      "spoor",
			})

		case "secret":
			if entry.Value == "" {
				continue
			}
			dedupKey := fmt.Sprintf("%s:%s:%s", runID, entry.File, entry.Value)
			if seen[dedupKey] {
				continue
			}
			seen[dedupKey] = true
			f := &models.Finding{
				ID:           util.GenerateID(),
				ProjectID:    projectID,
				SourceTool:   "spoor",
				SourceRuleID: entry.SecretType,
				DedupKey:     dedupKey,
				Title:        fmt.Sprintf("Secret: %s in %s", entry.SecretType, entry.File),
				Severity:     mapSpoorSeverity(entry.Severity),
				Confidence:   mapSpoorConfidence(entry.Confidence),
				Priority:     2,
				Status:       models.FindingPendingReview,
				Summary:      entry.Origin.Snippet,
			}
			if runID != "" {
				f.RunID = &runID
			}
			findings = append(findings, f)

		case "path":
			// Log only, not persisted
			log.Printf("[scanengine] spoor found path in %s: %s", entry.File, entry.Value)
		}
	}

	return endpoints, findings, nil
}

// spoorFinding represents a single finding from Spoor JSONL output.
type spoorFinding struct {
	File       string        `json:"file"`
	Kind       string        `json:"kind"`
	Value      string        `json:"value"`
	Confidence string        `json:"confidence"`
	Method     string        `json:"method"`
	SecretType string        `json:"secret_type"`
	Severity   string        `json:"severity"`
	Origin     spoorOrigin   `json:"origin"`
}

// spoorOrigin represents the origin context of a Spoor finding.
type spoorOrigin struct {
	Pattern string `json:"pattern"`
	Snippet string `json:"snippet"`
	Line    int    `json:"line"`
	Column  int    `json:"column"`
}

// mapSpoorSeverity maps Spoor severity strings to Anchor FindingSeverity.
func mapSpoorSeverity(s string) models.FindingSeverity {
	switch strings.ToLower(s) {
	case "critical":
		return models.SeverityCritical
	case "high":
		return models.SeverityHigh
	case "medium":
		return models.SeverityMedium
	case "low":
		return models.SeverityLow
	default:
		return models.SeverityInfo
	}
}

// mapSpoorConfidence maps Spoor confidence strings to numeric values.
func mapSpoorConfidence(c string) int {
	switch strings.ToLower(c) {
	case "high":
		return 90
	case "medium":
		return 60
	case "low":
		return 30
	default:
		return 50
	}
}
