package report

import (
	"fmt"
	"sort"
	"strings"
	"time"

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

func statusLabelCN(st models.FindingStatus) string {
	if v, ok := statusLabelsCN[st]; ok {
		return v
	}
	return string(st)
}

// GenerateMarkdown produces a security assessment report in Markdown format (Chinese).
// All edge cases (nil data, empty slices) are handled gracefully.
func GenerateMarkdown(data *ReportData) string {
	if data == nil || data.Project == nil {
		return "错误：报告数据不完整（缺少项目信息）。"
	}

	var sb strings.Builder

	// --- 1. 标题 ---
	sb.WriteString("# 安全评估报告\n\n")

	// --- 2. 执行摘要 ---
	sb.WriteString("## 执行摘要\n\n")
	fmt.Fprintf(&sb, "| 项目字段 | 值 |\n")
	fmt.Fprintf(&sb, "|---|---|\n")
	fmt.Fprintf(&sb, "| **项目名称** | %s |\n", escapeMDTable(data.Project.Name))
	if data.Project.Organization != "" {
		fmt.Fprintf(&sb, "| **所属组织** | %s |\n", escapeMDTable(data.Project.Organization))
	}
	fmt.Fprintf(&sb, "| **报告生成时间** | %s |\n", data.GeneratedAt.Format("2006-01-02 15:04:05"))
	sb.WriteString("\n")

	severityCounts := severityCountMap(data.Findings)
	sb.WriteString("### 风险等级分布\n\n")
	sb.WriteString("| 严重程度 | 数量 |\n")
	sb.WriteString("|---|---|\n")
	for _, sev := range []models.FindingSeverity{
		models.SeverityCritical,
		models.SeverityHigh,
		models.SeverityMedium,
		models.SeverityLow,
		models.SeverityInfo,
	} {
		fmt.Fprintf(&sb, "| %s | %d |\n", severityLabelCN(sev), severityCounts[sev])
	}
	sb.WriteString("\n")

	statusCounts := statusCountMap(data.Findings)
	sb.WriteString("### 审核状态分布\n\n")
	sb.WriteString("| 状态 | 数量 |\n")
	sb.WriteString("|---|---|\n")
	for _, st := range []models.FindingStatus{
		models.FindingPendingReview,
		models.FindingConfirmed,
		models.FindingFalsePositive,
		models.FindingAcceptedRisk,
		models.FindingIgnored,
	} {
		fmt.Fprintf(&sb, "| %s | %d |\n", statusLabelCN(st), statusCounts[st])
	}
	sb.WriteString("\n")

	totalFindings := len(data.Findings)
	if totalFindings == 0 {
		sb.WriteString("> **本次扫描未发现任何漏洞。**\n\n")
	}

	// --- 3. 扫描范围 ---
	sb.WriteString("## 扫描范围\n\n")
	sb.WriteString("### 目标列表\n\n")
	if len(data.Targets) > 0 {
		for _, t := range data.Targets {
			fmt.Fprintf(&sb, "- `%s` (%s)\n", t.Value, t.Type)
		}
	} else {
		sb.WriteString("*未定义目标。*\n")
	}
	sb.WriteString("\n")

	sb.WriteString("### 范围规则\n\n")
	if len(data.ScopeRules) > 0 {
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
	} else {
		sb.WriteString("*未定义范围规则。*\n")
	}
	sb.WriteString("\n")

	// --- 4. 工具方法 ---
	sb.WriteString("## 检测方法\n\n")
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

		sb.WriteString("| 工具 | 版本 | 路径 |\n")
		sb.WriteString("|---|---|---|\n")
		for _, name := range toolNames {
			inv := toolMap[name]
			version := inv.Version
			if version == "" {
				version = "未知"
			}
			fmt.Fprintf(&sb, "| %s | %s | %s |\n", escapeMDTable(name), escapeMDTable(version), escapeMDTable(inv.BinaryPath))
		}
	} else {
		sb.WriteString("*无工具调用记录。*\n")
	}
	sb.WriteString("\n")

	// --- 5. 漏洞清单（按严重程度倒序展示全部 findings）---
	sb.WriteString("## 漏洞清单\n\n")
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
			fmt.Fprintf(&sb, "### %s 漏洞（共 %d 条）\n\n", severityLabelCN(sev), len(group))
			for _, rf := range group {
				num++
				writeFindingDetailCN(&sb, num, rf)
			}
		}
	}

	// --- 6. 附录 ---
	sb.WriteString("## 附录\n\n")
	sb.WriteString("### 工具版本明细\n\n")
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
			fmt.Fprintf(&sb, "- **%s**: %s (`%s`)\n", name, version, inv.BinaryPath)
		}
	} else {
		sb.WriteString("*无工具调用记录。*\n")
	}
	sb.WriteString("\n")

	sb.WriteString("### 报告元数据\n\n")
	fmt.Fprintf(&sb, "- **生成工具**：Anchor v0.1.0\n")
	fmt.Fprintf(&sb, "- **生成时间**：%s\n", data.GeneratedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&sb, "- **项目 ID**：%s\n", data.Project.ID)
	sb.WriteString("\n")

	return sb.String()
}

