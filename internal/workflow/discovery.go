package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/P0m32Kun/Anchor/internal/asset"
	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/parser"
	"github.com/P0m32Kun/Anchor/internal/scope"
	"github.com/P0m32Kun/Anchor/internal/util"
	"github.com/P0m32Kun/Anchor/internal/worker"
)

// AssetDiscoveryWorkflow runs subfinder → httpx → naabu in sequence for each domain target,
// and supports direct IP and CIDR scanning as well.
type AssetDiscoveryWorkflow struct {
	queries *db.Queries
	runner  *worker.Runner
	scope   *scope.Engine
	merger  *asset.Merger
	dataDir string
	runID   string // optional: links tasks to a Run record
}

// WithRunID sets the Run ID to link all created tasks to a Run record.
func (w *AssetDiscoveryWorkflow) WithRunID(runID string) *AssetDiscoveryWorkflow {
	w.runID = runID
	return w
}

// DiscoveryResult holds the summary of an asset discovery run.
type DiscoveryResult struct {
	DomainsFound       int `json:"domains_found"`
	WebEndpointsFound  int `json:"web_endpoints_found"`
	IPsFound           int `json:"ips_found"`
	PortsFound         int `json:"ports_found"`
	SubfinderTaskCount int `json:"subfinder_task_count"`
	HttpxTaskCount     int `json:"httpx_task_count"`
	NaabuTaskCount     int `json:"naabu_task_count"`
}

// NewAssetDiscoveryWorkflow creates a new workflow instance.
func NewAssetDiscoveryWorkflow(queries *db.Queries, runner *worker.Runner, scopeEng *scope.Engine, dataDir string) *AssetDiscoveryWorkflow {
	return &AssetDiscoveryWorkflow{
		queries: queries,
		runner:  runner,
		scope:   scopeEng,
		merger:  asset.NewMerger(queries),
		dataDir: dataDir,
	}
}

// Run executes the asset discovery workflow for the given project.
// It processes domain, IP, and CIDR targets through their respective chains.
func (w *AssetDiscoveryWorkflow) Run(ctx context.Context, projectID string) (*DiscoveryResult, error) {
	result := &DiscoveryResult{}

	targets, err := w.queries.ListTargetsByProject(projectID)
	if err != nil {
		return nil, fmt.Errorf("list targets: %w", err)
	}

	var domainTargets, ipTargets, cidrTargets []*models.Target
	for _, t := range targets {
		switch t.Type {
		case models.TargetTypeDomain:
			domainTargets = append(domainTargets, t)
		case models.TargetTypeIP:
			ipTargets = append(ipTargets, t)
		case models.TargetTypeCIDR:
			cidrTargets = append(cidrTargets, t)
		}
	}

	if len(domainTargets) > 0 {
		if err := w.runDomainChain(ctx, projectID, domainTargets, result); err != nil {
			log.Printf("domain chain error: %v", err)
		}
	}

	if len(ipTargets) > 0 {
		if err := w.runIPChain(ctx, projectID, ipTargets, result); err != nil {
			log.Printf("ip chain error: %v", err)
		}
	}

	if len(cidrTargets) > 0 {
		if err := w.runCIDRChain(ctx, projectID, cidrTargets, result); err != nil {
			log.Printf("cidr chain error: %v", err)
		}
	}

	return result, nil
}

