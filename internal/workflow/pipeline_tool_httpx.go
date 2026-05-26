package workflow

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/parser"
	"github.com/P0m32Kun/Anchor/internal/toolregistry"
	"github.com/P0m32Kun/Anchor/internal/toolrun"
	"github.com/P0m32Kun/Anchor/internal/util"
)

func (p *Pipeline) runHttpx(ctx context.Context, hosts []string) ([]*models.WebEndpoint, error) {
	if len(hosts) == 0 {
		return nil, nil
	}

	hostFile := filepath.Join(p.dataDir, "workdirs", p.projectID, fmt.Sprintf("httpx-%s.txt", util.GenerateID()))
	if err := os.MkdirAll(filepath.Dir(hostFile), 0750); err != nil {
		return nil, err
	}
	if err := os.WriteFile(hostFile, []byte(strings.Join(hosts, "\n")), 0640); err != nil {
		return nil, err
	}
	if abs, err := filepath.Abs(hostFile); err == nil {
		hostFile = abs
	}

	customFpFile, err := p.prepareHttpxFingerprints()
	if err != nil {
		log.Printf("[pipeline] prepare httpx fingerprints: %v", err)
	}

	res := toolrun.Invoke(ctx, p.queries, p.runner, p.tools, toolrun.InvokeInput{
		ProjectID: p.projectID,
		RunID:     &p.runID,
		ToolID:    "httpx",
		Params: toolregistry.RenderParams{
			"host_file":      hostFile,
			"rate_limit":     p.config.HttpxRateLimit,
			"threads":        p.config.HttpxThreads,
			"custom_fp_file": customFpFile,
		},
	})
	if res.Err != nil {
		return nil, res.Err
	}

	if customFpFile != "" {
		defer os.Remove(customFpFile)
	}

	endpoints := parser.ParseHttpxOutput(bytes.NewReader(res.Stdout))
	var saved []*models.WebEndpoint
	for _, ep := range endpoints {
		host := ep.Host
		if host == "" {
			continue
		}
		assetType := "domain"
		if net.ParseIP(host) != nil {
			assetType = "ip"
		}
		hostAsset, _, err := p.merger.MergeOrCreateAsset(p.projectID, assetType, host, "httpx")
		if err != nil {
			log.Printf("[pipeline] merge/create asset %s: %v", host, err)
			continue
		}
		we, _, err := p.merger.CreateWebEndpointIfNotExists(
			p.projectID, hostAsset.ID, ep.URL, ep.Scheme, ep.Host,
			ep.Port, ep.Path, ep.Title, ep.StatusCode, ep.Technologies, "httpx",
		)
		if err != nil {
			log.Printf("[pipeline] save web endpoint %s: %v", ep.URL, err)
			continue
		}
		if we != nil {
			saved = append(saved, we)
		}
	}
	return saved, nil
}

func (p *Pipeline) prepareHttpxFingerprints() (customFpFile string, err error) {
	fingerprints, err := p.queries.ListEnabledHttpxFingerprints("")
	if err != nil {
		log.Printf("[pipeline] list enabled fingerprints: %v", err)
		return "", err
	}
	if len(fingerprints) == 0 {
		return "", nil
	}

	customFpFile, err = p.mergeFingerprintFiles(fingerprints)
	if err != nil {
		log.Printf("[pipeline] merge fingerprint files: %v", err)
		return "", err
	}

	return customFpFile, nil
}

func (p *Pipeline) mergeFingerprintFiles(fingerprints []*models.HttpxFingerprint) (string, error) {
	workDir := filepath.Join(p.dataDir, "workdirs", p.projectID)
	if err := os.MkdirAll(workDir, 0750); err != nil {
		return "", err
	}

	tempFile := filepath.Join(workDir, fmt.Sprintf("httpx-cff-%s.tmp", util.GenerateID()))
	var mergedContent []byte

	for _, fp := range fingerprints {
		content, err := os.ReadFile(fp.FilePath)
		if err != nil {
			log.Printf("[pipeline] read fingerprint file %s: %v", fp.FilePath, err)
			continue
		}
		if len(mergedContent) > 0 {
			mergedContent = append(mergedContent, '\n')
		}
		mergedContent = append(mergedContent, content...)
	}

	if len(mergedContent) == 0 {
		return "", fmt.Errorf("no valid fingerprint content")
	}

	if err := os.WriteFile(tempFile, mergedContent, 0640); err != nil {
		return "", err
	}

	abs, err := filepath.Abs(tempFile)
	if err != nil {
		return tempFile, nil
	}
	return abs, nil
}
