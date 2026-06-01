package evaluator

import (
	"fmt"
	"time"
)

// DefaultRules returns the predefined evaluation rules.
func DefaultRules() []Rule {
	return []Rule{
		// Tool reliability rules
		{
			ID:          "tool_reliability_low",
			Category:    "工具可靠性",
			Name:        "工具成功率过低",
			Description: "工具调用成功率低于 80%",
			Condition: func(m *ScanMetrics) bool {
				for _, stat := range m.ToolStats {
					if stat.TotalCalls >= 5 && stat.SuccessRate < 0.8 {
						return true
					}
				}
				return false
			},
			Severity: "high",
			Suggestion: func(m *ScanMetrics) string {
				var suggestions []string
				for tool, stat := range m.ToolStats {
					if stat.TotalCalls >= 5 && stat.SuccessRate < 0.8 {
						suggestions = append(suggestions,
							fmt.Sprintf("%s 成功率 %.0f%%，建议检查配置和网络连接", tool, stat.SuccessRate*100))
					}
				}
				return joinSuggestions(suggestions)
			},
		},
		// Tool efficiency rules
		{
			ID:          "tool_efficiency_slow",
			Category:    "工具效率",
			Name:        "工具平均耗时过长",
			Description: "工具平均执行时间超过 10 分钟",
			Condition: func(m *ScanMetrics) bool {
				for _, stat := range m.ToolStats {
					if stat.AvgDuration > 10*time.Minute {
						return true
					}
				}
				return false
			},
			Severity: "medium",
			Suggestion: func(m *ScanMetrics) string {
				var suggestions []string
				for tool, stat := range m.ToolStats {
					if stat.AvgDuration > 10*time.Minute {
						suggestions = append(suggestions,
							fmt.Sprintf("%s 平均耗时 %v，考虑优化参数或减少目标范围", tool, stat.AvgDuration.Round(time.Second)))
					}
				}
				return joinSuggestions(suggestions)
			},
		},
		// Tool output rules
		{
			ID:          "tool_output_low",
			Category:    "工具产出",
			Name:        "工具无效运行率过高",
			Description: "工具运行但无产出的比例超过 30%",
			Condition: func(m *ScanMetrics) bool {
				for _, stat := range m.ToolStats {
					if stat.TotalCalls >= 5 {
						// Skipped count indicates no output
						noOutputRate := float64(stat.SkippedCount) / float64(stat.TotalCalls)
						if noOutputRate > 0.3 {
							return true
						}
					}
				}
				return false
			},
			Severity: "medium",
			Suggestion: func(m *ScanMetrics) string {
				var suggestions []string
				for tool, stat := range m.ToolStats {
					if stat.TotalCalls >= 5 {
						noOutputRate := float64(stat.SkippedCount) / float64(stat.TotalCalls)
						if noOutputRate > 0.3 {
							suggestions = append(suggestions,
								fmt.Sprintf("%s 有 %.0f%% 的运行无产出，建议优化目标筛选逻辑", tool, noOutputRate*100))
						}
					}
				}
				return joinSuggestions(suggestions)
			},
		},
		// Stage bottleneck rule
		{
			ID:          "stage_bottleneck",
			Category:    "执行瓶颈",
			Name:        "某阶段耗时占比过高",
			Description: "单个阶段耗时超过总耗时的 50%",
			Condition: func(m *ScanMetrics) bool {
				if m.TotalDuration == 0 {
					return false
				}
				for _, duration := range m.StageDurations {
					if float64(duration)/float64(m.TotalDuration) > 0.5 {
						return true
					}
				}
				return false
			},
			Severity: "high",
			Suggestion: func(m *ScanMetrics) string {
				if m.TotalDuration == 0 {
					return ""
				}
				var suggestions []string
				for stage, duration := range m.StageDurations {
					ratio := float64(duration) / float64(m.TotalDuration)
					if ratio > 0.5 {
						suggestions = append(suggestions,
							fmt.Sprintf("%s 阶段占总耗时 %.0f%%，建议分批执行或增加并行度", stage, ratio*100))
					}
				}
				return joinSuggestions(suggestions)
			},
		},
		// Stage failure rule
		{
			ID:          "stage_failure_high",
			Category:    "阶段失败",
			Name:        "阶段失败率过高",
			Description: "阶段失败率超过 20%",
			Condition: func(m *ScanMetrics) bool {
				if len(m.StageStatuses) == 0 {
					return false
				}
				failedCount := 0
				for _, status := range m.StageStatuses {
					if status == "failed" {
						failedCount++
					}
				}
				return float64(failedCount)/float64(len(m.StageStatuses)) > 0.2
			},
			Severity: "high",
			Suggestion: func(m *ScanMetrics) string {
				var failedStages []string
				for stage, status := range m.StageStatuses {
					if status == "failed" {
						failedStages = append(failedStages, stage)
					}
				}
				if len(failedStages) > 0 {
					return fmt.Sprintf("以下阶段执行失败: %v，检查目标可达性和工具配置", failedStages)
				}
				return ""
			},
		},
		// Finding confidence rule
		{
			ID:          "finding_confidence_low",
			Category:    "漏洞质量",
			Name:        "低置信度漏洞占比过高",
			Description: "置信度低于 60% 的漏洞占比超过 40%",
			Condition: func(m *ScanMetrics) bool {
				if m.TotalFindings == 0 {
					return false
				}
				return m.AvgConfidence < 60
			},
			Severity: "medium",
			Suggestion: func(m *ScanMetrics) string {
				return fmt.Sprintf("平均置信度 %.0f%%，建议优化检测规则或增加验证逻辑", m.AvgConfidence)
			},
		},
		// Finding unlinked rule
		{
			ID:          "finding_unlinked",
			Category:    "关联完整性",
			Name:        "未关联资产漏洞占比过高",
			Description: "未关联资产的漏洞占比超过 30%",
			Condition: func(m *ScanMetrics) bool {
				if m.TotalFindings == 0 {
					return false
				}
				return float64(m.UnlinkedFindings)/float64(m.TotalFindings) > 0.3
			},
			Severity: "medium",
			Suggestion: func(m *ScanMetrics) string {
				ratio := float64(m.UnlinkedFindings) / float64(m.TotalFindings) * 100
				return fmt.Sprintf("%.0f%% 漏洞未关联资产，检查资产解析逻辑", ratio)
			},
		},
	}
}

func joinSuggestions(suggestions []string) string {
	if len(suggestions) == 0 {
		return ""
	}
	result := ""
	for i, s := range suggestions {
		if i > 0 {
			result += "; "
		}
		result += s
	}
	return result
}
