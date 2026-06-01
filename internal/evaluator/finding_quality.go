package evaluator

import "github.com/P0m32Kun/Anchor/internal/db"

// FindingQualityEvaluator evaluates finding quality.
type FindingQualityEvaluator struct {
	queries *db.Queries
}

// NewFindingQualityEvaluator creates a new evaluator.
func NewFindingQualityEvaluator(queries *db.Queries) *FindingQualityEvaluator {
	return &FindingQualityEvaluator{queries: queries}
}

// FindingQualityResult holds finding quality evaluation results.
type FindingQualityResult struct {
	FindingsBySeverity map[string]int
	FindingsByStatus   map[string]int
	AvgConfidence      float64
	UnlinkedFindings   int
	TotalFindings      int
}

// Evaluate collects finding quality metrics for a run.
func (e *FindingQualityEvaluator) Evaluate(runID string) (*FindingQualityResult, error) {
	result := &FindingQualityResult{
		FindingsBySeverity: make(map[string]int),
		FindingsByStatus:   make(map[string]int),
	}

	// Get severity stats
	severityStats, err := e.queries.GetFindingStatsBySeverity(runID)
	if err != nil {
		return nil, err
	}
	for _, ss := range severityStats {
		result.FindingsBySeverity[ss.Severity] = ss.Count
		result.TotalFindings += ss.Count
	}

	// Get status stats
	statusStats, err := e.queries.GetFindingStatsByStatus(runID)
	if err != nil {
		return nil, err
	}
	for _, ss := range statusStats {
		result.FindingsByStatus[ss.Status] = ss.Count
	}

	// Get average confidence
	result.AvgConfidence, err = e.queries.GetFindingAvgConfidence(runID)
	if err != nil {
		return nil, err
	}

	// Get unlinked findings
	result.UnlinkedFindings, err = e.queries.GetUnlinkedFindingCount(runID)
	if err != nil {
		return nil, err
	}

	return result, nil
}
