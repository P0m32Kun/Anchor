// Package service provides business logic for the Anchor API.
// It decouples HTTP handlers from database operations and domain logic.
package service

import (
	"context"

	"github.com/P0m32Kun/Anchor/internal/models"
)

// PaginationParams is used across service methods.
type PaginationParams struct {
	Page     int
	PageSize int
}

func (p PaginationParams) Offset() int {
	return (p.Page - 1) * p.PageSize
}

// PaginatedList is the generic response for paginated queries.
type PaginatedList[T any] struct {
	Data     []T `json:"data"`
	Total    int `json:"total"`
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

// ProjectService handles project-related business logic.
type ProjectService interface {
	Create(ctx context.Context, req CreateProjectRequest) (*models.Project, error)
	List(ctx context.Context) ([]*models.Project, error)
	ListPaginated(ctx context.Context, p PaginationParams) (*PaginatedList[*models.Project], error)
	Get(ctx context.Context, id string) (*models.Project, error)
	Delete(ctx context.Context, id string) error
}

// TargetService handles target-related business logic.
type TargetService interface {
	Create(ctx context.Context, projectID string, req CreateTargetRequest) (*TargetResponse, error)
	List(ctx context.Context, projectID string) ([]*models.Target, error)
	ListPaginated(ctx context.Context, projectID string, p PaginationParams) (*PaginatedList[*models.Target], error)
	Import(ctx context.Context, projectID string, targets []ImportTarget) (*ImportResult, error)
}

// FindingService handles finding-related business logic.
type FindingService interface {
	List(ctx context.Context, projectID string, status string) ([]*models.Finding, error)
	ListPaginated(ctx context.Context, projectID string, status string, p PaginationParams) (*PaginatedList[*models.Finding], error)
	Get(ctx context.Context, id string) (*models.Finding, error)
	UpdateStatus(ctx context.Context, id string, status string) error
	AddEvidence(ctx context.Context, findingID string, req AddEvidenceRequest) (*models.Evidence, error)
	ListEvidence(ctx context.Context, findingID string) ([]*models.Evidence, error)
}

// Request/Response types

type CreateProjectRequest struct {
	Name         string `json:"name"`
	Organization string `json:"organization"`
	Purpose      string `json:"purpose"`
	RateLimit    int    `json:"rate_limit"`
}

type CreateTargetRequest struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type TargetResponse struct {
	Target                 *models.Target
	NeedsScopeConfirmation bool
	Message                string
	SuggestedRule          *ScopeRuleSuggestion
}

type ScopeRuleSuggestion struct {
	Action string `json:"action"`
	Type   string `json:"type"`
	Value  string `json:"value"`
}

type ImportTarget struct {
	Type  models.TargetType
	Value string
}

type ImportResult struct {
	Imported               int                   `json:"imported"`
	Duplicates             int                   `json:"duplicates"`
	Denied                 int                   `json:"denied"`
	Errors                 int                   `json:"errors"`
	Targets                []*models.Target      `json:"targets,omitempty"`
	DeniedTargets          []DeniedTarget        `json:"denied_targets,omitempty"`
	NeedsScopeConfirmation bool                  `json:"needs_scope_confirmation,omitempty"`
	Message                string                `json:"message,omitempty"`
	SuggestedRules         []ScopeRuleSuggestion `json:"suggested_rules,omitempty"`
}

type DeniedTarget struct {
	Value  string `json:"value"`
	Reason string `json:"reason"`
}

type AddEvidenceRequest struct {
	Type      string `json:"type"`
	Excerpt   string `json:"excerpt"`
	CreatedBy string `json:"created_by"`
}
