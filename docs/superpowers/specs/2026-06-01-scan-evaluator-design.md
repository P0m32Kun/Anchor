---
status: approved
source_of_truth: false
owner: kun
last_updated: 2026-06-17
scope: scan-evaluator
reason: "正文确认已批准，补充结构化 frontmatter"
---

# 扫描质量评估系统设计文档

> **创建日期**：2026-06-01
> **状态**：已批准
> **作者**：AI Assistant

---

## 一、背景与目标

### 1.1 背景

当前 Anchor 系统已经具备完整的资产驱动扫描能力，包括：
- 资产发现与管理
- 多工具协同扫描（subfinder、naabu、httpx、nuclei 等）
- 阶段状态追踪（pipeline_run_stages）
- 漏洞发现与管理

但缺少一个系统性的"自评估"机制，无法回答以下问题：
- 哪些工具效率高/低？
- 哪些模板/字典经常被命中？
- 扫描执行效率如何？
- 漏洞质量如何（误报率、置信度）？

### 1.2 目标

建立一个自动化的扫描质量评估系统，实现：
1. **工具效果评估**：识别高效/低效工具
2. **模板/字典效果评估**：识别热门/冷门模板和字典
3. **执行效率评估**：识别瓶颈阶段
4. **漏洞质量评估**：评估发现质量
5. **规则引擎**：基于规则识别问题并生成优化建议
6. **趋势分析**：与历史数据对比，识别长期趋势

---

## 二、整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                    ScanEngine（资产驱动）                    │
│  资产发现 → 优先级队列 → 并行执行 → 阶段聚合                │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼ (扫描完成，异步触发)
┌─────────────────────────────────────────────────────────────┐
│                    ScanEvaluator（独立服务）                 │
│                                                             │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
│  │ 工具效果    │  │ 模板/字典   │  │ 执行效率    │         │
│  │ 评估器      │  │ 评估器      │  │ 评估器      │         │
│  └─────────────┘  └─────────────┘  └─────────────┘         │
│            │              │              │                  │
│            └──────────────┼──────────────┘                  │
│                           ▼                                 │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              规则引擎 (RuleEngine)                   │   │
│  │  - 识别问题模式                                      │   │
│  │  - 生成优化建议                                      │   │
│  └─────────────────────────────────────────────────────┘   │
│                           │                                 │
│                           ▼                                 │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              趋势分析 (TrendAnalyzer)                │   │
│  │  - 与最近 N 次扫描对比                              │   │
│  │  - 识别长期趋势                                      │   │
│  └─────────────────────────────────────────────────────┘   │
│                           │                                 │
│                           ▼                                 │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              报告生成 (ReportGenerator)              │   │
│  │  - Markdown 格式评估报告                            │   │
│  │  - 存储到项目目录                                    │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

---

## 三、评估维度详细设计

### 3.1 工具效果评估 (ToolEffectivenessEvaluator)

**评估指标**：

| 指标 | 计算方式 | 说明 |
|------|---------|------|
| 工具调用成功率 | `成功次数 / 总调用次数 × 100%` | 按工具分组 |
| 工具平均耗时 | `AVG(completed_at - started_at)` | 按工具分组 |
| 工具产出效率 | `产生的资产数 / 运行时间(秒)` | 每秒产出资产数 |
| 工具无效运行率 | `运行但无产出的次数 / 总调用次数` | 识别低效工具 |
| 工具失败原因分布 | `GROUP BY error_message` | 识别常见失败模式 |

**数据来源**：
```go
// 从 scan_work_items 表查询
- 工具调用统计：GROUP BY tool, status
- 运行耗时：AVG(completed_at - started_at) GROUP BY tool
- 失败原因：WHERE status = 'failed' GROUP BY tool, error_message

// 从 tool_invocations 表查询
- 工具版本信息：version, command_redacted
- 执行结果：exit_code
```

---

### 3.2 模板/字典效果评估 (TemplateDictionaryEvaluator)

**评估指标**：

| 指标 | 计算方式 | 说明 |
|------|---------|------|
| 模板命中率 | `匹配该模板的漏洞数` | 按模板排序，识别热门模板 |
| 模板有效性 | `该模板产生的 confirmed 漏洞 / 总命中数` | 识别高质量模板 |
| 字典命中率 | `字典匹配的请求数 / 总请求数` | 评估字典质量 |
| 字典覆盖范围 | `命中的唯一路径数` | 评估字典广度 |
| 未使用模板 | `存在但从未命中的模板` | 识别冗余模板 |

**数据来源**：
```go
// 从 findings 表查询
- 模板命中：GROUP BY source_tool, source_rule_id
- 模板有效性：JOIN finding_templates GROUP BY template_id, status

// 从 finding_templates 表查询
- 模板总数：COUNT(*) WHERE enabled = true
- 内置 vs 自定义：GROUP BY is_builtin
```

