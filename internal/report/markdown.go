package report

import (
	"fmt"
	"sort"
	"strings"

	"github.com/P0m32Kun/Anchor/internal/models"
)

// escapeMDTable escapes special Markdown characters in table cell values.
func escapeMDTable(v string) string {
	v = strings.ReplaceAll(v, "|", "\\|")
	v = strings.ReplaceAll(v, "\n", " ")
	return v
}

var severityLabelsCN = map[models.FindingSeverity]string{
	models.SeverityCritical: "严重",
	models.SeverityHigh:     "高危",
	models.SeverityMedium:   "中危",
	models.SeverityLow:      "低危",
	models.SeverityInfo:     "信息",
}

var severityEmoji = map[models.FindingSeverity]string{
	models.SeverityCritical: "🔴",
	models.SeverityHigh:     "🟠",
	models.SeverityMedium:   "🟡",
	models.SeverityLow:      "🔵",
	models.SeverityInfo:     "⚪",
}

var statusLabelsCN = map[models.FindingStatus]string{
	models.FindingPendingReview: "待审核",
	models.FindingConfirmed:     "已确认",
	models.FindingFalsePositive: "误报",
	models.FindingAcceptedRisk:  "已接受风险",
	models.FindingIgnored:       "已忽略",
}

func severityLabelCN(sev models.FindingSeverity) string {
	if v, ok := severityLabelsCN[sev]; ok {
		return v
	}
	return string(sev)
}

func severityEmojiOf(sev models.FindingSeverity) string {
	if v, ok := severityEmoji[sev]; ok {
		return v
	}
	return "⚪"
}

func statusLabelCN(st models.FindingStatus) string {
	if v, ok := statusLabelsCN[st]; ok {
		return v
	}
	return string(st)
}

