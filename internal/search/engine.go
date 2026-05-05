package search

import (
	"context"
	"net/http"
	"time"
)

var defaultHTTPClient = &http.Client{Timeout: 30 * time.Second}

// Engine defines a common interface for all internet search engines.
type Engine interface {
	Search(ctx context.Context, query string, page, size int) ([]SearchResult, error)
}

// SearchResult is a unified result format across all engines.
type SearchResult struct {
	Engine   string `json:"engine"`
	IP       string `json:"ip"`
	Port     int    `json:"port,omitempty"`
	Domain   string `json:"domain,omitempty"`
	Title    string `json:"title,omitempty"`
	Service  string `json:"service,omitempty"`
	Protocol string `json:"protocol,omitempty"`
	Location string `json:"location,omitempty"`
	OS       string `json:"os,omitempty"`
	Raw      any    `json:"raw,omitempty"`
}
