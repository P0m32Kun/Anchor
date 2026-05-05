package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// QuakeClient is a client for the 360 Quake API.
type QuakeClient struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// QuakeResult represents a single result from Quake.
type QuakeResult struct {
	IP       string `json:"ip"`
	Port     int    `json:"port"`
	Service  struct {
		Name    string `json:"name"`
		Version string `json:"version"`
		Banner  string `json:"banner"`
	} `json:"service"`
	Location struct {
		Country string `json:"country_cn"`
		City    string `json:"city_cn"`
		Lat     string `json:"lat"`
		Lon     string `json:"lon"`
	} `json:"location"`
	ASN      struct {
		Number       string `json:"number"`
		Organization string `json:"organization"`
	} `json:"asn"`
	Org      string `json:"org"`
	Hostname string `json:"hostname"`
	Domain   string `json:"domain"`
	OS       string `json:"os_name"`
}

// NewQuakeClient creates a new Quake client.
func NewQuakeClient(apiKey string) *QuakeClient {
	return &QuakeClient{
		apiKey:  apiKey,
		baseURL: "https://quake.360.net",
		client:  defaultHTTPClient,
	}
}

// Search performs a Quake search with the given query.
func (c *QuakeClient) Search(ctx context.Context, query string, start, size int) ([]SearchResult, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("Quake API key not configured")
	}
	if start < 0 {
		start = 0
	}
	if size < 1 {
		size = 10
	}
	if size > 100 {
		size = 100
	}

	body, _ := json.Marshal(map[string]interface{}{
		"query": query,
		"start": start,
		"size":  size,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v3/search/quake_service", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-QuakeToken", c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Quake API returned status %d", resp.StatusCode)
	}

	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Total   int    `json:"total_count"`
		Data    []*QuakeResult `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("Quake API error: %s", result.Message)
	}

	return convertQuakeResults(result.Data), nil
}

func convertQuakeResults(raw []*QuakeResult) []SearchResult {
	results := make([]SearchResult, 0, len(raw))
	for _, r := range raw {
		if r == nil {
			continue
		}
		location := r.Location.Country
		if r.Location.City != "" {
			location = location + " " + r.Location.City
		}
		results = append(results, SearchResult{
			Engine:   "quake",
			IP:       r.IP,
			Port:     r.Port,
			Domain:   r.Domain,
			Title:    r.Hostname,
			Service:  r.Service.Name,
			Protocol: r.Service.Name,
			Location: location,
			OS:       r.OS,
			Raw:      r,
		})
	}
	return results
}
