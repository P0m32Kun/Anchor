package seed

import (
	"context"
	"log"

	"github.com/P0m32Kun/Anchor/internal/passive"
	"github.com/P0m32Kun/Anchor/internal/scanengine/core"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// AssetProcessor is the callback signature for processing newly discovered assets.
type AssetProcessor func(ctx context.Context, a *core.DiscoveryAsset)

// PassiveInjector injects passive discovery results as seed assets.
type PassiveInjector struct {
	processor AssetProcessor
}

// NewPassiveInjector creates a new PassiveInjector.
func NewPassiveInjector(processor AssetProcessor) *PassiveInjector {
	return &PassiveInjector{processor: processor}
}

// InjectCrt queries crt.sh for subdomains and injects them as assets.
func (p *PassiveInjector) InjectCrt(ctx context.Context, domain string, projectID string) {
	subs, err := passive.FetchSubdomains(ctx, domain)
	if err != nil {
		log.Printf("[seed] crt.sh error for %s: %v", domain, err)
		return
	}
	for _, sub := range subs {
		p.processor(ctx, &core.DiscoveryAsset{
			ID:              util.GenerateID(),
			Type:            core.AssetSubdomain,
			Value:           sub,
			NormalizedValue: sub,
			DiscoveryDepth:  0,
			SourceTool:      "crt",
		})
	}
	log.Printf("[seed] crt.sh injected %d subdomains for %s", len(subs), domain)
}

// InjectFromTargets injects initial targets as seed assets.
func (p *PassiveInjector) InjectFromTargets(ctx context.Context, targets []string, projectID string) {
	for _, target := range targets {
		p.processor(ctx, &core.DiscoveryAsset{
			ID:              util.GenerateID(),
			Type:            core.ClassifySeedTarget(target),
			Value:           target,
			NormalizedValue: target,
			DiscoveryDepth:  0,
			SourceTool:      "seed",
		})
	}
}
