package workflow

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/P0m32Kun/Anchor/internal/cdn"
	"github.com/P0m32Kun/Anchor/internal/fingerprint"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/nuclei"
	"github.com/P0m32Kun/Anchor/internal/parser"
	"github.com/P0m32Kun/Anchor/internal/toolregistry"
	"github.com/P0m32Kun/Anchor/internal/toolrun"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// --- Tool execution helpers ---

func (p *Pipeline) runSubfinder(ctx context.Context, domain string) ([]string, error) {
	target := &models.Target{Type: models.TargetTypeDomain, Value: domain}
	decision, err := p.scope.ValidateBeforeRun(ctx, p.projectID, target, "")
	if err != nil || decision.Decision == models.ScopeDeny {
		return nil, fmt.Errorf("scope denied")
	}

	// If the user provided a custom provider-config, write it to a temp file
	// whose absolute path is embedded in the command. The dispatcher's
	// collectInputFiles will automatically sync it to remote workers.
	providerConfigPath := ""
	if p.config.SubfinderProviderConfig != "" {
		workdir := filepath.Join(p.dataDir, "workdirs", p.projectID)
		_ = os.MkdirAll(workdir, 0750)
		tmpFile := filepath.Join(workdir, fmt.Sprintf("subfinder-provider-%s.yaml", util.GenerateID()))
		if err := os.WriteFile(tmpFile, []byte(p.config.SubfinderProviderConfig), 0640); err != nil {
			log.Printf("[pipeline] write subfinder provider config: %v", err)
		} else {
			abs, err := filepath.Abs(tmpFile)
			if err == nil {
				providerConfigPath = abs
			}
		}
	}

	res := toolrun.Invoke(ctx, p.queries, p.runner, p.tools, toolrun.InvokeInput{
		ProjectID: p.projectID,
		RunID:     &p.runID,
		ToolID:    "subfinder",
		Params: toolregistry.RenderParams{
			"domain":              domain,
			"rate_limit":          p.config.SubfinderRateLimit,
			"threads":             p.config.SubfinderThreads,
			"timeout":             p.config.SubfinderTimeout,
			"mode":                p.config.SubfinderMode,
			"provider_config_path": providerConfigPath,
		},
	})
	if res.Err != nil {
		return nil, res.Err
	}

	subs := parser.ParseSubfinderOutput(bytes.NewReader(res.Stdout))
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

	res := toolrun.Invoke(ctx, p.queries, p.runner, p.tools, toolrun.InvokeInput{
		ProjectID: p.projectID,
		RunID:     &p.runID,
		ToolID:    "nmap_alive",
		Params: toolregistry.RenderParams{
			"host_file": hostFile,
		},
	})
	if res.Err != nil {
		return nil, res.Err
	}

	alive := parser.ParseNmapAlive(bytes.NewReader(res.Stdout))
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

	res := toolrun.Invoke(ctx, p.queries, p.runner, p.tools, toolrun.InvokeInput{
		ProjectID: p.projectID,
		RunID:     &p.runID,
		ToolID:    "naabu",
		Params: toolregistry.RenderParams{
			"host_file":  hostFile,
			"port_range": p.config.PortRange,
			"rate":       p.config.NaabuRate,
			"threads":    p.config.NaabuThreads,
			"timeout":    p.config.NaabuTimeout,
		},
	})
	if res.Err != nil {
		return nil, res.Err
	}

	ports := parser.ParseNaabuOutput(bytes.NewReader(res.Stdout))
	log.Printf("[pipeline] naabu parsed %d ports for project %s", len(ports), p.projectID)
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

	portsStr := make([]string, len(portList))
	for i, p := range portList {
		portsStr[i] = fmt.Sprintf("%d", p)
	}
	res := toolrun.Invoke(ctx, p.queries, p.runner, p.tools, toolrun.InvokeInput{
		ProjectID: p.projectID,
		RunID:     &p.runID,
		ToolID:    "nmap_service",
		Params: toolregistry.RenderParams{
			"host_file":    hostFile,
			"ports":        portsStr,
			"host_timeout": p.config.NmapServiceTimeout,
		},
	})
	if res.Err != nil {
		return nil, res.Err
	}

	results := fingerprint.ParseNmapXMLOutput(string(res.Stdout))
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

	res := toolrun.Invoke(ctx, p.queries, p.runner, p.tools, toolrun.InvokeInput{
		ProjectID: p.projectID,
		RunID:     &p.runID,
		ToolID:    "dnsx",
		Params: toolregistry.RenderParams{
			"host_file":  hostFile,
			"rate_limit": p.config.DNSxRateLimit,
			"threads":    p.config.DNSxThreads,
			"timeout":    p.config.DNSxTimeout,
		},
	})
	if res.Err != nil {
		return nil, res.Err
	}

	results := parser.ParseDNSxOutput(bytes.NewReader(res.Stdout))
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

// runCDNCheck runs cdncheck via the worker so stdout is persisted as a scan task
// (visible under the cdn_filter stage in Runs UI).
func (p *Pipeline) runCDNCheck(ctx context.Context, ips []string) ([]string, []models.CDNResult, error) {
	if len(ips) == 0 {
		return nil, nil, fmt.Errorf("no IPs to classify (DNS produced no A/AAAA records)")
	}
	input := strings.Join(ips, ",")
	res := toolrun.Invoke(ctx, p.queries, p.runner, p.tools, toolrun.InvokeInput{
		ProjectID: p.projectID,
		RunID:     &p.runID,
		ToolID:    "cdncheck",
		Params: toolregistry.RenderParams{
			"ips": input,
		},
	})
	if res.Err != nil {
		return nil, nil, fmt.Errorf("cdncheck: %w", res.Err)
	}
	return cdn.ParseJSONLOutput(res.Stdout, ips)
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

	res := toolrun.Invoke(ctx, p.queries, p.runner, p.tools, toolrun.InvokeInput{
		ProjectID: p.projectID,
		RunID:     &p.runID,
		ToolID:    "httpx",
		Params: toolregistry.RenderParams{
			"host_file":      hostFile,
			"rate_limit":     p.config.HttpxRateLimit,
			"threads":        p.config.HttpxThreads,
			"custom_fp_file": customFpFile,
		},
	})
	if res.Err != nil {
		return nil, res.Err
	}

	// Clean up temporary fingerprint file
	if customFpFile != "" {
		defer os.Remove(customFpFile)
	}

	endpoints := parser.ParseHttpxOutput(bytes.NewReader(res.Stdout))
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
		if p.config.NucleiRequireFingerprint && len(endpoints) > 0 {
			log.Printf("[pipeline] nuclei web: skipped %d endpoints (no fingerprint, nuclei_require_fingerprint=true)", len(endpoints))
		}
		return nil
	}

	urlToEndpoint := make(map[string]*models.WebEndpoint)
	for _, ep := range endpoints {
		urlToEndpoint[ep.URL] = ep
	}

	scanDepth := p.config.NucleiScanDepth
	useTags := scanDepth == "tags" || scanDepth == "both"
	useWf := scanDepth == "workflow" || scanDepth == "both"

	// Per-service workflow paths: /root/templates-{sourceID}/workflows/{tag}.yaml
	wfPaths := p.customWorkflowPaths()

	for tagKey, urls := range groups {
		tags := strings.Split(tagKey, ",")

		// Tag-based scan (if tags or both mode)
		if useTags {
			scanURLs := dedupHTTPTargetsByOrigin(dedupStrings(urls))
			targetFile := filepath.Join(p.dataDir, "workdirs", p.projectID, fmt.Sprintf("nuclei-%s.txt", util.GenerateID()))
			if err := os.WriteFile(targetFile, []byte(strings.Join(scanURLs, "\n")), 0640); err != nil {
				continue
			}
			if abs, err := filepath.Abs(targetFile); err == nil {
				targetFile = abs
			}
			res := toolrun.Invoke(ctx, p.queries, p.runner, p.tools, toolrun.InvokeInput{
				ProjectID: p.projectID,
				RunID:     &p.runID,
				ToolID:    "nuclei",
				Params: toolregistry.RenderParams{
					"target_file":       targetFile,
					"profile":           "deep",
					"tags":              tags,
					"scan_depth":        scanDepth,
					"concurrency":       p.config.NucleiConcurrency,
					"rate_limit":        p.config.NucleiRateLimit,
					"rate_limit_per_min": p.config.NucleiRateLimitPerMinute,
				},
			})
			if res.Err != nil {
				log.Printf("nuclei tags task for %s: %v", tagKey, res.Err)
			} else {
				p.saveNucleiFindings(res.Stdout, urlToEndpoint, nil)
			}
		}

		// Workflow scan (if workflow or both mode): one call per tag per source
		if useWf {
			for _, tag := range tags {
				for _, wfPath := range wfPaths {
					wfFile := filepath.Join(wfPath, tag+".yaml")
					scanURLs := dedupHTTPTargetsByOrigin(dedupStrings(urls))
					targetFile := filepath.Join(p.dataDir, "workdirs", p.projectID, fmt.Sprintf("nuclei-%s.txt", util.GenerateID()))
					if err := os.WriteFile(targetFile, []byte(strings.Join(scanURLs, "\n")), 0640); err != nil {
						continue
					}
					if abs, err := filepath.Abs(targetFile); err == nil {
						targetFile = abs
					}
					res := toolrun.Invoke(ctx, p.queries, p.runner, p.tools, toolrun.InvokeInput{
						ProjectID: p.projectID,
						RunID:     &p.runID,
						ToolID:    "nuclei",
						Params: toolregistry.RenderParams{
							"target_file":       targetFile,
							"profile":           "deep",
							"concurrency":       p.config.NucleiConcurrency,
							"rate_limit":        p.config.NucleiRateLimit,
							"rate_limit_per_min": p.config.NucleiRateLimitPerMinute,
							"template_path":     wfFile,
						},
					})
					if res.Err != nil {
						log.Printf("nuclei wf task %s for tag %s: %v", wfFile, tag, res.Err)
					} else {
						p.saveNucleiFindings(res.Stdout, urlToEndpoint, nil)
					}
				}
			}
		}
	}

	return nil
}

