package search

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// HunterClient is a client for the Qianxin Hunter API.
type HunterClient struct {
	baseClient
	apiKey  string
	baseURL string
}

// HunterResult represents a single result from Hunter.
type HunterResult struct {
	IP         string `json:"ip"`
	Port       int    `json:"port"`
	Domain     string `json:"domain"`
	WebTitle   string `json:"web_title"`
	WebServer  string `json:"web_server"`
	StatusCode int    `json:"status_code"`
	Protocol   string `json:"protocol"`
	Component  string `json:"component"`
	OS         string `json:"os"`
	Company    string `json:"company"`
	ICP        string `json:"icp"`
	Banner     string `json:"banner"`
	IsRisk     string `json:"is_risk"`
}

// NewHunterClient creates a new Hunter client.
func NewHunterClient(apiKey string) *HunterClient {
	return &HunterClient{
		apiKey:  apiKey,
		baseURL: "https://hunter.qianxin.com",
		client:  defaultHTTPClient,
	}
}

// Search performs a Hunter search with the given query.
func (c *HunterClient) Search(ctx context.Context, query string, page, pageSize int) ([]SearchResult, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("Hunter API key not configured")
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	u, _ := url.Parse(c.baseURL + "/openApi/search")
	q := u.Query()
	q.Set("search", query)
	q.Set("page", strconv.Itoa(page))
	q.Set("page_size", strconv.Itoa(pageSize))
	q.Set("api-key", c.apiKey)
	q.Set("is_web", "3") // 1=web, 2=non-web, 3=all
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Hunter API returned status %d", resp.StatusCode)
	}

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Total int             `json:"total"`
			Arr   []*HunterResult `json:"arr"`
			Rest  string          `json:"rest"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if result.Code != 200 {
		return nil, fmt.Errorf("Hunter API error: %s", result.Msg)
	}

	return convertHunterResults(result.Data.Arr), nil
}

func convertHunterResults(raw []*HunterResult) []SearchResult {
	results := make([]SearchResult, 0, len(raw))
	for _, r := range raw {
		if r == nil {
			continue
		}
		results = append(results, SearchResult{
			Engine:   "hunter",
			IP:       r.IP,
			Port:     r.Port,
			Domain:   r.Domain,
			Title:    r.WebTitle,
			Service:  r.WebServer,
			Protocol: r.Protocol,
			OS:       r.OS,
			Raw:      r,
		})
	}
	return results
}
