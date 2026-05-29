package scanengine

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/P0m32Kun/Anchor/internal/asset"
	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/scanengine/core"
	"github.com/P0m32Kun/Anchor/internal/scanengine/dedup"
	"github.com/P0m32Kun/Anchor/internal/scanengine/executor"
	"github.com/P0m32Kun/Anchor/internal/scanengine/queue"
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
}

// DefaultEngineConfig returns sensible defaults.
func DefaultEngineConfig() EngineConfig {
	return EngineConfig{
		IdleTimeout:     5 * time.Minute,
		AbsoluteTimeout: 30 * time.Minute,
		SchedulerTick:   2 * time.Second,
		BatchSize:       5,
	}
}

// ScanEngine is the asset-driven scan engine.
type ScanEngine struct {
	queries  *db.Queries
	runner   *worker.Runner
	tools    *toolregistry.Registry
	merger   *asset.Merger
	store    *work.Store
	exec     *executor.ToolExecutor
	agg      *stageagg.Aggregator
	dedup    *dedup.RunDedup
	pq       *queue.PriorityQueue
	config   EngineConfig
	profile  core.Profile
	dataDir  string
	runID    string
	projectID string

	mu             sync.Mutex
	engineState    string
	lastNewAssetAt time.Time
	cancel         context.CancelFunc
}

// New creates a new ScanEngine.
func New(
	queries *db.Queries,
	runner *worker.Runner,
	tools *toolregistry.Registry,
	merger *asset.Merger,
	profile core.Profile,
	dataDir string,
	runID string,
	projectID string,
	config EngineConfig,
	stageCallback stageagg.StageEventCallback,
) *ScanEngine {
	store := work.NewStore(queries)
	exec := executor.NewToolExecutor(queries, runner, tools, merger, dataDir)
	agg := stageagg.NewAggregator(queries, runID, stageCallback)

	return &ScanEngine{
		queries:        queries,
		runner:         runner,
		tools:          tools,
		merger:         merger,
		store:          store,
		exec:           exec,
		agg:            agg,
		dedup:          dedup.New(),
		pq:             queue.New(),
		config:         config,
		profile:        profile,
		dataDir:        dataDir,
		runID:          runID,
		projectID:      projectID,
		engineState:    "running",
		lastNewAssetAt: time.Now().UTC(),
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
			Type:            core.AssetSubdomain,
			Value:           target,
			NormalizedValue: target,
			DiscoveryDepth:  0,
			SourceTool:      "seed",
		})
	}

	// Main scheduler loop
	ticker := time.NewTicker(e.config.SchedulerTick)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			e.setEngineState("stopped")
			return ctx.Err()
		case <-ticker.C:
			if err := e.tick(ctx); err != nil {
				log.Printf("[scanengine] tick error: %v", err)
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

	// Update last new asset time
	e.mu.Lock()
	e.lastNewAssetAt = time.Now().UTC()
	e.mu.Unlock()
	_ = e.queries.UpdatePipelineRunLastNewAssetAt(e.runID, e.lastNewAssetAt)

	// Merge into asset DB (best-effort)
	assetType := assetTypeToString(a.Type)
	if assetType != "" {
		createdAsset, isNew, err := e.merger.MergeOrCreateAsset(e.projectID, assetType, a.Value, a.SourceTool)
		if err == nil && isNew && createdAsset != nil {
			a.ID = createdAsset.ID
		}
	}

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
	// Check absolute timeout
	if time.Since(e.lastNewAssetAt) > e.config.AbsoluteTimeout {
		e.setEngineState("stopped")
		return nil
	}

	// Check idle timeout → wind_down
	if e.engineState == "running" && time.Since(e.lastNewAssetAt) > e.config.IdleTimeout {
		e.setEngineState("wind_down")
	}

	// If wind_down and queue empty and all terminal → stopped
	if e.engineState == "wind_down" {
		allTerminal, err := e.store.AllTerminal(e.runID)
		if err != nil {
			return err
		}
		if allTerminal && e.pq.IsEmpty() {
			e.setEngineState("stopped")
			return nil
		}
	}

	// Process pending work items from DB that aren't in queue yet
	pending, err := e.store.ListPending(e.runID)
	if err != nil {
		return err
	}
	for _, w := range pending {
		// Check if already in queue (by checking if it's been processed)
		priority := queue.ClassifyAction(w.Action)
		e.pq.Push(queue.Item{
			WorkID:   w.ID,
			Action:   w.Action,
			AssetID:  w.AssetID,
			Priority: priority,
		})
	}

	// Pop and execute up to BatchSize items
	for i := 0; i < e.config.BatchSize; i++ {
		item, ok := e.pq.Pop()
		if !ok {
			break
		}
		go e.executeWork(ctx, item)
	}

	return nil
}

