package scanengine

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/P0m32Kun/Anchor/internal/cdn"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/parser"
	"github.com/P0m32Kun/Anchor/internal/scanengine/core"
	"github.com/P0m32Kun/Anchor/internal/scanengine/domainpool"
	"github.com/P0m32Kun/Anchor/internal/scanengine/pool"
	"github.com/P0m32Kun/Anchor/internal/scanengine/queue"
	"github.com/P0m32Kun/Anchor/internal/scanengine/work"
	"github.com/P0m32Kun/Anchor/internal/toolregistry"
	"github.com/P0m32Kun/Anchor/internal/util"
	"time"
)

func (e *ScanEngine) initTier1Pools(ctx context.Context) {
	e.tier1Mu.Lock()
	if e.tier1Scheduled == nil {
		e.tier1Scheduled = make(map[string]struct{})
	}
	e.tier1Mu.Unlock()

	flushTimeout := e.config.Tier1FlushTimeout
	if flushTimeout <= 0 {
		flushTimeout = 10 * time.Second
	}

	flush := func(action core.TaskAction, ev pool.FlushEvent) {
		e.onTier1PoolFlush(ctx, action, ev)
	}

	hostCfg := pool.DefaultHostPoolConfig(e.dataDir)
	hostCfg.FlushTimeout = flushTimeout
	e.hostPool = pool.New(hostCfg, func(ev pool.FlushEvent) {
		flush(core.ActionDNSResolve, ev)
	})

	cdnCfg := pool.DefaultIPPoolConfig(e.dataDir, "cdn_batch", "cdnpool", 100)
	cdnCfg.FlushTimeout = flushTimeout
	e.cdnPool = pool.New(cdnCfg, func(ev pool.FlushEvent) {
		flush(core.ActionCDNCheck, ev)
	})

	portCfg := pool.DefaultIPPoolConfig(e.dataDir, "port_batch", "portpool", 50)
	portCfg.FlushTimeout = flushTimeout
	e.portPool = pool.New(portCfg, func(ev pool.FlushEvent) {
		flush(core.ActionPortScan, ev)
	})
	e.hostPool.Start()
	e.cdnPool.Start()
	e.portPool.Start()

	e.domainPool = domainpool.New(domainpool.Config{
		BatchSize:    50,
		FlushTimeout: flushTimeout,
		DataDir:      e.dataDir,
	}, func(ev domainpool.FlushEvent) {
		members := make([]pool.Member, 0, len(ev.Domains))
		for _, d := range ev.Domains {
			bk := e.bucketForAssetID(d.AssetID)
			members = append(members, pool.Member{
				Value:     d.Value,
				AssetID:   d.AssetID,
				BucketKey: bk,
			})
		}
		e.onTier1PoolFlush(ctx, core.ActionSubdomainEnum, pool.FlushEvent{
			Members:    members,
			FilePath:   ev.FilePath,
			Generation: ev.Generation,
		})
	})
	e.domainPool.Start()
}

func (e *ScanEngine) stopTier1Pools() {
	if e.hostPool != nil {
		e.hostPool.Stop()
	}
	if e.cdnPool != nil {
		e.cdnPool.Stop()
	}
	if e.portPool != nil {
		e.portPool.Stop()
	}
	if e.domainPool != nil {
		e.domainPool.Stop()
	}
}

func (e *ScanEngine) stopTier1PoolsOnce() {
	e.tier1Stopped.Do(func() {
		e.stopTier1Pools()
	})
}

func (e *ScanEngine) flushTier1IfBlockingHigherStages() {
	if e.pqIsEmptyOrOnlyDiscovery() {
		return
	}
	if e.hostPool != nil && e.hostPool.Len() > 0 {
		e.hostPool.FlushNow()
	}
	if e.cdnPool != nil && e.cdnPool.Len() > 0 {
		e.cdnPool.FlushNow()
	}
	if e.portPool != nil && e.portPool.Len() > 0 {
		e.portPool.FlushNow()
	}
	if e.domainPool != nil && e.domainPool.Len() > 0 {
		e.domainPool.FlushNow()
	}
}

func (e *ScanEngine) pqIsEmptyOrOnlyDiscovery() bool {
	if e.pq.IsEmpty() {
		return true
	}
	depth := e.pq.StageDepth()
	for stage, count := range depth {
		if count > 0 && stage > queue.StageSubdomain {
			return false
		}
	}
	return true
}

func (e *ScanEngine) isTier1Action(action core.TaskAction) bool {
	switch action {
	case core.ActionSubdomainEnum, core.ActionDNSResolve, core.ActionCDNCheck, core.ActionPortScan:
		return true
	default:
		return false
	}
}

func (e *ScanEngine) claimTier1(action core.TaskAction, assetID string) bool {
	key := string(action) + ":" + assetID
	e.tier1Mu.Lock()
	defer e.tier1Mu.Unlock()
	if _, ok := e.tier1Scheduled[key]; ok {
		return false
	}
	e.tier1Scheduled[key] = struct{}{}
	return true
}

