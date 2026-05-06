// Package safefs provides path-safety primitives for serving user-supplied
// relative paths from a fixed root on disk. Every operation that joins
// untrusted input with a host directory must go through these helpers.
package safefs

import (
	stdErrors "errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var (
	ErrEmptyPath        = stdErrors.New("safefs: path is empty")
	ErrAbsolutePath     = stdErrors.New("safefs: path must be relative")
	ErrNULByte          = stdErrors.New("safefs: path contains NUL byte")
	ErrTraversal        = stdErrors.New("safefs: path contains traversal segment")
	ErrBackslash        = stdErrors.New("safefs: backslash not allowed in path")
	ErrEscapesRoot      = stdErrors.New("safefs: path escapes root")
	ErrSymlinkEscape    = stdErrors.New("safefs: symlink resolves outside root")
	ErrDisallowedExt    = stdErrors.New("safefs: file extension not allowed")
)

// ValidateRelPath rejects relative paths that are unsafe to join with a root.
// The input is expected to use POSIX-style separators.
func ValidateRelPath(rel string) error {
	if rel == "" {
		return ErrEmptyPath
	}
	if strings.ContainsRune(rel, 0) {
		return ErrNULByte
	}
	if runtime.GOOS != "windows" && strings.ContainsRune(rel, '\\') {
		return ErrBackslash
	}
	if filepath.IsAbs(rel) || strings.HasPrefix(rel, "/") {
		return ErrAbsolutePath
	}

	cleaned := filepath.ToSlash(filepath.Clean(rel))
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") || strings.Contains("/"+cleaned+"/", "/../") {
		return ErrTraversal
	}
	if cleaned == "." {
		return ErrEmptyPath
	}
	return nil
}

// JoinUnderRoot validates rel and joins it under root, returning the absolute
// path. It guarantees the result lies inside root before returning.
func JoinUnderRoot(root, rel string) (string, error) {
	if root == "" {
		return "", fmt.Errorf("safefs: root is empty")
	}
	if err := ValidateRelPath(rel); err != nil {
		return "", err
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("safefs: resolve root: %w", err)
	}
	joined := filepath.Join(absRoot, filepath.FromSlash(rel))
	cleaned := filepath.Clean(joined)
	if cleaned != absRoot && !strings.HasPrefix(cleaned, absRoot+string(filepath.Separator)) {
		return "", ErrEscapesRoot
	}
	return cleaned, nil
}

// EnsureNoSymlinkEscape walks the existing parent chain of abs and verifies
// that no resolved component escapes root. If abs itself does not yet exist
// (i.e. we are about to create it), only the existing prefix is checked.
//
// Both root and abs must already be absolute paths (typically the output of
// JoinUnderRoot for abs and filepath.Abs for root).
func EnsureNoSymlinkEscape(root, abs string) error {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("safefs: resolve root: %w", err)
	}
	resolvedRoot, err := filepath.EvalSymlinks(absRoot)
	if err != nil {
		// Root must exist for this check; surface the original error.
		return fmt.Errorf("safefs: resolve root symlinks: %w", err)
	}

	cur := abs
	for {
		if cur == absRoot || cur == resolvedRoot {
			return nil
		}
		if _, err := os.Lstat(cur); err == nil {
			resolved, rerr := filepath.EvalSymlinks(cur)
			if rerr != nil {
				return fmt.Errorf("safefs: resolve symlinks: %w", rerr)
			}
			if resolved != resolvedRoot && !strings.HasPrefix(resolved, resolvedRoot+string(filepath.Separator)) {
				return ErrSymlinkEscape
			}
			return nil
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return nil
		}
		cur = parent
	}
}

// IsAllowedTemplateFile returns true if the relative path may be served or
// written via the file CRUD APIs. Allowed:
//   - any *.yaml / *.yml file under the source root
//   - any file under the payloads/ subtree
func IsAllowedTemplateFile(rel string) bool {
	cleaned := filepath.ToSlash(filepath.Clean(rel))
	if cleaned == "." || cleaned == "" {
		return false
	}
	lower := strings.ToLower(cleaned)
	if strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".yml") {
		return true
	}
	if cleaned == "payloads" || strings.HasPrefix(cleaned, "payloads/") {
		return true
	}
	return false
}
