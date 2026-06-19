package seed

import (
	"context"
	"testing"

	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/search"
)

func TestFilterPassiveSeeds_DropsJunkTitle(t *testing.T) {
	raw := search.SearchResult{Title: "澳门博彩官方平台", Domain: "evil.example.com", IP: "198.51.100.1"}
	seeds := []SeedAsset{{
		Value: "evil.example.com", ValueType: "domain", Source: "fofa", Raw: &raw,
	}}
	cfg := models.DefaultExternalPipelineConfig()

	out, stats := FilterPassiveSeeds(seeds, cfg)
	if len(out) != 0 {
		t.Fatalf("expected junk seed dropped, got %+v", out)
	}
	if stats.Dropped != 1 {
		t.Fatalf("Dropped = %d, want 1", stats.Dropped)
	}
}

func TestFilterPassiveSeeds_KeepsLegitimateDomain(t *testing.T) {
	raw := search.SearchResult{Title: "理想汽车", Domain: "account.lixiang.com", IP: "198.51.100.1"}
	seeds := []SeedAsset{{
		Value: "account.lixiang.com", ValueType: "domain", Source: "hunter", Raw: &raw,
	}}
	cfg := models.DefaultExternalPipelineConfig()

	out, stats := FilterPassiveSeeds(seeds, cfg)
	if len(out) != 1 {
		t.Fatalf("expected seed kept, got %+v stats=%+v", out, stats)
	}
}

func TestFilterPassiveSeeds_KeepsBareIPWithoutJunkTitle(t *testing.T) {
	raw := search.SearchResult{Domain: "", IP: "198.51.100.50"}
	seeds := []SeedAsset{{
		Value: "198.51.100.50", ValueType: "ip", Source: "quake", Raw: &raw,
	}}
	cfg := models.DefaultExternalPipelineConfig()

	out, _ := FilterPassiveSeeds(seeds, cfg)
	if len(out) != 1 {
		t.Fatalf("expected bare IP kept, got %+v", out)
	}
}

func TestFilterPassiveSeeds_DropsIPWhenTitleJunk(t *testing.T) {
	raw := search.SearchResult{Title: "在线赌场", Domain: "", IP: "198.51.100.50"}
	seeds := []SeedAsset{{
		Value: "198.51.100.50", ValueType: "ip", Source: "fofa", Raw: &raw,
	}}
	cfg := models.DefaultExternalPipelineConfig()

	out, _ := FilterPassiveSeeds(seeds, cfg)
	if len(out) != 0 {
		t.Fatalf("expected IP dropped due to junk title, got %+v", out)
	}
}

func TestFilterPassiveSeeds_KeepsManualTarget(t *testing.T) {
	seeds := []SeedAsset{{
		Value: "junk-bocai.example.com", ValueType: "domain", Source: "target",
	}}
	cfg := models.DefaultExternalPipelineConfig()

	out, _ := FilterPassiveSeeds(seeds, cfg)
	if len(out) != 1 {
		t.Fatalf("manual targets should not be filtered, got %+v", out)
	}
}

func TestFilterPassiveSeeds_Disabled(t *testing.T) {
	raw := search.SearchResult{Title: "博彩平台", Domain: "evil.example.com"}
	seeds := []SeedAsset{{
		Value: "evil.example.com", ValueType: "domain", Source: "fofa", Raw: &raw,
	}}
	cfg := models.DefaultExternalPipelineConfig()
	cfg.EnablePassiveJunkFilter = false

	out, stats := FilterPassiveSeeds(seeds, cfg)
	if len(out) != 1 || stats.Dropped != 0 {
		t.Fatalf("filter disabled should keep seed, out=%+v stats=%+v", out, stats)
	}
}

func TestFilterPassiveSeeds_CustomKeyword(t *testing.T) {
	raw := search.SearchResult{Title: "正常业务系统", Domain: "foo.example.com"}
	seeds := []SeedAsset{{
		Value: "foo.example.com", ValueType: "domain", Source: "fofa", Raw: &raw,
	}}
	cfg := models.DefaultExternalPipelineConfig()
	cfg.PassiveJunkKeywords = "foo.example.com"

	out, _ := FilterPassiveSeeds(seeds, cfg)
	if len(out) != 0 {
		t.Fatalf("expected custom keyword to drop seed, got %+v", out)
	}
}

func TestExpandTargets_AppliesJunkFilter(t *testing.T) {
	fofa := &stubCompanySearcher{
		name: "fofa",
		results: []search.SearchResult{
			{Engine: "fofa", Domain: "good.example.com", Title: "Corp Portal"},
			{Engine: "fofa", Domain: "bad.example.com", Title: "博彩娱乐城"},
		},
	}
	cfg := models.DefaultExternalPipelineConfig()
	cfg.EnablePassiveSearch = true

	seeds := expandCompanyPassiveSearch(context.Background(), cfg, "TestCorp", "t1", fofa)
	filtered, stats := FilterPassiveSeeds(seeds, cfg)
	if stats.Dropped == 0 {
		t.Fatalf("expected junk filter to drop at least one seed, stats=%+v seeds=%+v", stats, filtered)
	}
	for _, s := range filtered {
		if s.Raw != nil && s.Raw.Title != "" {
			if field, _ := matchSeedJunk(s, activeJunkKeywords(nil)); field != "" {
				t.Fatalf("junk seed survived filter: %+v", s)
			}
		}
	}
}
