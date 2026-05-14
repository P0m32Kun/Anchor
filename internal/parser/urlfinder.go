package parser

import (
	"encoding/json"
	"io"
)

// URLFinderLink represents a single URL or JS link extracted by pingc0y/URLFinder.
type URLFinderLink struct {
	URL      string `json:"Url"`      // URLFinder uses capital-U "Url"
	Status   string `json:"Status"`
	Size     string `json:"Size"`
	Title    string `json:"Title"`
	Redirect string `json:"Redirect"`
	Source   string `json:"Source"`
}

// URLFinderOutput is the top-level JSON structure produced by URLFinder -o.
type URLFinderOutput struct {
	JS      []URLFinderLink `json:"js"`
	JSOther []URLFinderLink `json:"jsOther"`
	URL     []URLFinderLink `json:"url"`
	URLOther []URLFinderLink `json:"urlOther"`
	Domain  []string        `json:"domain"`
}

// URLFinderResult is a flattened result combining URLs and JS links.
type URLFinderResult struct {
	URL    string `json:"url"`
	Type   string `json:"type"` // "url" or "js"
	Source string `json:"source"`
}

// ParseURLFinderOutput parses the JSON output from pingc0y/URLFinder.
// The output is a single JSON object (not JSONL).
func ParseURLFinderOutput(r io.Reader) ([]URLFinderResult, []ParseError) {
	var output URLFinderOutput
	if err := json.NewDecoder(r).Decode(&output); err != nil {
		return nil, []ParseError{{Line: 1, Raw: "", Message: "decode URLFinder JSON: " + err.Error()}}
	}

	var results []URLFinderResult
	for _, link := range output.URL {
		if link.URL != "" {
			results = append(results, URLFinderResult{URL: link.URL, Type: "url", Source: link.Source})
		}
	}
	for _, link := range output.URLOther {
		if link.URL != "" {
			results = append(results, URLFinderResult{URL: link.URL, Type: "url", Source: link.Source})
		}
	}
	for _, link := range output.JS {
		if link.URL != "" {
			results = append(results, URLFinderResult{URL: link.URL, Type: "js", Source: link.Source})
		}
	}
	for _, link := range output.JSOther {
		if link.URL != "" {
			results = append(results, URLFinderResult{URL: link.URL, Type: "js", Source: link.Source})
		}
	}

	return results, nil
}
