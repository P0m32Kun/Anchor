package executor

import (
	"testing"
)

func TestParseHttpxOutput_Normal(t *testing.T) {
	stdout := []byte(`{"url":"https://example.com","input":"example.com","status_code":200,"tech":["nginx"],"title":"Example","host":"example.com","port":"443","scheme":"https","path":""}
{"url":"http://test.local","input":"test.local","status_code":404,"tech":[],"title":"Not Found","host":"test.local","port":"80","scheme":"http","path":""}`)
	assets, attrs, endpoints, err := ParseHttpxOutput(stdout, "proj1")
	if err != nil {
		t.Fatal(err)
	}
	if len(assets) != 2 {
		t.Fatalf("expected 2 assets, got %d", len(assets))
	}
	if assets[0].Value != "https://example.com" {
		t.Errorf("asset[0].Value = %s", assets[0].Value)
	}
	if !attrs.Fingerprinted {
		t.Error("expected Fingerprinted=true")
	}
	// httpxEntry.StatusCode is populated from JSON, but ParseHttpxOutput only
	// sets attrs.StatusCode when StatusCode > 0; the last entry wins.
	// With 200 and 404, the final attrs.StatusCode should be 404.
	if attrs.StatusCode == nil || *attrs.StatusCode != 404 {
		t.Errorf("expected StatusCode=404 (last entry), got %v", attrs.StatusCode)
	}
	// Verify web endpoints are returned with technologies
	if len(endpoints) != 2 {
		t.Fatalf("expected 2 endpoints, got %d", len(endpoints))
	}
	if len(endpoints[0].Technologies) != 1 || endpoints[0].Technologies[0] != "nginx" {
		t.Errorf("expected endpoint[0] technologies=[nginx], got %v", endpoints[0].Technologies)
	}
	if endpoints[0].ProjectID != "proj1" {
		t.Errorf("expected endpoint[0].ProjectID=proj1, got %s", endpoints[0].ProjectID)
	}
}

func TestParseHttpxOutput_Empty(t *testing.T) {
	assets, _, _, err := ParseHttpxOutput([]byte(""), "proj1")
	if err != nil {
		t.Fatal(err)
	}
	if len(assets) != 0 {
		t.Fatalf("expected 0 assets, got %d", len(assets))
	}
}

func TestParseHttpxOutput_MalformedLines(t *testing.T) {
	stdout := []byte(`not json
{"url":"https://ok.com","input":"ok.com","status_code":200}
{"broken`)
	assets, _, _, err := ParseHttpxOutput(stdout, "proj1")
	if err != nil {
		t.Fatal(err)
	}
	if len(assets) != 1 {
		t.Fatalf("expected 1 asset (skipping malformed), got %d", len(assets))
	}
}

func TestParseNucleiOutput_Normal(t *testing.T) {
	stdout := []byte(`{"template-id":"CVE-2021-1234","info":{"name":"Test Vuln","severity":"high"},"host":"http://example.com","ip":"1.2.3.4","matched-at":"http://example.com/vuln"}
{"template-id":"CVE-2021-5678","info":{"name":"Another","severity":"low"},"host":"http://test.com","ip":"5.6.7.8","matched-at":"http://test.com/bug"}`)
	findings, err := ParseNucleiOutput(stdout)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 2 {
		t.Fatalf("expected 2 findings, got %d", len(findings))
	}
	if findings[0] != "CVE-2021-1234" {
		t.Errorf("findings[0] = %s", findings[0])
	}
}

func TestParseNucleiOutput_Empty(t *testing.T) {
	findings, err := ParseNucleiOutput([]byte(""))
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Fatalf("expected 0, got %d", len(findings))
	}
}

func TestParseFFUFOutput_JSON(t *testing.T) {
	stdout := []byte(`{"results":[{"url":"http://example.com/admin","status":200,"content-length":1234},{"url":"http://example.com/login","status":200,"content-length":5678}]}`)
	assets, err := ParseFFUFOutput(stdout)
	if err != nil {
		t.Fatal(err)
	}
	if len(assets) != 2 {
		t.Fatalf("expected 2 assets, got %d", len(assets))
	}
	if assets[0].Value != "http://example.com/admin" {
		t.Errorf("value = %s", assets[0].Value)
	}
}

func TestParseFFUFOutput_Empty(t *testing.T) {
	assets, err := ParseFFUFOutput([]byte(""))
	if err != nil {
		t.Fatal(err)
	}
	if len(assets) != 0 {
		t.Fatalf("expected 0, got %d", len(assets))
	}
}

func TestParseKatanaOutput_Normal(t *testing.T) {
	stdout := []byte(`{"url":"http://example.com/page1","source":"http://example.com"}
{"url":"http://example.com/page2","source":"http://example.com"}`)
	assets, err := ParseKatanaOutput(stdout)
	if err != nil {
		t.Fatal(err)
	}
	if len(assets) != 2 {
		t.Fatalf("expected 2, got %d", len(assets))
	}
}

func TestParseKatanaOutput_Dedup(t *testing.T) {
	stdout := []byte(`{"url":"http://example.com/same","source":"http://example.com"}
{"url":"http://example.com/same","source":"http://example.com"}`)
	assets, err := ParseKatanaOutput(stdout)
	if err != nil {
		t.Fatal(err)
	}
	if len(assets) != 1 {
		t.Fatalf("expected 1 (deduped), got %d", len(assets))
	}
}

