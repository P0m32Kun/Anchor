package parser

import (
	"encoding/json"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
)

// ParseHTTPX reads httpx JSONL output and returns discovered web endpoints.
// httpx uses hyphenated keys like "status-code".
func ParseHTTPX(r io.Reader) ([]HTTPXResult, []ParseError) {
	return parseJSONLines(r, func(line []byte, lineNo int) (HTTPXResult, ParseError) {
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(line, &raw); err != nil {
			return HTTPXResult{}, ParseError{Line: lineNo, Raw: string(line), Message: "invalid JSON: " + err.Error()}
		}

		urlBytes, ok := raw["url"]
		if !ok {
			return HTTPXResult{}, ParseError{Line: lineNo, Raw: string(line), Message: "missing url field"}
		}

		var u string
		if err := json.Unmarshal(urlBytes, &u); err != nil {
			return HTTPXResult{}, ParseError{Line: lineNo, Raw: string(line), Message: "invalid url field: " + err.Error()}
		}
		if u == "" {
			return HTTPXResult{}, ParseError{Line: lineNo, Raw: string(line), Message: "empty url field"}
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

		return res, ParseError{}
	})
}

// ParseHttpxOutput parses httpx -json JSONL output into WebEndpoint models.
func ParseHttpxOutput(r io.Reader) []*models.WebEndpoint {
	results, _ := ParseHTTPX(r)
	var endpoints []*models.WebEndpoint
	for _, res := range results {
		if res.URL == "" {
			continue
		}
		endpoint := &models.WebEndpoint{
			URL:          res.URL,
			Scheme:       res.Scheme,
			Host:         res.Host,
			Title:        res.Title,
			Technologies: res.Tech,
			SourceTool:   "httpx",
			CreatedAt:    time.Now().UTC(),
		}
		if res.StatusCode > 0 {
			endpoint.StatusCode = &res.StatusCode
		}
		endpoints = append(endpoints, endpoint)
	}
	return endpoints
}
