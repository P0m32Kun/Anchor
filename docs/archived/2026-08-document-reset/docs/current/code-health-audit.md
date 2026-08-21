---
status: active
source_of_truth: true
owner: kun
last_updated: 2026-05-26
scope: engineering-health-audit
audit_baseline_date: 2026-05-13
---

# 代码健康审计计划（多模型可执行）

> **文档角色**：这是分阶段、可分配给不同模型独立执行的任务推进文件。每阶段自包含，有明确的输入、输出、验证方法。

## 审计基线（2026-05-13 快照）

| 指标 | 数值 (2026-05-13) | 数值 (2026-05-26) | 健康阈值 |
|------|-------------------|-------------------|----------|
| Go 源文件 | 172 | 175 | — |
| TSX/TS 源文件 | 52 | 56 | — |
| Go 测试文件 | 28 (16%) | **64 (37%)** | >40% |
| 前端测试文件 | 4 (8%) | 4 (7%) | >30% |
| Go 超 400 行文件 | 13 | 14 | <5 |
| Go 超 800 行文件 | 1 (worker/server.go 730L) | — | 0 |
| DB 迁移版本 | 20 | 20 | v0.x 阶段偏高 |
| api 直接 import db | 4 | 3 | 0 |
| api 直接 import models | 17 | 15 | 逼近 0 |

> 注：本表基于 2026-05-26 Slimdown 审计更新。更多变化详见
> [`docs/active/review/slimdown-2026-summary.md`](../active/review/slimdown-2026-summary.md) 的§6。

---

## 执行协议（给每个执行模型的指令）

### 必须遵守

1. **只审计，不改代码** — 本计划产出报告，不产出 patch
2. **每个发现必须包含**：文件路径、行号、问题描述、风险等级、修复建议
3. **每完成一个检查项**：在对应 checkbox 标记 `[x]`，附简短发现概要
4. **遇到阻塞**：记录阻塞原因，跳过该项继续执行
5. **工具链就绪检查**：执行前先跑 `go vet ./internal/...` 和 `cd frontend && npx tsc --noEmit` 确认环境正常

### 风险分级标准

| 等级 | 定义 | 示例 |
|------|------|------|
| **Critical** | 生产数据丢失/越权/安全绕过 | scope check 绕过、null scan 崩溃 |
| **High** | 功能完全不可用/数据损坏 | 契约不匹配导致功能失败 |
| **Medium** | 代码腐化/可维护性问题 | 上帝文件、分层违规 |
| **Low** | 风格/命名/轻微重复 | 重复代码 < 20 行 |

---

## 阶段一：安全性 & 契约一致性审计（预计 2-3 工日）

> 优先级 **P0** — 7 条 pitfall 中 5 条属于此阶段范围，生产风险最高。

### 1.1 前后端契约对比

**目标**：找出所有前后端字段不一致。

**输入文件**：
- Go 模型：`internal/models/*.go`（所有含 `json:"..."` tag 的结构体）
- 前端类型：`frontend/src/lib/api.ts`（所有 interface/type 定义）

**执行步骤**：

- [ ] 1.1.1 提取 Go 模型的所有 JSON 字段
  - 执行：`grep -rn 'json:"' internal/models/*.go | grep -v _test.go`
  - 输出格式：`文件:行号 | 结构体名 | 字段名 | json tag | 类型 | omitempty`
  
- [ ] 1.1.2 提取前端 API 类型的所有字段
  - 读取 `frontend/src/lib/api.ts`，列出所有 interface/type 的字段
  - 输出格式：`类型名 | 字段名 | TypeScript 类型 | 可选?`
  
- [ ] 1.1.3 逐结构体对比
  - 对每个后端模型，找到对应的前端类型
  - 检查：字段名是否一致、类型是否兼容、nullable 是否对齐、enum 值是否一致
  - 重点结构体（按 pitfall 频率）：`Target`、`Finding`、`Evidence`、`Asset`、`ScanConfig`、`Report`
  - 输出：差异清单，每条注明风险等级

- [ ] 1.1.4 列出前后端未对齐的风险字段
  - 特别关注：`*string` vs `string`、`*int` vs `number | null`、enum string vs union type

**产出物**：`## 契约差异报告` — Markdown 表格格式，列：后端字段 | 前端字段 | 差异 | 风险

