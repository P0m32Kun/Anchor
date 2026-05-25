package workflow

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/passive"
	"github.com/P0m32Kun/Anchor/internal/search"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// runPassiveSearch runs the passive search phase (P1) for a company name.
// It fans out across FOFA, Hunter, and Quake (when credentials are available
// and the respective feature flags are on), collecting domain and IP results.
func (p *Pipeline) runPassiveSearch(ctx context.Context, companyName string) error {
	p.setStage(StageSearch)

	var allDomains, allIPs []string
	var errs []error

	// 1. FOFA
	if p.config.EnablePassiveSearch && p.config.EnableFOFA && p.fofa != nil {
		domains, ips, err := p.fofaExpandCompany(ctx, companyName)
		if err != nil {
			log.Printf("[passive] fofa: %v", err)
			errs = append(errs, fmt.Errorf("fofa: %w", err))
		} else {
			allDomains = append(allDomains, domains...)
			allIPs = append(allIPs, ips...)
		}
	}

	// 2. Hunter
	if p.config.EnablePassiveSearch {
		domains, ips, err := p.hunterSearchCompany(ctx, companyName)
		if err != nil {
			log.Printf("[passive] hunter: %v", err)
			errs = append(errs, fmt.Errorf("hunter: %w", err))
		} else {
			allDomains = append(allDomains, domains...)
			allIPs = append(allIPs, ips...)
		}
	}

	allDomains = dedupStrings(allDomains)
	allIPs = dedupStrings(allIPs)

	log.Printf("[passive] company %q: FOFA+Hunter found %d domains, %d IPs",
		companyName, len(allDomains), len(allIPs))

	// Persist as targets and route to downstream flows.
	var flowErr error
	if len(allDomains) > 0 {
		if err := p.runDomainFlow(ctx, makeTargets(allDomains, models.TargetTypeDomain)); err != nil {
			log.Printf("domain flow from company: %v", err)
			flowErr = err
		}
	}
	if len(allIPs) > 0 {
		if err := p.runIPFlow(ctx, makeTargets(allIPs, models.TargetTypeIP)); err != nil {
			log.Printf("ip flow from company: %v", err)
			if flowErr == nil {
				flowErr = err
			}
		}
	}

	if len(errs) > 0 && allDomains == nil && allIPs == nil {
		// All engines failed and nothing was found.
		p.failStage(StageSearch, fmt.Sprintf("all passive search engines failed: %v", errs))
		return fmt.Errorf("passive search: %v", errs)
	}
	p.completeStage(StageSearch)
	return flowErr
}

// fofaExpandCompany runs a FOFA company search and returns deduplicated
// domain and IP string slices. The targets are already persisted to the
// database with source="fofa".
func (p *Pipeline) fofaExpandCompany(ctx context.Context, name string) (domains, ips []string, _ error) {
	if p.fofa == nil {
		return nil, nil, nil
	}
	results, err := p.fofa.SearchCompany(ctx, name)
	if err != nil {
		return nil, nil, err
	}
	for _, r := range results {
		if search.IsDomain(r.Host) {
			domains = append(domains, r.Host)
		}
		if r.IP != "" {
			ips = append(ips, r.IP)
		}
	}
	domains = dedupStrings(domains)
	ips = dedupStrings(ips)
	p.persistSearchResults(domains, ips, "fofa")
	return domains, ips, nil
}

// hunterSearchCompany loads the Hunter credential from the database and,
// if configured, runs a company search returning domains and IPs.
func (p *Pipeline) hunterSearchCompany(ctx context.Context, name string) (domains, ips []string, _ error) {
	cred, err := p.queries.GetEngineCredential("hunter")
	if err != nil {
		log.Printf("[passive] hunter credential lookup: %v", err)
		return nil, nil, nil // fail-soft: no credential = skip
	}
	if cred == nil || cred.APIKey == "" {
		return nil, nil, nil
	}
	client := search.NewHunterClient(cred.APIKey)
	// Hunter's company search: use the org name as the query.
	// The free-tier API limits paging; we request one page of results.
	results, err := client.Search(ctx, fmt.Sprintf(`org="%s"`, name), 1, p.config.PassiveSearchResultLimit)
	if err != nil {
		return nil, nil, fmt.Errorf("hunter search: %w", err)
	}
	for _, r := range results {
		if search.IsDomain(r.Domain) {
			domains = append(domains, r.Domain)
		}
		if r.IP != "" {
			ips = append(ips, r.IP)
		}
	}
	domains = dedupStrings(domains)
	ips = dedupStrings(ips)
	p.persistSearchResults(domains, ips, "hunter")
	return domains, ips, nil
}