func (p *Pipeline) runNucleiNonWeb(ctx context.Context, results []fingerprint.NmapServiceResult) error {
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

	scanDepth := p.config.NucleiScanDepth
	useTags := scanDepth == "tags" || scanDepth == "both"
	useWf := scanDepth == "workflow" || scanDepth == "both"

	// Per-service workflow paths
	wfPaths := p.customWorkflowPaths()

	for tag, targets := range groups {
		// Tag-based scan
		if useTags {
			scanTargets := dedupHTTPTargetsByOrigin(dedupStrings(targets))
			targetFile := filepath.Join(p.dataDir, "workdirs", p.projectID, fmt.Sprintf("nuclei-%s.txt", util.GenerateID()))
			if err := os.WriteFile(targetFile, []byte(strings.Join(scanTargets, "\n")), 0640); err != nil {
				continue
			}
			if abs, err := filepath.Abs(targetFile); err == nil {
				targetFile = abs
			}
			res := toolrun.Invoke(ctx, p.queries, p.runner, p.tools, toolrun.InvokeInput{
				ProjectID: p.projectID,
				RunID:     &p.runID,
				ToolID:    "nuclei",
				Params: toolregistry.RenderParams{
					"target_file":       targetFile,
					"profile":           "deep",
					"tags":              []string{tag},
					"scan_depth":        scanDepth,
					"concurrency":       p.config.NucleiConcurrency,
					"rate_limit":        p.config.NucleiRateLimit,
					"rate_limit_per_min": p.config.NucleiRateLimitPerMinute,
				},
			})
			if res.Err != nil {
				log.Printf("nuclei tags task for %s: %v", tag, res.Err)
			} else {
				p.saveNucleiFindings(res.Stdout, nil, nil)
			}
		}

		// Workflow scan: one call per tag per source
		if useWf {
			for _, wfPath := range wfPaths {
				wfFile := filepath.Join(wfPath, tag+".yaml")
				scanTargets := dedupHTTPTargetsByOrigin(dedupStrings(targets))
				targetFile := filepath.Join(p.dataDir, "workdirs", p.projectID, fmt.Sprintf("nuclei-%s.txt", util.GenerateID()))
				if err := os.WriteFile(targetFile, []byte(strings.Join(scanTargets, "\n")), 0640); err != nil {
					continue
				}
				if abs, err := filepath.Abs(targetFile); err == nil {
					targetFile = abs
				}
				res := toolrun.Invoke(ctx, p.queries, p.runner, p.tools, toolrun.InvokeInput{
					ProjectID: p.projectID,
					RunID:     &p.runID,
					ToolID:    "nuclei",
					Params: toolregistry.RenderParams{
						"target_file":       targetFile,
						"profile":           "deep",
						"concurrency":       p.config.NucleiConcurrency,
						"rate_limit":        p.config.NucleiRateLimit,
						"rate_limit_per_min": p.config.NucleiRateLimitPerMinute,
						"template_path":     wfFile,
					},
				})
				if res.Err != nil {
					log.Printf("nuclei wf task %s for tag %s: %v", wfFile, tag, res.Err)
				} else {
					p.saveNucleiFindings(res.Stdout, nil, nil)
				}
			}
		}
	}

	return nil
}

