package toolguard

import (
	"strings"
	"testing"
)

func TestAllowlist_AllowedBinary(t *testing.T) {
	a := NewAllowlist()
	for _, name := range []string{"subfinder", "dnsx", "httpx", "naabu", "nmap", "nuclei", "cdncheck", "ffuf", "urlfinder", "git", "sh"} {
		if err := a.Validate(name, nil); err != nil {
			t.Errorf("%q should be allowed: %v", name, err)
		}
	}
}

func TestAllowlist_RejectUnknownBinary(t *testing.T) {
	a := NewAllowlist()
	if err := a.Validate("curl", []string{"-I", "example.com"}); err == nil {
		t.Fatal("expected curl to be rejected")
	}
}

func TestAllowlist_RejectPathTraversalBinary(t *testing.T) {
	a := NewAllowlist()
	if err := a.Validate("/tmp/evil", nil); err == nil {
		t.Fatal("expected absolute path binary to be rejected")
	}
	if err := a.Validate("../../bin/rm", nil); err == nil {
		t.Fatal("expected relative path traversal binary to be rejected")
	}
}

func TestAllowlist_AllowPathWithAllowedBasename(t *testing.T) {
	a := NewAllowlist()
	// Basename is "nuclei", which is allowed, even though the path is absolute.
	if err := a.Validate("/usr/local/bin/nuclei", []string{"-version"}); err != nil {
		t.Fatalf("expected /usr/local/bin/nuclei to be allowed: %v", err)
	}
}

func TestAllowlist_RejectShellMetaInArgs(t *testing.T) {
	a := NewAllowlist()
	metas := []string{
		"; rm -rf /",
		"| cat /etc/passwd",
		"& echo pwned",
		"> /dev/null",
		"< /etc/shadow",
		"`whoami`",
		"$(id)",
		"${PATH}",
		"hello\nworld",
	}
	for _, m := range metas {
		if err := a.Validate("nuclei", []string{"-u", m}); err == nil {
			t.Fatalf("expected arg %q to be rejected", m)
		}
	}
}

func TestAllowlist_AllowNormalArgs(t *testing.T) {
	a := NewAllowlist()
	args := []string{
		"-u", "https://example.com",
		"-json",
		"-o", "/tmp/output.json",
		"-tags", "cves",
		"-w", "/opt/rbkd-templates/workflows",
	}
	if err := a.Validate("nuclei", args); err != nil {
		t.Fatalf("expected normal args to pass: %v", err)
	}
}

func TestAllowlist_AllowRegister(t *testing.T) {
	a := NewAllowlist()
	if err := a.Validate("custom-tool", nil); err == nil {
		t.Fatal("expected custom-tool to be rejected before Allow")
	}
	a.Allow("custom-tool")
	if err := a.Validate("custom-tool", nil); err != nil {
		t.Fatalf("expected custom-tool to be allowed after Allow: %v", err)
	}
}

func TestAllowlist_ErrorMessages(t *testing.T) {
	a := NewAllowlist()

	err := a.Validate("curl", nil)
	if err == nil || !strings.Contains(err.Error(), "not in allowlist") {
		t.Fatalf("expected 'not in allowlist' error, got: %v", err)
	}

	err = a.Validate("nuclei", []string{"-u", "; rm -rf /"})
	if err == nil || !strings.Contains(err.Error(), "shell metacharacters") {
		t.Fatalf("expected 'shell metacharacters' error, got: %v", err)
	}
}
