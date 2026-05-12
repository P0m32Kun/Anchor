package httpxfp

import (
	"context"
	"fmt"
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// Manager owns the lifecycle of httpx custom fingerprint files.
type Manager struct {
	q      *db.Queries
	layout Layout
}

func NewManager(q *db.Queries, dataDir string) *Manager {
	return &Manager{
		q:      q,
		layout: NewLayout(dataDir),
	}
}

func (m *Manager) EnsureLayout() error { return m.layout.EnsureRoot() }

func (m *Manager) Create(name, description string, fpType models.HttpxFingerprintType, content []byte) (*models.HttpxFingerprint, error) {
	id := util.GenerateID()
	now := time.Now().UTC()

	f := &models.HttpxFingerprint{
		ID:          id,
		Name:        name,
		Description: description,
		Type:        fpType,
		FilePath:    m.layout.FilePath(id, string(fpType)),
		Enabled:     true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := m.layout.WriteFile(id, string(fpType), content); err != nil {
		return nil, fmt.Errorf("write file: %w", err)
	}
	if err := m.q.CreateHttpxFingerprint(f); err != nil {
		_ = m.layout.DeleteFile(id, string(fpType))
		return nil, fmt.Errorf("create fingerprint: %w", err)
	}
	return f, nil
}

func (m *Manager) List(fpType string) ([]*models.HttpxFingerprint, error) {
	return m.q.ListHttpxFingerprints(fpType)
}

func (m *Manager) ListEnabled(fpType string) ([]*models.HttpxFingerprint, error) {
	return m.q.ListEnabledHttpxFingerprints(fpType)
}

func (m *Manager) Get(id string) (*models.HttpxFingerprint, error) {
	return m.q.GetHttpxFingerprint(id)
}

func (m *Manager) Update(id, name, description string, enabled bool) (*models.HttpxFingerprint, error) {
	f, err := m.q.GetHttpxFingerprint(id)
	if err != nil {
		return nil, err
	}
	if f == nil {
		return nil, fmt.Errorf("fingerprint %s not found", id)
	}
	f.Name = name
	f.Description = description
	f.Enabled = enabled
	f.UpdatedAt = time.Now().UTC()
	if err := m.q.UpdateHttpxFingerprint(f); err != nil {
		return nil, err
	}
	return f, nil
}

func (m *Manager) UpdateContent(id string, content []byte) (*models.HttpxFingerprint, error) {
	f, err := m.q.GetHttpxFingerprint(id)
	if err != nil {
		return nil, err
	}
	if f == nil {
		return nil, fmt.Errorf("fingerprint %s not found", id)
	}
	if err := m.layout.WriteFile(id, string(f.Type), content); err != nil {
		return nil, fmt.Errorf("write file: %w", err)
	}
	f.UpdatedAt = time.Now().UTC()
	if err := m.q.UpdateHttpxFingerprint(f); err != nil {
		return nil, err
	}
	return f, nil
}

func (m *Manager) ReadContent(id string) ([]byte, error) {
	f, err := m.q.GetHttpxFingerprint(id)
	if err != nil {
		return nil, err
	}
	if f == nil {
		return nil, fmt.Errorf("fingerprint %s not found", id)
	}
	return m.layout.ReadFile(id, string(f.Type))
}

func (m *Manager) Delete(ctx context.Context, id string) error {
	f, err := m.q.GetHttpxFingerprint(id)
	if err != nil {
		return err
	}
	if f != nil {
		_ = m.layout.DeleteFile(id, string(f.Type))
	}
	return m.q.DeleteHttpxFingerprint(id)
}
