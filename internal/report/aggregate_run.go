package report

import (
	"context"
	"fmt"
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/models"
)

// AggregateByRun collects report data scoped to a specific pipeline run.
// It reuses the project-level data but scopes findings to the run's project.
func AggregateByRun(ctx context.Context, q *db.Queries, runID string) (*ReportData, *models.PipelineRun, error) {
	// Fetch the pipeline run.
	run, err := q.GetPipelineRun(runID)
	if err != nil {
		return nil, nil, fmt.Errorf("get pipeline run: %w", err)
	}
	if run == nil {
		return nil, nil, fmt.Errorf("pipeline run %s not found", runID)
	}

	// Fetch the project.
	project, err := q.GetProject(run.ProjectID)
	if err != nil {
		return nil, nil, fmt.Errorf("get project: %w", err)
	}
	if project == nil {
		return nil, nil, fmt.Errorf("project %s not found", run.ProjectID)
	}

	// Reuse existing Aggregate for the project-level data.
	data, err := Aggregate(ctx, q, project)
	if err != nil {
		return nil, nil, fmt.Errorf("aggregate: %w", err)
	}

	return data, run, nil
}

// AggregateByRunWithBatchEvidence is like AggregateByRun but uses batch evidence
// queries to avoid N+1 when there are many findings.
func AggregateByRunWithBatchEvidence(ctx context.Context, q *db.Queries, runID string) (*ReportData, *models.Run, error) {
	run, err := q.GetRun(runID)
	if err != nil {
		return nil, nil, fmt.Errorf("get run: %w", err)
	}
	if run == nil {
		return nil, nil, fmt.Errorf("run %s not found", runID)
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

	// Fetch report-eligible findings.
	findings, err := q.ListFindingsForReport(project.ID)
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
		data.Findings = append(data.Findings, rf)
	}

	// Fetch tool invocations.
	data.ToolVersions, err = q.ListToolInvocationsByProject(project.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("list tool invocations: %w", err)
	}

	return data, run, nil
}