---

### 3.3 执行效率评估 (EfficiencyEvaluator)

**评估指标**：

| 指标 | 计算方式 | 说明 |
|------|---------|------|
| 总耗时 | `completed_at - started_at` | 整体执行时间 |
| 阶段耗时分布 | `各阶段 completed_at - started_at` | 识别瓶颈阶段 |
| 阶段失败率 | `failed 阶段数 / 总阶段数` | 评估稳定性 |
| 资产处理速度 | `资产数 / 总耗时(秒)` | 每秒处理资产数 |
| 阶段间等待时间 | `下阶段 started_at - 上阶段 completed_at` | 识别调度延迟 |

**数据来源**：
```go
// 从 pipeline_runs 表查询
- 总耗时：started_at, completed_at
- 引擎状态：engine_state

// 从 pipeline_run_stages 表查询
- 各阶段耗时：started_at, completed_at
- 阶段状态：status (running/completed/failed)
```

---

### 3.4 漏洞质量评估 (FindingQualityEvaluator)

**评估指标**：

| 指标 | 计算方式 | 说明 |
|------|---------|------|
| 漏洞严重程度分布 | `COUNT(*) GROUP BY severity` | 识别风险集中度 |
| 置信度分布 | `AVG(confidence) GROUP BY severity` | 评估结果质量 |
| 人工处理率 | `已处理漏洞 / 总漏洞` | 评估人工介入程度 |
| 误报率 | `false_positive / 已处理漏洞` | 需要人工标注后计算 |
| 确认率 | `confirmed / 已处理漏洞` | 评估发现有效性 |
| 漏洞关联资产覆盖率 | `有资产关联的漏洞 / 总漏洞` | 评估关联完整性 |

**数据来源**：
```go
// 从 findings 表查询
- 漏洞统计：GROUP BY severity, status
- 置信度分布：AVG(confidence) GROUP BY severity
- 状态分布：GROUP BY status

// 从 findings + assets 表联合查询
- 资产关联：WHERE asset_id IS NOT NULL
```

---

## 四、规则引擎设计

### 4.1 规则分类

| 类别 | 触发条件 | 严重程度 | 示例建议 |
|------|---------|---------|---------|
| **工具可靠性** | 工具成功率 < 80% | 高 | "subfinder 成功率仅 65%，建议检查 API 配额或网络连接" |
| **工具效率** | 工具平均耗时 > 阈值 | 中 | "naabu 平均耗时 45 分钟，考虑减少端口范围或增加超时" |
| **工具产出** | 工具无效运行率 > 30% | 中 | "httpx 有 40% 的运行无产出，建议优化目标筛选逻辑" |
| **模板质量** | 模板命中率 < 5% 且启用 > 30 天 | 低 | "模板 X 启用 60 天但从未命中，考虑禁用或更新" |
| **字典效果** | 字典命中率 < 10% | 中 | "字典 Y 命中率仅 8%，考虑更换或扩充字典" |
| **执行瓶颈** | 某阶段耗时占比 > 50% | 高 | "nuclei 阶段占总耗时 65%，建议分批执行或增加并行度" |
| **阶段失败** | 阶段失败率 > 20% | 高 | "portscan 阶段失败率 25%，检查目标可达性" |
| **漏洞质量** | 置信度 < 60% 的漏洞占比 > 40% | 中 | "40% 漏洞置信度低于 60%，建议优化检测规则" |
| **关联完整性** | 无资产关联的漏洞 > 30% | 中 | "30% 漏洞未关联资产，检查资产解析逻辑" |

---

### 4.2 规则结构

```go
// Rule 定义一条评估规则
type Rule struct {
    ID          string
    Category    string        // 工具可靠性/工具效率/模板质量/...
    Name        string        // 规则名称
    Description string        // 规则描述
    Condition   func(metrics *ScanMetrics) bool  // 触发条件
    Severity    string        // high/medium/low
    Suggestion  func(metrics *ScanMetrics) string // 生成建议
}
```

---

## 五、趋势分析设计

### 5.1 分析维度

| 维度 | 对比内容 | 趋势指标 |
|------|---------|---------|
| **工具效果趋势** | 最近 N 次扫描的工具成功率、耗时 | 上升/下降/稳定 |
| **漏洞发现趋势** | 最近 N 次扫描的漏洞数量和严重程度分布 | 增长/减少/稳定 |
| **执行效率趋势** | 最近 N 次扫描的总耗时和阶段耗时 | 优化/退化/稳定 |
| **模板命中趋势** | 最近 N 次扫描的模板命中率变化 | 热门/冷门/稳定 |

---

### 5.2 趋势计算

使用简单线性回归计算趋势方向：

