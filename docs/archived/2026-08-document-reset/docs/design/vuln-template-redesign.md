---
status: in_review
source_of_truth: false
owner: kun
last_updated: 2026-05-19
scope: vulnerability-template-knowledge-base
verification: pending_implementation
---

# 漏洞模板模块重设计 — 漏洞辞典 (Vulnerability Dictionary)

> Status: In Review
> Audience: Claude Code implementation agent
> Default rule: 提案,不是当前架构基线。当前基线仍以 `docs/current/architecture.md` 为准。

## 1. 背景

Anchor 当前已有「漏洞模板」(`finding_templates`)模块。设计初衷是:扫描工具(主要是 nuclei)输出的 finding 缺少结构化的中文漏洞名、详细描述与修复建议;直接拿这些工具原始字段写报告读者会看到不连贯的英文术语 + 一句话摘要 + 空白的修复建议。模板模块的目的是把工具原始 finding **翻译**成符合用户笔下漏洞报告口径的条目。

实际跑了一段时间后,这个模块没达到初衷的几个原因:

1. **生效面太窄** — 实际产 finding 的工具只有 nuclei(httpx 只发现 web endpoint;ffuf/naabu/dnsx/sqlmap/hydra 路径里都没有 finding 构造)。匹配走 `source_rule_id → matched_template → title` 三级精确串,其中 `matched_template` 字段**从未在 finding 创建路径里被赋值**(死字段)。
2. **匹配粒度死板** — 一个模板对应一个精确的 match_key。nuclei 的 `cve-XXXX-name` 细分变体太多,SQL 注入这类需要归并的漏洞,要为每个变体写一条模板。
3. **内置库为空** — `docs/templates/vuln-templates.json` 是 `[]`,虽然内置-上游同步机制(`is_builtin` / `user_modified` / `builtin_payload`)写好了,但没东西可同步。
4. **无可观测性** — 模板维护页和报告生成页之间没有上下文关联。用户改了模板,得下载 md 文件后肉眼对比才知道哪些 finding 命中了。
5. **修复建议只能来自模板** — finding 创建路径从不填 remediation 字段。前端 UI 并未强调这点,用户不知道修复建议是「必须靠模板提供」的。
6. **报告渲染冗余** — 同种漏洞的多个 finding 各自一节,导致同样的描述/修复重复出现 N 次。

## 2. 产品定位

**这个模块的定位是「漏洞辞典(Vulnerability Dictionary)」。** 用户笔下手上有一份 70+ 常见漏洞的内部手册,他希望:

- 把这 70+ 漏洞的中文名、描述、修复建议沉淀进系统;
- 扫描结果出来后,常见漏洞自动套用辞典文案、生成可直接交付的 markdown;
- 长尾(只出现一次、未编入辞典的)漏洞**降级为原始数据展示**,跟命中辞典的漏洞**混排进同一份报告**,而不是被排除或被堆进附录;
- 慢慢扩充辞典(每次扫描发现的新漏洞,可一键加入),不强求一次写完。

## 3. 关键决策

| 维度 | 决策 |
|---|---|
| **范围** | MVP 推倒重来 — 重新定义这个模块要解决的问题 |
| **产品定位** | 漏洞辞典(知识库为中心,不是字段覆盖工具) |
| **粒度** | 70+ 类型级为主;支持单点漏洞(如 MS17-010) 1:1、聚合漏洞(如 SQL 注入) 1:多 |
| **未命中表现** | 同位混排 — 与命中条目穿插进入报告正文,使用 finding 原始字段 |
| **匹配机制** | 词条挂多个匹配键(显式枚举的字符串数组,不用正则) |
| **报告渲染** | 按词条聚合 — 一个词条一节,受影响资产以表格列出 |
| **词条字段** | 维持现有 4 件套(`title` / `severity` / `summary` / `remediation`) |
| **词条来源** | 双轨(仓库 `vuln-templates.json` + 个人 DB 新增) |
| **绑定时机** | 报告生成时查,零侵入,不写 finding 表,不支持 finding 级 override |
| **报告导出** | 只保留 Markdown — 删 HTML、删报告 JSON |
| **辞典 JSON 导出** | 保留 — 用于把当前 DB 词条贴回仓库 `vuln-templates.json` 形成团队共享 |
| **RunsPage 报告流程** | 删除整套异步持久化报告(handleCreateReport/HTML 文件/SSE report_progress 等),改为「查看报告」按钮跳转到 ReportsPage |

