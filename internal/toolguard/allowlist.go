package toolguard

import (
	"fmt"
	"path/filepath"
	"strings"
)

// systemBinaries are non-scanning utilities that Worker may need for
// builtin sync or lifecycle scripts. These are NOT declared in tools/*.yaml
// but must still pass the allowlist gate.
var systemBinaries = []string{"git", "git.exe", "sh", "bash"}

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
			"subfinder":  {},
			"dnsx":       {},
			"httpx":      {},
			"naabu":      {},
			"nmap":       {},
			"nuclei":     {},
			"cdncheck":   {},
			"ffuf":       {},
			"gau":        {},
		"katana":     {},
		"spoor":      {},
		"chromium":   {},
		"google-chrome":      {},
		"google-chrome-stable": {},
		"chromium-browser":    {},
		"git":        {},
			"git.exe":    {},
			"sh":         {},
			"bash":       {},
		},
	}
}

// NewAllowlistFromBinaries returns an Allowlist derived from a list of
// binary names (typically from toolregistry.Registry.Binaries()),
// avoiding the hardcoded binary list. System utilities (git, sh, bash)
// are added automatically.
func NewAllowlistFromBinaries(binaries []string) *Allowlist {
	bins := make(map[string]struct{})
	for _, b := range binaries {
		bins[b] = struct{}{}
	}
	for _, s := range systemBinaries {
		bins[s] = struct{}{}
	}
	return &Allowlist{binaries: bins}
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
