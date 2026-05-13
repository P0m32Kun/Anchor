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
	return os.MkdirAll(l.Root, 0o750)
}

func (l Layout) FilePath(id string) string {
	return filepath.Join(l.Root, id+".json")
}

func (l Layout) WriteFile(id string, data []byte) error {
	path := l.FilePath(id)
	if err := safefs.ValidateRelPath(id + ".json"); err != nil {
		return fmt.Errorf("validate path: %w", err)
	}
	return os.WriteFile(path, data, 0o640)
}

func (l Layout) ReadFile(id string) ([]byte, error) {
	path := l.FilePath(id)
	return os.ReadFile(path)
}

func (l Layout) DeleteFile(id string) error {
	path := l.FilePath(id)
	return os.Remove(path)
}
