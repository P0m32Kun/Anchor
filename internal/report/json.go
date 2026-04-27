package report

import (
	"encoding/json"
	"time"
)

// JSONExport represents the top-level JSON export structure.
type JSONExport struct {
	Meta           JSONMeta            `json:"meta"`
	Project        *JSONProject        `json:"project"`
	Targets        []*JSONTarget       `json:"targets"`
	ScopeRules     []*JSONScopeRule    `json:"scope_rules"`
	Assets         []*JSONAsset        `json:"assets"`
	WebEndpoints   []*JSONWebEndpoint  `json:"web_endpoints"`
	Findings       []*JSONReportFinding `json:"findings"`
	ToolInvocations []*JSONToolInvocation `json:"tool_invocations"`
}

type JSONMeta struct {
	GeneratedAt time.Time `json:"generated_at"`
	Tool        string    `json:"tool"`
	Version     string    `json:"version"`
}

type JSONProject struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Organization string `json:"organization"`
	Purpose      string `json:"purpose"`
}

type JSONTarget struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type JSONScopeRule struct {
	Action string `json:"action"`
	Type   string `json:"type"`
	Value  string `json:"value"`
	Reason string `json:"reason"`
}

type JSONAsset struct {
	ID              string            `json:"id"`
	Type            string            `json:"type"`
	Value           string            `json:"value"`
	NormalizedValue string            `json:"normalized_value"`
	SourceTools     []string          `json:"source_tools"`
	Tags            map[string]string `json:"tags"`
}

type JSONWebEndpoint struct {
	ID           string   `json:"id"`
	AssetID      string   `json:"asset_id"`
	URL          string   `json:"url"`
	Scheme       string   `json:"scheme"`
	Host         string   `json:"host"`
	Port         *int     `json:"port"`
	StatusCode   *int     `json:"status_code"`
	Title        string   `json:"title"`
	Technologies []string `json:"technologies"`
	SourceTool   string   `json:"source_tool"`
}

type JSONReportFinding struct {
	Finding       JSONFinding      `json:"finding"`
	Asset         *JSONAsset       `json:"asset"`
	WebEndpoint   *JSONWebEndpoint `json:"web_endpoint"`
	Evidence      []*JSONEvidence  `json:"evidence"`
}

type JSONFinding struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Severity    string `json:"severity"`
	Confidence  int    `json:"confidence"`
	Priority    int    `json:"priority"`
	Status      string `json:"status"`
	Summary     string `json:"summary"`
	Remediation string `json:"remediation"`
	SourceTool  string `json:"source_tool"`
	DedupKey    string `json:"dedup_key"`
}

type JSONEvidence struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Excerpt   string `json:"excerpt"`
	CreatedBy string `json:"created_by"`
	CreatedAt string `json:"created_at"`
}

type JSONToolInvocation struct {
	Tool    string `json:"tool"`
	Version string `json:"version"`
}

// GenerateJSON produces a JSON export of the report data.
func GenerateJSON(data *ReportData) ([]byte, error) {
	if data == nil {
		return json.Marshal(map[string]string{"error": "report data is nil"})
	}

	export := &JSONExport{
		Meta: JSONMeta{
			GeneratedAt: data.GeneratedAt,
			Tool:        "anchor",
			Version:     "0.1.0",
		},
	}

	// Project.
	if data.Project != nil {
		export.Project = &JSONProject{
			ID:           data.Project.ID,
			Name:         data.Project.Name,
			Organization: data.Project.Organization,
			Purpose:      data.Project.Purpose,
		}
	}

	// Targets.
	for _, t := range data.Targets {
		export.Targets = append(export.Targets, &JSONTarget{
			Type:  string(t.Type),
			Value: t.Value,
		})
	}

	// Scope rules.
	for _, r := range data.ScopeRules {
		export.ScopeRules = append(export.ScopeRules, &JSONScopeRule{
			Action: string(r.Action),
			Type:   string(r.Type),
			Value:  r.Value,
			Reason: r.Reason,
		})
	}

	// Assets.
	for _, a := range data.Assets {
		export.Assets = append(export.Assets, &JSONAsset{
			ID:              a.ID,
			Type:            string(a.Type),
			Value:           a.Value,
			NormalizedValue: a.NormalizedValue,
			SourceTools:     a.SourceTools,
			Tags:            a.Tags,
		})
	}

	// Web endpoints.
	for _, ep := range data.WebEndpoints {
		export.WebEndpoints = append(export.WebEndpoints, &JSONWebEndpoint{
			ID:           ep.ID,
			AssetID:      ep.AssetID,
			URL:          ep.URL,
			Scheme:       ep.Scheme,
			Host:         ep.Host,
			Port:         ep.Port,
			StatusCode:   ep.StatusCode,
			Title:        ep.Title,
			Technologies: ep.Technologies,
			SourceTool:   ep.SourceTool,
		})
	}

	// Findings.
	for _, rf := range data.Findings {
		jrf := &JSONReportFinding{
			Finding: JSONFinding{
				ID:          rf.Finding.ID,
				Title:       rf.Finding.Title,
				Severity:    string(rf.Finding.Severity),
				Confidence:  rf.Finding.Confidence,
				Priority:    rf.Finding.Priority,
				Status:      string(rf.Finding.Status),
				Summary:     rf.Finding.Summary,
				Remediation: rf.Finding.Remediation,
				SourceTool:  rf.Finding.SourceTool,
				DedupKey:    rf.Finding.DedupKey,
			},
		}

		if rf.Asset != nil {
			jrf.Asset = &JSONAsset{
				ID:              rf.Asset.ID,
				Type:            string(rf.Asset.Type),
				Value:           rf.Asset.Value,
				NormalizedValue: rf.Asset.NormalizedValue,
				SourceTools:     rf.Asset.SourceTools,
				Tags:            rf.Asset.Tags,
			}
		}

		if rf.WebEndpoint != nil {
			jrf.WebEndpoint = &JSONWebEndpoint{
				ID:           rf.WebEndpoint.ID,
				AssetID:      rf.WebEndpoint.AssetID,
				URL:          rf.WebEndpoint.URL,
				Scheme:       rf.WebEndpoint.Scheme,
				Host:         rf.WebEndpoint.Host,
				Port:         rf.WebEndpoint.Port,
				StatusCode:   rf.WebEndpoint.StatusCode,
				Title:        rf.WebEndpoint.Title,
				Technologies: rf.WebEndpoint.Technologies,
				SourceTool:   rf.WebEndpoint.SourceTool,
			}
		}

		for _, ev := range rf.EvidenceList {
			jrf.Evidence = append(jrf.Evidence, &JSONEvidence{
				ID:        ev.ID,
				Type:      string(ev.Type),
				Excerpt:   ev.Excerpt,
				CreatedBy: ev.CreatedBy,
				CreatedAt: ev.CreatedAt.Format(time.RFC3339),
			})
		}

		export.Findings = append(export.Findings, jrf)
	}

	// Tool invocations.
	toolSet := make(map[string]string) // tool -> version, deduplicate
	for _, inv := range data.ToolVersions {
		if _, exists := toolSet[inv.Tool]; !exists || inv.Version != "" {
			toolSet[inv.Tool] = inv.Version
		}
	}
	for tool, version := range toolSet {
		export.ToolInvocations = append(export.ToolInvocations, &JSONToolInvocation{
			Tool:    tool,
			Version: version,
		})
	}

	return json.MarshalIndent(export, "", "  ")
}