func (e *ScanEngine) enqueueTier1Asset(ctx context.Context, a *core.DiscoveryAsset, dw core.DerivedWork, bucketKey string) {
	if !e.claimTier1(dw.Action, dw.AssetID) {
		return
	}

	hostValue := strings.TrimSpace(a.Value)
	switch dw.Action {
	case core.ActionSubdomainEnum:
		if e.domainPool == nil {
			return
		}
		e.domainPool.Add(domainpool.PooledDomain{
			Value:    hostValue,
			AssetID:  dw.AssetID,
			ParentID: a.ParentID,
			Depth:    a.DiscoveryDepth,
		})
	case core.ActionDNSResolve:
		if e.hostPool == nil {
			return
		}
		e.hostPool.Add(pool.Member{Value: hostValue, AssetID: dw.AssetID, BucketKey: bucketKey})
	case core.ActionCDNCheck:
		if e.cdnPool == nil {
			return
		}
		ip := hostWithoutPort(hostValue)
		e.cdnPool.Add(pool.Member{Value: ip, AssetID: dw.AssetID, BucketKey: bucketKey})
	case core.ActionPortScan:
		if e.portPool == nil {
			return
		}
		ip := hostWithoutPort(hostValue)
		e.portPool.Add(pool.Member{Value: ip, AssetID: dw.AssetID, BucketKey: bucketKey})
	default:
		return
	}
	_ = ctx
}

func (e *ScanEngine) onTier1PoolFlush(ctx context.Context, action core.TaskAction, ev pool.FlushEvent) {
	if len(ev.Members) == 0 {
		return
	}
	bucketKey := "tier1:" + string(action)
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
		log.Printf("[scanengine] create batch work %s gen %d: %v", action, ev.Generation, err)
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

func parseBatchMembers(w *models.ScanWorkItem) []models.WorkBatchMember {
	if w == nil || w.MemberAssetIDs == "" {
		return nil
	}
	var members []models.WorkBatchMember
	if err := json.Unmarshal([]byte(w.MemberAssetIDs), &members); err != nil {
		log.Printf("[scanengine] parse batch members for %s: %v", w.ID, err)
	}
	return members
}

func batchMemberByValue(members []models.WorkBatchMember) map[string]models.WorkBatchMember {
	out := make(map[string]models.WorkBatchMember, len(members))
	for _, m := range members {
		key := strings.ToLower(strings.TrimSpace(m.Value))
		if key != "" {
			out[key] = m
		}
	}
	return out
}

func batchMemberByIP(members []models.WorkBatchMember) map[string]models.WorkBatchMember {
	out := make(map[string]models.WorkBatchMember, len(members))
	for _, m := range members {
		key := strings.ToLower(hostWithoutPort(m.Value))
		if key != "" {
			out[key] = m
		}
	}
	return out
}

func (e *ScanEngine) workHostForThrottle(w *models.ScanWorkItem) (string, error) {
	if w != nil && w.BatchMode {
		members := parseBatchMembers(w)
		if len(members) > 0 {
			return members[0].Value, nil
		}
	}
	return e.assetHostValue(w.AssetID)
}

func (e *ScanEngine) buildBatchParams(w *models.ScanWorkItem) (toolregistry.RenderParams, func(), error) {
	switch core.TaskAction(w.Action) {
	case core.ActionHTTPXFingerprint, core.ActionServiceFingerprint, core.ActionNucleiScan:
		return e.buildTier2BatchParams(w)
	}
	cfg := e.config.Pipeline
	switch core.TaskAction(w.Action) {
	case core.ActionDNSResolve:
		return toolregistry.RenderParams{
			"host_file": w.InputFile,
		}, nil, nil
	case core.ActionPortScan:
		return toolregistry.RenderParams{
			"host_file":  w.InputFile,
			"port_range": cfg.PortRange,
			"rate":       cfg.NaabuRate,
			"threads":    cfg.NaabuThreads,
			"timeout":    cfg.NaabuTimeout,
		}, nil, nil
	case core.ActionCDNCheck:
		lines, err := pool.ReadLines(w.InputFile)
		if err != nil {
			return nil, nil, err
		}
		return toolregistry.RenderParams{
			"ips": strings.Join(lines, ","),
		}, nil, nil
	case core.ActionSubdomainEnum:
		return toolregistry.RenderParams{
			"domain_file": w.InputFile,
		}, nil, nil
	default:
		return nil, nil, fmt.Errorf("unsupported batch action: %s", w.Action)
	}
}

func parentAssetForSubdomain(host string, members []models.WorkBatchMember) string {
	host = strings.ToLower(strings.TrimSpace(host))
	for _, m := range members {
		dom := strings.ToLower(strings.TrimSpace(m.Value))
		if dom == "" {
			continue
		}
		if host == dom || strings.HasSuffix(host, "."+dom) {
			return m.AssetID
		}
	}
	if len(members) > 0 {
		return members[0].AssetID
	}
	return ""
}

func (e *ScanEngine) onBatchDNSComplete(ctx context.Context, w *models.ScanWorkItem, stdout []byte) {
	members := parseBatchMembers(w)
	byHost := batchMemberByValue(members)
	results, parseErrs := parser.ParseDNSx(strings.NewReader(string(stdout)))
	for _, pe := range parseErrs {
		log.Printf("[scanengine] parse dnsx line %d: %s", pe.Line, pe.Message)
	}
	alive := true
	for _, rec := range results {
		parentID := w.AssetID
		if m, ok := byHost[strings.ToLower(rec.Host)]; ok {
			parentID = m.AssetID
		}
		for _, ip := range parser.ExtractDNSxIPs(rec) {
			a := &core.DiscoveryAsset{
				ID:              util.GenerateID(),
				Type:            core.AssetIP,
				Value:           ip,
				NormalizedValue: ip,
				ParentID:        parentID,
				SourceTool:      "dnsx",
				Attrs:           core.AssetAttrs{Alive: &alive},
			}
			e.prepareChildAsset(a, parentID)
			e.processNewAsset(ctx, a)
		}
		for _, cname := range parser.ExtractDNSxCNAMEs(rec) {
			a := &core.DiscoveryAsset{
				ID:              util.GenerateID(),
				Type:            core.AssetSubdomain,
				Value:           cname,
				NormalizedValue: cname,
				ParentID:        parentID,
				SourceTool:      "dnsx",
			}
			e.prepareChildAsset(a, parentID)
			e.processNewAsset(ctx, a)
		}
	}
}

func (e *ScanEngine) onBatchPortScanComplete(ctx context.Context, w *models.ScanWorkItem, stdout []byte) {
	members := parseBatchMembers(w)
	byIP := batchMemberByIP(members)
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
		parentID := w.AssetID
		if m, ok := byIP[strings.ToLower(hostWithoutPort(host))]; ok {
			parentID = m.AssetID
		}
		if _, _, err := e.merger.CreatePortIfNotExists(parentID, r.Port, "tcp", "naabu"); err != nil {
			log.Printf("[scanengine] create port %s:%d: %v", host, r.Port, err)
		}
		a := &core.DiscoveryAsset{
			ID:              util.GenerateID(),
			Type:            core.AssetIPPort,
			Value:           fmt.Sprintf("%s:%d", host, r.Port),
			NormalizedValue: fmt.Sprintf("%s:%d", host, r.Port),
			ParentID:        parentID,
			SourceTool:      "naabu",
		}
		e.prepareChildAsset(a, parentID)
		e.processNewAsset(ctx, a)
	}
}

