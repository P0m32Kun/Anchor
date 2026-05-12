package workflow

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/nuclei"
	"github.com/P0m32Kun/Anchor/internal/parser"
	"github.com/P0m32Kun/Anchor/internal/util"
	"github.com/P0m32Kun/Anchor/internal/worker"
)

// SlowScanOrchestrator manages background slow scanning tasks (urlfinder + ffuf).
// It runs after the main pipeline completes and feeds discovered URLs back into
// the httpx -> nuclei loop.
type SlowScanOrchestrator struct {
	queries   *db.Queries
	runner    *worker.Runner
	dataDir   string
	config    models.PipelineConfig
	projectID string
	// sem controls concurrency — max N slow scan tasks running at once.
	sem chan struct{}
}

// NewSlowScanOrchestrator creates a new orchestrator with default concurrency limit.
func NewSlowScanOrchestrator(queries *db.Queries, runner *worker.Runner, dataDir string) *SlowScanOrchestrator {
	return &SlowScanOrchestrator{
		queries: queries,
		runner:  runner,
		dataDir: dataDir,
		sem:     make(chan struct{}, 5), // max 5 concurrent slow scans
	}
}

func (s *SlowScanOrchestrator) WithConfig(cfg models.PipelineConfig) *SlowScanOrchestrator {
	s.config = cfg
	return s
}

// Run executes slow scans for all domain/URL targets in a project.
// It blocks until all tasks complete (or context is cancelled).
func (s *SlowScanOrchestrator) Run(ctx context.Context, projectID string, runID string) error {
	s.projectID = projectID

	targets, err := s.queries.ListTargetsByProject(projectID)
	if err != nil {
		return fmt.Errorf("list targets: %w", err)
	}

	var wg sync.WaitGroup
	var discoveredURLs []string
	var mu sync.Mutex

	for _, t := range targets {
		if t.Type != models.TargetTypeDomain && t.Type != models.TargetTypeURL {
			continue
		}

		if s.config.EnableUrlfinder {
			wg.Add(1)
			go func(target *models.Target) {
				defer wg.Done()
				urls, err := s.runUrlfinder(ctx, target, runID)
				if err != nil {
					log.Printf("[slow-scan] urlfinder %s: %v", target.Value, err)
					return
				}
				mu.Lock()
				discoveredURLs = append(discoveredURLs, urls...)
				mu.Unlock()
			}(t)
		}

		if s.config.EnableFfuf && s.config.FfufDictionaryID != "" {
			wg.Add(1)
			go func(target *models.Target) {
				defer wg.Done()
				urls, err := s.runFfuf(ctx, target, runID)
				if err != nil {
					log.Printf("[slow-scan] ffuf %s: %v", target.Value, err)
					return
				}
				mu.Lock()
				discoveredURLs = append(discoveredURLs, urls...)
				mu.Unlock()
			}(t)
		}
	}

	wg.Wait()

	// Deduplicate and feed to httpx -> nuclei
	if len(discoveredURLs) > 0 {
		unique := dedupStrings(discoveredURLs)
		if err := s.feedToHttpxNuclei(ctx, unique); err != nil {
			log.Printf("[slow-scan] feed to httpx/nuclei: %v", err)
		}
	}

	return nil
}

func (s *SlowScanOrchestrator) runUrlfinder(ctx context.Context, target *models.Target, runID string) ([]string, error) {
	s.sem <- struct{}{}
	defer func() { <-s.sem }()

	// Create target file
	targetFile := filepath.Join(s.dataDir, "workdirs", s.projectID, fmt.Sprintf("urlfinder-%s.txt", util.GenerateID()))
	if err := os.MkdirAll(filepath.Dir(targetFile), 0750); err != nil {
		return nil, err
	}
	if err := os.WriteFile(targetFile, []byte(target.Value), 0640); err != nil {
		return nil, err
	}

	// Create slow scan task record
	task := &models.SlowScanTask{
		ID:        util.GenerateID(),
		ProjectID: s.projectID,
		TargetID:  &target.ID,
		RunID:     &runID,
		Tool:      models.SlowScanToolUrlfinder,
		Status:    models.SlowScanPending,
		RateLimit: s.config.UrlfinderRateLimit,
		Timeout:   s.config.UrlfinderTimeout,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := s.queries.CreateSlowScanTask(task); err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}

	// Build and run via worker
	cmd := worker.BuildUrlfinderCommand(targetFile, s.config.UrlfinderRateLimit, s.config.UrlfinderTimeout)
	scanTask := &models.ScanTask{
		ID:              util.GenerateID(),
		ProjectID:       s.projectID,
		RunID:           &runID,
		TargetID:        &target.ID,
		Tool:            "urlfinder",
		CommandTemplate: strings.Join(cmd, " "),
		Status:          models.TaskCreated,
		CreatedAt:       time.Now().UTC(),
	}
	if err := s.queries.CreateScanTask(scanTask); err != nil {
		return nil, fmt.Errorf("create scan task: %w", err)
	}

	now := time.Now().UTC()
	s.queries.SetSlowScanRunning(task.ID, now)

	if err := s.runner.Run(ctx, scanTask.ID); err != nil {
		s.queries.UpdateSlowScanStatus(task.ID, models.SlowScanFailed, err.Error(), &now)
		return nil, err
	}

	stdout, err := s.readTaskStdout(scanTask.ID)
	if err != nil {
		s.queries.UpdateSlowScanStatus(task.ID, models.SlowScanFailed, err.Error(), &now)
		return nil, err
	}

	results, _ := parser.ParseUrlfinderOutput(bytes.NewReader(stdout))
	var urls []string
	for _, r := range results {
		if r.URL != "" {
			urls = append(urls, r.URL)
		}
	}

	s.queries.UpdateSlowScanStatus(task.ID, models.SlowScanCompleted, "", &now)
	return urls, nil
}

