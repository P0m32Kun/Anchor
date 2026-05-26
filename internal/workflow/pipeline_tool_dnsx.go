package workflow

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/parser"
	"github.com/P0m32Kun/Anchor/internal/toolregistry"
	"github.com/P0m32Kun/Anchor/internal/toolrun"
	"github.com/P0m32Kun/Anchor/internal/util"
)

func (p *Pipeline) runDNSx(ctx context.Context, domains []string) ([]models.DNSRecord, error) {
	if len(domains) == 0 {
		return nil, nil
	}

	hostFile := filepath.Join(p.dataDir, "workdirs", p.projectID, fmt.Sprintf("dnsx-%s.txt", util.GenerateID()))
	if err := os.MkdirAll(filepath.Dir(hostFile), 0750); err != nil {
		return nil, err
	}
	if err := os.WriteFile(hostFile, []byte(strings.Join(domains, "\n")), 0640); err != nil {
		return nil, err
	}
	if abs, err := filepath.Abs(hostFile); err == nil {
		hostFile = abs
	}

	res := toolrun.Invoke(ctx, p.queries, p.runner, p.tools, toolrun.InvokeInput{
		ProjectID: p.projectID,
		RunID:     &p.runID,
		ToolID:    "dnsx",
		Params: toolregistry.RenderParams{
			"host_file":  hostFile,
			"rate_limit": p.config.DNSxRateLimit,
			"threads":    p.config.DNSxThreads,
			"timeout":    p.config.DNSxTimeout,
		},
	})
	if res.Err != nil {
		return nil, res.Err
	}

	results := parser.ParseDNSxOutput(bytes.NewReader(res.Stdout))
	var records []models.DNSRecord
	for domain, rec := range results {
		records = append(records, models.DNSRecord{
			Domain: domain,
			IPs:    parser.ExtractDNSxIPs(rec),
			CNAMEs: parser.ExtractDNSxCNAMEs(rec),
			TTL:    uint32(rec.TTL),
		})
	}
	return records, nil
}
