package custom

import (
	"fmt"
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
	apperrors "github.com/P0m32Kun/Anchor/internal/errors"
	"github.com/P0m32Kun/Anchor/internal/models"
)

// Manager owns the RBKD builtin nuclei template source row in DB.
// Templates are served from ANCHOR_BUILTIN_TEMPLATES_ROOT (/opt/rbkd-templates)
// via Worker symlink — no per-source git clone or bundle publish.
type Manager struct {
	q *db.Queries
}

// NewManager constructs a Manager.
func NewManager(q *db.Queries) *Manager {
	return &Manager{q: q}
}

// List returns every known source (newest first).
func (m *Manager) List() ([]*models.NucleiCustomSource, error) {
	return m.q.ListNucleiCustomSources()
}

// GetByID returns the source row, or *AppError(NotFound) if missing.
func (m *Manager) GetByID(id string) (*models.NucleiCustomSource, error) {
	src, err := m.q.GetNucleiCustomSource(id)
	if err != nil {
		return nil, fmt.Errorf("get source: %w", err)
	}
	if src == nil {
		return nil, apperrors.Newf(apperrors.ErrNotFound, "nuclei custom source %q not found", id)
	}
	return src, nil
}

// UpdateEnabled toggles enabled for a builtin source row.
func (m *Manager) UpdateEnabled(id string, enabled bool) (*models.NucleiCustomSource, error) {
	src, err := m.GetByID(id)
	if err != nil {
		return nil, err
	}
	if !src.Builtin {
		return nil, ErrNotBuiltin
	}
	now := time.Now().UTC()
	if err := m.q.UpdateNucleiCustomSourceEnabled(id, enabled, now); err != nil {
		return nil, fmt.Errorf("update enabled: %w", err)
	}
	return m.GetByID(id)
}
