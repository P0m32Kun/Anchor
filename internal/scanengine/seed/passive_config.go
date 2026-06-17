package seed

import "strings"

type PassiveRuntimeConfig struct {
	ResultLimit   int
	Concurrency   int
	FofaQueries   []string
	HunterQueries []string
	QuakeQuery    string
}

var runtimePassive PassiveRuntimeConfig

func SetPassiveConfig(cfg PassiveRuntimeConfig) {
	if cfg.ResultLimit > 0 {
		runtimePassive.ResultLimit = cfg.ResultLimit
	}
	if cfg.Concurrency > 0 {
		runtimePassive.Concurrency = cfg.Concurrency
	}
	if len(cfg.FofaQueries) > 0 {
		runtimePassive.FofaQueries = append([]string(nil), cfg.FofaQueries...)
	}
	if len(cfg.HunterQueries) > 0 {
		runtimePassive.HunterQueries = append([]string(nil), cfg.HunterQueries...)
	}
	if cfg.QuakeQuery != "" {
		runtimePassive.QuakeQuery = cfg.QuakeQuery
	}
}

func passiveFofaQueries(company string) []string {
	return expandQueryTemplates(passiveConfig().FofaQueries, company)
}

func passiveHunterQueries(company string) []string {
	return expandQueryTemplates(passiveConfig().HunterQueries, company)
}

func passiveQuakeQuery(company string) string {
	q := passiveConfig().QuakeQuery
	if q == "" {
		return ""
	}
	return strings.ReplaceAll(q, "{{company}}", company)
}

func passiveConfig() PassiveRuntimeConfig {
	if runtimePassive.ResultLimit > 0 || len(runtimePassive.FofaQueries) > 0 {
		return runtimePassive
	}
	return PassiveRuntimeConfig{
		ResultLimit:   500,
		Concurrency:   3,
		FofaQueries:   []string{`org="{{company}}"`, `cert="{{company}}"`, `title="{{company}}"`},
		HunterQueries: []string{`icp.name="{{company}}"`, `cert="{{company}}"`},
		QuakeQuery:    `cert:"{{company}}" OR title:"{{company}}"`,
	}
}

func expandQueryTemplates(templates []string, company string) []string {
	out := make([]string, 0, len(templates))
	for _, tpl := range templates {
		tpl = strings.TrimSpace(tpl)
		if tpl == "" {
			continue
		}
		out = append(out, strings.ReplaceAll(tpl, "{{company}}", company))
	}
	return out
}
