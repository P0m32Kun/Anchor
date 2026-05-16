package toolguard

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Allowlist gates execution of external binaries. It validates both the
// binary name (via basename, so /tmp/nuclei is rejected even if nuclei is
// allowed) and arguments (shell metacharacters are rejected).
type Allowlist struct {
	binaries map[string]struct{}
}

// NewAllowlist returns an Allowlist preloaded with the default set of
// security-scanning tools used by Anchor.
func NewAllowlist() *Allowlist {
	return &Allowlist{
		binaries: map[string]struct{}{
			"subfinder": {},
			"dnsx":      {},
			"httpx":     {},
			"naabu":     {},
			"nmap":      {},
			"nuclei":    {},
			"cdncheck":  {},
			"git":       {},
			"git.exe":   {},
			"sh":        {},
			"bash":      {},
		},
	}
}

// Validate returns an error if binary is not on the allowlist or if any
// argument contains shell metacharacters.
func (a *Allowlist) Validate(binary string, args []string) error {
	base := filepath.Base(binary)
	if _, ok := a.binaries[base]; !ok {
		return fmt.Errorf("binary %q not in allowlist", base)
	}
	for i, arg := range args {
		if hasShellMeta(arg) {
			return fmt.Errorf("arg[%d] of %q contains shell metacharacters: %q", i, base, arg)
		}
	}
	return nil
}

// Allow registers an additional binary name. Used in tests and for
// extending the list at runtime.
func (a *Allowlist) Allow(name string) {
	a.binaries[name] = struct{}{}
}

// hasShellMeta reports whether s contains characters that could be
// interpreted as shell metacharacters if the argument were ever
// concatenated into a shell string.
func hasShellMeta(s string) bool {
	return strings.ContainsAny(s, ";|&><`$(){}[]\n\r")
}
