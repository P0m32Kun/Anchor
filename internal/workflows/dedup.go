package workflows

import (
	"crypto/sha256"
	"fmt"
	"net/url"
	"strings"

	"github.com/P0m32Kun/Anchor/internal/util"
)

// normalizeScanURL canonicalizes a URL for exact-match deduplication (path-level).
func normalizeScanURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return raw
	}
	u.Fragment = ""
	if u.Path == "" {
		u.Path = "/"
	}
	u.Path = strings.TrimSuffix(u.Path, "/")
	if u.Path == "" {
		u.Path = "/"
	}
	return strings.ToLower(u.String())
}

// computeDedupKey hashes template + scan origin + matcher so the same
// vulnerability on different paths of one host:port collapses to one finding.
func computeDedupKey(templateID, host, matcherName string) string {
	origin := util.ScanOrigin(host)
	h := sha256.New()
	fmt.Fprintf(h, "%s|%s|%s", templateID, origin, matcherName)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// dedupHTTPTargetsByOrigin keeps one target per host:port so httpx is not run
// multiple times on the same service with different URL forms.
func dedupHTTPTargetsByOrigin(targets []string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, t := range targets {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		origin := util.ScanOrigin(t)
		if origin == "" || seen[origin] {
			continue
		}
		seen[origin] = true
		out = append(out, t)
	}
	return out
}

// dedupStrings removes duplicate strings from a slice.
func dedupStrings(s []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, v := range s {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	return result
}
