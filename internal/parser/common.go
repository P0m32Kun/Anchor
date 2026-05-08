package parser

import (
	"bufio"
	"fmt"
	"io"
)

// ParseError records a single line parse failure.
type ParseError struct {
	Line    int    `json:"line"`
	Raw     string `json:"raw"`
	Message string `json:"message"`
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("line %d: %s (raw: %s)", e.Line, e.Message, e.Raw)
}

// SubfinderResult is the extracted data from a Subfinder JSONL line.
type SubfinderResult struct {
	Host   string `json:"host"`
	Input  string `json:"input"`
	Source string `json:"source"`
}

// HTTPXResult is the extracted data from an httpx JSONL line.
type HTTPXResult struct {
	URL        string `json:"url"`
	Input      string `json:"input"`
	Title      string `json:"title"`
	WebServer  string `json:"webserver"`
	StatusCode int    `json:"status_code"`
	Host       string `json:"host"`
	Port       string `json:"port"`
	Scheme     string `json:"scheme"`
	Path       string `json:"path"`
	Tech       []string
}

// NaabuResult is the extracted data from a Naabu JSONL line.
type NaabuResult struct {
	Host string `json:"host"`
	Port int    `json:"port"`
	IP   string `json:"ip"`
}

// PortInfo represents a single open port from naabu output.
type PortInfo struct {
	IP       string `json:"ip"`
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
}
