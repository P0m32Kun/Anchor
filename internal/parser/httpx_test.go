package parser

import (
	"strings"
	"testing"
)

func TestParseHTTPX(t *testing.T) {
	input := `{"url":"https://sub.example.com","input":"sub.example.com","title":"Example","webserver":"nginx","status-code":200,"host":"sub.example.com","port":"443","scheme":"https","path":"/"}
{"url":"http://sub.example.com:8080","input":"sub.example.com","title":"Dev","status-code":302}
invalid json
{"input":"sub.example.com","title":"Missing URL"}
{"url":"","input":"sub.example.com"}
`

	results, errs := ParseHTTPX(strings.NewReader(input))

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].URL != "https://sub.example.com" {
		t.Errorf("expected url https://sub.example.com, got %s", results[0].URL)
	}
	if results[0].StatusCode != 200 {
		t.Errorf("expected status 200, got %d", results[0].StatusCode)
	}
	if results[0].WebServer != "nginx" {
		t.Errorf("expected webserver nginx, got %s", results[0].WebServer)
	}
	if results[1].URL != "http://sub.example.com:8080" {
		t.Errorf("expected url http://sub.example.com:8080, got %s", results[1].URL)
	}
	if results[1].StatusCode != 302 {
		t.Errorf("expected status 302, got %d", results[1].StatusCode)
	}

	if len(errs) != 3 {
		t.Fatalf("expected 3 parse errors, got %d", len(errs))
	}
}

func TestParseHTTPXEmpty(t *testing.T) {
	results, errs := ParseHTTPX(strings.NewReader(""))
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
	if len(errs) != 0 {
		t.Errorf("expected 0 errors, got %d", len(errs))
	}
}

func TestParseHTTPXPortInt(t *testing.T) {
	input := `{"url":"https://a.com","port":8443}`
	results, errs := ParseHTTPX(strings.NewReader(input))
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Port != "8443" {
		t.Errorf("expected port 8443, got %s", results[0].Port)
	}
	if len(errs) != 0 {
		t.Errorf("expected 0 errors, got %d", len(errs))
	}
}

func TestParseHTTPXTechnologies(t *testing.T) {
	input := `{"url":"https://a.com","tech":["nginx","php"]}`
	results, errs := ParseHTTPX(strings.NewReader(input))
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if len(results[0].Tech) != 2 || results[0].Tech[0] != "nginx" {
		t.Errorf("expected tech [nginx php], got %v", results[0].Tech)
	}
	if len(errs) != 0 {
		t.Errorf("expected 0 errors, got %d", len(errs))
	}
}

func TestParseHTTPXTechnologiesFallback(t *testing.T) {
	input := `{"url":"http://example.com","technologies":["Vue","Nginx"],"status-code":200}`
	results, errs := ParseHTTPX(strings.NewReader(input))
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	res := results[0]
	if len(res.Tech) != 2 || res.Tech[0] != "Vue" || res.Tech[1] != "Nginx" {
		t.Errorf("expected tech [Vue Nginx], got %v", res.Tech)
	}
	if res.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", res.StatusCode)
	}
}

func TestParseHTTPXCPE(t *testing.T) {
	input := `{"url":"http://127.0.0.1:8080","status-code":404,"cpe":[{"product":"tomcat","vendor":"apache","cpe":"cpe:2.3:a:apache:tomcat:*:*:*:*:*:*:*:*"}]}`
	results, errs := ParseHTTPX(strings.NewReader(input))
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	res := results[0]
	if len(res.Tech) != 1 || res.Tech[0] != "tomcat" {
		t.Errorf("expected tech [tomcat] from cpe, got %v", res.Tech)
	}
}

func TestParseHTTPXCPEMerge(t *testing.T) {
	input := `{"url":"http://127.0.0.1:3000","tech":["grafana"],"cpe":[{"product":"grafana","vendor":"grafana","cpe":"cpe:2.3:a:grafana:grafana:*:*:*:*:*:*:*:*"}]}`
	results, errs := ParseHTTPX(strings.NewReader(input))
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	res := results[0]
	if len(res.Tech) != 1 || res.Tech[0] != "grafana" {
		t.Errorf("expected tech [grafana] without duplication, got %v", res.Tech)
	}
}