## 4. 数据模型

### 4.1 `FindingTemplate` 字段变更

```go
// internal/models/finding_template.go
type FindingTemplate struct {
    ID             string    `json:"id" db:"id"`
    SourceTool     string    `json:"source_tool" db:"source_tool"`
    MatchKeys      []string  `json:"match_keys" db:"-"`              // 新增,内存中以 slice 形式
    MatchKeysJSON  string    `json:"-"            db:"match_keys"`   // DB 存 JSON 编码字符串
    Title          string    `json:"title" db:"title"`
    Severity       string    `json:"severity" db:"severity"`
    Summary        string    `json:"summary" db:"summary"`
    Remediation    string    `json:"remediation" db:"remediation"`
    Enabled        bool      `json:"enabled" db:"enabled"`
    IsBuiltin      bool      `json:"is_builtin" db:"is_builtin"`
    UserModified   bool      `json:"user_modified" db:"user_modified"`
    BuiltinPayload string    `json:"builtin_payload" db:"builtin_payload"`
    CreatedAt      time.Time `json:"created_at" db:"created_at"`
    UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}
```

**变更细节:**

- 删除原 `MatchKey string` 字段。
- 新增 `MatchKeys []string` 内存字段 + `MatchKeysJSON string` 存储字段。DB 列名 `match_keys`,类型 `TEXT`,内容为 JSON 编码的字符串数组(例:`["sqli-blind","sqli-error-based"]`)。
- 序列化辅助:在 `scanFindingTemplate` 里把 `MatchKeysJSON` 反序列化进 `MatchKeys`;在 `CreateFindingTemplate` / `UpdateFindingTemplate` 里序列化 `MatchKeys` 回 `MatchKeysJSON`。
- 上游 JSON 形态(`SeedFindingTemplate`)对应字段也改为 `match_keys []string`。

### 4.2 Schema migration

新增 migration `vN.go`(`N` = 当前最大版本 + 1):

```sql
-- Step 1: 添加新列
ALTER TABLE finding_templates ADD COLUMN match_keys TEXT NOT NULL DEFAULT '[]';

-- Step 2: 把已有 match_key 的单值迁移成单元素数组
UPDATE finding_templates SET match_keys = json_array(match_key) WHERE match_key != '';

-- Step 3: 老 match_key 列保留一个 release 以兼容回滚;下一版本(vN+1)中删除。
```

**老 `match_key` 列**:本次 migration 不删,留作回滚兜底。下一版本的 migration 单独清理。

**唯一索引**:原本对 `(source_tool, match_key)` 的唯一索引需要重做。新方案下唯一性约束在「应用层」检查(创建/更新时,不允许同一 `source_tool` 下的某个 key 出现在多条词条里)。

### 4.3 `vuln-templates.json` 新 schema

```json
[
  {
    "source_tool": "nuclei",
    "match_keys": ["ms17-010-smbtouch-poc", "smb-eternalblue"],
    "title": "MS17-010 永恒之蓝远程代码执行漏洞",
    "severity": "critical",
    "summary": "...",
    "remediation": "...",
    "enabled": true
  },
  {
    "source_tool": "nuclei",
    "match_keys": ["sqli-blind", "sqli-error-based", "generic-sqli"],
    "title": "SQL 注入漏洞",
    "severity": "high",
    "summary": "...",
    "remediation": "..."
  }
]
```

`SeedFindingTemplate` Go 结构体同步把 `match_key string` 改为 `match_keys []string`。`SyncFindingTemplatesFromFile` 比对逻辑不变(按 `source_tool` + `match_keys` 内容比对),内置-上游同步行为(insert / update / preserve / delete / skip)保持现有契约。

### 4.4 死字段处理

`Finding.MatchedTemplate` 字段:报告匹配路径不再消费,模型字段保留(不破坏 schema),后续单独的清理 PR 移除。

## 5. 匹配逻辑

### 5.1 重写 `GetFindingTemplateForFinding`

