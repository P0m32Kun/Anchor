package parser

import (
	"encoding/json"
	"io"
)

// NucleiInfo holds the nested info object in Nuclei JSONL output.
type NucleiInfo struct {
	Name     string   `json:"name"`
	Severity string   `json:"severity"`
	Tags     []string `json:"tags"`
}

// NucleiResult holds the extracted fields from a Nuclei JSONL line.
type NucleiResult struct {
	TemplateID       string   `json:"template-id"`
	TemplatePath     string   `json:"template-path"`
	Name             string   `json:"-"`
	Severity         string   `json:"-"`
	Tags             []string `json:"-"`
	Host             string   `json:"host"`
	MatchedAt        string   `json:"matched-at"`
	MatcherName      string   `json:"matcher-name"`
	ExtractedResults []string `json:"extracted-results"`
	Request          string   `json:"request"`
	Response         string   `json:"response"`
	Timestamp        string   `json:"timestamp"`
	RawLine          string   `json:"-"` // Original JSON line, populated by parser
}

// ParseNuclei reads Nuclei JSONL output and returns parsed results.
func ParseNuclei(r io.Reader) ([]NucleiResult, []ParseError) {
	return parseJSONLines(r, func(line []byte, lineNo int) (NucleiResult, ParseError) {
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(line, &raw); err != nil {
			return NucleiResult{}, ParseError{Line: lineNo, Raw: string(line), Message: "invalid JSON: " + err.Error()}
		}

		var res NucleiResult
		if b, ok := raw["template-id"]; ok {
			_ = json.Unmarshal(b, &res.TemplateID)
		}
		if b, ok := raw["template-path"]; ok {
			_ = json.Unmarshal(b, &res.TemplatePath)
		}
		if b, ok := raw["host"]; ok {
			_ = json.Unmarshal(b, &res.Host)
		}
		if b, ok := raw["matched-at"]; ok {
			_ = json.Unmarshal(b, &res.MatchedAt)
		}
		if b, ok := raw["matcher-name"]; ok {
			_ = json.Unmarshal(b, &res.MatcherName)
		}
		if b, ok := raw["extracted-results"]; ok {
			_ = json.Unmarshal(b, &res.ExtractedResults)
		}
		if b, ok := raw["request"]; ok {
			_ = json.Unmarshal(b, &res.Request)
		}
		if b, ok := raw["response"]; ok {
			_ = json.Unmarshal(b, &res.Response)
		}
		if b, ok := raw["timestamp"]; ok {
			_ = json.Unmarshal(b, &res.Timestamp)
		}

		// Parse nested info object.
		if b, ok := raw["info"]; ok {
			var info NucleiInfo
			if err := json.Unmarshal(b, &info); err == nil {
				res.Name = info.Name
				res.Severity = info.Severity
				res.Tags = info.Tags
			}
		}

		return res, ParseError{}
	})
}
