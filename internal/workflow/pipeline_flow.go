package workflow

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/P0m32Kun/Anchor/internal/fingerprint"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/parser"
	"github.com/P0m32Kun/Anchor/internal/resolve"
	"github.com/P0m32Kun/Anchor/internal/search"
	"github.com/P0m32Kun/Anchor/internal/util"
)

type targetGroup struct {
	Type    models.TargetType
	Targets []*models.Target
}

func groupTargetsByType(targets []*models.Target) []targetGroup {
	m := make(map[models.TargetType][]*models.Target)
	for _, t := range targets {
		m[t.Type] = append(m[t.Type], t)
	}
	var groups []targetGroup
	for typ, list := range m {
		groups = append(groups, targetGroup{Type: typ, Targets: list})
	}
	return groups
}

func (p *Pipeline) runFlow(ctx context.Context, group targetGroup) error {
	switch group.Type {
	case models.TargetTypeCompany:
		return p.runCompanyFlow(ctx, group.Targets)
	case models.TargetTypeDomain:
		return p.runDomainFlow(ctx, group.Targets)
	case models.TargetTypeIP:
		return p.runIPFlow(ctx, group.Targets)
	case models.TargetTypeCIDR:
		// CIDR targets are expanded to atomic IPs by scope.FilterTargets at
		// the pipeline entry, so this case should never fire in normal
		// operation. Treat any CIDR that makes it here as a programming
		// error (someone bypassed FilterTargets) and surface it loudly.
		return fmt.Errorf("unexpected CIDR target at runFlow — should have been expanded by scope.FilterTargets (project=%s)", p.projectID)
	case models.TargetTypeURL:
		return p.runURLFlow(ctx, group.Targets)
	default:
		return fmt.Errorf("unsupported target type: %s", group.Type)
	}
}

// runCompanyFlow: FOFA search → expand to domains/ips → route to respective flows.
func (p *Pipeline) runCompanyFlow(ctx context.Context, targets []*models.Target) error {
	p.setStage(StageSearch)

	if !p.config.EnableFOFA || p.fofa == nil {
		log.Printf("FOFA disabled or not configured, skipping company targets")
		p.completeStage(StageSearch)
		return nil
	}

	var flowErr error
	for _, t := range targets {
		results, err := p.fofa.SearchCompany(ctx, t.Value)
		if err != nil {
			log.Printf("fofa search for %s: %v", t.Value, err)
			p.failStage(StageSearch, err.Error())
			continue
		}

		var domains, ips []string
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

		log.Printf("company %s: found %d domains, %d ips from FOFA", t.Value, len(domains), len(ips))

		for _, d := range domains {
			target := &models.Target{
				ID:        util.GenerateID(),
				ProjectID: p.projectID,
				Type:      models.TargetTypeDomain,
				Value:     d,
				Source:    "fofa",
				Status:    "active",
				CreatedAt: time.Now().UTC(),
			}
			if err := p.queries.CreateTarget(target); err != nil {
				log.Printf("[pipeline] save fofa domain target: %v", err)
			}
		}
		for _, ip := range ips {
			target := &models.Target{
				ID:        util.GenerateID(),
				ProjectID: p.projectID,
				Type:      models.TargetTypeIP,
				Value:     ip,
				Source:    "fofa",
				Status:    "active",
				CreatedAt: time.Now().UTC(),
			}
			if err := p.queries.CreateTarget(target); err != nil {
				log.Printf("[pipeline] save fofa ip target: %v", err)
			}
		}

		if len(domains) > 0 {
			if err := p.runDomainFlow(ctx, makeTargets(domains, models.TargetTypeDomain)); err != nil {
				log.Printf("domain flow from company: %v", err)
				if flowErr == nil {
					flowErr = err
				}
			}
		}
		if len(ips) > 0 {
			if err := p.runIPFlow(ctx, makeTargets(ips, models.TargetTypeIP)); err != nil {
				log.Printf("ip flow from company: %v", err)
				if flowErr == nil {
					flowErr = err
				}
			}
		}
	}

	if flowErr == nil {
		p.completeStage(StageSearch)
	}
	return flowErr
}