---

### 1.2 Null-Safety 扫描

**目标**：找出所有可能导致 nil panic 的反引用点。

**输入**：`internal/` 下所有 Go 文件。

**执行步骤**：

- [ ] 1.2.1 定位所有可能为 nil 的数据库字段
  - 搜索 `sql.NullString`、`sql.NullInt64`、`sql.NullBool`、`sql.NullTime` 的使用
  - 搜索 models 中 `*string`、`*int`、`*bool`、`*time.Time` 字段
  - 列出所有在 queries 中 scan 到这些字段但未检查 `.Valid` 的地方
  - 命令辅助：`grep -rn 'NullString\|NullInt64\|NullBool\|NullTime' internal/db/`

- [ ] 1.2.2 审计所有 `*string` / `*int` 反引用点
  - 搜索所有 `*variable` 形式的解引用（在赋值/比较/传参中）
  - 判断该变量是否能保证非 nil（刚赋值？函数返回？）
  - 标记高风险点（从外部输入、DB 查询、JSON 解析得到的指针）
  - 命令辅助：需要逐文件人工审查，可先搜索 `\.Valid` 缺失模式

- [ ] 1.2.3 验证已修复的 pitfall 是否真正修复
  - `docs/pitfalls/20260426-artifact-type-mismatch.md` — 检查 `internal/worker/` 中的 Artifact 类型赋值
  - `docs/pitfalls/20260426-raw-artifact-redaction-loss.md` — 检查 sanitize 逻辑是否保留原始数据
  - `docs/pitfalls/20260427-null-scan-crash.md` — 检查 `created_by` 列是否有 nil guard
  - `docs/pitfalls/20260427-markdown-pipe-corruption.md` — 检查 `internal/report/markdown.go` 是否转义管道符

**产出物**：`## Nil-Safety 风险清单` — 表格：文件:行号 | 变量 | nil 可能性 | 风险 | 修复建议

---

### 1.3 Scope Check 完整性审计

**目标**：确认 scope gate 不能被绕过。

**输入**：`internal/scope/`、`internal/workflow/` 中调用 scope check 的代码。

**执行步骤**：

- [ ] 1.3.1 理解 scope check 数据流
  - 读取 `internal/scope/scope.go` — Scope 检查入口和核心逻辑
  - 读取 `internal/scope/import.go` — 批量导入时的 scope 检查
  - 输出：scope check 的决策树（什么条件 allow、什么条件 deny）

- [ ] 1.3.2 追踪 scope check 的所有调用点
  - 搜索所有调用 scope 相关函数的代码
  - 命令：`grep -rn 'scope\.' internal/ --include='*.go' | grep -v _test.go`
  - 对每个调用点，确认 deny 后是否真的阻止了后续流程

- [ ] 1.3.3 检查所有资产入库路径是否都经过 scope
  - 资产入库路径：FOFA 结果、Subfinder 结果、httpx 结果、Naabu 结果
  - 读取 `internal/workflow/discovery.go` 中各资产发现步骤
  - 读取 `internal/asset/merger.go` — 资产合并时是否重新验证 scope
  - 验证每个入库点都有 scope gate

- [ ] 1.3.4 边界条件测试（不写代码，列出应测场景）
  - CDN IP 是否被正确处理（CDN IP 可能不属于目标组织）
  - CIDR 展开后的每个 IP 是否单独过 scope
  - URL 重定向后的域名是否重新过 scope
  - Company 展开的资产是否过 scope

**产出物**：`## Scope Check 调用图` + `## 绕过风险清单`

---

### 1.4 Ghost Worker 清理逻辑审查

**目标**：确认 worker 生命周期管理无资源泄漏。

**输入**：`internal/worker/server.go` (730L)、`internal/api/worker_handlers.go` (294L)。

**执行步骤**：

- [ ] 1.4.1 Worker 注册逻辑审查
  - Worker 注册时是否检查了重复（同名 worker 重复注册）
  - 注册时的心跳时间戳是否正确初始化

- [ ] 1.4.2 心跳超时处理审查
  - 搜索所有心跳检查/超时标记逻辑
  - 命令：`grep -rn 'heartbeat\|timeout\|offline\|dead' internal/worker/ internal/api/`
  - 确认超时 worker 是否被正确标记为 offline
  - 确认超时 worker 的任务是否被重新分配

