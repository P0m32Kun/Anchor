---
status: accepted
source_of_truth: false
owner: kun
last_updated: 2026-06-17
scope: scan-evaluator
reason: "设计已批准，评估模块已纳入 architecture.md"
---

# 扫描质量评估系统实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 构建一个独立的扫描质量评估服务，在扫描完成后自动分析扫描效果并生成优化建议报告

**Architecture:** 创建 `internal/evaluator/` 包作为独立评估服务，通过 `db.Queries` 查询扫描数据，使用规则引擎识别问题，生成 Markdown 格式评估报告存储到项目目录。API 层在扫描完成后异步触发评估。

**Tech Stack:** Go, SQLite, Markdown 生成

---

## 文件结构

```
internal/
└── evaluator/
    ├── evaluator.go              # 评估器入口，协调各子组件
    ├── tool_effectiveness.go     # 工具效果评估
    ├── template_effectiveness.go # 模板/字典效果评估
    ├── efficiency.go             # 执行效率评估
    ├── finding_quality.go        # 漏洞质量评估
    ├── rules.go                  # 规则引擎核心
    ├── rules_definitions.go      # 预定义规则列表
    ├── trend.go                  # 趋势分析
    ├── report.go                 # 报告生成器
    ├── metrics.go                # 指标数据结构
    ├── evaluator_test.go         # 评估器单元测试
    └── rules_test.go             # 规则引擎测试

internal/api/
    └── evaluation_handlers.go    # API 端点
```

---

## Task 1: 创建指标数据结构

**Files:**
- Create: `internal/evaluator/metrics.go`

- [ ] **Step 1: 创建 metrics.go 文件**

```go
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
```

- [ ] **Step 2: 验证编译**

Run: `cd /Users/kun/DEV/Anchor && go build ./internal/evaluator/`
Expected: 编译成功（可能需要先创建目录）

- [ ] **Step 3: Commit**

```bash
git add internal/evaluator/metrics.go
git commit -m "feat(evaluator): add metrics data structures"
```

---

## Task 2: 创建数据库查询方法

**Files:**
- Modify: `internal/db/queries_scan_work.go` (添加工具统计查询)
- Modify: `internal/db/queries_finding.go` (添加漏洞统计查询)
- Modify: `internal/db/queries_scan.go` (添加历史运行查询)

- [ ] **Step 1: 在 queries_scan_work.go 添加工具统计查询**

在文件末尾添加：

```go
// ToolStats holds aggregated statistics for a single tool.
type ToolStats struct {
	Tool         string
	TotalCalls   int
	SuccessCount int
	FailedCount  int
	SkippedCount int
	AvgDuration  float64 // seconds
}

// GetToolStatsByRun returns aggregated tool statistics for a run.
func (q *Queries) GetToolStatsByRun(runID string) ([]*ToolStats, error) {
	rows, err := q.db.Query(`
		SELECT
			action as tool,
			COUNT(*) as total_calls,
			SUM(CASE WHEN status = 'done' THEN 1 ELSE 0 END) as success_count,
			SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed_count,
			SUM(CASE WHEN status = 'skipped' THEN 1 ELSE 0 END) as skipped_count,
			AVG(CASE
				WHEN started_at IS NOT NULL AND completed_at IS NOT NULL
				THEN (julianday(completed_at) - julianday(started_at)) * 86400
				ELSE NULL
			END) as avg_duration
		FROM scan_work_items
		WHERE run_id = ?
		GROUP BY action`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []*ToolStats
	for rows.Next() {
		s := &ToolStats{}
		var avgDuration sql.NullFloat64
		if err := rows.Scan(&s.Tool, &s.TotalCalls, &s.SuccessCount, &s.FailedCount, &s.SkippedCount, &avgDuration); err != nil {
			return nil, err
		}
		if avgDuration.Valid {
			s.AvgDuration = avgDuration.Float64
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

// ToolErrorStats holds error distribution for a tool.
type ToolErrorStats struct {
	Tool  string
	Error string
	Count int
}

// GetToolErrorStatsByRun returns error distribution for tools in a run.
func (q *Queries) GetToolErrorStatsByRun(runID string) ([]*ToolErrorStats, error) {
	rows, err := q.db.Query(`
		SELECT action, error, COUNT(*) as cnt
		FROM scan_work_items
		WHERE run_id = ? AND status = 'failed' AND error != ''
		GROUP BY action, error
		ORDER BY cnt DESC`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []*ToolErrorStats
	for rows.Next() {
		s := &ToolErrorStats{}
		if err := rows.Scan(&s.Tool, &s.Error, &s.Count); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}
```

- [ ] **Step 2: 在 queries_finding.go 添加漏洞统计查询**

在文件末尾添加：

