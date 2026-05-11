package report

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
)

func testProject() *models.Project {
	return &models.Project{
		ID:           "proj-001",
		Name:         "Test Project",
		Organization: "TestOrg",
		Purpose:      "Security assessment",
	}
}

func testReportData() *ReportData {
	now := time.Now()
	asset := &models.Asset{
		ID:              "asset-001",
		Type:            models.AssetTypeDomain,
		Value:           "example.com",
		NormalizedValue: "example.com",
		SourceTools:     []string{"subfinder"},
	}
	ep := &models.WebEndpoint{
		ID:           "ep-001",
		AssetID:      "asset-001",
		URL:          "https://example.com",
		Scheme:       "https",
		Host:         "example.com",
		Title:        "Example Site",
		Technologies: []string{"nginx", "react"},
		SourceTool:   "httpx",
	}
	ev1 := &models.Evidence{
		ID:        "ev-001",
		FindingID: "finding-001",
		Type:      models.EvidenceRequest,
		Excerpt:   "GET / HTTP/1.1\nHost: example.com",
		CreatedBy: "nuclei",
		CreatedAt: now,
	}
	ev2 := &models.Evidence{
		ID:        "ev-002",
		FindingID: "finding-001",
		Type:      models.EvidenceResponse,
		Excerpt:   "HTTP/1.1 200 OK\nX-Powered-By: Express",
		CreatedBy: "nuclei",
		CreatedAt: now,
	}

	return &ReportData{
		Project: testProject(),
		Targets: []*models.Target{
			{ID: "t-001", ProjectID: "proj-001", Type: models.TargetTypeDomain, Value: "example.com"},
			{ID: "t-002", ProjectID: "proj-001", Type: models.TargetTypeIP, Value: "10.0.0.1"},
		},
		ScopeRules: []*models.ScopeRule{
			{ID: "sr-001", ProjectID: "proj-001", Action: models.ScopeActionInclude, Type: models.TargetTypeDomain, Value: "*.example.com", Reason: "in scope"},
			{ID: "sr-002", ProjectID: "proj-001", Action: models.ScopeActionExclude, Type: models.TargetTypeDomain, Value: "admin.example.com", Reason: "out of scope"},
		},
		Assets: []*models.Asset{asset},
		WebEndpoints: []*models.WebEndpoint{ep},
		Findings: []*ReportFinding{
			{
				Finding: &models.Finding{
					ID:            "finding-001",
					ProjectID:     "proj-001",
					AssetID:       strPtr("asset-001"),
					WebEndpointID: strPtr("ep-001"),
					SourceTool:    "nuclei",
					Title:         "Exposed .git Repository",
					Severity:      models.SeverityHigh,
					Confidence:    90,
					Priority:      85,
					Status:        models.FindingConfirmed,
					Summary:       "A .git directory was found accessible.",
					Remediation:   "Block access to .git directories via web server config.",
					DedupKey:      "nuclei:exposed-git:example.com",
				},
				Asset:       asset,
				WebEndpoint: ep,
				EvidenceList: []*models.Evidence{ev1, ev2},
			},
			{
				Finding: &models.Finding{
					ID:            "finding-002",
					ProjectID:     "proj-001",
					AssetID:       strPtr("asset-001"),
					WebEndpointID: strPtr("ep-001"),
					SourceTool:    "nuclei",
					Title:         "Missing CSP Header",
					Severity:      models.SeverityLow,
					Confidence:    75,
					Priority:      30,
					Status:        models.FindingAcceptedRisk,
					Summary:       "CSP header is not set.",
					Remediation:   "Add a Content-Security-Policy header.",
					DedupKey:      "nuclei:missing-csp:example.com",
				},
				Asset:       asset,
				WebEndpoint: ep,
				EvidenceList: []*models.Evidence{ev2},
			},
		},
		ToolVersions: []*models.ToolInvocation{
			{Tool: "subfinder", Version: "2.6.4", BinaryPath: "/usr/local/bin/subfinder"},
			{Tool: "httpx", Version: "1.3.8", BinaryPath: "/usr/local/bin/httpx"},
			{Tool: "nuclei", Version: "3.2.5", BinaryPath: "/usr/local/bin/nuclei"},
		},
		GeneratedAt: now,
	}
}

func strPtr(s string) *string { return &s }

// --- Markdown tests ---