- [ ] 1.4.3 Worker 注销与清理
  - Worker 注销时：数据库记录状态？未完成任务？心跳记录？
  - Server 重启时：所有 worker 是否被清理？
  - Docker 容器重启场景（最常出 ghost 的场景）

- [ ] 1.4.4 验证 pitfall 修复
  - `docs/pitfalls/20260428-ghost-worker-cleanup.md` — 确认清理逻辑已实现

**产出物**：`## Ghost Worker 风险清单`

---

### 1.5 契约回归测试骨架

**目标**：为最严重的 3 个 pitfall 建立回归测试。

**执行步骤**：

- [ ] 1.5.1 为 null-scan-crash 写 Go 单元测试
  - 创建 `internal/db/queries_finding_null_test.go` 或补充现有测试
  - 测试：查询包含 NULL `created_by` 的 finding 时不崩溃
  - 测试：查询包含 NULL `created_by` 的 evidence 时不崩溃

- [ ] 1.5.2 为 markdown-pipe-corruption 写 Go 单元测试
  - 在 `internal/report/report_test.go` 中补充测试
  - 测试：Finding title 含 `|` 时表格不破坏
  - 测试：Asset value 含 `|` 时表格不破坏

- [ ] 1.5.3 为 artifact-type-mismatch 写集成测试
  - 在 `internal/worker/` 中补充测试
  - 测试：worker 保存的 artifact 类型与 workflow 期望的类型一致

**产出物**：3 个测试文件/测试用例的补充

---

## 阶段二：架构分层 & 耦合度审计（预计 2-3 工日）

> 优先级 **P1** — 量化重构范围，可以和阶段一并行。

### 2.1 分层违规扫描

**目标**：找出所有跨层直接依赖。

**已知基线（已发现）**：

| 违规类型 | 文件 | 数量 |
|----------|------|------|
| api -> db 直接 import | server.go, finding_template_handlers.go, retest_handlers.go, handlers_test.go | 4 处 |
| api -> models 直接 import | 17 个文件 | 17 处 |

**执行步骤**：

- [ ] 2.1.1 审计 api -> db 的 4 处直接依赖
  - `internal/api/server.go:14` — 为什么 Server 需要直接 import db？
  - `internal/api/finding_template_handlers.go:9` — 是否可走 service？
  - `internal/api/retest_handlers.go:8` — 是否可走 service？
  - `internal/api/handlers_test.go:15` — 测试中的直接依赖是否合理？
  - 输出：每处的修复建议（走 service / 保留）

- [ ] 2.1.2 审计 api -> models 的 17 处直接依赖
  - 分类：哪些是返回 JSON 时用 models 做序列化（可接受）
  - 哪些是用 models 做业务判断（应下沉到 service）
  - 输出：分类清单

- [ ] 2.1.3 检查其他分层违规
  - `internal/db/` 是否 import 了 `internal/api/`？（逆依赖）
  - `internal/models/` 是否 import 了 `internal/db/`？（循环依赖风险）
  - `internal/workflow/` 是否 import 了 `internal/api/`？
  - 命令：每对包互相 grep import 语句检查

**产出物**：`## 分层违规清单` — 表格：文件 | 违规类型 | 严重性 | 修复方向

---

### 2.2 Service 层覆盖率评估

**目标**：判断哪些业务逻辑还在 handler 里，评估迁移价值。

**输入**：`internal/service/` 和 `internal/api/`。

**执行步骤**：

- [ ] 2.2.1 列出当前 Service 层的覆盖范围
  - `internal/service/service.go` — 现有 Service 接口定义
  - `internal/service/project.go` — ProjectService 实现
  - `internal/service/target.go` — TargetService 实现
  - `internal/service/finding.go` — FindingService 实现
  - 输出：已覆盖的业务域列表

- [ ] 2.2.2 扫描 handler 中超过 30 行的函数体
  - 对每个 handler 文件，找出函数体 > 30 行的函数
  - 判断其中是否包含业务逻辑（vs 纯 HTTP 处理）
  - 重点关注：
    - `asset_handlers.go` (343L)
    - `pipeline_handlers.go` (404L)
    - `nuclei_custom_handlers.go` (351L)
    - `report_handlers.go` (339L)
  - 输出：应迁移到 service 的函数清单