// runDomainChain runs subfinder → httpx → naabu for each domain target.
func (w *AssetDiscoveryWorkflow) runDomainChain(ctx context.Context, projectID string, domainTargets []*models.Target, result *DiscoveryResult) error {
	for _, dt := range domainTargets {
		decision, err := w.scope.Check(ctx, projectID, dt)
		if err != nil {
			continue
		}
		if decision.Decision == models.ScopeDeny {
			continue
		}

		subfinderTask, err := w.createAndRunTask(ctx, projectID, dt.ID, "subfinder", worker.BuildSubfinderCommand(dt.Value, 0, 0, 0))
		if err != nil {
			continue
		}
		result.SubfinderTaskCount++

		discoveredDomains, err := w.parseSubfinderOutput(subfinderTask.ID)
		if err != nil {
			continue
		}
		var domainAssets []*models.Asset
		for _, d := range discoveredDomains {
			tmpTarget := &models.Target{Type: models.TargetTypeDomain, Value: d.Host}
			decision, err := w.scope.Check(ctx, projectID, tmpTarget)
			if err != nil || decision.Decision == models.ScopeDeny {
				continue
			}
			a, _, err := w.merger.MergeOrCreateAsset(projectID, "domain", d.Host, "subfinder")
			if err != nil {
				continue
			}
			domainAssets = append(domainAssets, a)
			result.DomainsFound++
		}

		if len(domainAssets) == 0 {
			continue
		}
		httpxHostFile, err := writeHostsFile(w.dataDir, projectID, extractValues(domainAssets))
		if err != nil {
			log.Printf("write httpx host file: %v", err)
			continue
		}
		httpxTask, err := w.createAndRunTask(ctx, projectID, dt.ID, "httpx", worker.BuildHttpxCommand(httpxHostFile, 0, 0, ""))
		if err != nil {
			continue
		}
		result.HttpxTaskCount++

		webResults, err := w.parseHttpxOutput(httpxTask.ID)
		if err != nil {
			continue
		}
		for _, hr := range webResults {
			tmpTarget := &models.Target{Type: models.TargetTypeURL, Value: hr.URL}
			decision, err := w.scope.Check(ctx, projectID, tmpTarget)
			if err != nil || decision.Decision == models.ScopeDeny {
				continue
			}
			urlAsset, _, err := w.merger.MergeOrCreateAsset(projectID, "url", hr.URL, "httpx")
			if err != nil {
				continue
			}
			hostAsset, _, err := w.merger.MergeOrCreateAsset(projectID, "domain", hr.Host, "httpx")
			if err != nil {
				continue
			}
			var port *int
			if hr.Port != "" {
				if p, err := parseInt(hr.Port); err == nil {
					port = &p
				}
			}
			var statusCode *int
			if hr.StatusCode > 0 {
				sc := hr.StatusCode
				statusCode = &sc
			}
			techs := hr.Tech
			if hr.WebServer != "" {
				techs = append(techs, hr.WebServer)
			}
			_, _, err = w.merger.CreateWebEndpointIfNotExists(
				projectID, hostAsset.ID, hr.URL, hr.Scheme, hr.Host, port, hr.Path, hr.Title, statusCode, techs, "httpx",
			)
			if err != nil {
				continue
			}
			result.WebEndpointsFound++
			_ = urlAsset
		}

		naabuHostFile, err := writeHostsFile(w.dataDir, projectID, extractValues(domainAssets))
		if err != nil {
			log.Printf("write naabu host file: %v", err)
			continue
		}
		portRange := w.getPortRange(projectID)
		naabuArgs := buildNaabuArgsWithPortRange(naabuHostFile, portRange)
		naabuTask, err := w.createAndRunTask(ctx, projectID, dt.ID, "naabu", naabuArgs)
		if err != nil {
			continue
		}
		result.NaabuTaskCount++

		naabuResults, err := w.parseNaabuOutput(naabuTask.ID)
		if err != nil {
			continue
		}
		if err := w.runPostDiscovery(ctx, projectID, dt.ID, naabuResults, result, true); err != nil {
			log.Printf("post-discovery for domain %s: %v", dt.Value, err)
		}
	}
	return nil
}

// runIPChain runs naabu → post-discovery for each IP target.
func (w *AssetDiscoveryWorkflow) runIPChain(ctx context.Context, projectID string, ipTargets []*models.Target, result *DiscoveryResult) error {
	for _, target := range ipTargets {
		decision, err := w.scope.Check(ctx, projectID, target)
		if err != nil {
			continue
		}
		if decision.Decision == models.ScopeDeny {
			continue
		}

		hostFile, err := writeHostsFile(w.dataDir, projectID, []string{target.Value})
		if err != nil {
			log.Printf("write host file for ip %s: %v", target.Value, err)
			continue
		}

		portRange := w.getPortRange(projectID)
		args := buildNaabuArgsWithPortRange(hostFile, portRange)
		naabuTask, err := w.createAndRunTask(ctx, projectID, target.ID, "naabu", args)
		if err != nil {
			continue
		}
		result.NaabuTaskCount++

		naabuResults, err := w.parseNaabuOutput(naabuTask.ID)
		if err != nil {
			continue
		}
		if err := w.runPostDiscovery(ctx, projectID, target.ID, naabuResults, result, false); err != nil {
			log.Printf("post-discovery for ip %s: %v", target.Value, err)
		}
	}
	return nil
}

