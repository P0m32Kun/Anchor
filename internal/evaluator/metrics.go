package evaluator

import "time"

// ScanMetrics 汇总的扫描指标
type ScanMetrics struct {
	RunID       string
	ProjectID   string
	RunName     string
	StartedAt   time.Time
	CompletedAt time.Time

	// 工具效果
	ToolStats map[string]*ToolStat

	// 模板/字典效果
	TemplateStats   map[string]*TemplateStat
	DictionaryStats map[string]*DictionaryStat

	// 执行效率
	TotalDuration   time.Duration
	StageDurations  map[string]time.Duration
	StageStatuses   map[string]string

	// 漏洞质量
	FindingsBySeverity map[string]int
	FindingsByStatus   map[string]int
	AvgConfidence      float64
	UnlinkedFindings   int
	TotalFindings      int
}

// ToolStat 单个工具的统计
type ToolStat struct {
	ToolName     string
	TotalCalls   int
	SuccessCount int
	FailedCount  int
	SkippedCount int
	SuccessRate  float64
	AvgDuration  time.Duration
	OutputRate   float64 // 产出效率：资产数/秒
	CommonErrors []ErrorCount
}

// ErrorCount 错误计数
type ErrorCount struct {
	Error string
	Count int
}

// TemplateStat 单个模板的统计
type TemplateStat struct {
	TemplateID     string
	SourceTool     string
	HitCount       int
	ConfirmedCount int
	Effectiveness  float64 // confirmed / hit_count
}

// DictionaryStat 单个字典的统计
type DictionaryStat struct {
	DictionaryName string
	UsedInTool     string
	HitRate        float64
	UniquePaths    int
}

// EvaluationReport 评估报告
type EvaluationReport struct {
	Path        string
	GeneratedAt time.Time
	Metrics     *ScanMetrics
	Issues      []Issue
	Trends      *TrendAnalysis
	Content     string // Markdown 内容
}

// Issue 规则引擎识别的问题
type Issue struct {
	RuleID      string
	Category    string
	Severity    string // high/medium/low
	Description string
	Suggestion  string
}

// TrendAnalysis 趋势分析结果
type TrendAnalysis struct {
	Period             string
	DataPoints         int
	ToolTrends         map[string]*ToolTrend
	FindingTrend       *FindingTrend
	EfficiencyTrend    *EfficiencyTrend
	SignificantChanges []Change
}

// TrendDirection 趋势方向
type TrendDirection string

const (
	TrendUp     TrendDirection = "up"
	TrendDown   TrendDirection = "down"
	TrendStable TrendDirection = "stable"
)

// ToolTrend 单个工具的趋势
type ToolTrend struct {
	ToolName      string
	SuccessRate   TrendDirection
	AvgDuration   TrendDirection
	OutputRate    TrendDirection
	CurrentValue  float64
	PreviousValue float64
	ChangePercent float64
}

// FindingTrend 漏洞趋势
type FindingTrend struct {
	TotalCount    TrendDirection
	CriticalCount TrendDirection
	HighCount     TrendDirection
	AvgConfidence TrendDirection
}

// EfficiencyTrend 效率趋势
type EfficiencyTrend struct {
	TotalDuration    TrendDirection
	NucleiDuration   TrendDirection
	PortscanDuration TrendDirection
}

// Change 显著变化
type Change struct {
	Dimension   string
	Entity      string
	Description string
	Severity    string // improvement/degradation/neutral
}
