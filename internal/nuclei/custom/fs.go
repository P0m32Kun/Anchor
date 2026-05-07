// Package custom owns the on-disk layout and lifecycle of user-managed Nuclei
// template sources. The Layout type centralises every path computation and
// mutation so callers never construct paths directly.
package custom

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/safefs"
)

// Layout describes the on-disk directory tree rooted at {dataDir}/nuclei/custom.
//
//	{Root}/
//	  sources/
//	    {sourceID}/
//	      source.json        ← metadata mirror (Phase 2 may use)
//	      files/
//	        templates/
//	        workflows/
//	        payloads/
//	  bundles/                ← Phase 2 only; not created in Phase 1
type Layout struct {
	Root string
}

// NewLayout returns a Layout rooted at filepath.Join(dataDir, "nuclei", "custom").
func NewLayout(dataDir string) Layout {
	return Layout{Root: filepath.Join(dataDir, "nuclei", "custom")}
}

// SourcesRoot returns {Root}/sources.
func (l Layout) SourcesRoot() string { return filepath.Join(l.Root, "sources") }

// SourceDir returns {Root}/sources/{id}.
func (l Layout) SourceDir(id string) string { return filepath.Join(l.SourcesRoot(), id) }

// FilesDir returns {Root}/sources/{id}/files — the writable surface for
// per-source template content.
func (l Layout) FilesDir(id string) string { return filepath.Join(l.SourceDir(id), "files") }

// SourceJSONPath returns the metadata mirror path for a source. Phase 1 does
// not write this file; the path is exposed for Phase 2 use.
func (l Layout) SourceJSONPath(id string) string { return filepath.Join(l.SourceDir(id), "source.json") }

// EnsureRoot creates {Root}/sources if missing. Phase 1 does not create
// bundles/ — Phase 2 will own that.
func (l Layout) EnsureRoot() error {
	if err := os.MkdirAll(l.SourcesRoot(), 0o755); err != nil {
		return fmt.Errorf("ensure sources root: %w", err)
	}
	return nil
}

// InitSource creates the source's standard subtree
// (files/templates, files/workflows, files/payloads). Idempotent.
func (l Layout) InitSource(id string) error {
	if id == "" {
		return errors.New("custom: source id is empty")
	}
	for _, sub := range []string{"templates", "workflows", "payloads"} {
		dir := filepath.Join(l.FilesDir(id), sub)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("init source %s: %w", sub, err)
		}
	}
	return nil
}

// RemoveSource deletes the entire {SourceDir(id)} subtree. Caller is
// responsible for serialising against in-flight writes.
func (l Layout) RemoveSource(id string) error {
	if id == "" {
		return errors.New("custom: source id is empty")
	}
	return os.RemoveAll(l.SourceDir(id))
}

// resolve validates rel against the source's files dir and returns the
// absolute target path. Symlink escape is rejected on the closest existing
// ancestor; the leaf may not exist yet (creation case).
func (l Layout) resolve(id, rel string) (string, error) {
	if id == "" {
		return "", errors.New("custom: source id is empty")
	}
	filesDir := l.FilesDir(id)
	abs, err := safefs.JoinUnderRoot(filesDir, rel)
	if err != nil {
		return "", err
	}
	if err := safefs.EnsureNoSymlinkEscape(filesDir, abs); err != nil {
		return "", err
	}
	return abs, nil
}

