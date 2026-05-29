package stageagg

import (
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/scanengine/core"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// StageEventCallback is called when a stage's state changes.
// It mirrors the workflow.StageEventCallback signature for SSE compatibility.
type StageEventCallback func(runID, stage, status, errMsg string)

// Aggregator projects work-item state changes into pipeline_run_stages
// records and fires SSE callbacks.
type Aggregator struct {
	queries  *db.Queries
	runID    string
	callback StageEventCallback
	// track which stages have been created
	created map[string]bool
	// track round numbers per stage
	rounds map[string]int
}

// NewAggregator creates a new Aggregator.
func NewAggregator(queries *db.Queries, runID string, callback StageEventCallback) *Aggregator {
	return &Aggregator{
		queries:  queries,
		runID:    runID,
		callback: callback,
		created:  make(map[string]bool),
		rounds:   make(map[string]int),
	}
}

// OnWorkStarted is called when a work item transitions to running.
// It ensures the stage exists and updates work counts.
func (a *Aggregator) OnWorkStarted(action core.TaskAction) error {
	stage := core.ActionToStage[action]
	if err := a.ensureStage(stage); err != nil {
		return err
	}
	// Increment round on transition to running
	a.rounds[stage]++
	return a.refreshCounts(stage)
}

// OnWorkCompleted is called when a work item reaches a terminal state.
func (a *Aggregator) OnWorkCompleted(action core.TaskAction) error {
	stage := core.ActionToStage[action]
	return a.refreshCounts(stage)
}

// ensureStage creates the pipeline_run_stages record if it doesn't exist yet.
func (a *Aggregator) ensureStage(stage string) error {
	if a.created[stage] {
		return nil
	}
	existing, err := a.queries.GetPipelineRunStage(a.runID, stage)
	if err != nil {
		return err
	}
	if existing == nil {
		now := time.Now().UTC()
		s := &models.PipelineRunStage{
			ID:        util.GenerateID(),
			RunID:     a.runID,
			Stage:     stage,
			Status:    models.StageStatusRunning,
			StartedAt: &now,
			CreatedAt: now,
		}
		if err := a.queries.CreatePipelineRunStage(s); err != nil {
			return err
		}
	}
	a.created[stage] = true
	return nil
}

// refreshCounts recalculates work counts for a stage and updates the DB.
func (a *Aggregator) refreshCounts(stage string) error {
	// Count works for this run+stage
	allWorks, err := a.queries.ListScanWorkItemsByRun(a.runID)
	if err != nil {
		return err
	}
	var total, done, running int
	for _, w := range allWorks {
		if w.Stage != stage {
			continue
		}
		total++
		switch w.Status {
		case models.WorkStatusDone, models.WorkStatusSkipped, models.WorkStatusFailed:
			done++
		case models.WorkStatusRunning:
			running++
		}
	}

	round := a.rounds[stage]
	if err := a.queries.UpdatePipelineRunStageWorkCounts(a.runID, stage, total, done, running, round); err != nil {
		return err
	}

	// Determine stage status
	status := "running"
	errMsg := ""
	if total > 0 && done == total {
		status = "completed"
		// Check if any failed
		for _, w := range allWorks {
			if w.Stage == stage && w.Status == models.WorkStatusFailed {
				status = "failed"
				errMsg = "one or more work items failed"
				break
			}
		}
	}

	// Update stage status
	existing, err := a.queries.GetPipelineRunStage(a.runID, stage)
	if err != nil {
		return err
	}
	if existing != nil && string(existing.Status) != status {
		var completedAt *time.Time
		if status == "completed" || status == "failed" {
			now := time.Now().UTC()
			completedAt = &now
		}
		if err := a.queries.UpdatePipelineRunStageRecord(existing.ID, models.PipelineRunStageStatus(status), errMsg, completedAt); err != nil {
			return err
		}
	}

	// Fire SSE callback
	if a.callback != nil {
		a.callback(a.runID, stage, status, errMsg)
	}
	return nil
}
