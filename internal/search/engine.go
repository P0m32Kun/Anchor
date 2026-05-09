package search

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

var defaultHTTPClient = &http.Client{Timeout: 30 * time.Second}

// Engine defines a common interface for all internet search engines.
type Engine interface {
	Search(ctx context.Context, query string, page, size int) ([]SearchResult, error)
}

// baseClient holds the shared HTTP client used by all search engine clients.
type baseClient struct {
	client *http.Client
}

// doRequest performs the HTTP request and returns the response, validating the status code.
func (b *baseClient) doRequest(req *http.Request) (*http.Response, error) {
	resp, err := b.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}
	return resp, nil
}

// doJSON performs the HTTP request and decodes the JSON response into the provided value.
func (b *baseClient) doJSON(req *http.Request, v any) error {
	resp, err := b.doRequest(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

// QuotaPoint represents a single quota/credit metric.
type QuotaPoint struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
	Unit  string `json:"unit"`
}

// QuotaInfo holds all quota/credit metrics for a search engine.
type QuotaInfo struct {
	Points []QuotaPoint `json:"points"`
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
