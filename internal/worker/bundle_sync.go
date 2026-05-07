package worker

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// BundleManifest mirrors the server-side manifest structure.
type BundleManifest struct {
	Version   string              `json:"version"`
	Sources   []BundleSourceEntry `json:"sources"`
	CreatedAt time.Time           `json:"created_at"`
}

// BundleSourceEntry describes one source in the bundle.
type BundleSourceEntry struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Files    []string `json:"files"`
	Checksum string   `json:"checksum"`
}

// BundleSyncer manages the local nuclei custom template bundle on a worker node.
// It fetches the manifest from the server, downloads and extracts new bundles,
// and atomically switches the "current" symlink.
type BundleSyncer struct {
	dataDir    string
	coreURL    string
	apiToken   string
	httpClient *http.Client
}

// NewBundleSyncer creates a syncer for the worker's local bundle cache.
func NewBundleSyncer(dataDir, coreURL, apiToken string) *BundleSyncer {
	return &BundleSyncer{
		dataDir:    dataDir,
		coreURL:    coreURL,
		apiToken:   apiToken,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

// BundlesRoot returns the local bundles directory.
func (s *BundleSyncer) BundlesRoot() string {
	return filepath.Join(s.dataDir, "nuclei", "custom", "bundles")
}

// CurrentBundleDir returns the path to the "current" symlink target.
func (s *BundleSyncer) CurrentBundleDir() string {
	return filepath.Join(s.BundlesRoot(), "current")
}

// CurrentVersion reads the active bundle version from the current symlink.
// Returns "" if no bundle is synced yet.
func (s *BundleSyncer) CurrentVersion() string {
	current := s.CurrentBundleDir()
	target, err := os.Readlink(current)
	if err != nil {
		return ""
	}
	// target is the version directory name
	return filepath.Base(target)
}

// Sync checks the server manifest and updates the local bundle if needed.
// Returns the current active version after sync.
func (s *BundleSyncer) Sync() (string, error) {
	manifest, err := s.fetchManifest()
	if err != nil {
		return s.CurrentVersion(), fmt.Errorf("fetch manifest: %w", err)
	}
	if manifest == nil {
		return s.CurrentVersion(), nil
	}

	cur := s.CurrentVersion()
	hasContent := s.hasBundleContent(manifest.Version)
	log.Printf("[worker] sync check: cur=%s manifest=%s hasContent=%v", cur, manifest.Version, hasContent)
	// Check if we need to sync: version differs OR directory is empty
	// (shared-volume scenario where server creates empty dir + symlink)
	needsSync := cur != manifest.Version || !hasContent

	if !needsSync {
		return cur, nil
	}

	log.Printf("[worker] bundle sync: %s -> %s", cur, manifest.Version)
	if err := s.downloadAndExtract(manifest.Version); err != nil {
		return cur, fmt.Errorf("download bundle: %w", err)
	}
	if err := s.switchCurrent(manifest.Version); err != nil {
		return cur, fmt.Errorf("switch current: %w", err)
	}

	log.Printf("[worker] bundle synced to %s", manifest.Version)
	return manifest.Version, nil
}

// hasBundleContent checks if the version directory has actual content (not just empty).
func (s *BundleSyncer) hasBundleContent(version string) bool {
	dir := filepath.Join(s.BundlesRoot(), version)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	return len(entries) > 0
}

// fetchManifest retrieves the active manifest from the server.
func (s *BundleSyncer) fetchManifest() (*BundleManifest, error) {
	req, _ := http.NewRequest("GET", s.coreURL+"/nuclei/custom/manifest", nil)
	req.Header.Set("Authorization", "Bearer "+s.apiToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode == http.StatusServiceUnavailable {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("manifest request: %s", resp.Status)
	}

	var manifest BundleManifest
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return nil, fmt.Errorf("decode manifest: %w", err)
	}
	return &manifest, nil
}

// downloadAndExtract fetches the bundle .tar.gz and extracts it to a version directory.
func (s *BundleSyncer) downloadAndExtract(version string) error {
	url := fmt.Sprintf("%s/nuclei/custom/bundles/%s", s.coreURL, version)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+s.apiToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	log.Printf("[worker] bundle download: %s -> status=%d, content-length=%d", url, resp.StatusCode, resp.ContentLength)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download bundle: %s", resp.Status)
	}

	bundlesRoot := s.BundlesRoot()
	if err := os.MkdirAll(bundlesRoot, 0o755); err != nil {
		return fmt.Errorf("mkdir bundles: %w", err)
	}

	// Extract to a temp dir first, then rename atomically
	tmpDir := filepath.Join(bundlesRoot, ".tmp-"+version)
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return fmt.Errorf("mkdir tmp: %w", err)
	}

	if err := extractTarGz(resp.Body, tmpDir); err != nil {
		os.RemoveAll(tmpDir)
		return fmt.Errorf("extract: %w", err)
	}

	target := filepath.Join(bundlesRoot, version)
	// Remove existing if present
	os.RemoveAll(target)
	if err := os.Rename(tmpDir, target); err != nil {
		os.RemoveAll(tmpDir)
		return fmt.Errorf("rename: %w", err)
	}

	return nil
}