func TestParseKatanaOutput_Empty(t *testing.T) {
	assets, err := ParseKatanaOutput([]byte(""))
	if err != nil {
		t.Fatal(err)
	}
	if len(assets) != 0 {
		t.Fatalf("expected 0, got %d", len(assets))
	}
}

func TestParseSpoorOutput_Endpoint(t *testing.T) {
	stdout := []byte(`{"file":"http://target.com/app.js","kind":"endpoint","value":"https://target.com/api/users","confidence":"high","method":"GET","origin":{"pattern":"fetch","snippet":"fetch(\"/api/users\")","line":9,"column":3}}
{"file":"http://target.com/app.js","kind":"endpoint","value":"https://target.com/api/admin","confidence":"medium","origin":{"pattern":"xhr.open","snippet":"xhr.open(\"GET\",\"/api/admin\")","line":15,"column":3}}`)
	endpoints, findings, err := ParseSpoorOutput(stdout, "run1", "proj1")
	if err != nil {
		t.Fatal(err)
	}
	if len(endpoints) != 2 {
		t.Fatalf("expected 2 endpoints, got %d", len(endpoints))
	}
	if endpoints[0].Value != "https://target.com/api/users" {
		t.Errorf("endpoint[0].Value = %s", endpoints[0].Value)
	}
	if endpoints[0].SourceTool != "spoor" {
		t.Errorf("endpoint[0].SourceTool = %s", endpoints[0].SourceTool)
	}
	if len(findings) != 0 {
		t.Fatalf("expected 0 findings, got %d", len(findings))
	}
}

func TestParseSpoorOutput_Secret(t *testing.T) {
	stdout := []byte(`{"file":"http://target.com/app.js","kind":"secret","value":"AKIAIOSFODNN7EXAMPLE","confidence":"high","secret_type":"aws_access_key","severity":"critical","origin":{"pattern":"string_literal","snippet":"const key = \"AKIAIOSFODNN7EXAMPLE\"","line":7,"column":15}}`)
	endpoints, findings, err := ParseSpoorOutput(stdout, "run1", "proj1")
	if err != nil {
		t.Fatal(err)
	}
	if len(endpoints) != 0 {
		t.Fatalf("expected 0 endpoints, got %d", len(endpoints))
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	f := findings[0]
	if f.SourceTool != "spoor" {
		t.Errorf("SourceTool = %s", f.SourceTool)
	}
	if f.SourceRuleID != "aws_access_key" {
		t.Errorf("SourceRuleID = %s", f.SourceRuleID)
	}
	if f.Severity != "critical" {
		t.Errorf("Severity = %s", f.Severity)
	}
	if f.Confidence != 90 {
		t.Errorf("Confidence = %d", f.Confidence)
	}
	if f.RunID == nil || *f.RunID != "run1" {
		t.Errorf("RunID = %v", f.RunID)
	}
}

func TestParseSpoorOutput_Path(t *testing.T) {
	stdout := []byte(`{"file":"http://target.com/app.js","kind":"path","value":"/api/admin","confidence":"medium","origin":{"pattern":"string_literal","snippet":"const path = \"/api/admin\"","line":12,"column":5}}`)
	endpoints, findings, err := ParseSpoorOutput(stdout, "run1", "proj1")
	if err != nil {
		t.Fatal(err)
	}
	if len(endpoints) != 0 {
		t.Fatalf("expected 0 endpoints, got %d", len(endpoints))
	}
	if len(findings) != 0 {
		t.Fatalf("expected 0 findings, got %d", len(findings))
	}
}

func TestParseSpoorOutput_Mixed(t *testing.T) {
	stdout := []byte(`{"file":"http://target.com/app.js","kind":"endpoint","value":"https://target.com/api/users","confidence":"high","origin":{"pattern":"fetch","snippet":"fetch(\"/api/users\")","line":9,"column":3}}
{"file":"http://target.com/app.js","kind":"secret","value":"AKIAIOSFODNN7EXAMPLE","confidence":"high","secret_type":"aws_access_key","severity":"critical","origin":{"pattern":"string_literal","snippet":"...","line":7,"column":15}}
{"file":"http://target.com/app.js","kind":"path","value":"/api/admin","confidence":"medium","origin":{"pattern":"string_literal","snippet":"...","line":12,"column":5}}`)
	endpoints, findings, err := ParseSpoorOutput(stdout, "run1", "proj1")
	if err != nil {
		t.Fatal(err)
	}
	if len(endpoints) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(endpoints))
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
}

func TestParseSpoorOutput_Dedup(t *testing.T) {
	stdout := []byte(`{"file":"http://target.com/app.js","kind":"endpoint","value":"https://target.com/api/users","confidence":"high","origin":{"pattern":"fetch","snippet":"...","line":9,"column":3}}
{"file":"http://target.com/app.js","kind":"endpoint","value":"https://target.com/api/users","confidence":"medium","origin":{"pattern":"xhr.open","snippet":"...","line":15,"column":3}}`)
	endpoints, _, err := ParseSpoorOutput(stdout, "run1", "proj1")
	if err != nil {
		t.Fatal(err)
	}
	if len(endpoints) != 1 {
		t.Fatalf("expected 1 (deduped), got %d", len(endpoints))
	}
}

func TestParseSpoorOutput_Empty(t *testing.T) {
	endpoints, findings, err := ParseSpoorOutput([]byte(""), "run1", "proj1")
	if err != nil {
		t.Fatal(err)
	}
	if len(endpoints) != 0 || len(findings) != 0 {
		t.Fatalf("expected 0, got %d endpoints, %d findings", len(endpoints), len(findings))
	}
}
