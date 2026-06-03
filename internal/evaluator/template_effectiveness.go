package evaluator

import (
	"bytes"
	"os"

	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/parser"
)

// TemplateEffectivenessEvaluator evaluates template and dictionary effectiveness.
type TemplateEffectivenessEvaluator struct {
	queries *db.Queries
}

// NewTemplateEffectivenessEvaluator creates a new evaluator.
func NewTemplateEffectivenessEvaluator(queries *db.Queries) *TemplateEffectivenessEvaluator {
	return &TemplateEffectivenessEvaluator{queries: queries}
}

// EvaluateTemplateStats collects template hit statistics for a run.
func (e *TemplateEffectivenessEvaluator) EvaluateTemplateStats(runID string) (map[string]*TemplateStat, error) {
	hits, err := e.queries.GetTemplateHitStats(runID)
	if err != nil {
		return nil, err
	}

	result := make(map[string]*TemplateStat)
	for _, h := range hits {
		key := h.SourceTool + ":" + h.SourceRuleID
		stat := &TemplateStat{
			TemplateID:     h.SourceRuleID,
			SourceTool:     h.SourceTool,
			HitCount:       h.HitCount,
			ConfirmedCount: h.ConfirmedCount,
		}
		if h.HitCount > 0 {
			stat.Effectiveness = float64(h.ConfirmedCount) / float64(h.HitCount)
		}
		result[key] = stat
	}

	return result, nil
}

// EvaluateDictionaryStats collects dictionary hit statistics for a run.
// Parses ffuf work items to count unique hits per dictionary.
func (e *TemplateEffectivenessEvaluator) EvaluateDictionaryStats(runID string) (map[string]*DictionaryStat, error) {
	stats, err := e.queries.GetDictionaryHitStats(runID)
	if err != nil {
		return nil, err
	}

	result := make(map[string]*DictionaryStat)
	for _, s := range stats {
		// Parse the ffuf output file
		if s.TaskID == "" {
			continue
		}

		// Get the artifact path from the task
		artifacts, err := e.queries.ListRawArtifactsByTask(s.TaskID)
		if err != nil {
			continue
		}

		for _, a := range artifacts {
			if a.Type != "stdout" {
				continue
			}

			// Read and parse ffuf output
			data, err := os.ReadFile(a.Path)
			if err != nil {
				continue
			}

			results, _ := parser.ParseFfufOutput(bytes.NewReader(data))
			if len(results) == 0 {
				continue
			}

			// Count unique paths and hits
			uniquePaths := make(map[string]bool)
			for _, r := range results {
				uniquePaths[r.URL] = true
			}

			// For now, use a generic dictionary name
			// In a real implementation, you would extract the dictionary name from the ffuf command
			dictName := "unknown"
			key := dictName + ":" + s.TaskID

			stat, exists := result[key]
			if !exists {
				stat = &DictionaryStat{
					DictionaryName: dictName,
					UsedInTool:     "ffuf",
				}
				result[key] = stat
			}

			stat.UniquePaths += len(uniquePaths)
			// Hit rate would be calculated as hits / total requests
			// For now, we don't have the total request count, so we leave it as 0
		}
	}

	return result, nil
}
