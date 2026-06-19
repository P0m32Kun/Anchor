package search

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// engine.go — doRequest / doJSON
// ---------------------------------------------------------------------------

func TestDoRequest_Non200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	bc := baseClient{client: srv.Client()}
	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	_, err := bc.doRequest(req)
	if err == nil {
		t.Fatal("expected error for non-200 status")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Fatalf("error should mention status code, got: %s", err.Error())
	}
}

func TestDoRequest_NetworkError(t *testing.T) {
	bc := baseClient{client: &http.Client{}}
	req, _ := http.NewRequest(http.MethodGet, "http://127.0.0.1:1", nil)
	_, err := bc.doRequest(req)
	if err == nil {
		t.Fatal("expected error for unreachable host")
	}
}

func TestDoJSON_DecodeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`not json`))
	}))
	defer srv.Close()

	bc := baseClient{client: srv.Client()}
	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	var out struct{ X string }
	err := bc.doJSON(req, &out)
	if err == nil {
		t.Fatal("expected decode error")
	}
	if !strings.Contains(err.Error(), "decode response") {
		t.Fatalf("unexpected error: %s", err.Error())
	}
}

func TestDoJSON_Non200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	bc := baseClient{client: srv.Client()}
	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	var out struct{}
	err := bc.doJSON(req, &out)
	if err == nil {
		t.Fatal("expected error for 500")
	}
}

func TestDoJSON_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"hello": "world"})
	}))
	defer srv.Close()

	bc := baseClient{client: srv.Client()}
	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	var out map[string]string
	if err := bc.doJSON(req, &out); err != nil {
		t.Fatal(err)
	}
	if out["hello"] != "world" {
		t.Fatalf("expected hello=world, got %v", out)
	}
}

// ---------------------------------------------------------------------------
// fofa.go — IsDomain
// ---------------------------------------------------------------------------

func TestIsDomain(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"example.com", true},
		{"sub.example.com", true},
		{"a.b.c.d", true},
		{"", false},
		{"  ", false},
		{"192.168.1.1", false},
		{"10.0.0.1", false},
		{"::1", false},
		{"192.168.1.1:8080", false},
		{"example", false},
		{"localhost", false},
		{"[::1]:8080", false},
	}
	for _, tc := range tests {
		got := IsDomain(tc.input)
		if got != tc.want {
			t.Errorf("IsDomain(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// fofa.go — parseFofaResults
// ---------------------------------------------------------------------------

func TestParseFofaResults_Empty(t *testing.T) {
	results := parseFofaResults(nil)
	if results != nil {
		t.Fatalf("expected nil, got %v", results)
	}
}

func TestParseFofaResults_ShortRow(t *testing.T) {
	// A row with fewer than 2 elements should be skipped
	raw := [][]string{{"only-one"}}
	results := parseFofaResults(raw)
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestParseFofaResults_FullRow(t *testing.T) {
	raw := [][]string{{
		"host.example", "1.2.3.4", "443", "Test Title", "https", "nginx",
		"200", "China", "Beijing", "Aliyun", "ICP123",
	}}
	results := parseFofaResults(raw)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.Host != "host.example" || r.IP != "1.2.3.4" || r.Port != 443 {
		t.Fatalf("unexpected basic fields: %+v", r)
	}
	if r.Title != "Test Title" || r.Protocol != "https" || r.Server != "nginx" {
		t.Fatalf("unexpected service fields: %+v", r)
	}
	if r.StatusCode != 200 || r.Country != "China" || r.City != "Beijing" {
		t.Fatalf("unexpected geo fields: %+v", r)
	}
	if r.ASOrganization != "Aliyun" || r.ICP != "ICP123" {
		t.Fatalf("unexpected org fields: %+v", r)
	}
}

func TestParseFofaResults_MinimalRow(t *testing.T) {
	raw := [][]string{{"host.example", "1.2.3.4"}}
	results := parseFofaResults(raw)
	if len(results) != 1 {
		t.Fatalf("expected 1, got %d", len(results))
	}
	if results[0].Port != 0 || results[0].StatusCode != 0 {
		t.Fatalf("expected zero-valued port/status, got port=%d status=%d", results[0].Port, results[0].StatusCode)
	}
}

func TestParseFofaResults_InvalidPort(t *testing.T) {
	raw := [][]string{{"host.example", "1.2.3.4", "notanumber"}}
	results := parseFofaResults(raw)
	if len(results) != 1 {
		t.Fatalf("expected 1, got %d", len(results))
	}
	if results[0].Port != 0 {
		t.Fatalf("expected port=0 for invalid, got %d", results[0].Port)
	}
}

func TestParseFofaResults_InvalidStatusCode(t *testing.T) {
	raw := [][]string{{"h", "1.2.3.4", "80", "t", "http", "s", "abc"}}
	results := parseFofaResults(raw)
	if len(results) != 1 {
		t.Fatalf("expected 1, got %d", len(results))
	}
	if results[0].StatusCode != 0 {
		t.Fatalf("expected status=0 for invalid, got %d", results[0].StatusCode)
	}
}

func TestParseFofaResults_EmptyPortString(t *testing.T) {
	// Empty port string should remain 0, not error
	raw := [][]string{{"h", "1.2.3.4", ""}}
	results := parseFofaResults(raw)
	if len(results) != 1 || results[0].Port != 0 {
		t.Fatalf("expected port=0 for empty string, got %d", results[0].Port)
	}
}

func TestParseFofaResults_MultipleRows(t *testing.T) {
	raw := [][]string{
		{"a.com", "1.1.1.1"},
		{"b.com", "2.2.2.2"},
	}
	results := parseFofaResults(raw)
	if len(results) != 2 {
		t.Fatalf("expected 2, got %d", len(results))
	}
	if results[0].Host != "a.com" || results[1].Host != "b.com" {
		t.Fatalf("unexpected hosts: %s, %s", results[0].Host, results[1].Host)
	}
}

// ---------------------------------------------------------------------------
// fofa.go — NewFofaClient
// ---------------------------------------------------------------------------

func TestNewFofaClient_DefaultBaseURL(t *testing.T) {
	c := NewFofaClient("key")
	if c.baseURL != "https://fofa.info" {
		t.Fatalf("expected default baseURL, got %s", c.baseURL)
	}
}

func TestNewFofaClient_EnvOverride(t *testing.T) {
	t.Setenv("FOFA_BASE_URL", "http://mock-fofa:9999/")
	c := NewFofaClient("key")
	if c.baseURL != "http://mock-fofa:9999" {
		t.Fatalf("expected trimmed override, got %s", c.baseURL)
	}
}

// ---------------------------------------------------------------------------
// fofa.go — SearchDomain, SearchIP, Search, SearchCompany, GetQuota
// ---------------------------------------------------------------------------

func fofaMockServer(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}

func TestFofa_SearchDomain_Success(t *testing.T) {
	srv := fofaMockServer(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": false,
			"results": [][]string{
				{"a.example.com", "1.1.1.1", "80", "A", "http", "nginx"},
				{"b.example.com", "2.2.2.2", "443", "B", "https", "apache"},
			},
		})
	})
	defer srv.Close()
	t.Setenv("FOFA_BASE_URL", srv.URL)

	c := NewFofaClient("key")
	results, err := c.SearchDomain(context.Background(), "example.com")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestFofa_SearchDomain_EmptyKey(t *testing.T) {
	c := NewFofaClient("")
	_, err := c.SearchDomain(context.Background(), "example.com")
	if err == nil || !strings.Contains(err.Error(), "FOFA credentials") {
		t.Fatalf("expected credentials error, got: %v", err)
	}
}

func TestFofa_SearchIP_Success(t *testing.T) {
	srv := fofaMockServer(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   false,
			"results": [][]string{{"host.example", "10.0.0.1", "22"}},
		})
	})
	defer srv.Close()
	t.Setenv("FOFA_BASE_URL", srv.URL)

	c := NewFofaClient("key")
	results, err := c.SearchIP(context.Background(), "10.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].IP != "10.0.0.1" {
		t.Fatalf("unexpected results: %+v", results)
	}
}