```go
// 输入:source_tool, source_rule_id, title
// 输出:*FindingTemplate | nil

func (q *Queries) GetFindingTemplateForFinding(
    sourceTool, sourceRuleID, title string,
) (*models.FindingTemplate, error) {
    tool := strings.TrimSpace(sourceTool)
    if tool == "" {
        return nil, nil
    }

    templates, err := q.listEnabledTemplatesByTool(tool)
    if err != nil {
        return nil, err
    }

    // Tier 1: 精确按 source_rule_id 在任一 match_keys 列表中查找
    if k := strings.TrimSpace(sourceRuleID); k != "" {
        for _, t := range templates {
            if containsExact(t.MatchKeys, k) {
                return t, nil
            }
        }
    }

    // Tier 2: 兜底按 title 在任一 match_keys 列表中查找
    if k := strings.TrimSpace(title); k != "" {
        for _, t := range templates {
            if containsExact(t.MatchKeys, k) {
                return t, nil
            }
        }
    }

    return nil, nil
}
```

**与现状的差异:**

- 函数签名从 4 参数 `(sourceTool, ruleID, matchedTemplate, title)` 改为 3 参数 `(sourceTool, sourceRuleID, title)`。删除对 `matchedTemplate` 参数的依赖,`Finding.MatchedTemplate` 字段不再被消费(模型字段保留不破坏 schema,见 4.4)。
- 调用方需要同步修改:
  - `internal/report/report.go::Aggregate` — 第 ~115 行调用点改为 3 参数
  - `internal/report/aggregate_run.go::AggregateByRunWithBatchEvidence` — 第 ~108 行调用点改为 3 参数
- 一次性把 `(source_tool, enabled=1)` 的全部词条拉进内存,O(n×m) 线性扫描。
- 同一 source_tool 下 70 条词条 × 每条 10 个 match_keys × 几百个 finding = 数万次字符串比较,完全在毫秒内完成。

### 5.2 单元测试覆盖点

- 单 key 命中(MS17-010 类)
- 多 key 命中任一(SQL 注入类)
- 同 match_key 不同 source_tool 不串
- source_rule_id 未命中时 title 兜底命中
- 禁用词条不参与
- 空匹配键、空 finding 字段不爆炸
- 唯一性应用层检查:创建时同 source_tool 下重复 key 报错

## 6. 报告渲染

### 6.1 `ReportData` 扩展

```go
// internal/report/report.go
type ReportData struct {
    Project      *models.Project
    Targets      []*models.Target
    ScopeRules   []*models.ScopeRule
    Assets       []*models.Asset
    WebEndpoints []*models.WebEndpoint
    Findings     []*ReportFinding              // 原始扁平列表保留
    Sections     []*ReportSection              // 新增:按 severity 倒序的渲染章节
    ToolVersions []*models.ToolInvocation
    GeneratedAt  time.Time
}

// 一个章节代表「报告漏洞详情」里的一节:
//   - 命中词条:Bucket 形,Template 非 nil,Findings 是同词条下的多个 finding
//   - 未命中:    Single 形,Template 为 nil,Findings 切片长度恰好为 1
type ReportSection struct {
    Template *models.FindingTemplate // nil = 未命中(同位混排原始块)
    Severity models.FindingSeverity  // 用于排序,命中时 = template.severity,未命中 = finding.severity
    Findings []*ReportFinding
}
```

### 6.2 分桶 + 排序算法

```
buckets := map[string][]*ReportFinding{}   // templateID → findings
unmatched := []*ReportFinding{}

for rf := range data.Findings {
    if rf.Template != nil {
        buckets[rf.Template.ID] = append(buckets[rf.Template.ID], rf)
    } else {
        unmatched = append(unmatched, rf)
    }
}

sections := []*ReportSection{}
for _, findings := range buckets {
    t := findings[0].Template
    sev := models.FindingSeverity(t.Severity)
    if sev == "" { sev = findings[0].Finding.Severity } // template severity 为空时 fallback
    sections = append(sections, &ReportSection{Template: t, Severity: sev, Findings: findings})
}
for _, rf := range unmatched {
    sections = append(sections, &ReportSection{Template: nil, Severity: rf.Finding.Severity, Findings: []*ReportFinding{rf}})
}

// 按 severity 倒序(critical → info),同级按 Template 优先于 Single 排
sort.SliceStable(sections, func(i, j int) bool {
    if severityRank[sections[i].Severity] != severityRank[sections[j].Severity] {
        return severityRank[sections[i].Severity] > severityRank[sections[j].Severity]
    }
    // 同级:命中(Template != nil)排在未命中前面
    return (sections[i].Template != nil) && (sections[j].Template == nil)
})

data.Sections = sections
```

