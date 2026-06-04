package scanengine

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/P0m32Kun/Anchor/internal/asset"
	"github.com/P0m32Kun/Anchor/internal/cdn"
	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/exclude"
	"github.com/P0m32Kun/Anchor/internal/finding"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/scope"
	"github.com/P0m32Kun/Anchor/internal/parser"
	"github.com/P0m32Kun/Anchor/internal/scanengine/core"
	"github.com/P0m32Kun/Anchor/internal/scanengine/dedup"
	"github.com/P0m32Kun/Anchor/internal/scanengine/executor"
	"github.com/P0m32Kun/Anchor/internal/scanengine/queue"
	"github.com/P0m32Kun/Anchor/internal/scanengine/seed"
	"github.com/P0m32Kun/Anchor/internal/scanengine/stageagg"
	"github.com/P0m32Kun/Anchor/internal/scanengine/work"
	"github.com/P0m32Kun/Anchor/internal/toolregistry"
	"github.com/P0m32Kun/Anchor/internal/util"
	"github.com/P0m32Kun/Anchor/internal/worker"
)

// EngineConfig holds configuration for the ScanEngine.
type EngineConfig struct {
	IdleTimeout     time.Duration // how long to wait for new assets before wind_down
	AbsoluteTimeout time.Duration // hard limit on run duration
	SchedulerTick   time.Duration // how often the scheduler checks for pending work
	BatchSize       int           // max concurrent work items
	Pipeline        models.PipelineConfig // tool-specific settings (rate, threads, timeout, port_range, etc.)
}

// DefaultEngineConfig returns sensible defaults.
func DefaultEngineConfig() EngineConfig {
	return EngineConfig{
		IdleTimeout:     5 * time.Minute,
		AbsoluteTimeout: 30 * time.Minute,
		SchedulerTick:   2 * time.Second,
		BatchSize:       5,
		Pipeline:        models.DefaultPipelineConfig(),
	}
}

// ScanEngine is the asset-driven scan engine.
type ScanEngine struct {
	queries    *db.Queries
	merger     *asset.Merger
	store      *work.Store
	exec       executor.Executor
	agg        *stageagg.Aggregator
	dedup      *dedup.RunDedup
	pq         *queue.PriorityQueue
	config     EngineConfig
	profile    core.Profile
	excludeMgr     *exclude.Manager
	scopeEng       *scope.Engine
	nucleiPersist  *finding.NucleiPersister
	dataDir        string
	runID          string
	projectID      string
	assetDepth     sync.Map // assetID -> int discovery depth

	mu             sync.Mutex
	engineState    string
	startedAt      time.Time     // when the engine started (for absolute timeout)
	lastNewAssetAt time.Time     // last time a new asset was discovered (for idle timeout)
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	inFlight       int32
	sem            chan struct{} // concurrency limiter
	maxRetries     int          // max retries for failed work items
}

// New creates a new ScanEngine.
func New(
	queries *db.Queries,
	runner *worker.Runner,
	tools *toolregistry.Registry,
	merger *asset.Merger,
	profile core.Profile,
	excludeMgr *exclude.Manager,
	scopeEng *scope.Engine,
	dataDir string,
	runID string,
	projectID string,
	config EngineConfig,
	stageCallback stageagg.StageEventCallback,
) *ScanEngine {
	exec := executor.NewToolExecutor(queries, runner, tools, merger, dataDir)
	return NewWithExecutor(queries, merger, profile, excludeMgr, scopeEng, dataDir, runID, projectID, config, stageCallback, exec)
}

// NewWithExecutor creates a ScanEngine with an injected Executor (for testing).
func NewWithExecutor(
	queries *db.Queries,
	merger *asset.Merger,
	profile core.Profile,
	excludeMgr *exclude.Manager,
	scopeEng *scope.Engine,
	dataDir string,
	runID string,
	projectID string,
	config EngineConfig,
	stageCallback stageagg.StageEventCallback,
	exec executor.Executor,
) *ScanEngine {
	store := work.NewStore(queries)
	agg := stageagg.NewAggregator(queries, runID, stageCallback)

	now := time.Now().UTC()
	return &ScanEngine{
		queries:        queries,
		merger:         merger,
		store:          store,
		exec:           exec,
		agg:            agg,
		dedup:          dedup.New(),
		pq:             queue.New(),
		config:         config,
		profile:        profile,
		excludeMgr:     excludeMgr,
		scopeEng:       scopeEng,
		nucleiPersist:  finding.NewNucleiPersister(queries, dataDir),
		dataDir:        dataDir,
		runID:          runID,
		projectID:      projectID,
		engineState:    "running",
		startedAt:      now,
		lastNewAssetAt: now,
		sem:            make(chan struct{}, config.BatchSize),
		maxRetries:     3,
	}
}

