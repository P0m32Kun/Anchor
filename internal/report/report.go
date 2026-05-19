package report

import (
	"context"
	"fmt"
	"sort"
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

	// Sections 是按词条聚合后的章节列表,已按 severity 倒序。
	// 用于 Markdown 渲染的「三、漏洞详情」段。
	Sections []*ReportSection
}

// ReportFinding wraps a Finding with its related Asset, WebEndpoint, Evidence,
// and the matched FindingTemplate (if any). Template fields override the
// finding's title / severity / summary / remediation at render time when
// non-empty.
type ReportFinding struct {
	Finding      *models.Finding
	Asset        *models.Asset       // nullable
	WebEndpoint  *models.WebEndpoint // nullable
	EvidenceList []*models.Evidence
	Template     *models.FindingTemplate // nullable
}

// ReportSection 是「漏洞详情」章节单元。
//   - 命中词条: Template 非 nil,Findings 是同词条下的多个 finding
//   - 未命中:    Template 为 nil,Findings 切片长度恰好为 1
type ReportSection struct {
	Template *models.FindingTemplate // nil = 未命中(同位混排原始块)
	Severity models.FindingSeverity  // 用于排序,命中时 = template.severity,未命中 = finding.severity
	Findings []*ReportFinding
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

		// Match against the vulnerability template knowledge base.
		// Errors are non-fatal — templates are an enhancement, not a requirement.
		if tmpl, terr := q.GetFindingTemplateForFinding(f.SourceTool, f.SourceRuleID, f.Title); terr == nil {
			rf.Template = tmpl
		}

		data.Findings = append(data.Findings, rf)
	}

	// Fetch tool invocations for methodology/appendix.
	toolInvs, err := q.ListToolInvocationsByProject(project.ID)
	if err != nil {
		return nil, fmt.Errorf("list tool invocations: %w", err)
	}
	data.ToolVersions = toolInvs

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

	return data, nil
}