// writeFindingDetailCN writes a single finding entry in Chinese.
func writeFindingDetailCN(sb *strings.Builder, num int, rf *ReportFinding) {
	f := rf.Finding

	fmt.Fprintf(sb, "#### %d. %s\n\n", num, escapeMDTable(f.Title))
	fmt.Fprintf(sb, "| 字段 | 值 |\n")
	fmt.Fprintf(sb, "|---|---|\n")

	assetVal := "N/A"
	if rf.Asset != nil {
		assetVal = fmt.Sprintf("%s (%s)", escapeMDTable(rf.Asset.Value), escapeMDTable(string(rf.Asset.Type)))
	}
	fmt.Fprintf(sb, "| **资产** | %s |\n", assetVal)

	epVal := "N/A"
	if rf.WebEndpoint != nil {
		epVal = escapeMDTable(rf.WebEndpoint.URL)
	}
	fmt.Fprintf(sb, "| **端点** | %s |\n", epVal)

	fmt.Fprintf(sb, "| **严重程度** | %s |\n", severityLabelCN(f.Severity))
	fmt.Fprintf(sb, "| **置信度** | %d |\n", f.Confidence)
	fmt.Fprintf(sb, "| **审核状态** | %s |\n", statusLabelCN(f.Status))
	fmt.Fprintf(sb, "| **检测工具** | %s |\n", escapeMDTable(f.SourceTool))
	if f.SourceRuleID != "" {
		fmt.Fprintf(sb, "| **规则 ID** | %s |\n", escapeMDTable(f.SourceRuleID))
	}
	sb.WriteString("\n")

	sb.WriteString("**漏洞描述**\n\n")
	if f.Summary != "" {
		sb.WriteString(f.Summary)
		sb.WriteString("\n\n")
	} else {
		sb.WriteString("*未提供描述。*\n\n")
	}

	sb.WriteString("**证据链**\n\n")
	if len(rf.EvidenceList) > 0 {
		for _, ev := range rf.EvidenceList {
			fmt.Fprintf(sb, "**%s** — %s\n", string(ev.Type), ev.CreatedAt.Format("2006-01-02 15:04:05"))
			if ev.Excerpt != "" {
				excerpt := ev.Excerpt
				if len(excerpt) > 2000 {
					excerpt = excerpt[:2000] + "\n... (已截断)"
				}
				sb.WriteString("```\n")
				sb.WriteString(excerpt)
				sb.WriteString("\n```\n\n")
			}
		}
	} else {
		sb.WriteString("*暂无原始证据记录。*\n\n")
	}

	sb.WriteString("**修复建议**\n\n")
	if f.Remediation != "" {
		sb.WriteString(f.Remediation)
		sb.WriteString("\n\n")
	} else {
		sb.WriteString("*暂无修复建议。*\n\n")
	}

	sb.WriteString("---\n\n")
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