```go
// FindingSeverityStats holds finding count by severity.
type FindingSeverityStats struct {
	Severity string
	Count    int
}

// GetFindingStatsBySeverity returns finding counts grouped by severity for a run.
func (q *Queries) GetFindingStatsBySeverity(runID string) ([]*FindingSeverityStats, error) {
	rows, err := q.db.Query(`
		SELECT severity, COUNT(*) as cnt
		FROM findings
		WHERE run_id = ?
		GROUP BY severity`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []*FindingSeverityStats
	for rows.Next() {
		s := &FindingSeverityStats{}
		if err := rows.Scan(&s.Severity, &s.Count); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

// FindingStatusStats holds finding count by status.
type FindingStatusStats struct {
	Status string
	Count  int
}

// GetFindingStatsByStatus returns finding counts grouped by status for a run.
func (q *Queries) GetFindingStatsByStatus(runID string) ([]*FindingStatusStats, error) {
	rows, err := q.db.Query(`
		SELECT status, COUNT(*) as cnt
		FROM findings
		WHERE run_id = ?
		GROUP BY status`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []*FindingStatusStats
	for rows.Next() {
		s := &FindingStatusStats{}
		if err := rows.Scan(&s.Status, &s.Count); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

// GetFindingAvgConfidence returns average confidence for a run.
func (q *Queries) GetFindingAvgConfidence(runID string) (float64, error) {
	var avg sql.NullFloat64
	err := q.db.QueryRow(`
		SELECT AVG(confidence)
		FROM findings
		WHERE run_id = ?`, runID).Scan(&avg)
	if err != nil {
		return 0, err
	}
	if avg.Valid {
		return avg.Float64, nil
	}
	return 0, nil
}

// GetUnlinkedFindingCount returns count of findings without asset_id for a run.
func (q *Queries) GetUnlinkedFindingCount(runID string) (int, error) {
	var count int
	err := q.db.QueryRow(`
		SELECT COUNT(*)
		FROM findings
		WHERE run_id = ? AND asset_id IS NULL`, runID).Scan(&count)
	return count, err
}

// TemplateHitStats holds template hit statistics.
type TemplateHitStats struct {
	SourceTool     string
	SourceRuleID   string
	HitCount       int
	ConfirmedCount int
}

// GetTemplateHitStats returns template hit statistics for a run.
func (q *Queries) GetTemplateHitStats(runID string) ([]*TemplateHitStats, error) {
	rows, err := q.db.Query(`
		SELECT
			source_tool,
			source_rule_id,
			COUNT(*) as hit_count,
			SUM(CASE WHEN status = 'confirmed' THEN 1 ELSE 0 END) as confirmed_count
		FROM findings
		WHERE run_id = ? AND source_rule_id != ''
		GROUP BY source_tool, source_rule_id
		ORDER BY hit_count DESC`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []*TemplateHitStats
	for rows.Next() {
		s := &TemplateHitStats{}
		if err := rows.Scan(&s.SourceTool, &s.SourceRuleID, &s.HitCount, &s.ConfirmedCount); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}
```

- [ ] **Step 3: 在 queries_scan.go 添加历史运行查询**

在文件末尾添加：

```go
// ListRecentCompletedRunsByProject returns recent completed runs for a project.
func (q *Queries) ListRecentCompletedRunsByProject(projectID string, limit int) ([]*models.PipelineRun, error) {
	rows, err := q.db.Query(`
		SELECT id, project_id, name, status, engine_state, started_at, completed_at, created_at
		FROM pipeline_runs
		WHERE project_id = ? AND status IN ('completed', 'failed')
		ORDER BY created_at DESC
		LIMIT ?`, projectID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []*models.PipelineRun
	for rows.Next() {
		r := &models.PipelineRun{}
		var startedAt, completedAt sql.NullTime
		if err := rows.Scan(&r.ID, &r.ProjectID, &r.Name, &r.Status, &r.EngineState, &startedAt, &completedAt, &r.CreatedAt); err != nil {
			return nil, err
		}
		if startedAt.Valid {
			r.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			r.CompletedAt = &completedAt.Time
		}
		runs = append(runs, r)
	}
	return runs, rows.Err()
}

// PipelineRunStageStats holds stage duration statistics.
type PipelineRunStageStats struct {
	Stage       string
	Status      string
	StartedAt   *time.Time
	CompletedAt *time.Time
	Duration    float64 // seconds
}

// GetPipelineRunStageStats returns stage statistics for a run.
func (q *Queries) GetPipelineRunStageStats(runID string) ([]*PipelineRunStageStats, error) {
	rows, err := q.db.Query(`
		SELECT
			stage,
			status,
			started_at,
			completed_at,
			CASE
				WHEN started_at IS NOT NULL AND completed_at IS NOT NULL
				THEN (julianday(completed_at) - julianday(started_at)) * 86400
				ELSE 0
			END as duration
		FROM pipeline_run_stages
		WHERE run_id = ?
		ORDER BY started_at`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []*PipelineRunStageStats
	for rows.Next() {
		s := &PipelineRunStageStats{}
		var startedAt, completedAt sql.NullTime
		if err := rows.Scan(&s.Stage, &s.Status, &startedAt, &completedAt, &s.Duration); err != nil {
			return nil, err
		}
		if startedAt.Valid {
			s.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			s.CompletedAt = &completedAt.Time
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}
```

