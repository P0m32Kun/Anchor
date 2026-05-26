package workflow

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/P0m32Kun/Anchor/internal/parser"
	"github.com/P0m32Kun/Anchor/internal/toolregistry"
	"github.com/P0m32Kun/Anchor/internal/toolrun"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// runKatana runs Katana against httpx seed URLs to discover linked pages and JS endpoints.
// Discovered URLs (including .js assets) are fed to httpx_2 in runPostPhase.
func (p *Pipeline) runKatana(ctx context.Context, urls []string) ([]string, error) {
	if !p.config.EnableKatana || len(urls) == 0 {
		return nil, nil
	}
	p.setStage(StageCrawl)

	listFile := filepath.Join(p.dataDir, "workdirs", p.projectID, fmt.Sprintf("katana-%s.txt", util.GenerateID()))
	var buf bytes.Buffer
	for _, u := range urls {
		buf.WriteString(u + "\n")
	}
	if err := os.WriteFile(listFile, buf.Bytes(), 0644); err != nil {
		p.failStage(StageCrawl, fmt.Sprintf("write katana list: %v", err))
		return nil, err
	}

	timeout := p.config.KatanaTimeout
	if timeout <= 0 {
		timeout = 10
	}
	res := toolrun.Invoke(ctx, p.queries, p.runner, p.tools, toolrun.InvokeInput{
		ProjectID: p.projectID,
		RunID:     &p.runID,
		ToolID:    "katana",
		Params: toolregistry.RenderParams{
			"list_file":  listFile,
			"depth":      p.config.KatanaMaxDepth,
			"rate_limit": p.config.KatanaRateLimit,
			"timeout":    timeout,
		},
	})
	if res.Err != nil {
		log.Printf("[pipeline] katana: %v", res.Err)
		p.failStage(StageCrawl, res.Err.Error())
		return nil, res.Err
	}

	discovered, parseErrs := parser.ParseKatanaJSONL(bytes.NewReader(res.Stdout))
	for _, pe := range parseErrs {
		log.Printf("[pipeline] katana parse: line %d: %s", pe.Line, pe.Message)
	}
	discovered = dedupStrings(discovered)
	log.Printf("[pipeline] katana: discovered %d URLs from %d seeds", len(discovered), len(urls))
	p.completeStage(StageCrawl)
	return discovered, nil
}