// Run starts the engine and blocks until the scan completes or is cancelled.
func (e *ScanEngine) Run(ctx context.Context, targets []string) error {
	ctx, e.cancel = context.WithCancel(ctx)
	defer e.cancel()

	// Update engine state in DB
	if err := e.queries.UpdatePipelineRunEngineState(e.runID, "running"); err != nil {
		return err
	}

	// Seed initial targets as assets
	for _, target := range targets {
		e.processNewAsset(ctx, &core.DiscoveryAsset{
			ID:              util.GenerateID(),
			Type:            core.ClassifySeedTarget(target),
			Value:           target,
			NormalizedValue: target,
			DiscoveryDepth:  0,
			SourceTool:      "seed",
		})
	}

	// External profile: passive cert discovery for root domains (crt.sh)
	if _, ok := e.profile.(*core.ExternalProfile); ok {
		inj := seed.NewPassiveInjector(func(c context.Context, a *core.DiscoveryAsset) {
			e.processNewAsset(c, a)
		})
		for _, target := range targets {
			if core.ClassifySeedTarget(target) == core.AssetSubdomain {
				inj.InjectCrt(ctx, target, e.projectID)
			}
		}
	}

	// Main scheduler loop
	ticker := time.NewTicker(e.config.SchedulerTick)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			e.wg.Wait() // wait for in-flight work to finish
			e.setEngineState("stopped")
			return ctx.Err()
		case <-ticker.C:
			if err := e.tick(ctx); err != nil {
				log.Printf("[scanengine] tick error: %v", err)
			}
			if e.EngineState() == "stopped" {
				e.wg.Wait()
				return nil
			}
		}
	}
}

// Cancel stops the engine.
func (e *ScanEngine) Cancel() {
	if e.cancel != nil {
		e.cancel()
	}
}

// EngineState returns the current engine state.
func (e *ScanEngine) EngineState() string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.engineState
}

// processNewAsset handles a newly discovered asset: dedup, derive works, enqueue.
func (e *ScanEngine) processNewAsset(ctx context.Context, a *core.DiscoveryAsset) {
	// Dedup check
	if !e.dedup.IsNew(a.NormalizedValue) {
		return
	}

	// Depth check
	if a.DiscoveryDepth > core.MaxDiscoveryDepth {
		return
	}

	if e.isAssetExcluded(a) {
		log.Printf("[scanengine] skipping excluded asset: %s", a.Value)
		return
	}

	// Update last new asset time
	e.mu.Lock()
	e.lastNewAssetAt = time.Now().UTC()
	e.mu.Unlock()
	_ = e.queries.UpdatePipelineRunLastNewAssetAt(e.runID, e.lastNewAssetAt)

	// Merge into asset DB (best-effort)
	assetType := assetTypeToString(a.Type)
	if assetType != "" {
		createdAsset, _, err := e.merger.MergeOrCreateAsset(e.projectID, assetType, a.Value, a.SourceTool)
		if err != nil {
			log.Printf("[scanengine] merge asset %s: %v", a.Value, err)
			return
		}
		if createdAsset != nil {
			a.ID = createdAsset.ID
		}
	}
	e.assetDepth.Store(a.ID, a.DiscoveryDepth)

	// Derive eligible works
	works := core.DeriveEligibleWorks(a, e.profile)
	for _, dw := range works {
		// Check if already exists
		exists, _ := e.store.Exists(e.runID, a.ID, string(dw.Action))
		if exists {
			continue
		}
		// Override asset ID with merged ID
		dw.AssetID = a.ID
		// Create and enqueue
		w, err := e.store.Create(e.runID, e.projectID, dw.AssetID, dw.Action, dw.Stage)
		if err != nil {
			log.Printf("[scanengine] create work: %v", err)
			continue
		}
		// Notify aggregator of new work item
		e.agg.OnWorkCreated(dw.Action)
		priority := queue.ClassifyAction(string(dw.Action))
		e.pq.Push(queue.Item{
			WorkID:   w.ID,
			Action:   string(dw.Action),
			AssetID:  dw.AssetID,
			Priority: priority,
		})
	}
}

