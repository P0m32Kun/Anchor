package dictionary

import (
	"context"
	"fmt"
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// Manager owns the lifecycle of dictionary wordlists.
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

func (m *Manager) Create(name, description string, category models.DictionaryCategory, content []byte) (*models.Dictionary, error) {
	id := util.GenerateID()
	now := time.Now().UTC()

	d := &models.Dictionary{
		ID:          id,
		Name:        name,
		Description: description,
		Category:    category,
		FilePath:    m.layout.FilePath(id),
		LineCount:   countLines(content),
		SizeBytes:   int64(len(content)),
		Enabled:     true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := m.layout.WriteFile(id, content); err != nil {
		return nil, fmt.Errorf("write file: %w", err)
	}
	if err := m.q.CreateDictionary(d); err != nil {
		_ = m.layout.DeleteFile(id)
		return nil, fmt.Errorf("create dictionary: %w", err)
	}
	return d, nil
}

func (m *Manager) List(category string) ([]*models.Dictionary, error) {
	return m.q.ListDictionaries(category)
}

func (m *Manager) ListEnabled(category string) ([]*models.Dictionary, error) {
	return m.q.ListEnabledDictionaries(category)
}

func (m *Manager) Get(id string) (*models.Dictionary, error) {
	return m.q.GetDictionary(id)
}

func (m *Manager) Update(id, name, description string, category models.DictionaryCategory) (*models.Dictionary, error) {
	d, err := m.q.GetDictionary(id)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, fmt.Errorf("dictionary %s not found", id)
	}
	d.Name = name
	d.Description = description
	d.Category = category
	d.UpdatedAt = time.Now().UTC()
	if err := m.q.UpdateDictionary(d); err != nil {
		return nil, err
	}
	return d, nil
}

func (m *Manager) UpdateContent(id string, content []byte) (*models.Dictionary, error) {
	d, err := m.q.GetDictionary(id)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, fmt.Errorf("dictionary %s not found", id)
	}
	if err := m.layout.WriteFile(id, content); err != nil {
		return nil, fmt.Errorf("write file: %w", err)
	}
	d.LineCount = countLines(content)
	d.SizeBytes = int64(len(content))
	d.UpdatedAt = time.Now().UTC()
	if err := m.q.UpdateDictionary(d); err != nil {
		return nil, err
	}
	return d, nil
}

func (m *Manager) ReadContent(id string) ([]byte, error) {
	return m.layout.ReadFile(id)
}

func (m *Manager) UpdateEnabled(id string, enabled bool) (*models.Dictionary, error) {
	d, err := m.q.GetDictionary(id)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, fmt.Errorf("dictionary %s not found", id)
	}
	if !d.Builtin {
		return nil, fmt.Errorf("only builtin dictionaries can be enabled or disabled")
	}
	now := time.Now().UTC()
	if err := m.q.UpdateDictionaryEnabled(id, enabled, now); err != nil {
		return nil, err
	}
	return m.q.GetDictionary(id)
}

func (m *Manager) Delete(ctx context.Context, id string) error {
	_ = m.layout.DeleteFile(id)
	return m.q.DeleteDictionary(id)
}

func countLines(data []byte) int {
	count := 0
	for _, b := range data {
		if b == '\n' {
			count++
		}
	}
	if len(data) > 0 && data[len(data)-1] != '\n' {
		count++
	}
	return count
}
