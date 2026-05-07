package parser

import (
	"bufio"
	"io"
	"strings"
)

// ParseNmapAlive parses nmap -oG (greppable) output and returns the IPs
// reported with "Status: Up". Lines we expect look like:
//
//	Host: 172.30.0.10 (rangefield-target-1)	Status: Up
//	Host: 172.30.0.5 ()	Status: Down
//
// Anything that isn't a "Host:" line, or that has any status other than Up,
// is silently skipped.
func ParseNmapAlive(r io.Reader) []string {
	var ips []string
	seen := make(map[string]struct{})

	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "Host:") {
			continue
		}
		if !strings.Contains(line, "Status: Up") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		ip := fields[1]
		if _, dup := seen[ip]; dup {
			continue
		}
		seen[ip] = struct{}{}
		ips = append(ips, ip)
	}
	return ips
}
