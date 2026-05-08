package workflow

import (
	"log"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// StageID identifies a pipeline stage.
type StageID string

const (
	StageClassify    StageID = "classify"
	StageSearch      StageID = "search"
	StageSubdomain   StageID = "subdomain"
	StageResolve     StageID = "resolve"
	StageCDNFilter   StageID = "cdn_filter"
	StageAlive       StageID = "alive"
	StagePortScan    StageID = "portscan"
	StageFingerprint StageID = "fingerprint"
	StageHTTPX       StageID = "httpx"
	StageVuln        StageID = "vuln"
)

// StageEventCallback is invoked when a pipeline stage changes state.
type StageEventCallback func(runID string, stage StageID, status string, errMsg string)

func (p *Pipeline) setStage(stage StageID) {
	if p.runID == "" {
		return
	}
	now := time.Now().UTC()
	if err := p.queries.UpdatePipelineRunStage(p.runID, string(stage)); err != nil {
		log.Printf("[pipeline] update stage: %v", err)
	}

	// Upsert stage record: if exists mark running, else create.
	existing, err := p.queries.GetPipelineRunStage(p.runID, string(stage))
	if err != nil {
		log.Printf("[pipeline] get stage record: %v", err)
	}
	if existing == nil {
		s := &models.PipelineRunStage{
			ID:        util.GenerateID(),
			RunID:     p.runID,
			Stage:     string(stage),
			Status:    models.StageStatusRunning,
			StartedAt: &now,
			CreatedAt: now,
		}
		if err := p.queries.CreatePipelineRunStage(s); err != nil {
			log.Printf("[pipeline] create stage record: %v", err)
		}
	} else {
		if err := p.queries.UpdatePipelineRunStageRecord(existing.ID, models.StageStatusRunning, "", nil); err != nil {
			log.Printf("[pipeline] update stage record: %v", err)
		}
	}

	if p.onStageChange != nil {
		p.onStageChange(p.runID, stage, "running", "")
	}
}

func (p *Pipeline) completeStage(stage StageID) {
	if p.runID == "" {
		return
	}
	now := time.Now().UTC()
	existing, err := p.queries.GetPipelineRunStage(p.runID, string(stage))
	if err != nil {
		log.Printf("[pipeline] get stage record for complete: %v", err)
		return
	}
	if existing != nil {
		if err := p.queries.UpdatePipelineRunStageRecord(existing.ID, models.StageStatusCompleted, "", &now); err != nil {
			log.Printf("[pipeline] complete stage record: %v", err)
		}
	}
	if p.onStageChange != nil {
		p.onStageChange(p.runID, stage, "completed", "")
	}
}

func (p *Pipeline) failStage(stage StageID, errMsg string) {
	if p.runID == "" {
		return
	}
	now := time.Now().UTC()
	existing, err := p.queries.GetPipelineRunStage(p.runID, string(stage))
	if err != nil {
		log.Printf("[pipeline] get stage record for fail: %v", err)
		return
	}
	if existing != nil {
		if err := p.queries.UpdatePipelineRunStageRecord(existing.ID, models.StageStatusFailed, errMsg, &now); err != nil {
			log.Printf("[pipeline] fail stage record: %v", err)
		}
	}
	if p.onStageChange != nil {
		p.onStageChange(p.runID, stage, "failed", errMsg)
	}
}