func TestFofa_SearchIP_EmptyKey(t *testing.T) {
	c := NewFofaClient("")
	_, err := c.SearchIP(context.Background(), "10.0.0.1")
	if err == nil || !strings.Contains(err.Error(), "FOFA credentials") {
		t.Fatalf("expected credentials error, got: %v", err)
	}
}

func TestFofa_Search_Success(t *testing.T) {
	srv := fofaMockServer(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   false,
			"results": [][]string{{"example.com", "1.2.3.4", "443", "Title", "https", "nginx", "200", "CN", "SH", "Org", "ICP"}},
		})
	})
	defer srv.Close()
	t.Setenv("FOFA_BASE_URL", srv.URL)

	c := NewFofaClient("key")
	results, err := c.Search(context.Background(), "domain=example.com", 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.Engine != "fofa" || r.IP != "1.2.3.4" || r.Port != 443 {
		t.Fatalf("unexpected result: %+v", r)
	}
	if r.Organization != "Org" || r.ICP != "ICP" {
		t.Fatalf("unexpected org/icp: %+v", r)
	}
}

func TestFofa_Search_EmptyKey(t *testing.T) {
	c := NewFofaClient("")
	_, err := c.Search(context.Background(), "q", 1, 20)
	if err == nil || !strings.Contains(err.Error(), "FOFA credentials") {
		t.Fatalf("expected credentials error, got: %v", err)
	}
}

func TestFofa_Search_SizeClamping(t *testing.T) {
	var capturedSize string
	srv := fofaMockServer(func(w http.ResponseWriter, r *http.Request) {
		capturedSize = r.URL.Query().Get("size")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": false, "results": [][]string{}})
	})
	defer srv.Close()
	t.Setenv("FOFA_BASE_URL", srv.URL)

	c := NewFofaClient("key")
	// size < 1 → should clamp to 20
	_, _ = c.Search(context.Background(), "q", 1, 0)
	if capturedSize != "20" {
		t.Fatalf("expected size=20 for input 0, got %s", capturedSize)
	}
	// size > 500 → should clamp to 500
	_, _ = c.Search(context.Background(), "q", 1, 999)
	if capturedSize != "500" {
		t.Fatalf("expected size=500 for input 999, got %s", capturedSize)
	}
}

func TestFofa_Search_APIError(t *testing.T) {
	srv := fofaMockServer(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": true, "errmsg": "invalid query",
		})
	})
	defer srv.Close()
	t.Setenv("FOFA_BASE_URL", srv.URL)

	c := NewFofaClient("key")
	_, err := c.Search(context.Background(), "bad", 1, 20)
	if err == nil || !strings.Contains(err.Error(), "invalid query") {
		t.Fatalf("expected API error, got: %v", err)
	}
}