// runCIDRChain expands each CIDR, runs naabu, and processes results with secondary scope checks.
func (w *AssetDiscoveryWorkflow) runCIDRChain(ctx context.Context, projectID string, cidrTargets []*models.Target, result *DiscoveryResult) error {
	for _, target := range cidrTargets {
		decision, err := w.scope.Check(ctx, projectID, target)
		if err != nil {
			continue
		}
		if decision.Decision == models.ScopeDeny {
			continue
		}

		ips, err := scope.ExpandCIDR(target.Value)
		if err != nil {
			log.Printf("expand CIDR %s: %v", target.Value, err)
			continue
		}

		hostFile, err := writeHostsFile(w.dataDir, projectID, ips)
		if err != nil {
			log.Printf("write host file for cidr %s: %v", target.Value, err)
			continue
		}

		portRange := w.getPortRange(projectID)
		args := buildNaabuArgsWithPortRange(hostFile, portRange)
		naabuTask, err := w.createAndRunTask(ctx, projectID, target.ID, "naabu", args)
		if err != nil {
			continue
		}
		result.NaabuTaskCount++

		naabuResults, err := w.parseNaabuOutput(naabuTask.ID)
		if err != nil {
			continue
		}
		if err := w.runPostDiscovery(ctx, projectID, target.ID, naabuResults, result, true); err != nil {
			log.Printf("post-discovery for cidr %s: %v", target.Value, err)
		}
	}
	return nil
}

// runPostDiscovery processes Naabu results, creates IP/port assets, and runs httpx on alive IPs.
func (w *AssetDiscoveryWorkflow) runPostDiscovery(ctx context.Context, projectID, targetID string, naabuResults []parser.NaabuResult, result *DiscoveryResult, needsScopeCheck bool) error {
	var aliveIPs []string
	seen := make(map[string]bool)

	for _, nr := range naabuResults {
		if nr.IP == "" {
			continue
		}
		if needsScopeCheck {
			decision, err := w.scope.CheckIP(ctx, projectID, nr.IP)
			if err != nil || decision.Decision == models.ScopeDeny {
				continue
			}
		}
		ipAsset, _, err := w.merger.MergeOrCreateAsset(projectID, "ip", nr.IP, "naabu")
		if err != nil {
			continue
		}
		_, _, err = w.merger.CreatePortIfNotExists(ipAsset.ID, nr.Port, "tcp", "naabu")
		if err != nil {
			continue
		}
		result.IPsFound++
		result.PortsFound++

		if !seen[nr.IP] {
			seen[nr.IP] = true
			aliveIPs = append(aliveIPs, nr.IP)
		}
	}

	if len(aliveIPs) == 0 {
		return nil
	}

	// Build host:port targets so httpx can probe non-standard ports.
	var httpxTargets []string
	for _, ip := range aliveIPs {
		for _, nr := range naabuResults {
			if nr.IP == ip {
				httpxTargets = append(httpxTargets, fmt.Sprintf("%s:%d", nr.IP, nr.Port))
			}
		}
	}
	httpxTargets = dedupHTTPTargetsByOrigin(dedupStrings(httpxTargets))

	httpxHostFile, err := writeHostsFile(w.dataDir, projectID, httpxTargets)
	if err != nil {
		return fmt.Errorf("write httpx host file: %w", err)
	}

	httpxTask, err := w.createAndRunTask(ctx, projectID, targetID, "httpx", worker.BuildHttpxCommand(httpxHostFile, 0, 0, ""))
	if err != nil {
		return fmt.Errorf("run httpx: %w", err)
	}
	result.HttpxTaskCount++

	webResults, err := w.parseHttpxOutput(httpxTask.ID)
	if err != nil {
		return fmt.Errorf("parse httpx: %w", err)
	}

	for _, hr := range webResults {
		tmpTarget := &models.Target{Type: models.TargetTypeURL, Value: hr.URL}
		decision, err := w.scope.Check(ctx, projectID, tmpTarget)
		if err != nil || decision.Decision == models.ScopeDeny {
			continue
		}
		urlAsset, _, err := w.merger.MergeOrCreateAsset(projectID, "url", hr.URL, "httpx")
		if err != nil {
			continue
		}
		hostAsset, _, err := w.merger.MergeOrCreateAsset(projectID, "domain", hr.Host, "httpx")
		if err != nil {
			continue
		}
		var port *int
		if hr.Port != "" {
			if p, err := parseInt(hr.Port); err == nil {
				port = &p
			}
		}
		var statusCode *int
		if hr.StatusCode > 0 {
			sc := hr.StatusCode
			statusCode = &sc
		}
		techs := hr.Tech
		if hr.WebServer != "" {
			techs = append(techs, hr.WebServer)
		}
		_, _, err = w.merger.CreateWebEndpointIfNotExists(
			projectID, hostAsset.ID, hr.URL, hr.Scheme, hr.Host, port, hr.Path, hr.Title, statusCode, techs, "httpx",
		)
		if err != nil {
			continue
		}
		result.WebEndpointsFound++
		_ = urlAsset
	}

	return nil
}

