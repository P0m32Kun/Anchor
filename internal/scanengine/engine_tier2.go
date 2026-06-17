package scanengine

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/P0m32Kun/Anchor/internal/fingerprint"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/parser"
	"github.com/P0m32Kun/Anchor/internal/scanconfig"
	"github.com/P0m32Kun/Anchor/internal/scanengine/core"
	"github.com/P0m32Kun/Anchor/internal/scanengine/executor"
	"github.com/P0m32Kun/Anchor/internal/scanengine/pool"
	"github.com/P0m32Kun/Anchor/internal/scanengine/queue"
	"github.com/P0m32Kun/Anchor/internal/scanengine/work"
	"github.com/P0m32Kun/Anchor/internal/toolregistry"
	"github.com/P0m32Kun/Anchor/internal/util"
)

func (e *ScanEngine) initTier2Pools(ctx context.Context) {
	e.tier2Mu.Lock()
	if e.tier2Scheduled == nil {
		e.tier2Scheduled = make(map[string]struct{})
	}
	e.tier2Mu.Unlock()

	if e.nucleiRouter == nil {
		if sc := scanconfig.Get(); sc != nil {
			e.nucleiRouter = sc.NucleiRouter()
		} else {
			e.nucleiRouter = scanconfig.DefaultNucleiRouter()
		}
	}

	flushTimeout := e.config.Tier2FlushTimeout
	if flushTimeout <= 0 {
		flushTimeout = e.config.Tier1FlushTimeout
	}
	if flushTimeout <= 0 {
		flushTimeout = 10 * time.Second
	}

	httpCfg := pool.DefaultHostPoolConfig(e.dataDir)
	httpCfg.BatchSize = 100
	httpCfg.FilePrefix = "httpx_batch"
	httpCfg.Label = "httpxpool"
	httpCfg.FlushTimeout = flushTimeout
	e.httpPool = pool.New(httpCfg, func(ev pool.FlushEvent) {
		e.onTier2PoolFlush(ctx, core.ActionHTTPXFingerprint, "tier2:httpx", ev)
	})
	e.httpPool.Start()

	e.ipPortAgg = pool.NewIPPortAggregator(e.dataDir, func(ip string, ev pool.FlushEvent) {
		_ = ip
		e.onTier2PoolFlush(ctx, core.ActionServiceFingerprint, "tier2:nmap:"+ip, ev)
	})

	e.nucleiBuckets = pool.NewNucleiTagBuckets(e.dataDir, 30, flushTimeout, func(bucketKey string, ev pool.FlushEvent) {
		e.onTier2PoolFlush(ctx, core.ActionNucleiScan, bucketKey, ev)
	})
}

func (e *ScanEngine) stopTier2Pools() {
	if e.httpPool != nil {
		e.httpPool.Stop()
	}
	if e.ipPortAgg != nil && e.ipPortAgg.Len() > 0 {
		e.ipPortAgg.FlushAll()
	}
	if e.nucleiBuckets != nil {
		e.nucleiBuckets.Stop()
	}
}

func (e *ScanEngine) stopTier2PoolsOnce() {
	e.tier2Stopped.Do(func() {
		e.stopTier2Pools()
	})
}

func (e *ScanEngine) flushTier2IfBlockingHigherStages() {
	if e.pqHasStageOrHigher(queue.StageWeb) {
		if e.ipPortAgg != nil && e.ipPortAgg.Len() > 0 {
			e.ipPortAgg.FlushAll()
		}
	}
	if e.pqHasStageOrHigher(queue.StageVuln) {
		if e.httpPool != nil && e.httpPool.Len() > 0 {
			e.httpPool.FlushNow()
		}
		if e.nucleiBuckets != nil && e.nucleiBuckets.Len() > 0 {
			e.nucleiBuckets.FlushAll()
		}
	}
}

func (e *ScanEngine) pqHasStageOrHigher(min queue.StageRank) bool {
	if e.pq.IsEmpty() {
		return false
	}
	depth := e.pq.StageDepth()
	for stage, count := range depth {
		if count > 0 && stage >= min {
			return true
		}
	}
	return false
}

func (e *ScanEngine) isTier2PooledAction(action core.TaskAction) bool {
	switch action {
	case core.ActionHTTPXFingerprint, core.ActionServiceFingerprint, core.ActionNucleiScan:
		return true
	default:
		return false
	}
}

func (e *ScanEngine) claimTier2(action core.TaskAction, assetID string) bool {
	key := string(action) + ":" + assetID
	e.tier2Mu.Lock()
	defer e.tier2Mu.Unlock()
	if _, ok := e.tier2Scheduled[key]; ok {
		return false
	}
	e.tier2Scheduled[key] = struct{}{}
	return true
}