func (s *SlowScanOrchestrator) runFfuf(ctx context.Context, target *models.Target, runID string) ([]string, error) {
	s.sem <- struct{}{}
	defer func() { <-s.sem }()

	// Get dictionary
	dict, err := s.queries.GetDictionary(s.config.FfufDictionaryID)
	if err != nil || dict == nil {
		return nil, fmt.Errorf("dictionary not found: %s", s.config.FfufDictionaryID)
	}

	// Build target URL with FUZZ placeholder
	var targetURL string
	if target.Type == models.TargetTypeDomain {
		targetURL = fmt.Sprintf("https://%s/FUZZ", target.Value)
	} else {
		base := strings.TrimSuffix(target.Value, "/")
		targetURL = base + "/FUZZ"
	}

	// Create slow scan task record
	task := &models.SlowScanTask{
		ID:        util.GenerateID(),
		ProjectID: s.projectID,
		TargetID:  &target.ID,
		RunID:     &runID,
		Tool:      models.SlowScanToolFfuf,
		Status:    models.SlowScanPending,
		RateLimit: s.config.FfufRateLimit,
		Timeout:   s.config.FfufTimeout,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := s.queries.CreateSlowScanTask(task); err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}

	// Build and run via worker
	cmd := worker.BuildFfufCommand(targetURL, dict.FilePath, s.config.FfufRateLimit, s.config.FfufTimeout)
	scanTask := &models.ScanTask{
		ID:              util.GenerateID(),
		ProjectID:       s.projectID,
		RunID:           &runID,
		TargetID:        &target.ID,
		Tool:            "ffuf",
		CommandTemplate: strings.Join(cmd, " "),
		Status:          models.TaskCreated,
		CreatedAt:       time.Now().UTC(),
	}
	if err := s.queries.CreateScanTask(scanTask); err != nil {
		return nil, fmt.Errorf("create scan task: %w", err)
	}

	now := time.Now().UTC()
	s.queries.SetSlowScanRunning(task.ID, now)

	if err := s.runner.Run(ctx, scanTask.ID); err != nil {
		s.queries.UpdateSlowScanStatus(task.ID, models.SlowScanFailed, err.Error(), &now)
		return nil, err
	}

	stdout, err := s.readTaskStdout(scanTask.ID)
	if err != nil {
		s.queries.UpdateSlowScanStatus(task.ID, models.SlowScanFailed, err.Error(), &now)
		return nil, err
	}

	results, _ := parser.ParseFfufOutput(bytes.NewReader(stdout))
	var urls []string
	for _, r := range results {
		if r.URL != "" {
			urls = append(urls, r.URL)
		}
	}

	s.queries.UpdateSlowScanStatus(task.ID, models.SlowScanCompleted, "", &now)
	return urls, nil
}