### 6.3 Markdown 渲染

入口 `GenerateMarkdown` 中的「三、漏洞详情」段重写。整体结构:

```markdown
## 三、漏洞详情

### 1. 🔴 严重 · MS17-010 永恒之蓝远程代码执行漏洞

**漏洞描述**

<template.summary>

**受影响资产**

| 资产 | 端口 | 访问地址 | 工具规则 |
|---|---|---|---|
| 10.0.0.5 | 445 | — | ms17-010-smbtouch-poc |
| 10.0.0.7 | 445 | — | smb-eternalblue |

**修复建议**

<template.remediation>

> 套用词条 · 检测来源:nuclei · 命中 2 项

---

### 2. 🟠 高危 · SQL 注入漏洞

(同上结构,可能有 8 个资产行)

---

### 3. 🟡 中危 · ${finding.title}     ← 未命中,同位混排

**漏洞描述**

<finding.summary>     (工具原始输出,可能是 "Host: ...\nMatched: ...\nMatcher: ...")

**受影响资产**

| 资产 | 端口 | 访问地址 |
|---|---|---|
| 10.0.0.9 | 8080 | http://... |

**修复建议**

*暂无修复建议 — 该漏洞类型尚未在辞典中维护。可在「漏洞模板」页补充。*

> 未套词条 · 检测来源:${finding.source_tool} · 规则:${finding.source_rule_id}

---
```

**章节编号是全局连续的**(命中 + 未命中混排),按 severity 倒序排列。

**视觉差异只在底部脚注**:命中 = `套用词条`,未命中 = `未套词条`。客户拿到 md 文件复制粘贴时,主要正文不区分,只是元数据小字提示。

### 6.4 删除 HTML 与 JSON 报告导出

- 删除文件:`internal/report/html.go`、`internal/report/json.go`
- 删除路由:`/projects/{id}/reports/export.html`(如有)、`/projects/{id}/reports/export.json`
- 删除前端调用:`api.exportReportJSON`、ReportsPage.tsx 中 JSON 导出按钮、`handleExport("json")` 分支、HTML 相关代码(如有)
- 报告交付物自此只有 markdown 一种

### 6.5 单元测试覆盖点

- 全命中场景:不同 severity 的多个词条桶,按 severity 倒序、章节编号连续
- 全未命中场景:每条 finding 单独一节,章节编号连续
- 混合场景:命中 + 未命中按 severity 穿插,同级时命中排在未命中前
- 同一词条命中 N 个 finding:受影响资产表格 N 行
- Template severity 为空字符串时,fallback 到 finding.severity 用于排序

## 7. API 层变更

### 7.1 `finding_template_handlers.go`

- `handleListFindingTemplates` / `handleGetFindingTemplate`:无逻辑变更,Response payload 中 `match_key` 字段去除,新增 `match_keys` 数组。
- `handleCreateFindingTemplate`:
  - Request payload `match_key *string` 替换为 `match_keys *[]string`。
  - 校验:`match_keys` 非空数组,每个 key trim 后非空,数组内去重。
  - 校验:同 `source_tool` 下,新提交的任一 key 不能已经被其他词条占用 → 409 Conflict。
- `handlePatchFindingTemplate`:
  - 同上,允许部分更新 `match_keys`。
  - 校验:与其他词条无 key 重叠。
- `handleExportFindingTemplates`:输出形态调整,`match_key string` 改为 `match_keys []string`。
- `handleAcceptFindingTemplateUpstream`:消费 `BuiltinPayload`(新 schema),应用 upstream 的 `match_keys` 数组。
- `handleDeleteFindingTemplate`:无变更。

### 7.2 `report_handlers.go`

**保留:** `handleExportReportMD`(ReportsPage 同步 markdown 下载)。

**删除以下 handler 与 goroutine:**

