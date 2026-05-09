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
	baseClient
	apiKey  string
	baseURL string
}

// QuakeComponent represents a detected component in a Quake result.
type QuakeComponent struct {
	ProductNameCN string   `json:"product_name_cn"`
	ProductNameEN string   `json:"product_name_en"`
	Version       string   `json:"version"`
	ProductType   []string `json:"product_type"`
}

// QuakeHTTP represents HTTP-specific info within a Quake service result.
type QuakeHTTP struct {
	Title        string `json:"title"`
	Server       string `json:"server"`
	Host         string `json:"host"`
	StatusCode   int    `json:"status_code"`
	ResponseHeaders string `json:"response_headers"`
}

// QuakeResult represents a single result from Quake.
type QuakeResult struct {
	IP         string `json:"ip"`
	Port       int    `json:"port"`
	Service    struct {
		Name     string    `json:"name"`
		Version  string    `json:"version"`
		Response string    `json:"response"`
		HTTP     QuakeHTTP `json:"http"`
	} `json:"service"`
	Location   struct {
		Country  string `json:"country_cn"`
		City     string `json:"city_cn"`
		Province string `json:"province_cn"`
		ISP      string `json:"isp"`
	} `json:"location"`
	ASN        int    `json:"asn"`
	Org        string `json:"org"`
	Hostname   string `json:"hostname"`
	Domain     string `json:"domain"`
	OS         string `json:"os_name"`
	Transport  string `json:"transport"`
	Time       string `json:"time"`
	Components []QuakeComponent `json:"components"`
}

// NewQuakeClient creates a new Quake client.
func NewQuakeClient(apiKey string) *QuakeClient {
	return &QuakeClient{
		baseClient: baseClient{client: defaultHTTPClient},
		apiKey:     apiKey,
		baseURL:    "https://quake.360.net",
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

	var result struct {
		Code    int            `json:"code"`
		Message string         `json:"message"`
		Total   int            `json:"total_count"`
		Data    []*QuakeResult `json:"data"`
	}

	if err := c.doJSON(req, &result); err != nil {
		return nil, err
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("Quake API error: %s", result.Message)
	}

	return convertQuakeResults(result.Data), nil
}

// GetQuota returns the remaining Quake quota.
func (c *QuakeClient) GetQuota(ctx context.Context) (*QuotaInfo, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("Quake API key not configured")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v3/user/info", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("X-QuakeToken", c.apiKey)

	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			MonthRemainingCredit int `json:"month_remaining_credit"`
			TotalRemainingCredit int `json:"total_remaining_credit"`
		} `json:"data"`
	}

	if err := c.doJSON(req, &result); err != nil {
		return nil, err
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("Quake API error: %s", result.Message)
	}

	return &QuotaInfo{
		Points: []QuotaPoint{
			{Name: "月度积分", Value: result.Data.MonthRemainingCredit, Unit: ""},
			{Name: "长效积分", Value: result.Data.TotalRemainingCredit, Unit: ""},
		},
	}, nil
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

		// Title: 优先取 HTTP title，回退到 hostname
		title := r.Service.HTTP.Title
		if title == "" {
			title = r.Hostname
		}

		// 服务指纹：优先 HTTP server，其次 service.name + version，最后 banner
		service := r.Service.HTTP.Server
		if service == "" {
			service = r.Service.Name
			if r.Service.Version != "" {
				service += " " + r.Service.Version
			}
		}
		if service == "" && r.Service.Banner != "" {
			service = r.Service.Banner
		}

		results = append(results, SearchResult{
			Engine:       "quake",
			IP:           r.IP,
			Port:         r.Port,
			Domain:       r.Domain,
			Title:        title,
			Service:      service,
			Protocol:     r.Service.Name,
			Location:     location,
			OS:           r.OS,
			StatusCode:   r.Service.HTTP.StatusCode,
			Organization: r.Org,
			Raw:          r,
		})
	}
	return results
}
