package workflow

import (
	"context"
	"fmt"
	"strings"

	"github.com/P0m32Kun/Anchor/internal/cdn"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/toolregistry"
	"github.com/P0m32Kun/Anchor/internal/toolrun"
)

func (p *Pipeline) runCDNCheck(ctx context.Context, ips []string) ([]string, []models.CDNResult, error) {
	if len(ips) == 0 {
		return nil, nil, fmt.Errorf("no IPs to classify (DNS produced no A/AAAA records)")
	}
	input := strings.Join(ips, ",")
	res := toolrun.Invoke(ctx, p.queries, p.runner, p.tools, toolrun.InvokeInput{
		ProjectID: p.projectID,
		RunID:     &p.runID,
		ToolID:    "cdncheck",
		Params: toolregistry.RenderParams{
			"ips": input,
		},
	})
	if res.Err != nil {
		return nil, nil, fmt.Errorf("cdncheck: %w", res.Err)
	}
	return cdn.ParseJSONLOutput(res.Stdout, ips)
}