// WriteFileAtomic writes data to {filesDir}/{rel} via a sibling .tmp file
// followed by os.Rename. Parent directories are created as needed.
//
// The caller must enforce its own size limit on data before calling; Layout
// only owns path safety and atomicity.
func (l Layout) WriteFileAtomic(id, rel string, data []byte) error {
	abs, err := l.resolve(id, rel)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return fmt.Errorf("mkdir parent: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(abs), filepath.Base(abs)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create tmp: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpPath) }

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("write tmp: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("sync tmp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close tmp: %w", err)
	}
	if err := os.Rename(tmpPath, abs); err != nil {
		cleanup()
		return fmt.Errorf("rename tmp: %w", err)
	}
	return nil
}

// ReadFile returns the contents of {filesDir}/{rel}. Returns os.ErrNotExist
// (wrapped) when missing.
func (l Layout) ReadFile(id, rel string) ([]byte, error) {
	abs, err := l.resolve(id, rel)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// Stat returns the FileInfo for {filesDir}/{rel}. Used for HEAD-style checks
// and for distinguishing 404 from validation errors at the API layer.
func (l Layout) Stat(id, rel string) (os.FileInfo, error) {
	abs, err := l.resolve(id, rel)
	if err != nil {
		return nil, err
	}
	return os.Stat(abs)
}

// DeleteFile removes {filesDir}/{rel}. Refuses to remove the files/ root
// itself or directories — this API is file-only.
func (l Layout) DeleteFile(id, rel string) error {
	abs, err := l.resolve(id, rel)
	if err != nil {
		return err
	}
	if abs == l.FilesDir(id) {
		return errors.New("custom: refuse to delete files/ root")
	}
	info, err := os.Lstat(abs)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return errors.New("custom: refuse to delete directory via file API")
	}
	return os.Remove(abs)
}

// WalkFiles enumerates regular files under {filesDir}, returning entries with
// POSIX-style relative paths. Hidden directories prefixed with "." are
// skipped. Results are sorted by path for deterministic output.
func (l Layout) WalkFiles(id string) ([]models.NucleiCustomFileEntry, error) {
	root := l.FilesDir(id)
	entries := make([]models.NucleiCustomFileEntry, 0)

	if _, err := os.Stat(root); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return entries, nil
		}
		return nil, err
	}

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		if d.IsDir() {
			// Skip hidden directories like .git that an ingest may produce.
			if d.Name() != "." && len(d.Name()) > 0 && d.Name()[0] == '.' {
				return fs.SkipDir
			}
			return nil
		}
		info, infoErr := d.Info()
		if infoErr != nil {
			return infoErr
		}
		entries = append(entries, models.NucleiCustomFileEntry{
			Path:    filepath.ToSlash(rel),
			IsDir:   false,
			Size:    info.Size(),
			ModTime: info.ModTime().UTC(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	return entries, nil
}

// --- Bundle directory methods (Phase 2) ---

// BundlesRoot returns {Root}/bundles.
func (l Layout) BundlesRoot() string { return filepath.Join(l.Root, "bundles") }

// BundleDir returns {Root}/bundles/{version}.
func (l Layout) BundleDir(version string) string {
	return filepath.Join(l.BundlesRoot(), version)
}

// BundleArchivePath returns the .tar.gz path for a bundle version.
func (l Layout) BundleArchivePath(version string) string {
	return filepath.Join(l.BundlesRoot(), version+".tar.gz")
}

// CurrentSymlink returns the path to the "current" symlink that points to the
// active bundle directory.
func (l Layout) CurrentSymlink() string {
	return filepath.Join(l.BundlesRoot(), "current")
}

// EnsureBundlesRoot creates {Root}/bundles if missing.
func (l Layout) EnsureBundlesRoot() error {
	if err := os.MkdirAll(l.BundlesRoot(), 0o755); err != nil {
		return fmt.Errorf("ensure bundles root: %w", err)
	}
	return nil
}

// SwapFilesDir is the atomic-rename primitive used by Refresh:
// it expects newDir to be a freshly built sibling of FilesDir(id), and
// renames it into place after preserving the previous tree under a
// time-stamped sibling name (which the caller may best-effort cleanup).
func (l Layout) SwapFilesDir(id, newDir string) (oldDir string, err error) {
	if id == "" {
		return "", errors.New("custom: source id is empty")
	}
	cur := l.FilesDir(id)
	stamp := time.Now().UTC().Format("20060102T150405.000000000")
	old := filepath.Join(l.SourceDir(id), "files.old-"+stamp)

	if _, statErr := os.Stat(cur); statErr == nil {
		if err := os.Rename(cur, old); err != nil {
			return "", fmt.Errorf("rotate current: %w", err)
		}
	} else if !errors.Is(statErr, fs.ErrNotExist) {
		return "", fmt.Errorf("stat current: %w", statErr)
	} else {
		old = ""
	}

	if err := os.Rename(newDir, cur); err != nil {
		// Best-effort restore.
		if old != "" {
			_ = os.Rename(old, cur)
		}
		return "", fmt.Errorf("install new files: %w", err)
	}
	return old, nil
}
