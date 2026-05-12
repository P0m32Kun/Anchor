package httpxfp

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/P0m32Kun/Anchor/internal/safefs"
)

// Layout describes the on-disk directory tree rooted at {dataDir}/httpx/fingerprints.
type Layout struct {
	Root string
}

func NewLayout(dataDir string) Layout {
	return Layout{Root: filepath.Join(dataDir, "httpx", "fingerprints")}
}

func (l Layout) EnsureRoot() error {
	for _, sub := range []string{"favicon", "tech_detect"} {
		if err := os.MkdirAll(filepath.Join(l.Root, sub), 0o750); err != nil {
			return err
		}
	}
	return nil
}

func (l Layout) FilePath(id string, fpType string) string {
	return filepath.Join(l.Root, fpType, id+".json")
}

func (l Layout) WriteFile(id string, fpType string, data []byte) error {
	path := l.FilePath(id, fpType)
	if err := safefs.ValidateRelPath(id + ".json"); err != nil {
		return fmt.Errorf("validate path: %w", err)
	}
	return os.WriteFile(path, data, 0o640)
}

func (l Layout) ReadFile(id string, fpType string) ([]byte, error) {
	path := l.FilePath(id, fpType)
	return os.ReadFile(path)
}

func (l Layout) DeleteFile(id string, fpType string) error {
	path := l.FilePath(id, fpType)
	return os.Remove(path)
}
