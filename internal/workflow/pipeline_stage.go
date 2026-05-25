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
	// URL discovery stages — directory brute-force (ffuf) and JS/HTML URL
	// extraction (urlfinder). Both consume httpx's first-pass output and
	// feed new URLs into a second httpx → nuclei pass.
	StageFfuf       StageID = "ffuf"
	StageURLFinder  StageID = "urlfinder"
	// Second-pass HTTP probing and vulnerability scanning for URLs
	// discovered by ffuf / urlfinder.
	StageHTTPX2 StageID = "httpx_2"
	StageVuln2  StageID = "vuln_2"
	// External-scan-only stages
	StagePassiveCert StageID = "passive_cert" // crt.sh certificate transparency subdomain discovery
	StagePassiveURL  StageID = "passive_url"  // gau historical URL collection
	StageCrawl       StageID = "crawl"        // Katana web crawling
)

// StageEventCallback is invoked when a pipeline stage changes state.
type StageEventCallback func(runID string, stage StageID, status string, errMsg string)

// setStage / completeStage / failStage are thin wrappers over p.emitter
// preserved so existing Pipeline call sites stay readable.
func (p *Pipeline) setStage(stage StageID)               { p.emitter.Set(stage) }
func (p *Pipeline) completeStage(stage StageID)          { p.emitter.Complete(stage) }
func (p *Pipeline) failStage(stage StageID, errMsg string) { p.emitter.Fail(stage, errMsg) }