// runDomainFlow: Subfinder → DNS → CDN → Naabu → nmap -sV → split Web/nonWeb → httpx → Nuclei.
func (p *Pipeline) runDomainFlow(ctx context.Context, targets []*models.Target) error {
	p.setStage(StageSubdomain)

	domains := extractTargetValues(targets)

	// S2: Subdomain discovery
	var allDomains []string
	if p.config.EnableSubfinder {
		for _, domain := range domains {
			subs, err := p.runSubfinder(ctx, domain)
			if err != nil {
				log.Printf("subfinder %s: %v", domain, err)
			}
			allDomains = append(allDomains, subs...)
			allDomains = append(allDomains, domain)
		}
		allDomains = dedupStrings(allDomains)
	} else {
		allDomains = domains
	}
	p.completeStage(StageSubdomain)

	// S3: DNS resolution via dnsx
	p.setStage(StageResolve)
	var dnsRecords []models.DNSRecord
	var err error
	if p.config.EnableDNSx {
		dnsRecords, err = p.runDNSx(ctx, allDomains)
		if err != nil {
			log.Printf("dnsx resolution: %v", err)
			p.failStage(StageResolve, err.Error())
		} else {
			p.completeStage(StageResolve)
		}
	} else {
		dnsRecords, err = p.resolver.WithParallel(p.config.DNSxThreads).
			WithTimeout(time.Duration(p.config.DNSxTimeout) * time.Second).
			Resolve(ctx, allDomains)
		if err != nil {
			log.Printf("dns resolution: %v", err)
			p.failStage(StageResolve, err.Error())
		} else {
			p.completeStage(StageResolve)
		}
	}

	for _, rec := range dnsRecords {
		rec.ProjectID = p.projectID
		rec.ID = util.GenerateID()
		rec.CreatedAt = time.Now().UTC()
		if err := p.queries.SaveDNSRecord(&rec); err != nil {
			log.Printf("[pipeline] save dns record: %v", err)
		}
	}

	// S4: CDN filtering (skipped entirely when disabled — no stage record is
	// created so the UI doesn't show a misleading "CDN 过滤 0.0s" row in
	// modes like internal scan where CDN filtering doesn't apply).
	allIPs := resolve.ExtractAllIPs(dnsRecords)
	var nonCDNIPs []string
	var cdnDomains []string
	var cdnResults []models.CDNResult
	if p.config.EnableCDNFilter {
		p.setStage(StageCDNFilter)
		nonCDNIPs, cdnResults, err = p.cdnDet.FilterCDNIPs(ctx, allIPs)
		if err != nil {
			log.Printf("cdn filter: %v", err)
			p.failStage(StageCDNFilter, err.Error())
			nonCDNIPs = allIPs
		} else {
			p.completeStage(StageCDNFilter)
		}
		for _, cdn := range cdnResults {
			cdn.ProjectID = p.projectID
			cdn.ID = util.GenerateID()
			cdn.CreatedAt = time.Now().UTC()
			if err := p.queries.SaveCDNResult(&cdn); err != nil {
				log.Printf("[pipeline] save cdn result: %v", err)
			}
		}
		cdnDomains = resolve.ExtractCDNDomains(dnsRecords, cdnResults)
	} else {
		nonCDNIPs = allIPs
	}

	// S4.5: Alive check via nmap (filters dead IPs so naabu only scans live hosts)
	p.setStage(StageAlive)
	aliveIPs, aliveErr := p.runNmapAlive(ctx, nonCDNIPs)
	if aliveErr != nil {
		log.Printf("nmap alive: %v", aliveErr)
		p.failStage(StageAlive, aliveErr.Error())
		aliveIPs = nonCDNIPs
	} else {
		if len(aliveIPs) == 0 && len(nonCDNIPs) > 0 {
			log.Printf("[pipeline] nmap reported 0 alive, falling back to %d input hosts", len(nonCDNIPs))
			aliveIPs = nonCDNIPs
		}
		p.completeStage(StageAlive)
	}

	// S5: Port scan
	p.setStage(StagePortScan)
	var ports []parser.PortInfo
	if len(aliveIPs) > 0 {
		ports, err = p.runNaabu(ctx, aliveIPs)
		if err != nil {
			log.Printf("naabu: %v", err)
			p.failStage(StagePortScan, err.Error())
		} else {
			p.completeStage(StagePortScan)
		}
	} else {
		p.completeStage(StagePortScan)
	}

	// S6: Service fingerprinting (nmap -sV)
	p.setStage(StageFingerprint)
	var fpResults []fingerprint.NmapServiceResult
	if p.config.EnableNmapService && len(ports) > 0 {
		fpResults, err = p.runNmapServiceScan(ctx, ports)
		if err != nil {
			log.Printf("nmap -sV: %v", err)
			p.failStage(StageFingerprint, err.Error())
		} else {
			p.completeStage(StageFingerprint)
		}
	} else {
		p.completeStage(StageFingerprint)
	}

	p.saveFingerprints(fpResults)

	extraTargets := append([]string{}, cdnDomains...)
	extraTargets = append(extraTargets, allDomains...)
	// Fallback: if nmap produced no results, also feed naabu ports directly to httpx.
	if len(fpResults) == 0 && len(ports) > 0 {
		for _, port := range ports {
			extraTargets = append(extraTargets, fmt.Sprintf("%s:%d", port.IP, port.Port))
		}
	}
	p.runHTTPXAndNuclei(ctx, fpResults, extraTargets)

	return nil
}