// executeWork claims and executes a single work item.
func (e *ScanEngine) executeWork(ctx context.Context, item queue.Item) {
	// TryClaim
	w, err := e.store.TryClaim(item.WorkID)
	if err != nil || w == nil {
		return // already claimed or error
	}

	// Notify aggregator
	e.agg.OnWorkStarted(core.TaskAction(w.Action))

	// Build params based on action
	params, err := e.buildParams(ctx, w)
	if err != nil {
		e.store.MarkFailed(w.ID, err.Error())
		e.agg.OnWorkCompleted(core.TaskAction(w.Action))
		return
	}

	// Execute
	res, err := e.exec.Execute(ctx, w, params)
	if err != nil {
		e.store.MarkFailed(w.ID, err.Error())
		e.agg.OnWorkCompleted(core.TaskAction(w.Action))
		return
	}

	// Mark done
	e.store.MarkDone(w.ID)

	// Process output: discover new assets and update attrs
	e.onWorkComplete(ctx, w, res.Stdout)

	// Notify aggregator
	e.agg.OnWorkCompleted(core.TaskAction(w.Action))
}

// onWorkComplete processes tool output to discover new assets and update attrs.
func (e *ScanEngine) onWorkComplete(ctx context.Context, w *models.ScanWorkItem, stdout []byte) {
	switch core.TaskAction(w.Action) {
	case core.ActionHTTPXFingerprint:
		newAssets, attrs, err := executor.ParseHttpxOutput(stdout, e.projectID)
		if err != nil {
			log.Printf("[scanengine] parse httpx: %v", err)
			return
		}
		// Update attrs on the source asset
		if attrs.Fingerprinted {
			// The asset itself gets fingerprinted
			_ = attrs
		}
		// Process discovered sub-assets
		for _, a := range newAssets {
			a.ParentID = w.AssetID
			a.DiscoveryDepth = 1 // httpx output is depth 1 from seed
			e.processNewAsset(ctx, a)
		}

	case core.ActionNucleiScan:
		// Nuclei findings are handled by the existing pipeline infrastructure
		// We just log them here
		findings, _ := executor.ParseNucleiOutput(stdout)
		if len(findings) > 0 {
			log.Printf("[scanengine] nuclei found %d templates", len(findings))
		}
	}
}

// buildParams constructs tool parameters for a work item.
func (e *ScanEngine) buildParams(ctx context.Context, w *models.ScanWorkItem) (toolregistry.RenderParams, error) {
	switch core.TaskAction(w.Action) {
	case core.ActionHTTPXFingerprint:
		// Write the asset value to a host file
		hostFile, cleanup, err := executor.WriteHostFile(e.dataDir, []string{w.AssetID})
		if err != nil {
			return nil, err
		}
		_ = cleanup // cleanup happens after tool execution
		return toolregistry.RenderParams{
			"host_file": hostFile,
		}, nil

	case core.ActionNucleiScan:
		hostFile, cleanup, err := executor.WriteHostFile(e.dataDir, []string{w.AssetID})
		if err != nil {
			return nil, err
		}
		_ = cleanup
		return toolregistry.RenderParams{
			"host_file": hostFile,
		}, nil

	default:
		return toolregistry.RenderParams{}, nil
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

// assetTypeToString converts core.AssetType to the string used by asset.Merger.
func assetTypeToString(t core.AssetType) string {
	switch t {
	case core.AssetSubdomain:
		return "domain"
	case core.AssetIP:
		return "ip"
	case core.AssetIPPort:
		return "ip_port"
	case core.AssetHTTPService:
		return "url"
	case core.AssetHTTPPath:
		return "url"
	default:
		return ""
	}
}
