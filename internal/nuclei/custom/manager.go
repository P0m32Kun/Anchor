package custom

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
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
func (m *Manager) CreateFromGit(ctx context.Context, name, installPath, uri, branch, routingPolicy string) (*models.NucleiCustomSource, error) {
	if err := validateName(name); err != nil {
		return nil, err
	}
	if err := validateInstallPath(installPath); err != nil {
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
		InstallPath:   installPath,
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
func (m *Manager) CreateFromUpload(ctx context.Context, name, installPath, routingPolicy, filename string, body io.Reader) (*models.NucleiCustomSource, error) {
	if err := validateName(name); err != nil {
		return nil, err
	}
	if err := validateInstallPath(installPath); err != nil {
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

// validateInstallPath checks that install_path is a simple directory name
// suitable for use under ~/nuclei-templates/. Must be non-empty, max 64 chars,
// contain only alphanumeric, hyphen, underscore, and dot.
func validateInstallPath(p string) error {
	p = strings.TrimSpace(p)
	if p == "" {
		return apperrors.New(apperrors.ErrBadRequest, "install_path is required (use the repo name)")
	}
	if len(p) > 64 {
		return apperrors.New(apperrors.ErrBadRequest, "install_path too long (max 64)")
	}
	if p == "." || p == ".." {
		return apperrors.New(apperrors.ErrBadRequest, "install_path cannot be . or ..")
	}
	for _, r := range p {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			continue
		}
		return apperrors.Newf(apperrors.ErrBadRequest, "install_path %q contains invalid character %q", p, r)
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

// --- Phase 2: Validation & Publishing ---

// ValidateSource validates all templates in a single source. Returns validation
// results including any workflow reference errors.
func (m *Manager) ValidateSource(id string) (*models.NucleiCustomValidationResult, error) {
	if _, err := m.GetByID(id); err != nil {
		return nil, err
	}

	mu := m.lockFor(id)
	mu.Lock()
	defer mu.Unlock()

	result := &models.NucleiCustomValidationResult{
		SourceID: id,
		OK:       true,
	}

	// Collect all available templates for workflow reference validation
	files, err := m.layout.WalkFiles(id)
	if err != nil {
		return nil, fmt.Errorf("walk source files: %w", err)
	}

	availableTemplates := make(map[string]bool)
	for _, f := range files {
		availableTemplates[f.Path] = true
	}

	// Validate each YAML file in nuclei template directories
	for _, f := range files {
		if !isYAMLFile(f.Path) {
			continue
		}
		// Only validate files in template/workflow/fingerprint directories
		if !isNucleiTemplatePath(f.Path) {
			continue
		}
		data, err := m.layout.ReadFile(id, f.Path)
		if err != nil {
			result.OK = false
			result.Errors = append(result.Errors, fmt.Sprintf("read %s: %v", f.Path, err))
			continue
		}
		vr := ValidateYAML(data, availableTemplates)
		if !vr.OK {
			result.OK = false
			for _, e := range vr.Errors {
				result.Errors = append(result.Errors, fmt.Sprintf("%s: %s", f.Path, e))
			}
		}
	}

	// Update validation timestamp
	now := time.Now().UTC()
	src, _ := m.GetByID(id)
	src.LastValidateAt = &now
	if !result.OK {
		errMsg := strings.Join(result.Errors, "; ")
		src.LastError = &errMsg
		// Workflow template references may be unresolved because the user
		// hasn't populated the custom bundle yet — this is expected and
		// should not mark the source as broken. nuclei will skip missing
		// templates at scan time. Only critical errors (syntax, corruption)
		// should block the source.
		if src.Status != models.NucleiCustomSourceStatusReady {
			src.Status = models.NucleiCustomSourceStatusError
		}
	} else {
		src.LastError = nil
		if src.Status == models.NucleiCustomSourceStatusError {
			src.Status = models.NucleiCustomSourceStatusReady
		}
	}
	src.UpdatedAt = now
	_ = m.q.UpdateNucleiCustomSource(src)

	return result, nil
}

// ValidateAll validates all enabled sources and returns per-source results.
func (m *Manager) ValidateAll() ([]*models.NucleiCustomValidationResult, error) {
	sources, err := m.q.ListNucleiCustomSources()
	if err != nil {
		return nil, fmt.Errorf("list sources: %w", err)
	}

	results := make([]*models.NucleiCustomValidationResult, 0, len(sources))
	for _, src := range sources {
		if !src.Enabled {
			continue
		}
		r, err := m.ValidateSource(src.ID)
		if err != nil {
			r = &models.NucleiCustomValidationResult{
				SourceID: src.ID,
				OK:       false,
				Errors:   []string{fmt.Sprintf("validation error: %v", err)},
			}
		}
		results = append(results, r)
	}
	return results, nil
}

// BuildSourceBundle creates a bundle for a single source.
// This is used by workers to sync individual sources to ~/templates-{sourceId}/.
// Returns version (content hash) and archive path.
func (m *Manager) BuildSourceBundle(sourceID string) (version string, archivePath string, err error) {
	_, err = m.GetByID(sourceID)
	if err != nil {
		return "", "", err
	}

	files, err := m.layout.WalkFiles(sourceID)
	if err != nil {
		return "", "", fmt.Errorf("walk source %s: %w", sourceID, err)
	}
	if len(files) == 0 {
		return "", "", fmt.Errorf("source %s has no files", sourceID)
	}

	// Compute version from source content (deterministic)
	h := sha256.New()
	filePaths := make([]string, 0, len(files))
	for _, f := range files {
		filePaths = append(filePaths, f.Path)
		data, err := m.layout.ReadFile(sourceID, f.Path)
		if err != nil {
			return "", "", fmt.Errorf("read file: %w", err)
		}
		h.Write([]byte(f.Path))
		h.Write(data)
	}
	version = "source:" + sourceID + ":" + hex.EncodeToString(h.Sum(nil))

	// Check cache
	cachedPath := filepath.Join(m.layout.BundlesRoot(), version+".tar.gz")
	if info, err := os.Stat(cachedPath); err == nil && info.Mode().IsRegular() {
		return version, cachedPath, nil
	}

	// Build archive
	if err := m.layout.EnsureBundlesRoot(); err != nil {
		return "", "", fmt.Errorf("ensure bundles root: %w", err)
	}

	tmpDir := filepath.Join(m.layout.BundlesRoot(), ".tmp-"+version)
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return "", "", fmt.Errorf("create tmp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	archiveFile := filepath.Join(tmpDir, "bundle.tar.gz")
	if err := m.createSourceArchive(sourceID, filePaths, archiveFile); err != nil {
		return "", "", err
	}

	// Move to final location
	os.RemoveAll(cachedPath)
	if err := os.Rename(archiveFile, cachedPath); err != nil {
		return "", "", fmt.Errorf("move archive: %w", err)
	}

	return version, cachedPath, nil
}

// createSourceArchive builds a .tar.gz archive for a single source.
// Files are placed under {install_path}/ so extracting to ~/nuclei-templates/
// creates the source subdirectory that nuclei natively searches.
func (m *Manager) createSourceArchive(sourceID string, filePaths []string, archivePath string) error {
	src, err := m.q.GetNucleiCustomSource(sourceID)
	if err != nil || src == nil || src.InstallPath == "" {
		return fmt.Errorf("source %s has no install_path", sourceID)
	}

	f, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("create archive: %w", err)
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	for _, filePath := range filePaths {
		data, err := m.layout.ReadFile(sourceID, filePath)
		if err != nil {
			return fmt.Errorf("read %s: %w", filePath, err)
		}

		// Archive path: {install_path}/{filePath}
		archiveEntryPath := src.InstallPath + "/" + filePath
		if err := tw.WriteHeader(&tar.Header{
			Name: archiveEntryPath,
			Mode: 0o644,
			Size: int64(len(data)),
		}); err != nil {
			return fmt.Errorf("write header %s: %w", archiveEntryPath, err)
		}
		if _, err := tw.Write(data); err != nil {
			return fmt.Errorf("write %s: %w", archiveEntryPath, err)
		}
	}
	return nil
}

// Publish builds a bundle from all enabled sources and activates it.
func (m *Manager) Publish() (version string, err error) {
	// First validate all sources
	results, err := m.ValidateAll()
	if err != nil {
		return "", fmt.Errorf("validate: %w", err)
	}
	for _, r := range results {
		if !r.OK {
			return "", fmt.Errorf("validation failed for source %s: %s", r.SourceID, strings.Join(r.Errors, "; "))
		}
	}

	// Build bundle
	version, _, err = m.BuildBundle()
	if err != nil {
		return "", fmt.Errorf("build bundle: %w", err)
	}

	// Activate bundle
	if err := m.ActivateBundle(version); err != nil {
		return "", fmt.Errorf("activate bundle: %w", err)
	}

	return version, nil
}

func isYAMLFile(p string) bool {
	lower := strings.ToLower(p)
	return strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".yml")
}

// isNucleiTemplatePath returns true if the path is in a directory that should
// contain nuclei templates (templates/, workflows/, fingerprints/, http/,
// network/, javascript/). Excludes metadata/, payloads/, scripts/, tests/, docs/.
func isNucleiTemplatePath(p string) bool {
	// Normalize path separators
	p = strings.ReplaceAll(p, "\\", "/")

	// Exclude non-template directories
	excludePrefixes := []string{
		"metadata/",
		"payloads/",
		"scripts/",
		"tests/",
		"docs/",
	}
	for _, prefix := range excludePrefixes {
		if strings.HasPrefix(p, prefix) {
			return false
		}
	}

	// Include nuclei template directories
	includePrefixes := []string{
		"templates/",
		"workflows/",
		"fingerprints/",
		"http/",
		"network/",
		"javascript/",
	}
	for _, prefix := range includePrefixes {
		if strings.HasPrefix(p, prefix) {
			return true
		}
	}

	return false
}
