package worker

import (
	"fmt"
	"strings"
)

// HighRiskPorts is a curated list of ports that commonly host vulnerable
// or sensitive services. Naabu's built-in top-100/top-1000 sets are biased
// toward Nmap's general-purpose service prevalence and miss several high-value
// targets (e.g. 6379 Redis, 9200 Elasticsearch, 27017 MongoDB, 11434 Ollama).
// This list intentionally favours attack-surface coverage over generality.
const HighRiskPorts = "21,22,23,25,53,80,81,88,110,135,139,143,389,443,445,465,587,636,873,993,995," +
	"1080,1099,1433,1521,1723,2049,2082,2375,2376,2480,3000,3128,3306,3389," +
	"4040,4194,4369,4444,4848,5000,5432,5601,5672,5900,5901,5984,6379,6443," +
	"7000,7001,7002,7077,7474,8000,8001,8008,8009,8020,8060,8080,8081,8086,8088,8090,8161,8200,8443,8500,8531,8888,8983," +
	"9000,9001,9042,9043,9060,9080,9090,9091,9092,9100,9200,9300,9418,9443,9981," +
	"10000,10250,10255,11211,11434,15672,15692,16379,18091," +
	"27017,27018,27019,28017,50000,50070,50075,61613,61616"

// BuildSubfinderCommand builds a Subfinder command for the given domain.
// Output goes to stdout as JSONL so the worker can capture it as an artifact.
func BuildSubfinderCommand(domain string, rateLimit, threads, timeout int) []string {
	args := []string{"subfinder", "-d", domain, "-oJ"}
	if rateLimit > 0 {
		args = append(args, "-rate-limit", fmt.Sprintf("%d", rateLimit))
	}
	if threads > 0 {
		args = append(args, "-t", fmt.Sprintf("%d", threads))
	}
	if timeout > 0 {
		args = append(args, "-timeout", fmt.Sprintf("%d", timeout))
	}
	return args
}

// BuildHttpxCommand builds an httpx command that reads hosts from a file.
// hostFile should contain one host per line.
// Output goes to stdout as JSONL so the worker can capture it as an artifact.
func BuildHttpxCommand(hostFile string, rateLimit, threads int) []string {
	args := []string{"httpx", "-json", "-l", hostFile, "-follow-redirects"}
	if rateLimit > 0 {
		args = append(args, "-rate-limit", fmt.Sprintf("%d", rateLimit))
	}
	if threads > 0 {
		args = append(args, "-threads", fmt.Sprintf("%d", threads))
	}
	return args
}

// BuildNaabuCommand builds a Naabu command that reads hosts from a file.
// hostFile should contain one host per line.
// portRange can be a custom port list (e.g. "80,443,8080"), or one of:
//
//	"top100" / "tp100"  → naabu default (no extra flag)
//	"top1000" / "tp1000" → -tp 1000
//	"full" / "tpfull" / "topfull" → -tp full
//	"high-risk" / "highrisk" / "hr" → curated high-risk ports (Redis,
//	    Elasticsearch, MongoDB, etc.) that the top-N presets miss
//
// Output goes to stdout as JSONL so the worker can capture it as an artifact.
func BuildNaabuCommand(hostFile, portRange string, rate, threads, timeout int) []string {
	args := []string{"naabu", "-json", "-list", hostFile}

	switch strings.ToLower(portRange) {
	case "", "tp100", "top100", "top-100":
		// Naabu default is top 100, no extra flag needed
	case "tp1000", "top1000", "top-1000":
		args = append(args, "-tp", "1000")
	case "tpfull", "topfull", "full", "top-full":
		args = append(args, "-tp", "full")
	case "high-risk", "highrisk", "hr":
		args = append(args, "-p", HighRiskPorts)
	default:
		// User-defined port list
		args = append(args, "-p", portRange)
	}

	if rate > 0 {
		args = append(args, "-rate", fmt.Sprintf("%d", rate))
	}
	if threads > 0 {
		args = append(args, "-c", fmt.Sprintf("%d", threads))
	}
	// Naabu's -timeout is in milliseconds (CLI default 1000ms).
	if timeout > 0 {
		args = append(args, "-timeout", fmt.Sprintf("%d", timeout))
	}

	return args
}