// switchCurrent atomically updates the "current" symlink to point to version.
func (s *BundleSyncer) switchCurrent(version string) error {
	bundlesRoot := s.BundlesRoot()
	current := s.CurrentBundleDir()
	target := filepath.Join(bundlesRoot, version)

	// Create relative symlink
	relPath, err := filepath.Rel(bundlesRoot, target)
	if err != nil {
		return fmt.Errorf("compute relative path: %w", err)
	}

	// Atomic: write to tmp link, then rename
	tmpLink := current + ".tmp"
	os.Remove(tmpLink)
	if err := os.Symlink(relPath, tmpLink); err != nil {
		return fmt.Errorf("create tmp symlink: %w", err)
	}
	if err := os.Rename(tmpLink, current); err != nil {
		os.Remove(tmpLink)
		return fmt.Errorf("rename symlink: %w", err)
	}
	return nil
}

// TemplatesDir returns the path to the current bundle's templates directory.
func (s *BundleSyncer) TemplatesDir() string {
	return filepath.Join(s.CurrentBundleDir(), "templates")
}

// WorkflowsDir returns the path to the current bundle's workflows directory.
func (s *BundleSyncer) WorkflowsDir() string {
	return filepath.Join(s.CurrentBundleDir(), "workflows")
}

// TemplateVersionsJSON returns the JSON string for heartbeat reporting.
func (s *BundleSyncer) TemplateVersionsJSON() string {
	v := s.CurrentVersion()
	if v == "" {
		return "{}"
	}
	b, _ := json.Marshal(map[string]string{"nuclei_custom": v})
	return string(b)
}

// extractTarGz extracts a .tar.gz stream into destDir.
func extractTarGz(r io.Reader, destDir string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("gzip reader: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	count := 0
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar next (after %d entries): %w", count, err)
		}

		// Sanitize path
		name := filepath.Clean(hdr.Name)
		if name == ".." || strings.HasPrefix(name, "../") || strings.HasPrefix(name, "/") {
			return fmt.Errorf("unsafe tar entry: %s", hdr.Name)
		}

		target := filepath.Join(destDir, name)

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return fmt.Errorf("mkdir %s: %w", name, err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return fmt.Errorf("mkdir parent %s: %w", name, err)
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
			if err != nil {
				return fmt.Errorf("create %s: %w", name, err)
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return fmt.Errorf("write %s: %w", name, err)
			}
			f.Close()
			count++
		default:
			// Skip non-regular files (symlinks etc.)
			continue
		}
	}
	log.Printf("[worker] extractTarGz: extracted %d files to %s", count, destDir)
	return nil
}