// tick is called on each scheduler cycle.
func (e *ScanEngine) tick(ctx context.Context) error {
	e.mu.Lock()
	lastAsset := e.lastNewAssetAt
	state := e.engineState
	e.mu.Unlock()

	// Check absolute timeout (based on run start time, not last asset)
	if time.Since(e.startedAt) > e.config.AbsoluteTimeout {
		log.Printf("[scanengine] absolute timeout reached (%v), stopping", e.config.AbsoluteTimeout)
		e.setEngineState("stopped")
		return nil
	}

	// Check idle timeout → wind_down
	if state == "running" && time.Since(lastAsset) > e.config.IdleTimeout {
		log.Printf("[scanengine] idle timeout reached (%v), entering wind_down", e.config.IdleTimeout)
		e.setEngineState("wind_down")
		state = "wind_down"
	}

	allTerminal, err := e.store.AllTerminal(e.runID)
	if err != nil {
		return err
	}
	queueEmpty := e.pq.IsEmpty()
	inFlight := atomic.LoadInt32(&e.inFlight)

	// Early completion: all work done, nothing queued or running
	if state == "running" && allTerminal && queueEmpty && inFlight == 0 {
		e.setEngineState("stopped")
		return nil
	}

	// wind_down → stopped when drained
	if state == "wind_down" && allTerminal && queueEmpty && inFlight == 0 {
		e.setEngineState("stopped")
		return nil
	}

	// Process pending work items from DB that aren't in queue yet
	pending, err := e.store.ListPending(e.runID)
	if err != nil {
		return err
	}
	for _, w := range pending {
		priority := queue.ClassifyAction(w.Action)
		e.pq.Push(queue.Item{
			WorkID:   w.ID,
			Action:   w.Action,
			AssetID:  w.AssetID,
			Priority: priority,
		})
	}

	// Pop and execute up to BatchSize items with concurrency control
	for i := 0; i < e.config.BatchSize; i++ {
		item, ok := e.pq.Pop()
		if !ok {
			break
		}
		e.wg.Add(1)
		e.sem <- struct{}{} // acquire concurrency slot
		go func(it queue.Item) {
			defer e.wg.Done()
			defer func() { <-e.sem }() // release slot
			e.executeWork(ctx, it)
		}(item)
	}

	return nil
}

