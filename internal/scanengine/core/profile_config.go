package core

import "github.com/P0m32Kun/Anchor/internal/models"

// ProfileFromConfig builds a scan profile for the given mode with PipelineConfig
// toggles applied to each action rule.
func ProfileFromConfig(mode string, cfg models.PipelineConfig) Profile {
	var base Profile
	switch mode {
	case "external":
		ext := DefaultExternalProfile()
		ext.SkipPortscanOnCDN = cfg.SkipPortscanOnCDNHost
		base = ext
	default:
		base = DefaultInternalProfile()
	}
	return &pipelineProfile{base: base, cfg: cfg}
}

type pipelineProfile struct {
	base Profile
	cfg  models.PipelineConfig
}

func (p *pipelineProfile) RequireFingerprint() bool {
	if !p.cfg.EnableNuclei {
		return p.base.RequireFingerprint()
	}
	return p.cfg.NucleiRequireFingerprint
}

func (p *pipelineProfile) Rules() []ActionRule {
	rules := p.base.Rules()
	out := make([]ActionRule, len(rules))
	for i, r := range rules {
		out[i] = r
		out[i].Enabled = r.Enabled && actionEnabledByConfig(p.cfg, r.Action)
	}
	return out
}

func actionEnabledByConfig(cfg models.PipelineConfig, action TaskAction) bool {
	switch action {
	case ActionSubdomainEnum:
		return cfg.EnableSubfinder
	case ActionDNSResolve:
		return cfg.EnableDNSx
	case ActionCDNCheck:
		return cfg.EnableCDNFilter
	case ActionPortScan:
		return true // naabu; port scan always eligible when rule precondition passes
	case ActionServiceFingerprint:
		return cfg.EnableNmapService
	case ActionHTTPXFingerprint:
		return cfg.EnableHttpx
	case ActionNucleiScan:
		return cfg.EnableNuclei
	case ActionKatanaCrawl:
		return cfg.EnableKatana
	case ActionFFUFBrute:
		return cfg.EnableFfuf
	case ActionPassiveSearch, ActionPassiveCert, ActionPassiveURL:
		// Passive discovery is handled by seed injectors / ExpandTargets, not executor work items.
		return false
	case ActionSpoorScan:
		return cfg.EnableKatana // spoor follows katana enable for external surface crawl
	default:
		return true
	}
}