// BuildNucleiCommand builds a Nuclei command.
// scanDepth controls the scanning strategy:
//   - "workflow": run pre-built workflows from workflowDir (precision scan)
//   - "tags": run tag-based template matching (current behavior, broad coverage)
//   - "both": run workflows first, then tag-based for uncovered targets
//
// rateLimit is -rl (requests/second), rateLimitPerMin is -rlm (requests/minute for sensitive targets).
// concurrency is -c (parallel templates/hosts).
//
// -stats -si 30 forces nuclei to emit a progress line to stderr every 30s.
// This keeps the worker's idle-output watchdog (server.go:idleOutputTimeout)
// from misfiring on long scans that simply produce no findings — without the
// stats heartbeat, a 100-target scan with zero matches would emit nothing on
// stdout and get killed at the 60s idle threshold.
func BuildNucleiCommand(targetFile, profile string, rateLimit, rateLimitPerMin, concurrency int, tags []string, scanDepth string, workflowDir string) []string {
	args := []string{"nuclei", "-jsonl", "-l", targetFile, "-stats", "-si", "30"}

	switch profile {
	case "light":
		args = append(args, "-severity", "critical,high", "-timeout", "3")
	case "standard", "":
		args = append(args, "-severity", "critical,high,medium", "-timeout", "5")
	case "deep":
		args = append(args, "-severity", "critical,high,medium,low,info", "-timeout", "10")
	}

	switch scanDepth {
	case "workflow":
		if workflowDir != "" {
			args = append(args, "-w", workflowDir)
		}
	case "both":
		if workflowDir != "" {
			args = append(args, "-w", workflowDir)
		}
		if len(tags) > 0 {
			args = append(args, "-tags", strings.Join(tags, ","))
		}
	default: // "tags" or empty
		if len(tags) > 0 {
			args = append(args, "-tags", strings.Join(tags, ","))
		}
	}

	if concurrency > 0 {
		args = append(args, "-c", fmt.Sprintf("%d", concurrency))
	}
	if rateLimitPerMin > 0 {
		args = append(args, "-rlm", fmt.Sprintf("%d", rateLimitPerMin))
	}
	if rateLimit > 0 {
		args = append(args, "-rl", fmt.Sprintf("%d", rateLimit))
	}

	return args
}

// BuildNucleiCustomCommand builds a Nuclei command that includes custom
// templates (-t) and workflows (-w) from the active custom bundle.
// customTemplatesDir and customWorkflowsDir are absolute paths on the worker;
// either may be empty if that directory does not exist.
func BuildNucleiCustomCommand(targetFile, profile string, rateLimit int, tags []string, customTemplatesDir, customWorkflowsDir string) []string {
	args := BuildNucleiCommand(targetFile, profile, rateLimit, 0, 0, tags, "tags", "")

	if customTemplatesDir != "" {
		args = append(args, "-t", customTemplatesDir)
	}
	if customWorkflowsDir != "" {
		args = append(args, "-w", customWorkflowsDir)
	}

	return args
}

// BuildUrlfinderCommand builds a urlfinder command.
// targetFile should contain one target (domain or URL) per line.
// rateLimit is requests per minute (low for background scanning).
func BuildUrlfinderCommand(targetFile string, rateLimit, timeout int) []string {
	args := []string{"urlfinder", "-list", targetFile, "-json"}
	if rateLimit > 0 {
		args = append(args, "-rate-limit", fmt.Sprintf("%d", rateLimit))
	}
	if timeout > 0 {
		args = append(args, "-timeout", fmt.Sprintf("%d", timeout))
	}
	return args
}

// BuildFfufCommand builds an ffuf directory brute-force command.
// target is a single base URL with FUZZ placeholder (e.g., https://example.com/FUZZ).
// wordlist is the absolute path to the dictionary file.
// rateLimit is requests per second.
func BuildFfufCommand(target, wordlist string, rateLimit, timeout int) []string {
	args := []string{"ffuf", "-u", target, "-w", wordlist, "-json", "-mc", "200,301,302,401,403,405,500"}
	if rateLimit > 0 {
		args = append(args, "-rate", fmt.Sprintf("%d", rateLimit))
	}
	if timeout > 0 {
		args = append(args, "-timeout", fmt.Sprintf("%d", timeout))
	}
	args = append(args, "-s")
	return args
}

