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

// runCompanyFlow: passive search (FOFA + Hunter + Quake) → expand to
// domains/ips → route to respective flows for each company target.
func (p *Pipeline) runCompanyFlow(ctx context.Context, targets []*models.Target) error {
	var flowErr error
	for _, t := range targets {
		if err := p.runPassiveSearch(ctx, t.Value); err != nil {
			log.Printf("passive search for %s: %v", t.Value, err)
			if flowErr == nil {
				flowErr = err
			}
		}
	}
	return flowErr
}

// runDomainFlow: PassiveCert → PassiveURL → Subfinder → DNS → CDN → Naabu → nmap -sV → split Web/nonWeb → httpx → Nuclei.
func (p *Pipeline) runDomainFlow(ctx context.Context, targets []*models.Target) error {
	// Check for cancellation before starting work.
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	p.setStage(StageSubdomain)

	domains := extractTargetValues(targets)

	// S1: Passive certificate transparency subdomain discovery
	for _, d := range domains {
		if err := p.runPassiveCert(ctx, d); err != nil {
			log.Printf("passive cert %s: %v", d, err)
		}
	}

	// S1b: Passive URL history collection
	for _, d := range domains {
		urls, err := p.runPassiveURL(ctx, d)
		if err != nil {
			log.Printf("passive url %s: %v", d, err)
		} else {
			_ = urls // stored in memory for post-phase URL discovery
		}
	}

	// S2: Subdomain discovery
	var allDomains []string
	if p.config.SubfinderMode == "off" || !p.config.EnableSubfinder {
		allDomains = domains
	} else if p.config.EnableSubfinder {
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

	// S4: CDN classification — UI stage when EnableCDNFilter; port-scan skip when
	// SkipPortscanOnCDNHost (external preset). Either flag triggers cdncheck.
	allIPs := resolve.ExtractAllIPs(dnsRecords)
	var nonCDNIPs []string
	var cdnDomains []string
	var cdnResults []models.CDNResult
	classifyCDN := p.config.EnableCDNFilter || p.config.SkipPortscanOnCDNHost
	if classifyCDN {
		if p.config.EnableCDNFilter {
			p.setStage(StageCDNFilter)
		}
		nonCDNIPs, cdnResults, err = p.runCDNCheck(ctx, allIPs)
		if err != nil {
			log.Printf("cdn filter: %v", err)
			if p.config.EnableCDNFilter {
				p.failStage(StageCDNFilter, err.Error())
			}
			nonCDNIPs = allIPs
		} else if p.config.EnableCDNFilter {
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
	portScanIPs := ipsForPortScan(allIPs, nonCDNIPs, classifyCDN, p.config.SkipPortscanOnCDNHost)
	if p.config.SkipPortscanOnCDNHost && classifyCDN && len(allIPs) > 0 && len(portScanIPs) < len(allIPs) {
		log.Printf("[pipeline] skip_portscan_on_cdn_host: port scan on %d/%d IPs (%d CDN skipped)",
			len(portScanIPs), len(allIPs), len(allIPs)-len(portScanIPs))
	}

	// S4.5: Alive check via nmap (filters dead IPs so naabu only scans live hosts)
	p.setStage(StageAlive)
	aliveIPs, aliveErr := p.runNmapAlive(ctx, portScanIPs)
	if aliveErr != nil {
		log.Printf("nmap alive: %v", aliveErr)
		p.failStage(StageAlive, aliveErr.Error())
		// Don't fall back to all IPs on error — skip port scan entirely
		aliveIPs = nil
	} else {
		if len(aliveIPs) == 0 && len(portScanIPs) > 0 {
			log.Printf("[pipeline] nmap reported 0 alive hosts out of %d inputs, skipping port scan", len(portScanIPs))
			// No fallback: naabu would waste time scanning dead hosts
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
	// Check for cancellation before starting work.
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	ips := extractTargetValues(targets)

	var nonCDNIPs []string
	classifyCDN := p.config.EnableCDNFilter || p.config.SkipPortscanOnCDNHost
	if classifyCDN {
		if p.config.EnableCDNFilter {
			p.setStage(StageCDNFilter)
		}
		var cdnResults []models.CDNResult
		var err error
		nonCDNIPs, cdnResults, err = p.runCDNCheck(ctx, ips)
		if err != nil {
			log.Printf("cdn filter: %v", err)
			if p.config.EnableCDNFilter {
				p.failStage(StageCDNFilter, err.Error())
			}
		} else if p.config.EnableCDNFilter {
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
	portScanIPs := ipsForPortScan(ips, nonCDNIPs, classifyCDN, p.config.SkipPortscanOnCDNHost)
	if p.config.SkipPortscanOnCDNHost && classifyCDN && len(ips) > 0 && len(portScanIPs) < len(ips) {
		log.Printf("[pipeline] skip_portscan_on_cdn_host: port scan on %d/%d IPs", len(portScanIPs), len(ips))
	}

	// Alive check via nmap
	p.setStage(StageAlive)
	aliveIPs, aliveErr := p.runNmapAlive(ctx, portScanIPs)
	if aliveErr != nil {
		log.Printf("nmap alive: %v", aliveErr)
		p.failStage(StageAlive, aliveErr.Error())
		// Don't fall back to all IPs on error — skip port scan entirely
		aliveIPs = nil
	} else {
		if len(aliveIPs) == 0 && len(portScanIPs) > 0 {
			log.Printf("[pipeline] nmap reported 0 alive hosts out of %d inputs, skipping port scan", len(portScanIPs))
			// No fallback: naabu would waste time scanning dead hosts
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
