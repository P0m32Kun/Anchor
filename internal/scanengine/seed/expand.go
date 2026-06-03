package seed

import (
	"context"
	"log"
	"net"
	"strings"

	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/search"
)

// ExpandTargets converts project targets into engine seed values.
// Company targets are expanded via FOFA when credentials are configured.
func ExpandTargets(ctx context.Context, queries *db.Queries, targets []*models.Target) []string {
	var vals []string
	seen := make(map[string]bool)
	add := func(v string) {
		v = strings.TrimSpace(v)
		if v == "" || seen[v] {
			return
		}
		seen[v] = true
		vals = append(vals, v)
	}

	for _, t := range targets {
		if t == nil {
			continue
		}
		switch t.Type {
		case models.TargetTypeCompany:
			for _, s := range expandCompany(ctx, queries, t.Value) {
				add(s)
			}
		default:
			add(t.Value)
		}
	}
	return vals
}

func expandCompany(ctx context.Context, queries *db.Queries, company string) []string {
	cred, err := queries.GetEngineCredential("fofa")
	if err != nil || cred == nil || cred.APIKey == "" {
		log.Printf("[seed] skip company %q: FOFA credentials not configured", company)
		return nil
	}

	client := search.NewFofaClient(cred.APIKey)
	results, err := client.SearchCompany(ctx, company)
	if err != nil {
		log.Printf("[seed] FOFA company search %q: %v", company, err)
		return nil
	}

	var seeds []string
	for _, r := range results {
		if h := strings.TrimSpace(r.Host); h != "" {
			if strings.Contains(h, "://") {
				seeds = append(seeds, h)
			} else if strings.Contains(h, ":") && net.ParseIP(strings.Split(h, ":")[0]) != nil {
				seeds = append(seeds, h)
			} else {
				seeds = append(seeds, h)
			}
		}
		if ip := strings.TrimSpace(r.IP); ip != "" && net.ParseIP(ip) != nil {
			seeds = append(seeds, ip)
		}
	}
	log.Printf("[seed] FOFA expanded company %q into %d seeds", company, len(seeds))
	return seeds
}
