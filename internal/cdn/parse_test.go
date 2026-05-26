package cdn

import "testing"

func TestParseJSONLOutput_CDN(t *testing.T) {
	out := []byte(`{"input":"1.1.1.1","ip":"1.1.1.1","cdn":true,"cdn_name":"cloudflare"}
`)
	nonCDN, hits, err := ParseJSONLOutput(out, []string{"1.1.1.1", "2.2.2.2"})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].IP != "1.1.1.1" || hits[0].Provider != "cloudflare" || hits[0].Type != "cdn" {
		t.Fatalf("hits: %+v", hits)
	}
	if len(nonCDN) != 1 || nonCDN[0] != "2.2.2.2" {
		t.Fatalf("nonCDN: %v", nonCDN)
	}
}

func TestParseJSONLOutput_Cloud(t *testing.T) {
	out := []byte(`{"input":"159.75.160.229","ip":"159.75.160.229","cloud":true,"cloud_name":"tencent"}
`)
	nonCDN, hits, err := ParseJSONLOutput(out, []string{"159.75.160.229"})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].Type != "cloud" || hits[0].Provider != "tencent" {
		t.Fatalf("hits: %+v", hits)
	}
	if len(nonCDN) != 0 {
		t.Fatalf("nonCDN: %v", nonCDN)
	}
}

func TestParseJSONLOutput_EmptyStdout(t *testing.T) {
	nonCDN, hits, err := ParseJSONLOutput(nil, []string{"10.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 0 || len(nonCDN) != 1 {
		t.Fatalf("hits=%v nonCDN=%v", hits, nonCDN)
	}
}

func TestParseJSONLOutput_InvalidLine(t *testing.T) {
	_, _, err := ParseJSONLOutput([]byte("{not-json}\n"), []string{"1.1.1.1"})
	if err == nil {
		t.Fatal("expected parse error")
	}
}
