---
status: active
owner: claude (stage3 testing audit)
audit_date: 2026-05-13
audit_scope: 测试策略 & 覆盖率
source_of_truth: false
parent: ../code-health-audit.md
---

# 阶段三：测试策略 & 覆盖率审计报告

> 只审计，不改代码、不写测试。数据采集自 `go test -coverprofile` + 静态分析 + 测试文件人工抽查。

## 摘要

| 指标 | 数值 | 阈值 | 状态 |
|------|------|------|------|
| Go 总覆盖率（含 ./internal/...）| **21.0%** | >30% | 不达标 |
| Go 测试文件数 | 28 (16% 文件覆盖率) | >40% | 不达标 |
| 前端单元测试文件 | 4 (193 行) | — | 几乎空白 |
| 前端 E2E 测试 | 22 个 spec (1799 行) | — | 充足但层级失衡 |
| 覆盖率 0% 的 internal 子包 | 11 / 24 | 0 | **Critical** |
| 安全关键包 worker 覆盖率 | 0% | >50% | **Critical** |
| 安全关键包 workflow 覆盖率 | 8.9% | >50% | **High** |
| 安全关键包 api 覆盖率 | 12.5% | >40% | **High** |

**核心结论**：测试栈呈倒金字塔。E2E 多（22 个 spec）但依赖 docker 全栈；中间层（集成 + service 单测）几乎空白；最危险的 worker / workflow / scope 入口几乎无单测保护。

---

## 3.1 当前测试质量审计

### 3.1.1 包级覆盖率矩阵

> 来源：`go test -coverprofile=coverage.out ./internal/...` (2026-05-13)

| 包 | 覆盖率 | 安全关键 | 风险等级 | 备注 |
|----|--------|----------|----------|------|
| `internal/nuclei` | **89.1%** | — | Low | 测试金字塔典范 |
| `internal/safefs` | 87.5% | 是 | Low | 路径安全核心，已充分 |
| `internal/parser` | 75.5% | 是 | Low | 输出解析，覆盖良好 |
| `internal/scope` | **70.7%** | 是 | Medium | 高分但仍漏 `Engine.Check`/`ValidateBeforeRun`/`matchIP`/`matchURL` |
| `internal/report` | 53.8% | 是 | Medium | markdown/json 双路径已覆盖，缺 finding aggregation 边界 |
| `internal/asset` | 47.1% | — | Medium | 仅 normalizer，merger 0 |
| `internal/nuclei/custom` | 40.4% | 是 | Medium | manager + git 已覆盖 |
| `internal/db` | 25.8% | 是 | **High** | 177 个 query 函数仅覆盖 finding_template / nuclei_custom / seed |
| `internal/service` | 14.1% | 是 | **High** | 4 个 service 仅 project 有单测 |
| `internal/api` | **12.5%** | 是 | **High** | 130 个 handler，仅 project CRUD + nuclei_custom 有测 |
| `internal/workflow` | **8.9%** | **是** | **Critical** | 76 个函数，discovery.go 公开方法 0 覆盖 |
| `internal/cdn` | 0% | 是 | **Critical** | CDN 检测未测，影响 scope 决策 |
| `internal/dictionary` | 0% | — | Medium | 字典加载、写入逻辑 |
| `internal/errors` | 0% | — | Low | 错误类型 |
| `internal/fingerprint` | 0% | — | Medium | 指纹库匹配 |
| `internal/health` | 0% | — | Low | 健康检查 |
| `internal/httpxfp` | 0% | — | Medium | httpx 指纹增强 |
| `internal/models` | 0% | — | Low | 数据结构（多为 struct） |
| `internal/resolve` | 0% | 是 | High | DNS 解析 |
| `internal/scoring` | 0% | — | Medium | Finding 评分 |
| `internal/search` | 0% (代码有但 test 不跑) | 是 | **High** | FOFA/Hunter/Quake live test 需网络 |
| `internal/util` | 0% | — | Low | 工具函数 |
| `internal/worker` | **0%** | **是** | **Critical** | 730L server.go + 405L worker.go，零测试 |

### 3.1.2 已有测试深度抽查（5 个文件）

