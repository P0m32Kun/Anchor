package workflow

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/P0m32Kun/Anchor/internal/fingerprint"
	"github.com/P0m32Kun/Anchor/internal/parser"
	"github.com/P0m32Kun/Anchor/internal/toolregistry"
	"github.com/P0m32Kun/Anchor/internal/toolrun"
	"github.com/P0m32Kun/Anchor/internal/util"
)

func (p *Pipeline) runNmapAlive(ctx context.Context, hosts []string) ([]string, error) {
	if len(hosts) == 0 {
		return nil, nil
	}

	hostFile := filepath.Join(p.dataDir, "workdirs", p.projectID, fmt.Sprintf("nmap-%s.txt", util.GenerateID()))
	if err := os.MkdirAll(filepath.Dir(hostFile), 0750); err != nil {
		return nil, err
	}
	if err := os.WriteFile(hostFile, []byte(strings.Join(hosts, "\n")), 0640); err != nil {
		return nil, err
	}
	if abs, err := filepath.Abs(hostFile); err == nil {
		hostFile = abs
	}

	res := toolrun.Invoke(ctx, p.queries, p.runner, p.tools, toolrun.InvokeInput{
		ProjectID: p.projectID,
		RunID:     &p.runID,
		ToolID:    "nmap_alive",
		Params: toolregistry.RenderParams{
			"host_file": hostFile,
		},
	})
	if res.Err != nil {
		return nil, res.Err
	}

	alive := parser.ParseNmapAlive(bytes.NewReader(res.Stdout))
	log.Printf("[pipeline] nmap alive: input=%d alive=%d for project %s", len(hosts), len(alive), p.projectID)
	return alive, nil
}

func (p *Pipeline) runNmapServiceScan(ctx context.Context, ports []parser.PortInfo) ([]fingerprint.NmapServiceResult, error) {
	if len(ports) == 0 {
		return nil, nil
	}

	ipSet := make(map[string]bool)
	portSet := make(map[int]bool)
	for _, port := range ports {
		ipSet[port.IP] = true
		portSet[port.Port] = true
	}
	var hosts []string
	for ip := range ipSet {
		hosts = append(hosts, ip)
	}
	var portList []int
	for port := range portSet {
		portList = append(portList, port)
	}

	hostFile := filepath.Join(p.dataDir, "workdirs", p.projectID, fmt.Sprintf("nmap-sv-%s.txt", util.GenerateID()))
	if err := os.MkdirAll(filepath.Dir(hostFile), 0750); err != nil {
		return nil, err
	}
	if err := os.WriteFile(hostFile, []byte(strings.Join(hosts, "\n")), 0640); err != nil {
		return nil, err
	}
	if abs, err := filepath.Abs(hostFile); err == nil {
		hostFile = abs
	}

	portsStr := make([]string, len(portList))
	for i, p := range portList {
		portsStr[i] = fmt.Sprintf("%d", p)
	}
	res := toolrun.Invoke(ctx, p.queries, p.runner, p.tools, toolrun.InvokeInput{
		ProjectID: p.projectID,
		RunID:     &p.runID,
		ToolID:    "nmap_service",
		Params: toolregistry.RenderParams{
			"host_file":    hostFile,
			"ports":        portsStr,
			"host_timeout": p.config.NmapServiceTimeout,
		},
	})
	if res.Err != nil {
		return nil, res.Err
	}

	results := fingerprint.ParseNmapXMLOutput(string(res.Stdout))
	log.Printf("[pipeline] nmap -sV: input=%d ports on %d hosts, results=%d for project %s", len(ports), len(hosts), len(results), p.projectID)
	return results, nil
}
