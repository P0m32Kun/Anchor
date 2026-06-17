package work

import (
	"encoding/json"
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/scanengine/core"
	"github.com/P0m32Kun/Anchor/internal/scanengine/pool"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// PooledBatchInput describes a Tier-1 batch work item to persist.
type PooledBatchInput struct {
	RunID      string
	ProjectID  string
	Action     core.TaskAction
	Stage      string
	InputFile  string
	Members    []pool.Member
	BucketKey  string
	Generation int
}

// Store wraps db.Queries to provide work-item lifecycle operations
// specific to the scan engine.
type Store struct {
	q *db.Queries
}

// NewStore creates a new work Store.
func NewStore(q *db.Queries) *Store {
	return &Store{q: q}
}

// Create inserts a new ScanWorkItem. Returns the created item.
func (s *Store) Create(runID, projectID, assetID string, action core.TaskAction, stage string) (*models.ScanWorkItem, error) {
	w := &models.ScanWorkItem{
		ID:        util.GenerateID(),
		RunID:     runID,
		ProjectID: projectID,
		AssetID:   assetID,
		Action:    string(action),
		Status:    models.WorkStatusPending,
		Stage:     stage,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.q.CreateScanWorkItem(w); err != nil {
		return nil, err
	}
	return w, nil
}

// CreateBatch inserts multiple work items in a single transaction.
func (s *Store) CreateBatch(runID, projectID string, works []core.DerivedWork) ([]*models.ScanWorkItem, error) {
	now := time.Now().UTC()
	var created []*models.ScanWorkItem
	for _, dw := range works {
		w := &models.ScanWorkItem{
			ID:        util.GenerateID(),
			RunID:     runID,
			ProjectID: projectID,
			AssetID:   dw.AssetID,
			Action:    string(dw.Action),
			Status:    models.WorkStatusPending,
			Stage:     dw.Stage,
			CreatedAt: now,
		}
		if err := s.q.CreateScanWorkItem(w); err != nil {
			return nil, err
		}
		created = append(created, w)
	}
	return created, nil
}

// CreatePooledBatch inserts one batch work item covering multiple assets.
func (s *Store) CreatePooledBatch(in PooledBatchInput) (*models.ScanWorkItem, error) {
	workID := util.GenerateID()
	batchMembers := make([]models.WorkBatchMember, 0, len(in.Members))
	for _, m := range in.Members {
		batchMembers = append(batchMembers, models.WorkBatchMember{
			AssetID:   m.AssetID,
			Value:     m.Value,
			BucketKey: m.BucketKey,
		})
	}
	membersJSON, err := json.Marshal(batchMembers)
	if err != nil {
		return nil, err
	}

	w := &models.ScanWorkItem{
		ID:             workID,
		RunID:          in.RunID,
		ProjectID:      in.ProjectID,
		AssetID:        "batch:" + workID,
		Action:         string(in.Action),
		Status:         models.WorkStatusPending,
		Stage:          in.Stage,
		InputFile:      in.InputFile,
		MemberAssetIDs: string(membersJSON),
		BucketKey:      in.BucketKey,
		Generation:     in.Generation,
		BatchMode:      true,
		CreatedAt:      time.Now().UTC(),
	}
	if err := s.q.CreateScanWorkItem(w); err != nil {
		return nil, err
	}
	return w, nil
}

// TryClaim atomically transitions a pending work item to running.
// Returns nil if the item is not in pending state (already claimed).
func (s *Store) TryClaim(id string) (*models.ScanWorkItem, error) {
	return s.q.ClaimScanWorkItem(id)
}

// SetTaskID links a work item to its scan task.
func (s *Store) SetTaskID(workID, taskID string) error {
	return s.q.UpdateScanWorkItemTaskID(workID, taskID)
}

// MarkDone transitions a work item to done.
func (s *Store) MarkDone(id string) error {
	now := time.Now().UTC()
	return s.q.UpdateScanWorkItemStatus(id, models.WorkStatusDone, nil, &now)
}

// MarkFailed transitions a work item to failed with an error message.
func (s *Store) MarkFailed(id, errMsg string) error {
	now := time.Now().UTC()
	return s.q.UpdateScanWorkItemError(id, models.WorkStatusFailed, errMsg, &now)
}

// MarkSkipped transitions a work item to skipped with a reason.
func (s *Store) MarkSkipped(id, reason string) error {
	now := time.Now().UTC()
	return s.q.UpdateScanWorkItemSkip(id, models.WorkStatusSkipped, reason, &now)
}

// AllTerminal returns true if all work items for the run are in a terminal state.
func (s *Store) AllTerminal(runID string) (bool, error) {
	return s.q.AllWorkItemsTerminal(runID)
}

// ListPending returns all pending work items for a run.
func (s *Store) ListPending(runID string) ([]*models.ScanWorkItem, error) {
	return s.q.ListScanWorkItemsByRunAndStatus(runID, models.WorkStatusPending)
}

// ListByAsset returns all work items for a specific asset in a run.
func (s *Store) ListByAsset(runID, assetID string) ([]*models.ScanWorkItem, error) {
	return s.q.ListScanWorkItemsByAsset(runID, assetID)
}

// Exists checks if a work item already exists for (run_id, asset_id, action).
func (s *Store) Exists(runID, assetID, action string) (bool, error) {
	w, err := s.q.GetScanWorkItemByRunAssetAction(runID, assetID, action)
	if err != nil {
		return false, err
	}
	return w != nil, nil
}
