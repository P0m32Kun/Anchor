package parser

import (
	"encoding/json"
	"io"
)

// ParseSubfinder reads Subfinder JSONL output and returns discovered subdomains.
func ParseSubfinder(r io.Reader) ([]SubfinderResult, []ParseError) {
	return parseJSONLines(r, func(line []byte, lineNo int) (SubfinderResult, ParseError) {
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(line, &raw); err != nil {
			return SubfinderResult{}, ParseError{Line: lineNo, Raw: string(line), Message: "invalid JSON: " + err.Error()}
		}

		hostBytes, ok := raw["host"]
		if !ok {
			return SubfinderResult{}, ParseError{Line: lineNo, Raw: string(line), Message: "missing host field"}
		}

		var host string
		if err := json.Unmarshal(hostBytes, &host); err != nil {
			return SubfinderResult{}, ParseError{Line: lineNo, Raw: string(line), Message: "invalid host field: " + err.Error()}
		}
		if host == "" {
			return SubfinderResult{}, ParseError{Line: lineNo, Raw: string(line), Message: "empty host field"}
		}

		var input string
		if b, ok := raw["input"]; ok {
			_ = json.Unmarshal(b, &input)
		}
		var source string
		if b, ok := raw["source"]; ok {
			_ = json.Unmarshal(b, &source)
		}

		return SubfinderResult{
			Host:   host,
			Input:  input,
			Source: source,
		}, ParseError{}
	})
}

// ParseSubfinderOutput parses subfinder -oJ JSONL output into a list of subdomains.
func ParseSubfinderOutput(r io.Reader) []string {
	results, _ := ParseSubfinder(r)
	seen := make(map[string]bool)
	var subs []string
	for _, res := range results {
		if res.Host != "" && !seen[res.Host] {
			seen[res.Host] = true
			subs = append(subs, res.Host)
		}
	}
	return subs
}
