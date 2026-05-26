package parser

import (
	"encoding/json"
	"io"
	"strings"
)

// ParseKatanaJSONL extracts discovered endpoint URLs from katana -jsonl stdout.
// Each line is a JSON object; we prefer request.endpoint, then request.url.
func ParseKatanaJSONL(r io.Reader) ([]string, []ParseError) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, []ParseError{{Line: 0, Raw: "", Message: "read katana output: " + err.Error()}}
	}
	if len(data) == 0 {
		return nil, nil
	}

	seen := make(map[string]struct{})
	var urls []string
	var errs []ParseError
	lineNo := 0

	for _, line := range strings.Split(string(data), "\n") {
		lineNo++
		line = strings.TrimSpace(line)
		if line == "" || line[0] != '{' {
			continue
		}

		var row struct {
			Request struct {
				Endpoint string `json:"endpoint"`
				URL      string `json:"url"`
			} `json:"request"`
			URL string `json:"url"`
		}
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			errs = append(errs, ParseError{Line: lineNo, Raw: line, Message: "decode katana jsonl: " + err.Error()})
			continue
		}

		u := strings.TrimSpace(row.Request.Endpoint)
		if u == "" {
			u = strings.TrimSpace(row.Request.URL)
		}
		if u == "" {
			u = strings.TrimSpace(row.URL)
		}
		if u == "" {
			continue
		}
		if _, ok := seen[u]; ok {
			continue
		}
		seen[u] = struct{}{}
		urls = append(urls, u)
	}

	return urls, errs
}
