package parser

import (
	"bufio"
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
	var results []DNSxResult
	var errs []ParseError

	scanner := bufio.NewScanner(r)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := scanner.Text()
		if line == "" {
			continue
		}

		var rec DNSxResult
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			errs = append(errs, ParseError{Line: lineNo, Raw: line, Message: "invalid JSON: " + err.Error()})
			continue
		}

		if rec.Host == "" {
			errs = append(errs, ParseError{Line: lineNo, Raw: line, Message: "missing host field"})
			continue
		}

		results = append(results, rec)
	}

	if err := scanner.Err(); err != nil {
		errs = append(errs, ParseError{Line: lineNo, Raw: "", Message: "scanner error: " + err.Error()})
	}

	return results, errs
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