| 函数 | 现状路由 / 用途 |
|---|---|
| `handleExportReportJSON` | `GET /projects/:id/reports/export.json` — 报告 JSON 导出 |
| `handleCreateReport` | `POST /runs/:runId/reports` — 触发异步报告生成 |
| `generateReport` | goroutine,异步生成持久化 HTML 文件 |
| `broadcastReportProgress` | 通过 SSE 推 `report_progress` 事件 |
| `handleGetReport` | `GET /reports/:reportId` |
| `handleGetReportByRun` | `GET /runs/:runId/reports` |
| `handleDownloadReport` | `GET /reports/:reportId/download` |
| `handleDeleteReport` | `DELETE /reports/:reportId` |
| `handleListReports` | `GET /reports` |
| `checkDiskSpace` | 仅被 `handleCreateReport` 调用,一并删 |

**对应路由全部从 `Register` / 路由表中移除。**

`SSE` 事件 `report_progress` 不再发出,但 SSE 框架本身保留(其他事件如 `task_progress` 仍用)。

### 7.3 删除其他 Report 相关代码

| 文件 | 操作 |
|---|---|
| `internal/db/queries_report.go` | 整文件删除 |
| `internal/models/scan.go` | 删除 `Report` 结构体、`ReportStatus` 类型与 `ReportGenerating`/`ReportComplete`/`ReportFailed`/`ReportPartial` 常量(精确改动,不删整文件) |
| `internal/report/html.go` | 整文件删除(无人再调用 `GenerateHTML`) |
| `internal/report/json.go` | 整文件删除(无人再调用 `GenerateJSON`) |

**`reports` 表 schema 不动:**`internal/db/v14.go` 中的 `CREATE TABLE reports` 保留,本次不写新 migration 删表。让数据库里的孤儿表存在,避免回滚困难。后续单独清理 PR。

### 7.4 OpenAPI / 文档

`internal/api/README.md` 的「Handler 文件总览」和「字段反向索引」按本次改动同步:
- finding_template handler 行的 payload schema 描述更新
- report handler 行去掉所有异步报告 endpoint,只保留 `GET /projects/:id/reports/export.md`

## 8. 前端工作流

### 8.1 `VulnTemplatesPage`(漏洞模板页)

**编辑器变更:**
- `match_key` 单输入框 → `match_keys` chip 输入(回车追加,chip 单独可删除)
- 校验:不能为空数组;每个 key 去前后空白后非空;数组内自动去重
- 错误显示:后端 409 Conflict 时,toast 提示「与其他词条匹配键重叠:<冲突 key>」

**列表表格变更:**
- 「匹配键」列改成 chip 形式,显示前 3 个 + 折叠「+N」
- 鼠标悬停在「+N」上时 popover 列出全部 key

**导出按钮文案:**
- 「导出 JSON」改为「**导出 JSON · 贴回仓库**」
- tooltip:「下载当前所有词条(含本地新增)为 vuln-templates.json 格式,可粘贴到仓库 `docs/templates/vuln-templates.json` 共享给团队」

**保留:** 双轨徽章(内置/自定义/本地已修改)、「应用上游」、启用/禁用、删除(仅对自定义可用)。

### 8.2 `ReportsPage`(报告页)

**finding 列表条目新增「词条覆盖状态」徽章:**
- 已命中:绿色 `已套词条:<template.title>` — 点击跳转到 VulnTemplatesPage 并打开该词条编辑
- 未命中:橙色 `未套词条 · + 加入辞典` — 点击弹出新词条编辑器

**「+ 加入辞典」弹窗:**
- 复用 VulnTemplatesPage 的编辑器组件(抽成共享组件 `<TemplateEditor>`)
- 预填值:
  - `source_tool` = finding.source_tool
  - `match_keys` = `[finding.source_rule_id]`(已含一项,用户可继续加)
  - `title` = `""`(留空,提示用户填写中文漏洞名)
  - `severity` = finding.severity(作为默认严重等级建议)
  - `summary` / `remediation` = `""`
- 保存成功后:toast「词条已创建,刷新报告查看效果」;**用户重新预览或下载报告时,这条 finding 自动套用**

**导出区:**
- 只保留 Markdown 导出按钮 + 预览。删除 JSON 导出按钮、删除「JSON 适用于自动化平台集成」类文案。

**覆盖率小统计:**
- 在「严重等级分布」上方加一行:「已套词条 X 项 · 未套词条 Y 项」
- 不点击、纯展示,辅助用户判断辞典覆盖度

### 8.3 共享组件 `<TemplateEditor>`

