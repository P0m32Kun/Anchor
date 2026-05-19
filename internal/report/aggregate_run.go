package report

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/models"
)

// AggregateByRun collects report data scoped to a specific pipeline run.
func AggregateByRun(ctx context.Context, q *db.Queries, runID string) (*ReportData, *models.PipelineRun, error) {
	return AggregateByRunWithBatchEvidence(ctx, q, runID)
}

// AggregateByRunWithBatchEvidence is like AggregateByRun but uses batch evidence
// queries to avoid N+1 when there are many findings.
func AggregateByRunWithBatchEvidence(ctx context.Context, q *db.Queries, runID string) (*ReportData, *models.PipelineRun, error) {
	run, err := q.GetPipelineRun(runID)
	if err != nil {
		return nil, nil, fmt.Errorf("get pipeline run: %w", err)
	}
	if run == nil {
		return nil, nil, fmt.Errorf("pipeline run %s not found", runID)
	}

	project, err := q.GetProject(run.ProjectID)
	if err != nil {
		return nil, nil, fmt.Errorf("get project: %w", err)
	}
	if project == nil {
		return nil, nil, fmt.Errorf("project %s not found", run.ProjectID)
	}

	select {
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	default:
	}

	data := &ReportData{
		Project:     project,
		GeneratedAt: time.Now(),
	}

	// Fetch targets.
	data.Targets, err = q.ListTargetsByProject(project.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("list targets: %w", err)
	}

	// Fetch scope rules.
	data.ScopeRules, err = q.ListScopeRulesByProject(project.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("list scope rules: %w", err)
	}

	// Fetch assets.
	data.Assets, err = q.ListAssetsByProject(project.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("list assets: %w", err)
	}

	// Fetch web endpoints.
	data.WebEndpoints, err = q.ListWebEndpointsByProject(project.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("list web endpoints: %w", err)
	}

	// Fetch report-eligible findings scoped to this run.
	findings, err := q.ListFindingsByRun(project.ID, runID)
	if err != nil {
		return nil, nil, fmt.Errorf("list findings: %w", err)
	}

	// Build lookup maps.
	assetByID := make(map[string]*models.Asset, len(data.Assets))
	for _, a := range data.Assets {
		assetByID[a.ID] = a
	}
	epByID := make(map[string]*models.WebEndpoint, len(data.WebEndpoints))
	for _, ep := range data.WebEndpoints {
		epByID[ep.ID] = ep
	}

	// Batch fetch evidence for all findings.
	findingIDs := make([]string, 0, len(findings))
	for _, f := range findings {
		findingIDs = append(findingIDs, f.ID)
	}
	evidenceMap, err := q.ListEvidenceByFindingIDs(findingIDs)
	if err != nil {
		return nil, nil, fmt.Errorf("list evidence batch: %w", err)
	}

	// Assemble ReportFinding with related entities.
	for _, f := range findings {
		rf := &ReportFinding{Finding: f}
		if f.AssetID != nil {
			rf.Asset = assetByID[*f.AssetID]
		}
		if f.WebEndpointID != nil {
			rf.WebEndpoint = epByID[*f.WebEndpointID]
		}
		rf.EvidenceList = evidenceMap[f.ID]
		// Match against vulnerability template knowledge base; errors are non-fatal.
		if tmpl, terr := q.GetFindingTemplateForFinding(f.SourceTool, f.SourceRuleID, f.Title); terr == nil {
			rf.Template = tmpl
		}
		data.Findings = append(data.Findings, rf)
	}

	// Fetch tool invocations.
	data.ToolVersions, err = q.ListToolInvocationsByProject(project.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("list tool invocations: %w", err)
	}

	// --- 分桶:把 findings 按词条聚合 ---
	buckets := make(map[string][]*ReportFinding)
	var unmatched []*ReportFinding
	for _, rf := range data.Findings {
		if rf.Template != nil {
			buckets[rf.Template.ID] = append(buckets[rf.Template.ID], rf)
		} else {
			unmatched = append(unmatched, rf)
		}
	}

	sections := make([]*ReportSection, 0, len(buckets)+len(unmatched))
	for _, findings := range buckets {
		if len(findings) == 0 {
			continue
		}
		t := findings[0].Template
		sev := models.FindingSeverity(t.Severity)
		if sev == "" {
			sev = findings[0].Finding.Severity
		}
		sections = append(sections, &ReportSection{Template: t, Severity: sev, Findings: findings})
	}
	for _, rf := range unmatched {
		sections = append(sections, &ReportSection{Template: nil, Severity: rf.Finding.Severity, Findings: []*ReportFinding{rf}})
	}

	// 按 severity 倒序排序;同级时命中排在未命中前
	severityRank := map[models.FindingSeverity]int{
		models.SeverityCritical: 5,
		models.SeverityHigh:     4,
		models.SeverityMedium:   3,
		models.SeverityLow:      2,
		models.SeverityInfo:     1,
	}
	sort.SliceStable(sections, func(i, j int) bool {
		if severityRank[sections[i].Severity] != severityRank[sections[j].Severity] {
			return severityRank[sections[i].Severity] > severityRank[sections[j].Severity]
		}
		// 同级:命中(Template != nil)排在未命中前面
		return (sections[i].Template != nil) && (sections[j].Template == nil)
	})

	data.Sections = sections

	return data, run, nil
}