func TestGenerateMarkdown_Normal(t *testing.T) {
	data := testReportData()
	md := GenerateMarkdown(data)

	// Verify key sections are present.
	sections := []string{
		"# 安全评估报告",
		"## 执行摘要",
		"## 扫描范围",
		"## 检测方法",
		"## 漏洞清单",
		"## 附录",
		"Test Project",
		"TestOrg",
		"Exposed .git Repository",
		"Missing CSP Header",
		"subfinder",
		"httpx",
		"nuclei",
		"Block access to .git",
		"Content-Security-Policy",
	}
	for _, section := range sections {
		if !strings.Contains(md, section) {
			t.Errorf("expected markdown to contain %q, but it was missing", section)
		}
	}
}

func TestGenerateMarkdown_NoFindings(t *testing.T) {
	data := testReportData()
	data.Findings = nil
	md := GenerateMarkdown(data)

	if !strings.Contains(md, "本次扫描未发现任何漏洞") {
		t.Error("expected '本次扫描未发现任何漏洞' message when there are no findings")
	}

	// Severity table should still be present with all-zero counts.
	for _, sev := range []string{"严重", "高危", "中危", "低危", "信息"} {
		if !strings.Contains(md, "| "+sev+" | 0 |") {
			t.Errorf("expected severity table row for %s with count 0", sev)
		}
	}
}

func TestGenerateMarkdown_NoToolInvocations(t *testing.T) {
	data := testReportData()
	data.ToolVersions = nil
	md := GenerateMarkdown(data)

	if !strings.Contains(md, "无工具调用记录") {
		t.Error("expected '无工具调用记录' message")
	}
}

func TestGenerateMarkdown_NilAssetAndEndpoint(t *testing.T) {
	data := testReportData()
	data.Findings[0].Asset = nil
	data.Findings[0].WebEndpoint = nil
	md := GenerateMarkdown(data)

	if !strings.Contains(md, "**资产** | N/A") {
		t.Error("expected 'N/A' for nil asset")
	}
	if !strings.Contains(md, "**端点** | N/A") {
		t.Error("expected 'N/A' for nil endpoint")
	}
}

func TestGenerateMarkdown_NoEvidence(t *testing.T) {
	data := testReportData()
	data.Findings[0].EvidenceList = nil
	md := GenerateMarkdown(data)

	if !strings.Contains(md, "暂无原始证据记录") {
		t.Error("expected '暂无原始证据记录' message")
	}
}

func TestGenerateMarkdown_NilData(t *testing.T) {
	md := GenerateMarkdown(nil)
	if !strings.Contains(md, "报告数据不完整") {
		t.Errorf("expected error message for nil data, got: %s", md)
	}
}

func TestGenerateMarkdown_NoRemediation(t *testing.T) {
	data := testReportData()
	data.Findings[0].Finding.Remediation = ""
	md := GenerateMarkdown(data)

	if !strings.Contains(md, "暂无修复建议") {
		t.Error("expected '暂无修复建议' for empty remediation")
	}
}

func TestGenerateMarkdown_NoSummary(t *testing.T) {
	data := testReportData()
	data.Findings[0].Finding.Summary = ""
	md := GenerateMarkdown(data)

	if !strings.Contains(md, "未提供描述") {
		t.Error("expected '未提供描述' for empty summary")
	}
}

// --- JSON tests ---