抽出独立组件,被 VulnTemplatesPage 和 ReportsPage 复用:
- props:`initialValue: Partial<TemplateForm>`、`mode: "create" | "edit"`、`onSaved: (t: FindingTemplate) => void`
- 内部包含所有字段(source_tool, match_keys chip 输入, title, severity, summary, remediation, enabled),保存按钮、校验逻辑

### 8.4 `RunsPage`(扫描运行页)清理 — 删除异步报告流程

**前端 lib/api.ts 删除以下方法与类型:**

| 删除项 | 备注 |
|---|---|
| `Report` 类型 | 异步报告的元数据类型 |
| `api.createReport` | POST /runs/:runId/reports |
| `api.getReport` | GET /reports/:reportId |
| `api.getReportByRun` | GET /runs/:runId/reports |
| `api.deleteReport` | DELETE /reports/:reportId |
| `api.listReports` | GET /reports |
| `api.downloadReport` | GET /reports/:reportId/download |
| `api.exportReportJSON` | GET /projects/:id/reports/export.json |

**`RunsPage.tsx` 删除以下功能:**
- `useState`:`reports` Map、`generatingReports` Set、`checkedReportRuns` Set
- `useEffect`:遍历 runs 调用 `getReportByRun` 拉取报告状态的逻辑
- SSE 监听里 `report_progress` 事件处理分支(改为忽略)
- 触发异步报告生成的「生成报告」按钮 / `handleGenerateReport` 函数
- 每个 run 行右侧的「报告」状态气泡 / 下载按钮 / 进度图标
- 删 `api.getReportByRun` / `api.createReport` / `api.getReport` / `api.downloadReport` 的所有调用点

**替代体验:** Run 行的「报告」按钮位置改为「**查看报告**」单按钮,点击导航到 `/projects/:projectId/reports`(用户在 ReportsPage 同步预览或下载 markdown)。

## 9. MVP 范围外(不做)

- **手册 markdown 文件 → vuln-templates.json 的自动转换工具** — 用户的 70+ 手册首批可以人工录入或交给 LLM 一次性转换,转换工具本身不是 MVP 一部分
- **finding 级 override**(强制套某词条 / 强制不套) — 当前的「改 status 为 false_positive 让它不进报告」已够用
- **词条管理页的搜索 / 标签筛选 / 分页** — 70 条规模不刚需,可以后续按需求加
- **词条字段扩展**(危害 / PoC / 参考链接 / 分类 tag) — 维持现有 4 件套,不引入新字段维护负担

## 10. 受影响文件清单

**数据模型与持久化:**

- `internal/models/finding_template.go` — 字段从 `MatchKey string` 改为 `MatchKeys []string` + `MatchKeysJSON string`
- `internal/models/scan.go` — **删除** `Report` 结构体、`ReportStatus` 类型与 `ReportGenerating`/`ReportComplete`/`ReportFailed`/`ReportPartial` 常量(精确改动)
- `internal/db/queries_finding_template.go` — scan / create / update 调整;`GetFindingTemplateForFinding` 重写
- `internal/db/queries_finding_template_test.go` — 测试覆盖新匹配语义
- `internal/db/seed_finding_templates.go` — `SeedFindingTemplate.MatchKey` → `MatchKeys`;同步逻辑跟着改
- `internal/db/seed_finding_templates_test.go` — 同步测试随之改
- `internal/db/v24.go` — **新建 migration**(添加 `match_keys TEXT` 列、数据迁移)
- `internal/db/queries_report.go` — **整文件删除**

**报告生成与导出:**

- `internal/report/report.go` — `ReportData` 加 `Sections`,Aggregate 阶段分桶
- `internal/report/aggregate_run.go` — 同上
- `internal/report/markdown.go` — `GenerateMarkdown` 渲染节奏重写
- `internal/report/report_test.go` — 测试覆盖分桶 + 渲染场景
- `internal/report/html.go` — **整文件删除**
- `internal/report/json.go` — **整文件删除**

**API 层:**

- `internal/api/finding_template_handlers.go` — payload schema 更新、应用层唯一性检查
- `internal/api/report_handlers.go` — 删除 `handleExportReportJSON` / `handleCreateReport` / `generateReport` / `broadcastReportProgress` / `handleGetReport` / `handleGetReportByRun` / `handleDownloadReport` / `handleDeleteReport` / `handleListReports` / `checkDiskSpace`;保留 `handleExportReportMD`
- `internal/api/handlers.go`(或 `server.go` 中 `Register`) — 删除所有报告异步 endpoint 的路由注册
- `internal/api/README.md` — 字段反向索引和 handler 总览同步