func (e *ScanEngine) enqueueTier2Asset(ctx context.Context, a *core.DiscoveryAsset, dw core.DerivedWork, bucketKey string) {
	if !e.claimTier2(dw.Action, dw.AssetID) {
		return
	}

	switch dw.Action {
	case core.ActionHTTPXFingerprint:
		if e.httpPool == nil {
			return
		}
		target := pool.ProbeTargetFromAsset(a.Value, a.Type)
		if target == "" {
			return
		}
		e.httpPool.Add(pool.Member{Value: target, AssetID: dw.AssetID, BucketKey: bucketKey})
	case core.ActionServiceFingerprint:
		if e.ipPortAgg == nil {
			return
		}
		host, port := pool.ParseHostPort(a.Value)
		if host == "" || port <= 0 {
			return
		}
		e.ipPortAgg.Add(host, port, dw.AssetID, bucketKey)
	case core.ActionNucleiScan:
		e.enqueueNucleiAsset(a, dw.AssetID, bucketKey)
	default:
		return
	}
	_ = ctx
}

func (e *ScanEngine) enqueueNucleiAsset(a *core.DiscoveryAsset, assetID, lineageBucket string) {
	if e.nucleiBuckets == nil || e.nucleiRouter == nil {
		return
	}
	url := strings.TrimSpace(a.Value)
	if url == "" {
		return
	}
	noiseLevel := e.config.Pipeline.NoiseLevel
	bucket, _, skip := e.nucleiRouter.Resolve(a.Attrs.Technologies, noiseLevel)
	if skip {
		return
	}
	e.nucleiBuckets.Add(bucket, url, assetID, lineageBucket)
}

func (e *ScanEngine) onTier2PoolFlush(ctx context.Context, action core.TaskAction, bucketKey string, ev pool.FlushEvent) {
	if len(ev.Members) == 0 {
		return
	}
	w, err := e.store.CreatePooledBatch(work.PooledBatchInput{
		RunID:      e.runID,
		ProjectID:  e.projectID,
		Action:     action,
		Stage:      core.ActionToStage[action],
		InputFile:  ev.FilePath,
		Members:    ev.Members,
		BucketKey:  bucketKey,
		Generation: ev.Generation,
	})
	if err != nil {
		log.Printf("[scanengine] create tier2 batch work %s gen %d: %v", action, ev.Generation, err)
		return
	}
	e.agg.OnWorkCreated(action)
	e.pq.Push(queue.Item{
		WorkID:    w.ID,
		Action:    string(action),
		AssetID:   w.AssetID,
		Priority:  queue.ClassifyAction(string(action)),
		BucketKey: bucketKey,
	})
	_ = ctx
}

func (e *ScanEngine) buildTier2BatchParams(w *models.ScanWorkItem) (toolregistry.RenderParams, func(), error) {
	cfg := e.config.Pipeline
	switch core.TaskAction(w.Action) {
	case core.ActionHTTPXFingerprint:
		return toolregistry.RenderParams{
			"host_file":  w.InputFile,
			"rate":       cfg.HttpxRateLimit,
			"threads":    cfg.HttpxThreads,
		}, nil, nil
	case core.ActionServiceFingerprint:
		members := parseBatchMembers(w)
		poolMembers := make([]pool.Member, len(members))
		for i, m := range members {
			poolMembers[i] = pool.Member{Value: m.Value, AssetID: m.AssetID, BucketKey: m.BucketKey}
		}
		ports := pool.SortedPortsFromMembers(poolMembers)
		portStrs := make([]string, len(ports))
		for i, p := range ports {
			portStrs[i] = strconv.Itoa(p)
		}
		return toolregistry.RenderParams{
			"host_file":    w.InputFile,
			"ports":        portStrs,
			"host_timeout": cfg.NmapServiceTimeout,
		}, nil, nil
	case core.ActionNucleiScan:
		tags := e.nucleiRouter.TagsForBucket(w.BucketKey)
		params := toolregistry.RenderParams{
			"target_file": w.InputFile,
			"scan_depth":  "tags",
			"tags":        tags,
			"rate_limit":  cfg.NucleiRateLimit,
			"concurrency": cfg.NucleiConcurrency,
		}
		if cfg.NucleiRateLimitPerMinute > 0 {
			params["rate_limit_per_min"] = cfg.NucleiRateLimitPerMinute
		}
		return params, nil, nil
	default:
		return nil, nil, fmt.Errorf("unsupported tier2 batch action: %s", w.Action)
	}
}

func batchMemberByHostPort(members []models.WorkBatchMember) map[string]models.WorkBatchMember {
	out := make(map[string]models.WorkBatchMember, len(members))
	for _, m := range members {
		host, port := pool.ParseHostPort(m.Value)
		if host == "" || port <= 0 {
			continue
		}
		key := fmt.Sprintf("%s:%d", strings.ToLower(host), port)
		out[key] = m
	}
	return out
}

func (e *ScanEngine) onBatchHTTPXComplete(ctx context.Context, w *models.ScanWorkItem, stdout []byte) {
	members := parseBatchMembers(w)
	byInput := batchMemberByValue(members)
	e.processHTTPXOutput(ctx, func(input string) string {
		input = strings.ToLower(strings.TrimSpace(input))
		if m, ok := byInput[input]; ok {
			return m.AssetID
		}
		for _, m := range members {
			if strings.Contains(input, strings.ToLower(m.Value)) {
				return m.AssetID
			}
		}
		if len(members) > 0 {
			return members[0].AssetID
		}
		return w.AssetID
	}, stdout)
}

