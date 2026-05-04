package service

import (
	"context"
	"time"

	"github.com/P0m32Kun/Anchor/internal/errors"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

type findingService struct {
	repo FindingRepository
}

// NewFindingServiceWithDeps creates a new FindingService with injected dependencies.
func NewFindingServiceWithDeps(repo FindingRepository) FindingService {
	return &findingService{repo: repo}
}

func (s *findingService) List(ctx context.Context, projectID string, status string) ([]*models.Finding, error) {
	var findings []*models.Finding
	var err error
	if status != "" {
		findings, err = s.repo.ListByStatus(projectID, models.FindingStatus(status))
	} else {
		findings, err = s.repo.ListByProject(projectID)
	}
	if err != nil {
		return nil, errors.Newf(errors.ErrInternal, "list findings failed: %v", err)
	}
	return findings, nil
}

func (s *findingService) Get(ctx context.Context, id string) (*models.Finding, error) {
	finding, err := s.repo.Get(id)
	if err != nil {
		return nil, errors.Newf(errors.ErrInternal, "get finding failed: %v", err)
	}
	if finding == nil {
		return nil, errors.New(errors.ErrNotFound, "finding not found")
	}
	return finding, nil
}

func (s *findingService) UpdateStatus(ctx context.Context, id string, status string) error {
	validStatuses := map[string]bool{
		"confirmed":      true,
		"false_positive": true,
		"accepted_risk":  true,
		"ignored":        true,
		"pending_review": true,
	}
	if !validStatuses[status] {
		return errors.New(errors.ErrBadRequest, "invalid status")
	}

	finding, err := s.repo.Get(id)
	if err != nil {
		return errors.Newf(errors.ErrInternal, "get finding failed: %v", err)
	}
	if finding == nil {
		return errors.New(errors.ErrNotFound, "finding not found")
	}

	if err := s.repo.UpdateStatus(id, models.FindingStatus(status)); err != nil {
		return errors.Newf(errors.ErrInternal, "update finding status failed: %v", err)
	}
	return nil
}

func (s *findingService) AddEvidence(ctx context.Context, findingID string, req AddEvidenceRequest) (*models.Evidence, error) {
	if req.Type == "" || req.Excerpt == "" {
		return nil, errors.New(errors.ErrBadRequest, "type and excerpt are required")
	}
	validTypes := map[string]bool{
		"note":       true,
		"screenshot": true,
		"file":       true,
	}
	if !validTypes[req.Type] {
		return nil, errors.New(errors.ErrBadRequest, "invalid evidence type")
	}

	ev := &models.Evidence{
		ID:        util.GenerateID(),
		FindingID: findingID,
		Type:      models.EvidenceType(req.Type),
		Excerpt:   req.Excerpt,
		CreatedBy: req.CreatedBy,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.repo.CreateEvidence(ev); err != nil {
		return nil, errors.Newf(errors.ErrInternal, "create evidence failed: %v", err)
	}
	return ev, nil
}