**仓库辞典:**

- `docs/templates/vuln-templates.json` — 当前是 `[]`,本次改动不强求填内容,但 schema 文档(若存在)需同步

**前端:**

- `frontend/src/lib/api.ts` — `FindingTemplate` 类型 `match_key: string` 改为 `match_keys: string[]`;删除 `Report` 类型;删除 `createReport` / `getReport` / `getReportByRun` / `deleteReport` / `listReports` / `downloadReport` / `exportReportJSON`
- `frontend/src/pages/VulnTemplatesPage.tsx` — chip 输入、新按钮文案
- `frontend/src/pages/ReportsPage.tsx` — 状态徽章、「加入辞典」弹窗、删 JSON 导出、覆盖率统计
- `frontend/src/pages/RunsPage.tsx` — 删除报告 useState/useEffect/SSE 监听/「生成报告」按钮/进度气泡,替换为「查看报告」单按钮(导航到 ReportsPage)
- `frontend/src/components/TemplateEditor.tsx` — **新建**,共享编辑器组件

**保留不动(后续清理):**

- `internal/db/v14.go` 中的 `CREATE TABLE reports` — 表 schema 保留,避免回滚困难,作为孤儿表,后续单独 PR 写新 migration 删表

## 11. 验收标准

- **单元测试**:`go test ./internal/db/...` 和 `go test ./internal/report/...` 全绿,新增的匹配语义和分桶/渲染场景有覆盖
- **E2E**:
  1. 起 server + worker,跑一次 nuclei 扫描(任意目标),生成几个 finding
  2. 在「漏洞模板」页新建一个词条,挂上其中一个 finding 的 `source_rule_id`
  3. 在「报告」页刷新,该 finding 显示绿色「已套词条:<title>」徽章
  4. 同 severity 的另一个未命中 finding 显示橙色「未套词条 · + 加入辞典」徽章
  5. 点击「+ 加入辞典」弹出预填弹窗,填中文 title/summary/remediation,保存
  6. 刷新报告,该 finding 也显示已套词条
  7. 下载 markdown 报告,验证章节按 severity 倒序、命中桶聚合渲染、未命中条目同位混排、章节编号连续
  8. 「漏洞模板」页点击「导出 JSON · 贴回仓库」,下载的 JSON 数组每条都有 `match_keys` 数组
- **回归**:
  - 内置-上游同步(`SyncFindingTemplatesFromFile`)仍能正确处理 insert/update/preserve/delete/skip
  - 老 `match_key` 列(保留作为兼容)读出来不报错
  - 删除 HTML / JSON 报告导出与异步报告流程后,前端没有死链接、Markdown 报告导出/预览正常
  - RunsPage 删除后,run 行的「查看报告」按钮正确导航到 `/projects/:projectId/reports`
  - 不再调用 `getReportByRun` / `createReport` 等被删除的 API(浏览器 Network 面板验证)
  - `reports` 孤儿表存在但无新写入(SQLite 上验证)
  - SSE 不再发送 `report_progress` 事件;其他 SSE 事件(如 `task_progress`)正常

## 12. 文档同步约束

按 `CLAUDE.md` 中的「文档同步约束」条款,本次改动需同步:

- `internal/api/README.md`:finding_template handler 行 payload schema 更新;report handler 行去掉 JSON/HTML 导出
- `docs/current/architecture.md`:若描述了报告导出格式或漏洞模板架构,需同步重写
- `docs/current/plan.md`:作为一次实现条目纳入,完成后勾掉
- 本提案文件实施完成后,迁移到 `docs/current/` 并把 `status` 改为 `active`

## 13. 验证检查清单(实施完成时)

- [ ] `go vet ./...` 通过
- [ ] `go test ./internal/db/... ./internal/report/... ./internal/api/...` 通过
- [ ] 前端 `npm run typecheck` 通过
- [ ] E2E 8 步全过(见第 11 节)
- [ ] `internal/api/README.md` 字段反向索引、handler 总览同步完成
- [ ] `docs/current/architecture.md` 已对应更新
- [ ] PR 描述显式列出本次修改的文档清单
