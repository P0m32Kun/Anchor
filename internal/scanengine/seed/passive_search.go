package seed

import (
	"context"
	"log"
	"net"
	"strings"
	"sync"

	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/search"
)

type companySearcher interface {
	engineName() string
	searchCompany(ctx context.Context, company string, limit int) ([]search.SearchResult, error)
}

type fofaCompanySearcher struct {
	client *search.FofaClient
}

func (f fofaCompanySearcher) engineName() string { return "fofa" }

func (f fofaCompanySearcher) searchCompany(ctx context.Context, company string, limit int) ([]search.SearchResult, error) {
	queries := passiveFofaQueries(company)
	seen := make(map[string]bool)
	var all []search.SearchResult
	for _, q := range queries {
		if limit > 0 && len(all) >= limit {
			break
		}
		batchSize := limit
		if batchSize <= 0 {
			batchSize = 500
		}
		results, err := f.client.Search(ctx, q, 1, batchSize)
		if err != nil {
			continue
		}
		for _, r := range results {
			key := r.Domain + "|" + r.IP
			if seen[key] {
				continue
			}
			seen[key] = true
			all = append(all, r)
			if limit > 0 && len(all) >= limit {
				break
			}
		}
	}
	return all, nil
}

type hunterCompanySearcher struct {
	client *search.HunterClient
}

func (h hunterCompanySearcher) engineName() string { return "hunter" }

func (h hunterCompanySearcher) searchCompany(ctx context.Context, company string, limit int) ([]search.SearchResult, error) {
	queries := passiveHunterQueries(company)
	seen := make(map[string]bool)
	var all []search.SearchResult
	for _, q := range queries {
		if limit > 0 && len(all) >= limit {
			break
		}
		batchSize := limit
		if batchSize <= 0 {
			batchSize = 100
		}
		if batchSize > 100 {
			batchSize = 100
		}
		results, err := h.client.Search(ctx, q, 1, batchSize)
		if err != nil {
			continue
		}
		for _, r := range results {
			key := r.Domain + "|" + r.IP
			if seen[key] {
				continue
			}
			seen[key] = true
			all = append(all, r)
			if limit > 0 && len(all) >= limit {
				break
			}
		}
	}
	return all, nil
}

type quakeCompanySearcher struct {
	client *search.QuakeClient
}

func (q quakeCompanySearcher) engineName() string { return "quake" }

func (q quakeCompanySearcher) searchCompany(ctx context.Context, company string, limit int) ([]search.SearchResult, error) {
	query := passiveQuakeQuery(company)
	size := limit
	if size <= 0 {
		size = 500
	}
	return q.client.Search(ctx, query, 0, size)
}

func buildCompanySearchers(queries *db.Queries) []companySearcher {
	var runners []companySearcher
	if cred, err := queries.GetEngineCredential("fofa"); err == nil && cred != nil && cred.APIKey != "" {
		runners = append(runners, fofaCompanySearcher{client: search.NewFofaClient(cred.APIKey)})
	}
	if cred, err := queries.GetEngineCredential("hunter"); err == nil && cred != nil && cred.APIKey != "" {
		runners = append(runners, hunterCompanySearcher{client: search.NewHunterClient(cred.APIKey)})
	}
	if cred, err := queries.GetEngineCredential("quake"); err == nil && cred != nil && cred.APIKey != "" {
		runners = append(runners, quakeCompanySearcher{client: search.NewQuakeClient(cred.APIKey)})
	}
	return runners
}

func expandCompanyPassiveSearch(ctx context.Context, cfg models.PipelineConfig, company, sourceRef string, runners ...companySearcher) []SeedAsset {
	if !cfg.EnablePassiveSearch {
		log.Printf("[seed] skip company %q: passive search disabled", company)
		return nil
	}
	if len(runners) == 0 {
		log.Printf("[seed] skip company %q: no search engine credentials configured", company)
		return nil
	}

	limit := cfg.PassiveSearchResultLimit
	if limit <= 0 {
		limit = passiveConfig().ResultLimit
	}
	if limit <= 0 {
		limit = 500
	}
	concurrency := cfg.PassiveSearchConcurrency
	if concurrency <= 0 {
		concurrency = passiveConfig().Concurrency
	}
	if concurrency <= 0 {
		concurrency = 3
	}
	sem := make(chan struct{}, concurrency)

	var (
		mu     sync.Mutex
		merged []SeedAsset
		seen   = make(map[string]bool)
		wg     sync.WaitGroup
	)

	for _, runner := range runners {
		wg.Add(1)
		go func(r companySearcher) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			results, err := r.searchCompany(ctx, company, limit)
			if err != nil {
				log.Printf("[seed] %s company search %q: %v", r.engineName(), company, err)
				return
			}
			local := make([]SeedAsset, 0, len(results))
			for _, res := range results {
				for _, s := range seedsFromSearchResult(res, r.engineName(), sourceRef) {
					key := s.ValueType + "|" + normalizeSeedValue(s.Value)
					mu.Lock()
					if seen[key] {
						mu.Unlock()
						continue
					}
					seen[key] = true
					mu.Unlock()
					local = append(local, s)
				}
			}
			mu.Lock()
			merged = append(merged, local...)
			mu.Unlock()
			log.Printf("[seed] %s expanded company %q into %d seeds", r.engineName(), company, len(local))
		}(runner)
	}
	wg.Wait()
	return merged
}

func seedsFromSearchResult(r search.SearchResult, sourceEngine, sourceRef string) []SeedAsset {
	raw := r
	var out []SeedAsset
	add := func(value, valueType string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		out = append(out, SeedAsset{
			Value:     value,
			ValueType: valueType,
			Source:    sourceEngine,
			SourceRef: sourceRef,
			Raw:       &raw,
		})
	}

	if r.Raw != nil {
		if hr, ok := r.Raw.(*search.HunterResult); ok {
			add(hr.URL, "url")
		}
	}

	domain := strings.TrimSpace(r.Domain)
	switch {
	case strings.HasPrefix(domain, "http://") || strings.HasPrefix(domain, "https://"):
		add(domain, "url")
	case domain != "":
		add(domain, "domain")
	}

	if ip := strings.TrimSpace(r.IP); ip != "" && net.ParseIP(ip) != nil {
		add(ip, "ip")
	}
	return out
}

func normalizeSeedValue(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}
