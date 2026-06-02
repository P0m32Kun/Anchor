package workflows

import (
	"crypto/sha256"
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/P0m32Kun/Anchor/internal/models"
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

// scanOrigin returns host:port for a URL or host string, stripping paths so the
// same service is not scanned or deduplicated separately per path.
func scanOrigin(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.Contains(raw, "://") {
		u, err := url.Parse(raw)
		if err == nil && u.Host != "" {
			host := u.Hostname()
			port := u.Port()
			if port == "" {
				switch strings.ToLower(u.Scheme) {
				case "http":
					port = "80"
				case "https":
					port = "443"
				}
			}
			if port != "" {
				return strings.ToLower(net.JoinHostPort(host, port))
			}
			return strings.ToLower(u.Host)
		}
	}
	if idx := strings.Index(raw, "/"); idx > 0 {
		raw = raw[:idx]
	}
	host, port, err := net.SplitHostPort(raw)
	if err != nil {
		return strings.ToLower(raw)
	}
	return strings.ToLower(net.JoinHostPort(host, port))
}

// computeDedupKey hashes template + scan origin + matcher so the same
// vulnerability on different paths of one host:port collapses to one finding.
func computeDedupKey(templateID, host, matcherName string) string {
	origin := scanOrigin(host)
	h := sha256.New()
	fmt.Fprintf(h, "%s|%s|%s", templateID, origin, matcherName)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// filterURLsForSecondaryScan drops URLs already covered by exact URL or by an
// existing endpoint on the same host:port (scan origin).
func filterURLsForSecondaryScan(urls []string, existingEndpoints []*models.WebEndpoint) []string {
	existingURL := make(map[string]bool, len(existingEndpoints))
	existingOrigin := make(map[string]bool, len(existingEndpoints))
	for _, ep := range existingEndpoints {
		if ep.URL != "" {
			existingURL[normalizeScanURL(ep.URL)] = true
			existingOrigin[scanOrigin(ep.URL)] = true
		}
	}

	seenURL := make(map[string]bool)
	seenOrigin := make(map[string]bool)
	var out []string
	for _, u := range urls {
		norm := normalizeScanURL(u)
		if norm == "" {
			continue
		}
		origin := scanOrigin(norm)
		if existingURL[norm] || existingOrigin[origin] {
			continue
		}
		if seenURL[norm] || seenOrigin[origin] {
			continue
		}
		seenURL[norm] = true
		seenOrigin[origin] = true
		out = append(out, u)
	}
	return out
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
		origin := scanOrigin(t)
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
