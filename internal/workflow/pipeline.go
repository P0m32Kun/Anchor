package workflow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/P0m32Kun/Anchor/internal/asset"
	"github.com/P0m32Kun/Anchor/internal/cdn"
	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/fingerprint"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/nuclei"
	"github.com/P0m32Kun/Anchor/internal/parser"
	"github.com/P0m32Kun/Anchor/internal/resolve"
	"github.com/P0m32Kun/Anchor/internal/scope"
	"github.com/P0m32Kun/Anchor/internal/search"
	"github.com/P0m32Kun/Anchor/internal/util"
	"github.com/P0m32Kun/Anchor/internal/worker"
)

// StageID identifies a pipeline stage.
type StageID string

const (
	StageClassify    StageID = "classify"
	StageSearch      StageID = "search"
	StageSubdomain   StageID = "subdomain"
	StageResolve     StageID = "resolve"
	StageCDNFilter   StageID = "cdn_filter"
	StagePortScan    StageID = "portscan"
	StageFingerprint StageID = "fingerprint"
	StageHTTPX       StageID = "httpx"
	StageVuln        StageID = "vuln"
)

// StageEventCallback is invoked when a pipeline stage changes state.
type StageEventCallback func(runID string, stage StageID, status string, errMsg string)

// Pipeline orchestrates the complete scan workflow.
type Pipeline struct {
	queries        *db.Queries
	runner         *worker.Runner
	scope          *scope.Engine
	resolver       *resolve.Resolver
	cdnDet         *cdn.Detector
	fofa           *search.FofaClient
	merger         *asset.Merger
	dataDir        string
	projectID      string
	config         models.PipelineConfig
	runID          string
	onStageChange  StageEventCallback
}

// NewPipeline creates a new Pipeline instance.
func NewPipeline(queries *db.Queries, runner *worker.Runner, scopeEng *scope.Engine, dataDir string) *Pipeline {
	return &Pipeline{
		queries:  queries,
		runner:   runner,
		scope:    scopeEng,
		resolver: resolve.NewResolver(),
		cdnDet:   cdn.NewDetector(),
		merger:   asset.NewMerger(queries),
		dataDir:  dataDir,
	}
}

// WithConfig sets the pipeline configuration.
func (p *Pipeline) WithConfig(cfg models.PipelineConfig) *Pipeline {
	p.config = cfg
	return p
}

// WithFOFA sets the FOFA client.
func (p *Pipeline) WithFOFA(email, apiKey string) *Pipeline {
	if email != "" && apiKey != "" {
		p.fofa = search.NewFofaClient(email, apiKey)
	}
	return p
}

func (p *Pipeline) WithRunID(runID string) *Pipeline {
	p.runID = runID
	return p
}

func (p *Pipeline) WithStageCallback(cb StageEventCallback) *Pipeline {
	p.onStageChange = cb
	return p
}

var requiredTools = []string{"subfinder", "naabu", "nerva", "cdncheck", "httpx", "nuclei", "dnsx"}

func (p *Pipeline) checkTools() []string {
	var missing []string
	for _, tool := range requiredTools {
		if _, err := exec.LookPath(tool); err != nil {
			missing = append(missing, tool)
		}
	}
	return missing
}

func (p *Pipeline) setStage(stage StageID) {
	if p.runID == "" {
		return
	}
	now := time.Now().UTC()
	if err := p.queries.UpdatePipelineRunStage(p.runID, string(stage)); err != nil {
		log.Printf("[pipeline] update stage: %v", err)
	}

	// Upsert stage record: if exists mark running, else create.
	existing, err := p.queries.GetPipelineRunStage(p.runID, string(stage))
	if err != nil {
		log.Printf("[pipeline] get stage record: %v", err)
	}
	if existing == nil {
		s := &models.PipelineRunStage{
			ID:        util.GenerateID(),
			RunID:     p.runID,
			Stage:     string(stage),
			Status:    models.StageStatusRunning,
			StartedAt: &now,
			CreatedAt: now,
		}
		if err := p.queries.CreatePipelineRunStage(s); err != nil {
			log.Printf("[pipeline] create stage record: %v", err)
		}
	} else {
		if err := p.queries.UpdatePipelineRunStageRecord(existing.ID, models.StageStatusRunning, "", nil); err != nil {
			log.Printf("[pipeline] update stage record: %v", err)
		}
	}

	if p.onStageChange != nil {
		p.onStageChange(p.runID, stage, "running", "")
	}
}

