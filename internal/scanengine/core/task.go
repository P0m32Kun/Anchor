package core

// TaskAction identifies the scan action to perform on an asset.
// Each action maps to exactly one tool invocation.
type TaskAction string

const (
	ActionPassiveSearch      TaskAction = "PASSIVE_SEARCH"
	ActionPassiveCert        TaskAction = "PASSIVE_CERT"
	ActionPassiveURL         TaskAction = "PASSIVE_URL"
	ActionSubdomainEnum      TaskAction = "SUBDOMAIN_ENUM"
	ActionDNSResolve         TaskAction = "DNS_RESOLVE"
	ActionCDNCheck           TaskAction = "CDN_CHECK"
	ActionPortScan           TaskAction = "PORT_SCAN"
	ActionServiceFingerprint TaskAction = "SERVICE_FINGERPRINT"
	ActionHTTPXFingerprint   TaskAction = "HTTPX_FINGERPRINT"
	ActionKatanaCrawl        TaskAction = "KATANA_CRAWL"
	ActionFFUFBrute          TaskAction = "FFUF_BRUTE"
	ActionNucleiScan         TaskAction = "NUCLEI_SCAN"
)

// ActionToTool maps a TaskAction to the tool ID in the registry.
var ActionToTool = map[TaskAction]string{
	ActionPassiveSearch:      "subfinder", // placeholder; passive uses multiple tools
	ActionPassiveCert:        "crt",
	ActionPassiveURL:         "gau",
	ActionSubdomainEnum:      "subfinder",
	ActionDNSResolve:         "dnsx",
	ActionCDNCheck:           "cdncheck",
	ActionPortScan:           "naabu",
	ActionServiceFingerprint: "nmap",
	ActionHTTPXFingerprint:   "httpx",
	ActionKatanaCrawl:        "katana",
	ActionFFUFBrute:          "ffuf",
	ActionNucleiScan:         "nuclei",
}

// ActionToStage maps a TaskAction to the stage ID for UI projection.
var ActionToStage = map[TaskAction]string{
	ActionPassiveSearch:      "search",
	ActionPassiveCert:        "passive_cert",
	ActionPassiveURL:         "passive_url",
	ActionSubdomainEnum:      "subdomain",
	ActionDNSResolve:         "resolve",
	ActionCDNCheck:           "cdn_filter",
	ActionPortScan:           "portscan",
	ActionServiceFingerprint: "fingerprint",
	ActionHTTPXFingerprint:   "httpx",
	ActionKatanaCrawl:        "crawl",
	ActionFFUFBrute:          "ffuf",
	ActionNucleiScan:         "vuln",
}

// DerivedWork represents a work item to be enqueued.
type DerivedWork struct {
	Action  TaskAction
	AssetID string
	Stage   string
}
