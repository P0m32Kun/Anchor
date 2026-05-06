package custom

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/P0m32Kun/Anchor/internal/safefs"
)

// Upload limits per the design doc §13.
const (
	uploadMaxYAMLBytes      = 1 << 20         // 1 MiB single-file
	uploadMaxZipBytes       = 32 << 20        // 32 MiB compressed input
	uploadMaxZipEntries     = 1000
	uploadMaxZipUncompressed = int64(10 << 20) // 10 MiB total
	uploadMaxEntryBytes     = int64(5 << 20)  // 5 MiB per entry
)

// ExtractUpload writes a single template (.yaml/.yml) or unpacks a .zip
// archive into the source's files/ tree. The caller is responsible for the
// outer multipart parsing and any HTTP-level size guard.
//
// filename is used only to dispatch on extension; the on-disk path is
// derived under the source's templates/ subtree.
func ExtractUpload(layout Layout, sourceID, filename string, body io.Reader) error {
	if filename == "" {
		return errors.New("custom: upload filename is empty")
	}
	ext := strings.ToLower(filepath.Ext(filename))

	switch ext {
	case ".yaml", ".yml":
		return extractYAMLUpload(layout, sourceID, filename, body)
	case ".zip":
		return extractZipUpload(layout, sourceID, body)
	default:
		return fmt.Errorf("custom: unsupported upload type %q; expected .yaml, .yml, or .zip", ext)
	}
}

func extractYAMLUpload(layout Layout, sourceID, filename string, body io.Reader) error {
	limited := io.LimitReader(body, uploadMaxYAMLBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return fmt.Errorf("read yaml upload: %w", err)
	}
	if int64(len(data)) > uploadMaxYAMLBytes {
		return fmt.Errorf("custom: yaml upload exceeds %d bytes", uploadMaxYAMLBytes)
	}

	base := filepath.Base(filename)
	stem := strings.TrimSuffix(base, filepath.Ext(base))
	stem = sanitizeUploadStem(stem)
	if stem == "" {
		return errors.New("custom: yaml upload has no usable filename")
	}
	rel := path.Join("templates", stem+".yaml")

	if !safefs.IsAllowedTemplateFile(rel) {
		return errors.New("custom: yaml upload rejected by extension policy")
	}
	if err := safefs.ValidateRelPath(rel); err != nil {
		return fmt.Errorf("custom: yaml upload path: %w", err)
	}

	if err := layout.WriteFileAtomic(sourceID, rel, data); err != nil {
		return fmt.Errorf("write yaml upload: %w", err)
	}
	return nil
}

func extractZipUpload(layout Layout, sourceID string, body io.Reader) error {
	limited := io.LimitReader(body, uploadMaxZipBytes+1)
	raw, err := io.ReadAll(limited)
	if err != nil {
		return fmt.Errorf("read zip upload: %w", err)
	}
	if int64(len(raw)) > uploadMaxZipBytes {
		return fmt.Errorf("custom: zip upload exceeds %d bytes", uploadMaxZipBytes)
	}

	zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return fmt.Errorf("custom: open zip: %w", err)
	}
	if len(zr.File) > uploadMaxZipEntries {
		return fmt.Errorf("custom: zip has %d entries (max %d)", len(zr.File), uploadMaxZipEntries)
	}

	var totalUncompressed int64

	for _, f := range zr.File {
		if f.Mode()&0o20000000 != 0 {
			// archive/zip exposes symlinks via mode bits (fs.ModeSymlink).
			return fmt.Errorf("custom: zip entry %q is a symlink", f.Name)
		}
		if f.Mode()&(1<<27) != 0 || f.Mode().IsDir() || strings.HasSuffix(f.Name, "/") {
			continue
		}

		rel, err := normaliseZipEntryPath(f.Name)
		if err != nil {
			return err
		}
		if !safefs.IsAllowedTemplateFile(rel) {
			return fmt.Errorf("custom: zip entry %q rejected by extension policy", f.Name)
		}
		if err := safefs.ValidateRelPath(rel); err != nil {
			return fmt.Errorf("custom: zip entry %q: %w", f.Name, err)
		}

		if int64(f.UncompressedSize64) > uploadMaxEntryBytes {
			return fmt.Errorf("custom: zip entry %q is %d bytes (max %d)", f.Name, f.UncompressedSize64, uploadMaxEntryBytes)
		}

		totalUncompressed += int64(f.UncompressedSize64)
		if totalUncompressed > uploadMaxZipUncompressed {
			return fmt.Errorf("custom: zip uncompressed total exceeds %d bytes", uploadMaxZipUncompressed)
		}

		rc, openErr := f.Open()
		if openErr != nil {
			return fmt.Errorf("custom: open zip entry %q: %w", f.Name, openErr)
		}
		// Defensive copy with LimitReader in case the header lied about size.
		buf := make([]byte, 0, f.UncompressedSize64)
		w := bytes.NewBuffer(buf)
		n, copyErr := io.Copy(w, io.LimitReader(rc, uploadMaxEntryBytes+1))
		_ = rc.Close()
		if copyErr != nil {
			return fmt.Errorf("custom: read zip entry %q: %w", f.Name, copyErr)
		}
		if n > uploadMaxEntryBytes {
			return fmt.Errorf("custom: zip entry %q exceeds per-entry limit", f.Name)
		}

		if err := layout.WriteFileAtomic(sourceID, rel, w.Bytes()); err != nil {
			return fmt.Errorf("custom: write zip entry %q: %w", f.Name, err)
		}
	}
	return nil
}

// normaliseZipEntryPath converts a zip entry name to a clean POSIX relative
// path suitable for safefs validation. Strips leading "./" and any leading
// slash, rejects empty results.
func normaliseZipEntryPath(raw string) (string, error) {
	cleaned := path.Clean(filepath.ToSlash(raw))
	cleaned = strings.TrimPrefix(cleaned, "./")
	cleaned = strings.TrimPrefix(cleaned, "/")
	if cleaned == "" || cleaned == "." {
		return "", fmt.Errorf("custom: zip entry %q has empty path", raw)
	}
	return cleaned, nil
}

// sanitizeUploadStem drops path separators and characters likely to confuse
// downstream tools, leaving a conservative ASCII-friendly stem.
func sanitizeUploadStem(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "..", "")
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '-' || r == '_' || r == '.':
			b.WriteRune(r)
		}
	}
	return b.String()
}