func TestGenerateJSON_Normal(t *testing.T) {
	data := testReportData()
	raw, err := GenerateJSON(data)
	if err != nil {
		t.Fatalf("GenerateJSON failed: %v", err)
	}

	// Parse back to verify structure.
	var export JSONExport
	if err := json.Unmarshal(raw, &export); err != nil {
		t.Fatalf("failed to parse generated JSON: %v", err)
	}

	// Verify top-level fields.
	if export.Meta.Tool != "anchor" {
		t.Errorf("meta.tool = %q, want 'anchor'", export.Meta.Tool)
	}
	if export.Meta.Version != "0.1.0" {
		t.Errorf("meta.version = %q, want '0.1.0'", export.Meta.Version)
	}
	if export.Project == nil {
		t.Fatal("project is nil")
	}
	if export.Project.Name != "Test Project" {
		t.Errorf("project.name = %q, want 'Test Project'", export.Project.Name)
	}

	// Verify targets.
	if len(export.Targets) != 2 {
		t.Errorf("targets count = %d, want 2", len(export.Targets))
	}

	// Verify scope rules.
	if len(export.ScopeRules) != 2 {
		t.Errorf("scope_rules count = %d, want 2", len(export.ScopeRules))
	}

	// Verify findings.
	if len(export.Findings) != 2 {
		t.Errorf("findings count = %d, want 2", len(export.Findings))
	}

	// Verify asset data in first finding.
	f1 := export.Findings[0]
	if f1.Asset == nil {
		t.Error("finding.asset is nil, expected asset data")
	} else if f1.Asset.Value != "example.com" {
		t.Errorf("finding.asset.value = %q, want 'example.com'", f1.Asset.Value)
	}

	// Verify web endpoint data.
	if f1.WebEndpoint == nil {
		t.Error("finding.web_endpoint is nil")
	} else if f1.WebEndpoint.URL != "https://example.com" {
		t.Errorf("finding.web_endpoint.url = %q, want 'https://example.com'", f1.WebEndpoint.URL)
	}

	// Verify evidence.
	if len(f1.Evidence) != 2 {
		t.Errorf("finding evidence count = %d, want 2", len(f1.Evidence))
	}

	// Verify accepted risk finding.
	f2 := export.Findings[1]
	if f2.Finding.Status != string(models.FindingAcceptedRisk) {
		t.Errorf("second finding status = %q, want 'accepted_risk'", f2.Finding.Status)
	}
}

func TestGenerateJSON_NilData(t *testing.T) {
	raw, err := GenerateJSON(nil)
	if err != nil {
		t.Fatalf("GenerateJSON(nil) failed: %v", err)
	}
	if !strings.Contains(string(raw), "error") {
		t.Error("expected error in JSON output for nil data")
	}
}

func TestGenerateJSON_EmptyData(t *testing.T) {
	data := &ReportData{
		Project:     testProject(),
		GeneratedAt: time.Now(),
	}
	raw, err := GenerateJSON(data)
	if err != nil {
		t.Fatalf("GenerateJSON(empty) failed: %v", err)
	}

	var export JSONExport
	if err := json.Unmarshal(raw, &export); err != nil {
		t.Fatalf("failed to parse generated JSON: %v", err)
	}

	if export.Project == nil {
		t.Error("project should not be nil even for empty data")
	}
	if export.Targets != nil && len(export.Targets) > 0 {
		t.Error("targets should be empty for empty data")
	}
	if export.Findings != nil && len(export.Findings) > 0 {
		t.Error("findings should be empty for empty data")
	}
}

// --- severityCountMap tests ---

func TestSeverityCountMap(t *testing.T) {
	rf := []*ReportFinding{
		{Finding: &models.Finding{Severity: models.SeverityCritical}},
		{Finding: &models.Finding{Severity: models.SeverityHigh}},
		{Finding: &models.Finding{Severity: models.SeverityHigh}},
		{Finding: &models.Finding{Severity: models.SeverityLow}},
	}

	counts := severityCountMap(rf)

	tests := []struct {
		severity models.FindingSeverity
		want     int
	}{
		{models.SeverityCritical, 1},
		{models.SeverityHigh, 2},
		{models.SeverityMedium, 0},
		{models.SeverityLow, 1},
	}

	for _, tt := range tests {
		if counts[tt.severity] != tt.want {
			t.Errorf("severityCountMap[%s] = %d, want %d", tt.severity, counts[tt.severity], tt.want)
		}
	}
}

func TestSeverityCountMap_Empty(t *testing.T) {
	counts := severityCountMap(nil)
	if len(counts) != 0 {
		t.Errorf("expected empty map for nil input, got %d entries", len(counts))
	}
}

// --- filterByStatus tests ---

func TestFilterByStatus(t *testing.T) {
	rf := []*ReportFinding{
		{Finding: &models.Finding{Status: models.FindingConfirmed}},
		{Finding: &models.Finding{Status: models.FindingAcceptedRisk}},
		{Finding: &models.Finding{Status: models.FindingConfirmed}},
	}

	confirmed := filterByStatus(rf, models.FindingConfirmed)
	if len(confirmed) != 2 {
		t.Errorf("confirmed count = %d, want 2", len(confirmed))
	}

	accepted := filterByStatus(rf, models.FindingAcceptedRisk)
	if len(accepted) != 1 {
		t.Errorf("accepted count = %d, want 1", len(accepted))
	}
}

func TestFilterByStatus_Empty(t *testing.T) {
	result := filterByStatus(nil, models.FindingConfirmed)
	if result != nil {
		t.Error("expected nil result for nil input")
	}
}