func (p *Pipeline) completeStage(stage StageID) {
	if p.runID == "" {
		return
	}
	now := time.Now().UTC()
	existing, err := p.queries.GetPipelineRunStage(p.runID, string(stage))
	if err != nil {
		log.Printf("[pipeline] get stage record for complete: %v", err)
		return
	}
	if existing != nil {
		if err := p.queries.UpdatePipelineRunStageRecord(existing.ID, models.StageStatusCompleted, "", &now); err != nil {
			log.Printf("[pipeline] complete stage record: %v", err)
		}
	}
	if p.onStageChange != nil {
		p.onStageChange(p.runID, stage, "completed", "")
	}
}

func (p *Pipeline) failStage(stage StageID, errMsg string) {
	if p.runID == "" {
		return
	}
	now := time.Now().UTC()
	existing, err := p.queries.GetPipelineRunStage(p.runID, string(stage))
	if err != nil {
		log.Printf("[pipeline] get stage record for fail: %v", err)
		return
	}
	if existing != nil {
		if err := p.queries.UpdatePipelineRunStageRecord(existing.ID, models.StageStatusFailed, errMsg, &now); err != nil {
			log.Printf("[pipeline] fail stage record: %v", err)
		}
	}
	if p.onStageChange != nil {
		p.onStageChange(p.runID, stage, "failed", errMsg)
	}
}

func (p *Pipeline) saveFingerprints(fpResults []fingerprint.NervaResult) {
	if len(fpResults) == 0 {
		return
	}
	fps := fingerprint.ConvertToServiceFingerprints(p.projectID, fpResults)
	for i := range fps {
		fps[i].ID = util.GenerateID()
		fps[i].ProjectID = p.projectID
		fps[i].CreatedAt = time.Now().UTC()
		if err := p.queries.SaveServiceFingerprint(&fps[i]); err != nil {
			log.Printf("[pipeline] save service fingerprint: %v", err)
		}
	}
}

func (p *Pipeline) runHTTPXAndNuclei(ctx context.Context, fpResults []fingerprint.NervaResult, extraHTTPXTargets []string) error {
	webResults, nonWebResults := fingerprint.SplitByServiceType(fpResults)

	httpxTargets := makeHTTPTargets(webResults)
	for _, t := range extraHTTPXTargets {
		httpxTargets = append(httpxTargets, t)
	}
	httpxTargets = dedupStrings(httpxTargets)

	var webEndpoints []*models.WebEndpoint
	if len(httpxTargets) > 0 {
		p.setStage(StageHTTPX)
		endpoints, err := p.runHttpx(ctx, httpxTargets)
		if err != nil {
			log.Printf("httpx: %v", err)
			p.failStage(StageHTTPX, err.Error())
		} else {
			webEndpoints = endpoints
			p.completeStage(StageHTTPX)
		}
	}

	if p.config.EnableNuclei {
		p.setStage(StageVuln)
		if len(webEndpoints) > 0 {
			if err := p.runNucleiWeb(ctx, webEndpoints); err != nil {
				log.Printf("nuclei web: %v", err)
				p.failStage(StageVuln, "nuclei web: "+err.Error())
			}
		}
		if len(nonWebResults) > 0 {
			if err := p.runNucleiNonWeb(ctx, nonWebResults); err != nil {
				log.Printf("nuclei non-web: %v", err)
				p.failStage(StageVuln, "nuclei non-web: "+err.Error())
			}
		}
		p.completeStage(StageVuln)
	}

	return nil
}

