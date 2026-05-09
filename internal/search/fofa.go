package search

import (
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// FofaClient is a client for the FOFA API.
type FofaClient struct {
	baseClient
	apiKey  string
	baseURL string
}

// FofaResult represents a single result from FOFA.
type FofaResult struct {
	Host     string `json:"host"`
	IP       string `json:"ip"`
	Port     int    `json:"port"`
	Title    string `json:"title"`
	Protocol string `json:"protocol"`
	Server   string `json:"server"`
	CName    string `json:"cname"`
}

// NewFofaClient creates a new FOFA client.
// The base URL defaults to https://fofa.info but can be overridden via the
// FOFA_BASE_URL environment variable (used by E2E tests to point at a mock).
func NewFofaClient(apiKey string) *FofaClient {
	baseURL := "https://fofa.info"
	if override := os.Getenv("FOFA_BASE_URL"); override != "" {
		baseURL = strings.TrimRight(override, "/")
	}
	return &FofaClient{
		baseClient: baseClient{client: defaultHTTPClient},
		apiKey:     apiKey,
		baseURL:    baseURL,
	}
}

// SearchCompany searches FOFA for assets associated with a company name.
// It combines org, cert, and title queries for better coverage.
func (c *FofaClient) SearchCompany(ctx context.Context, company string) ([]FofaResult, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("FOFA credentials not configured")
	}

	queries := []string{
		fmt.Sprintf(`org="%s"`, company),
		fmt.Sprintf(`cert="%s"`, company),
		fmt.Sprintf(`title="%s"`, company),
	}

	seen := make(map[string]bool)
	var allResults []FofaResult

	for _, q := range queries {
		results, err := c.search(ctx, q, 500)
		if err != nil {
			// Log but continue with other queries
			continue
		}
		for _, r := range results {
			key := r.Host + "|" + r.IP
			if !seen[key] {
				seen[key] = true
				allResults = append(allResults, r)
			}
		}
	}

	return allResults, nil
}

// SearchDomain searches FOFA for assets associated with a domain.
func (c *FofaClient) SearchDomain(ctx context.Context, domain string) ([]FofaResult, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("FOFA credentials not configured")
	}
	q := fmt.Sprintf(`domain="%s"`, domain)
	return c.search(ctx, q, 500)
}

// SearchIP searches FOFA for assets associated with an IP.
func (c *FofaClient) SearchIP(ctx context.Context, ip string) ([]FofaResult, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("FOFA credentials not configured")
	}
	q := fmt.Sprintf(`ip="%s"`, ip)
	return c.search(ctx, q, 500)
}

// Search performs a direct FOFA search with the given raw query.
// The query is passed directly to FOFA without any wrapping.
func (c *FofaClient) Search(ctx context.Context, query string, page, size int) ([]SearchResult, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("FOFA credentials not configured")
	}
	if size < 1 {
		size = 20
	}
	if size > 500 {
		size = 500
	}
	fofaResults, err := c.search(ctx, query, size)
	if err != nil {
		return nil, err
	}
	results := make([]SearchResult, 0, len(fofaResults))
	for _, fr := range fofaResults {
		results = append(results, SearchResult{
			Engine:   "fofa",
			IP:       fr.IP,
			Port:     fr.Port,
			Domain:   fr.Host,
			Title:    fr.Title,
			Protocol: fr.Protocol,
			Service:  fr.Server,
		})
	}
	return results, nil
}

func (c *FofaClient) search(ctx context.Context, query string, size int) ([]FofaResult, error) {
	qbase64 := base64.StdEncoding.EncodeToString([]byte(query))

	u, _ := url.Parse(c.baseURL + "/api/v1/search/all")
	q := u.Query()
	q.Set("key", c.apiKey)
	q.Set("qbase64", qbase64)
	q.Set("size", strconv.Itoa(size))
	q.Set("fields", "host,ip,port,title,protocol,server")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	var result struct {
		Error   bool         `json:"error"`
		ErrMsg  string       `json:"errmsg"`
		Size    int          `json:"size"`
		Results [][]string   `json:"results"`
	}

	if err := c.doJSON(req, &result); err != nil {
		return nil, err
	}

	if result.Error {
		return nil, fmt.Errorf("fofa API error: %s", result.ErrMsg)
	}

	return parseFofaResults(result.Results), nil
}

func parseFofaResults(raw [][]string) []FofaResult {
	var results []FofaResult
	for _, row := range raw {
		if len(row) < 2 {
			continue
		}
		r := FofaResult{
			Host: row[0],
			IP:   row[1],
		}
		if len(row) > 2 && row[2] != "" {
			if port, err := strconv.Atoi(row[2]); err == nil {
				r.Port = port
			}
		}
		if len(row) > 3 {
			r.Title = row[3]
		}
		if len(row) > 4 {
			r.Protocol = row[4]
		}
		if len(row) > 5 {
			r.Server = row[5]
		}
		results = append(results, r)
	}
	return results
}

// GetQuota returns the remaining FOFA quota (F-points and API query count).
func (c *FofaClient) GetQuota(ctx context.Context) (*QuotaInfo, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("FOFA credentials not configured")
	}

	u, _ := url.Parse(c.baseURL + "/api/v1/info")
	q := u.Query()
	q.Set("key", c.apiKey)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	var result struct {
		Error        bool   `json:"error"`
		ErrMsg       string `json:"errmsg"`
		Fcoin        int    `json:"fcoin"`
		RemainQuery  int    `json:"remain_api_query"`
	}

	if err := c.doJSON(req, &result); err != nil {
		return nil, err
	}

	if result.Error {
		return nil, fmt.Errorf("fofa API error: %s", result.ErrMsg)
	}

	return &QuotaInfo{
		Remain: result.Fcoin,
		Total:  result.RemainQuery,
		Unit:   "F点",
	}, nil
}

// IsDomain checks if a host string looks like a domain.
func IsDomain(host string) bool {
	host = strings.TrimSpace(host)
	if host == "" {
		return false
	}
	// If it contains a colon, it's likely ip:port
	if strings.Contains(host, ":") {
		return false
	}
	// If it's a pure IP, it's not a domain
	if net.ParseIP(host) != nil {
		return false
	}
	// Must contain at least one dot
	return strings.Contains(host, ".")
}

