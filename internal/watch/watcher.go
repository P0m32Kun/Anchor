package watch

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/P0m32Kun/Anchor/internal/asset"
	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/exclude"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/scanengine"
	"github.com/P0m32Kun/Anchor/internal/scanengine/core"
	"github.com/P0m32Kun/Anchor/internal/scanengine/seed"
	"github.com/P0m32Kun/Anchor/internal/scope"
	"github.com/P0m32Kun/Anchor/internal/signal"
	"github.com/P0m32Kun/Anchor/internal/util"
	"github.com/P0m32Kun/Anchor/internal/worker"

	toolreg "github.com/P0m32Kun/Anchor/internal/toolregistry"
)

// Watcher periodically triggers scans for watch-enabled projects.
type Watcher struct {
	queries    *db.Queries
	worker     *worker.Runner
	merger     *asset.Merger
	excludeMgr *exclude.Manager
	scopeEng   *scope.Engine
	dataDir    string
	signalSvc  *signal.Service
	// onScanStarted is called when a watch-triggered scan starts (for SSE broadcast).
	onScanStarted func(projectID, runID, mode string)
	// onScanCompleted is called when a watch-triggered scan completes.
	onScanCompleted func(projectID, runID string)
	// running keeps track of projects with active watch-triggered scans.
	running map[string]bool
}

// NewWatcher creates a new Watcher.
func NewWatcher(
	queries *db.Queries,
	worker *worker.Runner,
	merger *asset.Merger,
	excludeMgr *exclude.Manager,
	scopeEng *scope.Engine,
	dataDir string,
) *Watcher {
	return &Watcher{
		queries:    queries,
		worker:     worker,
		merger:     merger,
		excludeMgr: excludeMgr,
		scopeEng:   scopeEng,
		dataDir:    dataDir,
		signalSvc:  signal.NewService(queries),
		running:    make(map[string]bool),
	}
}

// SetOnScanStarted sets a callback for when a watch scan starts.
func (w *Watcher) SetOnScanStarted(fn func(projectID, runID, mode string)) {
	w.onScanStarted = fn
}

// SetOnScanCompleted sets a callback for when a watch scan completes.
func (w *Watcher) SetOnScanCompleted(fn func(projectID, runID string)) {
	w.onScanCompleted = fn
}

// Start begins the watch loop. Runs until ctx is cancelled.
func (w *Watcher) Start(ctx context.Context) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	log.Printf("[watch] watcher started (tick interval: 60s)")
	for {
		select {
		case <-ctx.Done():
			log.Printf("[watch] watcher stopped")
			return
		case <-ticker.C:
			w.tick(ctx)
		}
	}
}

func (w *Watcher) tick(ctx context.Context) {
	projects, err := w.queries.ListWatchEnabledProjects()
	if err != nil {
		log.Printf("[watch] list watch-enabled projects: %v", err)
		return
	}

	now := time.Now().UTC()
	for _, p := range projects {
		if w.running[p.ID] {
			continue
		}
		if !w.isDue(p, now) {
			continue
		}
		w.startScan(ctx, p)
	}
}

func (w *Watcher) isDue(p *models.WatchProject, now time.Time) bool {
	if p.WatchLastTickAt == nil {
		return true
	}
	interval := time.Duration(p.WatchIntervalHours) * time.Hour
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	return now.Sub(*p.WatchLastTickAt) >= interval
}