// persistSearchResults saves domain and IP strings as Target rows with the
// given source label, skipping duplicates by (project_id, value).
func (p *Pipeline) persistSearchResults(domains, ips []string, source string) {
	now := time.Now().UTC()
	for _, d := range domains {
		target := &models.Target{
			ID:        util.GenerateID(),
			ProjectID: p.projectID,
			Type:      models.TargetTypeDomain,
			Value:     d,
			Source:    source,
			Status:    "active",
			CreatedAt: now,
		}
		if err := p.queries.CreateTarget(target); err != nil {
			log.Printf("[passive] save %s domain target: %v", source, err)
		}
	}
	for _, ip := range ips {
		target := &models.Target{
			ID:        util.GenerateID(),
			ProjectID: p.projectID,
			Type:      models.TargetTypeIP,
			Value:     ip,
			Source:    source,
			Status:    "active",
			CreatedAt: now,
		}
		if err := p.queries.CreateTarget(target); err != nil {
			log.Printf("[passive] save %s ip target: %v", source, err)
		}
	}
}

// runPassiveCert queries crt.sh for SSL certificate transparency subdomains
// and persists discovered domains as targets with source="crt".
func (p *Pipeline) runPassiveCert(ctx context.Context, rootDomain string) error {
	if !p.config.EnablePassiveCert {
		return nil
	}
	p.setStage(StagePassiveCert)

	subs, err := passive.FetchSubdomains(ctx, rootDomain)
	if err != nil {
		log.Printf("[passive] crt.sh for %s: %v", rootDomain, err)
		p.failStage(StagePassiveCert, err.Error())
		return err
	}
	if len(subs) == 0 {
		p.completeStage(StagePassiveCert)
		return nil
	}

	// Filter discovered subdomains through scope and persist.
	now := time.Now().UTC()
	var persisted int
	for _, s := range subs {
		target := &models.Target{
			ID:        util.GenerateID(),
			ProjectID: p.projectID,
			Type:      models.TargetTypeDomain,
			Value:     s,
			Source:    "crt",
			Status:    "active",
			CreatedAt: now,
		}
		// Skip if not in scope
		decision, err := p.scope.ValidateBeforeRun(ctx, p.projectID, target, "")
		if err != nil || decision.Decision == models.ScopeDeny {
			continue
		}
		if err := p.queries.CreateTarget(target); err != nil {
			log.Printf("[passive] save crt domain %s: %v", s, err)
			continue
		}
		persisted++
	}
	log.Printf("[passive] crt.sh: %d/%d subdomains in scope for %s", persisted, len(subs), rootDomain)
	p.completeStage(StagePassiveCert)
	return nil
}

// runPassiveURL runs gau to collect historical URLs for the given domain
// and stores them temporarily for the URL discovery stage.
func (p *Pipeline) runPassiveURL(ctx context.Context, rootDomain string) ([]string, error) {
	if !p.config.EnablePassiveURL {
		return nil, nil
	}
	p.setStage(StagePassiveURL)

	urls, err := passive.RunGau(ctx, rootDomain)
	if err != nil {
		log.Printf("[passive] gau for %s: %v", rootDomain, err)
		p.failStage(StagePassiveURL, err.Error())
		return nil, err
	}
	log.Printf("[passive] gau: collected %d URLs for %s", len(urls), rootDomain)
	p.completeStage(StagePassiveURL)
	return urls, nil
}
