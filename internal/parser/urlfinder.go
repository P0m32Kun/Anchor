package parser

import (
	"encoding/json"
	"io"
)

// UrlfinderResult represents a single URL discovered by urlfinder.
type UrlfinderResult struct {
	URL    string `json:"url"`
	Source string `json:"source"`
}

// ParseUrlfinderOutput parses JSONL output from urlfinder.
func ParseUrlfinderOutput(r io.Reader) ([]UrlfinderResult, []ParseError) {
	return parseJSONLines(r, func(line []byte, lineNo int) (UrlfinderResult, ParseError) {
		var res UrlfinderResult
		if err := json.Unmarshal(line, &res); err != nil {
			return UrlfinderResult{}, ParseError{Line: lineNo, Raw: string(line), Message: err.Error()}
		}
		return res, ParseError{}
	})
}
