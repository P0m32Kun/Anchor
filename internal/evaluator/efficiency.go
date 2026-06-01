package evaluator

import (
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
)

// EfficiencyEvaluator evaluates scan execution efficiency.
type EfficiencyEvaluator struct {
	queries *db.Queries
}

// NewEfficiencyEvaluator creates a new evaluator.
func NewEfficiencyEvaluator(queries *db.Queries) *EfficiencyEvaluator {
	return &EfficiencyEvaluator{queries: queries}
}

// EfficiencyResult holds efficiency evaluation results.
type EfficiencyResult struct {
	TotalDuration  time.Duration
	StageDurations map[string]time.Duration
	StageStatuses  map[string]string
}

// Evaluate collects efficiency metrics for a run.
func (e *EfficiencyEvaluator) Evaluate(runID string) (*EfficiencyResult, error) {
	// Get run info for total duration
	run, err := e.queries.GetPipelineRun(runID)
	if err != nil {
		return nil, err
	}

	result := &EfficiencyResult{
		StageDurations: make(map[string]time.Duration),
		StageStatuses:  make(map[string]string),
	}

	if run.CompletedAt != nil {
		result.TotalDuration = run.CompletedAt.Sub(run.StartedAt)
	}

	// Get stage stats
	stageStats, err := e.queries.GetPipelineRunStageStats(runID)
	if err != nil {
		return nil, err
	}

	for _, ss := range stageStats {
		result.StageDurations[ss.Stage] = time.Duration(ss.Duration * float64(time.Second))
		result.StageStatuses[ss.Stage] = ss.Status
	}

	return result, nil
}