| 文件 | 行数 | 用例数 | 价值定级 | 评估 |
|------|------|--------|----------|------|
| `internal/parser/httpx_test.go` | 132 | 7 | 中价值（工具函数 + 真实样本） | 表驱动 + 真 fixture，质量高 |
| `internal/scope/scope_test.go` | 415 | 5 | **高价值**（业务逻辑） | 覆盖 domain/IP/CIDR 三轨匹配；缺 URL 重定向、CDN 边界 |
| `internal/db/queries_finding_template_test.go` | 136 | 3 | 中价值（DB roundtrip） | sqlite memory + roundtrip，模式正确，但 finding/asset/scan 等核心查询无对应测试 |
| `internal/api/handlers_test.go` | 297 | 5 | 中价值（HTTP roundtrip） | httptest+sqlite，可扩展性强，但仅测 project，缺核心查询/写入路径 |
| `internal/workflow/slow_scan_test.go` | 250+ | 7 | **高价值**（业务逻辑回归） | 直接源自一次 ffuf 字典踩坑的回归，正是金字塔中段应有的样子 |

**结论**：测试模式没问题，**缺的是数量和分布**。`slow_scan_test.go` / `stageemitter_test.go` 这种从踩坑回归的写法，是模板，应该复用到 scope/discovery/worker 路径。

### 3.1.3 测试 CRUD 占比

- 已测 28 个 Go 测试文件总共约 130 个 `func Test*`
- 其中 `internal/scope/*_test.go` 一家占了 18 个用例
- 其余 20+ 个测试文件平均不到 5 个用例

---

## 3.2 关键路径测试缺口清单

完整扫描流程（来自审计计划）：

```
Target 导入 → Scope Check → 资产发现(FOFA/Subfinder) → DNSx → CDN → Naabu → nmap → httpx → Nuclei → Finding → 人工验证 → Report
```

### 3.2.1 流程节点测试覆盖矩阵

| 流程节点 | 关键函数（文件:行号） | 测试状态 | 风险 | 缺口 |
|----------|----------------------|----------|------|------|
| Target 导入 | `internal/scope/import.go:46 ParseTXT` / `:71 ParseCSV` / `:119 parseLine` | 有 (`scope/import_test.go` 13 case) | Low | 已覆盖 CSV/TXT/范围/逗号扩展 |
| Scope Check（域名） | `internal/scope/scope.go:168 matchDomain` / `:183 matchDomainRule` | 有 (单元) | Medium | 单测覆盖正则，但 `Engine.Check`（含 DB 调用）无集成测试 |
| Scope Check（IP/CIDR） | `internal/scope/scope.go:290 matchIP` / `:217 ExpandCIDR` | 有 | Medium | 缺 IPv6、CIDR 越界、CDN IP 场景 |
| Scope Check（URL） | `internal/scope/scope.go:256 matchURL` | **0%** | **High** | URL 重定向后域名重检场景完全无覆盖 |
| Scope ValidateBeforeRun | `internal/scope/scope.go:85` | **0%** | **High** | Worker 启动前 gate，零测试，绕过风险无回归 |
| 资产发现 - FOFA | `internal/search/live_test.go` | 有 mock | Medium | 需要凭据，CI 默认 skip；live 测试不计入覆盖率 |
| 资产发现 - Subfinder | `internal/parser/subfinder_test.go` + `internal/workflow/discovery.go:485 parseSubfinderOutput` | 解析有 / 编排 **0%** | **High** | parseSubfinderOutput / `runDomainChain` 无测试 |
| DNSx 解析 | `internal/resolve/` | **0%** | **High** | 包覆盖率 0%，DNS 解析失败/超时行为无回归 |
| CDN 检测 | `internal/cdn/` | **0%** | **Critical** | 整个包零测试，CDN IP 是否进 scope 决策无保护 |
| Naabu 端口扫描 | `internal/workflow/discovery.go:423 buildNaabuArgsWithPortRange` / `:537 parseNaabuOutput` | args 已测 / parse **0%** | High | parseNaabuOutput 路径未覆盖 |
| HTTPX 探活 | `internal/parser/httpx_test.go` + `internal/workflow/discovery.go:511 parseHttpxOutput` | parser 有 / 编排 **0%** | High | parseHttpxOutput / fingerprint 合并未覆盖 |
| Nuclei 扫描 | `internal/nuclei/` 89.1% + `internal/parser/nuclei_test.go` | 有 | Low | 高质量 |
| Finding 入库 | `internal/db/queries_finding.go:12 CreateFinding` 等 18 个 | **0%** | **High** | finding CRUD / dedup / status 流转零测试 |
| Finding 状态机 | `internal/service/finding.go:71 UpdateStatus` | **0%** | High | 状态转换合法性无回归 |
| 人工验证 / 证据 | `internal/api/finding_handlers.go:57 handleAddEvidence` | **0%** | Medium | evidence 上传链路无 HTTP 层测试 |
| Report 生成 | `internal/report/report_test.go` (12 case) | 有 | Low | markdown / json 双路径已覆盖 |
| Report 模板优先级 | `internal/db/queries_finding_template_test.go:78 TestGetFindingTemplateForFinding_PriorityFallback` | 有 | Low | DB 层有；service 层组合调用无回归 |