- [ ] **Step 4: 验证编译**

Run: `cd /Users/kun/DEV/Anchor && go build ./internal/db/`
Expected: 编译成功

- [ ] **Step 5: Commit**

```bash
git add internal/db/queries_scan_work.go internal/db/queries_finding.go internal/db/queries_scan.go
git commit -m "feat(db): add evaluation query methods"
```

---

## Task 3: 创建工具效果评估器

**Files:**
- Create: `internal/evaluator/tool_effectiveness.go`

- [ ] **Step 1: 创建 tool_effectiveness.go**

```go
package evaluator

import (
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
)

// ToolEffectivenessEvaluator evaluates tool effectiveness.
type ToolEffectivenessEvaluator struct {
	queries *db.Queries
}

// NewToolEffectivenessEvaluator creates a new evaluator.
func NewToolEffectivenessEvaluator(queries *db.Queries) *ToolEffectivenessEvaluator {
	return &ToolEffectivenessEvaluator{queries: queries}
}

// Evaluate collects tool effectiveness metrics for a run.
func (e *ToolEffectivenessEvaluator) Evaluate(runID string) (map[string]*ToolStat, error) {
	// Get tool stats
	toolStats, err := e.queries.GetToolStatsByRun(runID)
	if err != nil {
		return nil, err
	}

	// Get error stats
	errorStats, err := e.queries.GetToolErrorStatsByRun(runID)
	if err != nil {
		return nil, err
	}

	// Build error map
	errorMap := make(map[string][]ErrorCount)
	for _, es := range errorStats {
		errorMap[es.Tool] = append(errorMap[es.Tool], ErrorCount{
			Error: es.Error,
			Count: es.Count,
		})
	}

	// Build result
	result := make(map[string]*ToolStat)
	for _, ts := range toolStats {
		stat := &ToolStat{
			ToolName:     ts.Tool,
			TotalCalls:   ts.TotalCalls,
			SuccessCount: ts.SuccessCount,
			FailedCount:  ts.FailedCount,
			SkippedCount: ts.SkippedCount,
			AvgDuration:  time.Duration(ts.AvgDuration * float64(time.Second)),
			CommonErrors: errorMap[ts.Tool],
		}

		if ts.TotalCalls > 0 {
			stat.SuccessRate = float64(ts.SuccessCount) / float64(ts.TotalCalls)
		}

		if ts.AvgDuration > 0 && ts.SuccessCount > 0 {
			stat.OutputRate = float64(ts.SuccessCount) / ts.AvgDuration
		}

		result[ts.Tool] = stat
	}

	return result, nil
}
```

- [ ] **Step 2: 验证编译**

Run: `cd /Users/kun/DEV/Anchor && go build ./internal/evaluator/`
Expected: 编译成功

- [ ] **Step 3: Commit**

```bash
git add internal/evaluator/tool_effectiveness.go
git commit -m "feat(evaluator): add tool effectiveness evaluator"
```

---

## Task 4: 创建模板/字典效果评估器

**Files:**
- Create: `internal/evaluator/template_effectiveness.go`

- [ ] **Step 1: 创建 template_effectiveness.go**

```go
package evaluator

import "github.com/P0m32Kun/Anchor/internal/db"

// TemplateEffectivenessEvaluator evaluates template and dictionary effectiveness.
type TemplateEffectivenessEvaluator struct {
	queries *db.Queries
}

// NewTemplateEffectivenessEvaluator creates a new evaluator.
func NewTemplateEffectivenessEvaluator(queries *db.Queries) *TemplateEffectivenessEvaluator {
	return &TemplateEffectivenessEvaluator{queries: queries}
}

// EvaluateTemplateStats collects template hit statistics for a run.
func (e *TemplateEffectivenessEvaluator) EvaluateTemplateStats(runID string) (map[string]*TemplateStat, error) {
	hits, err := e.queries.GetTemplateHitStats(runID)
	if err != nil {
		return nil, err
	}

	result := make(map[string]*TemplateStat)
	for _, h := range hits {
		key := h.SourceTool + ":" + h.SourceRuleID
		stat := &TemplateStat{
			TemplateID:     h.SourceRuleID,
			SourceTool:     h.SourceTool,
			HitCount:       h.HitCount,
			ConfirmedCount: h.ConfirmedCount,
		}
		if h.HitCount > 0 {
			stat.Effectiveness = float64(h.ConfirmedCount) / float64(h.HitCount)
		}
		result[key] = stat
	}

	return result, nil
}

// EvaluateDictionaryStats collects dictionary hit statistics for a run.
// Note: Dictionary stats are derived from ffuf work items.
func (e *TemplateEffectivenessEvaluator) EvaluateDictionaryStats(runID string) (map[string]*DictionaryStat, error) {
	// TODO: Implement dictionary stats when ffuf output parsing is available
	// For now, return empty stats
	return make(map[string]*DictionaryStat), nil
}
```