```go
// CalculateTrend 计算趋势方向
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
    
    slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)
    
    if slope > threshold {
        return TrendUp
    } else if slope < -threshold {
        return TrendDown
    }
    return TrendStable
}
```

---

### 5.3 显著变化检测

检测两次扫描之间的显著变化（如成功率下降超过 10%）：

```go
// DetectSignificantChanges 检测显著变化
func DetectSignificantChanges(current, previous *TrendData) []Change {
    var changes []Change
    
    // 工具成功率变化
    for tool, currStat := range current.ToolStats {
        prevStat, ok := previous.ToolStats[tool]
        if !ok {
            continue
        }
        
        change := currStat.SuccessRate - prevStat.SuccessRate
        if change < -0.1 {  // 下降超过 10%
            changes = append(changes, Change{
                Dimension:   "tool_efficiency",
                Entity:      tool,
                Description: fmt.Sprintf("%s 成功率从 %.0f%% 下降到 %.0f%%", 
                    tool, prevStat.SuccessRate*100, currStat.SuccessRate*100),
                Severity:    "degradation",
            })
        }
    }
    
    return changes
}
```

---

## 六、报告生成设计

### 6.1 报告结构

评估报告包含以下章节：

1. **执行摘要**：总体评估结果和趋势概览
2. **工具效果分析**：工具调用统计、趋势、问题识别
3. **模板/字典效果分析**：热门模板、字典效果、问题识别
4. **执行效率分析**：阶段耗时分布、效率趋势、问题识别
5. **漏洞质量分析**：严重程度分布、处理状态、质量指标
6. **优化建议**：按优先级排列的建议列表
7. **趋势分析**：显著变化和长期趋势
8. **附录**：评估规则说明和数据来源

---

### 6.2 文件存储

```
data/
└── projects/
    └── {project_id}/
        └── reports/
            ├── {run_id}_report.md          # 漏洞报告（已有）
            └── {run_id}_evaluation.md      # 评估报告（新增）
```

---

## 七、API 设计

### 7.1 API 端点

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/projects/{id}/runs/{runId}/evaluation` | 获取评估报告内容 |
| `POST` | `/projects/{id}/runs/{runId}/evaluation/retry` | 手动触发重新评估 |
| `GET` | `/projects/{id}/evaluations` | 获取项目所有评估报告列表 |

---

### 7.2 触发机制

**自动触发**：扫描完成后由 API 层异步触发评估

```go
// 当扫描状态变为 completed/failed 时，异步触发评估
func (s *Server) handleScanCompletion(projectID, runID string) {
    go func() {
        evaluator := NewEvaluator(s.queries, s.dataDir, projectID, runID)
        report, err := evaluator.Evaluate(context.Background())
        if err != nil {
            log.Printf("[evaluation] failed for run %s: %v", runID, err)
            return
        }
        log.Printf("[evaluation] report saved: %s", report.Path)
    }()
}
```

**手动触发**：用户可通过 API 手动触发重新评估

---

## 八、目录结构

```
internal/
└── evaluator/
    ├── evaluator.go              # 评估器入口，协调各子组件
    ├── coverage.go               # 工具效果评估
    ├── detection.go              # 模板/字典效果评估
    ├── efficiency.go             # 执行效率评估
    ├── finding_quality.go        # 漏洞质量评估
    ├── rules.go                  # 规则引擎核心
    ├── rules_definitions.go      # 预定义规则列表
    ├── trend.go                  # 趋势分析
    ├── report.go                 # 报告生成器
    ├── metrics.go                # 指标数据结构
    ├── evaluator_test.go         # 评估器单元测试
    ├── rules_test.go             # 规则引擎测试
    └── trend_test.go             # 趋势分析测试