### 3.2.2 完全没有测试的核心函数清单

**discovery.go (workflow 编排核心 — 全部 0%)**

| 函数 | 位置 | 职责 | 风险 |
|------|------|------|------|
| `(*AssetDiscoveryWorkflow).Run` | discovery.go:63 | 资产发现主入口 | **Critical** — 整流水线无回归 |
| `runDomainChain` | discovery.go:105 | 域名 → subfinder → httpx → naabu 编排 | **Critical** |
| `runIPChain` | discovery.go:222 | IP 直扫编排 | **High** |
| `runCIDRChain` | discovery.go:258 | CIDR 展开 + scope 复检 + 扫描 | **Critical** — CIDR 复检逻辑没保护 |
| `runPostDiscovery` | discovery.go:300 | 扫完落库的中枢 | **Critical** |
| `parseSubfinderOutput` / `parseHttpxOutput` / `parseNaabuOutput` | discovery.go:485/511/537 | artifact → 解析 | High — artifact-type 踩坑就是这层 |
| `findArtifactPath` | discovery.go:564 | artifact 路径检索 | High |
| `getPortRange` | discovery.go:405 | 端口范围读取 | Medium |
| `createAndRunTask` | discovery.go:441 | 任务创建 + worker 调度 | **High** |

**worker（server.go + worker.go — 全部 0%）**

| 函数 | 位置 | 职责 | 风险 |
|------|------|------|------|
| `(*Runner).Run` | worker.go:51 | Worker 任务执行入口（215L） | **Critical** |
| `saveArtifact` | worker.go:266 | artifact 落盘（踩坑核心） | **Critical** — `20260426-artifact-type-mismatch` 直接相关 |
| `(*Runner).Cancel` | worker.go:289 | 任务取消 | High |
| `injectCustomNucleiTemplates` | worker.go:324 | 自定义模板注入 | High |
| `isUnreachableError` | worker.go:386 | 网络错误分类 | Medium |
| `executeTask` | server.go:91 | HTTP server 端任务执行（243L） | **Critical** |
| `reportResult` | server.go:334 | 结果回传 core | **Critical** |
| `dynamicRunningTimeout` / `estimateScanScale` | server.go:529/488 | 动态超时（v0.4 新增） | **High** — 超时计算错误会导致 ghost worker |
| `resolveTimeoutConfig` | server.go:584 | 超时配置解析 | High |
| `readProcessState` / `isProcessStateHung` | server.go:654/670 | 进程挂起检测（与 ghost-worker 踩坑直接相关） | **Critical** — `20260428-ghost-worker-cleanup` 配套保护缺失 |

**scope（已覆盖 70% 但仍漏 Engine 入口）**

