package seed

import (
	"testing"

	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/scope"
)

func TestFilterSeedsByBoundary(t *testing.T) {
	eng := &scope.Engine{}

	t.Run("off mode passes all seeds", func(t *testing.T) {
		seeds := []SeedAsset{
			{Value: "evil.com", ValueType: "domain"},
			{Value: "api.example.com", ValueType: "domain"},
		}
		rules := []*models.ScopeRule{
			{Action: models.ScopeActionInclude, Type: models.TargetTypeDomain, Value: "*.example.com"},
		}
		got := FilterSeedsByBoundary(seeds, eng, rules, models.ScopeBoundaryOff)
		if len(got) != 2 {
			t.Errorf("off mode should pass all seeds, got %d", len(got))
		}
	})

	t.Run("strict mode filters non-matching domains", func(t *testing.T) {
		seeds := []SeedAsset{
			{Value: "evil.com", ValueType: "domain"},
			{Value: "api.example.com", ValueType: "domain"},
		}
		rules := []*models.ScopeRule{
			{Action: models.ScopeActionInclude, Type: models.TargetTypeDomain, Value: "*.example.com"},
		}
		got := FilterSeedsByBoundary(seeds, eng, rules, models.ScopeBoundaryStrict)
		if len(got) != 1 {
			t.Fatalf("expected 1 seed, got %d", len(got))
		}
		if got[0].Value != "api.example.com" {
			t.Errorf("expected api.example.com, got %s", got[0].Value)
		}
	})

	t.Run("strict mode exclude takes priority", func(t *testing.T) {
		seeds := []SeedAsset{
			{Value: "staging.example.com", ValueType: "domain"},
			{Value: "api.example.com", ValueType: "domain"},
		}
		rules := []*models.ScopeRule{
			{Action: models.ScopeActionInclude, Type: models.TargetTypeDomain, Value: "*.example.com"},
			{Action: models.ScopeActionExclude, Type: models.TargetTypeDomain, Value: "staging.example.com"},
		}
		got := FilterSeedsByBoundary(seeds, eng, rules, models.ScopeBoundaryStrict)
		if len(got) != 1 {
			t.Fatalf("expected 1 seed, got %d", len(got))
		}
		if got[0].Value != "api.example.com" {
			t.Errorf("expected api.example.com, got %s", got[0].Value)
		}
	})

	t.Run("strict mode IP in CIDR include", func(t *testing.T) {
		seeds := []SeedAsset{
			{Value: "10.0.0.5", ValueType: "ip"},
			{Value: "192.168.1.1", ValueType: "ip"},
		}
		rules := []*models.ScopeRule{
			{Action: models.ScopeActionInclude, Type: models.TargetTypeCIDR, Value: "10.0.0.0/24"},
		}
		got := FilterSeedsByBoundary(seeds, eng, rules, models.ScopeBoundaryStrict)
		if len(got) != 1 {
			t.Fatalf("expected 1 seed, got %d", len(got))
		}
		if got[0].Value != "10.0.0.5" {
			t.Errorf("expected 10.0.0.5, got %s", got[0].Value)
		}
	})

	t.Run("strict mode no include rules passes all non-excluded", func(t *testing.T) {
		seeds := []SeedAsset{
			{Value: "anything.com", ValueType: "domain"},
		}
		rules := []*models.ScopeRule{
			{Action: models.ScopeActionExclude, Type: models.TargetTypeDomain, Value: "blocked.com"},
		}
		got := FilterSeedsByBoundary(seeds, eng, rules, models.ScopeBoundaryStrict)
		if len(got) != 1 {
			t.Errorf("no include rules + no exclude match should pass, got %d", len(got))
		}
	})

	t.Run("empty mode defaults to off", func(t *testing.T) {
		seeds := []SeedAsset{
			{Value: "evil.com", ValueType: "domain"},
		}
		rules := []*models.ScopeRule{
			{Action: models.ScopeActionInclude, Type: models.TargetTypeDomain, Value: "*.example.com"},
		}
		got := FilterSeedsByBoundary(seeds, eng, rules, "")
		if len(got) != 1 {
			t.Errorf("empty mode should default to off (pass all), got %d", len(got))
		}
	})
}
