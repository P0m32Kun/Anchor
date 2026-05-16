package custom

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
)

// BundleManifest is the JSON manifest embedded in every published bundle.
type BundleManifest struct {
	Version   string              `json:"version"`
	Sources   []BundleSourceEntry `json:"sources"`
	CreatedAt time.Time           `json:"created_at"`
}

// BundleSourceEntry describes one source included in the bundle.
type BundleSourceEntry struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	InstallPath string   `json:"install_path"`
	Files       []string `json:"files"`
	Checksum    string   `json:"checksum"` // sha256 of the source's file content
}

// BuildBundle creates an immutable .tar.gz bundle from all enabled sources.
// Returns the content-hash version (sha256:...) and the archive path.
//
// The bundle is built in a temp directory, hashed, then moved to its final
// location under BundlesRoot. The version is deterministic: identical content
// always produces the same version.
func (m *Manager) BuildBundle() (version string, archivePath string, err error) {
	if err := m.layout.EnsureBundlesRoot(); err != nil {
		return "", "", fmt.Errorf("ensure bundles root: %w", err)
	}

	// Collect enabled sources and their files.
	sources, err := m.q.ListNucleiCustomSources()
	if err != nil {
		return "", "", fmt.Errorf("list sources: %w", err)
	}

	var entries []BundleSourceEntry
	for _, src := range sources {
		if !src.Enabled {
			continue
		}
		files, err := m.layout.WalkFiles(src.ID)
		if err != nil {
			return "", "", fmt.Errorf("walk source %s: %w", src.ID, err)
		}
		if len(files) == 0 {
			continue
		}

		filePaths := make([]string, 0, len(files))
		for _, f := range files {
			filePaths = append(filePaths, f.Path)
		}
		sort.Strings(filePaths)

		// Compute source checksum from file contents
		checksum, err := m.computeSourceChecksum(src.ID, filePaths)
		if err != nil {
			return "", "", fmt.Errorf("checksum source %s: %w", src.ID, err)
		}

		entries = append(entries, BundleSourceEntry{
			ID:          src.ID,
			Name:        src.Name,
			InstallPath: src.InstallPath,
			Files:       filePaths,
			Checksum:    checksum,
		})
	}

	if len(entries) == 0 {
		return "", "", fmt.Errorf("no enabled sources with files to bundle")
	}

	// Sort entries by ID for deterministic output
	sort.Slice(entries, func(i, j int) bool { return entries[i].ID < entries[j].ID })

	// Build manifest
	manifest := BundleManifest{
		Sources:   entries,
		CreatedAt: time.Now().UTC(),
	}
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		return "", "", fmt.Errorf("marshal manifest: %w", err)
	}

	// Compute version from manifest content (deterministic)
	hash := sha256.Sum256(manifestJSON)
	version = "sha256:" + hex.EncodeToString(hash[:])

	// Check if this version already exists
	existing, err := m.q.GetNucleiCustomBundle(version)
	if err != nil {
		return "", "", fmt.Errorf("check existing bundle: %w", err)
	}
	if existing != nil {
		return version, existing.ArchivePath, nil
	}

	// Build archive in temp dir
	tmpDir := filepath.Join(m.layout.BundlesRoot(), ".tmp-"+version)
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return "", "", fmt.Errorf("create tmp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	archivePath = m.layout.BundleArchivePath(version)
	if err := m.createArchive(tmpDir, entries, manifestJSON); err != nil {
		return "", "", err
	}

	// Move archive to final location
	if err := os.Rename(filepath.Join(tmpDir, "bundle.tar.gz"), archivePath); err != nil {
		return "", "", fmt.Errorf("move archive: %w", err)
	}

	// Persist bundle record
	manifest.Version = version
	manifestJSON, _ = json.Marshal(manifest)
	bundle := &models.NucleiCustomBundle{
		Version:      version,
		ManifestJSON: string(manifestJSON),
		ArchivePath:  archivePath,
		Status:       "draft",
		CreatedAt:    time.Now().UTC(),
	}
	if err := m.q.CreateNucleiCustomBundle(bundle); err != nil {
		return "", "", fmt.Errorf("save bundle: %w", err)
	}

	return version, archivePath, nil
}

