package toolrun

import (
	"fmt"
	"io"
	"os"
)

// ArtifactInlineMax is the threshold below which the full artifact content is
// returned inline. Above this, callers should use offset/limit pagination.
const ArtifactInlineMax = 256 * 1024 // 256 KiB

// ArtifactHandle provides lazy access to a task's stdout artifact without
// loading the entire file into memory.
type ArtifactHandle struct {
	Path string
	Size int64
}

// Open returns an io.ReadCloser for the artifact file.
func (h *ArtifactHandle) Open() (io.ReadCloser, error) {
	return os.Open(h.Path)
}

// ReadRange returns a byte range from the artifact file.
func (h *ArtifactHandle) ReadRange(offset, limit int64) ([]byte, error) {
	f, err := os.Open(h.Path)
	if err != nil {
		return nil, fmt.Errorf("open artifact: %w", err)
	}
	defer f.Close()

	if offset > 0 {
		if _, err := f.Seek(offset, io.SeekStart); err != nil {
			return nil, fmt.Errorf("seek artifact: %w", err)
		}
	}

	if limit <= 0 || limit > ArtifactInlineMax {
		limit = ArtifactInlineMax
	}

	buf := make([]byte, limit)
	n, err := io.ReadFull(f, buf)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return buf[:n], fmt.Errorf("read artifact range: %w", err)
	}
	return buf[:n], nil
}