| 函数 | 位置 | 风险 |
|------|------|------|
| `(*Engine).Check` | scope.go:27 | **Critical** — 主入口，所有 scope 决策走这里 |
| `(*Engine).ValidateBeforeRun` | scope.go:85 | **Critical** — worker 启动前 gate |
| `(*Engine).CheckIP` | scope.go:248 | **High** — CIDR 展开后逐 IP 检查走这里 |
| `(*Engine).matchURL` | scope.go:256 | High — URL scope 0% 覆盖 |
| `(*Engine).matchIP` | scope.go:290 | High — IP scope 引擎方法（虽然 MatchIP 单测有，但 Engine 入口无） |

**API handler（130 个函数，只有 5 个 project CRUD 有 HTTP 层测试）**

完全无 HTTP 层测试的关键 handler 文件：
- `asset_handlers.go` (7 个 handler)
- `finding_handlers.go` (4 个，含 patch status / add evidence)
- `pipeline_handlers.go` (9 个，含 start pipeline)
- `report_handlers.go` (10 个，含导出)
- `run_handlers.go` (10 个，含 run 状态查询)
- `scope_handlers.go` (7 个，scope 规则 CRUD)
- `worker_handlers.go` (9 个，含 worker 注册 / 心跳)
- `target_handlers.go` (3 个，含批量导入)
- `engine_handlers.go` (7 个，含搜索代理)
- `dictionary_handlers.go` (8 个)
- `httpx_fingerprint_handlers.go` (7 个)

### 3.2.3 业务影响排序（缺口 → 风险）

| 排名 | 缺口 | 业务影响 | 推荐补测顺序 |
|------|------|----------|--------------|
| #1 | worker 全包 0% | 任务执行错误无回归，artifact-type 类踩坑可能再发 | M1 |
| #2 | workflow/discovery.go 编排 0% | scope 复检、CIDR 展开、流水线编排无任何保护 | M1 |
| #3 | scope `Engine.Check/ValidateBeforeRun` 0% | scope gate 主入口零回归，bypass 风险无监控 | M1 |
| #4 | api/finding + pipeline + scope handler 0% | 契约变更（前后端字段）无 HTTP 层快速反馈 | M2 |
| #5 | db/queries_finding/scan 0% | 字段增删 / NULL 处理（null-scan-crash 类踩坑）易复发 | M2 |
| #6 | cdn / resolve 0% | CDN IP 是否进 scope、DNS 失败回退 — 无保护 | M2 |
| #7 | service 层（finding/target）0% | 业务规则下沉后无单测，重构风险高 | M3 |
| #8 | scoring / fingerprint / httpxfp 0% | Finding 评分 / 指纹 — 影响最终报告质量 | M3 |

---

## 3.3 测试层级分析

### 3.3.1 测试金字塔现状

```
        E2E (Playwright)          22 spec × 平均 80 行 = ~1800 行
        ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
        集成（HTTP roundtrip）     5 个 (api + nuclei_custom)
        ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
        Service 层单测              4 个 (project_test.go)
        ━━━━━━━━━━━━━━━━━━━━━
        DB roundtrip                18 个 (主要 finding_template / nuclei_custom / seed)
        ━━━━━━━━━━━━━━━━━━━━━━━━━
        纯单元（parser/scope/safefs/nuclei） 88+ 个
        ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
        前端单元（vitest）           4 个文件，14 用例
        ━━━━━━━━━━━
```

**问题**：金字塔倒挂——底层（纯单元）相对完善，但**中间层（集成 / service / API HTTP roundtrip）极薄**，顶层 E2E 又非常重（需要 docker compose 全栈 + worker + rangefield 容器），单测/集成层兜不住的 bug 必须等 E2E 才发现，反馈链特别长。

### 3.3.2 E2E 测试分类（22 个 spec）

| 类型 | spec 数 | 列表 |
|------|---------|------|
| **真正 E2E**（需要全栈 docker / 真实 worker） | 6 | full-flow / high-risk-pipeline / internal-scan-live / smoke / qa-regression / v0.4-company-flow |
| **可下沉到 HTTP 集成层**（仅打 API + 验证返回值，不需要 worker 真扫描） | 8 | App / AssetPage / DashboardPage / FindingsPage / ProjectPage / ProjectLayout / ReportsPage / RunsPage / TargetPage / WorkersPage |
| **可下沉到 Vitest 组件测**（纯前端逻辑） | 4 | error-paths / LegacyRouteGuard / SettingsPage / TokenAuth / scan-modal / scan-modal-real |