- [ ] **Step 2: 验证编译**

Run: `cd /Users/kun/DEV/Anchor && go build ./internal/evaluator/`
Expected: 编译成功

- [ ] **Step 3: Commit**

```bash
git add internal/evaluator/template_effectiveness.go
git commit -m "feat(evaluator): add template effectiveness evaluator"
```

---

## Task 5: 创建执行效率评估器

**Files:**
- Create: `internal/evaluator/efficiency.go`

- [ ] **Step 1: 创建 efficiency.go**

```go
package evaluator

import (
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
)

// EfficiencyEvaluator evaluates scan execution efficiency.
type EfficiencyEvaluator struct {
	queries *db.Queries
}

// NewEfficiencyEvaluator creates a new evaluator.
func NewEfficiencyEvaluator(queries *db.Queries) *EfficiencyEvaluator {
	return &EfficiencyEvaluator{queries: queries}
}

// EfficiencyResult holds efficiency evaluation results.
type EfficiencyResult struct {
	TotalDuration  time.Duration
	StageDurations map[string]time.Duration
	StageStatuses  map[string]string
}

// Evaluate collects efficiency metrics for a run.
func (e *EfficiencyEvaluator) Evaluate(runID string) (*EfficiencyResult, error) {
	// Get run info for total duration
	run, err := e.queries.GetPipelineRun(runID)
	if err != nil {
		return nil, err
	}

	result := &EfficiencyResult{
		StageDurations: make(map[string]time.Duration),
		StageStatuses:  make(map[string]string),
	}

	if run.StartedAt != nil && run.CompletedAt != nil {
		result.TotalDuration = run.CompletedAt.Sub(*run.StartedAt)
	}

	// Get stage stats
	stageStats, err := e.queries.GetPipelineRunStageStats(runID)
	if err != nil {
		return nil, err
	}

	for _, ss := range stageStats {
		result.StageDurations[ss.Stage] = time.Duration(ss.Duration * float64(time.Second))
		result.StageStatuses[ss.Stage] = ss.Status
	}

	return result, nil
}
```

- [ ] **Step 2: 验证编译**

Run: `cd /Users/kun/DEV/Anchor && go build ./internal/evaluator/`
Expected: 编译成功

- [ ] **Step 3: Commit**

```bash
git add internal/evaluator/efficiency.go
git commit -m "feat(evaluator): add efficiency evaluator"
```

---

## Task 6: 创建漏洞质量评估器

**Files:**
- Create: `internal/evaluator/finding_quality.go`

- [ ] **Step 1: 创建 finding_quality.go**

```go
package evaluator

import "github.com/P0m32Kun/Anchor/internal/db"

// FindingQualityEvaluator evaluates finding quality.
type FindingQualityEvaluator struct {
	queries *db.Queries
}

// NewFindingQualityEvaluator creates a new evaluator.
func NewFindingQualityEvaluator(queries *db.Queries) *FindingQualityEvaluator {
	return &FindingQualityEvaluator{queries: queries}
}

// FindingQualityResult holds finding quality evaluation results.
type FindingQualityResult struct {
	FindingsBySeverity map[string]int
	FindingsByStatus   map[string]int
	AvgConfidence      float64
	UnlinkedFindings   int
	TotalFindings      int
}

// Evaluate collects finding quality metrics for a run.
func (e *FindingQualityEvaluator) Evaluate(runID string) (*FindingQualityResult, error) {
	result := &FindingQualityResult{
		FindingsBySeverity: make(map[string]int),
		FindingsByStatus:   make(map[string]int),
	}

	// Get severity stats
	severityStats, err := e.queries.GetFindingStatsBySeverity(runID)
	if err != nil {
		return nil, err
	}
	for _, ss := range severityStats {
		result.FindingsBySeverity[ss.Severity] = ss.Count
		result.TotalFindings += ss.Count
	}

	// Get status stats
	statusStats, err := e.queries.GetFindingStatsByStatus(runID)
	if err != nil {
		return nil, err
	}
	for _, ss := range statusStats {
		result.FindingsByStatus[ss.Status] = ss.Count
	}

	// Get average confidence
	result.AvgConfidence, err = e.queries.GetFindingAvgConfidence(runID)
	if err != nil {
		return nil, err
	}

	// Get unlinked findings
	result.UnlinkedFindings, err = e.queries.GetUnlinkedFindingCount(runID)
	if err != nil {
		return nil, err
	}

	return result, nil
}
```

