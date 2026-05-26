package workflow

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/P0m32Kun/Anchor/internal/fingerprint"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/nuclei"
	"github.com/P0m32Kun/Anchor/internal/toolregistry"
	"github.com/P0m32Kun/Anchor/internal/toolrun"
	"github.com/P0m32Kun/Anchor/internal/util"
)

func (p *Pipeline) runNucleiWeb(ctx context.Context, endpoints []*models.WebEndpoint) error {
	groups := nuclei.GroupEndpointsByTags(endpoints)
	if len(groups) == 0 {
		if p.config.NucleiRequireFingerprint && len(endpoints) > 0 {
			log.Printf("[pipeline] nuclei web: skipped %d endpoints (no fingerprint, nuclei_require_fingerprint=true)", len(endpoints))
		}
		return nil
	}

	urlToEndpoint := make(map[string]*models.WebEndpoint)
	for _, ep := range endpoints {
		urlToEndpoint[ep.URL] = ep
	}

	scanDepth := p.config.NucleiScanDepth
	useTags := scanDepth == "tags" || scanDepth == "both"
	useWf := scanDepth == "workflow" || scanDepth == "both"

	wfPaths := p.customWorkflowPaths()

	for tagKey, urls := range groups {
		tags := strings.Split(tagKey, ",")

		if useTags {
			scanURLs := dedupHTTPTargetsByOrigin(dedupStrings(urls))
			targetFile := filepath.Join(p.dataDir, "workdirs", p.projectID, fmt.Sprintf("nuclei-%s.txt", util.GenerateID()))
			if err := os.WriteFile(targetFile, []byte(strings.Join(scanURLs, "\n")), 0640); err != nil {
				continue
			}
			if abs, err := filepath.Abs(targetFile); err == nil {
				targetFile = abs
			}
			res := toolrun.Invoke(ctx, p.queries, p.runner, p.tools, toolrun.InvokeInput{
				ProjectID: p.projectID,
				RunID:     &p.runID,
				ToolID:    "nuclei",
				Params: toolregistry.RenderParams{
					"target_file":        targetFile,
					"profile":            "deep",
					"tags":               tags,
					"scan_depth":         scanDepth,
					"concurrency":        p.config.NucleiConcurrency,
					"rate_limit":         p.config.NucleiRateLimit,
					"rate_limit_per_min": p.config.NucleiRateLimitPerMinute,
				},
			})
			if res.Err != nil {
				log.Printf("nuclei tags task for %s: %v", tagKey, res.Err)
			} else {
				p.saveNucleiFindings(res.Stdout, urlToEndpoint, nil)
			}
		}

		if useWf {
			for _, tag := range tags {
				for _, wfPath := range wfPaths {
					wfFile := filepath.Join(wfPath, tag+".yaml")
					scanURLs := dedupHTTPTargetsByOrigin(dedupStrings(urls))
					targetFile := filepath.Join(p.dataDir, "workdirs", p.projectID, fmt.Sprintf("nuclei-%s.txt", util.GenerateID()))
					if err := os.WriteFile(targetFile, []byte(strings.Join(scanURLs, "\n")), 0640); err != nil {
						continue
					}
					if abs, err := filepath.Abs(targetFile); err == nil {
						targetFile = abs
					}
					res := toolrun.Invoke(ctx, p.queries, p.runner, p.tools, toolrun.InvokeInput{
						ProjectID: p.projectID,
						RunID:     &p.runID,
						ToolID:    "nuclei",
						Params: toolregistry.RenderParams{
							"target_file":        targetFile,
							"profile":            "deep",
							"concurrency":        p.config.NucleiConcurrency,
							"rate_limit":         p.config.NucleiRateLimit,
							"rate_limit_per_min": p.config.NucleiRateLimitPerMinute,
							"template_path":      wfFile,
						},
					})
					if res.Err != nil {
						log.Printf("nuclei wf task %s for tag %s: %v", wfFile, tag, res.Err)
					} else {
						p.saveNucleiFindings(res.Stdout, urlToEndpoint, nil)
					}
				}
			}
		}
	}

	return nil
}