// getPortRange returns the project's configured port range or the default "tp100".
func (w *AssetDiscoveryWorkflow) getPortRange(projectID string) string {
	project, _ := w.queries.GetProject(projectID)
	if project != nil {
		if project.PortRange != nil && *project.PortRange != "" {
			return *project.PortRange
		}
		// Fallback to pipeline config port_range
		if project.PipelineConfig != nil && *project.PipelineConfig != "" {
			var cfg models.PipelineConfig
			if err := json.Unmarshal([]byte(*project.PipelineConfig), &cfg); err == nil && cfg.PortRange != "" {
				return cfg.PortRange
			}
		}
	}
	return "tp100"
}

// buildNaabuArgsWithPortRange builds Naabu command arguments with the specified port range.
func buildNaabuArgsWithPortRange(hostFile, portRange string) []string {
	args := []string{"naabu", "-json", "-list", hostFile}
	switch strings.ToLower(portRange) {
	case "", "tp100", "top100":
		// default top 100, no extra args needed
	case "tp1000", "top1000":
		args = append(args, "-tp", "1000")
	case "tpfull", "topfull", "full":
		args = append(args, "-tp", "full")
	case "high-risk", "highrisk", "hr":
		args = append(args, "-p", worker.HighRiskPorts)
	default:
		args = append(args, "-p", portRange)
	}
	return args
}

// createAndRunTask creates a scan plan and task, then runs it.
func (w *AssetDiscoveryWorkflow) createAndRunTask(ctx context.Context, projectID, targetID, tool string, args []string) (*models.ScanTask, error) {
	plan := &models.ScanPlan{
		ID:           util.GenerateID(),
		ProjectID:    projectID,
		WorkflowType: "asset-discovery",
		Profile:      models.ProfileStandard,
		Status:       "approved",
		CreatedBy:    "system",
		CreatedAt:    time.Now().UTC(),
	}
	if err := w.queries.CreateScanPlan(plan); err != nil {
		return nil, fmt.Errorf("create plan: %w", err)
	}

	task := &models.ScanTask{
		ID:                util.GenerateID(),
		ProjectID:         projectID,
		PlanID:            plan.ID,
		TargetID:          &targetID,
		Tool:              tool,
		CommandTemplate:   args[0] + " " + joinArgs(args[1:]),
		ArgumentsRedacted: joinArgs(args[1:]),
		Status:            models.TaskQueued,
		CreatedAt:         time.Now().UTC(),
	}
	if w.runID != "" {
		task.RunID = &w.runID
	}
	// Record active custom bundle version for nuclei tasks
	if tool == "nuclei" {
		if version, err := w.queries.GetActiveNucleiCustomBundleVersion(); err == nil && version != "" {
			task.NucleiCustomBundleVersion = &version
		}
	}
	if err := w.queries.CreateScanTask(task); err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}

	if err := w.runner.Run(ctx, task.ID); err != nil {
		return task, err
	}
	return task, nil
}

