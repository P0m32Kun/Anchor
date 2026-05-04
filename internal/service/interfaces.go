// Package service provides business logic for the Anchor API.
// This file defines interfaces for dependencies to enable mocking in tests.
package service

import (
	"context"

	"github.com/P0m32Kun/Anchor/internal/models"
)

// ProjectRepository defines the interface for project data access.
type ProjectRepository interface {
	Create(project *models.Project) error
	Get(id string) (*models.Project, error)
	List() ([]*models.Project, error)
	Delete(id string) error
}

// TargetRepository defines the interface for target data access.
type TargetRepository interface {
	Create(target *models.Target) error
	ListByProject(projectID string) ([]*models.Target, error)
	ExistsByValue(projectID, value string) (bool, error)
	BulkCreate(targets []*models.Target) error
}

// FindingRepository defines the interface for finding data access.
type FindingRepository interface {
	Get(id string) (*models.Finding, error)
	ListByProject(projectID string) ([]*models.Finding, error)
	ListByStatus(projectID string, status models.FindingStatus) ([]*models.Finding, error)
	UpdateStatus(id string, status models.FindingStatus) error
	CreateEvidence(evidence *models.Evidence) error
	ListEvidenceByFinding(findingID string) ([]*models.Evidence, error)
}

// ScopeChecker defines the interface for scope checking.
type ScopeChecker interface {
	Check(ctx context.Context, projectID string, target *models.Target) (*ScopeDecision, error)
}

// ScopeDecision represents the result of a scope check.
type ScopeDecision struct {
	Decision string
	Reason   string
}

// AuditLogger defines the interface for audit logging.
type AuditLogger interface {
	Create(log *models.AuditLog) error
}
