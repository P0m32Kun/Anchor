package seed

import (
	"context"
	"testing"

	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/search"
)

type stubCompanySearcher struct {
	name    string
	results []search.SearchResult
	err     error
	calls   int
}

func (s *stubCompanySearcher) engineName() string { return s.name }

func (s *stubCompanySearcher) searchCompany(ctx context.Context, company string, limit int) ([]search.SearchResult, error) {
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	out := s.results
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func TestExpandCompanyPassiveSearch_MergesEngines(t *testing.T) {
	fofa := &stubCompanySearcher{
		name: "fofa",
		results: []search.SearchResult{
			{Engine: "fofa", Domain: "a.example.com", IP: "198.51.100.1"},
		},
	}
	hunter := &stubCompanySearcher{
		name: "hunter",
		results: []search.SearchResult{
			{Engine: "hunter", Domain: "b.example.com", IP: "198.51.100.2"},
		},
	}
	quake := &stubCompanySearcher{
		name: "quake",
		results: []search.SearchResult{
			{Engine: "quake", Domain: "a.example.com", IP: "198.51.100.3"}, // duplicate domain
		},
	}

	cfg := models.DefaultExternalPipelineConfig()
	cfg.EnablePassiveSearch = true
	cfg.PassiveSearchResultLimit = 500

	seeds := expandCompanyPassiveSearch(context.Background(), cfg, "TestCorp", "tgt-1", fofa, hunter, quake)
	if len(seeds) < 3 {
		t.Fatalf("expected at least 3 seeds, got %d: %+v", len(seeds), seeds)
	}
	if fofa.calls != 1 || hunter.calls != 1 || quake.calls != 1 {
		t.Fatalf("expected each engine called once: fofa=%d hunter=%d quake=%d", fofa.calls, hunter.calls, quake.calls)
	}

	seen := map[string]bool{}
	for _, s := range seeds {
		key := s.ValueType + "|" + s.Value
		if seen[key] {
			t.Fatalf("duplicate seed: %+v", s)
		}
		seen[key] = true
		if s.SourceRef != "tgt-1" {
			t.Fatalf("SourceRef = %q", s.SourceRef)
		}
	}
}

func TestExpandCompanyPassiveSearch_FailSoft(t *testing.T) {
	fofa := &stubCompanySearcher{name: "fofa", err: context.DeadlineExceeded}
	hunter := &stubCompanySearcher{
		name:    "hunter",
		results: []search.SearchResult{{Engine: "hunter", Domain: "ok.example.com"}},
	}

	cfg := models.DefaultExternalPipelineConfig()
	cfg.EnablePassiveSearch = true

	seeds := expandCompanyPassiveSearch(context.Background(), cfg, "TestCorp", "tgt-1", fofa, hunter)
	if len(seeds) == 0 {
		t.Fatal("expected hunter results despite fofa failure")
	}
}

func TestExpandCompanyPassiveSearch_Disabled(t *testing.T) {
	fofa := &stubCompanySearcher{
		name:    "fofa",
		results: []search.SearchResult{{Domain: "a.example.com"}},
	}
	cfg := models.DefaultExternalPipelineConfig()
	cfg.EnablePassiveSearch = false

	seeds := expandCompanyPassiveSearch(context.Background(), cfg, "TestCorp", "tgt-1", fofa)
	if len(seeds) != 0 {
		t.Fatalf("expected no seeds when disabled, got %v", seeds)
	}
	if fofa.calls != 0 {
		t.Fatalf("engine should not be called when disabled")
	}
}

func TestExpandCompanyPassiveSearch_LimitPerEngine(t *testing.T) {
	fofa := &stubCompanySearcher{
		name: "fofa",
		results: []search.SearchResult{
			{Domain: "a.example.com"},
			{Domain: "b.example.com"},
			{Domain: "c.example.com"},
		},
	}

	cfg := models.DefaultExternalPipelineConfig()
	cfg.EnablePassiveSearch = true
	cfg.PassiveSearchResultLimit = 2

	seeds := expandCompanyPassiveSearch(context.Background(), cfg, "TestCorp", "tgt-1", fofa)
	domains := 0
	for _, s := range seeds {
		if s.ValueType == "domain" {
			domains++
		}
	}
	if domains > 2 {
		t.Fatalf("expected at most 2 domain seeds, got %d", domains)
	}
}

func TestSeedsFromSearchResult(t *testing.T) {
	raw := &search.HunterResult{URL: "https://app.example.com/login"}
	r := search.SearchResult{
		Engine: "hunter",
		Domain: "app.example.com",
		IP:     "198.51.100.10",
		Raw:    raw,
	}
	seeds := seedsFromSearchResult(r, "hunter", "tgt-1")
	types := map[string]bool{}
	for _, s := range seeds {
		types[s.ValueType] = true
	}
	if !types["url"] || !types["domain"] || !types["ip"] {
		t.Fatalf("expected url/domain/ip seeds, got %+v", seeds)
	}
}