// Run executes the pipeline for a project.
func (p *Pipeline) Run(ctx context.Context, projectID string) error {
	p.projectID = projectID

	// Load project config if not set
	if p.config == (models.PipelineConfig{}) {
		project, err := p.queries.GetProject(projectID)
		if err != nil {
			return fmt.Errorf("get project: %w", err)
		}
		if project != nil && project.PipelineConfig != nil && *project.PipelineConfig != "" {
			if err := json.Unmarshal([]byte(*project.PipelineConfig), &p.config); err != nil {
				log.Printf("[pipeline] unmarshal pipeline config: %v", err)
				p.config = models.DefaultPipelineConfig()
			}
		} else {
			p.config = models.DefaultPipelineConfig()
		}
	}

	// Initialize FOFA if enabled and not already set
	if p.config.EnableFOFA && p.fofa == nil {
		cred, err := p.queries.GetEngineCredential("fofa")
		if err == nil && cred != nil && cred.Email != nil && *cred.Email != "" && cred.APIKey != "" {
			p.fofa = search.NewFofaClient(*cred.Email, cred.APIKey)
		}
	}

	// Check required tools (non-blocking: tools run on workers, not server)
	if missing := p.checkTools(); len(missing) > 0 {
		log.Printf("[pipeline] warning: required tools not found on server (OK if workers have them): %s", strings.Join(missing, ", "))
	}

	// Get all targets
	targets, err := p.queries.ListTargetsByProject(projectID)
	if err != nil {
		if p.runID != "" {
			p.queries.UpdatePipelineRunError(p.runID, err.Error())
			p.queries.UpdatePipelineRunStatus(p.runID, "failed")
		}
		return fmt.Errorf("list targets: %w", err)
	}

	// Group by type
	groups := groupTargetsByType(targets)

	// Execute flows for each target type
	var flowErr error
	for _, group := range groups {
		if err := p.runFlow(ctx, group); err != nil {
			log.Printf("pipeline flow error for type %s: %v", group.Type, err)
			flowErr = err
		}
	}

	if p.runID != "" {
		now := time.Now().UTC()
		if flowErr != nil {
			p.queries.UpdatePipelineRunError(p.runID, flowErr.Error())
			p.queries.UpdatePipelineRunStatus(p.runID, "failed")
		} else {
			p.queries.UpdatePipelineRunCompleted(p.runID, now)
		}
		// Mark any still-running stages as completed (or failed if overall failed).
		stages, _ := p.queries.ListPipelineRunStages(p.runID)
		for _, s := range stages {
			if s.Status == models.StageStatusRunning {
				if flowErr != nil {
					p.queries.UpdatePipelineRunStageRecord(s.ID, models.StageStatusFailed, "pipeline aborted", &now)
				} else {
					p.queries.UpdatePipelineRunStageRecord(s.ID, models.StageStatusCompleted, "", &now)
				}
			}
		}
	}

	return flowErr
}

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
		return p.runCIDRFlow(ctx, group.Targets)
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

