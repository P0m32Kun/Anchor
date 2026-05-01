package parser

import (
	"bufio"
	"encoding/json"
	"io"
	"strconv"
	"strings"
)

// ParseHTTPX reads httpx JSONL output and returns discovered web endpoints.
// httpx uses hyphenated keys like "status-code".
func ParseHTTPX(r io.Reader) ([]HTTPXResult, []ParseError) {
	var results []HTTPXResult
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

		urlBytes, ok := raw["url"]
		if !ok {
			errs = append(errs, ParseError{Line: lineNo, Raw: line, Message: "missing url field"})
			continue
		}

		var u string
		if err := json.Unmarshal(urlBytes, &u); err != nil {
			errs = append(errs, ParseError{Line: lineNo, Raw: line, Message: "invalid url field: " + err.Error()})
			continue
		}
		if u == "" {
			errs = append(errs, ParseError{Line: lineNo, Raw: line, Message: "empty url field"})
			continue
		}

		res := HTTPXResult{URL: u}

		if b, ok := raw["input"]; ok {
			_ = json.Unmarshal(b, &res.Input)
		}
		if b, ok := raw["title"]; ok {
			_ = json.Unmarshal(b, &res.Title)
		}
		if b, ok := raw["webserver"]; ok {
			_ = json.Unmarshal(b, &res.WebServer)
		}
		if b, ok := raw["host"]; ok {
			_ = json.Unmarshal(b, &res.Host)
		}
		if b, ok := raw["scheme"]; ok {
			_ = json.Unmarshal(b, &res.Scheme)
		}
		if b, ok := raw["path"]; ok {
			_ = json.Unmarshal(b, &res.Path)
		}
		if b, ok := raw["port"]; ok {
			var portStr string
			if err := json.Unmarshal(b, &portStr); err == nil {
				res.Port = portStr
			} else {
				var portInt int
				if err := json.Unmarshal(b, &portInt); err == nil {
					res.Port = strconv.Itoa(portInt)
				}
			}
		}
		if b, ok := raw["status-code"]; ok {
			var sc int
			if err := json.Unmarshal(b, &sc); err == nil {
				res.StatusCode = sc
			}
		}
		if b, ok := raw["tech"]; ok {
			var tech []string
			if err := json.Unmarshal(b, &tech); err == nil {
				res.Tech = tech
			}
		} else if b, ok := raw["technologies"]; ok {
			var tech []string
			if err := json.Unmarshal(b, &tech); err == nil {
				res.Tech = tech
			}
		}

		// Extract product names from CPE when tech is sparse (e.g. 404/302 pages).
		if b, ok := raw["cpe"]; ok {
			var cpeList []struct {
				Product string `json:"product"`
			}
			if err := json.Unmarshal(b, &cpeList); err == nil {
				seen := make(map[string]bool, len(res.Tech))
				for _, t := range res.Tech {
					seen[strings.ToLower(t)] = true
				}
				for _, c := range cpeList {
					if c.Product != "" && !seen[strings.ToLower(c.Product)] {
						res.Tech = append(res.Tech, c.Product)
						seen[strings.ToLower(c.Product)] = true
					}
			}
			}
		}

		results = append(results, res)
	}

	if err := scanner.Err(); err != nil {
		errs = append(errs, ParseError{Line: lineNo, Raw: "", Message: "scanner error: " + err.Error()})
	}

	return results, errs
}
