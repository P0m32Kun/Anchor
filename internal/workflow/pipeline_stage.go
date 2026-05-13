package workflow

// StageID identifies a pipeline stage.
type StageID string

const (
	StageClassify    StageID = "classify"
	StageSearch      StageID = "search"
	StageSubdomain   StageID = "subdomain"
	StageResolve     StageID = "resolve"
	StageCDNFilter   StageID = "cdn_filter"
	StageAlive       StageID = "alive"
	StagePortScan    StageID = "portscan"
	StageFingerprint StageID = "fingerprint"
	StageHTTPX       StageID = "httpx"
	StageVuln        StageID = "vuln"
	// Slow-scan stages emitted by SlowScanOrchestrator post-pipeline. They
	// only show up when the main flow has at least one live web endpoint and
	// the matching tool is enabled.
	StageFfuf StageID = "ffuf"
)

// StageEventCallback is invoked when a pipeline stage changes state.
type StageEventCallback func(runID string, stage StageID, status string, errMsg string)

// setStage / completeStage / failStage are thin wrappers over p.emitter,
// preserved so existing Pipeline call sites stay readable. SlowScanOrchestrator
// uses the same StageEmitter directly via WithStageEmitter to share semantics
// without reaching into Pipeline.
func (p *Pipeline) setStage(stage StageID)               { p.emitter.Set(stage) }
func (p *Pipeline) completeStage(stage StageID)          { p.emitter.Complete(stage) }
func (p *Pipeline) failStage(stage StageID, errMsg string) { p.emitter.Fail(stage, errMsg) }
