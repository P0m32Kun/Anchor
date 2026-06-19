package worker

import (
	"fmt"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// BuildSubfinderCommand
// ---------------------------------------------------------------------------

func TestBuildSubfinderCommand(t *testing.T) {
	tests := []struct {
		name             string
		domain           string
		rateLimit        int
		threads          int
		timeout          int
		mode             string
		providerConfig   string
		wantArgs         []string
		wantNotPresent   []string
	}{
		{
			name:    "defaults only",
			domain:  "example.com",
			wantArgs: []string{"subfinder", "-d", "example.com", "-oJ"},
			wantNotPresent: []string{"-passive", "-pc", "-rate-limit", "-t", "-timeout"},
		},
		{
			name:       "passive mode",
			domain:     "example.com",
			mode:       "passive",
			wantArgs:   []string{"subfinder", "-d", "example.com", "-oJ", "-passive"},
			wantNotPresent: []string{"-pc", "-rate-limit", "-t", "-timeout"},
		},
		{
			name:           "with provider config",
			domain:         "example.com",
			providerConfig: "/opt/providers.yaml",
			wantArgs:       []string{"subfinder", "-d", "example.com", "-oJ", "-pc", "/opt/providers.yaml"},
			wantNotPresent: []string{"-passive", "-rate-limit", "-t", "-timeout"},
		},
		{
			name:      "all options set",
			domain:    "example.com",
			rateLimit: 100,
			threads:   50,
			timeout:   30,
			mode:      "passive",
			providerConfig: "/tmp/pc.yaml",
			wantArgs: []string{
				"subfinder", "-d", "example.com", "-oJ",
				"-passive", "-pc", "/tmp/pc.yaml",
				"-rate-limit", "100", "-t", "50", "-timeout", "30",
			},
		},
		{
			name:      "zero values omitted",
			domain:    "test.org",
			rateLimit: 0,
			threads:   0,
			timeout:   0,
			wantArgs:  []string{"subfinder", "-d", "test.org", "-oJ"},
			wantNotPresent: []string{"-passive", "-rate-limit", "-t", "-timeout"},
		},
		{
			name:      "non-passive mode ignored",
			domain:    "a.com",
			mode:      "active",
			wantArgs:  []string{"subfinder", "-d", "a.com", "-oJ"},
			wantNotPresent: []string{"-passive"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := BuildSubfinderCommand(tt.domain, tt.rateLimit, tt.threads, tt.timeout, tt.mode, tt.providerConfig)
			for _, want := range tt.wantArgs {
				if !containsArg(args, want) {
					t.Errorf("missing %q in %s", want, stringsJoin(args))
				}
			}
			for _, absent := range tt.wantNotPresent {
				if containsArg(args, absent) {
					t.Errorf("unexpected %q in %s", absent, stringsJoin(args))
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// BuildHttpxCommand
// ---------------------------------------------------------------------------

func TestBuildHttpxCommand(t *testing.T) {
	tests := []struct {
		name           string
		hostFile       string
		rateLimit      int
		threads        int
		customFpFile   string
		wantArgs       []string
		wantNotPresent []string
	}{
		{
			name:     "defaults",
			hostFile: "/tmp/hosts.txt",
			wantArgs: []string{"httpx", "-json", "-l", "/tmp/hosts.txt", "-follow-redirects", "-td"},
			wantNotPresent: []string{"-rate-limit", "-threads", "-cff"},
		},
		{
			name:      "with rate and threads",
			hostFile:  "/tmp/hosts.txt",
			rateLimit: 200,
			threads:   50,
			wantArgs:  []string{"httpx", "-json", "-l", "/tmp/hosts.txt", "-follow-redirects", "-td", "-rate-limit", "200", "-threads", "50"},
			wantNotPresent: []string{"-cff"},
		},
		{
			name:         "with custom fingerprint file",
			hostFile:     "/tmp/hosts.txt",
			customFpFile: "/opt/wappalyzer.json",
			wantArgs:     []string{"httpx", "-json", "-l", "/tmp/hosts.txt", "-follow-redirects", "-td", "-cff", "/opt/wappalyzer.json"},
			wantNotPresent: []string{"-rate-limit", "-threads"},
		},
		{
			name:         "all options",
			hostFile:     "/tmp/h.txt",
			rateLimit:    50,
			threads:      10,
			customFpFile: "/opt/fp.json",
			wantArgs:     []string{"httpx", "-json", "-l", "/tmp/h.txt", "-follow-redirects", "-td", "-rate-limit", "50", "-threads", "10", "-cff", "/opt/fp.json"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := BuildHttpxCommand(tt.hostFile, tt.rateLimit, tt.threads, tt.customFpFile)
			for _, want := range tt.wantArgs {
				if !containsArg(args, want) {
					t.Errorf("missing %q in %s", want, stringsJoin(args))
				}
			}
			for _, absent := range tt.wantNotPresent {
				if containsArg(args, absent) {
					t.Errorf("unexpected %q in %s", absent, stringsJoin(args))
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// BuildNaabuCommand
// ---------------------------------------------------------------------------

func TestBuildNaabuCommand(t *testing.T) {
	tests := []struct {
		name         string
		hostFile     string
		portRange    string
		rate         int
		threads      int
		timeout      int
		wantArgs     []string
		wantNotPresent []string
	}{
		{
			name:      "empty port range defaults to top100",
			hostFile:  "/tmp/hosts.txt",
			portRange: "",
			wantArgs:  []string{"naabu", "-json", "-list", "/tmp/hosts.txt"},
			wantNotPresent: []string{"-tp", "-p", "-rate", "-c", "-timeout"},
		},
		{
			name:      "tp100 alias",
			hostFile:  "/tmp/hosts.txt",
			portRange: "tp100",
			wantArgs:  []string{"naabu", "-json", "-list", "/tmp/hosts.txt"},
			wantNotPresent: []string{"-tp", "-p"},
		},
		{
			name:      "top100 alias",
			hostFile:  "/tmp/hosts.txt",
			portRange: "top100",
			wantArgs:  []string{"naabu", "-json", "-list", "/tmp/hosts.txt"},
			wantNotPresent: []string{"-tp", "-p"},
		},
		{
			name:      "tp1000",
			hostFile:  "/tmp/hosts.txt",
			portRange: "tp1000",
			wantArgs:  []string{"naabu", "-json", "-list", "/tmp/hosts.txt", "-tp", "1000"},
			wantNotPresent: []string{"-p"},
		},
		{
			name:      "top1000 alias",
			hostFile:  "/tmp/hosts.txt",
			portRange: "top1000",
			wantArgs:  []string{"naabu", "-json", "-list", "/tmp/hosts.txt", "-tp", "1000"},
		},
		{
			name:      "tpfull",
			hostFile:  "/tmp/hosts.txt",
			portRange: "tpfull",
			wantArgs:  []string{"naabu", "-json", "-list", "/tmp/hosts.txt", "-tp", "full"},
		},
		{
			name:      "full alias",
			hostFile:  "/tmp/hosts.txt",
			portRange: "full",
			wantArgs:  []string{"naabu", "-json", "-list", "/tmp/hosts.txt", "-tp", "full"},
		},
		{
			name:      "high-risk",
			hostFile:  "/tmp/hosts.txt",
			portRange: "high-risk",
			wantArgs:  []string{"naabu", "-json", "-list", "/tmp/hosts.txt", "-p", HighRiskPorts},
		},
		{
			name:      "highrisk alias",
			hostFile:  "/tmp/hosts.txt",
			portRange: "highrisk",
			wantArgs:  []string{"naabu", "-json", "-list", "/tmp/hosts.txt", "-p", HighRiskPorts},
		},
		{
			name:      "hr alias",
			hostFile:  "/tmp/hosts.txt",
			portRange: "hr",
			wantArgs:  []string{"naabu", "-json", "-list", "/tmp/hosts.txt", "-p", HighRiskPorts},
		},
		{
			name:      "custom port list",
			hostFile:  "/tmp/hosts.txt",
			portRange: "80,443,8080",
			wantArgs:  []string{"naabu", "-json", "-list", "/tmp/hosts.txt", "-p", "80,443,8080"},
		},
		{
			name:      "rate and threads and timeout",
			hostFile:  "/tmp/hosts.txt",
			portRange: "",
			rate:      1000,
			threads:   50,
			timeout:   5000,
			wantArgs:  []string{"naabu", "-json", "-list", "/tmp/hosts.txt", "-rate", "1000", "-c", "50", "-timeout", "5000"},
		},
		{
			name:      "case insensitive port range",
			hostFile:  "/tmp/hosts.txt",
			portRange: "TP1000",
			wantArgs:  []string{"naabu", "-json", "-list", "/tmp/hosts.txt", "-tp", "1000"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := BuildNaabuCommand(tt.hostFile, tt.portRange, tt.rate, tt.threads, tt.timeout)
			for _, want := range tt.wantArgs {
				if !containsArg(args, want) {
					t.Errorf("missing %q in %s", want, stringsJoin(args))
				}
			}
			for _, absent := range tt.wantNotPresent {
				if containsArg(args, absent) {
					t.Errorf("unexpected %q in %s", absent, stringsJoin(args))
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// BuildNucleiCommand
// ---------------------------------------------------------------------------

func TestBuildNucleiCommand(t *testing.T) {
	baseArgs := []string{"nuclei", "-jsonl", "-l", "/tmp/targets.txt", "-stats", "-si", "30", "-dut=false", "-code"}

	tests := []struct {
		name             string
		profile          string
		rateLimit        int
		rateLimitPerMin  int
		concurrency      int
		tags             []string
		scanDepth        string
		workflowDir      string
		templatePath     string
		wantArgs         []string
		wantNotPresent   []string
	}{
		{
			name:    "default profile (empty string)",
			profile: "",
			wantArgs: append(append([]string{}, baseArgs...), "-severity", "critical,high,medium", "-timeout", "5"),
			wantNotPresent: []string{"-w", "-tags", "-c", "-rlm", "-rl"},
		},
		{
			name:    "light profile",
			profile: "light",
			wantArgs: append(append([]string{}, baseArgs...), "-severity", "critical,high", "-timeout", "3"),
		},
		{
			name:    "standard profile",
			profile: "standard",
			wantArgs: append(append([]string{}, baseArgs...), "-severity", "critical,high,medium", "-timeout", "5"),
		},
		{
			name:    "deep profile",
			profile: "deep",
			wantArgs: append(append([]string{}, baseArgs...), "-severity", "critical,high,medium,low,info", "-timeout", "10"),
		},
		{
			name:        "scanDepth tags with tags",
			profile:     "standard",
			scanDepth:   "tags",
			tags:        []string{"cve", "rce"},
			wantArgs:    append(append([]string{}, baseArgs...), "-severity", "critical,high,medium", "-timeout", "5", "-tags", "cve,rce"),
		},
		{
			name:        "scanDepth empty (default to tags) with tags",
			profile:     "standard",
			scanDepth:   "",
			tags:        []string{"xss"},
			wantArgs:    append(append([]string{}, baseArgs...), "-severity", "critical,high,medium", "-timeout", "5", "-tags", "xss"),
		},
		{
			name:        "scanDepth workflow",
			profile:     "standard",
			scanDepth:   "workflow",
			workflowDir: "/opt/workflows",
			wantArgs:    append(append([]string{}, baseArgs...), "-severity", "critical,high,medium", "-timeout", "5", "-w", "/opt/workflows"),
			wantNotPresent: []string{"-tags"},
		},
		{
			name:        "scanDepth workflow without workflowDir",
			profile:     "standard",
			scanDepth:   "workflow",
			workflowDir: "",
			wantArgs:    append(append([]string{}, baseArgs...), "-severity", "critical,high,medium", "-timeout", "5"),
			wantNotPresent: []string{"-w"},
		},
		{
			name:        "scanDepth both",
			profile:     "standard",
			scanDepth:   "both",
			workflowDir: "/opt/workflows",
			tags:        []string{"cve"},
			wantArgs:    append(append([]string{}, baseArgs...), "-severity", "critical,high,medium", "-timeout", "5", "-w", "/opt/workflows", "-tags", "cve"),
		},
		{
			name:        "scanDepth both without workflowDir",
			profile:     "standard",
			scanDepth:   "both",
			tags:        []string{"sqli"},
			wantArgs:    append(append([]string{}, baseArgs...), "-severity", "critical,high,medium", "-timeout", "5", "-tags", "sqli"),
			wantNotPresent: []string{"-w"},
		},
		{
			name:         "templatePath overrides scanDepth",
			profile:      "standard",
			scanDepth:    "both",
			tags:         []string{"cve"},
			workflowDir:  "/opt/workflows",
			templatePath: "/opt/tpl/custom.yaml",
			wantArgs:     append(append([]string{}, baseArgs...), "-severity", "critical,high,medium", "-timeout", "5", "-w", "/opt/tpl/custom.yaml"),
			wantNotPresent: []string{"-tags"},
		},
		{
			name:            "concurrency and rate limits",
			profile:         "standard",
			concurrency:     25,
			rateLimit:       100,
			rateLimitPerMin: 3000,
			wantArgs:        append(append([]string{}, baseArgs...), "-severity", "critical,high,medium", "-timeout", "5", "-c", "25", "-rlm", "3000", "-rl", "100"),
		},
		{
			name:            "zero rate limits omitted",
			profile:         "light",
			concurrency:     0,
			rateLimit:       0,
			rateLimitPerMin: 0,
			wantArgs:        append(append([]string{}, baseArgs...), "-severity", "critical,high", "-timeout", "3"),
			wantNotPresent:  []string{"-c", "-rlm", "-rl"},
		},
		{
			name:    "tags without scanDepth",
			profile: "standard",
			tags:    []string{"misconfig"},
			wantArgs: append(append([]string{}, baseArgs...), "-severity", "critical,high,medium", "-timeout", "5", "-tags", "misconfig"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := BuildNucleiCommand("/tmp/targets.txt", tt.profile, tt.rateLimit, tt.rateLimitPerMin, tt.concurrency, tt.tags, tt.scanDepth, tt.workflowDir, tt.templatePath)
			for _, want := range tt.wantArgs {
				if !containsArg(args, want) {
					t.Errorf("missing %q in %s", want, stringsJoin(args))
				}
			}
			for _, absent := range tt.wantNotPresent {
				if containsArg(args, absent) {
					t.Errorf("unexpected %q in %s", absent, stringsJoin(args))
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// BuildNucleiCustomCommand
// ---------------------------------------------------------------------------

func TestBuildNucleiCustomCommand(t *testing.T) {
	tests := []struct {
		name              string
		profile           string
		rateLimit         int
		tags              []string
		customTemplatesDir string
		customWorkflowsDir string
		wantArgs          []string
		wantNotPresent    []string
	}{
		{
			name:              "all empty",
			profile:           "standard",
			customTemplatesDir: "",
			customWorkflowsDir: "",
			wantArgs:          []string{"nuclei", "-jsonl", "-l", "/tmp/t.txt", "-stats", "-si", "30", "-dut=false", "-code", "-severity", "critical,high,medium", "-timeout", "5"},
			wantNotPresent:    []string{"-t", "-w"},
		},
		{
			name:              "with custom templates dir",
			profile:           "standard",
			customTemplatesDir: "/opt/custom-templates",
			wantArgs:          []string{"nuclei", "-jsonl", "-l", "/tmp/t.txt", "-stats", "-si", "30", "-dut=false", "-code", "-severity", "critical,high,medium", "-timeout", "5", "-t", "/opt/custom-templates"},
			wantNotPresent:    []string{"-w"},
		},
		{
			name:              "with custom workflows dir",
			profile:           "standard",
			customWorkflowsDir: "/opt/custom-workflows",
			wantArgs:          []string{"nuclei", "-jsonl", "-l", "/tmp/t.txt", "-stats", "-si", "30", "-dut=false", "-code", "-severity", "critical,high,medium", "-timeout", "5", "-w", "/opt/custom-workflows"},
			wantNotPresent:    []string{"-t"},
		},
		{
			name:              "both dirs set",
			profile:           "deep",
			tags:              []string{"cve"},
			rateLimit:         50,
			customTemplatesDir: "/opt/tpl",
			customWorkflowsDir: "/opt/wf",
			wantArgs:          []string{"nuclei", "-jsonl", "-l", "/tmp/t.txt", "-stats", "-si", "30", "-dut=false", "-code", "-severity", "critical,high,medium,low,info", "-timeout", "10", "-tags", "cve", "-rl", "50", "-t", "/opt/tpl", "-w", "/opt/wf"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := BuildNucleiCustomCommand("/tmp/t.txt", tt.profile, tt.rateLimit, tt.tags, tt.customTemplatesDir, tt.customWorkflowsDir)
			for _, want := range tt.wantArgs {
				if !containsArg(args, want) {
					t.Errorf("missing %q in %s", want, stringsJoin(args))
				}
			}
			for _, absent := range tt.wantNotPresent {
				if containsArg(args, absent) {
					t.Errorf("unexpected %q in %s", absent, stringsJoin(args))
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// BuildFfufCommand
// ---------------------------------------------------------------------------

func TestBuildFfufCommand(t *testing.T) {
	tests := []struct {
		name           string
		target         string
		wordlist       string
		rateLimit      int
		timeout        int
		wantArgs       []string
		wantNotPresent []string
	}{
		{
			name:      "defaults only",
			target:    "https://example.com/FUZZ",
			wordlist:  "/opt/wordlist.txt",
			wantArgs:  []string{"ffuf", "-u", "https://example.com/FUZZ", "-w", "/opt/wordlist.txt", "-json", "-mc", "200,301,302,401,403,405,500", "-s"},
			wantNotPresent: []string{"-rate", "-timeout"},
		},
		{
			name:      "with rate and timeout",
			target:    "https://example.com/FUZZ",
			wordlist:  "/opt/wl.txt",
			rateLimit: 500,
			timeout:   10,
			wantArgs:  []string{"ffuf", "-u", "https://example.com/FUZZ", "-w", "/opt/wl.txt", "-json", "-mc", "200,301,302,401,403,405,500", "-rate", "500", "-timeout", "10", "-s"},
		},
		{
			name:      "zero values omitted",
			target:    "http://localhost/FUZZ",
			wordlist:  "/tmp/dict.txt",
			rateLimit: 0,
			timeout:   0,
			wantArgs:  []string{"ffuf", "-u", "http://localhost/FUZZ", "-w", "/tmp/dict.txt", "-json", "-mc", "200,301,302,401,403,405,500", "-s"},
			wantNotPresent: []string{"-rate", "-timeout"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := BuildFfufCommand(tt.target, tt.wordlist, tt.rateLimit, tt.timeout)
			for _, want := range tt.wantArgs {
				if !containsArg(args, want) {
					t.Errorf("missing %q in %s", want, stringsJoin(args))
				}
			}
			for _, absent := range tt.wantNotPresent {
				if containsArg(args, absent) {
					t.Errorf("unexpected %q in %s", absent, stringsJoin(args))
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// BuildNmapServiceScanCommand
// ---------------------------------------------------------------------------

func TestBuildNmapServiceScanCommand(t *testing.T) {
	tests := []struct {
		name         string
		hostFile     string
		ports        []int
		hostTimeout  int
		wantArgs     []string
		wantNotPresent []string
	}{
		{
			name:        "basic scan",
			hostFile:    "/tmp/hosts.txt",
			ports:       []int{80, 443},
			hostTimeout: 0,
			wantArgs:    []string{"nmap", "-sV", "-p", "80,443", "-iL", "/tmp/hosts.txt", "-oX", "-", "-T4", "-n", "--open"},
			wantNotPresent: []string{"--host-timeout"},
		},
		{
			name:        "with host timeout",
			hostFile:    "/tmp/hosts.txt",
			ports:       []int{22, 80, 443, 8080},
			hostTimeout: 300,
			wantArgs:    []string{"nmap", "-sV", "-p", "22,80,443,8080", "-iL", "/tmp/hosts.txt", "-oX", "-", "-T4", "-n", "--open", "--host-timeout", "300s"},
		},
		{
			name:        "single port",
			hostFile:    "/tmp/h.txt",
			ports:       []int{443},
			hostTimeout: 0,
			wantArgs:    []string{"nmap", "-sV", "-p", "443", "-iL", "/tmp/h.txt", "-oX", "-", "-T4", "-n", "--open"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := BuildNmapServiceScanCommand(tt.hostFile, tt.ports, tt.hostTimeout)
			for _, want := range tt.wantArgs {
				if !containsArg(args, want) {
					t.Errorf("missing %q in %s", want, stringsJoin(args))
				}
			}
			for _, absent := range tt.wantNotPresent {
				if containsArg(args, absent) {
					t.Errorf("unexpected %q in %s", absent, stringsJoin(args))
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// BuildDNSxCommand
// ---------------------------------------------------------------------------

func TestBuildDNSxCommand(t *testing.T) {
	tests := []struct {
		name           string
		hostFile       string
		recordTypes    []string
		rateLimit      int
		threads        int
		timeout        int
		wantArgs       []string
		wantNotPresent []string
	}{
		{
			name:        "default record types (empty)",
			hostFile:    "/tmp/hosts.txt",
			recordTypes: []string{},
			wantArgs:    []string{"dnsx", "-l", "/tmp/hosts.txt", "-json", "-a", "-aaaa", "-cname"},
			wantNotPresent: []string{"-rl", "-t", "-timeout"},
		},
		{
			name:        "nil record types uses defaults",
			hostFile:    "/tmp/hosts.txt",
			recordTypes: nil,
			wantArgs:    []string{"dnsx", "-l", "/tmp/hosts.txt", "-json", "-a", "-aaaa", "-cname"},
		},
		{
			name:        "custom record types",
			hostFile:    "/tmp/hosts.txt",
			recordTypes: []string{"a", "mx", "txt"},
			wantArgs:    []string{"dnsx", "-l", "/tmp/hosts.txt", "-json", "-a", "-mx", "-txt"},
			wantNotPresent: []string{"-aaaa", "-cname"},
		},
		{
			name:        "with rate limit and threads",
			hostFile:    "/tmp/hosts.txt",
			recordTypes: []string{"a"},
			rateLimit:   100,
			threads:     20,
			timeout:     5,
			wantArgs:    []string{"dnsx", "-l", "/tmp/hosts.txt", "-json", "-a", "-rl", "100", "-t", "20", "-timeout", "5"},
		},
		{
			name:        "case insensitive record types",
			hostFile:    "/tmp/hosts.txt",
			recordTypes: []string{"A", "AAAA"},
			wantArgs:    []string{"dnsx", "-l", "/tmp/hosts.txt", "-json", "-a", "-aaaa"},
		},
		{
			name:        "zero values omitted",
			hostFile:    "/tmp/hosts.txt",
			recordTypes: []string{"a"},
			rateLimit:   0,
			threads:     0,
			timeout:     0,
			wantArgs:    []string{"dnsx", "-l", "/tmp/hosts.txt", "-json", "-a"},
			wantNotPresent: []string{"-rl", "-t", "-timeout"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := BuildDNSxCommand(tt.hostFile, tt.recordTypes, tt.rateLimit, tt.threads, tt.timeout)
			for _, want := range tt.wantArgs {
				if !containsArg(args, want) {
					t.Errorf("missing %q in %s", want, stringsJoin(args))
				}
			}
			for _, absent := range tt.wantNotPresent {
				if containsArg(args, absent) {
					t.Errorf("unexpected %q in %s", absent, stringsJoin(args))
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// BuildCDNCheckCommand
// ---------------------------------------------------------------------------

func TestBuildCDNCheckCommand(t *testing.T) {
	args := BuildCDNCheckCommand("1.2.3.4,5.6.7.8")
	wantArgs := []string{"cdncheck", "-i", "1.2.3.4,5.6.7.8", "-jsonl", "-silent", "-duc"}
	for _, want := range wantArgs {
		if !containsArg(args, want) {
			t.Errorf("missing %q in %s", want, stringsJoin(args))
		}
	}
	// Verify no unexpected extras
	if len(args) != len(wantArgs) {
		t.Errorf("expected %d args, got %d: %s", len(wantArgs), len(args), stringsJoin(args))
	}
}

// ---------------------------------------------------------------------------
// BuildNmapAliveCommand
// ---------------------------------------------------------------------------

func TestBuildNmapAliveCommand(t *testing.T) {
	args := BuildNmapAliveCommand("/tmp/hosts.txt")
	wantArgs := []string{"nmap", "-sn", "-n", "-T4", "-oG", "-", "-iL", "/tmp/hosts.txt"}
	for _, want := range wantArgs {
		if !containsArg(args, want) {
			t.Errorf("missing %q in %s", want, stringsJoin(args))
		}
	}
	if len(args) != len(wantArgs) {
		t.Errorf("expected %d args, got %d: %s", len(wantArgs), len(args), stringsJoin(args))
	}
}

// ---------------------------------------------------------------------------
// BuildScreenshotCommand
// ---------------------------------------------------------------------------

func TestBuildScreenshotCommand(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		outputFile string
		width      int
		height     int
		wantArgs   []string
	}{
		{
			name:       "default dimensions",
			url:        "https://example.com",
			outputFile: "/tmp/screenshot.png",
			width:      0,
			height:     0,
			wantArgs: []string{
				"chromium", "--headless", "--disable-gpu", "--no-sandbox",
				"--screenshot=/tmp/screenshot.png",
				"--window-size=1920,1080",
				"--hide-scrollbars",
				"https://example.com",
			},
		},
		{
			name:       "negative dimensions use defaults",
			url:        "https://example.com",
			outputFile: "/tmp/shot.png",
			width:      -1,
			height:     -1,
			wantArgs: []string{
				"chromium", "--headless", "--disable-gpu", "--no-sandbox",
				"--screenshot=/tmp/shot.png",
				"--window-size=1920,1080",
				"--hide-scrollbars",
				"https://example.com",
			},
		},
		{
			name:       "custom dimensions",
			url:        "https://example.com",
			outputFile: "/tmp/wide.png",
			width:      2560,
			height:     1440,
			wantArgs: []string{
				"chromium", "--headless", "--disable-gpu", "--no-sandbox",
				"--screenshot=/tmp/wide.png",
				"--window-size=2560,1440",
				"--hide-scrollbars",
				"https://example.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := BuildScreenshotCommand(tt.url, tt.outputFile, tt.width, tt.height)
			for _, want := range tt.wantArgs {
				if !containsArg(args, want) {
					t.Errorf("missing %q in %s", want, stringsJoin(args))
				}
			}
			if len(args) != len(tt.wantArgs) {
				t.Errorf("expected %d args, got %d: %s", len(tt.wantArgs), len(args), stringsJoin(args))
			}
		})
	}
}

// ---------------------------------------------------------------------------
// appendRateLimitArgs
// ---------------------------------------------------------------------------

func TestAppendRateLimitArgs(t *testing.T) {
	tools := []struct {
		tool      string
		rate      int
		wantFlag  string
		wantValue string
	}{
		{"naabu", 100, "-rate", "100"},
		{"nuclei", 200, "-rl", "200"},
		{"httpx", 50, "-rate-limit", "50"},
		{"subfinder", 75, "-rate-limit", "75"},
		{"dnsx", 30, "-rl", "30"},
		{"ffuf", 500, "-rate", "500"},
	}

	for _, tt := range tools {
		t.Run(fmt.Sprintf("%s_rate_%d", tt.tool, tt.rate), func(t *testing.T) {
			base := []string{"tool"}
			result := appendRateLimitArgs(base, tt.tool, tt.rate)
			if !containsArg(result, tt.wantFlag) {
				t.Errorf("missing flag %q for tool %q in %s", tt.wantFlag, tt.tool, stringsJoin(result))
			}
			if !containsArg(result, tt.wantValue) {
				t.Errorf("missing value %q for tool %q in %s", tt.wantValue, tt.tool, stringsJoin(result))
			}
		})
	}

	t.Run("rate_zero_returns_unchanged", func(t *testing.T) {
		base := []string{"tool", "-x"}
		for _, toolName := range []string{"naabu", "nuclei", "httpx", "subfinder", "dnsx", "ffuf"} {
			result := appendRateLimitArgs(base, toolName, 0)
			if len(result) != len(base) {
				t.Errorf("tool %q: expected no change with rate=0, got %s", toolName, stringsJoin(result))
			}
		}
	})

	t.Run("unknown_tool_returns_unchanged", func(t *testing.T) {
		base := []string{"tool"}
		result := appendRateLimitArgs(base, "unknown", 100)
		if len(result) != len(base) {
			t.Errorf("expected no change for unknown tool, got %s", stringsJoin(result))
		}
	})

	t.Run("negative_rate_returns_unchanged", func(t *testing.T) {
		base := []string{"tool"}
		result := appendRateLimitArgs(base, "naabu", -1)
		if len(result) != len(base) {
			t.Errorf("expected no change for negative rate, got %s", stringsJoin(result))
		}
	})
}

// ---------------------------------------------------------------------------
// HighRiskPorts alias
// ---------------------------------------------------------------------------

func TestHighRiskPortsAlias(t *testing.T) {
	if HighRiskPorts == "" {
		t.Error("HighRiskPorts should not be empty")
	}
	if !strings.Contains(HighRiskPorts, ",") {
		t.Errorf("HighRiskPorts should contain comma-separated ports, got %q", HighRiskPorts)
	}
}
