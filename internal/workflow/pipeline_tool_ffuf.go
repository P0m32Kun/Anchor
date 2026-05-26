package workflow

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/parser"
	"github.com/P0m32Kun/Anchor/internal/toolregistry"
	"github.com/P0m32Kun/Anchor/internal/toolrun"
)

func (p *Pipeline) runFfuf(ctx context.Context, endpoint *models.WebEndpoint, dictID string) ([]string, error) {
	if !p.config.EnableFfuf || dictID == "" {
		return nil, nil
	}

	dict, err := p.queries.GetDictionary(dictID)
	if err != nil || dict == nil {
		return nil, fmt.Errorf("dictionary not found: %s", dictID)
	}
	if !dict.Enabled {
		return nil, fmt.Errorf("dictionary disabled: %s", dictID)
	}

	base := strings.TrimSuffix(endpoint.URL, "/")
	targetURL := base + "/FUZZ"

	res := toolrun.Invoke(ctx, p.queries, p.runner, p.tools, toolrun.InvokeInput{
		ProjectID: p.projectID,
		RunID:     &p.runID,
		ToolID:    "ffuf",
		Params: toolregistry.RenderParams{
			"target":     targetURL,
			"wordlist":   dict.FilePath,
			"rate_limit": p.config.FfufRateLimit,
			"timeout":    p.config.FfufTimeout,
		},
	})
	if res.Err != nil {
		return nil, res.Err
	}

	results, _ := parser.ParseFfufOutput(bytes.NewReader(res.Stdout))
	var urls []string
	for _, r := range results {
		if r.URL != "" {
			urls = append(urls, r.URL)
		}
	}
	return urls, nil
}
