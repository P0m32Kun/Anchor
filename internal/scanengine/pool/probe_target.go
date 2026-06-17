package pool

import (
	"fmt"
	"net"
	"strings"

	"github.com/P0m32Kun/Anchor/internal/scanengine/core"
)

// ProbeTargetFromAsset returns the httpx probe string for an asset value/type.
func ProbeTargetFromAsset(value string, assetType core.AssetType) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	switch assetType {
	case core.AssetSubdomain:
		return strings.ToLower(value)
	case core.AssetIP, core.AssetCIDR:
		return value
	case core.AssetIPPort:
		if host, port, err := net.SplitHostPort(value); err == nil {
			return fmt.Sprintf("%s:%s", host, port)
		}
		return value
	case core.AssetHTTPService, core.AssetHTTPPath:
		return value
	default:
		return strings.ToLower(value)
	}
}

// ParseHostPort splits "host:port" into host and port. Port 0 if absent.
func ParseHostPort(value string) (host string, port int) {
	value = strings.TrimSpace(value)
	if h, p, err := net.SplitHostPort(value); err == nil {
		host = h
		fmt.Sscanf(p, "%d", &port)
		return host, port
	}
	return value, 0
}
