package workflow

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/parser"
	"github.com/P0m32Kun/Anchor/internal/toolregistry"
	"github.com/P0m32Kun/Anchor/internal/toolrun"
	"github.com/P0m32Kun/Anchor/internal/util"
)

func (p *Pipeline) runSubfinder(ctx context.Context, domain string) ([]string, error) {
	target := &models.Target{Type: models.TargetTypeDomain, Value: domain}
	decision, err := p.scope.ValidateBeforeRun(ctx, p.projectID, target, "")
	if err != nil || decision.Decision == models.ScopeDeny {
		return nil, fmt.Errorf("scope denied")
	}

	providerConfigPath := ""
	if p.config.SubfinderProviderConfig != "" {
		workdir := filepath.Join(p.dataDir, "workdirs", p.projectID)
		_ = os.MkdirAll(workdir, 0750)
		tmpFile := filepath.Join(workdir, fmt.Sprintf("subfinder-provider-%s.yaml", util.GenerateID()))
		if err := os.WriteFile(tmpFile, []byte(p.config.SubfinderProviderConfig), 0640); err != nil {
			log.Printf("[pipeline] write subfinder provider config: %v", err)
		} else {
			abs, err := filepath.Abs(tmpFile)
			if err == nil {
				providerConfigPath = abs
			}
		}
	}

	res := toolrun.Invoke(ctx, p.queries, p.runner, p.tools, toolrun.InvokeInput{
		ProjectID: p.projectID,
		RunID:     &p.runID,
		ToolID:    "subfinder",
		Params: toolregistry.RenderParams{
			"domain":               domain,
			"rate_limit":           p.config.SubfinderRateLimit,
			"threads":              p.config.SubfinderThreads,
			"timeout":              p.config.SubfinderTimeout,
			"mode":                 p.config.SubfinderMode,
			"provider_config_path": providerConfigPath,
		},
	})
	if res.Err != nil {
		return nil, res.Err
	}

	subs := parser.ParseSubfinderOutput(bytes.NewReader(res.Stdout))
	return subs, nil
}
