package service

import (
	"database/sql"
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/models"
)

// --- Adapters: wrap *db.Queries to satisfy Repository interfaces ---

type projectRepoAdapter struct {
	queries *db.Queries
}

func (a *projectRepoAdapter) Create(project *models.Project) error {
	return a.queries.CreateProject(project)
}

func (a *projectRepoAdapter) Get(id string) (*models.Project, error) {
	return a.queries.GetProject(id)
}

func (a *projectRepoAdapter) List() ([]*models.Project, error) {
	return a.queries.ListProjects()
}

func (a *projectRepoAdapter) Count() (int, error) {
	return a.queries.CountProjects()
}

func (a *projectRepoAdapter) ListPaginated(limit, offset int) ([]*models.Project, error) {
	return a.queries.ListProjectsPaginated(limit, offset)
}

func (a *projectRepoAdapter) Delete(id string) error {
	return a.queries.DeleteProject(id)
}

type targetRepoAdapter struct {
	queries *db.Queries
	rawDB   *sql.DB
}

func (a *targetRepoAdapter) Create(target *models.Target) error {
	return a.queries.CreateTarget(target)
}

func (a *targetRepoAdapter) ListByProject(projectID string) ([]*models.Target, error) {
	return a.queries.ListTargetsByProject(projectID)
}

func (a *targetRepoAdapter) CountByProject(projectID string) (int, error) {
	return a.queries.CountTargetsByProject(projectID)
}

func (a *targetRepoAdapter) ListByProjectPaginated(projectID string, limit, offset int) ([]*models.Target, error) {
	return a.queries.ListTargetsByProjectPaginated(projectID, limit, offset)
}

func (a *targetRepoAdapter) ExistsByValue(projectID, value string) (bool, error) {
	return a.queries.TargetExistsByValue(projectID, value)
}

func (a *targetRepoAdapter) BulkCreate(targets []*models.Target) error {
	return db.WithTx(a.rawDB, func(tx *db.Queries) error {
		return tx.BulkCreateTargets(targets)
	})
}

type findingRepoAdapter struct {
	queries *db.Queries
}

func (a *findingRepoAdapter) Get(id string) (*models.Finding, error) {
	return a.queries.GetFinding(id)
}

func (a *findingRepoAdapter) ListByProject(projectID string) ([]*models.Finding, error) {
	return a.queries.ListFindingsByProject(projectID)
}

func (a *findingRepoAdapter) ListByStatus(projectID string, status models.FindingStatus) ([]*models.Finding, error) {
	return a.queries.ListFindingsByStatus(projectID, status)
}

func (a *findingRepoAdapter) CountByProject(projectID string, status models.FindingStatus) (int, error) {
	return a.queries.CountFindingsByProject(projectID, status)
}

func (a *findingRepoAdapter) ListByProjectPaginated(projectID string, limit, offset int) ([]*models.Finding, error) {
	return a.queries.ListFindingsByProjectPaginated(projectID, limit, offset)
}

func (a *findingRepoAdapter) ListByStatusPaginated(projectID string, status models.FindingStatus, limit, offset int) ([]*models.Finding, error) {
	return a.queries.ListFindingsByStatusPaginated(projectID, status, limit, offset)
}

func (a *findingRepoAdapter) UpdateStatus(id string, status models.FindingStatus) error {
	now := time.Now().UTC()
	return a.queries.UpdateFindingStatus(id, status, now)
}

func (a *findingRepoAdapter) CreateEvidence(evidence *models.Evidence) error {
	return a.queries.CreateEvidence(evidence)
}

func (a *findingRepoAdapter) ListEvidenceByFinding(findingID string) ([]*models.Evidence, error) {
	return a.queries.ListEvidenceByFinding(findingID)
}

type auditLogAdapter struct {
	queries *db.Queries
}

func (a *auditLogAdapter) Create(log *models.AuditLog) error {
	return a.queries.CreateAuditLog(log)
}

// --- Convenience constructors that accept *db.Queries directly ---

// NewProjectService creates a new ProjectService backed by *db.Queries.
func NewProjectService(queries *db.Queries) ProjectService {
	return NewProjectServiceWithDeps(
		&projectRepoAdapter{queries: queries},
		&auditLogAdapter{queries: queries},
	)
}

// NewFindingService creates a new FindingService backed by *db.Queries.
func NewFindingService(queries *db.Queries) FindingService {
	return NewFindingServiceWithDeps(
		&findingRepoAdapter{queries: queries},
	)
}