// runIPFlow: CDN → Naabu → nmap -sV → split → httpx → Nuclei.
func (p *Pipeline) runIPFlow(ctx context.Context, targets []*models.Target) error {
	ips := extractTargetValues(targets)

	// CDN filter (skipped entirely when disabled — no stage record so the UI
	// doesn't show a misleading "CDN 过滤 0.0s" row in internal scan mode).
	var nonCDNIPs []string
	if p.config.EnableCDNFilter {
		p.setStage(StageCDNFilter)
		var cdnResults []models.CDNResult
		var err error
		nonCDNIPs, cdnResults, err = p.cdnDet.FilterCDNIPs(ctx, ips)
		if err != nil {
			log.Printf("cdn filter: %v", err)
			p.failStage(StageCDNFilter, err.Error())
		} else {
			p.completeStage(StageCDNFilter)
		}
		for _, cdn := range cdnResults {
			cdn.ProjectID = p.projectID
			cdn.ID = util.GenerateID()
			cdn.CreatedAt = time.Now().UTC()
			if err := p.queries.SaveCDNResult(&cdn); err != nil {
				log.Printf("[pipeline] save cdn result: %v", err)
			}
		}
	} else {
		nonCDNIPs = ips
	}

	// Alive check via nmap
	p.setStage(StageAlive)
	aliveIPs, aliveErr := p.runNmapAlive(ctx, nonCDNIPs)
	if aliveErr != nil {
		log.Printf("nmap alive: %v", aliveErr)
		p.failStage(StageAlive, aliveErr.Error())
		aliveIPs = nonCDNIPs
	} else {
		if len(aliveIPs) == 0 && len(nonCDNIPs) > 0 {
			log.Printf("[pipeline] nmap reported 0 alive, falling back to %d input hosts", len(nonCDNIPs))
			aliveIPs = nonCDNIPs
		}
		p.completeStage(StageAlive)
	}

	// Port scan
	p.setStage(StagePortScan)
	ports, err := p.runNaabu(ctx, aliveIPs)
	if err != nil {
		log.Printf("naabu: %v", err)
		p.failStage(StagePortScan, err.Error())
	} else {
		p.completeStage(StagePortScan)
	}

	// Service fingerprint (nmap -sV)
	p.setStage(StageFingerprint)
	var fpResults []fingerprint.NmapServiceResult
	if p.config.EnableNmapService && len(ports) > 0 {
		fpResults, err = p.runNmapServiceScan(ctx, ports)
		if err != nil {
			log.Printf("nmap -sV: %v", err)
			p.failStage(StageFingerprint, err.Error())
		} else {
			p.completeStage(StageFingerprint)
		}
	} else {
		p.completeStage(StageFingerprint)
	}

	p.saveFingerprints(fpResults)

	// Fallback: if nmap produced no results, feed naabu ports directly to httpx.
	var extraHTTPXTargets []string
	if len(fpResults) == 0 && len(ports) > 0 {
		for _, port := range ports {
			extraHTTPXTargets = append(extraHTTPXTargets, fmt.Sprintf("%s:%d", port.IP, port.Port))
		}
	}

	p.runHTTPXAndNuclei(ctx, fpResults, extraHTTPXTargets)

	return nil
}

// runCIDRFlow was removed in favor of single-point scope filtering: see
// scope.Engine.FilterTargets, which now expands CIDR Targets to atomic IP
// Targets at the pipeline entry. Subsequent processing reuses runIPFlow.
// runFlow's TargetTypeCIDR case is a guard against any caller that bypasses
// the entry filter.

// runURLFlow: httpx → Nuclei.
func (p *Pipeline) runURLFlow(ctx context.Context, targets []*models.Target) error {
	urls := extractTargetValues(targets)

	webEndpoints, err := p.runHttpx(ctx, urls)
	if err != nil {
		log.Printf("httpx: %v", err)
		p.failStage(StageHTTPX, err.Error())
	} else {
		p.completeStage(StageHTTPX)
	}

	if p.config.EnableNuclei && len(webEndpoints) > 0 {
		p.setStage(StageVuln)
		if err := p.runNucleiWeb(ctx, webEndpoints); err != nil {
			log.Printf("nuclei: %v", err)
			p.failStage(StageVuln, err.Error())
		} else {
			p.completeStage(StageVuln)
		}
	}

	return nil
}
