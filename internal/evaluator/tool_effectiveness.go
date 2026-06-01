package evaluator

import (
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
)

// ToolEffectivenessEvaluator evaluates tool effectiveness.
type ToolEffectivenessEvaluator struct {
	queries *db.Queries
}

// NewToolEffectivenessEvaluator creates a new evaluator.
func NewToolEffectivenessEvaluator(queries *db.Queries) *ToolEffectivenessEvaluator {
	return &ToolEffectivenessEvaluator{queries: queries}
}

// Evaluate collects tool effectiveness metrics for a run.
func (e *ToolEffectivenessEvaluator) Evaluate(runID string) (map[string]*ToolStat, error) {
	// Get tool stats
	toolStats, err := e.queries.GetToolStatsByRun(runID)
	if err != nil {
		return nil, err
	}

	// Get error stats
	errorStats, err := e.queries.GetToolErrorStatsByRun(runID)
	if err != nil {
		return nil, err
	}

	// Build error map
	errorMap := make(map[string][]ErrorCount)
	for _, es := range errorStats {
		errorMap[es.Tool] = append(errorMap[es.Tool], ErrorCount{
			Error: es.Error,
			Count: es.Count,
		})
	}

	// Build result
	result := make(map[string]*ToolStat)
	for _, ts := range toolStats {
		stat := &ToolStat{
			ToolName:     ts.Tool,
			TotalCalls:   ts.TotalCalls,
			SuccessCount: ts.SuccessCount,
			FailedCount:  ts.FailedCount,
			SkippedCount: ts.SkippedCount,
			AvgDuration:  time.Duration(ts.AvgDuration * float64(time.Second)),
			CommonErrors: errorMap[ts.Tool],
		}

		if ts.TotalCalls > 0 {
			stat.SuccessRate = float64(ts.SuccessCount) / float64(ts.TotalCalls)
		}

		if ts.AvgDuration > 0 && ts.SuccessCount > 0 {
			stat.OutputRate = float64(ts.SuccessCount) / ts.AvgDuration
		}

		result[ts.Tool] = stat
	}

	return result, nil
}