- [ ] 2.2.3 评估 Service 层扩展工作量
  - 当前 4 个 Service，预估还需要几个
  - 每个新 Service 的工作量（参考已有实现的代码量）

**产出物**：`## Service 层缺口清单`

---

### 2.3 上帝文件优先级排序

**目标**：对 13 个超 400 行的 Go 文件做职责分析。

**输入文件清单**（按行数降序）：

| 文件 | 行数 | 核心职责 |
|------|------|----------|
| internal/worker/server.go | 730 | Worker HTTP 服务 + 任务状态管理 |
| internal/db/queries_scan.go | 721 | 扫描相关的全部 SQL 查询 |
| internal/workflow/discovery.go | 646 | 资产发现流程编排 |
| internal/nuclei/custom/manager.go | 628 | Nuclei 自定义模板管理 |
| internal/scope/import.go | 536 (含 test) | Scope 批量导入与检查 |
| internal/scope/scope.go | 415 | Scope 规则匹配引擎 |
| internal/worker/worker.go | 405 | Worker 任务执行 |
| internal/api/pipeline_handlers.go | 404 | Pipeline API handler |
| internal/models/scan.go | 388 | Scan 相关全部模型 |

**执行步骤**：

- [ ] 2.3.1 逐个分析上帝文件的职责内聚性
  - 读文件，列出其中所有公开函数/方法
  - 按职责分组（如 worker/server.go：HTTP handler / 任务队列 / 心跳管理 / 注册管理）
  - 判断每组职责是否可以独立为一个文件

- [ ] 2.3.2 排序拆分优先级
  - 优先级 = 风险 × 改动频率 × 可拆分性
  - worker/server.go：730L，职责混杂 — P0 拆分
  - db/queries_scan.go：721L，单职责但过大 — P1 拆分
  - workflow/discovery.go：646L，可拆分 — P1
  - nuclei/custom/manager.go：628L — P2
  - 其余 — P3

- [ ] 2.3.3 每个上帝文件输出拆分方案
  - 建议新文件列表
  - 每文件预估行数
  - 拆分风险点

**产出物**：`## 上帝文件拆分方案`

---

### 2.4 DB 迁移债务评估

**目标**：评估 20 个迁移文件的健康状况。

**输入**：`internal/db/v1.go` ~ `v20.go`。

**执行步骤**：

- [ ] 2.4.1 迁移文件概览
  - v1.go (352L) — 初始 schema，巨量
  - v2-v7 — 早期迭代
  - v8-v14 — 中期扩展
  - v15-v20 — 近期增量（全部 < 60 行）
  - 输出：按时间线标注每个迁移的变更内容

