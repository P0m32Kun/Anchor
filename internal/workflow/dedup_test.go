package workflow

import (
	"testing"

	"github.com/P0m32Kun/Anchor/internal/models"
)

func TestScanOrigin(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"https://1.2.3.4:8080/admin", "1.2.3.4:8080"},
		{"http://example.com/path", "example.com:80"},
		{"https://example.com/", "example.com:443"},
		{"192.168.1.1:6379", "192.168.1.1:6379"},
		{"https://example.com:443/", "example.com:443"},
	}
	for _, tt := range tests {
		if got := scanOrigin(tt.in); got != tt.want {
			t.Errorf("scanOrigin(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestComputeDedupKey_SameOriginDifferentPath(t *testing.T) {
	k1 := computeDedupKey("tpl-1", "https://1.2.3.4:8080/admin", "matcher")
	k2 := computeDedupKey("tpl-1", "https://1.2.3.4:8080/login", "matcher")
	if k1 != k2 {
		t.Fatalf("expected same dedup key for same origin, got %q vs %q", k1, k2)
	}
}

func TestComputeDedupKey_DifferentMatcher(t *testing.T) {
	k1 := computeDedupKey("tpl-1", "https://1.2.3.4:8080/", "a")
	k2 := computeDedupKey("tpl-1", "https://1.2.3.4:8080/", "b")
	if k1 == k2 {
		t.Fatal("expected different dedup keys for different matchers")
	}
}

func TestFilterURLsForSecondaryScan(t *testing.T) {
	existing := []*models.WebEndpoint{
		{URL: "https://10.0.0.1:8080/"},
	}
	in := []string{
		"https://10.0.0.1:8080/admin",
		"https://10.0.0.1:8080/",
		"https://10.0.0.2:8080/",
		"https://10.0.0.2:8080/login",
	}
	got := filterURLsForSecondaryScan(in, existing)
	if len(got) != 1 || got[0] != "https://10.0.0.2:8080/" {
		t.Fatalf("filterURLsForSecondaryScan = %#v, want single new origin URL", got)
	}
}

func TestDedupHTTPTargetsByOrigin(t *testing.T) {
	in := []string{
		"http://1.2.3.4:80",
		"https://1.2.3.4:443",
		"1.2.3.4:8080",
	}
	got := dedupHTTPTargetsByOrigin(in)
	if len(got) != 3 {
		t.Fatalf("dedupHTTPTargetsByOrigin len = %d, want 3 distinct origins", len(got))
	}
}
