package dictionary

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/P0m32Kun/Anchor/internal/builtin"
	"github.com/P0m32Kun/Anchor/internal/models"
)

// categoryMap routes top-level directories inside the seed root to the
// project's existing DictionaryCategory enum. Adding a new top-level dir
// to the source repo only requires extending this map.
var categoryMap = map[string]models.DictionaryCategory{
	"path":     models.DictionaryCategoryDirscan,
	"api":      models.DictionaryCategoryDirscan,
	"backup":   models.DictionaryCategoryDirscan,
	"password": models.DictionaryCategoryCustom,
	"username": models.DictionaryCategoryCustom,
}

// SeedBuiltin walks rootDir for <category>/*.txt files and upserts each one as
// a builtin dictionary. Idempotent: existing rows are updated in place,
// orphaned builtin rows whose backing file vanished are removed.
//
// rootDir layout (matches RBKD-SEC/dict):
//
//	rootDir/
//	  path/*.txt        → dirscan
//	  api/*.txt         → dirscan
//	  backup/*.txt      → dirscan
//	  password/*.txt    → custom
//	  username/*.txt    → custom
//
// File IDs are derived from their relative path so they stay stable across
// restarts (frontend dropdowns and saved scan configs keep working).
func (m *Manager) SeedBuiltin(rootDir string) error {
	info, err := os.Stat(rootDir)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("[dictionary] seed: %s missing, skipping builtin seed", rootDir)
			return nil
		}
		return fmt.Errorf("stat seed root: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("seed root %s is not a directory", rootDir)
	}

	seen := make(map[string]bool)
	now := time.Now().UTC()
	rev := builtin.HeadShort(rootDir)
	descSuffix := rev
	if descSuffix != "" {
		descSuffix = " @ " + rev
	}

	err = filepath.Walk(rootDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(info.Name(), ".txt") {
			return nil
		}

		rel, err := filepath.Rel(rootDir, path)
		if err != nil {
			return err
		}
		parts := strings.SplitN(filepath.ToSlash(rel), "/", 2)
		if len(parts) < 2 {
			return nil
		}
		topDir := parts[0]
		category, ok := categoryMap[topDir]
		if !ok {
			log.Printf("[dictionary] seed: unmapped top-level dir %q (file %s), skipping", topDir, rel)
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			log.Printf("[dictionary] seed: read %s: %v", rel, err)
			return nil
		}

		id := "builtin:" + filepath.ToSlash(rel)
		seen[id] = true

		d := &models.Dictionary{
			ID:          id,
			Name:        filepath.ToSlash(rel),
			Description: fmt.Sprintf("RBKD-SEC built-in %s%s", topDir, descSuffix),
			Category:    category,
			FilePath:    path,
			LineCount:   countLines(content),
			SizeBytes:   int64(len(content)),
			Builtin:     true,
			Enabled:     true,
			UpdatedAt:   now,
		}

		existing, err := m.q.GetDictionary(id)
		if err != nil {
			log.Printf("[dictionary] seed: get %s: %v", id, err)
			return nil
		}
		if existing == nil {
			d.CreatedAt = now
			if err := m.q.CreateDictionary(d); err != nil {
				log.Printf("[dictionary] seed: insert %s: %v", id, err)
			}
			return nil
		}

		// Preserve CreatedAt and user-disabled state across re-seeds.
		d.CreatedAt = existing.CreatedAt
		d.Enabled = existing.Enabled
		if err := m.q.UpdateDictionary(d); err != nil {
			log.Printf("[dictionary] seed: update %s: %v", id, err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("walk seed root: %w", err)
	}

	// Drop builtin rows whose backing file was removed from the repo.
	existing, err := m.q.ListBuiltinDictionaries()
	if err != nil {
		return fmt.Errorf("list builtin dictionaries: %w", err)
	}
	for _, d := range existing {
		if seen[d.ID] {
			continue
		}
		if err := m.q.DeleteDictionary(d.ID); err != nil {
			log.Printf("[dictionary] seed: remove orphan %s: %v", d.ID, err)
		}
	}

	log.Printf("[dictionary] seed: %d builtin dictionaries reconciled from %s", len(seen), rootDir)
	return nil
}
