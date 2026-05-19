package custom

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/P0m32Kun/Anchor/internal/builtin"
	"github.com/P0m32Kun/Anchor/internal/models"
)

const builtinNucleiID = "builtin:rbkd-templates"

// SeedBuiltin upserts the RBKD-templates nuclei custom source row. Templates are
// served from /opt/rbkd-templates via Worker symlink; this does not clone into
// dataDir or init layout files. Idempotent: refreshes URI metadata on each run;
// preserves Enabled if the user disabled the builtin row.
func (m *Manager) SeedBuiltin() error {
	cfg := builtin.LoadConfig()
	templatesRoot := cfg.TemplatesRoot

	rev := builtin.HeadShort(templatesRoot)
	desc := fmt.Sprintf("RBKD-SEC/RBKD-templates (%s)", templatesRoot)
	if rev != "" {
		desc = fmt.Sprintf("RBKD-SEC/RBKD-templates @ %s", rev)
	}
	log.Printf("[nuclei-custom] seed builtin: %s", desc)

	uri := strings.TrimSuffix(cfg.TemplatesRepo, ".git")
	branch := cfg.TemplatesRef
	now := time.Now().UTC()

	src := &models.NucleiCustomSource{
		ID:            builtinNucleiID,
		Name:          "RBKD Templates",
		InstallPath:   "RBKD-templates",
		Type:          models.NucleiCustomSourceTypeGit,
		URI:           strPtr(uri),
		Branch:        strPtr(branch),
		Enabled:       true,
		Builtin:       true,
		RoutingPolicy: "manual",
		Status:        models.NucleiCustomSourceStatusReady,
		UpdatedAt:     now,
	}
	if rev != "" {
		syncedAt := now
		src.LastSyncAt = &syncedAt
	}

	existing, err := m.q.GetNucleiCustomSource(builtinNucleiID)
	if err != nil {
		return fmt.Errorf("get builtin nuclei source: %w", err)
	}
	if existing != nil {
		src.CreatedAt = existing.CreatedAt
		src.Enabled = existing.Enabled
		if err := m.q.UpdateNucleiCustomSource(src); err != nil {
			return fmt.Errorf("update builtin nuclei source: %w", err)
		}
		return nil
	}

	src.CreatedAt = now
	if err := m.q.CreateNucleiCustomSource(src); err != nil {
		return fmt.Errorf("create builtin nuclei source: %w", err)
	}
	return nil
}