// customWorkflowPaths returns the absolute paths to custom workflow directories
// on the worker. Since all custom templates live under ~/nuclei-templates/ (nuclei's
// default search path), we only need to point -w at the per-source workflow files.
// The worker path is /root/nuclei-templates/{install_path}/workflows/.
func (p *Pipeline) customWorkflowPaths() []string {
	sources, err := p.queries.ListNucleiCustomSources()
	if err != nil {
		log.Printf("[pipeline] list nuclei custom sources: %v", err)
		return nil
	}
	var paths []string
	for _, src := range sources {
		if !src.Enabled || src.InstallPath == "" {
			continue
		}
		paths = append(paths, filepath.Join("/root", "nuclei-templates", src.InstallPath, "workflows"))
	}
	return paths
}

func (p *Pipeline) createAndRunTask(ctx context.Context, tool string, args []string) (*models.ScanTask, []byte, error) {
	return p.legacyCreateAndRunTask(ctx, util.GenerateID(), tool, args)
}

// legacyCreateAndRunTask is the original implementation preserved for
// callers that still pass args directly (nuclei multi-round, discovery.go).
// Phase 3 will convert remaining callers to Registry.Render + toolrun.Invoke.
func (p *Pipeline) legacyCreateAndRunTask(ctx context.Context, taskID, tool string, args []string) (*models.ScanTask, []byte, error) {
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

	if tool == "nuclei" {
		if version, err := p.queries.GetActiveNucleiCustomBundleVersion(); err == nil && version != "" {
			task.NucleiCustomBundleVersion = &version
		}
	}

	if err := p.queries.CreateScanTask(task); err != nil {
		return nil, nil, fmt.Errorf("create scan task: %w", err)
	}

	if err := p.runner.Run(ctx, task.ID); err != nil {
		log.Printf("[pipeline] task %s (%s) run error: %v", task.ID, tool, err)
		stdout, _ := p.readTaskStdout(task.ID)
		return task, stdout, err
	}

	stdout, err := p.readTaskStdout(task.ID)
	if err != nil {
		log.Printf("[pipeline] task %s (%s) read stdout: %v", task.ID, tool, err)
	}

	return task, stdout, nil
}