- [ ] **Step 2: 验证编译**

Run: `cd /Users/kun/DEV/Anchor && go build ./internal/evaluator/`
Expected: 编译成功

- [ ] **Step 3: Commit**

```bash
git add internal/evaluator/finding_quality.go
git commit -m "feat(evaluator): add finding quality evaluator"
```

---

## Task 7: 创建规则引擎

**Files:**
- Create: `internal/evaluator/rules.go`
- Create: `internal/evaluator/rules_definitions.go`

- [ ] **Step 1: 创建 rules.go**

```go
package evaluator

// Rule defines an evaluation rule.
type Rule struct {
	ID          string
	Category    string
	Name        string
	Description string
	Condition   func(metrics *ScanMetrics) bool
	Severity    string // high/medium/low
	Suggestion  func(metrics *ScanMetrics) string
}

// RuleEngine evaluates rules against metrics.
type RuleEngine struct {
	rules []Rule
}

// NewRuleEngine creates a new rule engine with predefined rules.
func NewRuleEngine() *RuleEngine {
	return &RuleEngine{
		rules: DefaultRules(),
	}
}

// Evaluate runs all rules against the metrics and returns triggered issues.
func (e *RuleEngine) Evaluate(metrics *ScanMetrics) []Issue {
	var issues []Issue

	for _, rule := range e.rules {
		if rule.Condition(metrics) {
			issues = append(issues, Issue{
				RuleID:      rule.ID,
				Category:    rule.Category,
				Severity:    rule.Severity,
				Description: rule.Description,
				Suggestion:  rule.Suggestion(metrics),
			})
		}
	}

	return issues
}
```

- [ ] **Step 2: 创建 rules_definitions.go**

```go
package evaluator

import "fmt"

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
					if stat.AvgDuration > 10*60*1000000000 { // 10 minutes in nanoseconds
						return true
					}
				}
				return false
			},
			Severity: "medium",
			Suggestion: func(m *ScanMetrics) string {
				var suggestions []string
				for tool, stat := range m.ToolStats {
					if stat.AvgDuration > 10*60*1000000000 {
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
```

- [ ] **Step 3: 添加 time import 到 rules.go**

在 rules.go 文件顶部添加：

```go
package evaluator

import "time"
```

- [ ] **Step 4: 验证编译**

Run: `cd /Users/kun/DEV/Anchor && go build ./internal/evaluator/`
Expected: 编译成功

- [ ] **Step 5: Commit**

```bash
git add internal/evaluator/rules.go internal/evaluator/rules_definitions.go
git commit -m "feat(evaluator): add rule engine with predefined rules"
```

---

## Task 8: 创建趋势分析器

**Files:**
- Create: `internal/evaluator/trend.go`

- [ ] **Step 1: 创建 trend.go**

```go
package evaluator

import (
	"context"
	"math"

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
			RunID:     run.ID,
			RunName:   run.Name,
			StartedAt: run.StartedAt,
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
				AvgDuration: ts.AvgDuration,
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
	RunID             string
	RunName           string
	StartedAt         *time.Time
	CompletedAt       *time.Time
	ToolStats         map[string]*ToolStat
	FindingsCount     int
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
		Period:     "最近 " + fmt.Sprintf("%d", len(history)) + " 次扫描",
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

		analysis.ToolTrends[tool] = &ToolTrend{
			ToolName:      tool,
			SuccessRate:   CalculateTrend(successRates, 0.05),
			CurrentValue:  currentStat.SuccessRate,
			PreviousValue: successRates[len(successRates)-2],
			ChangePercent: (currentStat.SuccessRate - successRates[len(successRates)-2]) * 100,
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
```

- [ ] **Step 2: 添加必要的 import**

在 trend.go 文件顶部添加：

```go
package evaluator

import (
	"context"
	"fmt"
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
)
```

- [ ] **Step 3: 验证编译**

Run: `cd /Users/kun/DEV/Anchor && go build ./internal/evaluator/`
Expected: 编译成功

- [ ] **Step 4: Commit**

```bash
git add internal/evaluator/trend.go
git commit -m "feat(evaluator): add trend analyzer"
```

---

## Task 9: 创建报告生成器

**Files:**
- Create: `internal/evaluator/report.go`

- [ ] **Step 1: 创建 report.go**

```go
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
	sb.WriteString(fmt.Sprintf("> **扫描运行**：%s\n", metrics.RunName))
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
```

- [ ] **Step 2: 验证编译**

Run: `cd /Users/kun/DEV/Anchor && go build ./internal/evaluator/`
Expected: 编译成功