func (w *AssetDiscoveryWorkflow) parseSubfinderOutput(taskID string) ([]parser.SubfinderResult, error) {
	path, err := w.findArtifactPath(taskID, models.ArtifactJSONL)
	if err != nil {
		return nil, err
	}
	if path == "" {
		path, err = w.findArtifactPath(taskID, models.ArtifactStdout)
		if err != nil {
			return nil, err
		}
	}
	if path == "" {
		return nil, fmt.Errorf("no artifact for task %s", taskID)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	results, errs := parser.ParseSubfinder(f)
	for _, e := range errs {
		log.Printf("subfinder parse error: %s", e.Error())
	}
	return results, nil
}

func (w *AssetDiscoveryWorkflow) parseHttpxOutput(taskID string) ([]parser.HTTPXResult, error) {
	path, err := w.findArtifactPath(taskID, models.ArtifactJSONL)
	if err != nil {
		return nil, err
	}
	if path == "" {
		path, err = w.findArtifactPath(taskID, models.ArtifactStdout)
		if err != nil {
			return nil, err
		}
	}
	if path == "" {
		return nil, fmt.Errorf("no artifact for task %s", taskID)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	results, errs := parser.ParseHTTPX(f)
	for _, e := range errs {
		log.Printf("httpx parse error: %s", e.Error())
	}
	return results, nil
}

func (w *AssetDiscoveryWorkflow) parseNaabuOutput(taskID string) ([]parser.NaabuResult, error) {
	path, err := w.findArtifactPath(taskID, models.ArtifactJSONL)
	if err != nil {
		return nil, err
	}
	if path == "" {
		// Try stdout as fallback.
		path, err = w.findArtifactPath(taskID, models.ArtifactStdout)
		if err != nil {
			return nil, err
		}
	}
	if path == "" {
		return nil, fmt.Errorf("no artifact for task %s", taskID)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	results, errs := parser.ParseNaabu(f)
	for _, e := range errs {
		log.Printf("naabu parse error: %s", e.Error())
	}
	return results, nil
}

func (w *AssetDiscoveryWorkflow) findArtifactPath(taskID string, artifactType models.ArtifactType) (string, error) {
	// 1. Try database first (local execution path)
	artifacts, err := w.queries.ListRawArtifactsByTask(taskID)
	if err != nil {
		return "", err
	}
	for _, a := range artifacts {
		if a.Type == artifactType {
			return a.Path, nil
		}
	}

	// 2. Fallback: check workdir for remote worker output files
	task, err := w.queries.GetScanTask(taskID)
	if err != nil || task == nil {
		return "", nil
	}
	workdir := filepath.Join(w.dataDir, "workdirs", task.ProjectID, taskID)

	switch artifactType {
	case models.ArtifactStdout, models.ArtifactJSONL:
		stdoutPath := filepath.Join(workdir, "stdout.txt")
		if _, err := os.Stat(stdoutPath); err == nil {
			return stdoutPath, nil
		}
	case models.ArtifactStderr:
		stderrPath := filepath.Join(workdir, "stderr.txt")
		if _, err := os.Stat(stderrPath); err == nil {
			return stderrPath, nil
		}
	}
	return "", nil
}

func extractValues(assets []*models.Asset) []string {
	seen := make(map[string]bool)
	var values []string
	for _, a := range assets {
		if !seen[a.Value] {
			seen[a.Value] = true
			values = append(values, a.Value)
		}
	}
	return values
}

func joinArgs(args []string) string {
	s := ""
	for i, a := range args {
		if i > 0 {
			s += " "
		}
		s += a
	}
	return s
}

func parseInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}

// writeHostsFile writes a list of hosts to a temporary file and returns the path.
// The file is placed under <dataDir>/workdirs/<projectID>/ so it shares the
// project's workdir lifecycle.
func writeHostsFile(dataDir, projectID string, hosts []string) (string, error) {
	dir := filepath.Join(dataDir, "workdirs", projectID, "hostlists")
	if err := os.MkdirAll(dir, 0750); err != nil {
		return "", fmt.Errorf("create hostlist dir: %w", err)
	}
	f, err := os.CreateTemp(dir, "hosts-*.txt")
	if err != nil {
		return "", fmt.Errorf("create hostlist file: %w", err)
	}
	defer f.Close()

	content := strings.Join(hosts, "\n") + "\n"
	if _, err := f.WriteString(content); err != nil {
		return "", fmt.Errorf("write hostlist: %w", err)
	}
	return f.Name(), nil
}
