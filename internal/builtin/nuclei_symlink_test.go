package builtin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApplyRBKDNucleiSymlink_EnableCreatesSymlink(t *testing.T) {
	home := t.TempDir()
	opt := filepath.Join(home, "opt-rbkd")
	t.Setenv("HOME", home)
	t.Setenv("ANCHOR_BUILTIN_TEMPLATES_ROOT", opt)
	if err := os.MkdirAll(opt, 0o755); err != nil {
		t.Fatal(err)
	}

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
	if target != opt {
		t.Fatalf("target = %q, want %q", target, opt)
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

func TestApplyRBKDNucleiSymlink_DisableRemovesLegacyDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	link := filepath.Join(home, "nuclei-templates", RBKDInstallPath)
	if err := os.MkdirAll(link, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if err := ApplyRBKDNucleiSymlink(false); err != nil {
		t.Fatalf("disable: %v", err)
	}

	if _, err := os.Lstat(link); !os.IsNotExist(err) {
		t.Fatal("expected legacy bundle directory removed")
	}
}

func TestApplyRBKDNucleiSymlink_EnableReplacesLegacyDirectory(t *testing.T) {
	home := t.TempDir()
	opt := filepath.Join(home, "opt-rbkd")
	t.Setenv("HOME", home)
	t.Setenv("ANCHOR_BUILTIN_TEMPLATES_ROOT", opt)
	if err := os.MkdirAll(opt, 0o755); err != nil {
		t.Fatalf("mkdir opt: %v", err)
	}

	link := filepath.Join(home, "nuclei-templates", RBKDInstallPath)
	if err := os.MkdirAll(link, 0o755); err != nil {
		t.Fatalf("mkdir legacy: %v", err)
	}

	if err := ApplyRBKDNucleiSymlink(true); err != nil {
		t.Fatalf("enable: %v", err)
	}

	fi, err := os.Lstat(link)
	if err != nil {
		t.Fatalf("lstat: %v", err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Fatal("expected symlink after replacing legacy directory")
	}
	target, _ := os.Readlink(link)
	if target != opt {
		t.Fatalf("readlink = %q, want %q", target, opt)
	}
}
