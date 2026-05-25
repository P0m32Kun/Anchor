package workflow

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/P0m32Kun/Anchor/internal/util"
	"github.com/P0m32Kun/Anchor/internal/worker"
)

// runKatana runs the Katana web crawler against a list of known web endpoints
// to discover additional URLs. It returns all discovered unique URLs.
func (p *Pipeline) runKatana(ctx context.Context, urls []string) ([]string, error) {
	if !p.config.EnableKatana || len(urls) == 0 {
		return nil, nil
	}
	p.setStage(StageCrawl)

	// Write target URLs to a temp file for Katana's -list mode.
	listFile := filepath.Join(p.dataDir, "workdirs", p.projectID, fmt.Sprintf("katana-%s.txt", util.GenerateID()))
	var buf bytes.Buffer
	for _, u := range urls {
		buf.WriteString(u + "\n")
	}
	if err := os.WriteFile(listFile, buf.Bytes(), 0644); err != nil {
		p.failStage(StageCrawl, fmt.Sprintf("write katana list: %v", err))
		return nil, err
	}

	args := worker.BuildKatanaCommand(listFile, p.config.KatanaMaxDepth, p.config.KatanaRateLimit)
	task, stdout, err := p.createAndRunTask(ctx, "katana", args)
	if err != nil {
		log.Printf("[pipeline] katana: %v", err)
		p.failStage(StageCrawl, err.Error())
		return nil, err
	}
	_ = task

	// Parse results — Katana emits one JSON object per line.
	var discovered []string
	scanner := bufio.NewScanner(bytes.NewReader(stdout))
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			discovered = append(discovered, line)
		}
	}
	discovered = dedupStrings(discovered)
	log.Printf("[pipeline] katana: discovered %d URLs from %d seeds", len(discovered), len(urls))
	p.completeStage(StageCrawl)
	return discovered, nil
}