// runFfuf runs a single ffuf brute-force against one web endpoint.
// Returns discovered URLs (url_list2).
func (p *Pipeline) runFfuf(ctx context.Context, endpoint *models.WebEndpoint, dictID string) ([]string, error) {
	if !p.config.EnableFfuf || dictID == "" {
		return nil, nil
	}

	dict, err := p.queries.GetDictionary(dictID)
	if err != nil || dict == nil {
		return nil, fmt.Errorf("dictionary not found: %s", dictID)
	}
	if !dict.Enabled {
		return nil, fmt.Errorf("dictionary disabled: %s", dictID)
	}

	// Build target URL with FUZZ placeholder
	base := strings.TrimSuffix(endpoint.URL, "/")
	targetURL := base + "/FUZZ"

	// Build and run via worker
	res := toolrun.Invoke(ctx, p.queries, p.runner, p.tools, toolrun.InvokeInput{
		ProjectID: p.projectID,
		RunID:     &p.runID,
		ToolID:    "ffuf",
		Params: toolregistry.RenderParams{
			"target":     targetURL,
			"wordlist":   dict.FilePath,
			"rate_limit": p.config.FfufRateLimit,
			"timeout":    p.config.FfufTimeout,
		},
	})
	if res.Err != nil {
		return nil, res.Err
	}

	results, _ := parser.ParseFfufOutput(bytes.NewReader(res.Stdout))
	var urls []string
	for _, r := range results {
		if r.URL != "" {
			urls = append(urls, r.URL)
		}
	}
	return urls, nil
}

