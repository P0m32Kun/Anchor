package recovery

import (
	"log"
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/models"
)

// RecoverOrphanRuns marks pipeline runs that were still running when the server
// restarted as failed and clears zombie work items.
func RecoverOrphanRuns(q *db.Queries) (int, error) {
	runs, err := q.ListPipelineRunsByStatus("running")
	if err != nil {
		return 0, err
	}
	if len(runs) == 0 {
		return 0, nil
	}

	now := time.Now().UTC()
	recovered := 0
	for _, run := range runs {
		if run == nil {
			continue
		}
		if err := failOrphanWorkItems(q, run.ID, now); err != nil {
			log.Printf("[recovery] orphan work items for run %s: %v", run.ID, err)
		}
		_ = q.UpdatePipelineRunEngineState(run.ID, "stopped")
		_ = q.UpdatePipelineRunError(run.ID, "orphan: server restarted during scan")
		if err := q.UpdatePipelineRunStatus(run.ID, "failed"); err != nil {
			log.Printf("[recovery] mark run %s failed: %v", run.ID, err)
			continue
		}
		recovered++
		log.Printf("[recovery] recovered orphan run %s (project %s)", run.ID, run.ProjectID)
	}
	return recovered, nil
}

func failOrphanWorkItems(q *db.Queries, runID string, now time.Time) error {
	running, err := q.ListScanWorkItemsByRunAndStatus(runID, models.WorkStatusRunning)
	if err != nil {
		return err
	}
	for _, w := range running {
		_ = q.UpdateScanWorkItemError(w.ID, models.WorkStatusFailed, "orphan: server restarted while work was running", &now)
	}

	pending, err := q.ListScanWorkItemsByRunAndStatus(runID, models.WorkStatusPending)
	if err != nil {
		return err
	}
	for _, w := range pending {
		_ = q.UpdateScanWorkItemSkip(w.ID, models.WorkStatusSkipped, "orphan: server restarted", &now)
	}
	return nil
}