// GenerateMarkdown produces a human-readable security assessment report in Markdown (Chinese).
// Layout per finding: emoji + severity + title → description → affected IP:Port → remediation → footer metadata.
// All edge cases (nil data, empty slices) are handled gracefully.
func GenerateMarkdown(data *ReportData) string {
	if data == nil || data.Project == nil {
		return "错误：报告数据不完整（缺少项目信息）。"
	}

	var sb strings.Builder

	// --- 1. 标题与项目信息 ---
	sb.WriteString("# 安全评估报告\n\n")
	fmt.Fprintf(&sb, "> **项目名称**：%s", escapeMDTable(data.Project.Name))
	if data.Project.Organization != "" {
		fmt.Fprintf(&sb, " · **所属组织**：%s", escapeMDTable(data.Project.Organization))
	}
	fmt.Fprintf(&sb, " · **生成时间**：%s\n\n", data.GeneratedAt.Format("2006-01-02 15:04:05"))

	// --- 2. 概览 ---
	sb.WriteString("## 一、风险概览\n\n")
	severityCounts := severityCountMap(data.Findings)
	totalFindings := len(data.Findings)

	// 横排 emoji 风险条
	sb.WriteString(formatSeverityBar(severityCounts) + "\n\n")

	if totalFindings == 0 {
		sb.WriteString("> ✅ **本次扫描未发现需要报告的漏洞。**\n\n")
	} else {
		statusCounts := statusCountMap(data.Findings)
		confirmed := statusCounts[models.FindingConfirmed]
		accepted := statusCounts[models.FindingAcceptedRisk]
		fmt.Fprintf(&sb, "本次报告共包含 **%d** 个漏洞", totalFindings)
		parts := []string{}
		if confirmed > 0 {
			parts = append(parts, fmt.Sprintf("**%d** 项已确认", confirmed))
		}
		if accepted > 0 {
			parts = append(parts, fmt.Sprintf("**%d** 项已接受风险", accepted))
		}
		if len(parts) > 0 {
			fmt.Fprintf(&sb, "，其中 %s", strings.Join(parts, " · "))
		}
		sb.WriteString("。\n\n")
	}

	// --- 3. 扫描范围 ---
	sb.WriteString("## 二、扫描范围\n\n")
	if len(data.Targets) > 0 {
		for _, t := range data.Targets {
			fmt.Fprintf(&sb, "- `%s` (%s)\n", t.Value, t.Type)
		}
	} else {
		sb.WriteString("*未定义目标。*\n")
	}
	sb.WriteString("\n")

	if len(data.ScopeRules) > 0 {
		sb.WriteString("**范围规则**\n\n")
		for _, r := range data.ScopeRules {
			action := "纳入"
			if r.Action == models.ScopeActionExclude {
				action = "排除"
			}
			fmt.Fprintf(&sb, "- [%s] `%s` (%s)", action, r.Value, r.Type)
			if r.Reason != "" {
				fmt.Fprintf(&sb, " — %s", r.Reason)
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	// --- 4. 漏洞详情(按 severity 倒序,使用 Sections 聚合渲染) ---
	sb.WriteString("## 三、漏洞详情\n\n")

	if len(data.Sections) == 0 {
		sb.WriteString("*无漏洞记录。*\n\n")
	} else {
		num := 0
		for _, section := range data.Sections {
			num++
			writeSectionCN(&sb, num, section)
		}
	}

	// --- 5. 附录 ---
	sb.WriteString("## 四、附录：检测工具\n\n")
	if len(data.ToolVersions) > 0 {
		toolMap := make(map[string]*models.ToolInvocation)
		for _, inv := range data.ToolVersions {
			toolMap[inv.Tool] = inv
		}
		toolNames := make([]string, 0, len(toolMap))
		for name := range toolMap {
			toolNames = append(toolNames, name)
		}
		sort.Strings(toolNames)

		for _, name := range toolNames {
			inv := toolMap[name]
			version := inv.Version
			if version == "" {
				version = "版本未知"
			}
			fmt.Fprintf(&sb, "- **%s** %s\n", name, version)
		}
	} else {
		sb.WriteString("*无工具调用记录。*\n")
	}
	sb.WriteString("\n")

	return sb.String()
}

// formatSeverityBar renders a single-line emoji-prefixed counter strip for the five severity levels.
func formatSeverityBar(counts map[models.FindingSeverity]int) string {
	order := []models.FindingSeverity{
		models.SeverityCritical,
		models.SeverityHigh,
		models.SeverityMedium,
		models.SeverityLow,
		models.SeverityInfo,
	}
	parts := make([]string, 0, len(order))
	for _, sev := range order {
		parts = append(parts, fmt.Sprintf("%s %s **%d**", severityEmojiOf(sev), severityLabelCN(sev), counts[sev]))
	}
	return strings.Join(parts, "　·　")
}

// writeSectionCN 渲染一个章节:命中词条合并一节,或未命中单条一节。
// 当 Template 字段非空时优先使用模板值,否则回退到 finding 自身值。
func writeSectionCN(sb *strings.Builder, num int, section *ReportSection) {
	rf0 := section.Findings[0]
	f0 := rf0.Finding

	// 标题:模板优先,空则回退
	title := f0.Title
	if section.Template != nil && strings.TrimSpace(section.Template.Title) != "" {
		title = section.Template.Title
	}

	// severity:模板优先,空则回退
	severity := f0.Severity
	if section.Template != nil && strings.TrimSpace(section.Template.Severity) != "" {
		severity = models.FindingSeverity(section.Template.Severity)
	}

	fmt.Fprintf(sb, "### %d. %s %s · %s\n\n",
		num, severityEmojiOf(severity), severityLabelCN(severity), escapeMDTable(title))

	// 漏洞描述:模板优先,空则回退
	sb.WriteString("**漏洞描述**\n\n")
	summary := f0.Summary
	if section.Template != nil && strings.TrimSpace(section.Template.Summary) != "" {
		summary = section.Template.Summary
	}
	if strings.TrimSpace(summary) != "" {
		sb.WriteString(summary)
		sb.WriteString("\n\n")
	} else {
		sb.WriteString("*暂无描述。*\n\n")
	}

	// 受影响资产
	sb.WriteString("**受影响资产**\n\n")
	writeAffectedAssetsTable(sb, section)

	// 修复建议:模板优先,空则回退
	sb.WriteString("**修复建议**\n\n")
	remediation := ""
	if section.Template != nil {
		remediation = section.Template.Remediation
	}
	if strings.TrimSpace(remediation) == "" {
		remediation = f0.Remediation
	}
	if strings.TrimSpace(remediation) != "" {
		sb.WriteString(remediation)
		sb.WriteString("\n\n")
	} else {
		if section.Template != nil {
			sb.WriteString("*暂无修复建议。*\n\n")
		} else {
			sb.WriteString("*暂无修复建议 — 该漏洞类型尚未在辞典中维护。可在「漏洞模板」页补充。*\n\n")
		}
	}

	// 底部元数据
	var meta []string
	if section.Template != nil {
		meta = append(meta, "套用词条")
	}
	meta = append(meta, fmt.Sprintf("检测来源:%s", f0.SourceTool))
	if len(section.Findings) > 1 && section.Template != nil {
		meta = append(meta, fmt.Sprintf("命中 %d 项", len(section.Findings)))
	}
	if section.Template == nil {
		meta = append(meta, "未套词条")
		if f0.SourceRuleID != "" {
			meta = append(meta, fmt.Sprintf("规则:%s", escapeMDTable(f0.SourceRuleID)))
		}
	}
	meta = append(meta, "状态:"+statusLabelCN(f0.Status))
	fmt.Fprintf(sb, "> %s\n\n", strings.Join(meta, " · "))

	sb.WriteString("---\n\n")
}

// writeAffectedAssetsTable 渲染受影响资产表格。
// 命中词条:多行表格;未命中:单行表格。
func writeAffectedAssetsTable(sb *strings.Builder, section *ReportSection) {
	sb.WriteString("| 资产 | 端口 | 访问地址 | 工具规则 |\n")
	sb.WriteString("|---|---|---|---|\n")

	for _, rf := range section.Findings {
		assetVal := "—"
		if rf.Asset != nil {
			assetVal = escapeMDTable(rf.Asset.Value)
		}

		portVal := "—"
		urlVal := "—"
		if rf.WebEndpoint != nil {
			if rf.WebEndpoint.Port != nil {
				portVal = fmt.Sprintf("%d", *rf.WebEndpoint.Port)
			} else if rf.WebEndpoint.Scheme == "https" {
				portVal = "443"
			} else if rf.WebEndpoint.Scheme == "http" {
				portVal = "80"
			}
			if rf.WebEndpoint.URL != "" {
				urlVal = escapeMDTable(rf.WebEndpoint.URL)
			}
		}

		ruleVal := "—"
		if rf.Finding.SourceRuleID != "" {
			ruleVal = escapeMDTable(rf.Finding.SourceRuleID)
		}

		fmt.Fprintf(sb, "| %s | %s | %s | %s |\n", assetVal, portVal, urlVal, ruleVal)
	}
	sb.WriteString("\n")
}

// severityCountMap counts findings by severity.
func severityCountMap(findings []*ReportFinding) map[models.FindingSeverity]int {
	counts := make(map[models.FindingSeverity]int)
	for _, rf := range findings {
		counts[rf.Finding.Severity]++
	}
	return counts
}

// statusCountMap counts findings by review status.
func statusCountMap(findings []*ReportFinding) map[models.FindingStatus]int {
	counts := make(map[models.FindingStatus]int)
	for _, rf := range findings {
		counts[rf.Finding.Status]++
	}
	return counts
}