func (e *ScanEngine) onBatchSubfinderComplete(ctx context.Context, w *models.ScanWorkItem, stdout []byte) {
	members := parseBatchMembers(w)
	results, parseErrs := parser.ParseSubfinder(strings.NewReader(string(stdout)))
	for _, pe := range parseErrs {
		log.Printf("[scanengine] parse subfinder line %d: %s", pe.Line, pe.Message)
	}
	for _, r := range results {
		if r.Host == "" {
			continue
		}
		parentID := parentAssetForSubdomain(r.Host, members)
		a := &core.DiscoveryAsset{
			ID:              util.GenerateID(),
			Type:            core.AssetSubdomain,
			Value:           r.Host,
			NormalizedValue: r.Host,
			ParentID:        parentID,
			SourceTool:      "subfinder",
		}
		e.prepareChildAsset(a, parentID)
		e.processNewAsset(ctx, a)
	}
}

func (e *ScanEngine) onBatchCDNComplete(ctx context.Context, w *models.ScanWorkItem, stdout []byte) {
	members := parseBatchMembers(w)
	byIP := batchMemberByIP(members)
	ips := make([]string, 0, len(members))
	for _, m := range members {
		ips = append(ips, hostWithoutPort(m.Value))
	}
	_, cdnResults, err := cdn.ParseJSONLOutput(stdout, ips)
	if err != nil {
		log.Printf("[scanengine] parse cdncheck batch: %v", err)
		return
	}
	isCDN := true
	for _, r := range cdnResults {
		parentID := w.AssetID
		if m, ok := byIP[strings.ToLower(r.IP)]; ok {
			parentID = m.AssetID
		}
		a := &core.DiscoveryAsset{
			ID:              util.GenerateID(),
			Type:            core.AssetIP,
			Value:           r.IP,
			NormalizedValue: r.IP,
			ParentID:        parentID,
			SourceTool:      "cdncheck",
			Attrs:           core.AssetAttrs{IsCDN: &isCDN},
		}
		e.prepareChildAsset(a, parentID)
		e.processNewAsset(ctx, a)
	}
}