func (e *ScanEngine) processHTTPXOutput(ctx context.Context, parentForInput func(string) string, stdout []byte) {
	newAssets, _, endpoints, err := executor.ParseHttpxOutput(stdout, e.projectID)
	if err != nil {
		log.Printf("[scanengine] parse httpx: %v", err)
		return
	}

	lineIndex := 0
	lines := strings.Split(strings.TrimSpace(string(stdout)), "\n")
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

		attrs := core.AssetAttrs{Fingerprinted: true}
		if len(ep.Technologies) > 0 {
			attrs.Technologies = append([]string(nil), ep.Technologies...)
		}
		if ep.StatusCode != nil {
			attrs.StatusCode = ep.StatusCode
		}
		if err := e.queries.UpdateAssetState(hostAsset.ID, attrs); err != nil {
			log.Printf("[scanengine] update asset state %s: %v", hostAsset.ID, err)
		}
	}

	for _, a := range newAssets {
		parentID := ""
		if lineIndex < len(lines) {
			var entry struct {
				Input string `json:"input"`
			}
			if json.Unmarshal([]byte(strings.TrimSpace(lines[lineIndex])), &entry) == nil && entry.Input != "" {
				parentID = parentForInput(entry.Input)
			}
		}
		lineIndex++
		if parentID == "" {
			parentID = parentForInput(a.NormalizedValue)
		}
		a.ParentID = parentID
		a.Attrs.Fingerprinted = true
		for _, ep := range endpoints {
			if ep.URL == a.Value || ep.Host == a.NormalizedValue {
				if len(ep.Technologies) > 0 {
					a.Attrs.Technologies = append([]string(nil), ep.Technologies...)
				}
				if ep.StatusCode != nil {
					a.Attrs.StatusCode = ep.StatusCode
				}
				break
			}
		}
		e.prepareChildAsset(a, parentID)
		e.processNewAsset(ctx, a)
	}
}

func (e *ScanEngine) onBatchNmapComplete(ctx context.Context, w *models.ScanWorkItem, stdout []byte) {
	members := parseBatchMembers(w)
	byHostPort := batchMemberByHostPort(members)
	results := fingerprint.ParseNmapXMLOutput(string(stdout))
	for _, r := range results {
		key := fmt.Sprintf("%s:%d", strings.ToLower(r.IP), r.Port)
		parentID := w.AssetID
		if m, ok := byHostPort[key]; ok {
			parentID = m.AssetID
		}
		if _, _, err := e.merger.CreatePortIfNotExists(parentID, r.Port, r.Protocol, "nmap"); err != nil {
			log.Printf("[scanengine] nmap port %s:%d: %v", r.IP, r.Port, err)
		}
		if fingerprint.IsWebService(r) {
			targets := fingerprint.MakeHTTPTargets([]fingerprint.NmapServiceResult{r})
			for _, target := range targets {
				a := &core.DiscoveryAsset{
					ID:              util.GenerateID(),
					Type:            core.AssetHTTPService,
					Value:           target,
					NormalizedValue: target,
					ParentID:        parentID,
					SourceTool:      "nmap",
				}
				e.prepareChildAsset(a, parentID)
				e.processNewAsset(ctx, a)
			}
		}
	}
}

func (e *ScanEngine) onBatchNucleiComplete(ctx context.Context, w *models.ScanWorkItem, stdout []byte) {
	members := parseBatchMembers(w)
	byURL := batchMemberByValue(members)
	taskID := ""
	if w.TaskID != nil {
		taskID = *w.TaskID
	}

	results, parseErrs := parser.ParseNuclei(strings.NewReader(string(stdout)))
	for _, pe := range parseErrs {
		log.Printf("[scanengine] parse nuclei line %d: %s", pe.Line, pe.Message)
	}

	for _, nr := range results {
		assetID := w.AssetID
		hostKey := strings.ToLower(strings.TrimSpace(nr.Host))
		if m, ok := byURL[hostKey]; ok {
			assetID = m.AssetID
		} else {
			for _, m := range members {
				if strings.Contains(hostKey, strings.ToLower(m.Value)) || strings.Contains(strings.ToLower(nr.MatchedAt), strings.ToLower(m.Value)) {
					assetID = m.AssetID
					break
				}
			}
		}
		created, updated, err := e.nucleiPersist.Persist(e.projectID, e.runID, taskID, assetID, []byte(nr.RawLine+"\n"))
		if err != nil {
			log.Printf("[scanengine] persist nuclei finding: %v", err)
			continue
		}
		if created+updated > 0 {
			log.Printf("[scanengine] nuclei batch finding asset=%s created=%d updated=%d", assetID, created, updated)
		}
	}
	_ = ctx
}