**说明**：22 个 spec 里只有 6 个真的需要全栈跑，其余 16 个可以下沉到更轻的层（HTTP 测 + Vitest）。

### 3.3.3 E2E mock 保真度评估

- `frontend/e2e/fixtures/api-helpers.ts` (320L) — **直接打真实 API `http://localhost:17421`**，token 硬编码 `p0m32kun`。**不是 mock**，是 fixture helper。
- `frontend/e2e/fixtures/test-data.ts` — 仅提供 `TEST_DATA` 常量（项目名/目标 IP 等），不构造 fake response。
- `fetchList` helper 显式 unwrap `{data, total, page, page_size}`（说明 E2E 跑的是真后端、真分页）。

**结论**：E2E 没有 mock 保真度问题（因为没在 mock）。问题是**所有 22 个 spec 都依赖 docker-compose.e2e.yml 启动**，反馈极慢。

### 3.3.4 前端单元测试现状（4 个文件 × 14 用例）

| 文件 | 用例 | 价值 |
|------|------|------|
| `frontend/src/lib/api.test.ts` | 5 | 低（仅测 APIError retryable + PAGE_ALL 常量） |
| `frontend/src/components/Button.test.tsx` | 4 | 低（render + disabled state） |
| `frontend/src/components/ScanModal.test.tsx` | 3 | **高**（ffuf 字典门控回归测试 — 典型示范） |
| `frontend/src/pages/RunsPage.test.tsx` | 6 | **高**（SSE reducer `mergeStageEvent`，含 slow-scan append 回归） |

**结论**：质量好的样本已有（ScanModal、RunsPage），但只覆盖 2 个组件。976 行的 `api.ts` 几乎没有单测，52 个 TSX 文件只有 4 个有测试。

---

## 3.4 测试提升路线图（3 个里程碑）

### Milestone 1 — 立即（覆盖最高风险路径）

**目标**：补 5 个最高风险测试，把 worker / scope-engine / workflow 三个 Critical 缺口堵上。

| # | 测试 | 位置 | 价值 |
|---|------|------|------|
| M1-1 | `Test_Engine_Check_*` — domain/IP/CIDR/URL 四类 scope 决策的主入口集成测试（mock DB） | 新建 `internal/scope/engine_check_test.go` | 防止 scope bypass 类 Critical 风险静默回归 |
| M1-2 | `Test_AssetDiscoveryWorkflow_Run_*` — discovery 主入口快路径测试（subfinder/httpx/naabu mock 三段） | 新建 `internal/workflow/discovery_test.go` | 流水线编排零保护 → 加 1 个金路径 + 2 个失败路径 |
| M1-3 | `Test_Runner_Run_ArtifactType_*` — worker artifact 类型一致性回归（覆盖 `20260426-artifact-type-mismatch` 踩坑） | 新建 `internal/worker/worker_test.go` | 复发概率最高的踩坑，0 个保护 |
| M1-4 | `Test_FindingQueries_NullColumns_*` — 覆盖 `20260427-null-scan-crash` 踩坑，包括 finding/evidence 的 created_by NULL 场景 | 新建 `internal/db/queries_finding_null_test.go` | DB 层 NULL 处理零回归 |
| M1-5 | `Test_MarkdownReport_PipeEscaping_*` — Finding title / Asset value 含 `|` 时表格不破坏（覆盖 `20260427-markdown-pipe-corruption`） | 扩 `internal/report/report_test.go` | 报告生成质量回归 |

**验收**：5 个新测试全绿，worker 覆盖率从 0% → ≥20%、workflow 从 8.9% → ≥25%、scope 从 70.7% → ≥80%。

### Milestone 2 — 2 周内（覆盖率到 30%）

**目标**：填齐契约层（HTTP handler + DB query）的测试矩阵，把架构分层报告里推 service 的部分配套保护起来。

