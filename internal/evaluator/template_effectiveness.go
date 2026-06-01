package evaluator

import "github.com/P0m32Kun/Anchor/internal/db"

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
// Note: Dictionary stats are derived from ffuf work items.
func (e *TemplateEffectivenessEvaluator) EvaluateDictionaryStats(runID string) (map[string]*DictionaryStat, error) {
	// TODO: Implement dictionary stats when ffuf output parsing is available
	// For now, return empty stats
	return make(map[string]*DictionaryStat), nil
}