func (w *Watcher) startScan(ctx context.Context, p *models.WatchProject) {
	w.running[p.ID] = true
	mode := "external"
	cfg := models.DefaultPipelineConfig()
	if p.WatchPassiveOnly {
		// In passive-only mode, use internal mode (passive search is injected during seed expansion).
		// The scan engine determines mode behavior via profile.
	}
	_ = mode // For future per-project mode override

	scanCtx, cancel := context.WithCancel(ctx)
	runID := util.GenerateID()
	now := time.Now().UTC()

	if err := w.queries.CreatePipelineRun(&models.PipelineRun{
		ID:        runID,
		ProjectID: p.ID,
		Mode:      mode,
		Status:    "running",
		StartedAt: now,
		CreatedAt: now,
	}); err != nil {
		log.Printf("[watch] create pipeline run for project %s: %v", p.ID, err)
		w.running[p.ID] = false
		cancel()
		return
	}

	log.Printf("[watch] starting scan for project %s (run %s, passive_only=%v)", p.Name, runID, p.WatchPassiveOnly)
	if w.onScanStarted != nil {
		w.onScanStarted(p.ID, runID, mode)
	}

	go func() {
		defer func() {
			cancel()
			w.running[p.ID] = false
		}()

		profile := core.ProfileFromConfig(mode, cfg)
		engineCfg := scanengine.DefaultEngineConfig()
		engineCfg.Pipeline = cfg
		engine := scanengine.New(
			w.queries, w.worker, toolreg.DefaultRegistry(),
			w.merger, profile, w.excludeMgr, w.scopeEng, w.dataDir, runID, p.ID, engineCfg,
			func(rid, stage, status, errMsg string) {
				log.Printf("[watch] run %s stage %s: %s", rid, stage, status)
			},
		)

		targets, _ := w.queries.ListTargetsByProject(p.ID)
		seeds := seed.ExpandTargets(scanCtx, w.queries, cfg, targets)
		if len(seeds) == 0 {
			for _, t := range targets {
				if t == nil || t.Value == "" {
					continue
				}
				seeds = append(seeds, seed.SeedAsset{
					Value:     t.Value,
					ValueType: string(t.Type),
					Source:    "target",
					SourceRef: t.ID,
				})
			}
		}

		runErr := engine.RunWithSeeds(scanCtx, seeds)
		if runErr != nil {
			log.Printf("[watch] scan engine run %s for project %s: %v", runID, p.ID, runErr)
		}

		w.finalizeRun(runID, runErr, p)
	}()

	w.queries.UpdateProjectWatchTick(p.ID, now)
}

func (w *Watcher) finalizeRun(runID string, runErr error, p *models.WatchProject) {
	if runErr != nil {
		msg := runErr.Error()
		_ = w.queries.UpdatePipelineRunError(runID, msg)
		_ = w.queries.UpdatePipelineRunStatus(runID, "failed")
		log.Printf("[watch] scan failed for project %s (run %s): %v", p.Name, runID, runErr)
	} else {
		_ = w.queries.UpdatePipelineRunCompleted(runID, time.Now().UTC())
		log.Printf("[watch] scan completed for project %s (run %s)", p.Name, runID)
	}

	// Create asset snapshot for change tracking.
	w.createSnapshot(runID, p.ID)

	// Generate signals for new/changed assets.
	if err := w.signalSvc.GenerateAssetSignals(p.ID, runID); err != nil {
		log.Printf("[watch] generate asset signals for project %s run %s: %v", p.ID, runID, err)
	}

	if w.onScanCompleted != nil {
		w.onScanCompleted(p.ID, runID)
	}
}

func (w *Watcher) createSnapshot(runID, projectID string) {
	assetCount, _ := w.queries.CountAssetsByProject(projectID)
	portCount, _ := w.queries.CountPortsByProject(projectID)
	endpointCount, _ := w.queries.CountWebEndpointsByProject(projectID)
	serviceCount, _ := w.queries.CountServicesByProject(projectID)

	// Collect current asset values for comparison.
	assets, err := w.queries.ListAssetsByProject(projectID)
	currentValues := make([]string, 0)
	if err == nil {
		for _, a := range assets {
			currentValues = append(currentValues, a.NormalizedValue)
		}
	}
	changesJSON, _ := json.Marshal(map[string]interface{}{
		"asset_values": currentValues,
	})

	snap := &db.AssetSnapshot{
		ProjectID:        projectID,
		RunID:            runID,
		AssetCount:       assetCount,
		PortCount:        portCount,
		EndpointCount:    endpointCount,
		ServiceCount:     serviceCount,
		AssetChangesJSON: string(changesJSON),
		CreatedAt:        time.Now().UTC(),
	}
	if err := w.queries.CreateAssetSnapshot(snap); err != nil {
		log.Printf("[watch] create asset snapshot for project %s run %s: %v", projectID, runID, err)
	}
}
