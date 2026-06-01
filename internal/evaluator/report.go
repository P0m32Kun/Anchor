package evaluator

import (
	"fmt"
	"strings"
	"time"
)

// ReportGenerator generates evaluation reports in Markdown format.
type ReportGenerator struct{}

// NewReportGenerator creates a new report generator.
func NewReportGenerator() *ReportGenerator {
	return &ReportGenerator{}
}

// Generate generates a Markdown evaluation report.
func (g *ReportGenerator) Generate(metrics *ScanMetrics, issues []Issue, trends *TrendAnalysis) string {
	var sb strings.Builder

	// Header
	sb.WriteString("# 扫描质量评估报告\n\n")
	sb.WriteString(fmt.Sprintf("> **项目 ID**：%s\n", metrics.ProjectID))
	sb.WriteString(fmt.Sprintf("> **扫描运行**：%s\n", metrics.RunID))
	sb.WriteString(fmt.Sprintf("> **评估时间**：%s\n", time.Now().Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("> **扫描耗时**：%v\n\n", metrics.TotalDuration.Round(time.Second)))

	// Executive Summary
	sb.WriteString("## 一、执行摘要\n\n")
	sb.WriteString(fmt.Sprintf("本次扫描共发现 **%d** 个漏洞，执行耗时 **%v**。\n\n", metrics.TotalFindings, metrics.TotalDuration.Round(time.Second)))

	// Tool Effectiveness
	sb.WriteString("## 二、工具效果分析\n\n")
	sb.WriteString("### 2.1 工具调用统计\n\n")
	sb.WriteString("| 工具 | 调用次数 | 成功率 | 平均耗时 |\n")
	sb.WriteString("|------|---------|--------|----------|\n")
	for tool, stat := range metrics.ToolStats {
		sb.WriteString(fmt.Sprintf("| %s | %d | %.0f%% | %v |\n",
			tool, stat.TotalCalls, stat.SuccessRate*100, stat.AvgDuration.Round(time.Second)))
	}
	sb.WriteString("\n")

	// Template Effectiveness
	sb.WriteString("## 三、模板/字典效果分析\n\n")
	sb.WriteString("### 3.1 热门模板（命中 Top 10）\n\n")
	sb.WriteString("| 模板 ID | 工具 | 命中次数 | 确认次数 | 有效率 |\n")
	sb.WriteString("|---------|------|---------|---------|--------|\n")
	count := 0
	for _, stat := range metrics.TemplateStats {
		if count >= 10 {
			break
		}
		sb.WriteString(fmt.Sprintf("| %s | %s | %d | %d | %.0f%% |\n",
			stat.TemplateID, stat.SourceTool, stat.HitCount, stat.ConfirmedCount, stat.Effectiveness*100))
		count++
	}
	sb.WriteString("\n")

	// Execution Efficiency
	sb.WriteString("## 四、执行效率分析\n\n")
	sb.WriteString("### 4.1 阶段耗时分布\n\n")
	sb.WriteString("| 阶段 | 耗时 | 占比 | 状态 |\n")
	sb.WriteString("|------|------|------|------|\n")
	for stage, duration := range metrics.StageDurations {
		percent := 0.0
		if metrics.TotalDuration > 0 {
			percent = float64(duration) / float64(metrics.TotalDuration) * 100
		}
		status := metrics.StageStatuses[stage]
		sb.WriteString(fmt.Sprintf("| %s | %v | %.0f%% | %s |\n",
			stage, duration.Round(time.Second), percent, status))
	}
	sb.WriteString("\n")

	// Finding Quality
	sb.WriteString("## 五、漏洞质量分析\n\n")
	sb.WriteString("### 5.1 漏洞严重程度分布\n\n")
	sb.WriteString("| 严重程度 | 数量 | 占比 |\n")
	sb.WriteString("|---------|------|------|\n")
	severityEmoji := map[string]string{
		"critical": "🔴",
		"high":     "🟠",
		"medium":   "🟡",
		"low":      "🔵",
		"info":     "⚪",
	}
	for severity, count := range metrics.FindingsBySeverity {
		percent := 0.0
		if metrics.TotalFindings > 0 {
			percent = float64(count) / float64(metrics.TotalFindings) * 100
		}
		emoji := severityEmoji[severity]
		if emoji == "" {
			emoji = "⚪"
		}
		sb.WriteString(fmt.Sprintf("| %s %s | %d | %.0f%% |\n", emoji, severity, count, percent))
	}
	sb.WriteString("\n")

	sb.WriteString(fmt.Sprintf("- **平均置信度**：%.0f%%\n", metrics.AvgConfidence))
	sb.WriteString(fmt.Sprintf("- **未关联资产漏洞**：%d\n\n", metrics.UnlinkedFindings))

	// Issues and Suggestions
	sb.WriteString("## 六、优化建议\n\n")
	if len(issues) == 0 {
		sb.WriteString("✅ 未发现需要优化的问题。\n\n")
	} else {
		// Group by severity
		highIssues := filterIssuesBySeverity(issues, "high")
		mediumIssues := filterIssuesBySeverity(issues, "medium")
		lowIssues := filterIssuesBySeverity(issues, "low")

		if len(highIssues) > 0 {
			sb.WriteString("### 6.1 高优先级建议\n\n")
			for _, issue := range highIssues {
				sb.WriteString(fmt.Sprintf("- **%s**: %s\n  - %s\n", issue.Category, issue.Description, issue.Suggestion))
			}
			sb.WriteString("\n")
		}

		if len(mediumIssues) > 0 {
			sb.WriteString("### 6.2 中优先级建议\n\n")
			for _, issue := range mediumIssues {
				sb.WriteString(fmt.Sprintf("- **%s**: %s\n  - %s\n", issue.Category, issue.Description, issue.Suggestion))
			}
			sb.WriteString("\n")
		}

		if len(lowIssues) > 0 {
			sb.WriteString("### 6.3 低优先级建议\n\n")
			for _, issue := range lowIssues {
				sb.WriteString(fmt.Sprintf("- **%s**: %s\n  - %s\n", issue.Category, issue.Description, issue.Suggestion))
			}
			sb.WriteString("\n")
		}
	}

	// Trend Analysis
	sb.WriteString("## 七、趋势分析\n\n")
	if trends != nil && trends.DataPoints > 0 {
		sb.WriteString(fmt.Sprintf("**分析周期**：%s（%d 个数据点）\n\n", trends.Period, trends.DataPoints))

		if len(trends.SignificantChanges) > 0 {
			sb.WriteString("### 7.1 显著变化\n\n")
			for _, change := range trends.SignificantChanges {
				emoji := "🔄"
				if change.Severity == "improvement" {
					emoji = "📈"
				} else if change.Severity == "degradation" {
					emoji = "📉"
				}
				sb.WriteString(fmt.Sprintf("- %s %s\n", emoji, change.Description))
			}
			sb.WriteString("\n")
		}
	} else {
		sb.WriteString("历史数据不足，无法进行趋势分析。\n\n")
	}

	return sb.String()
}

func filterIssuesBySeverity(issues []Issue, severity string) []Issue {
	var filtered []Issue
	for _, issue := range issues {
		if issue.Severity == severity {
			filtered = append(filtered, issue)
		}
	}
	return filtered
}