// executeWork claims and executes a single work item with retry logic.
func (e *ScanEngine) executeWork(ctx context.Context, item queue.Item) {
	atomic.AddInt32(&e.inFlight, 1)
	defer atomic.AddInt32(&e.inFlight, -1)

	// wind_down filter: only allow finishing actions
	if e.EngineState() == "wind_down" {
		if !isWindDownAllowed(item.Action) {
			e.store.MarkSkipped(item.WorkID, "wind_down")
			e.agg.OnWorkSkipped(core.TaskAction(item.Action))
			return
		}
	}

	// TryClaim
	w, err := e.store.TryClaim(item.WorkID)
	if err != nil || w == nil {
		return // already claimed or error
	}

	// Notify aggregator
	e.agg.OnWorkStarted(core.TaskAction(w.Action))

	// Retry loop
	var lastErr error
	for attempt := 0; attempt <= e.maxRetries; attempt++ {
		if attempt > 0 {
			log.Printf("[scanengine] retrying work %s (attempt %d/%d)", w.ID, attempt, e.maxRetries)
			// Exponential backoff: 1s, 2s, 4s
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			select {
			case <-ctx.Done():
				e.store.MarkFailed(w.ID, "cancelled during retry backoff")
				e.agg.OnWorkFailed(core.TaskAction(w.Action))
				return
			case <-time.After(backoff):
			}
		}

		// Build params based on action
		params, cleanup, buildErr := e.buildParams(ctx, w)
		if buildErr != nil {
			// Build errors are not retryable (wrong config, etc.)
			e.store.MarkFailed(w.ID, buildErr.Error())
			e.agg.OnWorkFailed(core.TaskAction(w.Action))
			return
		}

		// Execute
		res, execErr := e.exec.Execute(ctx, w, params)
		if cleanup != nil {
			cleanup()
		}

		if execErr == nil {
			// Success
			if res != nil && res.Task != nil {
				_ = e.store.SetTaskID(w.ID, res.Task.ID)
				w.TaskID = &res.Task.ID
			}

			// Mark done
			if markErr := e.store.MarkDone(w.ID); markErr != nil {
				log.Printf("[scanengine] MarkDone failed for %s: %v", w.ID, markErr)
			}

			// Process output: discover new assets and update attrs
			e.onWorkComplete(ctx, w, res.Stdout)

			// Notify aggregator
			e.agg.OnWorkCompleted(core.TaskAction(w.Action))
			return
		}

		lastErr = execErr
		// Check if error is retryable (context canceled = not retryable)
		if ctx.Err() != nil {
			break
		}
	}

	// All retries exhausted
	e.store.MarkFailed(w.ID, fmt.Sprintf("failed after %d attempts: %v", e.maxRetries+1, lastErr))
	e.agg.OnWorkFailed(core.TaskAction(w.Action))
}

