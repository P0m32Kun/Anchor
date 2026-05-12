package parser

import (
	"encoding/json"
	"io"
)

// FfufResult represents a single match from ffuf directory scanning.
type FfufResult struct {
	URL        string            `json:"url"`
	StatusCode int               `json:"status"`
	Length     int               `json:"length"`
	Words      int               `json:"words"`
	Lines      int               `json:"lines"`
	Input      map[string]string `json:"input"`
}

// ParseFfufOutput parses JSONL output from ffuf.
func ParseFfufOutput(r io.Reader) ([]FfufResult, []ParseError) {
	return parseJSONLines(r, func(line []byte, lineNo int) (FfufResult, ParseError) {
		var res FfufResult
		if err := json.Unmarshal(line, &res); err != nil {
			return FfufResult{}, ParseError{Line: lineNo, Raw: string(line), Message: err.Error()}
		}
		return res, ParseError{}
	})
}
