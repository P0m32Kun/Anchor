package util

import (
	"net"
	"net/url"
	"strings"
)

// ScanOrigin returns host:port for a URL or host string, stripping paths so the
// same service is not scanned or deduplicated separately per path.
func ScanOrigin(raw string) string {
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