- [ ] **Step 3: Commit**

```bash
git add internal/evaluator/report.go
git commit -m "feat(evaluator): add report generator"
```

---

## Task 10: 创建评估器入口

**Files:**
- Create: `internal/evaluator/evaluator.go`

- [ ] **Step 1: 创建 evaluator.go**

```go
package evaluator

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
)

// Evaluator is the main entry point for scan quality evaluation.
type Evaluator struct {
	queries   *db.Queries
	dataDir   string
	projectID string
	runID     string

	// Sub-evaluators
	toolEval       *ToolEffectivenessEvaluator
	templateEval   *TemplateEffectivenessEvaluator
	efficiencyEval *EfficiencyEvaluator
	findingEval    *FindingQualityEvaluator
	ruleEngine     *RuleEngine
	trendAnalyzer  *TrendAnalyzer
	reportGen      *ReportGenerator
}

// NewEvaluator creates a new evaluator instance.
func NewEvaluator(queries *db.Queries, dataDir, projectID, runID string) *Evaluator {
	return &Evaluator{
		queries:        queries,
		dataDir:        dataDir,
		projectID:      projectID,
		runID:          runID,
		toolEval:       NewToolEffectivenessEvaluator(queries),
		templateEval:   NewTemplateEffectivenessEvaluator(queries),
		efficiencyEval: NewEfficiencyEvaluator(queries),
		findingEval:    NewFindingQualityEvaluator(queries),
		ruleEngine:     NewRuleEngine(),
		trendAnalyzer:  NewTrendAnalyzer(queries),
		reportGen:      NewReportGenerator(),
	}
}

// Evaluate executes the full evaluation pipeline.
func (e *Evaluator) Evaluate(ctx context.Context) (*EvaluationReport, error) {
	// 1. Collect metrics
	metrics, err := e.collectMetrics()
	if err != nil {
		return nil, err
	}

	// 2. Run rule engine
	issues := e.ruleEngine.Evaluate(metrics)

	// 3. Analyze trends
	history, err := e.trendAnalyzer.QueryHistory(ctx, e.projectID, 10)
	if err != nil {
		// Non-fatal: continue without trends
		history = []*TrendData{}
	}
	trends := e.trendAnalyzer.Analyze(history, metrics)

	// 4. Generate report
	content := e.reportGen.Generate(metrics, issues, trends)

	// 5. Save report
	reportPath := e.GetReportPath()
	if err := e.saveReport(reportPath, content); err != nil {
		return nil, err
	}

	return &EvaluationReport{
		Path:        reportPath,
		GeneratedAt: time.Now(),
		Metrics:     metrics,
		Issues:      issues,
		Trends:      trends,
		Content:     content,
	}, nil
}

// GetReportPath returns the path where the evaluation report should be saved.
func (e *Evaluator) GetReportPath() string {
	return filepath.Join(e.dataDir, "projects", e.projectID, "reports",
		e.runID+"_evaluation.md")
}

// collectMetrics collects all metrics from the database.
func (e *Evaluator) collectMetrics() (*ScanMetrics, error) {
	metrics := &ScanMetrics{
		RunID:             e.runID,
		ProjectID:         e.projectID,
		FindingsBySeverity: make(map[string]int),
		FindingsByStatus:   make(map[string]int),
		StageDurations:    make(map[string]time.Duration),
		StageStatuses:     make(map[string]string),
	}

	// Get run info
	run, err := e.queries.GetPipelineRun(e.runID)
	if err != nil {
		return nil, err
	}
	if run != nil {
		metrics.RunName = run.Name
		if run.StartedAt != nil {
			metrics.StartedAt = *run.StartedAt
		}
		if run.CompletedAt != nil {
			metrics.CompletedAt = *run.CompletedAt
			metrics.TotalDuration = run.CompletedAt.Sub(*run.StartedAt)
		}
	}

	// Collect tool effectiveness
	toolStats, err := e.toolEval.Evaluate(e.runID)
	if err != nil {
		return nil, err
	}
	metrics.ToolStats = toolStats

	// Collect template effectiveness
	templateStats, err := e.templateEval.EvaluateTemplateStats(e.runID)
	if err != nil {
		return nil, err
	}
	metrics.TemplateStats = templateStats

	// Collect dictionary stats
	dictStats, err := e.templateEval.EvaluateDictionaryStats(e.runID)
	if err != nil {
		return nil, err
	}
	metrics.DictionaryStats = dictStats

	// Collect efficiency
	efficiency, err := e.efficiencyEval.Evaluate(e.runID)
	if err != nil {
		return nil, err
	}
	metrics.TotalDuration = efficiency.TotalDuration
	metrics.StageDurations = efficiency.StageDurations
	metrics.StageStatuses = efficiency.StageStatuses

	// Collect finding quality
	findingQuality, err := e.findingEval.Evaluate(e.runID)
	if err != nil {
		return nil, err
	}
	metrics.FindingsBySeverity = findingQuality.FindingsBySeverity
	metrics.FindingsByStatus = findingQuality.FindingsByStatus
	metrics.AvgConfidence = findingQuality.AvgConfidence
	metrics.UnlinkedFindings = findingQuality.UnlinkedFindings
	metrics.TotalFindings = findingQuality.TotalFindings

	return metrics, nil
}

// saveReport saves the report content to a file.
func (e *Evaluator) saveReport(path, content string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(path, []byte(content), 0644)
}
```

