package evaluator

import (
	"context"
	"fmt"
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
)

// TrendAnalyzer analyzes trends across multiple scans.
type TrendAnalyzer struct {
	queries *db.Queries
}

// NewTrendAnalyzer creates a new trend analyzer.
func NewTrendAnalyzer(queries *db.Queries) *TrendAnalyzer {
	return &TrendAnalyzer{queries: queries}
}

// QueryHistory queries historical scan data for a project.
func (a *TrendAnalyzer) QueryHistory(ctx context.Context, projectID string, limit int) ([]*TrendData, error) {
	// Get recent completed runs
	runs, err := a.queries.ListRecentCompletedRunsByProject(projectID, limit)
	if err != nil {
		return nil, err
	}

	var history []*TrendData
	for _, run := range runs {
		td := &TrendData{
			RunID:       run.ID,
			StartedAt:   &run.StartedAt,
			CompletedAt: run.CompletedAt,
		}

		// Get tool stats for this run
		toolStats, err := a.queries.GetToolStatsByRun(run.ID)
		if err != nil {
			continue // Skip on error
		}
		td.ToolStats = make(map[string]*ToolStat)
		for _, ts := range toolStats {
			td.ToolStats[ts.Tool] = &ToolStat{
				ToolName:   ts.Tool,
				TotalCalls: ts.TotalCalls,
				SuccessRate: func() float64 {
					if ts.TotalCalls > 0 {
						return float64(ts.SuccessCount) / float64(ts.TotalCalls)
					}
					return 0
				}(),
				AvgDuration: time.Duration(ts.AvgDuration * float64(time.Second)),
			}
		}

		// Get finding stats
		severityStats, err := a.queries.GetFindingStatsBySeverity(run.ID)
		if err == nil {
			td.FindingsBySeverity = make(map[string]int)
			for _, ss := range severityStats {
				td.FindingsBySeverity[ss.Severity] = ss.Count
				td.FindingsCount += ss.Count
			}
		}

		history = append(history, td)
	}

	return history, nil
}

// TrendData holds historical trend data for a single run.
type TrendData struct {
	RunID              string
	StartedAt          *time.Time
	CompletedAt        *time.Time
	ToolStats          map[string]*ToolStat
	FindingsCount      int
	FindingsBySeverity map[string]int
}

// Analyze analyzes trends between historical data and current metrics.
func (a *TrendAnalyzer) Analyze(history []*TrendData, current *ScanMetrics) *TrendAnalysis {
	if len(history) < 2 {
		return &TrendAnalysis{
			Period:     "数据不足",
			DataPoints: len(history),
		}
	}

	analysis := &TrendAnalysis{
		Period:     fmt.Sprintf("最近 %d 次扫描", len(history)),
		DataPoints: len(history),
		ToolTrends: make(map[string]*ToolTrend),
	}

	// Analyze tool trends
	for tool, currentStat := range current.ToolStats {
		var successRates []float64
		for _, h := range history {
			if ts, ok := h.ToolStats[tool]; ok {
				successRates = append(successRates, ts.SuccessRate)
			}
		}
		successRates = append(successRates, currentStat.SuccessRate)

		if len(successRates) >= 2 {
			analysis.ToolTrends[tool] = &ToolTrend{
				ToolName:      tool,
				SuccessRate:   CalculateTrend(successRates, 0.05),
				CurrentValue:  currentStat.SuccessRate,
				PreviousValue: successRates[len(successRates)-2],
				ChangePercent: (currentStat.SuccessRate - successRates[len(successRates)-2]) * 100,
			}
		}
	}

	// Analyze finding trends
	var findingCounts []float64
	for _, h := range history {
		findingCounts = append(findingCounts, float64(h.FindingsCount))
	}
	findingCounts = append(findingCounts, float64(current.TotalFindings))

	analysis.FindingTrend = &FindingTrend{
		TotalCount: CalculateTrend(findingCounts, 0.1),
	}

	// Detect significant changes
	analysis.SignificantChanges = a.DetectSignificantChanges(history, current)

	return analysis
}

// CalculateTrend calculates trend direction using linear regression.
func CalculateTrend(values []float64, threshold float64) TrendDirection {
	if len(values) < 2 {
		return TrendStable
	}

	n := float64(len(values))
	var sumX, sumY, sumXY, sumX2 float64
	for i, v := range values {
		x := float64(i)
		sumX += x
		sumY += v
		sumXY += x * v
		sumX2 += x * x
	}

	denominator := n*sumX2 - sumX*sumX
	if denominator == 0 {
		return TrendStable
	}

	slope := (n*sumXY - sumX*sumY) / denominator

	if slope > threshold {
		return TrendUp
	} else if slope < -threshold {
		return TrendDown
	}
	return TrendStable
}

// DetectSignificantChanges detects significant changes between scans.
func (a *TrendAnalyzer) DetectSignificantChanges(history []*TrendData, current *ScanMetrics) []Change {
	var changes []Change

	if len(history) == 0 {
		return changes
	}

	lastScan := history[len(history)-1]

	// Check tool success rate changes
	for tool, currentStat := range current.ToolStats {
		if prevStat, ok := lastScan.ToolStats[tool]; ok {
			change := currentStat.SuccessRate - prevStat.SuccessRate
			if change < -0.1 { // Decreased by more than 10%
				changes = append(changes, Change{
					Dimension:   "tool_efficiency",
					Entity:      tool,
					Description: fmt.Sprintf("%s 成功率从 %.0f%% 下降到 %.0f%%", tool, prevStat.SuccessRate*100, currentStat.SuccessRate*100),
					Severity:    "degradation",
				})
			} else if change > 0.1 { // Increased by more than 10%
				changes = append(changes, Change{
					Dimension:   "tool_efficiency",
					Entity:      tool,
					Description: fmt.Sprintf("%s 成功率从 %.0f%% 上升到 %.0f%%", tool, prevStat.SuccessRate*100, currentStat.SuccessRate*100),
					Severity:    "improvement",
				})
			}
		}
	}

	// Check finding count changes
	if lastScan.FindingsCount > 0 {
		change := float64(current.TotalFindings-lastScan.FindingsCount) / float64(lastScan.FindingsCount)
		if change > 0.5 { // Increased by more than 50%
			changes = append(changes, Change{
				Dimension:   "finding_count",
				Description: fmt.Sprintf("漏洞数量从 %d 增长到 %d (+%.0f%%)", lastScan.FindingsCount, current.TotalFindings, change*100),
				Severity:    "neutral",
			})
		}
	}

	return changes
}