func TestFofa_Search_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{broken`))
	}))
	defer srv.Close()
	t.Setenv("FOFA_BASE_URL", srv.URL)

	c := NewFofaClient("key")
	_, err := c.Search(context.Background(), "q", 1, 20)
	if err == nil || !strings.Contains(err.Error(), "decode") {
		t.Fatalf("expected decode error, got: %v", err)
	}
}

func TestFofa_Search_Non200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()
	t.Setenv("FOFA_BASE_URL", srv.URL)

	c := NewFofaClient("key")
	_, err := c.Search(context.Background(), "q", 1, 20)
	if err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("expected 401 error, got: %v", err)
	}
}

func TestFofa_SearchCompany_EmptyKey(t *testing.T) {
	c := NewFofaClient("")
	_, err := c.SearchCompany(context.Background(), "Corp")
	if err == nil || !strings.Contains(err.Error(), "FOFA credentials") {
		t.Fatalf("expected credentials error, got: %v", err)
	}
}

func TestFofa_SearchCompany_Dedup(t *testing.T) {
	// Return same host+IP from two different queries → should deduplicate
	callCount := 0
	srv := fofaMockServer(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   false,
			"results": [][]string{{"same.example", "1.1.1.1", "80"}},
		})
	})
	defer srv.Close()
	t.Setenv("FOFA_BASE_URL", srv.URL)

	c := NewFofaClient("key")
	results, err := c.SearchCompany(context.Background(), "Corp")
	if err != nil {
		t.Fatal(err)
	}
	// 3 queries called (org, cert, title), but all return the same host+IP
	if len(results) != 1 {
		t.Fatalf("expected 1 deduplicated result, got %d", len(results))
	}
	if callCount != 3 {
		t.Fatalf("expected 3 API calls, got %d", callCount)
	}
}

func TestFofa_SearchCompany_PartialFailure(t *testing.T) {
	// First query fails, second and third succeed with different results
	callCount := 0
	srv := fofaMockServer(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		host := "host.example"
		ip := "2.2.2.2"
		if callCount == 3 {
			host = "other.example"
			ip = "3.3.3.3"
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   false,
			"results": [][]string{{host, ip, "80"}},
		})
	})
	defer srv.Close()
	t.Setenv("FOFA_BASE_URL", srv.URL)

	c := NewFofaClient("key")
	results, err := c.SearchCompany(context.Background(), "Corp")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results (1st failed, 2nd+3rd different), got %d", len(results))
	}
}

func TestFofa_GetQuota_Success(t *testing.T) {
	srv := fofaMockServer(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": false, "fcoin": 5000, "remain_api_query": 100,
		})
	})
	defer srv.Close()
	t.Setenv("FOFA_BASE_URL", srv.URL)

	c := NewFofaClient("key")
	qi, err := c.GetQuota(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(qi.Points) != 2 {
		t.Fatalf("expected 2 points, got %d", len(qi.Points))
	}
	if qi.Points[0].Value != 5000 || qi.Points[1].Value != 100 {
		t.Fatalf("unexpected values: %+v", qi.Points)
	}
}

func TestFofa_GetQuota_EmptyKey(t *testing.T) {
	c := NewFofaClient("")
	_, err := c.GetQuota(context.Background())
	if err == nil || !strings.Contains(err.Error(), "FOFA credentials") {
		t.Fatalf("expected credentials error, got: %v", err)
	}
}

func TestFofa_GetQuota_APIError(t *testing.T) {
	srv := fofaMockServer(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": true, "errmsg": "bad key",
		})
	})
	defer srv.Close()
	t.Setenv("FOFA_BASE_URL", srv.URL)

	c := NewFofaClient("key")
	_, err := c.GetQuota(context.Background())
	if err == nil || !strings.Contains(err.Error(), "bad key") {
		t.Fatalf("expected API error, got: %v", err)
	}
}

func TestFofa_GetQuota_Non200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	t.Setenv("FOFA_BASE_URL", srv.URL)

	c := NewFofaClient("key")
	_, err := c.GetQuota(context.Background())
	if err == nil || !strings.Contains(err.Error(), "503") {
		t.Fatalf("expected 503 error, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// hunter.go — NewHunterClient, Search, GetQuota, convertHunterResults
// ---------------------------------------------------------------------------

func TestNewHunterClient(t *testing.T) {
	c := NewHunterClient("mykey")
	if c.apiKey != "mykey" {
		t.Fatalf("expected apiKey=mykey, got %s", c.apiKey)
	}
	if c.baseURL != "https://hunter.qianxin.com" {
		t.Fatalf("expected default baseURL, got %s", c.baseURL)
	}
}

func TestHunter_Search_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"code": 200, "message": "ok",
			"data": map[string]interface{}{
				"total": 1,
				"arr": []map[string]interface{}{
					{
						"ip": "10.0.0.1", "port": 80, "domain": "a.com",
						"web_title": "Hello", "header_server": "nginx",
						"status_code": 200, "protocol": "http",
						"os": "Linux", "company": "Corp", "number": "ICP1",
						"country": "中国", "province": "北京", "city": "北京", "isp": "电信",
					},
				},
			},
		})
	}))
	defer srv.Close()

	c := NewHunterClient("key")
	c.baseURL = srv.URL
	results, err := c.Search(context.Background(), "domain=a.com", 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.Engine != "hunter" || r.IP != "10.0.0.1" || r.Port != 80 {
		t.Fatalf("unexpected result: %+v", r)
	}
	if r.Service != "nginx" {
		t.Fatalf("expected service=nginx, got %s", r.Service)
	}
	if r.Location != "中国 北京 | 电信" {
		t.Fatalf("unexpected location: %s", r.Location)
	}
}

func TestHunter_Search_EmptyKey(t *testing.T) {
	c := NewHunterClient("")
	_, err := c.Search(context.Background(), "q", 1, 10)
	if err == nil || !strings.Contains(err.Error(), "Hunter API key") {
		t.Fatalf("expected key error, got: %v", err)
	}
}

func TestHunter_Search_PageSizeClamping(t *testing.T) {
	var capturedPageSize string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPageSize = r.URL.Query().Get("page_size")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"code": 200, "data": map[string]interface{}{"arr": []interface{}{}},
		})
	}))
	defer srv.Close()

	c := NewHunterClient("key")
	c.baseURL = srv.URL

	// pageSize < 1 → 10
	_, _ = c.Search(context.Background(), "q", 1, 0)
	if capturedPageSize != "10" {
		t.Fatalf("expected 10 for pageSize=0, got %s", capturedPageSize)
	}
	// pageSize > 100 → 100
	_, _ = c.Search(context.Background(), "q", 1, 500)
	if capturedPageSize != "100" {
		t.Fatalf("expected 100 for pageSize=500, got %s", capturedPageSize)
	}
	// page < 1 → 1
	var capturedPage string
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPage = r.URL.Query().Get("page")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"code": 200, "data": map[string]interface{}{"arr": []interface{}{}},
		})
	}))
	defer srv2.Close()
	c.baseURL = srv2.URL
	_, _ = c.Search(context.Background(), "q", -1, 10)
	if capturedPage != "1" {
		t.Fatalf("expected page=1 for input -1, got %s", capturedPage)
	}
}

func TestHunter_Search_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"code": 401, "message": "invalid key",
		})
	}))
	defer srv.Close()

	c := NewHunterClient("key")
	c.baseURL = srv.URL
	_, err := c.Search(context.Background(), "q", 1, 10)
	if err == nil || !strings.Contains(err.Error(), "invalid key") {
		t.Fatalf("expected API error, got: %v", err)
	}
}

func TestHunter_Search_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`not json`))
	}))
	defer srv.Close()

	c := NewHunterClient("key")
	c.baseURL = srv.URL
	_, err := c.Search(context.Background(), "q", 1, 10)
	if err == nil || !strings.Contains(err.Error(), "decode") {
		t.Fatalf("expected decode error, got: %v", err)
	}
}

func TestHunter_Search_Non200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	c := NewHunterClient("key")
	c.baseURL = srv.URL
	_, err := c.Search(context.Background(), "q", 1, 10)
	if err == nil || !strings.Contains(err.Error(), "502") {
		t.Fatalf("expected 502 error, got: %v", err)
	}
}

func TestHunter_Search_EmptyResults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"code": 200, "data": map[string]interface{}{"arr": []interface{}{}, "total": 0},
		})
	}))
	defer srv.Close()

	c := NewHunterClient("key")
	c.baseURL = srv.URL
	results, err := c.Search(context.Background(), "q", 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestHunter_GetQuota_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"code": 200, "data": map[string]interface{}{
				"rest_free_point": 500, "rest_equity_point": 1000,
			},
		})
	}))
	defer srv.Close()

	c := NewHunterClient("key")
	c.baseURL = srv.URL
	qi, err := c.GetQuota(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(qi.Points) != 2 {
		t.Fatalf("expected 2 points, got %d", len(qi.Points))
	}
	if qi.Points[0].Name != "免费积分" || qi.Points[0].Value != 500 {
		t.Fatalf("unexpected free point: %+v", qi.Points[0])
	}
	if qi.Points[1].Name != "权益积分" || qi.Points[1].Value != 1000 {
		t.Fatalf("unexpected equity point: %+v", qi.Points[1])
	}
}

func TestHunter_GetQuota_ZeroPoints(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"code": 200, "data": map[string]interface{}{
				"rest_free_point": 0, "rest_equity_point": 0,
			},
		})
	}))
	defer srv.Close()

	c := NewHunterClient("key")
	c.baseURL = srv.URL
	qi, err := c.GetQuota(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// Both zero → no points returned
	if len(qi.Points) != 0 {
		t.Fatalf("expected 0 points when both zero, got %d", len(qi.Points))
	}
}

func TestHunter_GetQuota_EmptyKey(t *testing.T) {
	c := NewHunterClient("")
	_, err := c.GetQuota(context.Background())
	if err == nil || !strings.Contains(err.Error(), "Hunter API key") {
		t.Fatalf("expected key error, got: %v", err)
	}
}

func TestHunter_GetQuota_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"code": 401, "message": "bad key",
		})
	}))
	defer srv.Close()

	c := NewHunterClient("key")
	c.baseURL = srv.URL
	_, err := c.GetQuota(context.Background())
	if err == nil || !strings.Contains(err.Error(), "bad key") {
		t.Fatalf("expected API error, got: %v", err)
	}
}

func TestHunter_GetQuota_Non200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewHunterClient("key")
	c.baseURL = srv.URL
	_, err := c.GetQuota(context.Background())
	if err == nil || !strings.Contains(err.Error(), "500") {
		t.Fatalf("expected 500 error, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// convertHunterResults
// ---------------------------------------------------------------------------

func TestConvertHunterResults_NilEntries(t *testing.T) {
	raw := []*HunterResult{nil, nil}
	results := convertHunterResults(raw)
	if len(results) != 0 {
		t.Fatalf("expected 0, got %d", len(results))
	}
}

func TestConvertHunterResults_HeaderServerPriority(t *testing.T) {
	raw := []*HunterResult{{
		IP: "1.1.1.1", Port: 80, HeaderServer: "nginx",
		Components: []HunterComponent{{Name: "apache", Version: "2.4"}},
		Banner:     "SSH-2.0",
	}}
	results := convertHunterResults(raw)
	if results[0].Service != "nginx" {
		t.Fatalf("expected header_server priority, got %s", results[0].Service)
	}
}

func TestConvertHunterResults_ComponentsFallback(t *testing.T) {
	raw := []*HunterResult{{
		IP: "1.1.1.1", Port: 80,
		Components: []HunterComponent{{Name: "nginx", Version: "1.21"}, {Name: "php", Version: "8.0"}},
	}}
	results := convertHunterResults(raw)
	if results[0].Service != "nginx 1.21 / php 8.0" {
		t.Fatalf("expected components fallback, got %s", results[0].Service)
	}
}

func TestConvertHunterResults_ComponentNoVersion(t *testing.T) {
	raw := []*HunterResult{{
		IP: "1.1.1.1", Port: 80,
		Components: []HunterComponent{{Name: "nginx"}},
	}}
	results := convertHunterResults(raw)
	if results[0].Service != "nginx" {
		t.Fatalf("expected component name only, got %s", results[0].Service)
	}
}

func TestConvertHunterResults_ComponentEmptyNameSkipped(t *testing.T) {
	raw := []*HunterResult{{
		IP: "1.1.1.1", Port: 80,
		Components: []HunterComponent{{Name: "", Version: ""}, {Name: "nginx"}},
	}}
	results := convertHunterResults(raw)
	if results[0].Service != "nginx" {
		t.Fatalf("expected empty component skipped, got %s", results[0].Service)
	}
}

func TestConvertHunterResults_BannerFallback(t *testing.T) {
	raw := []*HunterResult{{
		IP: "1.1.1.1", Port: 80, Banner: "SSH-2.0-OpenSSH",
	}}
	results := convertHunterResults(raw)
	if results[0].Service != "SSH-2.0-OpenSSH" {
		t.Fatalf("expected banner fallback, got %s", results[0].Service)
	}
}

func TestConvertHunterResults_LocationProvinceSameAsCountry(t *testing.T) {
	raw := []*HunterResult{{
		IP: "1.1.1.1", Country: "中国", Province: "中国", City: "北京", ISP: "电信",
	}}
	results := convertHunterResults(raw)
	// Province == Country → skip province; City != Province → include
	if results[0].Location != "中国 北京 | 电信" {
		t.Fatalf("unexpected location: %q", results[0].Location)
	}
}

func TestConvertHunterResults_LocationProvinceSameCity(t *testing.T) {
	raw := []*HunterResult{{
		IP: "1.1.1.1", Country: "中国", Province: "上海", City: "上海", ISP: "联通",
	}}
	results := convertHunterResults(raw)
	// Province != Country → include; City == Province → skip city
	if results[0].Location != "中国 上海 | 联通" {
		t.Fatalf("unexpected location: %q", results[0].Location)
	}
}

func TestConvertHunterResults_LocationProvinceDifferent(t *testing.T) {
	raw := []*HunterResult{{
		IP: "1.1.1.1", Country: "中国", Province: "北京", City: "朝阳", ISP: "移动",
	}}
	results := convertHunterResults(raw)
	if results[0].Location != "中国 北京 朝阳 | 移动" {
		t.Fatalf("unexpected location: %q", results[0].Location)
	}
}

func TestConvertHunterResults_LocationISPOnly(t *testing.T) {
	raw := []*HunterResult{{
		IP: "1.1.1.1", ISP: "电信",
	}}
	results := convertHunterResults(raw)
	if results[0].Location != "电信" {
		t.Fatalf("unexpected location: %q", results[0].Location)
	}
}

func TestConvertHunterResults_LocationEmpty(t *testing.T) {
	raw := []*HunterResult{{
		IP: "1.1.1.1",
	}}
	results := convertHunterResults(raw)
	if results[0].Location != "" {
		t.Fatalf("expected empty location, got %q", results[0].Location)
	}
}

func TestConvertHunterResults_FullMapping(t *testing.T) {
	raw := []*HunterResult{{
		IP: "10.0.0.1", Port: 443, Domain: "example.com",
		WebTitle: "Test", HeaderServer: "nginx", StatusCode: 200,
		Protocol: "https", OS: "Linux", Company: "Corp", Number: "ICP123",
	}}
	results := convertHunterResults(raw)
	r := results[0]
	if r.Engine != "hunter" || r.IP != "10.0.0.1" || r.Port != 443 {
		t.Fatalf("unexpected basic: %+v", r)
	}
	if r.Domain != "example.com" || r.Title != "Test" || r.Service != "nginx" {
		t.Fatalf("unexpected service: %+v", r)
	}
	if r.OS != "Linux" || r.Organization != "Corp" || r.ICP != "ICP123" {
		t.Fatalf("unexpected meta: %+v", r)
	}
	if r.Raw == nil {
		t.Fatal("expected Raw to be set")
	}
}

// ---------------------------------------------------------------------------
// quake.go — NewQuakeClient
// ---------------------------------------------------------------------------

func TestNewQuakeClient_DefaultBaseURL(t *testing.T) {
	c := NewQuakeClient("key")
	if c.baseURL != "https://quake.360.net" {
		t.Fatalf("expected default baseURL, got %s", c.baseURL)
	}
}

func TestNewQuakeClient_EnvOverride(t *testing.T) {
	t.Setenv("QUAKE_BASE_URL", "http://mock-quake:8888/")
	c := NewQuakeClient("key")
	if c.baseURL != "http://mock-quake:8888" {
		t.Fatalf("expected trimmed override, got %s", c.baseURL)
	}
}

// ---------------------------------------------------------------------------
// quake.go — Search
// ---------------------------------------------------------------------------

func TestQuake_Search_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"code": 0, "message": "success", "total_count": 1,
			"data": []map[string]interface{}{
				{
					"ip": "10.0.0.1", "port": 80, "domain": "a.com", "os_name": "Linux",
					"service": map[string]interface{}{
						"name": "http", "version": "1.1",
						"http": map[string]interface{}{
							"title": "Welcome", "server": "nginx", "status_code": 200,
						},
					},
					"location": map[string]interface{}{
						"country_cn": "中国", "province_cn": "北京", "city_cn": "海淀", "isp": "联通",
					},
					"org": "Corp",
				},
			},
		})
	}))
	defer srv.Close()

	c := NewQuakeClient("key")
	c.baseURL = srv.URL
	results, err := c.Search(context.Background(), "domain:a.com", 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.Engine != "quake" || r.IP != "10.0.0.1" || r.Port != 80 {
		t.Fatalf("unexpected result: %+v", r)
	}
	if r.Title != "Welcome" || r.Service != "nginx" {
		t.Fatalf("unexpected title/service: title=%q service=%q", r.Title, r.Service)
	}
	if r.Location != "中国 北京 海淀 | 联通" {
		t.Fatalf("unexpected location: %q", r.Location)
	}
	if r.StatusCode != 200 || r.OS != "Linux" || r.Organization != "Corp" {
		t.Fatalf("unexpected meta: %+v", r)
	}
}

func TestQuake_Search_EmptyKey(t *testing.T) {
	c := NewQuakeClient("")
	_, err := c.Search(context.Background(), "q", 0, 10)
	if err == nil || !strings.Contains(err.Error(), "Quake API key") {
		t.Fatalf("expected key error, got: %v", err)
	}
}

func TestQuake_Search_SizeClamping(t *testing.T) {
	var capturedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&capturedBody)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"code": 0, "data": []interface{}{},
		})
	}))
	defer srv.Close()

	c := NewQuakeClient("key")
	c.baseURL = srv.URL

	// size < 1 → 10
	_, _ = c.Search(context.Background(), "q", 0, 0)
	if capturedBody["size"].(float64) != 10 {
		t.Fatalf("expected size=10 for input 0, got %v", capturedBody["size"])
	}
	// size > 100 → 100
	_, _ = c.Search(context.Background(), "q", 0, 999)
	if capturedBody["size"].(float64) != 100 {
		t.Fatalf("expected size=100 for input 999, got %v", capturedBody["size"])
	}
}

func TestQuake_Search_StartClamping(t *testing.T) {
	var capturedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&capturedBody)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"code": 0, "data": []interface{}{},
		})
	}))
	defer srv.Close()

	c := NewQuakeClient("key")
	c.baseURL = srv.URL
	_, _ = c.Search(context.Background(), "q", -5, 10)
	if capturedBody["start"].(float64) != 0 {
		t.Fatalf("expected start=0 for input -5, got %v", capturedBody["start"])
	}
}

func TestQuake_Search_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"code": -1, "message": "quota exceeded",
		})
	}))
	defer srv.Close()

	c := NewQuakeClient("key")
	c.baseURL = srv.URL
	_, err := c.Search(context.Background(), "q", 0, 10)
	if err == nil || !strings.Contains(err.Error(), "quota exceeded") {
		t.Fatalf("expected API error, got: %v", err)
	}
}

func TestQuake_Search_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{broken`))
	}))
	defer srv.Close()

	c := NewQuakeClient("key")
	c.baseURL = srv.URL
	_, err := c.Search(context.Background(), "q", 0, 10)
	if err == nil || !strings.Contains(err.Error(), "decode") {
		t.Fatalf("expected decode error, got: %v", err)
	}
}