// createArchive builds the .tar.gz file in tmpDir.
// Files are placed under {install_path}/ so that extracting to ~/nuclei-templates/
// creates the per-source subdirectories nuclei natively searches.
func (m *Manager) createArchive(tmpDir string, entries []BundleSourceEntry, manifestJSON []byte) error {
	archiveFile := filepath.Join(tmpDir, "bundle.tar.gz")
	f, err := os.Create(archiveFile)
	if err != nil {
		return fmt.Errorf("create archive: %w", err)
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	// Write manifest.json
	if err := tw.WriteHeader(&tar.Header{
		Name: "manifest.json",
		Mode: 0o644,
		Size: int64(len(manifestJSON)),
	}); err != nil {
		return fmt.Errorf("write manifest header: %w", err)
	}
	if _, err := tw.Write(manifestJSON); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}

	// Write source files under {install_path}/{relative_path}
	for _, entry := range entries {
		if entry.InstallPath == "" {
			continue
		}

		for _, filePath := range entry.Files {
			data, err := m.layout.ReadFile(entry.ID, filePath)
			if err != nil {
				return fmt.Errorf("read %s/%s: %w", entry.ID, filePath, err)
			}

			// Archive path: {install_path}/{filePath}
			// This ensures extraction to ~/nuclei-templates/ creates
			//  ~/nuclei-templates/{install_path}/network/... etc.
			archivePath := entry.InstallPath + "/" + filePath

			if err := tw.WriteHeader(&tar.Header{
				Name: archivePath,
				Mode: 0o644,
				Size: int64(len(data)),
			}); err != nil {
				return fmt.Errorf("write header %s: %w", archivePath, err)
			}
			if _, err := tw.Write(data); err != nil {
				return fmt.Errorf("write %s: %w", archivePath, err)
			}
		}
	}
	return nil
}

// computeSourceChecksum computes a sha256 hash over the sorted file contents
// of a source.
func (m *Manager) computeSourceChecksum(sourceID string, filePaths []string) (string, error) {
	h := sha256.New()
	for _, filePath := range filePaths {
		data, err := m.layout.ReadFile(sourceID, filePath)
		if err != nil {
			return "", err
		}
		h.Write([]byte(filePath))
		h.Write(data)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// ActivateBundle marks a bundle version as active and deactivates any
// previously active bundle.
func (m *Manager) ActivateBundle(version string) error {
	bundle, err := m.q.GetNucleiCustomBundle(version)
	if err != nil {
		return fmt.Errorf("get bundle: %w", err)
	}
	if bundle == nil {
		return fmt.Errorf("bundle %s not found", version)
	}

	// Deactivate current active bundle (if any)
	bundles, err := m.q.ListNucleiCustomBundles()
	if err != nil {
		return fmt.Errorf("list bundles: %w", err)
	}
	now := time.Now().UTC()
	for _, b := range bundles {
		if b.Status == "active" {
			if err := m.q.SetNucleiCustomBundleStatus(b.Version, "archived", &now); err != nil {
				return fmt.Errorf("deactivate bundle %s: %w", b.Version, err)
			}
		}
	}

	// Activate new bundle
	if err := m.q.SetNucleiCustomBundleStatus(version, "active", &now); err != nil {
		return fmt.Errorf("activate bundle: %w", err)
	}

	// Update symlink
	currentLink := m.layout.CurrentSymlink()
	bundleDir := m.layout.BundleDir(version)
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		return fmt.Errorf("create bundle dir: %w", err)
	}

	// Remove existing symlink if present
	if _, err := os.Lstat(currentLink); err == nil {
		if err := os.Remove(currentLink); err != nil {
			return fmt.Errorf("remove old symlink: %w", err)
		}
	}

	// Create relative symlink
	relPath, err := filepath.Rel(filepath.Dir(currentLink), bundleDir)
	if err != nil {
		return fmt.Errorf("compute relative path: %w", err)
	}
	if err := os.Symlink(relPath, currentLink); err != nil {
		return fmt.Errorf("create symlink: %w", err)
	}

	return nil
}

// GetActiveBundle returns the currently active bundle, or nil if none.
func (m *Manager) GetActiveBundle() (*models.NucleiCustomBundle, error) {
	bundles, err := m.q.ListNucleiCustomBundles()
	if err != nil {
		return nil, fmt.Errorf("list bundles: %w", err)
	}
	for _, b := range bundles {
		if b.Status == "active" {
			return b, nil
		}
	}
	return nil, nil
}

// GetBundleManifest returns the manifest for a specific bundle version.
func (m *Manager) GetBundleManifest(version string) (*BundleManifest, error) {
	bundle, err := m.q.GetNucleiCustomBundle(version)
	if err != nil {
		return nil, fmt.Errorf("get bundle: %w", err)
	}
	if bundle == nil {
		return nil, nil
	}
	var manifest BundleManifest
	if err := json.Unmarshal([]byte(bundle.ManifestJSON), &manifest); err != nil {
		return nil, fmt.Errorf("unmarshal manifest: %w", err)
	}
	return &manifest, nil
}

// GetActiveManifest returns the manifest for the active bundle, suitable for
// workers to fetch.
func (m *Manager) GetActiveManifest() (*BundleManifest, error) {
	bundle, err := m.GetActiveBundle()
	if err != nil {
		return nil, err
	}
	if bundle == nil {
		return nil, nil
	}
	return m.GetBundleManifest(bundle.Version)
}
