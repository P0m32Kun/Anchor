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

	// --- 4. 漏洞详情(按严重程度倒序) ---
	sb.WriteString("## 三、漏洞详情\n\n")
	if totalFindings == 0 {
		sb.WriteString("*无漏洞记录。*\n\n")
	} else {
		bySeverity := groupBySeverity(data.Findings)
		num := 0
		for _, sev := range []models.FindingSeverity{
			models.SeverityCritical,
			models.SeverityHigh,
			models.SeverityMedium,
			models.SeverityLow,
			models.SeverityInfo,
		} {
			group := bySeverity[sev]
			if len(group) == 0 {
				continue
			}
			for _, rf := range group {
				num++
				writeFindingDetailCN(&sb, num, rf)
			}
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

// writeFindingDetailCN renders a single finding in the human-friendly layout.
// Sections: emoji+severity+title heading → description → affected IP:Port table → remediation → footer metadata.
func writeFindingDetailCN(sb *strings.Builder, num int, rf *ReportFinding) {
	f := rf.Finding

	fmt.Fprintf(sb, "### %d. %s %s · %s\n\n",
		num,
		severityEmojiOf(f.Severity),
		severityLabelCN(f.Severity),
		escapeMDTable(f.Title),
	)

	// 漏洞描述
	sb.WriteString("**漏洞描述**\n\n")
	if strings.TrimSpace(f.Summary) != "" {
		sb.WriteString(f.Summary)
		sb.WriteString("\n\n")
	} else {
		sb.WriteString("*暂无描述。*\n\n")
	}

	// 涉及 IP:Port
	sb.WriteString("**涉及 IP:Port**\n\n")
	writeAffectedTargets(sb, rf)
	sb.WriteString("\n")

	// 修复建议
	sb.WriteString("**修复建议**\n\n")
	if strings.TrimSpace(f.Remediation) != "" {
		sb.WriteString(f.Remediation)
		sb.WriteString("\n\n")
	} else {
		sb.WriteString("*暂无修复建议，请咨询安全团队进一步分析。*\n\n")
	}

	// 底部元数据小字
	meta := []string{}
	if f.SourceTool != "" {
		meta = append(meta, "检测来源："+escapeMDTable(f.SourceTool))
	}
	if f.SourceRuleID != "" {
		meta = append(meta, "规则："+escapeMDTable(f.SourceRuleID))
	}
	if f.Confidence > 0 {
		meta = append(meta, fmt.Sprintf("置信度 %d%%", f.Confidence))
	}
	meta = append(meta, "状态："+statusLabelCN(f.Status))
	fmt.Fprintf(sb, "> %s\n\n", strings.Join(meta, " · "))

	sb.WriteString("---\n\n")
}

// writeAffectedTargets renders the IP:Port section as a compact table.
// Falls back to a plain "*暂无具体目标信息。*" line when nothing is known.
func writeAffectedTargets(sb *strings.Builder, rf *ReportFinding) {
	if rf.Asset == nil && rf.WebEndpoint == nil {
		sb.WriteString("*暂无具体目标信息。*\n")
		return
	}

	sb.WriteString("| 资产 | 端口 | 访问地址 |\n")
	sb.WriteString("|---|---|---|\n")

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

	fmt.Fprintf(sb, "| %s | %s | %s |\n", assetVal, portVal, urlVal)
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

// filterByStatus returns findings matching a given status. Kept for html.go template.
func filterByStatus(findings []*ReportFinding, status models.FindingStatus) []*ReportFinding {
	var result []*ReportFinding
	for _, rf := range findings {
		if rf.Finding.Status == status {
			result = append(result, rf)
		}
	}
	return result
}

// groupBySeverity buckets findings by severity, preserving input ordering inside each bucket.
func groupBySeverity(findings []*ReportFinding) map[models.FindingSeverity][]*ReportFinding {
	groups := make(map[models.FindingSeverity][]*ReportFinding)
	for _, rf := range findings {
		groups[rf.Finding.Severity] = append(groups[rf.Finding.Severity], rf)
	}
	return groups
}