func (p *Pipeline) runNucleiNonWeb(ctx context.Context, results []fingerprint.NmapServiceResult) error {
	groups := make(map[string][]string)
	for _, r := range results {
		tags := nuclei.MapServiceToTags(r.Service)
		for _, tag := range tags {
			target := fmt.Sprintf("%s:%d", r.IP, r.Port)
			groups[tag] = append(groups[tag], target)
		}
	}

	if len(groups) == 0 {
		return nil
	}

	scanDepth := p.config.NucleiScanDepth
	useTags := scanDepth == "tags" || scanDepth == "both"
	useWf := scanDepth == "workflow" || scanDepth == "both"

	wfPaths := p.customWorkflowPaths()

	for tag, targets := range groups {
		if useTags {
			scanTargets := dedupHTTPTargetsByOrigin(dedupStrings(targets))
			targetFile := filepath.Join(p.dataDir, "workdirs", p.projectID, fmt.Sprintf("nuclei-%s.txt", util.GenerateID()))
			if err := os.WriteFile(targetFile, []byte(strings.Join(scanTargets, "\n")), 0640); err != nil {
				continue
			}
			if abs, err := filepath.Abs(targetFile); err == nil {
				targetFile = abs
			}
			res := toolrun.Invoke(ctx, p.queries, p.runner, p.tools, toolrun.InvokeInput{
				ProjectID: p.projectID,
				RunID:     &p.runID,
				ToolID:    "nuclei",
				Params: toolregistry.RenderParams{
					"target_file":        targetFile,
					"profile":            "deep",
					"tags":               []string{tag},
					"scan_depth":         scanDepth,
					"concurrency":        p.config.NucleiConcurrency,
					"rate_limit":         p.config.NucleiRateLimit,
					"rate_limit_per_min": p.config.NucleiRateLimitPerMinute,
				},
			})
			if res.Err != nil {
				log.Printf("nuclei tags task for %s: %v", tag, res.Err)
			} else {
				p.saveNucleiFindings(res.Stdout, nil, nil)
			}
		}

		if useWf {
			for _, wfPath := range wfPaths {
				wfFile := filepath.Join(wfPath, tag+".yaml")
				scanTargets := dedupHTTPTargetsByOrigin(dedupStrings(targets))
				targetFile := filepath.Join(p.dataDir, "workdirs", p.projectID, fmt.Sprintf("nuclei-%s.txt", util.GenerateID()))
				if err := os.WriteFile(targetFile, []byte(strings.Join(scanTargets, "\n")), 0640); err != nil {
					continue
				}
				if abs, err := filepath.Abs(targetFile); err == nil {
					targetFile = abs
				}
				res := toolrun.Invoke(ctx, p.queries, p.runner, p.tools, toolrun.InvokeInput{
					ProjectID: p.projectID,
					RunID:     &p.runID,
					ToolID:    "nuclei",
					Params: toolregistry.RenderParams{
						"target_file":        targetFile,
						"profile":            "deep",
						"concurrency":        p.config.NucleiConcurrency,
						"rate_limit":         p.config.NucleiRateLimit,
						"rate_limit_per_min": p.config.NucleiRateLimitPerMinute,
						"template_path":      wfFile,
					},
				})
				if res.Err != nil {
					log.Printf("nuclei wf task %s for tag %s: %v", wfFile, tag, res.Err)
				} else {
					p.saveNucleiFindings(res.Stdout, nil, nil)
				}
			}
		}
	}

	return nil
}

func (p *Pipeline) customWorkflowPaths() []string {
	sources, err := p.queries.ListNucleiCustomSources()
	if err != nil {
		log.Printf("[pipeline] list nuclei custom sources: %v", err)
		return nil
	}
	var paths []string
	for _, src := range sources {
		if !src.Enabled || src.InstallPath == "" {
			continue
		}
		paths = append(paths, filepath.Join("/root", "nuclei-templates", src.InstallPath, "workflows"))
	}
	return paths
}