func TestQuake_Search_Non200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusGatewayTimeout)
	}))
	defer srv.Close()

	c := NewQuakeClient("key")
	c.baseURL = srv.URL
	_, err := c.Search(context.Background(), "q", 0, 10)
	if err == nil || !strings.Contains(err.Error(), "504") {
		t.Fatalf("expected 504 error, got: %v", err)
	}
}

func TestQuake_Search_EmptyResults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"code": 0, "data": []interface{}{}, "total_count": 0,
		})
	}))
	defer srv.Close()

	c := NewQuakeClient("key")
	c.baseURL = srv.URL
	results, err := c.Search(context.Background(), "q", 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestQuake_Search_TokenHeader(t *testing.T) {
	var capturedToken string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedToken = r.Header.Get("X-QuakeToken")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"code": 0, "data": []interface{}{},
		})
	}))
	defer srv.Close()

	c := NewQuakeClient("my-secret-token")
	c.baseURL = srv.URL
	_, _ = c.Search(context.Background(), "q", 0, 10)
	if capturedToken != "my-secret-token" {
		t.Fatalf("expected token=my-secret-token, got %s", capturedToken)
	}
}

// ---------------------------------------------------------------------------
// quake.go — GetQuota
// ---------------------------------------------------------------------------

func TestQuake_GetQuota_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"code": 0, "data": map[string]interface{}{
				"month_remaining_credit": 5000, "total_remaining_credit": 50000,
			},
		})
	}))
	defer srv.Close()

	c := NewQuakeClient("key")
	c.baseURL = srv.URL
	qi, err := c.GetQuota(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(qi.Points) != 2 {
		t.Fatalf("expected 2 points, got %d", len(qi.Points))
	}
	if qi.Points[0].Name != "月度积分" || qi.Points[0].Value != 5000 {
		t.Fatalf("unexpected monthly: %+v", qi.Points[0])
	}
	if qi.Points[1].Name != "长效积分" || qi.Points[1].Value != 50000 {
		t.Fatalf("unexpected total: %+v", qi.Points[1])
	}
}

