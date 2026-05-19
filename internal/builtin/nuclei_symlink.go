package builtin

import (
	"os"
	"path/filepath"
)

const (
	RBKDInstallPath   = "RBKD-templates"
	RBKDTemplatesRoot = "/opt/rbkd-templates"
)

// ApplyRBKDNucleiSymlink creates or removes the RBKD-templates symlink under
// ~/nuclei-templates. When enabled, link points at RBKDTemplatesRoot; when
// disabled, only an existing symlink is removed (never /opt/rbkd-templates).
func ApplyRBKDNucleiSymlink(enabled bool) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	link := filepath.Join(home, "nuclei-templates", RBKDInstallPath)

	if !enabled {
		fi, err := os.Lstat(link)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if fi.Mode()&os.ModeSymlink != 0 {
			return os.Remove(link)
		}
		return nil
	}

	parent := filepath.Join(home, "nuclei-templates")
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}

	if fi, err := os.Lstat(link); err == nil {
		if fi.Mode()&os.ModeSymlink != 0 {
			if err := os.Remove(link); err != nil {
				return err
			}
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	return os.Symlink(RBKDTemplatesRoot, link)
}
