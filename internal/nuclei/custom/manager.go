package custom

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
	apperrors "github.com/P0m32Kun/Anchor/internal/errors"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/safefs"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// MaxWriteFileBytes is the per-file size limit for the file CRUD API. Acts as
// the inner half of the double guard; the outer guard is http.MaxBytesReader
// in the HTTP handler.
const MaxWriteFileBytes = 1 << 20 // 1 MiB

// Routing policies recognised at the Manager layer. The handler also accepts
// these strings verbatim — Manager is the canonical authority.
var allowedRoutingPolicies = map[string]struct{}{
	"manual": {},
	"auto":   {},
}

// allowedSourceTypes mirrors the DB CHECK constraint.
var allowedSourceTypes = map[models.NucleiCustomSourceType]struct{}{
	models.NucleiCustomSourceTypeGit:    {},
	models.NucleiCustomSourceTypeUpload: {},
	models.NucleiCustomSourceTypeFile:   {},
}

// Manager owns the lifecycle of Nuclei custom template sources.
//
// It serialises operations on a single source via per-source mutexes so that
// clone/refresh/file-CRUD never race. Different sources operate in parallel.
type Manager struct {
	q      *db.Queries
	rawDB  *sql.DB
	layout Layout
	cloner Cloner
	locks  sync.Map // sourceID -> *sync.Mutex
}

// NewManager constructs a Manager. Pass a real ExecCloner in production; tests
// substitute a fake.
func NewManager(q *db.Queries, rawDB *sql.DB, dataDir string, cloner Cloner) *Manager {
	if cloner == nil {
		cloner = ExecCloner{}
	}
	return &Manager{
		q:      q,
		rawDB:  rawDB,
		layout: NewLayout(dataDir),
		cloner: cloner,
	}
}

// EnsureLayout creates the on-disk root if missing.
func (m *Manager) EnsureLayout() error { return m.layout.EnsureRoot() }

// Layout exposes the on-disk layout for tests.
func (m *Manager) Layout() Layout { return m.layout }