// BuildNmapServiceScanCommand builds an nmap -sV command for service fingerprinting.
// hostFile should contain one host per line; ports is the list of ports to scan.
// Output is XML to stdout for reliable parsing.
func BuildNmapServiceScanCommand(hostFile string, ports []int, hostTimeout int) []string {
	portStrs := make([]string, len(ports))
	for i, p := range ports {
		portStrs[i] = fmt.Sprintf("%d", p)
	}
	cmd := []string{
		"nmap", "-sV",
		"-p", strings.Join(portStrs, ","),
		"-iL", hostFile,
		"-oX", "-",
		"-T4", "-n", "--open",
	}
	if hostTimeout > 0 {
		cmd = append(cmd, "--host-timeout", fmt.Sprintf("%ds", hostTimeout))
	}
	return cmd
}

// BuildDNSxCommand builds a dnsx command for DNS resolution.
// hostFile should contain one host per line.
// recordTypes can be a list of record types (e.g. ["a", "aaaa", "cname"]).
func BuildDNSxCommand(hostFile string, recordTypes []string, rateLimit, threads, timeout int) []string {
	args := []string{"dnsx", "-l", hostFile, "-json"}

	if len(recordTypes) > 0 {
		for _, rt := range recordTypes {
			args = append(args, "-"+strings.ToLower(rt))
		}
	} else {
		args = append(args, "-a", "-aaaa", "-cname")
	}

	if rateLimit > 0 {
		args = append(args, "-rl", fmt.Sprintf("%d", rateLimit))
	}
	if threads > 0 {
		args = append(args, "-t", fmt.Sprintf("%d", threads))
	}
	if timeout > 0 {
		args = append(args, "-timeout", fmt.Sprintf("%d", timeout))
	}

	return args
}

// BuildCDNCheckCommand builds a cdncheck command for CDN detection.
// ips should be comma-separated.
func BuildCDNCheckCommand(ips string) []string {
	return []string{"cdncheck", "-i", ips, "-jsonl"}
}

// BuildNmapAliveCommand builds an nmap host-discovery command that reads
// targets from a file (IPs and/or CIDRs, one per line) and writes greppable
// output to stdout. Same-subnet targets are detected via ARP automatically;
// cross-subnet uses ICMP echo + TCP SYN/ACK probes (nmap defaults).
//
//   -sn        ping scan only (no port scan)
//   -n         skip DNS resolution
//   -T4        aggressive timing (safe in trusted/internal networks)
//   -oG -      greppable output to stdout (parsed by parser.ParseNmapAlive)
//   -iL <file> input from file
func BuildNmapAliveCommand(hostFile string) []string {
	return []string{
		"nmap",
		"-sn",
		"-n",
		"-T4",
		"-oG", "-",
		"-iL", hostFile,
	}
}

// appendRateLimitArgs appends tool-specific rate limit flags to the argument list.
// Only adds flags when rate > 0 and the tool supports it.
func appendRateLimitArgs(args []string, tool string, rate int) []string {
	if rate <= 0 {
		return args
	}
	switch strings.ToLower(tool) {
	case "naabu":
		return append(args, "-rate", fmt.Sprintf("%d", rate))
	case "nuclei":
		return append(args, "-rl", fmt.Sprintf("%d", rate))
	case "httpx":
		return append(args, "-rate-limit", fmt.Sprintf("%d", rate))
	case "subfinder":
		return append(args, "-rate-limit", fmt.Sprintf("%d", rate))
	case "dnsx":
		return append(args, "-rl", fmt.Sprintf("%d", rate))
	case "urlfinder":
		return append(args, "-rate-limit", fmt.Sprintf("%d", rate))
	case "ffuf":
		return append(args, "-rate", fmt.Sprintf("%d", rate))
	default:
		return args
	}
}