| 主题 | 范围 | 预估用例 |
|------|------|----------|
| API handler HTTP roundtrip | finding / scope / target / pipeline / worker 5 个 handler 文件，每个 4-6 个用例（success + 400 + 404 + 主要 query 参数） | 25 |
| DB query roundtrip | queries_finding / queries_scan / queries_asset 三个文件主路径 | 20 |
| Service 层单测 | finding service（含 status 状态机）/ target service（含 import 链路）补测 | 12 |
| Worker 单元 | timeout 计算（`estimateScanScale` / `dynamicRunningTimeout`）+ 进程挂起检测 (`isProcessStateHung`) | 8 |
| Scope 边界 | matchURL、CDN IP、CIDR 越界、ValidateBeforeRun | 6 |
| 前端关键页面 smoke test | TargetPage / FindingsPage / AssetPage 用 RTL + 替身 api，验证渲染 + 状态切换 | 9 |

**验收**：
- 总覆盖率 21% → ≥30%
- api 包覆盖率 12.5% → ≥35%
- db 包覆盖率 25.8% → ≥40%
- workflow 包 ≥30%
- 22 个 E2E spec 中至少 5 个下沉到 HTTP 集成层

### Milestone 3 — 1 月内（覆盖率到 50%，关键路径全覆盖）

**目标**：补齐所有 0% 包，把 E2E 金字塔顶端瘦下来。

| 主题 | 范围 |
|------|------|
| 0% 包补测 | cdn / resolve / dictionary / fingerprint / httpxfp / scoring 六个包，每包至少 3 个核心函数有测试 |
| 集成层加固 | 补 archive / dashboard / dictionary / engine / httpx_fingerprint / nuclei_custom_template handler 的 HTTP 测试 |
| E2E 瘦身 | 把 8 个"可下沉到 HTTP 测"的 spec 改为 vitest+RTL，保留 6 个真正 E2E（full-flow / high-risk-pipeline / smoke / internal-scan-live / qa-regression / v0.4-company-flow）|
| 前端单测 | api.ts 按 domain 拆分后，每个文件配单测（约 20 个用例）；ScanModal / RunsPage / TargetPage 三个大组件每个 ≥5 用例 |
| 契约测试 | 用 stage1 报告的契约差异表，为每个差异写 1 个 vitest 用例（前端解析后端真实响应不崩） |

**验收**：
- 总覆盖率 ≥50%
- 11 个 0% 包全部 ≥30%
- 安全关键包（worker / workflow / scope / api / db）≥50%
- E2E spec ≤8 个，单次 E2E 套件 wall time ≤15 分钟（当前估计 30+ 分钟）

---

## 附录：测试编写参考样本（已有的、可复用的模式）

| 想测什么 | 抄哪个测试 |
|----------|------------|
| 工具函数 / 解析器 | `internal/parser/httpx_test.go`（表驱动） |
| 业务规则 / 状态机 | `internal/workflow/slow_scan_test.go`（场景 + 断言执行序） |
| HTTP handler roundtrip | `internal/api/handlers_test.go`（sqlite memory + httptest） |
| DB query roundtrip | `internal/db/queries_finding_template_test.go`（sqlite memory + Migrate） |
| 集成 E2E 风格 | `internal/workflow/pipeline_e2e_test.go`（带 worker mock） |
| 前端组件回归 | `frontend/src/components/ScanModal.test.tsx` / `frontend/src/pages/RunsPage.test.tsx` |
| 前端 E2E（真后端） | `frontend/e2e/tests/smoke.spec.ts` / `full-flow.spec.ts` |

## 附录：原始数据

- 覆盖率 profile: `/Users/kun/DEV/p0m32kun/coverage.out`
- 测试函数清单：见正文 3.2.2
- 包级覆盖率原始输出 (`go test -cover ./internal/...`)：
  ```
  api 12.5%  asset 47.1%  cdn 0%  db 25.8%  dictionary 0%  errors 0%
  fingerprint 0%  health 0%  httpxfp 0%  models 0%  nuclei 89.1%
  nuclei/custom 40.4%  parser 75.5%  report 53.8%  resolve 0%
  safefs 87.5%  scope 70.7%  scoring 0%  search 0%  service 14.1%
  util 0%  worker 0%  workflow 8.9%
  ```
