package builtin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApplyRBKDNucleiSymlink_EnableCreatesSymlink(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := ApplyRBKDNucleiSymlink(true); err != nil {
		t.Fatalf("enable: %v", err)
	}

	link := filepath.Join(home, "nuclei-templates", RBKDInstallPath)
	fi, err := os.Lstat(link)
	if err != nil {
		t.Fatalf("lstat link: %v", err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Fatal("expected symlink")
	}

	target, err := os.Readlink(link)
	if err != nil {
		t.Fatalf("readlink: %v", err)
	}
	if target != RBKDTemplatesRoot {
		t.Fatalf("target = %q, want %q", target, RBKDTemplatesRoot)
	}
}

func TestApplyRBKDNucleiSymlink_DisableRemovesSymlink(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := ApplyRBKDNucleiSymlink(true); err != nil {
		t.Fatalf("enable: %v", err)
	}
	if err := ApplyRBKDNucleiSymlink(false); err != nil {
		t.Fatalf("disable: %v", err)
	}

	link := filepath.Join(home, "nuclei-templates", RBKDInstallPath)
	if _, err := os.Lstat(link); !os.IsNotExist(err) {
		t.Fatal("expected symlink removed")
	}
}

func TestApplyRBKDNucleiSymlink_DisablePreservesDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	link := filepath.Join(home, "nuclei-templates", RBKDInstallPath)
	if err := os.MkdirAll(link, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if err := ApplyRBKDNucleiSymlink(false); err != nil {
		t.Fatalf("disable: %v", err)
	}

	fi, err := os.Stat(link)
	if err != nil {
		t.Fatalf("directory removed unexpectedly: %v", err)
	}
	if !fi.IsDir() {
		t.Fatal("expected directory to remain")
	}
}