// runDomainFlow: Subfinder → DNS → CDN → Naabu → nerva → split Web/nonWeb → httpx → Nuclei.
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

	// S4: CDN filtering
	p.setStage(StageCDNFilter)
	allIPs := resolve.ExtractAllIPs(dnsRecords)
	var nonCDNIPs []string
	var cdnDomains []string
	var cdnResults []models.CDNResult
	if p.config.EnableCDNFilter {
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
		p.completeStage(StageCDNFilter)
	}

	// S5: Port scan
	p.setStage(StagePortScan)
	var ports []parser.PortInfo
	if len(nonCDNIPs) > 0 {
		ports, err = p.runNaabu(ctx, nonCDNIPs)
		if err != nil {
			log.Printf("naabu: %v", err)
			p.failStage(StagePortScan, err.Error())
		} else {
			p.completeStage(StagePortScan)
		}
	} else {
		p.completeStage(StagePortScan)
	}

	// S6: Service fingerprinting
	p.setStage(StageFingerprint)
	var fpResults []fingerprint.NervaResult
	if p.config.EnableNerva && len(ports) > 0 {
		fpResults, err = p.runNerva(ctx, ports)
		if err != nil {
			log.Printf("nerva: %v", err)
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
	// Fallback: if nerva produced no results, also feed naabu ports directly to httpx.
	if len(fpResults) == 0 && len(ports) > 0 {
		for _, port := range ports {
			extraTargets = append(extraTargets, fmt.Sprintf("%s:%d", port.IP, port.Port))
		}
	}
	p.runHTTPXAndNuclei(ctx, fpResults, extraTargets)

	return nil
}

// runIPFlow: CDN → Naabu → nerva → split → httpx → Nuclei.
func (p *Pipeline) runIPFlow(ctx context.Context, targets []*models.Target) error {
	p.setStage(StageCDNFilter)

	ips := extractTargetValues(targets)

	// CDN filter
	var nonCDNIPs []string
	if p.config.EnableCDNFilter {
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
		p.completeStage(StageCDNFilter)
	}

	// Port scan
	p.setStage(StagePortScan)
	ports, err := p.runNaabu(ctx, nonCDNIPs)
	if err != nil {
		log.Printf("naabu: %v", err)
		p.failStage(StagePortScan, err.Error())
	} else {
		p.completeStage(StagePortScan)
	}

	// Service fingerprint
	p.setStage(StageFingerprint)
	var fpResults []fingerprint.NervaResult
	if p.config.EnableNerva && len(ports) > 0 {
		fpResults, err = p.runNerva(ctx, ports)
		if err != nil {
			log.Printf("nerva: %v", err)
			p.failStage(StageFingerprint, err.Error())
		} else {
			p.completeStage(StageFingerprint)
		}
	} else {
		p.completeStage(StageFingerprint)
	}

	p.saveFingerprints(fpResults)

	// Fallback: if nerva produced no results, feed naabu ports directly to httpx.
	var extraHTTPXTargets []string
	if len(fpResults) == 0 && len(ports) > 0 {
		for _, port := range ports {
			extraHTTPXTargets = append(extraHTTPXTargets, fmt.Sprintf("%s:%d", port.IP, port.Port))
		}
	}

	p.runHTTPXAndNuclei(ctx, fpResults, extraHTTPXTargets)

	return nil
}

// runCIDRFlow: Naabu → nerva → split → httpx → Nuclei.
func (p *Pipeline) runCIDRFlow(ctx context.Context, targets []*models.Target) error {
	p.setStage(StagePortScan)

	cidrs := extractTargetValues(targets)

	// Port scan
	ports, err := p.runNaabu(ctx, cidrs)
	if err != nil {
		log.Printf("naabu: %v", err)
	}

	// Service fingerprint
	p.setStage(StageFingerprint)
	var fpResults []fingerprint.NervaResult
	if p.config.EnableNerva && len(ports) > 0 {
		fpResults, err = p.runNerva(ctx, ports)
		if err != nil {
			log.Printf("nerva: %v", err)
			p.failStage(StageFingerprint, err.Error())
		} else {
			p.completeStage(StageFingerprint)
		}
	} else {
		p.completeStage(StageFingerprint)
	}

	p.saveFingerprints(fpResults)

	// Fallback: if nerva produced no results, feed naabu ports directly to httpx.
	var extraHTTPXTargets []string
	if len(fpResults) == 0 && len(ports) > 0 {
		for _, port := range ports {
			extraHTTPXTargets = append(extraHTTPXTargets, fmt.Sprintf("%s:%d", port.IP, port.Port))
		}
	}

	p.runHTTPXAndNuclei(ctx, fpResults, extraHTTPXTargets)

	return nil
}

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

// --- Tool execution helpers ---

func (p *Pipeline) runSubfinder(ctx context.Context, domain string) ([]string, error) {
	target := &models.Target{Type: models.TargetTypeDomain, Value: domain}
	decision, err := p.scope.ValidateBeforeRun(ctx, p.projectID, target, "")
	if err != nil || decision.Decision == models.ScopeDeny {
		return nil, fmt.Errorf("scope denied")
	}

	task, stdout, err := p.createAndRunTask(ctx, "subfinder", worker.BuildSubfinderCommand(domain, p.config.SubfinderRateLimit, p.config.SubfinderThreads, p.config.SubfinderTimeout))
	if err != nil {
		return nil, err
	}
	_ = task

	subs := parser.ParseSubfinderOutput(bytes.NewReader(stdout))
	return subs, nil
}

func (p *Pipeline) runNaabu(ctx context.Context, hosts []string) ([]parser.PortInfo, error) {
	if len(hosts) == 0 {
		return nil, nil
	}

	hostFile := filepath.Join(p.dataDir, "workdirs", p.projectID, fmt.Sprintf("naabu-%s.txt", util.GenerateID()))
	if err := os.MkdirAll(filepath.Dir(hostFile), 0750); err != nil {
		return nil, err
	}
	if err := os.WriteFile(hostFile, []byte(strings.Join(hosts, "\n")), 0640); err != nil {
		return nil, err
	}
	// Ensure absolute path so worker can find it regardless of cmd.Dir.
	if abs, err := filepath.Abs(hostFile); err == nil {
		hostFile = abs
	}

	task, stdout, err := p.createAndRunTask(ctx, "naabu", worker.BuildNaabuCommand(hostFile, p.config.PortRange, p.config.NaabuRate, p.config.NaabuThreads, p.config.NaabuTimeout))
	if err != nil {
		return nil, err
	}
	_ = task

	ports := parser.ParseNaabuOutput(bytes.NewReader(stdout))
	log.Printf("[pipeline] naabu parsed %d ports for project %s (stdout len=%d)", len(ports), p.projectID, len(stdout))
	for _, port := range ports {
		ipAsset, _, err := p.merger.MergeOrCreateAsset(p.projectID, "ip", port.IP, "naabu")
		if err != nil {
			log.Printf("[pipeline] merge/create asset %s: %v", port.IP, err)
			continue
		}
		_, _, err = p.merger.CreatePortIfNotExists(ipAsset.ID, port.Port, "tcp", "naabu")
		if err != nil {
			log.Printf("[pipeline] create port %s:%d: %v", port.IP, port.Port, err)
		}
	}
	return ports, nil
}

func (p *Pipeline) runNerva(ctx context.Context, ports []parser.PortInfo) ([]fingerprint.NervaResult, error) {
	if len(ports) == 0 {
		return nil, nil
	}

	var targets []string
	for _, port := range ports {
		targets = append(targets, fmt.Sprintf("%s:%d", port.IP, port.Port))
	}

	cmd := worker.BuildNervaCommand(strings.Join(targets, ","), p.config.NervaRateLimit, p.config.NervaWorkers, p.config.NervaTimeout)
	task, stdout, err := p.createAndRunTask(ctx, "nerva", cmd)
	if err != nil {
		return nil, err
	}
	_ = task

	results := fingerprint.ParseNervaOutput(string(stdout))
	return results, nil
}

func (p *Pipeline) runDNSx(ctx context.Context, domains []string) ([]models.DNSRecord, error) {
	if len(domains) == 0 {
		return nil, nil
	}

	hostFile := filepath.Join(p.dataDir, "workdirs", p.projectID, fmt.Sprintf("dnsx-%s.txt", util.GenerateID()))
	if err := os.MkdirAll(filepath.Dir(hostFile), 0750); err != nil {
		return nil, err
	}
	if err := os.WriteFile(hostFile, []byte(strings.Join(domains, "\n")), 0640); err != nil {
		return nil, err
	}
	if abs, err := filepath.Abs(hostFile); err == nil {
		hostFile = abs
	}

	cmd := worker.BuildDNSxCommand(hostFile, nil, p.config.DNSxRateLimit, p.config.DNSxThreads)
	task, stdout, err := p.createAndRunTask(ctx, "dnsx", cmd)
	if err != nil {
		return nil, err
	}
	_ = task

	results := parser.ParseDNSxOutput(bytes.NewReader(stdout))
	var records []models.DNSRecord
	for domain, rec := range results {
		records = append(records, models.DNSRecord{
			Domain: domain,
			IPs:    parser.ExtractDNSxIPs(rec),
			CNAMEs: parser.ExtractDNSxCNAMEs(rec),
			TTL:    uint32(rec.TTL),
		})
	}
	return records, nil
}

func (p *Pipeline) runHttpx(ctx context.Context, hosts []string) ([]*models.WebEndpoint, error) {
	if len(hosts) == 0 {
		return nil, nil
	}

	hostFile := filepath.Join(p.dataDir, "workdirs", p.projectID, fmt.Sprintf("httpx-%s.txt", util.GenerateID()))
	if err := os.MkdirAll(filepath.Dir(hostFile), 0750); err != nil {
		return nil, err
	}
	if err := os.WriteFile(hostFile, []byte(strings.Join(hosts, "\n")), 0640); err != nil {
		return nil, err
	}
	if abs, err := filepath.Abs(hostFile); err == nil {
		hostFile = abs
	}

	task, stdout, err := p.createAndRunTask(ctx, "httpx", worker.BuildHttpxCommand(hostFile, p.config.HttpxRateLimit, p.config.HttpxThreads))
	if err != nil {
		return nil, err
	}
	_ = task

	endpoints := parser.ParseHttpxOutput(bytes.NewReader(stdout))
	for _, ep := range endpoints {
		ep.ID = util.GenerateID()
		ep.ProjectID = p.projectID
		ep.CreatedAt = time.Now().UTC()
		if err := p.queries.CreateWebEndpoint(ep); err != nil {
			log.Printf("[pipeline] save web endpoint %s: %v", ep.URL, err)
		}
	}
	return endpoints, nil
}

func (p *Pipeline) runNucleiWeb(ctx context.Context, endpoints []*models.WebEndpoint) error {
	groups := nuclei.GroupEndpointsByTags(endpoints)
	if len(groups) == 0 {
		return nil
	}

	// Build URL -> endpoint map for finding linkage.
	urlToEndpoint := make(map[string]*models.WebEndpoint)
	for _, ep := range endpoints {
		urlToEndpoint[ep.URL] = ep
	}

	for tagKey, urls := range groups {
		tags := strings.Split(tagKey, ",")
		targetFile := filepath.Join(p.dataDir, "workdirs", p.projectID, fmt.Sprintf("nuclei-%s.txt", util.GenerateID()))
		if err := os.WriteFile(targetFile, []byte(strings.Join(urls, "\n")), 0640); err != nil {
			continue
		}
		if abs, err := filepath.Abs(targetFile); err == nil {
			targetFile = abs
		}

		task, stdout, err := p.createAndRunTask(ctx, "nuclei", worker.BuildNucleiCommand(targetFile, "deep", p.config.NucleiRateLimit, tags))
		if err != nil {
			log.Printf("nuclei task for tags %s: %v", tagKey, err)
			continue
		}
		_ = task
		p.saveNucleiFindings(stdout, urlToEndpoint, nil)
	}

	return nil
}

func (p *Pipeline) runNucleiNonWeb(ctx context.Context, results []fingerprint.NervaResult) error {
	// Group by service tag
	groups := make(map[string][]string)
	for _, r := range results {
		tags := nuclei.MapServiceToTags(r.Protocol)
		for _, tag := range tags {
			target := fmt.Sprintf("%s:%d", r.IP, r.Port)
			groups[tag] = append(groups[tag], target)
		}
	}

	if len(groups) == 0 {
		return nil
	}

	for tag, targets := range groups {
		targetFile := filepath.Join(p.dataDir, "workdirs", p.projectID, fmt.Sprintf("nuclei-%s.txt", util.GenerateID()))
		if err := os.WriteFile(targetFile, []byte(strings.Join(targets, "\n")), 0640); err != nil {
			continue
		}
		if abs, err := filepath.Abs(targetFile); err == nil {
			targetFile = abs
		}

		task, stdout, err := p.createAndRunTask(ctx, "nuclei", worker.BuildNucleiCommand(targetFile, "deep", p.config.NucleiRateLimit, []string{tag}))
		if err != nil {
			log.Printf("nuclei task for tag %s: %v", tag, err)
			continue
		}
		_ = task
		p.saveNucleiFindings(stdout, nil, nil)
	}

	return nil
}

func (p *Pipeline) saveNucleiFindings(stdout []byte, urlToEndpoint map[string]*models.WebEndpoint, ipToAsset map[string]*models.Asset) {
	if len(stdout) == 0 {
		return
	}
	results, parseErrs := parser.ParseNuclei(bytes.NewReader(stdout))
	for _, pe := range parseErrs {
		log.Printf("[saveNucleiFindings] parse error line=%d msg=%s", pe.Line, pe.Message)
	}
	for _, nr := range results {
		dedupKey := fmt.Sprintf("%s|%s|%s", nr.TemplateID, nr.Host, nr.MatcherName)
		existing, err := p.queries.GetFindingByDedupKey(p.projectID, dedupKey)
		if err != nil {
			continue
		}
		if existing != nil {
			continue
		}
		var assetID, webEndpointID *string
		if urlToEndpoint != nil {
			if ep, ok := urlToEndpoint[nr.Host]; ok {
				assetID = &ep.AssetID
				webEndpointID = &ep.ID
			}
		}
		if assetID == nil && ipToAsset != nil {
			if a, ok := ipToAsset[nr.Host]; ok {
				assetID = &a.ID
			}
		}
		f := &models.Finding{
			ID:            util.GenerateID(),
			ProjectID:     p.projectID,
			AssetID:       assetID,
			WebEndpointID: webEndpointID,
			SourceTool:    "nuclei",
			SourceRuleID:  nr.TemplateID,
			DedupKey:      dedupKey,
			Title:         nr.Name,
			Severity:      models.FindingSeverity(nr.Severity),
			Confidence:    80,
			Priority:      3,
			Status:        models.FindingPendingReview,
			Summary:       fmt.Sprintf("Host: %s\nMatched: %s\nMatcher: %s", nr.Host, nr.MatchedAt, nr.MatcherName),
			CreatedAt:     time.Now().UTC(),
			UpdatedAt:     time.Now().UTC(),
		}
		if err := p.queries.CreateFinding(f); err != nil {
			log.Printf("[pipeline] create finding %s: %v", nr.Name, err)
		}
	}
}

func (p *Pipeline) createAndRunTask(ctx context.Context, tool string, args []string) (*models.ScanTask, []byte, error) {
	taskID := util.GenerateID()
	now := time.Now().UTC()

	task := &models.ScanTask{
		ID:              taskID,
		ProjectID:       p.projectID,
		RunID:           &p.runID,
		Tool:            tool,
		CommandTemplate: strings.Join(args, " "),
		Status:          models.TaskCreated,
		CreatedAt:       now,
	}

	if err := p.queries.CreateScanTask(task); err != nil {
		return nil, nil, fmt.Errorf("create scan task: %w", err)
	}

	// Run task synchronously via worker
	if err := p.runner.Run(ctx, task.ID); err != nil {
		log.Printf("[pipeline] task %s (%s) run error: %v", task.ID, tool, err)
	}

	// Read stdout artifact
	stdout, err := p.readTaskStdout(task.ID)
	if err != nil {
		log.Printf("[pipeline] task %s (%s) read stdout: %v", task.ID, tool, err)
	}

	return task, stdout, nil
}

func (p *Pipeline) readTaskStdout(taskID string) ([]byte, error) {
	artifacts, err := p.queries.ListRawArtifactsByTask(taskID)
	if err != nil {
		return nil, err
	}
	for _, a := range artifacts {
		if a.Type == models.ArtifactStdout {
			return os.ReadFile(a.Path)
		}
	}
	return nil, fmt.Errorf("no stdout artifact found for task %s", taskID)
}

// --- Utility functions ---

func extractTargetValues(targets []*models.Target) []string {
	var vals []string
	for _, t := range targets {
		vals = append(vals, t.Value)
	}
	return vals
}

func makeTargets(values []string, typ models.TargetType) []*models.Target {
	var targets []*models.Target
	for _, v := range values {
		targets = append(targets, &models.Target{Type: typ, Value: v})
	}
	return targets
}

func dedupStrings(s []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, v := range s {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	return result
}

func makeHTTPTargets(results []fingerprint.NervaResult) []string {
	var targets []string
	for _, r := range results {
		host := r.IP
		if host == "" {
			host = r.Host
		}
		if r.Port == 443 {
			targets = append(targets, fmt.Sprintf("https://%s", host))
		} else {
			targets = append(targets, fmt.Sprintf("http://%s:%d", host, r.Port))
		}
	}
	return targets
}