// onWorkComplete processes tool output to discover new assets and update attrs.
func (e *ScanEngine) onWorkComplete(ctx context.Context, w *models.ScanWorkItem, stdout []byte) {
	switch core.TaskAction(w.Action) {
	case core.ActionHTTPXFingerprint:
		newAssets, attrs, endpoints, err := executor.ParseHttpxOutput(stdout, e.projectID)
		if err != nil {
			log.Printf("[scanengine] parse httpx: %v", err)
			return
		}
		_ = attrs // attrs are tracked at the asset level; technologies are stored per-endpoint below

		// Persist web endpoints with technologies
		for _, ep := range endpoints {
			host := ep.Host
			if host == "" {
				continue
			}
			assetType := "domain"
			if net.ParseIP(host) != nil {
				assetType = "ip"
			}
			hostAsset, _, err := e.merger.MergeOrCreateAsset(e.projectID, assetType, host, "httpx")
			if err != nil {
				log.Printf("[scanengine] merge/create asset %s: %v", host, err)
				continue
			}
			ep.AssetID = hostAsset.ID
			if _, _, err := e.merger.CreateWebEndpointIfNotExists(
				e.projectID, hostAsset.ID, ep.URL, ep.Scheme, ep.Host,
				ep.Port, ep.Path, ep.Title, ep.StatusCode, ep.Technologies, "httpx",
			); err != nil {
				log.Printf("[scanengine] save web endpoint %s: %v", ep.URL, err)
			}
		}

		// Process discovered sub-assets (httpx fingerprinted HTTP services)
		for _, a := range newAssets {
			a.ParentID = w.AssetID
			a.Attrs.Fingerprinted = true
			if len(attrs.Technologies) > 0 {
				a.Attrs.Technologies = append([]string(nil), attrs.Technologies...)
			}
			e.prepareChildAsset(a, w.AssetID)
			e.processNewAsset(ctx, a)
		}

	case core.ActionKatanaCrawl:
		newAssets, err := executor.ParseKatanaOutput(stdout)
		if err != nil {
			log.Printf("[scanengine] parse katana: %v", err)
			return
		}
		for _, a := range newAssets {
			a.ParentID = w.AssetID
			e.prepareChildAsset(a, w.AssetID)
			e.processNewAsset(ctx, a)
		}

	case core.ActionFFUFBrute:
		newAssets, err := executor.ParseFFUFOutput(stdout)
		if err != nil {
			log.Printf("[scanengine] parse ffuf: %v", err)
			return
		}
		for _, a := range newAssets {
			a.ParentID = w.AssetID
			e.prepareChildAsset(a, w.AssetID)
			e.processNewAsset(ctx, a)
		}

	case core.ActionNucleiScan:
		taskID := ""
		if w.TaskID != nil {
			taskID = *w.TaskID
		}
		created, updated, err := e.nucleiPersist.Persist(e.projectID, e.runID, taskID, w.AssetID, stdout)
		if err != nil {
			log.Printf("[scanengine] persist nuclei: %v", err)
		} else if created+updated > 0 {
			log.Printf("[scanengine] nuclei findings created=%d updated=%d", created, updated)
		}

	case core.ActionSpoorScan:
		endpoints, secrets, err := executor.ParseSpoorOutput(stdout, e.runID, e.projectID)
		if err != nil {
			log.Printf("[scanengine] parse spoor: %v", err)
			return
		}
		// 回注 endpoint 为新资产
		for _, a := range endpoints {
			a.ParentID = w.AssetID
			e.prepareChildAsset(a, w.AssetID)
			e.processNewAsset(ctx, a)
		}
		// 创建 secret findings
		for _, f := range secrets {
			f.AssetID = &w.AssetID
			if err := e.queries.CreateFinding(f); err != nil {
				log.Printf("[scanengine] create spoor finding %s: %v", f.Title, err)
			}
		}

	case core.ActionPortScan:
		results, parseErrs := parser.ParseNaabu(strings.NewReader(string(stdout)))
		for _, pe := range parseErrs {
			log.Printf("[scanengine] parse naabu line %d: %s", pe.Line, pe.Message)
		}
		for _, r := range results {
			host := r.Host
			if host == "" {
				host = r.IP
			}
			if host == "" || r.Port == 0 {
				continue
			}
			if _, _, err := e.merger.CreatePortIfNotExists(w.AssetID, r.Port, "tcp", "naabu"); err != nil {
				log.Printf("[scanengine] create port %s:%d: %v", host, r.Port, err)
			}
			a := &core.DiscoveryAsset{
				ID:              util.GenerateID(),
				Type:            core.AssetIPPort,
				Value:           fmt.Sprintf("%s:%d", host, r.Port),
				NormalizedValue: fmt.Sprintf("%s:%d", host, r.Port),
				ParentID:        w.AssetID,
				SourceTool:      "naabu",
			}
			e.prepareChildAsset(a, w.AssetID)
			e.processNewAsset(ctx, a)
		}

	case core.ActionSubdomainEnum:
		results, parseErrs := parser.ParseSubfinder(strings.NewReader(string(stdout)))
		for _, pe := range parseErrs {
			log.Printf("[scanengine] parse subfinder line %d: %s", pe.Line, pe.Message)
		}
		for _, r := range results {
			if r.Host == "" {
				continue
			}
			a := &core.DiscoveryAsset{
				ID:              util.GenerateID(),
				Type:            core.AssetSubdomain,
				Value:           r.Host,
				NormalizedValue: r.Host,
				ParentID:        w.AssetID,
				SourceTool:      "subfinder",
			}
			e.prepareChildAsset(a, w.AssetID)
			e.processNewAsset(ctx, a)
		}

	case core.ActionDNSResolve:
		results, parseErrs := parser.ParseDNSx(strings.NewReader(string(stdout)))
		for _, pe := range parseErrs {
			log.Printf("[scanengine] parse dnsx line %d: %s", pe.Line, pe.Message)
		}
		alive := true
		for _, rec := range results {
			for _, ip := range parser.ExtractDNSxIPs(rec) {
				a := &core.DiscoveryAsset{
					ID:              util.GenerateID(),
					Type:            core.AssetIP,
					Value:           ip,
					NormalizedValue: ip,
					ParentID:        w.AssetID,
					SourceTool:      "dnsx",
					Attrs:           core.AssetAttrs{Alive: &alive},
				}
				e.prepareChildAsset(a, w.AssetID)
				e.processNewAsset(ctx, a)
			}
			for _, cname := range parser.ExtractDNSxCNAMEs(rec) {
				a := &core.DiscoveryAsset{
					ID:              util.GenerateID(),
					Type:            core.AssetSubdomain,
					Value:           cname,
					NormalizedValue: cname,
					ParentID:        w.AssetID,
					SourceTool:      "dnsx",
				}
				e.prepareChildAsset(a, w.AssetID)
				e.processNewAsset(ctx, a)
			}
		}

	case core.ActionCDNCheck:
		host, err := e.assetHostValue(w.AssetID)
		if err != nil {
			log.Printf("[scanengine] cdn check asset: %v", err)
			return
		}
		ips := []string{host}
		if net.ParseIP(host) == nil {
			// cdncheck expects IPs; domain-level check is best-effort via resolved child IPs
			return
		}
		_, cdnResults, err := cdn.ParseJSONLOutput(stdout, ips)
		if err != nil {
			log.Printf("[scanengine] parse cdncheck: %v", err)
			return
		}
		isCDN := true
		for _, r := range cdnResults {
			a := &core.DiscoveryAsset{
				ID:              util.GenerateID(),
				Type:            core.AssetIP,
				Value:           r.IP,
				NormalizedValue: r.IP,
				ParentID:        w.AssetID,
				SourceTool:      "cdncheck",
				Attrs:           core.AssetAttrs{IsCDN: &isCDN},
			}
			e.prepareChildAsset(a, w.AssetID)
			e.processNewAsset(ctx, a)
		}
	}
}