func (m *Manager) lockFor(id string) *sync.Mutex {
	v, _ := m.locks.LoadOrStore(id, &sync.Mutex{})
	return v.(*sync.Mutex)
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

// CreateFromGit clones a public HTTPS git repo into the source's files/ dir.
// On success the row is moved to status=ready; on failure the DB row and any
// partial files are removed before returning.
func (m *Manager) CreateFromGit(ctx context.Context, name, uri, branch, routingPolicy string) (*models.NucleiCustomSource, error) {
	if err := validateName(name); err != nil {
		return nil, err
	}
	if err := validateRoutingPolicy(routingPolicy); err != nil {
		return nil, err
	}
	if err := validateGitURL(uri); err != nil {
		return nil, apperrors.New(apperrors.ErrBadRequest, err.Error())
	}

	now := time.Now().UTC()
	src := &models.NucleiCustomSource{
		ID:            util.GenerateID(),
		Name:          name,
		Type:          models.NucleiCustomSourceTypeGit,
		URI:           strPtr(uri),
		Branch:        nullableStr(branch),
		Enabled:       true,
		RoutingPolicy: routingPolicy,
		Status:        models.NucleiCustomSourceStatusDraft,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	mu := m.lockFor(src.ID)
	mu.Lock()
	defer mu.Unlock()

	if err := m.q.CreateNucleiCustomSource(src); err != nil {
		return nil, fmt.Errorf("insert source: %w", err)
	}

	rollback := func(reason string, cause error) (*models.NucleiCustomSource, error) {
		_ = m.q.DeleteNucleiCustomSource(src.ID)
		_ = m.layout.RemoveSource(src.ID)
		return nil, fmt.Errorf("%s: %w", reason, cause)
	}

	if err := m.layout.InitSource(src.ID); err != nil {
		return rollback("init source dir", err)
	}

	// Clone into a sibling tmp, then atomically swap into place.
	tmp := filepath.Join(m.layout.SourceDir(src.ID), fmt.Sprintf("clone-%d", time.Now().UnixNano()))
	if err := m.cloner.Clone(ctx, uri, branch, tmp); err != nil {
		_ = os.RemoveAll(tmp)
		return rollback("clone", err)
	}
	old, err := m.layout.SwapFilesDir(src.ID, tmp)
	if err != nil {
		_ = os.RemoveAll(tmp)
		return rollback("install clone", err)
	}
	if old != "" {
		go func(p string) { _ = os.RemoveAll(p) }(old)
	}

	syncedAt := time.Now().UTC()
	src.Status = models.NucleiCustomSourceStatusReady
	src.LastSyncAt = &syncedAt
	src.UpdatedAt = syncedAt
	if err := m.q.UpdateNucleiCustomSource(src); err != nil {
		return rollback("set ready", err)
	}
	return src, nil
}

// CreateFromUpload accepts a single yaml/yml file or a zip archive, extracts
// it under the source's templates/ subtree, and marks the source ready.
func (m *Manager) CreateFromUpload(ctx context.Context, name, routingPolicy, filename string, body io.Reader) (*models.NucleiCustomSource, error) {
	if err := validateName(name); err != nil {
		return nil, err
	}
	if err := validateRoutingPolicy(routingPolicy); err != nil {
		return nil, err
	}
	if filename == "" {
		return nil, apperrors.New(apperrors.ErrBadRequest, "upload filename is required")
	}

	now := time.Now().UTC()
	src := &models.NucleiCustomSource{
		ID:            util.GenerateID(),
		Name:          name,
		Type:          models.NucleiCustomSourceTypeUpload,
		Enabled:       true,
		RoutingPolicy: routingPolicy,
		Status:        models.NucleiCustomSourceStatusDraft,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	mu := m.lockFor(src.ID)
	mu.Lock()
	defer mu.Unlock()

	if err := m.q.CreateNucleiCustomSource(src); err != nil {
		return nil, fmt.Errorf("insert source: %w", err)
	}

	rollback := func(reason string, cause error) (*models.NucleiCustomSource, error) {
		_ = m.q.DeleteNucleiCustomSource(src.ID)
		_ = m.layout.RemoveSource(src.ID)
		return nil, fmt.Errorf("%s: %w", reason, cause)
	}

	if err := m.layout.InitSource(src.ID); err != nil {
		return rollback("init source dir", err)
	}
	if err := ExtractUpload(m.layout, src.ID, filename, body); err != nil {
		return rollback("extract upload", err)
	}

	syncedAt := time.Now().UTC()
	src.Status = models.NucleiCustomSourceStatusReady
	src.LastSyncAt = &syncedAt
	src.UpdatedAt = syncedAt
	if err := m.q.UpdateNucleiCustomSource(src); err != nil {
		return rollback("set ready", err)
	}
	return src, nil
}

// Refresh re-clones a git source's tree into a sibling directory and atomically
// replaces the existing files/ with the new tree. Only valid for git sources.
func (m *Manager) Refresh(ctx context.Context, id string) (*models.NucleiCustomSource, error) {
	mu := m.lockFor(id)
	mu.Lock()
	defer mu.Unlock()

	src, err := m.GetByID(id)
	if err != nil {
		return nil, err
	}
	if src.Type != models.NucleiCustomSourceTypeGit {
		return nil, apperrors.New(apperrors.ErrBadRequest, "refresh is only valid for git sources")
	}
	if src.URI == nil || *src.URI == "" {
		return nil, apperrors.New(apperrors.ErrBadRequest, "git source has no uri")
	}
	branch := ""
	if src.Branch != nil {
		branch = *src.Branch
	}

	tmp := filepath.Join(m.layout.SourceDir(id), fmt.Sprintf("refresh-%d", time.Now().UnixNano()))
	if err := m.cloner.Clone(ctx, *src.URI, branch, tmp); err != nil {
		_ = os.RemoveAll(tmp)
		return nil, fmt.Errorf("refresh clone: %w", err)
	}
	old, err := m.layout.SwapFilesDir(id, tmp)
	if err != nil {
		_ = os.RemoveAll(tmp)
		return nil, fmt.Errorf("refresh install: %w", err)
	}
	if old != "" {
		go func(p string) { _ = os.RemoveAll(p) }(old)
	}

	now := time.Now().UTC()
	src.Status = models.NucleiCustomSourceStatusReady
	src.LastSyncAt = &now
	src.LastError = nil
	src.UpdatedAt = now
	if err := m.q.UpdateNucleiCustomSource(src); err != nil {
		return nil, fmt.Errorf("update after refresh: %w", err)
	}
	return src, nil
}

// SourcePatch carries the mutable fields a PATCH may modify. nil means leave
// the field unchanged. type/uri/branch are intentionally omitted: they are
// immutable and require delete+recreate.
type SourcePatch struct {
	Name          *string `json:"name,omitempty"`
	Enabled       *bool   `json:"enabled,omitempty"`
	RoutingPolicy *string `json:"routing_policy,omitempty"`
}

// Patch applies the allowed mutable fields to an existing source row.
func (m *Manager) Patch(id string, p SourcePatch) (*models.NucleiCustomSource, error) {
	mu := m.lockFor(id)
	mu.Lock()
	defer mu.Unlock()

	src, err := m.GetByID(id)
	if err != nil {
		return nil, err
	}
	if p.Name != nil {
		if err := validateName(*p.Name); err != nil {
			return nil, err
		}
		src.Name = *p.Name
	}
	if p.Enabled != nil {
		src.Enabled = *p.Enabled
	}
	if p.RoutingPolicy != nil {
		if err := validateRoutingPolicy(*p.RoutingPolicy); err != nil {
			return nil, err
		}
		src.RoutingPolicy = *p.RoutingPolicy
	}
	src.UpdatedAt = time.Now().UTC()

	if err := m.q.UpdateNucleiCustomSource(src); err != nil {
		return nil, fmt.Errorf("update source: %w", err)
	}
	return src, nil
}

// Delete removes the source row in a single transaction; once committed it
// best-effort removes the on-disk tree. FS residue is logged but does not
// fail the API call — DB is the source of truth.
func (m *Manager) Delete(ctx context.Context, id string) error {
	mu := m.lockFor(id)
	mu.Lock()
	defer mu.Unlock()

	if _, err := m.GetByID(id); err != nil {
		return err
	}

	tx, err := m.rawDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM nuclei_custom_sources WHERE id = ?`, id); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("delete row: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit delete: %w", err)
	}

	if err := m.layout.RemoveSource(id); err != nil {
		log.Printf("[nuclei-custom] best-effort remove %s: %v", id, err)
	}
	return nil
}

// ListFiles returns all regular files under the source's files/ tree.
func (m *Manager) ListFiles(id string) ([]models.NucleiCustomFileEntry, error) {
	if _, err := m.GetByID(id); err != nil {
		return nil, err
	}
	mu := m.lockFor(id)
	mu.Lock()
	defer mu.Unlock()
	return m.layout.WalkFiles(id)
}

// ReadFile returns the contents of a single file, gated by the extension
// policy. Returns *AppError on validation failures and 404 when missing.
func (m *Manager) ReadFile(id, rel string) ([]byte, error) {
	if _, err := m.GetByID(id); err != nil {
		return nil, err
	}
	if err := validateAllowed(rel); err != nil {
		return nil, err
	}
	mu := m.lockFor(id)
	mu.Lock()
	defer mu.Unlock()
	data, err := m.layout.ReadFile(id, rel)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, apperrors.Newf(apperrors.ErrNotFound, "file %q not found", rel)
		}
		return nil, fmt.Errorf("read file: %w", err)
	}
	return data, nil
}

// WriteFile writes data atomically into the source. Caller is responsible for
// any HTTP-level size cap; this method also enforces MaxWriteFileBytes as a
// defence-in-depth check.
func (m *Manager) WriteFile(id, rel string, data []byte) error {
	if _, err := m.GetByID(id); err != nil {
		return err
	}
	if err := validateAllowed(rel); err != nil {
		return err
	}
	if int64(len(data)) > MaxWriteFileBytes {
		return apperrors.Newf(apperrors.ErrValidation, "file exceeds %d bytes", MaxWriteFileBytes)
	}
	mu := m.lockFor(id)
	mu.Lock()
	defer mu.Unlock()
	if err := m.layout.WriteFileAtomic(id, rel, data); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return nil
}

// DeleteFile removes a single file inside the source.
func (m *Manager) DeleteFile(id, rel string) error {
	if _, err := m.GetByID(id); err != nil {
		return err
	}
	if err := validateAllowed(rel); err != nil {
		return err
	}
	mu := m.lockFor(id)
	mu.Lock()
	defer mu.Unlock()
	if err := m.layout.DeleteFile(id, rel); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return apperrors.Newf(apperrors.ErrNotFound, "file %q not found", rel)
		}
		return fmt.Errorf("delete file: %w", err)
	}
	return nil
}

func validateAllowed(rel string) error {
	if err := safefs.ValidateRelPath(rel); err != nil {
		return apperrors.New(apperrors.ErrValidation, err.Error())
	}
	if !safefs.IsAllowedTemplateFile(rel) {
		return apperrors.Newf(apperrors.ErrValidation, "extension not allowed for %q", rel)
	}
	return nil
}

func validateName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return apperrors.New(apperrors.ErrBadRequest, "name is required")
	}
	if len(name) > 128 {
		return apperrors.New(apperrors.ErrBadRequest, "name too long (max 128)")
	}
	return nil
}

func validateRoutingPolicy(p string) error {
	if _, ok := allowedRoutingPolicies[p]; !ok {
		return apperrors.Newf(apperrors.ErrBadRequest, "routing_policy %q is not supported", p)
	}
	return nil
}

func validateSourceType(t models.NucleiCustomSourceType) error {
	if _, ok := allowedSourceTypes[t]; !ok {
		return apperrors.Newf(apperrors.ErrBadRequest, "source type %q is not supported", t)
	}
	return nil
}

func strPtr(s string) *string { return &s }

func nullableStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