- [ ] 2.4.2 检测 schema drift
  - 对比 v1.go 中的初始 schema 与当前 models/*.go 中的结构体定义
  - 搜索所有 `CREATE TABLE` 语句，确认与 models 定义一致
  - 标记不一致的地方（字段缺失、类型不同、约束差异）

- [ ] 2.4.3 评估迁移合并机会
  - v13 (20L)、v14 (39L)、v18 (29L)、v19 (27L)、v20 (19L) — 都是微型 ALTER TABLE
  - 建议合并为一个"v0.4 合并迁移"
  - 评估合并风险（是否有不可逆 DDL）

- [ ] 2.4.4 检查回滚安全性
  - 哪些迁移包含 DROP TABLE / DROP COLUMN（不可逆）
  - 哪些迁移可以在开发环境安全回滚

**产出物**：`## DB 迁移健康报告`

---

### 2.5 裸 SQL 治理方案

**目标**：评估 SQL 治理的最佳路径。

**现状**：182 条裸 SQL 分布在 10+ 个 `queries_*.go` 文件中。

**执行步骤**：

- [ ] 2.5.1 SQL 分布统计
  - 统计每个 queries_*.go 文件中的 SQL 语句数量
  - 分类：简单 CRUD / 复杂 JOIN / 聚合查询 / 动态 SQL

- [ ] 2.5.2 SQL 注入风险评估
  - 搜索所有使用 `fmt.Sprintf` 拼接 SQL 的地方
  - 搜索所有使用字符串拼接构建 WHERE 子句的地方
  - 命令：`grep -rn 'fmt.Sprintf.*SELECT\|fmt.Sprintf.*INSERT\|fmt.Sprintf.*UPDATE\|fmt.Sprintf.*DELETE' internal/db/`
  - 已有参数化查询的占比

- [ ] 2.5.3 治理方案对比
  - 方案 A：引入 sqlc（从 SQL 生成 Go 代码，零运行时开销）
  - 方案 B：引入 ent（Go ORM，schema-first）
  - 方案 C：手工 Repository 模式（渐进式，不引入新工具）
  - 对每个方案评估：学习成本、迁移工作量、性能影响、AI 友好度

**产出物**：`## SQL 治理选型建议`

---

## 阶段三：测试策略 & 覆盖率规划（预计 1-2 工日）

> 优先级 **P2** — 依赖阶段一的契约发现来定优先级。

### 3.1 当前测试质量审计

**执行步骤**：

- [ ] 3.1.1 生成覆盖率报告
  - 执行：`go test -coverprofile=coverage.out ./internal/...`
  - 执行：`go tool cover -func=coverage.out | sort -t: -k2 -rn | head -40`
  - 输出：覆盖率最低的 10 个包

- [ ] 3.1.2 已有测试的深度审计
  - 抽查 5 个测试文件，判断它们测的是：
    - 纯工具函数（低价值）
    - 业务逻辑（中价值）
    - 集成场景（高价值）
  - 输出：测试价值评估

- [ ] 3.1.3 包粒度覆盖率矩阵
  - 每个 internal 子包的覆盖率百分比
  - 标注哪些包是安全关键（scope、workflow、worker）

**产出物**：`## 包级覆盖率矩阵`

---

### 3.2 关键路径测试缺口

**目标**：识别核心业务流程中没有测试覆盖的步骤。

**完整扫描流程**：
```
Target 导入 → Scope Check → 资产发现(FOFA/Subfinder) → DNSx → CDN → Naabu → nmap → httpx → Nuclei → Finding → 人工验证 → Report
```

**执行步骤**：

- [ ] 3.2.1 对扫描流程的每个步骤，检查是否有测试覆盖
  - 搜索每个步骤对应的测试文件
  - 命令：`grep -rn 'func Test' internal/ | grep -i 'scope\|discovery\|nuclei\|report\|finding'`

- [ ] 3.2.2 列出完全没有测试的关键函数
  - 关注 `internal/workflow/` 下的所有公开函数
  - 关注 `internal/scope/scope.go` 的核心匹配函数

- [ ] 3.2.3 按业务影响排序测试缺口

**产出物**：`## 关键路径测试缺口清单`

---

### 3.3 测试金字塔分析

**目标**：调整测试层级分布。

**当前分布**：
- 单元测试：28 个 Go test 文件
- 集成测试：少量（需评估）
- E2E：22 个 Playwright spec（大部分 mock）

**执行步骤**：

- [ ] 3.3.1 分类每个 E2E 测试
  - 真正的 E2E（需要全栈运行）→ 保留在 E2E 层
  - 可以下沉为集成测试的 → 建议迁移
  - 纯前端逻辑测试 → 建议迁移为 Vitest

- [ ] 3.3.2 评估 E2E mock 保真度
  - `frontend/e2e/fixtures/api-helpers.ts` — 查看 mock 实现
  - `frontend/e2e/fixtures/test-data.ts` — 查看 mock 数据
  - 判断 mock 是否和生产 API 行为一致

**产出物**：`## 测试层级调整建议`

---

### 3.4 测试策略改进路线图

**目标**：制定 3 个里程碑的测试提升计划。

- [ ] 3.4.1 Milestone 1（立即）：补 5 个最高风险的测试
- [ ] 3.4.2 Milestone 2（2 周内）：覆盖率到 30%
- [ ] 3.4.3 Milestone 3（1 月内）：覆盖率到 50%，关键路径全覆盖

**产出物**：`## 测试提升路线图`

---

## 阶段四：前端代码健康审计（预计 1-2 工日）

> 优先级 **P3** — 后端稳定后再做，减少返工。

### 4.1 api.ts 拆分方案

**目标**：将 976 行的 API 客户端按 domain 拆分。

**输入**：`frontend/src/lib/api.ts`。

**执行步骤**：

- [ ] 4.1.1 分析 api.ts 的职责分组
  - 列出所有导出的函数和类型
  - 按 domain 分组：project、target、finding、report、worker、scan、engine、dictionary
  - 识别公共逻辑：请求封装、错误处理、类型定义

- [ ] 4.1.2 设计拆分后的文件结构
  - 建议目录：`frontend/src/lib/api/`
  - 建议文件：`client.ts`（基础请求）、`projects.ts`、`targets.ts`、`findings.ts` 等
  - 每个文件预估行数

- [ ] 4.1.3 评估拆分风险
  - 哪些类型是跨 domain 共享的（如 `PaginatedResponse`）
  - 拆分后哪些 import 需要更新

**产出物**：`## api.ts 拆分方案`

---

### 4.2 大页面组件审计

**目标**：对 4 个大页面提出拆分建议。

**输入**：
- `frontend/src/pages/RunsPage.tsx` (833L)
- `frontend/src/pages/TemplatesPage.tsx` (803L)
- `frontend/src/pages/TargetPage.tsx` (715L)
- `frontend/src/components/ScanModal.tsx` (634L)

**执行步骤**：

- [ ] 4.2.1 逐个分析大页面的职责分组
  - 每个页面拆分为：UI 组件 + 业务 hook + 数据加载
  - 标记可提取的子组件（表格、表单、筛选器、详情面板）

- [ ] 4.2.2 识别可复用的模式
  - 哪些表格逻辑在多个页面重复
  - 哪些筛选/分页逻辑可以抽取

- [ ] 4.2.3 输出每个页面的拆分方案

**产出物**：`## 大页面拆分方案`

---

### 4.3 状态管理审查

**目标**：评估 Zustand store 和本地 state 的使用边界。

**输入**：`frontend/src/lib/store.ts`、各页面的 `useState` 调用。

**执行步骤**：

- [ ] 4.3.1 审查 store.ts 中的全局状态
  - 哪些状态确实需要全局共享（用户、项目、token）
  - 哪些状态应该局部化（页面级 UI 状态）

- [ ] 4.3.2 检查是否缺少应有的全局状态
  - Worker 在线状态是否应该在 store 中
  - 当前项目 ID 是否应该在 store 中
  - SSE 连接状态是否应该统一管理

- [ ] 4.3.3 输出状态管理规范建议

**产出物**：`## 状态管理建议`

---

### 4.4 前端测试路线图

**目标**：制定前端测试的渐进提升计划。

**执行步骤**：

- [ ] 4.4.1 分析现有 4 个测试文件的覆盖范围
  - `api.test.ts`、`Button.test.tsx`、`ScanModal.test.tsx`、`RunsPage.test.tsx`
  - 判断是否测试了关键逻辑

- [ ] 4.4.2 制定前端测试优先级
  - 优先级 1：api.ts 的单元测试（数据转换、错误处理）
  - 优先级 2：关键页面的 smoke test（能渲染、能加载数据）
  - 优先级 3：表单交互测试

- [ ] 4.4.3 输出前端测试提升计划

**产出物**：`## 前端测试路线图`

---

## 执行追踪

### 阶段分配建议

| 阶段 | 可并行 | 适合模型类型 | 关键技能 |
|------|--------|-------------|----------|
| 阶段一 | 可和阶段二并行 | 安全审计 + Go | null-safety、契约分析 |
| 阶段二 | 可和阶段一并行 | 架构 + Go | 分层设计、DB schema |
| 阶段三 | 依赖阶段一 | 测试工程 | 覆盖率、测试设计 |
| 阶段四 | 依赖阶段一二 | 前端 | React、TypeScript |

### 进度记录

| 阶段 | 分配模型 | 开始日期 | 完成日期 | 状态 |
|------|---------|----------|----------|------|
| 阶段一 | — | — | — | pending |
| 阶段二 | — | — | — | pending |
| 阶段三 | — | — | — | pending |
| 阶段四 | — | — | — | pending |

---

## 参考文档

- `docs/pitfalls/` — 7 条历史踩坑记录
- `docs/refactoring-plan.md` — 已完成阶段 1-6 的重构记录
- `docs/current/architecture.md` — 当前架构基线
- `docs/conventions/` — 后端/前端/测试规范
