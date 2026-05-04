package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/P0m32Kun/Anchor/internal/errors"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

type projectService struct {
	repo  ProjectRepository
	audit AuditLogger
}

// NewProjectServiceWithDeps creates a new ProjectService with injected dependencies.
func NewProjectServiceWithDeps(repo ProjectRepository, audit AuditLogger) ProjectService {
	return &projectService{repo: repo, audit: audit}
}

func (s *projectService) Create(ctx context.Context, req CreateProjectRequest) (*models.Project, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, errors.New(errors.ErrBadRequest, "name is required")
	}
	if len(req.Name) > 200 {
		return nil, errors.New(errors.ErrBadRequest, "name too long (max 200)")
	}
	if req.RateLimit < 0 {
		return nil, errors.New(errors.ErrBadRequest, "rate_limit must be >= 0")
	}

	p := &models.Project{
		ID:             util.GenerateID(),
		Name:           req.Name,
		Organization:   req.Organization,
		Purpose:        req.Purpose,
		RateLimit:      req.RateLimit,
		DefaultProfile: "standard",
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}

	if err := s.repo.Create(p); err != nil {
		return nil, errors.Newf(errors.ErrInternal, "create project failed: %v", err)
	}

	if s.audit != nil {
		_ = s.audit.Create(&models.AuditLog{
			ID:           util.GenerateID(),
			ProjectID:    p.ID,
			Actor:        "user",
			Action:       "project.create",
			ResourceType: "project",
			ResourceID:   p.ID,
			Summary:      fmt.Sprintf("Created project %s", p.Name),
			CreatedAt:    time.Now().UTC(),
		})
	}

	return p, nil
}

func (s *projectService) List(ctx context.Context) ([]*models.Project, error) {
	projects, err := s.repo.List()
	if err != nil {
		return nil, errors.Newf(errors.ErrInternal, "list projects failed: %v", err)
	}
	return projects, nil
}

func (s *projectService) Get(ctx context.Context, id string) (*models.Project, error) {
	p, err := s.repo.Get(id)
	if err != nil {
		return nil, errors.Newf(errors.ErrInternal, "get project failed: %v", err)
	}
	if p == nil {
		return nil, errors.New(errors.ErrNotFound, "project not found")
	}
	return p, nil
}

func (s *projectService) Delete(ctx context.Context, id string) error {
	p, err := s.repo.Get(id)
	if err != nil {
		return errors.Newf(errors.ErrInternal, "get project failed: %v", err)
	}
	if p == nil {
		return errors.New(errors.ErrNotFound, "project not found")
	}

	if err := s.repo.Delete(id); err != nil {
		return errors.Newf(errors.ErrInternal, "delete project failed: %v", err)
	}

	if s.audit != nil {
		_ = s.audit.Create(&models.AuditLog{
			ID:           util.GenerateID(),
			ProjectID:    id,
			Actor:        "user",
			Action:       "project.delete",
			ResourceType: "project",
			ResourceID:   id,
			Summary:      fmt.Sprintf("Deleted project %s", p.Name),
			CreatedAt:    time.Now().UTC(),
		})
	}

	return nil
}