// recordPassiveTask creates a completed ScanTask with a stdout artifact for
// passive/API-based pipeline steps (FOFA, Hunter, Quake, crt.sh, gau). These
// steps don't invoke external tools via the worker — they call HTTP APIs
// directly — so there is no real subprocess stdout. This helper synthesises a
// task+artifact pair so the frontend can display the raw API response in the
// pipeline detail view for auditability.
func (p *Pipeline) recordPassiveTask(tool string, summary string, data []byte) {
	taskID := util.GenerateID()
	now := time.Now().UTC()

	task := &models.ScanTask{
		ID:              taskID,
		ProjectID:       p.projectID,
		RunID:           &p.runID,
		Tool:            tool,
		CommandTemplate: summary,
		Status:          models.TaskCompleted,
		CreatedAt:       now,
		StartedAt:       &now,
		FinishedAt:      &now,
	}
	if err := p.queries.CreateScanTask(task); err != nil {
		log.Printf("[pipeline] record passive task %s: %v", tool, err)
		return
	}

	// Save stdout artifact so it's fetchable via /tasks/{id}/artifacts
	workdir := filepath.Join(p.dataDir, "workdirs", p.projectID, taskID)
	_ = os.MkdirAll(workdir, 0750)
	filename := fmt.Sprintf("stdout_%d.json", time.Now().UnixNano())
	path := filepath.Join(workdir, filename)
	if err := os.WriteFile(path, data, 0640); err != nil {
		log.Printf("[pipeline] write passive artifact %s: %v", tool, err)
		return
	}

	sum := sha256.Sum256(data)
	a := &models.RawArtifact{
		ID:              util.GenerateID(),
		ProjectID:       p.projectID,
		TaskID:          &taskID,
		Type:            models.ArtifactStdout,
		Path:            path,
		SHA256:          fmt.Sprintf("%x", sum),
		Size:            int64(len(data)),
		RedactionStatus: "unchecked",
		CreatedAt:       now,
	}
	if err := p.queries.CreateRawArtifact(a); err != nil {
		log.Printf("[pipeline] create passive artifact %s: %v", tool, err)
	}
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
