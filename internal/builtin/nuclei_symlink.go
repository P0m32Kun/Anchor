package builtin

import (
	"os"
	"path/filepath"
)

const (
	RBKDInstallPath   = "RBKD-templates"
	RBKDTemplatesRoot = "/opt/rbkd-templates"
)

// ApplyRBKDNucleiSymlink creates or removes the RBKD-templates entry under
// ~/nuclei-templates. When enabled, link is a symlink to RBKDTemplatesRoot.
// When disabled, the install path is removed (symlink or legacy bundle directory)
// so nuclei -tags does not scan RBKD templates. /opt/rbkd-templates is never deleted.
func ApplyRBKDNucleiSymlink(enabled bool) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	link := filepath.Join(home, "nuclei-templates", RBKDInstallPath)

	if !enabled {
		return removeInstallPath(link)
	}

	parent := filepath.Join(home, "nuclei-templates")
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}
	if err := removeInstallPath(link); err != nil {
		return err
	}
	target := LoadConfig().TemplatesRoot
	if target == "" {
		target = RBKDTemplatesRoot
	}
	return os.Symlink(target, link)
}

func removeInstallPath(path string) error {
	fi, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if fi.IsDir() && fi.Mode()&os.ModeSymlink == 0 {
		return os.RemoveAll(path)
	}
	return os.Remove(path)
}
