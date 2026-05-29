package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/P0m32Kun/Anchor/internal/asset"
	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/scanengine/core"
	"github.com/P0m32Kun/Anchor/internal/toolregistry"
	"github.com/P0m32Kun/Anchor/internal/toolrun"
	"github.com/P0m32Kun/Anchor/internal/util"
	"github.com/P0m32Kun/Anchor/internal/worker"
)

// OnWorkComplete is called after a work item finishes execution.
// It returns new assets discovered and attribute updates.
type OnWorkComplete func(workItem *models.ScanWorkItem, stdout []byte) ([]*core.DiscoveryAsset, core.AssetAttrs, error)

// ToolExecutor executes individual work items via toolrun.Invoke.
type ToolExecutor struct {
	queries  *db.Queries
	runner   *worker.Runner
	tools    *toolregistry.Registry
	merger   *asset.Merger
	dataDir  string
}

// NewToolExecutor creates a new ToolExecutor.
func NewToolExecutor(queries *db.Queries, runner *worker.Runner, tools *toolregistry.Registry, merger *asset.Merger, dataDir string) *ToolExecutor {
	return &ToolExecutor{
		queries: queries,
		runner:  runner,
		tools:   tools,
		merger:  merger,
		dataDir: dataDir,
	}
}

// Execute runs a single work item and returns the result.
func (e *ToolExecutor) Execute(ctx context.Context, w *models.ScanWorkItem, params toolregistry.RenderParams) (*toolrun.InvokeResult, error) {
	toolID := actionToToolID(w.Action)
	if toolID == "" {
		return nil, fmt.Errorf("no tool mapping for action %s", w.Action)
	}

	res := toolrun.Invoke(ctx, e.queries, e.runner, e.tools, toolrun.InvokeInput{
		ProjectID: w.ProjectID,
		RunID:     &w.RunID,
		TaskID:    util.GenerateID(),
		ToolID:    toolID,
		Params:    params,
	})
	if res.Err != nil {
		return nil, res.Err
	}
	return res, nil
}

// actionToToolID maps a work action string to the tool registry ID.
func actionToToolID(action string) string {
	switch action {
	case string(core.ActionHTTPXFingerprint):
		return "httpx"
	case string(core.ActionNucleiScan):
		return "nuclei"
	case string(core.ActionPortScan):
		return "naabu"
	case string(core.ActionServiceFingerprint):
		return "nmap"
	case string(core.ActionKatanaCrawl):
		return "katana"
	case string(core.ActionFFUFBrute):
		return "ffuf"
	case string(core.ActionSubdomainEnum):
		return "subfinder"
	case string(core.ActionDNSResolve):
		return "dnsx"
	case string(core.ActionCDNCheck):
		return "cdncheck"
	default:
		return ""
	}
}

// ParseHttpxOutput parses httpx JSONL output into discovered assets and attrs.
func ParseHttpxOutput(stdout []byte, projectID string) ([]*core.DiscoveryAsset, core.AssetAttrs, error) {
	var assets []*core.DiscoveryAsset
	var attrs core.AssetAttrs

	lines := strings.Split(strings.TrimSpace(string(stdout)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var entry httpxEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue // skip malformed lines
		}
		a := &core.DiscoveryAsset{
			ID:              util.GenerateID(),
			Type:            core.AssetHTTPService,
			Value:           entry.URL,
			NormalizedValue: entry.Input,
			SourceTool:      "httpx",
		}
		assets = append(assets, a)

		// Update attrs from httpx output
		if entry.StatusCode > 0 {
			attrs.StatusCode = &entry.StatusCode
		}
		attrs.Fingerprinted = true
		if len(entry.Technologies) > 0 {
			attrs.Technologies = append(attrs.Technologies, entry.Technologies...)
		}
	}
	return assets, attrs, nil
}

// ParseNucleiOutput parses nuclei JSONL output (currently a no-op placeholder
// for finding creation — findings are handled by the existing pipeline).
func ParseNucleiOutput(stdout []byte) ([]string, error) {
	var findings []string
	lines := strings.Split(strings.TrimSpace(string(stdout)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var entry nucleiEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		if entry.TemplateID != "" {
			findings = append(findings, entry.TemplateID)
		}
	}
	return findings, nil
}

// WriteHostFile writes a list of targets to a temporary file for tool input.
func WriteHostFile(dataDir string, targets []string) (string, func(), error) {
	dir := filepath.Join(dataDir, "tmp")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", nil, err
	}
	f, err := os.CreateTemp(dir, "scanengine-*.txt")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { os.Remove(f.Name()) }
	for _, t := range targets {
		fmt.Fprintln(f, t)
	}
	if err := f.Close(); err != nil {
		cleanup()
		return "", nil, err
	}
	return f.Name(), cleanup, nil
}

// --- internal JSON structs ---

type httpxEntry struct {
	URL          string   `json:"url"`
	Input        string   `json:"input"`
	StatusCode   int      `json:"status_code"`
	Technologies []string `json:"tech"`
	Title        string   `json:"title"`
	Host         string   `json:"host"`
	Port         string   `json:"port"`
	Scheme       string   `json:"scheme"`
	Path         string   `json:"path"`
}

type nucleiEntry struct {
	TemplateID string `json:"template-id"`
	Info       struct {
		Name     string `json:"name"`
		Severity string `json:"severity"`
	} `json:"info"`
	Host string `json:"host"`
	IP   string `json:"ip"`
	Matched string `json:"matched-at"`
}
