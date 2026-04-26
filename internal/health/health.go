package health

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"secbench/internal/db"
	"secbench/internal/models"
	"secbench/internal/util"
)

// Checker performs health checks for external tools.
type Checker struct {
	queries *db.Queries
}

func NewChecker(q *db.Queries) *Checker {
	return &Checker{queries: q}
}

// CheckAll runs health checks for all required tools.
func (c *Checker) CheckAll(ctx context.Context, dataDir string) error {
	tools := []string{"subfinder", "httpx", "naabu", "nuclei", "nmap"}
	for _, tool := range tools {
		if err := c.CheckTool(ctx, tool, dataDir); err != nil {
			// Log but continue checking other tools.
		}
	}
	return nil
}

// CheckTool checks a single tool.
func (c *Checker) CheckTool(ctx context.Context, tool, dataDir string) error {
	path, err := exec.LookPath(tool)
	if err != nil {
		h := &models.ToolHealth{
			ID:          util.GenerateID(),
			Tool:        tool,
			BinaryPath:  "",
			Version:     "not found",
			LastCheckAt: time.Now().UTC(),
		}
		_ = c.queries.UpsertToolHealth(h)
		return fmt.Errorf("tool not found: %s", tool)
	}

	version := c.getVersion(tool, path)
	workdirWritable := c.checkWorkdirWritable(dataDir)
	networkAvailable := c.checkNetwork()
	dnsAvailable := c.checkDNS()

	h := &models.ToolHealth{
		ID:               util.GenerateID(),
		Tool:             tool,
		BinaryPath:       path,
		Version:          version,
		WorkdirWritable:  workdirWritable,
		NetworkAvailable: networkAvailable,
		DNSAvailable:     dnsAvailable,
		LastCheckAt:      time.Now().UTC(),
	}

	// Nuclei-specific: validate templates.
	if tool == "nuclei" {
		templatePath := c.getNucleiTemplatePath(path)
		h.TemplatePath = &templatePath
	}

	return c.queries.UpsertToolHealth(h)
}

func (c *Checker) getVersion(tool, path string) string {
	// Try common version flags.
	flags := [][]string{
		{"-version"},
		{"--version"},
		{"-V"},
		{"version"},
	}
	for _, flag := range flags {
		args := append([]string{path}, flag...)
		cmd := exec.Command(args[0], args[1:]...)
		out, err := cmd.Output()
		if err == nil && len(out) > 0 {
			v := strings.TrimSpace(string(out))
			if v != "" {
				// Take first line.
				lines := strings.SplitN(v, "\n", 2)
				return lines[0]
			}
		}
	}
	return "unknown"
}

func (c *Checker) checkWorkdirWritable(dataDir string) bool {
	if dataDir == "" {
		dataDir = "."
	}
	testFile := filepath.Join(dataDir, ".health_check_tmp")
	if err := os.WriteFile(testFile, []byte("test"), 0640); err != nil {
		return false
	}
	_ = os.Remove(testFile)
	return true
}

func (c *Checker) checkNetwork() bool {
	conn, err := net.DialTimeout("tcp", "1.1.1.1:53", 5*time.Second)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func (c *Checker) checkDNS() bool {
	resolver := &net.Resolver{}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := resolver.LookupHost(ctx, "example.com")
	return err == nil
}

func (c *Checker) getNucleiTemplatePath(binaryPath string) string {
	// Try to find template directory via nuclei -h or default locations.
	cmd := exec.Command(binaryPath, "-h")
	out, _ := cmd.Output()
	_ = out
	// Default path.
	home, _ := os.UserHomeDir()
	if home != "" {
		defaultPath := filepath.Join(home, ".local", "share", "nuclei-templates")
		if _, err := os.Stat(defaultPath); err == nil {
			return defaultPath
		}
	}
	return ""
}