// buildParams constructs tool parameters for a work item.
func (e *ScanEngine) buildParams(ctx context.Context, w *models.ScanWorkItem) (toolregistry.RenderParams, func(), error) {
	cfg := e.config.Pipeline
	host, err := e.assetHostValue(w.AssetID)
	if err != nil {
		return nil, nil, err
	}

	switch core.TaskAction(w.Action) {
	case core.ActionHTTPXFingerprint:
		hostFile, cleanup, err := executor.WriteHostFile(e.dataDir, []string{host})
		if err != nil {
			return nil, nil, err
		}
		return toolregistry.RenderParams{
			"host_file": hostFile,
			"rate":      cfg.HttpxRateLimit,
			"threads":   cfg.HttpxThreads,
		}, cleanup, nil

	case core.ActionNucleiScan:
		hostFile, cleanup, err := executor.WriteHostFile(e.dataDir, []string{host})
		if err != nil {
			return nil, nil, err
		}
		profile := "standard"
		if cfg.NucleiScanDepth == "workflow" {
			profile = "deep"
		}
		params := toolregistry.RenderParams{
			"host_file":  hostFile,
			"profile":    profile,
			"scan_depth": cfg.NucleiScanDepth,
			"rate_limit": cfg.NucleiRateLimit,
			"concurrency": cfg.NucleiConcurrency,
		}
		if cfg.NucleiRateLimitPerMinute > 0 {
			params["rate_limit_per_min"] = cfg.NucleiRateLimitPerMinute
		}
		return params, cleanup, nil

	case core.ActionPortScan:
		hostFile, cleanup, err := executor.WriteHostFile(e.dataDir, []string{host})
		if err != nil {
			return nil, nil, err
		}
		return toolregistry.RenderParams{
			"host_file":  hostFile,
			"port_range": cfg.PortRange,
			"rate":       cfg.NaabuRate,
			"threads":    cfg.NaabuThreads,
			"timeout":    cfg.NaabuTimeout,
		}, cleanup, nil

	case core.ActionServiceFingerprint:
		hostFile, cleanup, err := executor.WriteHostFile(e.dataDir, []string{host})
		if err != nil {
			return nil, nil, err
		}
		port := host
		if h, p, ok := strings.Cut(host, ":"); ok {
			hostFile, cleanup, err = executor.WriteHostFile(e.dataDir, []string{h})
			if err != nil {
				return nil, nil, err
			}
			port = p
		}
		return toolregistry.RenderParams{
			"host_file": hostFile,
			"ports":     []string{port},
			"timeout":   cfg.NmapServiceTimeout,
		}, cleanup, nil

	case core.ActionSubdomainEnum:
		return toolregistry.RenderParams{
			"domain": host,
		}, nil, nil

	case core.ActionDNSResolve:
		hostFile, cleanup, err := executor.WriteHostFile(e.dataDir, []string{host})
		if err != nil {
			return nil, nil, err
		}
		return toolregistry.RenderParams{
			"host_file": hostFile,
		}, cleanup, nil

	case core.ActionCDNCheck:
		return toolregistry.RenderParams{
			"ips": host,
		}, nil, nil

	case core.ActionKatanaCrawl:
		return toolregistry.RenderParams{
			"url":        host,
			"max_depth":  cfg.KatanaMaxDepth,
			"rate_limit": cfg.KatanaRateLimit,
			"timeout":    cfg.KatanaTimeout,
		}, nil, nil

	case core.ActionFFUFBrute:
		return toolregistry.RenderParams{
			"url":     host,
			"rate":    cfg.FfufRateLimit,
			"timeout": cfg.FfufTimeout,
		}, nil, nil

	case core.ActionSpoorScan:
		return toolregistry.RenderParams{
			"target": host,
		}, nil, nil

	default:
		return toolregistry.RenderParams{}, nil, nil
	}
}