- [ ] **Step 2: 验证编译**

Run: `cd /Users/kun/DEV/Anchor && go build ./internal/evaluator/`
Expected: 编译成功

- [ ] **Step 3: Commit**

```bash
git add internal/evaluator/evaluator.go
git commit -m "feat(evaluator): add evaluator entry point"
```

---

## Task 11: 创建 API 端点

**Files:**
- Create: `internal/api/evaluation_handlers.go`
- Modify: `internal/api/server.go` (注册路由)

- [ ] **Step 1: 创建 evaluation_handlers.go**

```go
package api

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/P0m32Kun/Anchor/internal/evaluator"
)

// handleGetEvaluation returns the evaluation report for a run.
// GET /projects/{id}/runs/{runId}/evaluation
func (s *Server) handleGetEvaluation(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	runID := r.PathValue("runId")

	reportPath := filepath.Join(s.dataDir, "projects", projectID, "reports",
		runID+"_evaluation.md")

	// Check if report exists
	if _, err := os.Stat(reportPath); os.IsNotExist(err) {
		writeError(w, http.StatusNotFound, "评估报告不存在，可能扫描尚未完成")
		return
	}

	content, err := os.ReadFile(reportPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"content": string(content),
	})
}

// handleRetryEvaluation manually triggers evaluation for a run.
// POST /projects/{id}/runs/{runId}/evaluation/retry
func (s *Server) handleRetryEvaluation(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	runID := r.PathValue("runId")

	go func() {
		eval := evaluator.NewEvaluator(s.queries, s.dataDir, projectID, runID)
		_, err := eval.Evaluate(context.Background())
		if err != nil {
			log.Printf("[evaluation] manual retry failed for run %s: %v", runID, err)
		}
	}()

	writeJSON(w, http.StatusAccepted, map[string]string{
		"message": "评估已触发，稍后刷新查看结果",
	})
}

// handleListEvaluations returns a list of evaluation reports for a project.
// GET /projects/{id}/evaluations
func (s *Server) handleListEvaluations(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")

	reportsDir := filepath.Join(s.dataDir, "projects", projectID, "reports")

	// Check if directory exists
	if _, err := os.Stat(reportsDir); os.IsNotExist(err) {
		writeJSON(w, http.StatusOK, []map[string]string{})
		return
	}

	entries, err := os.ReadDir(reportsDir)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var evaluations []map[string]string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), "_evaluation.md") {
			runID := strings.TrimSuffix(entry.Name(), "_evaluation.md")
			evaluations = append(evaluations, map[string]string{
				"run_id":     runID,
				"filename":   entry.Name(),
				"created_at": entry.ModTime().Format("2006-01-02T15:04:05Z"),
			})
		}
	}

	writeJSON(w, http.StatusOK, evaluations)
}
```

- [ ] **Step 2: 在 server.go 注册路由**

在 `server.go` 的路由注册部分（约第 308 行附近）添加：

```go
	// Evaluation
	mux.Handle("GET /projects/{id}/runs/{runId}/evaluation", auth(http.HandlerFunc(s.handleGetEvaluation)))
	mux.Handle("POST /projects/{id}/runs/{runId}/evaluation/retry", auth(http.HandlerFunc(s.handleRetryEvaluation)))
	mux.Handle("GET /projects/{id}/evaluations", auth(http.HandlerFunc(s.handleListEvaluations)))
```

- [ ] **Step 3: 验证编译**

Run: `cd /Users/kun/DEV/Anchor && go build ./internal/api/`
Expected: 编译成功

- [ ] **Step 4: Commit**

```bash
git add internal/api/evaluation_handlers.go internal/api/server.go
git commit -m "feat(api): add evaluation endpoints"
```

---

## Task 12: 集成到扫描完成流程

**Files:**
- Modify: `internal/api/run_handlers.go` 或相关文件

- [ ] **Step 1: 找到扫描完成的处理逻辑**

搜索扫描状态更新为 completed/failed 的代码位置。

- [ ] **Step 2: 添加异步评估触发**

在扫描状态更新为 completed/failed 后添加：

