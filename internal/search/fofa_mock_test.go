package search

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFofaClient_SearchCompany_MockServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": false, "size": 1,
			"results": [][]string{{"sub.fofa.example", "198.51.100.1", "80", "t", "http", "nginx"}},
		})
	}))
	defer srv.Close()

	t.Setenv("FOFA_BASE_URL", srv.URL)
	c := NewFofaClient("k")
	res, err := c.SearchCompany(context.Background(), "TestCorp")
	if err != nil {
		t.Fatal(err)
	}
	if len(res) == 0 {
		t.Fatal("expected results")
	}
	if res[0].Host != "sub.fofa.example" {
		t.Fatalf("host = %q", res[0].Host)
	}
}
