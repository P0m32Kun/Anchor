package parser

import (
	"encoding/json"
	"io"
	"strings"
)

// DNSxResult represents a single DNS resolution result from dnsx JSONL output.
type DNSxResult struct {
	Host  string   `json:"host"`
	A     []string `json:"a"`
	AAAA  []string `json:"aaaa"`
	CNAME []string `json:"cname"`
	MX    []string `json:"mx"`
	NS    []string `json:"ns"`
	TXT   []string `json:"txt"`
	TTL   int      `json:"ttl"`
}

// ParseDNSx reads dnsx -json JSONL output and returns parsed DNS results.
func ParseDNSx(r io.Reader) ([]DNSxResult, []ParseError) {
	return parseJSONLines(r, func(line []byte, lineNo int) (DNSxResult, ParseError) {
		var rec DNSxResult
		if err := json.Unmarshal(line, &rec); err != nil {
			return rec, ParseError{Line: lineNo, Raw: string(line), Message: "invalid JSON: " + err.Error()}
		}
		if rec.Host == "" {
			return rec, ParseError{Line: lineNo, Raw: string(line), Message: "missing host field"}
		}
		return rec, ParseError{}
	})
}

// ParseDNSxOutput parses dnsx JSONL output into DNS records.
// Returns a map of domain -> (IPs, CNAMEs).
func ParseDNSxOutput(r io.Reader) map[string]DNSxResult {
	results, _ := ParseDNSx(r)
	m := make(map[string]DNSxResult)
	for _, res := range results {
		m[res.Host] = res
	}
	return m
}

// ExtractDNSxIPs extracts all unique IPs (A and AAAA) from dnsx results.
func ExtractDNSxIPs(rec DNSxResult) []string {
	seen := make(map[string]bool)
	var ips []string
	for _, ip := range rec.A {
		if ip != "" && !seen[ip] {
			seen[ip] = true
			ips = append(ips, ip)
		}
	}
	for _, ip := range rec.AAAA {
		if ip != "" && !seen[ip] {
			seen[ip] = true
			ips = append(ips, ip)
		}
	}
	return ips
}

// ExtractDNSxCNAMEs extracts all CNAMEs from dnsx results.
func ExtractDNSxCNAMEs(rec DNSxResult) []string {
	var cnames []string
	for _, c := range rec.CNAME {
		if c != "" {
			cnames = append(cnames, c)
		}
	}
	return cnames
}

// JoinDNSxIPs joins all IPs from a dnsx result into a comma-separated string.
func JoinDNSxIPs(rec DNSxResult) string {
	return strings.Join(ExtractDNSxIPs(rec), ",")
}