// feedToHttpxNuclei runs httpx on discovered URLs, saves new WebEndpoints, then runs nuclei.
func (s *SlowScanOrchestrator) feedToHttpxNuclei(ctx context.Context, urls []string) error {
	if len(urls) == 0 {
		return nil
	}

	// Deduplicate against existing web endpoints
	existing, err := s.queries.ListWebEndpointsByProject(s.projectID)
	if err != nil {
		log.Printf("[slow-scan] list existing endpoints: %v", err)
	}
	existingSet := make(map[string]bool)
	for _, ep := range existing {
		existingSet[ep.URL] = true
	}

	var newURLs []string
	for _, u := range urls {
		if !existingSet[u] {
			newURLs = append(newURLs, u)
		}
	}
	if len(newURLs) == 0 {
		log.Printf("[slow-scan] all %d discovered URLs already exist, skipping", len(urls))
		return nil
	}
	log.Printf("[slow-scan] feeding %d new URLs to httpx/nuclei (deduped from %d)", len(newURLs), len(urls))

	// Write to temp file
	hostFile := filepath.Join(s.dataDir, "workdirs", s.projectID, fmt.Sprintf("slowscan-httpx-%s.txt", util.GenerateID()))
	if err := os.MkdirAll(filepath.Dir(hostFile), 0750); err != nil {
		return err
	}
	if err := os.WriteFile(hostFile, []byte(strings.Join(newURLs, "\n")), 0640); err != nil {
		return err
	}

	// Run httpx
	cmd := worker.BuildHttpxCommand(hostFile, s.config.HttpxRateLimit, s.config.HttpxThreads)
	scanTask := &models.ScanTask{
		ID:              util.GenerateID(),
		ProjectID:       s.projectID,
		Tool:            "httpx",
		CommandTemplate: strings.Join(cmd, " "),
		Status:          models.TaskCreated,
		CreatedAt:       time.Now().UTC(),
	}
	if err := s.queries.CreateScanTask(scanTask); err != nil {
		return fmt.Errorf("create httpx task: %w", err)
	}
	if err := s.runner.Run(ctx, scanTask.ID); err != nil {
		log.Printf("[slow-scan] httpx run error: %v", err)
		return err
	}

	stdout, err := s.readTaskStdout(scanTask.ID)
	if err != nil {
		return fmt.Errorf("read httpx stdout: %w", err)
	}

	endpoints := parser.ParseHttpxOutput(bytes.NewReader(stdout))
	var savedEndpoints []*models.WebEndpoint
	for _, ep := range endpoints {
		ep.ID = util.GenerateID()
		ep.ProjectID = s.projectID
		ep.CreatedAt = time.Now().UTC()
		if err := s.queries.CreateWebEndpoint(ep); err != nil {
			log.Printf("[slow-scan] save endpoint %s: %v", ep.URL, err)
			continue
		}
		savedEndpoints = append(savedEndpoints, ep)
	}
	log.Printf("[slow-scan] saved %d new web endpoints", len(savedEndpoints))

	if len(savedEndpoints) == 0 {
		return nil
	}

	// Run nuclei on new endpoints
	groups := make(map[string][]string)
	for _, ep := range savedEndpoints {
		key := strings.Join(nuclei.MapPreciseTags(ep.Technologies, ""), ",")
		if key == "" {
			key = "generic"
		}
		groups[key] = append(groups[key], ep.URL)
	}

	for tagKey, urls := range groups {
		tags := strings.Split(tagKey, ",")
		targetFile := filepath.Join(s.dataDir, "workdirs", s.projectID, fmt.Sprintf("slowscan-nuclei-%s.txt", util.GenerateID()))
		if err := os.WriteFile(targetFile, []byte(strings.Join(urls, "\n")), 0640); err != nil {
			continue
		}

		cmd := worker.BuildNucleiCommand(targetFile, "deep", s.config.NucleiRateLimit, s.config.NucleiRateLimitPerMinute, s.config.NucleiConcurrency, tags, s.config.NucleiScanDepth, "")
		scanTask := &models.ScanTask{
			ID:              util.GenerateID(),
			ProjectID:       s.projectID,
			Tool:            "nuclei",
			CommandTemplate: strings.Join(cmd, " "),
			Status:          models.TaskCreated,
			CreatedAt:       time.Now().UTC(),
		}
		if err := s.queries.CreateScanTask(scanTask); err != nil {
			continue
		}
		if err := s.runner.Run(ctx, scanTask.ID); err != nil {
			log.Printf("[slow-scan] nuclei run error for tags %s: %v", tagKey, err)
			continue
		}
	}

	return nil
}

func (s *SlowScanOrchestrator) readTaskStdout(taskID string) ([]byte, error) {
	artifacts, err := s.queries.ListRawArtifactsByTask(taskID)
	if err != nil {
		return nil, err
	}
	for _, a := range artifacts {
		if a.Type == models.ArtifactStdout {
			return os.ReadFile(a.Path)
		}
	}
	return nil, fmt.Errorf("no stdout artifact for task %s", taskID)
}
