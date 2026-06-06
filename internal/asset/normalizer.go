package asset

import (
	"net"
	"net/url"
	"strconv"
	"strings"
)

// NormalizeDomain lowercases, trims space, and strips leading www.
func NormalizeDomain(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	v = strings.TrimPrefix(v, "www.")
	return v
}

// NormalizeURL parses, lowercases scheme/host, strips default port and trailing slash.
func NormalizeURL(v string) string {
	u, err := url.Parse(strings.TrimSpace(v))
	if err != nil {
		return strings.ToLower(strings.TrimSpace(v))
	}
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)
	if host, port, err := net.SplitHostPort(u.Host); err == nil {
		defPort := defaultPort(u.Scheme)
		host = strings.TrimPrefix(host, "www.")
		if port == defPort {
			u.Host = host
		} else {
			u.Host = net.JoinHostPort(host, port)
		}
	} else {
		u.Host = strings.TrimPrefix(u.Host, "www.")
	}
	u.Path = strings.TrimSuffix(u.Path, "/")
	if u.Path == "" {
		u.Path = "/"
	}
	u.RawQuery = ""
	u.Fragment = ""
	return u.String()
}

// NormalizeIP parses and returns canonical IP representation (strips leading zeros in IPv4).
func NormalizeIP(v string) string {
	v = strings.TrimSpace(v)
	ip := net.ParseIP(v)
	if ip == nil {
		// Go 1.17+ rejects leading zeros in IPv4; strip them and retry.
		ip = net.ParseIP(stripIPv4LeadingZeros(v))
	}
	if ip == nil {
		return v
	}
	if ip4 := ip.To4(); ip4 != nil {
		return ip4.String()
	}
	return ip.String()
}

// NormalizeCIDR parses and returns canonical CIDR representation.
func NormalizeCIDR(v string) string {
	v = strings.TrimSpace(v)
	ip, ipNet, err := net.ParseCIDR(v)
	if err != nil {
		return v
	}
	// Use the network address as the canonical form
	_ = ip
	return ipNet.String()
}

// stripIPv4LeadingZeros removes leading zeros from each IPv4 octet.
func stripIPv4LeadingZeros(s string) string {
	parts := strings.Split(s, ".")
	if len(parts) != 4 {
		return s
	}
	for i := range parts {
		parts[i] = strconv.Itoa(atoiOrZero(parts[i]))
	}
	return strings.Join(parts, ".")
}

func atoiOrZero(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

// InferStorageType detects the storage type from a raw value.
// IP/CIDR/URL literals must not be stored as domains (NormalizeDomain would alias them).
func InferStorageType(value string) string {
	v := strings.TrimSpace(value)
	if v == "" {
		return ""
	}
	if strings.HasPrefix(v, "http://") || strings.HasPrefix(v, "https://") {
		return "url"
	}
	if _, _, err := net.ParseCIDR(v); err == nil {
		return "cidr"
	}
	if host, _, err := net.SplitHostPort(v); err == nil {
		if net.ParseIP(host) != nil {
			return "ip"
		}
		return "domain"
	}
	if net.ParseIP(v) != nil {
		return "ip"
	}
	return "domain"
}

// Normalize chooses the appropriate normalizer based on asset type.
func Normalize(assetType, value string) string {
	if inferred := InferStorageType(value); inferred == "ip" || inferred == "cidr" || inferred == "url" {
		assetType = inferred
	}
	switch assetType {
	case "domain":
		return NormalizeDomain(value)
	case "url":
		return NormalizeURL(value)
	case "ip":
		return NormalizeIP(value)
	case "cidr":
		return NormalizeCIDR(value)
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func defaultPort(scheme string) string {
	switch scheme {
	case "http":
		return "80"
	case "https":
		return "443"
	case "ftp":
		return "21"
	case "ssh":
		return "22"
	default:
		return ""
	}
}

// ExtractHostFromURL returns the host part of a URL string.
func ExtractHostFromURL(v string) string {
	u, err := url.Parse(strings.TrimSpace(v))
	if err != nil {
		return ""
	}
	host := strings.ToLower(u.Host)
	if h, _, err := net.SplitHostPort(host); err == nil {
		return h
	}
	return host
}

// ExtractPortFromURL returns the explicit port from a URL string, or 0 if default/unspecified.
func ExtractPortFromURL(v string) int {
	u, err := url.Parse(strings.TrimSpace(v))
	if err != nil {
		return 0
	}
	if _, port, err := net.SplitHostPort(u.Host); err == nil {
		if p, err := strconv.Atoi(port); err == nil {
			return p
		}
	}
	return 0
}
