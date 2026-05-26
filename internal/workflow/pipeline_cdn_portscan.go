package workflow

// ipsForPortScan selects which resolved IPs enter alive/naabu after optional CDN classification.
// When classified is false, all IPs are scanned. When classified is true and SkipPortscanOnCDNHost
// is set, CDN IPs are excluded from port scan; when SkipPortscanOnCDNHost is false, CDN IPs are
// still port-scanned while CDN domains may still be routed to httpx via cdnDomains.
func ipsForPortScan(allIPs, nonCDNIPs []string, classified, skipPortscanOnCDNHost bool) []string {
	if !classified {
		return allIPs
	}
	if skipPortscanOnCDNHost {
		return nonCDNIPs
	}
	return allIPs
}
