package queue

// StageRank defines the execution order for scan stages.
// Lower rank = higher priority (executed first).
// The engine only pops items from a stage if no higher-priority stage
// has pending (unfinished) work.
type StageRank int

const (
	StageDiscovery StageRank = 10 // PASSIVE_*
	StageSubdomain StageRank = 20 // SUBDOMAIN_ENUM
	StageResolve   StageRank = 30 // DNS_RESOLVE
	StageCDN       StageRank = 40 // CDN_CHECK
	StagePort      StageRank = 50 // PORT_SCAN
	StageService   StageRank = 60 // SERVICE_FINGERPRINT
	StageWeb       StageRank = 70 // HTTPX_FINGERPRINT
	StageCrawl     StageRank = 80 // KATANA_CRAWL, SPOOR_SCAN
	StageBrute     StageRank = 90 // FFUF_BRUTE
	StageVuln      StageRank = 100 // NUCLEI_SCAN
)

// ActionToStageRank maps a TaskAction string to its StageRank.
// Unknown actions default to StageVuln (lowest priority).
func ActionToStageRank(action string) StageRank {
	switch action {
	case "PASSIVE_SEARCH", "PASSIVE_CERT", "PASSIVE_URL":
		return StageDiscovery
	case "SUBDOMAIN_ENUM":
		return StageSubdomain
	case "DNS_RESOLVE":
		return StageResolve
	case "CDN_CHECK":
		return StageCDN
	case "PORT_SCAN":
		return StagePort
	case "SERVICE_FINGERPRINT":
		return StageService
	case "HTTPX_FINGERPRINT":
		return StageWeb
	case "KATANA_CRAWL", "SPOOR_SCAN":
		return StageCrawl
	case "FFUF_BRUTE":
		return StageBrute
	case "NUCLEI_SCAN":
		return StageVuln
	default:
		return StageVuln
	}
}
