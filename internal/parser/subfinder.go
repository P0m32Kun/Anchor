package parser

import (
	"bufio"
	"encoding/json"
	"io"
)

// ParseSubfinder reads Subfinder JSONL output and returns discovered subdomains.
func ParseSubfinder(r io.Reader) ([]SubfinderResult, []ParseError) {
	var results []SubfinderResult
	var errs []ParseError

	scanner := bufio.NewScanner(r)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := scanner.Text()
		if line == "" {
			continue
		}

		var raw map[string]json.RawMessage
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			errs = append(errs, ParseError{Line: lineNo, Raw: line, Message: "invalid JSON: " + err.Error()})
			continue
		}

		hostBytes, ok := raw["host"]
		if !ok {
			errs = append(errs, ParseError{Line: lineNo, Raw: line, Message: "missing host field"})
			continue
		}

		var host string
		if err := json.Unmarshal(hostBytes, &host); err != nil {
			errs = append(errs, ParseError{Line: lineNo, Raw: line, Message: "invalid host field: " + err.Error()})
			continue
		}
		if host == "" {
			errs = append(errs, ParseError{Line: lineNo, Raw: line, Message: "empty host field"})
			continue
		}

		var input string
		if b, ok := raw["input"]; ok {
			_ = json.Unmarshal(b, &input)
		}
		var source string
		if b, ok := raw["source"]; ok {
			_ = json.Unmarshal(b, &source)
		}

		results = append(results, SubfinderResult{
			Host:   host,
			Input:  input,
			Source: source,
		})
	}

	if err := scanner.Err(); err != nil {
		errs = append(errs, ParseError{Line: lineNo, Raw: "", Message: "scanner error: " + err.Error()})
	}

	return results, errs
}