func TestQuake_GetQuota_EmptyKey(t *testing.T) {
	c := NewQuakeClient("")
	_, err := c.GetQuota(context.Background())
	if err == nil || !strings.Contains(err.Error(), "Quake API key") {
		t.Fatalf("expected key error, got: %v", err)
	}
}

func TestQuake_GetQuota_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"code": -1, "message": "auth failed",
		})
	}))
	defer srv.Close()

	c := NewQuakeClient("key")
	c.baseURL = srv.URL
	_, err := c.GetQuota(context.Background())
	if err == nil || !strings.Contains(err.Error(), "auth failed") {
		t.Fatalf("expected API error, got: %v", err)
	}
}

func TestQuake_GetQuota_Non200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	c := NewQuakeClient("key")
	c.baseURL = srv.URL
	_, err := c.GetQuota(context.Background())
	if err == nil || !strings.Contains(err.Error(), "403") {
		t.Fatalf("expected 403 error, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// convertQuakeResults
// ---------------------------------------------------------------------------

func TestConvertQuakeResults_NilEntries(t *testing.T) {
	raw := []*QuakeResult{nil, nil}
	results := convertQuakeResults(raw)
	if len(results) != 0 {
		t.Fatalf("expected 0, got %d", len(results))
	}
}

func TestConvertQuakeResults_HTTPServerPriority(t *testing.T) {
	raw := []*QuakeResult{{
		IP: "1.1.1.1", Port: 80,
		Service: struct {
			Name     string    `json:"name"`
			Version  string    `json:"version"`
			Response string    `json:"response"`
			HTTP     QuakeHTTP `json:"http"`
		}{
			Name: "http", HTTP: QuakeHTTP{Server: "nginx"},
		},
		Components: []QuakeComponent{{ProductNameCN: "apache", Version: "2.4"}},
	}}
	results := convertQuakeResults(raw)
	if results[0].Service != "nginx" {
		t.Fatalf("expected HTTP.Server priority, got %s", results[0].Service)
	}
}

func TestConvertQuakeResults_ComponentsFallback(t *testing.T) {
	raw := []*QuakeResult{{
		IP: "1.1.1.1", Port: 80,
		Service: struct {
			Name     string    `json:"name"`
			Version  string    `json:"version"`
			Response string    `json:"response"`
			HTTP     QuakeHTTP `json:"http"`
		}{Name: "http"},
		Components: []QuakeComponent{
			{ProductNameCN: "Nginx", Version: "1.21"},
			{ProductNameCN: "", ProductNameEN: "PHP", Version: "8.0"},
		},
	}}
	results := convertQuakeResults(raw)
	if results[0].Service != "Nginx 1.21 / PHP 8.0" {
		t.Fatalf("expected components fallback, got %s", results[0].Service)
	}
}

func TestConvertQuakeResults_ComponentEmptyNameSkipped(t *testing.T) {
	raw := []*QuakeResult{{
		IP: "1.1.1.1", Port: 80,
		Service: struct {
			Name     string    `json:"name"`
			Version  string    `json:"version"`
			Response string    `json:"response"`
			HTTP     QuakeHTTP `json:"http"`
		}{Name: "http"},
		Components: []QuakeComponent{{ProductNameCN: "", ProductNameEN: "", Version: ""}},
	}}
	results := convertQuakeResults(raw)
	// All empty components → fallback to service.name
	if results[0].Service != "http" {
		t.Fatalf("expected service.name fallback, got %s", results[0].Service)
	}
}

func TestConvertQuakeResults_ServiceNameFallback(t *testing.T) {
	raw := []*QuakeResult{{
		IP: "1.1.1.1", Port: 80,
		Service: struct {
			Name     string    `json:"name"`
			Version  string    `json:"version"`
			Response string    `json:"response"`
			HTTP     QuakeHTTP `json:"http"`
		}{Name: "ssh", Version: "2.0"},
	}}
	results := convertQuakeResults(raw)
	if results[0].Service != "ssh 2.0" {
		t.Fatalf("expected service fallback, got %s", results[0].Service)
	}
}

func TestConvertQuakeResults_ServiceNameNoVersion(t *testing.T) {
	raw := []*QuakeResult{{
		IP: "1.1.1.1", Port: 80,
		Service: struct {
			Name     string    `json:"name"`
			Version  string    `json:"version"`
			Response string    `json:"response"`
			HTTP     QuakeHTTP `json:"http"`
		}{Name: "dns"},
	}}
	results := convertQuakeResults(raw)
	if results[0].Service != "dns" {
		t.Fatalf("expected dns, got %s", results[0].Service)
	}
}

func TestConvertQuakeResults_TitleFromHTTP(t *testing.T) {
	raw := []*QuakeResult{{
		IP: "1.1.1.1",
		Service: struct {
			Name     string    `json:"name"`
			Version  string    `json:"version"`
			Response string    `json:"response"`
			HTTP     QuakeHTTP `json:"http"`
		}{HTTP: QuakeHTTP{Title: "My Site"}},
	}}
	results := convertQuakeResults(raw)
	if results[0].Title != "My Site" {
		t.Fatalf("expected HTTP title, got %s", results[0].Title)
	}
}

func TestConvertQuakeResults_TitleFromHostname(t *testing.T) {
	raw := []*QuakeResult{{
		IP: "1.1.1.1", Hostname: "fallback.host",
		Service: struct {
			Name     string    `json:"name"`
			Version  string    `json:"version"`
			Response string    `json:"response"`
			HTTP     QuakeHTTP `json:"http"`
		}{},
	}}
	results := convertQuakeResults(raw)
	if results[0].Title != "fallback.host" {
		t.Fatalf("expected hostname fallback, got %s", results[0].Title)
	}
}

func TestConvertQuakeResults_LocationFull(t *testing.T) {
	raw := []*QuakeResult{{
		IP: "1.1.1.1",
		Location: struct {
			Country  string `json:"country_cn"`
			City     string `json:"city_cn"`
			Province string `json:"province_cn"`
			ISP      string `json:"isp"`
		}{Country: "中国", Province: "广东", City: "深圳", ISP: "电信"},
	}}
	results := convertQuakeResults(raw)
	if results[0].Location != "中国 广东 深圳 | 电信" {
		t.Fatalf("unexpected location: %q", results[0].Location)
	}
}

func TestConvertQuakeResults_LocationProvinceSameAsCountry(t *testing.T) {
	raw := []*QuakeResult{{
		IP: "1.1.1.1",
		Location: struct {
			Country  string `json:"country_cn"`
			City     string `json:"city_cn"`
			Province string `json:"province_cn"`
			ISP      string `json:"isp"`
		}{Country: "中国", Province: "中国", City: "上海", ISP: "移动"},
	}}
	results := convertQuakeResults(raw)
	// Province == Country → skip province; City != Province → include
	if results[0].Location != "中国 上海 | 移动" {
		t.Fatalf("unexpected location: %q", results[0].Location)
	}
}

func TestConvertQuakeResults_LocationSameProvinceCity(t *testing.T) {
	raw := []*QuakeResult{{
		IP: "1.1.1.1",
		Location: struct {
			Country  string `json:"country_cn"`
			City     string `json:"city_cn"`
			Province string `json:"province_cn"`
			ISP      string `json:"isp"`
		}{Country: "中国", Province: "北京", City: "北京", ISP: "联通"},
	}}
	results := convertQuakeResults(raw)
	// Province == City → skip city
	if results[0].Location != "中国 北京 | 联通" {
		t.Fatalf("unexpected location: %q", results[0].Location)
	}
}

func TestConvertQuakeResults_LocationEmpty(t *testing.T) {
	raw := []*QuakeResult{{IP: "1.1.1.1"}}
	results := convertQuakeResults(raw)
	if results[0].Location != "" {
		t.Fatalf("expected empty location, got %q", results[0].Location)
	}
}

func TestConvertQuakeResults_FullMapping(t *testing.T) {
	raw := []*QuakeResult{{
		IP: "10.0.0.1", Port: 443, Domain: "example.com", OS: "Windows",
		Service: struct {
			Name     string    `json:"name"`
			Version  string    `json:"version"`
			Response string    `json:"response"`
			HTTP     QuakeHTTP `json:"http"`
		}{
			Name: "https", HTTP: QuakeHTTP{StatusCode: 200, Title: "Test", Server: "IIS"},
		},
		Org: "Corp",
	}}
	results := convertQuakeResults(raw)
	r := results[0]
	if r.Engine != "quake" || r.IP != "10.0.0.1" || r.Port != 443 {
		t.Fatalf("unexpected basic: %+v", r)
	}
	if r.Domain != "example.com" || r.OS != "Windows" || r.Organization != "Corp" {
		t.Fatalf("unexpected meta: %+v", r)
	}
	if r.StatusCode != 200 || r.Protocol != "https" {
		t.Fatalf("unexpected protocol/status: %+v", r)
	}
	if r.Raw == nil {
		t.Fatal("expected Raw to be set")
	}
}

// ---------------------------------------------------------------------------
// joinParts
// ---------------------------------------------------------------------------

func TestJoinParts_Empty(t *testing.T) {
	if joinParts(nil, ",") != "" {
		t.Fatal("expected empty for nil")
	}
	if joinParts([]string{}, ",") != "" {
		t.Fatal("expected empty for empty slice")
	}
}

func TestJoinParts_Single(t *testing.T) {
	if joinParts([]string{"a"}, ",") != "a" {
		t.Fatal("expected 'a'")
	}
}

func TestJoinParts_Multiple(t *testing.T) {
	got := joinParts([]string{"a", "b", "c"}, " / ")
	if got != "a / b / c" {
		t.Fatalf("expected 'a / b / c', got %q", got)
	}
}

// ---------------------------------------------------------------------------
// Engine interface compliance
// ---------------------------------------------------------------------------

func TestEngineInterface(t *testing.T) {
	// Verify all three clients implement the Engine interface
	var _ Engine = &FofaClient{}
	var _ Engine = &HunterClient{}
	var _ Engine = &QuakeClient{}
}