```

---

## 九、实现计划

### Phase 1：核心框架 + 数据采集（2-3 小时）

**目标**：搭建评估器骨架，实现从数据库采集指标

**任务**：
- [ ] 创建 `internal/evaluator/` 包
- [ ] 实现 `metrics.go` - 指标数据结构
- [ ] 实现 `evaluator.go` - 评估器入口和指标采集
- [ ] 实现 `coverage.go` - 工具效果评估
- [ ] 实现 `detection.go` - 模板/字典效果评估
- [ ] 实现 `efficiency.go` - 执行效率评估
- [ ] 实现 `finding_quality.go` - 漏洞质量评估
- [ ] 编写单元测试

**验收标准**：
- 能够从数据库采集所有指标数据
- 各维度评估器独立可测
- 单元测试覆盖核心逻辑

---

### Phase 2：规则引擎 + 报告生成（2-3 小时）

**目标**：实现规则引擎识别问题，生成 Markdown 评估报告

**任务**：
- [ ] 实现 `rules.go` - 规则引擎核心
- [ ] 实现 `rules_definitions.go` - 预定义规则（9 条规则）
- [ ] 实现 `report.go` - 报告生成器
- [ ] 实现 `evaluator.go` 中的 `Evaluate()` 方法
- [ ] 编写规则引擎测试
- [ ] 编写报告生成测试

**验收标准**：
- 规则引擎能正确识别问题
- 生成的报告格式正确、内容完整
- 规则可独立测试

---

### Phase 3：趋势分析（1-2 小时）

**目标**：实现与历史数据对比的趋势分析

**任务**：
- [ ] 实现 `trend.go` - 趋势分析器
- [ ] 实现历史数据查询（最近 N 次扫描）
- [ ] 实现趋势计算逻辑（线性回归）
- [ ] 实现显著变化检测
- [ ] 编写趋势分析测试

**验收标准**：
- 能够查询历史扫描数据
- 正确计算趋势方向
- 识别显著变化

---

### Phase 4：API 集成 + 触发机制（1 小时）

**目标**：通过 API 端点访问评估报告，扫描完成后自动触发

**任务**：
- [ ] 实现 `api/evaluation_handlers.go` - API 端点
- [ ] 在 `server.go` 注册新路由
- [ ] 实现扫描完成后的异步触发逻辑
- [ ] 实现手动重试端点
- [ ] 编写 API 测试

**验收标准**：
- `GET /projects/{id}/runs/{runId}/evaluation` 返回评估报告
- `POST /projects/{id}/runs/{runId}/evaluation/retry` 手动触发评估
- 扫描完成后自动生成评估报告

---

### Phase 5：前端展示（可选，2-3 小时）

**目标**：在前端展示评估报告

**任务**：
- [ ] 在 `api.ts` 添加评估 API 方法
- [ ] 在扫描详情页添加"评估报告" Tab
- [ ] 使用 Markdown 渲染器展示报告
- [ ] 添加手动重试按钮

**验收标准**：
- 用户可在扫描详情页查看评估报告
- 报告格式正确、可读性好
- 支持手动触发重新评估

---

## 十、未来扩展

### 10.1 大模型增强（Phase 2）

在规则引擎基础上，引入大模型增强评估能力：

**分层设计**：
- **第一层：规则引擎**：处理确定性问题
- **第二层：大模型**：处理需要理解上下文的问题

**降级策略**：
- 大模型调用失败时，降级到纯规则引擎
- 可配置是否启用大模型（默认关闭）
- 支持本地模型（Ollama）和云端模型（OpenAI/Claude）

---

### 10.2 自定义规则

支持用户自定义评估规则：

```go
// 用户可通过配置文件定义自定义规则
type CustomRule struct {
    ID          string `json:"id"`
    Category    string `json:"category"`
    Name        string `json:"name"`
    Condition   string `json:"condition"`  // 表达式
    Severity    string `json:"severity"`
    Suggestion  string `json:"suggestion"`
}
```

---

### 10.3 评估报告订阅

支持订阅评估报告，扫描完成后自动通知：

- 邮件通知
- Webhook 回调
- 消息队列推送

---

## 附录

### A. 数据来源表

| 表名 | 用途 |
|------|------|
| `pipeline_runs` | 扫描运行记录 |
| `pipeline_run_stages` | 阶段执行记录 |
| `scan_work_items` | 工作项记录（资产 × 动作） |
| `findings` | 漏洞记录 |
| `finding_templates` | 漏洞模板 |
| `tool_invocations` | 工具调用记录 |

---

### B. 评估指标汇总

| 维度 | 指标 | 数据来源 |
|------|------|---------|
| 工具效果 | 调用成功率、平均耗时、产出效率、无效运行率 | scan_work_items |
| 模板效果 | 命中率、有效性 | findings, finding_templates |
| 字典效果 | 命中率、覆盖范围 | scan_work_items (ffuf) |
| 执行效率 | 总耗时、阶段耗时、失败率 | pipeline_runs, pipeline_run_stages |
| 漏洞质量 | 严重程度分布、置信度、处理率、误报率 | findings |

---

### C. 规则列表

| ID | 类别 | 名称 | 严重程度 |
|----|------|------|---------|
| tool_reliability_low | 工具可靠性 | 工具成功率过低 | high |
| tool_efficiency_slow | 工具效率 | 工具平均耗时过长 | medium |
| tool_output_low | 工具产出 | 工具无效运行率过高 | medium |
| template_unused | 模板质量 | 模板长期未命中 | low |
| dictionary_effective_low | 字典效果 | 字典命中率过低 | medium |
| stage_bottleneck | 执行瓶颈 | 某阶段耗时占比过高 | high |
| stage_failure_high | 阶段失败 | 阶段失败率过高 | high |
| finding_confidence_low | 漏洞质量 | 低置信度漏洞占比过高 | medium |
| finding_unlinked | 关联完整性 | 未关联资产漏洞占比过高 | medium |
