package workflow

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"secbench/internal/asset"
	"secbench/internal/db"
	"secbench/internal/models"
	"secbench/internal/parser"
	"secbench/internal/scope"
	"secbench/internal/util"
	"secbench/internal/worker"
)

// AssetDiscoveryWorkflow runs subfinder → httpx → naabu in sequence for each domain target.
type AssetDiscoveryWorkflow struct {
	queries *db.Queries
	runner  *worker.Runner
	scope   *scope.Engine
	merger  *asset.Merger
	dataDir string
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
func (w *AssetDiscoveryWorkflow) Run(ctx context.Context, projectID string) (*DiscoveryResult, error) {
	result := &DiscoveryResult{}

	// 1. Get domain targets.
	targets, err := w.queries.ListTargetsByProject(projectID)
	if err != nil {
		return nil, fmt.Errorf("list targets: %w", err)
	}

	var domainTargets []*models.Target
	for _, t := range targets {
		if t.Type == models.TargetTypeDomain {
			domainTargets = append(domainTargets, t)
		}
	}

	if len(domainTargets) == 0 {
		return result, nil
	}

	// 2. For each domain target run subfinder → httpx → naabu serially.
	for _, dt := range domainTargets {
		// a. Scope check.
		decision, err := w.scope.Check(ctx, projectID, dt)
		if err != nil {
			continue
		}
		if decision.Decision == models.ScopeDeny {
			continue
		}

		// b. Run Subfinder.
		subfinderTask, err := w.createAndRunTask(ctx, projectID, dt.ID, "subfinder", worker.BuildSubfinderCommand(dt.Value))
		if err != nil {
			continue
		}
		result.SubfinderTaskCount++

		// c. Parse Subfinder output and create domain assets.
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

		// d. Run httpx on all discovered domain assets.
		if len(domainAssets) == 0 {
			continue
		}
		httpxHostFile, err := writeHostsFile(w.dataDir, projectID, extractValues(domainAssets))
		if err != nil {
			log.Printf("write httpx host file: %v", err)
			continue
		}
		httpxTask, err := w.createAndRunTask(ctx, projectID, dt.ID, "httpx", worker.BuildHttpxCommand(httpxHostFile))
		if err != nil {
			continue
		}
		result.HttpxTaskCount++

		// e. Parse httpx output and create web endpoints + url assets.
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
			// Ensure URL asset exists.
			urlAsset, _, err := w.merger.MergeOrCreateAsset(projectID, "url", hr.URL, "httpx")
			if err != nil {
				continue
			}
			// Create or get domain asset for host.
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
			// Merge webserver into technologies.
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

		// f. Run Naabu on all discovered domain assets.
		naabuHostFile, err := writeHostsFile(w.dataDir, projectID, extractValues(domainAssets))
		if err != nil {
			log.Printf("write naabu host file: %v", err)
			continue
		}
		naabuTask, err := w.createAndRunTask(ctx, projectID, dt.ID, "naabu", worker.BuildNaabuCommand(naabuHostFile))
		if err != nil {
			continue
		}
		result.NaabuTaskCount++

		// g. Parse Naabu output and create IP assets + ports.
		naabuResults, err := w.parseNaabuOutput(naabuTask.ID)
		if err != nil {
			continue
		}
		for _, nr := range naabuResults {
			if nr.IP == "" {
				continue
			}
			tmpTarget := &models.Target{Type: models.TargetTypeIP, Value: nr.IP}
			decision, err := w.scope.Check(ctx, projectID, tmpTarget)
			if err != nil || decision.Decision == models.ScopeDeny {
				continue
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
		}
	}

	return result, nil
}

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
	artifacts, err := w.queries.ListRawArtifactsByTask(taskID)
	if err != nil {
		return "", err
	}
	for _, a := range artifacts {
		if a.Type == artifactType {
			return a.Path, nil
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
