package workflow

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/P0m32Kun/Anchor/internal/fingerprint"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/nuclei"
	"github.com/P0m32Kun/Anchor/internal/parser"
	"github.com/P0m32Kun/Anchor/internal/util"
	"github.com/P0m32Kun/Anchor/internal/worker"
)

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

func (p *Pipeline) runNmapAlive(ctx context.Context, hosts []string) ([]string, error) {
	if len(hosts) == 0 {
		return nil, nil
	}

	hostFile := filepath.Join(p.dataDir, "workdirs", p.projectID, fmt.Sprintf("nmap-%s.txt", util.GenerateID()))
	if err := os.MkdirAll(filepath.Dir(hostFile), 0750); err != nil {
		return nil, err
	}
	if err := os.WriteFile(hostFile, []byte(strings.Join(hosts, "\n")), 0640); err != nil {
		return nil, err
	}
	if abs, err := filepath.Abs(hostFile); err == nil {
		hostFile = abs
	}

	task, stdout, err := p.createAndRunTask(ctx, "nmap", worker.BuildNmapAliveCommand(hostFile))
	if err != nil {
		return nil, err
	}
	_ = task

	alive := parser.ParseNmapAlive(bytes.NewReader(stdout))
	log.Printf("[pipeline] nmap alive: input=%d alive=%d for project %s", len(hosts), len(alive), p.projectID)
	return alive, nil
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

func (p *Pipeline) runNmapServiceScan(ctx context.Context, ports []parser.PortInfo) ([]fingerprint.NmapServiceResult, error) {
	if len(ports) == 0 {
		return nil, nil
	}

	// Collect unique IPs and ports.
	ipSet := make(map[string]bool)
	portSet := make(map[int]bool)
	for _, port := range ports {
		ipSet[port.IP] = true
		portSet[port.Port] = true
	}
	var hosts []string
	for ip := range ipSet {
		hosts = append(hosts, ip)
	}
	var portList []int
	for port := range portSet {
		portList = append(portList, port)
	}

	hostFile := filepath.Join(p.dataDir, "workdirs", p.projectID, fmt.Sprintf("nmap-sv-%s.txt", util.GenerateID()))
	if err := os.MkdirAll(filepath.Dir(hostFile), 0750); err != nil {
		return nil, err
	}
	if err := os.WriteFile(hostFile, []byte(strings.Join(hosts, "\n")), 0640); err != nil {
		return nil, err
	}
	if abs, err := filepath.Abs(hostFile); err == nil {
		hostFile = abs
	}

	cmd := worker.BuildNmapServiceScanCommand(hostFile, portList, p.config.NmapServiceTimeout)
	task, stdout, err := p.createAndRunTask(ctx, "nmap", cmd)
	if err != nil {
		return nil, err
	}
	_ = task

	results := fingerprint.ParseNmapXMLOutput(string(stdout))
	log.Printf("[pipeline] nmap -sV: input=%d ports on %d hosts, results=%d for project %s", len(ports), len(hosts), len(results), p.projectID)
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

	cmd := worker.BuildDNSxCommand(hostFile, nil, p.config.DNSxRateLimit, p.config.DNSxThreads, p.config.DNSxTimeout)
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

	// Get enabled custom fingerprint files (all types merged into one file)
	customFpFile, err := p.prepareHttpxFingerprints()
	if err != nil {
		log.Printf("[pipeline] prepare httpx fingerprints: %v", err)
		// Continue without custom fingerprints
	}

	task, stdout, err := p.createAndRunTask(ctx, "httpx", worker.BuildHttpxCommand(hostFile, p.config.HttpxRateLimit, p.config.HttpxThreads, customFpFile))
	if err != nil {
		return nil, err
	}
	_ = task

	// Clean up temporary fingerprint file
	if customFpFile != "" {
		defer os.Remove(customFpFile)
	}

	endpoints := parser.ParseHttpxOutput(bytes.NewReader(stdout))
	var saved []*models.WebEndpoint
	for _, ep := range endpoints {
		host := ep.Host
		if host == "" {
			continue
		}
		assetType := "domain"
		if net.ParseIP(host) != nil {
			assetType = "ip"
		}
		hostAsset, _, err := p.merger.MergeOrCreateAsset(p.projectID, assetType, host, "httpx")
		if err != nil {
			log.Printf("[pipeline] merge/create asset %s: %v", host, err)
			continue
		}
		we, _, err := p.merger.CreateWebEndpointIfNotExists(
			p.projectID, hostAsset.ID, ep.URL, ep.Scheme, ep.Host,
			ep.Port, ep.Path, ep.Title, ep.StatusCode, ep.Technologies, "httpx",
		)
		if err != nil {
			log.Printf("[pipeline] save web endpoint %s: %v", ep.URL, err)
			continue
		}
		if we != nil {
			saved = append(saved, we)
		}
	}
	return saved, nil
}

// prepareHttpxFingerprints collects all enabled custom fingerprint files and
// returns the path to a single merged temporary file for httpx -cff.
// Returns empty string if no enabled fingerprints exist.
func (p *Pipeline) prepareHttpxFingerprints() (customFpFile string, err error) {
	fingerprints, err := p.queries.ListEnabledHttpxFingerprints("")
	if err != nil {
		log.Printf("[pipeline] list enabled fingerprints: %v", err)
		return "", err
	}
	if len(fingerprints) == 0 {
		return "", nil
	}

	customFpFile, err = p.mergeFingerprintFiles(fingerprints)
	if err != nil {
		log.Printf("[pipeline] merge fingerprint files: %v", err)
		return "", err
	}

	return customFpFile, nil
}

// mergeFingerprintFiles merges multiple fingerprint files into a single temporary file.
// Returns the path to the temporary file.
func (p *Pipeline) mergeFingerprintFiles(fingerprints []*models.HttpxFingerprint) (string, error) {
	// Create a temporary file in the workdir
	workDir := filepath.Join(p.dataDir, "workdirs", p.projectID)
	if err := os.MkdirAll(workDir, 0750); err != nil {
		return "", err
	}

	tempFile := filepath.Join(workDir, fmt.Sprintf("httpx-cff-%s.tmp", util.GenerateID()))
	var mergedContent []byte

	for _, fp := range fingerprints {
		// Read the fingerprint file content
		content, err := os.ReadFile(fp.FilePath)
		if err != nil {
			log.Printf("[pipeline] read fingerprint file %s: %v", fp.FilePath, err)
			continue
		}
		// Merge: for JSON files, we need to merge the arrays
		// For simplicity, we just concatenate the content (assuming line-delimited JSON)
		if len(mergedContent) > 0 {
			mergedContent = append(mergedContent, '\n')
		}
		mergedContent = append(mergedContent, content...)
	}

	if len(mergedContent) == 0 {
		return "", fmt.Errorf("no valid fingerprint content")
	}

	if err := os.WriteFile(tempFile, mergedContent, 0640); err != nil {
		return "", err
	}

	// Return absolute path
	abs, err := filepath.Abs(tempFile)
	if err != nil {
		return tempFile, nil
	}
	return abs, nil
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

	if p.config.NucleiScanDepth == "workflow" {
		// Workflow mode: run each tag's workflow file separately for precision
		// tagKey format: "tag1,tag2" → split and run each workflow individually
		tagToURLs := make(map[string][]string)
		for tagKey, urls := range groups {
			tags := strings.Split(tagKey, ",")
			for _, tag := range tags {
				tagToURLs[tag] = append(tagToURLs[tag], urls...)
			}
		}

		for tag, urls := range tagToURLs {
			targetFile := filepath.Join(p.dataDir, "workdirs", p.projectID, fmt.Sprintf("nuclei-%s.txt", util.GenerateID()))
			if err := os.WriteFile(targetFile, []byte(strings.Join(urls, "\n")), 0640); err != nil {
				continue
			}
			if abs, err := filepath.Abs(targetFile); err == nil {
				targetFile = abs
			}

			templatePath := fmt.Sprintf("workflows/%s.yaml", tag)
			task, stdout, err := p.createAndRunTask(ctx, "nuclei", worker.BuildNucleiCommand(targetFile, "deep", p.config.NucleiRateLimit, p.config.NucleiRateLimitPerMinute, p.config.NucleiConcurrency, nil, p.config.NucleiScanDepth, "", templatePath))
			if err != nil {
				log.Printf("nuclei task for workflow %s: %v", tag, err)
				continue
			}
			_ = task
			p.saveNucleiFindings(stdout, urlToEndpoint, nil)
		}
	} else {
		// Tags or both mode: use tag-based matching
		for tagKey, urls := range groups {
			tags := strings.Split(tagKey, ",")
			targetFile := filepath.Join(p.dataDir, "workdirs", p.projectID, fmt.Sprintf("nuclei-%s.txt", util.GenerateID()))
			if err := os.WriteFile(targetFile, []byte(strings.Join(urls, "\n")), 0640); err != nil {
				continue
			}
			if abs, err := filepath.Abs(targetFile); err == nil {
				targetFile = abs
			}

			task, stdout, err := p.createAndRunTask(ctx, "nuclei", worker.BuildNucleiCommand(targetFile, "deep", p.config.NucleiRateLimit, p.config.NucleiRateLimitPerMinute, p.config.NucleiConcurrency, tags, p.config.NucleiScanDepth, DefaultWorkflowDir, ""))
			if err != nil {
				log.Printf("nuclei task for tags %s: %v", tagKey, err)
				continue
			}
			_ = task
			p.saveNucleiFindings(stdout, urlToEndpoint, nil)
		}
	}

	return nil
}

func (p *Pipeline) runNucleiNonWeb(ctx context.Context, results []fingerprint.NmapServiceResult) error {
	// Group by service tag
	groups := make(map[string][]string)
	for _, r := range results {
		tags := nuclei.MapServiceToTags(r.Service)
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

		// When scanDepth is "workflow", use -t workflows/{tag}.yaml instead of -tags
		// This allows precise per-service template targeting (e.g., workflows/mysql.yaml)
		var tags []string
		var templatePath string
		if p.config.NucleiScanDepth == "workflow" {
			templatePath = fmt.Sprintf("workflows/%s.yaml", tag)
		} else {
			tags = []string{tag}
		}

		task, stdout, err := p.createAndRunTask(ctx, "nuclei", worker.BuildNucleiCommand(targetFile, "deep", p.config.NucleiRateLimit, p.config.NucleiRateLimitPerMinute, p.config.NucleiConcurrency, tags, p.config.NucleiScanDepth, "", templatePath))
		if err != nil {
			log.Printf("nuclei task for tag %s: %v", tag, err)
			continue
		}
		_ = task
		p.saveNucleiFindings(stdout, nil, nil)
	}

	return nil
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

	// Record active custom bundle version for nuclei tasks
	if tool == "nuclei" {
		if version, err := p.queries.GetActiveNucleiCustomBundleVersion(); err == nil && version != "" {
			task.NucleiCustomBundleVersion = &version
		}
	}

	if err := p.queries.CreateScanTask(task); err != nil {
		return nil, nil, fmt.Errorf("create scan task: %w", err)
	}

	// Run task synchronously via worker
	if err := p.runner.Run(ctx, task.ID); err != nil {
		log.Printf("[pipeline] task %s (%s) run error: %v", task.ID, tool, err)
		stdout, _ := p.readTaskStdout(task.ID)
		return task, stdout, err
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
