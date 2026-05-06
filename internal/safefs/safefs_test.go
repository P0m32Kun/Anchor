package safefs

import (
	stdErrors "errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestValidateRelPath(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{"empty", "", ErrEmptyPath},
		{"dot", ".", ErrEmptyPath},
		{"absolute unix", "/etc/passwd", ErrAbsolutePath},
		{"leading slash", "/foo/bar.yaml", ErrAbsolutePath},
		{"traversal direct", "..", ErrTraversal},
		{"traversal prefix", "../etc/passwd", ErrTraversal},
		{"traversal in middle", "templates/../../etc/passwd", ErrTraversal},
		{"nul byte", "templates/foo\x00bar.yaml", ErrNULByte},
		{"valid template", "templates/web/foo.yaml", nil},
		{"valid yml", "templates/web/foo.yml", nil},
		{"valid payload", "payloads/wordlists/x.txt", nil},
		{"valid nested", "workflows/a/b/c.yaml", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRelPath(tt.input)
			if !stdErrors.Is(err, tt.wantErr) {
				t.Fatalf("ValidateRelPath(%q) = %v, want %v", tt.input, err, tt.wantErr)
			}
		})
	}

	// Backslash is only rejected on non-Windows platforms.
	t.Run("backslash on non-windows", func(t *testing.T) {
		err := ValidateRelPath(`templates\foo.yaml`)
		if runtime.GOOS == "windows" {
			if err != nil {
				t.Fatalf("backslash should be allowed on Windows, got %v", err)
			}
		} else {
			if !stdErrors.Is(err, ErrBackslash) {
				t.Fatalf("expected ErrBackslash, got %v", err)
			}
		}
	})
}

func TestJoinUnderRoot(t *testing.T) {
	root := t.TempDir()

	t.Run("valid rel joins under root", func(t *testing.T) {
		got, err := JoinUnderRoot(root, "templates/web/foo.yaml")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := filepath.Join(root, "templates", "web", "foo.yaml")
		if got != want {
			t.Fatalf("got %q want %q", got, want)
		}
	})

	t.Run("traversal rejected", func(t *testing.T) {
		_, err := JoinUnderRoot(root, "../etc/passwd")
		if !stdErrors.Is(err, ErrTraversal) {
			t.Fatalf("got %v want ErrTraversal", err)
		}
	})

	t.Run("absolute rejected", func(t *testing.T) {
		_, err := JoinUnderRoot(root, "/etc/passwd")
		if !stdErrors.Is(err, ErrAbsolutePath) {
			t.Fatalf("got %v want ErrAbsolutePath", err)
		}
	})

	t.Run("empty rel rejected", func(t *testing.T) {
		_, err := JoinUnderRoot(root, "")
		if !stdErrors.Is(err, ErrEmptyPath) {
			t.Fatalf("got %v want ErrEmptyPath", err)
		}
	})
}

func TestEnsureNoSymlinkEscape(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink semantics differ on Windows")
	}

	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("nope"), 0o600); err != nil {
		t.Fatal(err)
	}

	// 1. Plain path inside the root: allowed.
	plain := filepath.Join(root, "templates", "ok.yaml")
	if err := os.MkdirAll(filepath.Dir(plain), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(plain, []byte("id: ok"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := EnsureNoSymlinkEscape(root, plain); err != nil {
		t.Fatalf("plain path inside root must succeed, got %v", err)
	}

	// 2. Path that does not exist yet (write-before-create): allowed.
	pending := filepath.Join(root, "workflows", "new.yaml")
	if err := EnsureNoSymlinkEscape(root, pending); err != nil {
		t.Fatalf("pending path under root must succeed, got %v", err)
	}

	// 3. Symlink whose target is outside root: rejected.
	link := filepath.Join(root, "leak")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	leaked := filepath.Join(link, "secret.txt")
	err := EnsureNoSymlinkEscape(root, leaked)
	if !stdErrors.Is(err, ErrSymlinkEscape) {
		t.Fatalf("symlink escape must be rejected, got %v", err)
	}

	// 4. Symlink whose target is inside root: allowed.
	innerTarget := filepath.Join(root, "templates")
	innerLink := filepath.Join(root, "alias")
	if err := os.Symlink(innerTarget, innerLink); err != nil {
		t.Fatal(err)
	}
	aliased := filepath.Join(innerLink, "ok.yaml")
	if err := EnsureNoSymlinkEscape(root, aliased); err != nil {
		t.Fatalf("inner-symlink path must succeed, got %v", err)
	}
}

func TestIsAllowedTemplateFile(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"templates/web/foo.yaml", true},
		{"templates/web/foo.yml", true},
		{"templates/cves/CVE-1.YAML", true},
		{"workflows/x.yaml", true},
		{"payloads/wordlists/users.txt", true},
		{"payloads", true},
		{"payloads/sub/file.bin", true},
		{"README.md", false},
		{"evil.sh", false},
		{"templates/web/run.sh", false},
		{"", false},
		{".", false},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := IsAllowedTemplateFile(tt.path); got != tt.want {
				t.Fatalf("IsAllowedTemplateFile(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}
