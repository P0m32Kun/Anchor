package stageagg

import (
	"log"
	"sync"
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/scanengine/core"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// StageEventCallback is called when a stage's state changes.
// It mirrors the workflow.StageEventCallback signature for SSE compatibility.
type StageEventCallback func(runID, stage, status, errMsg string)

// stageCounters tracks incremental work counts for a stage.
type stageCounters struct {
	total   int
	done    int
	running int
	failed  int
}

// Aggregator projects work-item state changes into pipeline_run_stages
// records and fires SSE callbacks.
type Aggregator struct {
	mu       sync.Mutex
	queries  *db.Queries
	runID    string
	callback StageEventCallback
	// track which stages have been created
	created map[string]bool
	// track round numbers per stage
	rounds map[string]int
	// incremental counters per stage (avoids full DB scan)
	counters map[string]*stageCounters
}

// NewAggregator creates a new Aggregator.
func NewAggregator(queries *db.Queries, runID string, callback StageEventCallback) *Aggregator {
	return &Aggregator{
		queries:  queries,
		runID:    runID,
		callback: callback,
		created:  make(map[string]bool),
		rounds:   make(map[string]int),
		counters: make(map[string]*stageCounters),
	}
}

// getOrCreateCounters returns counters for a stage, creating if needed.
func (a *Aggregator) getOrCreateCounters(stage string) *stageCounters {
	c, ok := a.counters[stage]
	if !ok {
		c = &stageCounters{}
		a.counters[stage] = c
	}
	return c
}

// OnWorkCreated is called when a new work item is created.
func (a *Aggregator) OnWorkCreated(action core.TaskAction) {
	a.mu.Lock()
	defer a.mu.Unlock()
	stage := core.ActionToStage[action]
	c := a.getOrCreateCounters(stage)
	c.total++
}

// OnWorkStarted is called when a work item transitions to running.
// It ensures the stage exists and updates work counts.
func (a *Aggregator) OnWorkStarted(action core.TaskAction) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	stage := core.ActionToStage[action]
	if err := a.ensureStage(stage); err != nil {
		return err
	}
	// Increment round on transition to running
	a.rounds[stage]++
	return a.updateStageCounts(stage)
}

// OnWorkCompleted is called when a work item reaches a terminal state.
func (a *Aggregator) OnWorkCompleted(action core.TaskAction) {
	a.mu.Lock()
	defer a.mu.Unlock()
	stage := core.ActionToStage[action]
	c := a.getOrCreateCounters(stage)
	c.done++
	if c.running > 0 {
		c.running--
	}
	if err := a.updateStageCounts(stage); err != nil {
		log.Printf("[stageagg] updateStageCounts error: %v", err)
	}
}

// OnWorkFailed is called when a work item fails.
func (a *Aggregator) OnWorkFailed(action core.TaskAction) {
	a.mu.Lock()
	defer a.mu.Unlock()
	stage := core.ActionToStage[action]
	c := a.getOrCreateCounters(stage)
	c.done++
	c.failed++
	if c.running > 0 {
		c.running--
	}
	if err := a.updateStageCounts(stage); err != nil {
		log.Printf("[stageagg] updateStageCounts error: %v", err)
	}
}

// OnWorkSkipped is called when a work item is skipped.
func (a *Aggregator) OnWorkSkipped(action core.TaskAction) {
	a.mu.Lock()
	defer a.mu.Unlock()
	stage := core.ActionToStage[action]
	c := a.getOrCreateCounters(stage)
	c.done++
	if c.running > 0 {
		c.running--
	}
	if err := a.updateStageCounts(stage); err != nil {
		log.Printf("[stageagg] updateStageCounts error: %v", err)
	}
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

// updateStageCounts updates DB with current counters and manages stage status.
func (a *Aggregator) updateStageCounts(stage string) error {
	c := a.getOrCreateCounters(stage)
	round := a.rounds[stage]

	if err := a.queries.UpdatePipelineRunStageWorkCounts(a.runID, stage, c.total, c.done, c.running, round); err != nil {
		return err
	}

	// Determine stage status
	status := "running"
	errMsg := ""
	if c.total > 0 && c.done == c.total {
		if c.failed > 0 {
			status = "failed"
			errMsg = "one or more work items failed"
		} else {
			status = "completed"
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

// LoadFromDB initializes counters from database (for engine restart recovery).
func (a *Aggregator) LoadFromDB() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	allWorks, err := a.queries.ListScanWorkItemsByRun(a.runID)
	if err != nil {
		return err
	}

	for _, w := range allWorks {
		stage := w.Stage
		c := a.getOrCreateCounters(stage)
		c.total++
		switch w.Status {
		case models.WorkStatusDone, models.WorkStatusSkipped:
			c.done++
		case models.WorkStatusFailed:
			c.done++
			c.failed++
		case models.WorkStatusRunning:
			c.running++
		}
	}
	return nil
}