func (e *ScanEngine) assetHostValue(assetID string) (string, error) {
	a, err := e.queries.GetAssetByID(assetID)
	if err != nil {
		return "", fmt.Errorf("get asset %s: %w", assetID, err)
	}
	if a == nil {
		return "", fmt.Errorf("asset not found: %s", assetID)
	}
	return a.Value, nil
}

// isWindDownAllowed returns true if the action should continue during wind_down.
func isWindDownAllowed(action string) bool {
	switch action {
	case string(core.ActionNucleiScan), string(core.ActionHTTPXFingerprint):
		return true
	default:
		return false
	}
}

// setEngineState updates the engine state in memory and DB.
func (e *ScanEngine) setEngineState(state string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.engineState == state {
		return
	}
	e.engineState = state
	if err := e.queries.UpdatePipelineRunEngineState(e.runID, state); err != nil {
		log.Printf("[scanengine] update engine state: %v", err)
	}
}

func (e *ScanEngine) isAssetExcluded(a *core.DiscoveryAsset) bool {
	if e.excludeMgr != nil && e.excludeMgr.IsExcluded(a.Value) {
		return true
	}
	if e.scopeEng == nil {
		return false
	}
	target := assetToScopeTarget(a)
	if target == nil {
		return false
	}
	excluded, err := e.scopeEng.IsExcludedForProject(e.projectID, target)
	if err != nil {
		log.Printf("[scanengine] scope check %s: %v", a.Value, err)
		return false
	}
	return excluded
}

func (e *ScanEngine) prepareChildAsset(a *core.DiscoveryAsset, parentAssetID string) {
	if parentAssetID == "" {
		return
	}
	if v, ok := e.assetDepth.Load(parentAssetID); ok {
		if depth, ok := v.(int); ok {
			a.DiscoveryDepth = depth + 1
			return
		}
	}
	a.DiscoveryDepth++
}

func assetToScopeTarget(a *core.DiscoveryAsset) *models.Target {
	if a == nil {
		return nil
	}
	value := strings.TrimSpace(a.Value)
	switch a.Type {
	case core.AssetSubdomain:
		return &models.Target{Type: models.TargetTypeDomain, Value: value}
	case core.AssetIP:
		return &models.Target{Type: models.TargetTypeIP, Value: value}
	case core.AssetIPPort:
		host, _, err := net.SplitHostPort(value)
		if err != nil {
			host = value
		}
		if net.ParseIP(host) != nil {
			return &models.Target{Type: models.TargetTypeIP, Value: host}
		}
		return &models.Target{Type: models.TargetTypeDomain, Value: host}
	case core.AssetHTTPService, core.AssetHTTPPath:
		return &models.Target{Type: models.TargetTypeURL, Value: value}
	default:
		return &models.Target{Type: models.TargetTypeDomain, Value: value}
	}
}

// assetTypeToString converts core.AssetType to the string used by asset.Merger.
func assetTypeToString(t core.AssetType) string {
	switch t {
	case core.AssetSubdomain:
		return "domain"
	case core.AssetIP:
		return "ip"
	case core.AssetIPPort:
		return "ip"
	case core.AssetHTTPService:
		return "url"
	case core.AssetHTTPPath:
		return "url"
	default:
		return ""
	}
}
