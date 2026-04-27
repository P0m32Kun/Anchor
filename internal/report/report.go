package report

import (
	"context"
	"fmt"
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/models"
)

// ReportData aggregates all data needed to generate a report for a project.
type ReportData struct {
	Project      *models.Project
	Targets      []*models.Target
	ScopeRules   []*models.ScopeRule
	Assets       []*models.Asset
	WebEndpoints []*models.WebEndpoint
	Findings     []*ReportFinding      // confirmed + accepted_risk
	ToolVersions []*models.ToolInvocation
	GeneratedAt  time.Time
}

// ReportFinding wraps a Finding with its related Asset, WebEndpoint, and Evidence.
type ReportFinding struct {
	Finding      *models.Finding
	Asset        *models.Asset       // nullable
	WebEndpoint  *models.WebEndpoint // nullable
	EvidenceList []*models.Evidence
}

// Aggregate collects all report data from the database for a given project.
func Aggregate(ctx context.Context, q *db.Queries, project *models.Project) (*ReportData, error) {
	data := &ReportData{
		Project:     project,
		GeneratedAt: time.Now(),
	}

	// Check context cancellation before each query.
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Fetch targets.
	targets, err := q.ListTargetsByProject(project.ID)
	if err != nil {
		return nil, fmt.Errorf("list targets: %w", err)
	}
	data.Targets = targets

	// Fetch scope rules.
	scopeRules, err := q.ListScopeRulesByProject(project.ID)
	if err != nil {
		return nil, fmt.Errorf("list scope rules: %w", err)
	}
	data.ScopeRules = scopeRules

	// Fetch assets.
	assets, err := q.ListAssetsByProject(project.ID)
	if err != nil {
		return nil, fmt.Errorf("list assets: %w", err)
	}
	data.Assets = assets

	// Fetch web endpoints.
	endpoints, err := q.ListWebEndpointsByProject(project.ID)
	if err != nil {
		return nil, fmt.Errorf("list web endpoints: %w", err)
	}
	data.WebEndpoints = endpoints

	// Fetch report-eligible findings (confirmed + accepted_risk).
	findings, err := q.ListFindingsForReport(project.ID)
	if err != nil {
		return nil, fmt.Errorf("list findings: %w", err)
	}

	// Build lookup maps for batch assembly.
	assetByID := make(map[string]*models.Asset, len(assets))
	for _, a := range assets {
		assetByID[a.ID] = a
	}
	epByID := make(map[string]*models.WebEndpoint, len(endpoints))
	for _, ep := range endpoints {
		epByID[ep.ID] = ep
	}

	// Assemble ReportFinding with related entities.
	for _, f := range findings {
		rf := &ReportFinding{
			Finding: f,
		}
		if f.AssetID != nil {
			rf.Asset = assetByID[*f.AssetID]
		}
		if f.WebEndpointID != nil {
			rf.WebEndpoint = epByID[*f.WebEndpointID]
		}

		// Fetch evidence for this finding.
		evList, err := q.ListEvidenceByFinding(f.ID)
		if err != nil {
			return nil, fmt.Errorf("list evidence for finding %s: %w", f.ID, err)
		}
		rf.EvidenceList = evList

		data.Findings = append(data.Findings, rf)
	}

	// Fetch tool invocations for methodology/appendix.
	toolInvs, err := q.ListToolInvocationsByProject(project.ID)
	if err != nil {
		return nil, fmt.Errorf("list tool invocations: %w", err)
	}
	data.ToolVersions = toolInvs

	return data, nil
}