```go
// Trigger evaluation asynchronously
go func() {
    eval := evaluator.NewEvaluator(s.queries, s.dataDir, projectID, runID)
    _, err := eval.Evaluate(context.Background())
    if err != nil {
        log.Printf("[evaluation] failed for run %s: %v", runID, err)
    } else {
        log.Printf("[evaluation] report generated for run %s", runID)
    }
}()
```

- [ ] **Step 3: 验证编译**

Run: `cd /Users/kun/DEV/Anchor && go build ./...`
Expected: 编译成功

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "feat: integrate evaluation into scan completion flow"
```

---

## Task 13: 编写单元测试

**Files:**
- Create: `internal/evaluator/evaluator_test.go`
- Create: `internal/evaluator/rules_test.go`

- [ ] **Step 1: 创建 rules_test.go**

```go
package evaluator

import (
	"testing"
	"time"
)

func TestRuleEngine_ToolReliabilityLow(t *testing.T) {
	engine := NewRuleEngine()

	// Test case: tool with low success rate
	metrics := &ScanMetrics{
		ToolStats: map[string]*ToolStat{
			"subfinder": {
				ToolName:   "subfinder",
				TotalCalls: 10,
				SuccessRate: 0.6, // 60% success rate
			},
		},
	}

	issues := engine.Evaluate(metrics)

	found := false
	for _, issue := range issues {
		if issue.RuleID == "tool_reliability_low" {
			found = true
			if issue.Severity != "high" {
				t.Errorf("Expected severity 'high', got '%s'", issue.Severity)
			}
		}
	}

	if !found {
		t.Error("Expected tool_reliability_low issue to be triggered")
	}
}

func TestRuleEngine_ToolReliabilityHigh(t *testing.T) {
	engine := NewRuleEngine()

	// Test case: tool with high success rate (should not trigger)
	metrics := &ScanMetrics{
		ToolStats: map[string]*ToolStat{
			"subfinder": {
				ToolName:   "subfinder",
				TotalCalls: 10,
				SuccessRate: 0.9, // 90% success rate
			},
		},
	}

	issues := engine.Evaluate(metrics)

	for _, issue := range issues {
		if issue.RuleID == "tool_reliability_low" {
			t.Error("tool_reliability_low should not be triggered for 90% success rate")
		}
	}
}

func TestCalculateTrend(t *testing.T) {
	tests := []struct {
		name      string
		values    []float64
		threshold float64
		expected  TrendDirection
	}{
		{
			name:      "upward trend",
			values:    []float64{10, 20, 30, 40, 50},
			threshold: 0.1,
			expected:  TrendUp,
		},
		{
			name:      "downward trend",
			values:    []float64{50, 40, 30, 20, 10},
			threshold: 0.1,
			expected:  TrendDown,
		},
		{
			name:      "stable trend",
			values:    []float64{30, 30, 30, 30, 30},
			threshold: 0.1,
			expected:  TrendStable,
		},
		{
			name:      "single value",
			values:    []float64{30},
			threshold: 0.1,
			expected:  TrendStable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateTrend(tt.values, tt.threshold)
			if result != tt.expected {
				t.Errorf("CalculateTrend() = %v, want %v", result, tt.expected)
			}
		})
	}
}
```

- [ ] **Step 2: 运行测试**

Run: `cd /Users/kun/DEV/Anchor && go test ./internal/evaluator/ -v`
Expected: 所有测试通过

- [ ] **Step 3: Commit**

```bash
git add internal/evaluator/rules_test.go
git commit -m "test(evaluator): add rule engine unit tests"
```

---

## Task 14: 端到端测试

- [ ] **Step 1: 启动服务**

Run: `cd /Users/kun/DEV/Anchor && go run ./cmd/server/`
Expected: 服务启动成功

- [ ] **Step 2: 创建测试项目和运行扫描**

通过 API 或 UI 创建一个项目并运行一次扫描。

- [ ] **Step 3: 检查评估报告**

扫描完成后，检查评估报告是否生成：

Run: `ls -la data/projects/{project_id}/reports/`
Expected: 看到 `{run_id}_evaluation.md` 文件

- [ ] **Step 4: 通过 API 获取评估报告**

Run: `curl http://localhost:8080/api/projects/{id}/runs/{runId}/evaluation`
Expected: 返回 JSON 包含 evaluation report content

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "feat: complete scan quality evaluator implementation"
```

---

## 实现计划总结

**总任务数**：14 个任务

**关键文件**：
- `internal/evaluator/` - 10 个文件
- `internal/api/evaluation_handlers.go` - API 端点
- `internal/db/queries_*.go` - 新增查询方法

**验收标准**：
1. 扫描完成后自动生成评估报告
2. 评估报告包含工具效果、模板效果、执行效率、漏洞质量分析
3. 规则引擎能识别问题并生成优化建议
4. 趋势分析能与历史数据对比
5. API 端点可访问评估报告
6. 所有单元测试通过
