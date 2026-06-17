package report

import (
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

	rf1 := &ReportFinding{
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
		Asset:        asset,
		WebEndpoint:  ep,
		EvidenceList: []*models.Evidence{ev1, ev2},
	}
	rf2 := &ReportFinding{
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
		Asset:        asset,
		WebEndpoint:  ep,
		EvidenceList: []*models.Evidence{ev2},
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
		Assets:       []*models.Asset{asset},
		WebEndpoints: []*models.WebEndpoint{ep},
		Findings:     []*ReportFinding{rf1, rf2},
		ToolVersions: []*models.ToolInvocation{
			{Tool: "subfinder", Version: "2.6.4", BinaryPath: "/usr/local/bin/subfinder"},
			{Tool: "httpx", Version: "1.3.8", BinaryPath: "/usr/local/bin/httpx"},
			{Tool: "nuclei", Version: "3.2.5", BinaryPath: "/usr/local/bin/nuclei"},
		},
		GeneratedAt: now,
		Sections: []*ReportSection{
			{Template: nil, Severity: models.SeverityHigh, Findings: []*ReportFinding{rf1}},
			{Template: nil, Severity: models.SeverityLow, Findings: []*ReportFinding{rf2}},
		},
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
		"## 一、风险概览",
		"## 二、扫描范围",
		"## 三、漏洞详情",
		"## 四、附录：检测工具",
		"Test Project",
		"TestOrg",
		"Exposed .git Repository",
		"Missing CSP Header",
		"subfinder",
		"httpx",
		"nuclei",
		"Block access to .git",
		"Content-Security-Policy",
		"**漏洞描述**",
		"**受影响资产**",
		"**修复建议**",
		"🟠 高危",
		"🔵 低危",
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
	data.Sections = nil
	md := GenerateMarkdown(data)

	if !strings.Contains(md, "本次扫描未发现需要报告的漏洞") {
		t.Error("expected '本次扫描未发现需要报告的漏洞' message when there are no findings")
	}

	// Severity bar should still show all five buckets with zero counts.
	expected := []string{
		"🔴 严重 **0**",
		"🟠 高危 **0**",
		"🟡 中危 **0**",
		"🔵 低危 **0**",
		"⚪ 信息 **0**",
	}
	for _, e := range expected {
		if !strings.Contains(md, e) {
			t.Errorf("expected severity bar entry %q, got missing", e)
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
	data.Sections[0].Findings[0].Asset = nil
	data.Sections[0].Findings[0].WebEndpoint = nil
	md := GenerateMarkdown(data)

	if !strings.Contains(md, "—") {
		t.Error("expected '—' placeholder when both asset and endpoint are nil")
	}
}

func TestGenerateMarkdown_NoEvidenceSection(t *testing.T) {
	// Human-friendly format renders only useful evidence (screenshots, notes, files).
	// Raw HTTP request/response evidence is excluded.
	// The test data contains only request/response evidence — so no evidence section appears.
	data := testReportData()
	md := GenerateMarkdown(data)

	if strings.Contains(md, "GET / HTTP/1.1") || strings.Contains(md, "X-Powered-By: Express") {
		t.Error("raw evidence excerpts should not appear in the human-friendly report")
	}
}

func TestGenerateMarkdown_WithScreenshotEvidence(t *testing.T) {
	data := testReportData()
	now := time.Now()
	screenshotEv := &models.Evidence{
		ID:        "ev-003",
		FindingID: "finding-001",
		Type:      models.EvidenceScreenshot,
		Excerpt:   "截图: https://example.com/login (1920x1080)",
		CreatedBy: "screenshot_bot",
		CreatedAt: now,
	}
	data.Findings[0].EvidenceList = append(data.Findings[0].EvidenceList, screenshotEv)
	data.Sections[0].Findings[0].EvidenceList = append(data.Sections[0].Findings[0].EvidenceList, screenshotEv)
	md := GenerateMarkdown(data)

	if !strings.Contains(md, "**证据**") {
		t.Error("expected evidence section header when screenshot evidence exists")
	}
	if !strings.Contains(md, "截图: https://example.com/login (1920x1080)") {
		t.Error("expected screenshot evidence excerpt in markdown")
	}
	if !strings.Contains(md, "📷") {
		t.Error("expected screenshot emoji marker")
	}
	// Raw request/response evidence should still be absent.
	if strings.Contains(md, "GET / HTTP/1.1") {
		t.Error("raw request evidence should not appear")
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
	data.Sections[0].Findings[0].Finding.Remediation = ""
	md := GenerateMarkdown(data)

	if !strings.Contains(md, "暂无修复建议") {
		t.Error("expected '暂无修复建议' for empty remediation")
	}
}

func TestGenerateMarkdown_NoSummary(t *testing.T) {
	data := testReportData()
	data.Findings[0].Finding.Summary = ""
	data.Sections[0].Findings[0].Finding.Summary = ""
	md := GenerateMarkdown(data)

	if !strings.Contains(md, "暂无描述") {
		t.Error("expected '暂无描述' for empty summary")
	}
}

// --- Template override tests ---

func TestGenerateMarkdown_TemplateOverridesAllFields(t *testing.T) {
	data := testReportData()
	tmpl := &models.FindingTemplate{
		Title:       "已套用：暴露的 Git 目录",
		Severity:    "critical",
		Summary:     "模板提供的中文描述。",
		Remediation: "模板提供的修复建议。",
		Enabled:     true,
	}
	data.Findings[0].Template = tmpl
	// Rebuild section with template
	data.Sections[0] = &ReportSection{
		Template: tmpl,
		Severity: models.SeverityCritical,
		Findings: []*ReportFinding{data.Findings[0]},
	}
	md := GenerateMarkdown(data)

	for _, want := range []string{
		"已套用：暴露的 Git 目录",
		"模板提供的中文描述",
		"模板提供的修复建议",
		"🔴 严重",
		"套用词条",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("expected template-supplied content %q in markdown, got missing", want)
		}
	}
	// Original finding values should not appear (they were overridden).
	if strings.Contains(md, "Exposed .git Repository") {
		t.Error("finding title should be overridden by template")
	}
	if strings.Contains(md, "A .git directory was found accessible.") {
		t.Error("finding summary should be overridden by template")
	}
}

func TestGenerateMarkdown_TemplateEmptyFieldsFallBackToFinding(t *testing.T) {
	data := testReportData()
	// Template only fills summary; title/severity/remediation should fall back.
	tmpl := &models.FindingTemplate{
		Summary: "仅描述被模板覆盖。",
		Enabled: true,
	}
	data.Findings[0].Template = tmpl
	// Rebuild section with template
	data.Sections[0] = &ReportSection{
		Template: tmpl,
		Severity: models.SeverityHigh,
		Findings: []*ReportFinding{data.Findings[0]},
	}
	md := GenerateMarkdown(data)

	if !strings.Contains(md, "仅描述被模板覆盖") {
		t.Error("expected template summary to apply")
	}
	if !strings.Contains(md, "Exposed .git Repository") {
		t.Error("expected finding title to remain when template title empty")
	}
	if !strings.Contains(md, "Block access to .git") {
		t.Error("expected finding remediation to remain when template remediation empty")
	}
}

func TestGenerateMarkdown_NoTemplateUsesFinding(t *testing.T) {
	data := testReportData()
	// No template set on either finding.
	md := GenerateMarkdown(data)
	if strings.Contains(md, "套用词条") {
		t.Error("template footer marker should not appear when no template matched")
	}
	if !strings.Contains(md, "Exposed .git Repository") {
		t.Error("finding title should render as-is when no template")
	}
	// Unmatched should show the hint
	if !strings.Contains(md, "未套词条") {
		t.Error("expected '未套词条' marker for unmatched findings")
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

// --- Pipe escaping regression (20260427-markdown-pipe-corruption) ---

func TestGenerateMarkdown_PipeInFindingTitle(t *testing.T) {
	data := testReportData()
	data.Findings[0].Finding.Title = "SQL Injection | Union Based"
	data.Findings[0].Asset = &models.Asset{
		ID:    "asset-001",
		Type:  models.AssetTypeDomain,
		Value: "example.com",
	}
	data.Findings[0].WebEndpoint = &models.WebEndpoint{
		ID:  "ep-001",
		URL: "https://example.com/search?q=test|123",
	}
	// Sync Sections
	data.Sections[0].Findings[0].Finding.Title = "SQL Injection | Union Based"
	data.Sections[0].Findings[0].Asset = data.Findings[0].Asset
	data.Sections[0].Findings[0].WebEndpoint = data.Findings[0].WebEndpoint
	md := GenerateMarkdown(data)

	// Title with | should render correctly in heading (not break markdown).
	if !strings.Contains(md, "SQL Injection \\| Union Based") {
		t.Errorf("pipe in title should be escaped, got:\n%s", md)
	}
	// URL with | should be escaped in table cell.
	if !strings.Contains(md, "test\\|123") {
		t.Errorf("pipe in URL should be escaped in table, got:\n%s", md)
	}
}

func TestGenerateMarkdown_PipeInAssetValue(t *testing.T) {
	data := testReportData()
	data.Findings[0].Asset = &models.Asset{
		ID:    "asset-001",
		Type:  models.AssetTypeDomain,
		Value: "a|b.example.com",
	}
	data.Findings[0].WebEndpoint = nil
	// Sync Sections
	data.Sections[0].Findings[0].Asset = data.Findings[0].Asset
	data.Sections[0].Findings[0].WebEndpoint = nil
	md := GenerateMarkdown(data)

	// Asset value with | in the table should be escaped.
	if !strings.Contains(md, "a\\|b.example.com") {
		t.Errorf("pipe in asset value should be escaped in table, got:\n%s", md)
	}
	// Table header should still be intact.
	if !strings.Contains(md, "| 资产 | 端口 | 访问地址 | 工具规则 |") {
		t.Error("table header should be intact")
	}
}

func TestGenerateMarkdown_PipeInMultipleFields(t *testing.T) {
	data := testReportData()
	data.Findings[0].Finding.Title = "Vuln A | B"
	data.Findings[0].Asset = &models.Asset{
		ID:    "asset-001",
		Type:  models.AssetTypeIP,
		Value: "10.0.0.1",
	}
	data.Findings[0].WebEndpoint = &models.WebEndpoint{
		ID:   "ep-001",
		URL:  "http://10.0.0.1/path?a=1|2",
		Port: intPtr(8080),
	}
	// Sync Sections
	data.Sections[0].Findings[0].Finding.Title = "Vuln A | B"
	data.Sections[0].Findings[0].Asset = data.Findings[0].Asset
	data.Sections[0].Findings[0].WebEndpoint = data.Findings[0].WebEndpoint
	md := GenerateMarkdown(data)

	// Both title and URL should have pipes escaped.
	if !strings.Contains(md, "Vuln A \\| B") {
		t.Errorf("pipe in title should be escaped")
	}
	if !strings.Contains(md, "a=1\\|2") {
		t.Errorf("pipe in URL should be escaped")
	}
}

func intPtr(v int) *int { return &v }

// --- escapeMDTable unit tests ---

func TestEscapeMDTable(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"no change", "hello", "hello"},
		{"single pipe", "a|b", `a\|b`},
		{"multiple pipes", "a|b|c", `a\|b\|c`},
		{"newline", "line1\nline2", "line1 line2"},
		{"pipe and newline", "a|b\nc", `a\|b c`},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := escapeMDTable(tt.input)
			if got != tt.want {
				t.Errorf("escapeMDTable(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
