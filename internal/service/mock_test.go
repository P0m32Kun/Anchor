package service

import (
	"context"
	"sync"

	"github.com/P0m32Kun/Anchor/internal/models"
)

// MockProjectRepository is a mock implementation of ProjectRepository.
type MockProjectRepository struct {
	mu       sync.RWMutex
	projects map[string]*models.Project
	CreateFn func(project *models.Project) error
	GetFn    func(id string) (*models.Project, error)
	ListFn   func() ([]*models.Project, error)
	DeleteFn func(id string) error
}

func NewMockProjectRepository() *MockProjectRepository {
	return &MockProjectRepository{
		projects: make(map[string]*models.Project),
	}
}

func (m *MockProjectRepository) Create(project *models.Project) error {
	if m.CreateFn != nil {
		return m.CreateFn(project)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.projects[project.ID] = project
	return nil
}

func (m *MockProjectRepository) Get(id string) (*models.Project, error) {
	if m.GetFn != nil {
		return m.GetFn(id)
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.projects[id]
	if !ok {
		return nil, nil
	}
	return p, nil
}

func (m *MockProjectRepository) List() ([]*models.Project, error) {
	if m.ListFn != nil {
		return m.ListFn()
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*models.Project
	for _, p := range m.projects {
		result = append(result, p)
	}
	return result, nil
}

func (m *MockProjectRepository) Delete(id string) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(id)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.projects, id)
	return nil
}

func (m *MockProjectRepository) Count() (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.projects), nil
}

func paginateSlice[T any](all []T, limit, offset int) []T {
	if offset > len(all) {
		return []T{}
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	return all[offset:end]
}

func (m *MockProjectRepository) ListPaginated(limit, offset int) ([]*models.Project, error) {
	all, err := m.List()
	if err != nil {
		return nil, err
	}
	return paginateSlice(all, limit, offset), nil
}

// MockTargetRepository is a mock implementation of TargetRepository.
type MockTargetRepository struct {
	mu       sync.RWMutex
	targets  map[string]*models.Target
	CreateFn func(target *models.Target) error
}

func NewMockTargetRepository() *MockTargetRepository {
	return &MockTargetRepository{
		targets: make(map[string]*models.Target),
	}
}

func (m *MockTargetRepository) Create(target *models.Target) error {
	if m.CreateFn != nil {
		return m.CreateFn(target)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.targets[target.ID] = target
	return nil
}

func (m *MockTargetRepository) ListByProject(projectID string) ([]*models.Target, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*models.Target
	for _, t := range m.targets {
		if t.ProjectID == projectID {
			result = append(result, t)
		}
	}
	return result, nil
}

func (m *MockTargetRepository) ExistsByValue(projectID, value string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, t := range m.targets {
		if t.ProjectID == projectID && t.Value == value {
			return true, nil
		}
	}
	return false, nil
}

func (m *MockTargetRepository) BulkCreate(targets []*models.Target) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, t := range targets {
		m.targets[t.ID] = t
	}
	return nil
}

func (m *MockTargetRepository) CountByProject(projectID string) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var count int
	for _, t := range m.targets {
		if t.ProjectID == projectID {
			count++
		}
	}
	return count, nil
}

func (m *MockTargetRepository) ListByProjectPaginated(projectID string, limit, offset int) ([]*models.Target, error) {
	all, err := m.ListByProject(projectID)
	if err != nil {
		return nil, err
	}
	return paginateSlice(all, limit, offset), nil
}

// MockFindingRepository is a mock implementation of FindingRepository.
type MockFindingRepository struct {
	mu       sync.RWMutex
	findings map[string]*models.Finding
	evidence map[string][]*models.Evidence
}

func NewMockFindingRepository() *MockFindingRepository {
	return &MockFindingRepository{
		findings: make(map[string]*models.Finding),
		evidence: make(map[string][]*models.Evidence),
	}
}

func (m *MockFindingRepository) Get(id string) (*models.Finding, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	f, ok := m.findings[id]
	if !ok {
		return nil, nil
	}
	return f, nil
}

func (m *MockFindingRepository) ListByProject(projectID string) ([]*models.Finding, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*models.Finding
	for _, f := range m.findings {
		if f.ProjectID == projectID {
			result = append(result, f)
		}
	}
	return result, nil
}

func (m *MockFindingRepository) ListByStatus(projectID string, status models.FindingStatus) ([]*models.Finding, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*models.Finding
	for _, f := range m.findings {
		if f.ProjectID == projectID && f.Status == status {
			result = append(result, f)
		}
	}
	return result, nil
}

func (m *MockFindingRepository) UpdateStatus(id string, status models.FindingStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	f, ok := m.findings[id]
	if !ok {
		return nil
	}
	f.Status = status
	return nil
}

func (m *MockFindingRepository) CreateEvidence(evidence *models.Evidence) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.evidence[evidence.FindingID] = append(m.evidence[evidence.FindingID], evidence)
	return nil
}

func (m *MockFindingRepository) ListEvidenceByFinding(findingID string) ([]*models.Evidence, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.evidence[findingID], nil
}

func (m *MockFindingRepository) CountByProject(projectID string, status models.FindingStatus) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var count int
	for _, f := range m.findings {
		if f.ProjectID == projectID && (status == "" || f.Status == status) {
			count++
		}
	}
	return count, nil
}

func (m *MockFindingRepository) ListByProjectPaginated(projectID string, limit, offset int) ([]*models.Finding, error) {
	all, err := m.ListByProject(projectID)
	if err != nil {
		return nil, err
	}
	return paginateSlice(all, limit, offset), nil
}

func (m *MockFindingRepository) ListByStatusPaginated(projectID string, status models.FindingStatus, limit, offset int) ([]*models.Finding, error) {
	all, err := m.ListByStatus(projectID, status)
	if err != nil {
		return nil, err
	}
	return paginateSlice(all, limit, offset), nil
}

// MockScopeChecker is a mock implementation of ScopeChecker.
type MockScopeChecker struct {
	CheckFn func(ctx context.Context, projectID string, target *models.Target) (*ScopeDecision, error)
}

func (m *MockScopeChecker) Check(ctx context.Context, projectID string, target *models.Target) (*ScopeDecision, error) {
	if m.CheckFn != nil {
		return m.CheckFn(ctx, projectID, target)
	}
	return &ScopeDecision{Decision: "allow"}, nil
}

// MockAuditLogger is a mock implementation of AuditLogger.
type MockAuditLogger struct {
	mu  sync.RWMutex
	log []*models.AuditLog
}

func NewMockAuditLogger() *MockAuditLogger {
	return &MockAuditLogger{}
}

func (m *MockAuditLogger) Create(log *models.AuditLog) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.log = append(m.log, log)
	return nil
}

func (m *MockAuditLogger) GetAll() []*models.AuditLog {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.log
}
