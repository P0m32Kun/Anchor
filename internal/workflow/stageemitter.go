package workflow

import (
	"log"
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// StageEmitter upserts pipeline_run_stages rows and fans out stage events to
// SSE subscribers. It is the single source of truth for "a pipeline stage
// changed status" — Pipeline uses it for the main flow stages, and
// SlowScanOrchestrator uses it for post-pipeline slow scan stages, so both
// surfaces share identical DB and SSE semantics.
//
// When runID is empty (manual single-task invocation, bench runs) all methods
// are no-ops, matching the prior behavior of the freestanding setStage helpers.
type StageEmitter struct {
	queries  *db.Queries
	runID    string
	callback StageEventCallback
}

func NewStageEmitter(queries *db.Queries, runID string, cb StageEventCallback) *StageEmitter {
	return &StageEmitter{queries: queries, runID: runID, callback: cb}
}

// Set marks the stage running. Upsert: existing row → flip to running; missing
// row → insert with started_at=now.
func (e *StageEmitter) Set(stage StageID) {
	if e == nil || e.runID == "" {
		return
	}
	now := time.Now().UTC()
	if err := e.queries.UpdatePipelineRunStage(e.runID, string(stage)); err != nil {
		log.Printf("[stage-emitter] update run stage: %v", err)
	}

	existing, err := e.queries.GetPipelineRunStage(e.runID, string(stage))
	if err != nil {
		log.Printf("[stage-emitter] get stage record: %v", err)
	}
	if existing == nil {
		s := &models.PipelineRunStage{
			ID:        util.GenerateID(),
			RunID:     e.runID,
			Stage:     string(stage),
			Status:    models.StageStatusRunning,
			StartedAt: &now,
			CreatedAt: now,
		}
		if err := e.queries.CreatePipelineRunStage(s); err != nil {
			log.Printf("[stage-emitter] create stage record: %v", err)
		}
	} else {
		if err := e.queries.UpdatePipelineRunStageRecord(existing.ID, models.StageStatusRunning, "", nil); err != nil {
			log.Printf("[stage-emitter] update stage record: %v", err)
		}
	}

	if e.callback != nil {
		e.callback(e.runID, stage, "running", "")
	}
}

// Complete marks the stage finished successfully.
func (e *StageEmitter) Complete(stage StageID) {
	if e == nil || e.runID == "" {
		return
	}
	now := time.Now().UTC()
	existing, err := e.queries.GetPipelineRunStage(e.runID, string(stage))
	if err != nil {
		log.Printf("[stage-emitter] get stage record for complete: %v", err)
		return
	}
	if existing != nil {
		if err := e.queries.UpdatePipelineRunStageRecord(existing.ID, models.StageStatusCompleted, "", &now); err != nil {
			log.Printf("[stage-emitter] complete stage record: %v", err)
		}
	}
	if e.callback != nil {
		e.callback(e.runID, stage, "completed", "")
	}
}

// Fail marks the stage failed with an error message.
//
// If the stage has no prior row (Fix 3 path: ffuf is enabled but no dictionary
// was configured, so we never ran Set), Fail inserts a new row directly in
// failed state with started_at=completed_at=now — so a user reloading the run
// detail page after the SSE event has flushed still sees the failure persisted.
func (e *StageEmitter) Fail(stage StageID, errMsg string) {
	if e == nil || e.runID == "" {
		return
	}
	now := time.Now().UTC()
	existing, err := e.queries.GetPipelineRunStage(e.runID, string(stage))
	if err != nil {
		log.Printf("[stage-emitter] get stage record for fail: %v", err)
		return
	}
	if existing == nil {
		s := &models.PipelineRunStage{
			ID:          util.GenerateID(),
			RunID:       e.runID,
			Stage:       string(stage),
			Status:      models.StageStatusFailed,
			StartedAt:   &now,
			CompletedAt: &now,
			Error:       errMsg,
			CreatedAt:   now,
		}
		if err := e.queries.CreatePipelineRunStage(s); err != nil {
			log.Printf("[stage-emitter] create failed stage record: %v", err)
		}
	} else {
		if err := e.queries.UpdatePipelineRunStageRecord(existing.ID, models.StageStatusFailed, errMsg, &now); err != nil {
			log.Printf("[stage-emitter] fail stage record: %v", err)
		}
	}
	if e.callback != nil {
		e.callback(e.runID, stage, "failed", errMsg)
	}
}
