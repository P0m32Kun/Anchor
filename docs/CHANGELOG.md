# Anchor 变更日志

> 按时间倒序记录项目关键变更、决策和里程碑。

---

## 2026-05-07 — QA 修复 + FOFA email 清理

### 关键修复:`scan_tasks.run_id` 外键 (问题3)

**根因**:`migrateV02RunID` 在 v0.2 时给 `scan_tasks` 添加了 `run_id TEXT REFERENCES runs(id)`,但 v6 之后应用真正写入的是 `pipeline_runs.id`。每次创建扫描任务都被 FK 检查否决,日志只留一行 `naabu: create scan task: FOREIGN KEY constraint failed`,UI 上看到"扫描 1 秒就完成且无结果"。

**修复**:`migrateV12` 重建 `scan_tasks` 把 FK 改指向 `pipeline_runs(id)`,无效 run_id 置 NULL。回归测试 `internal/db/migrate_v12_test.go` 已沉淀。

### Stage 进度修复 (问题2)

`runCIDRFlow` 里 naabu 报错时只记日志没调 `failStage`/`completeStage`,导致端口扫描阶段卡 running 直至收尾被误标 completed。已补全分支。

### FOFA email 清理

FOFA V1 API 不再需要 email,清掉:
- `internal/search/fofa.go::NewFofaClient(apiKey)` 单参,请求 URL 不带 `email=`
- `internal/workflow/pipeline.go::Pipeline.WithFOFA(apiKey)` 单参
- `models.Project` 删 `FofaEmail`/`FofaAPIKey`,`models.EngineCredential` 删 `Email`
- `migrateV11` 重建 `projects` 和 `engine_credentials` 去掉对应列
- 前端 `EngineKeysPage.tsx` 移除 email 输入

### UI 修复

- ScanModal 端口范围卡片溢出 (问题1):卡片加 `min-w-0` + 描述文本 `break-words`
- ScanModal 速度选项不持久 (问题4):配置 + 模式持久化到 localStorage(`anchor.scanModal.config` / `anchor.scanModal.mode`)

### 沉淀的测试

- `internal/db/migrate_v12_test.go` — Go 单元测试,验证 FK 修复后 scan_tasks 能用 pipeline_runs.id 插入
- `frontend/e2e/tests/qa-regression.spec.ts` — Playwright spec,覆盖问题1+4

### E2E 实测验证

| 问题 | 验证证据 |
|------|---------|
| 1 | playwright `qa-regression.spec.ts` 实跑 PASS |
| 2 | 真实扫描后 stages 时间线独立(portscan 4s / fingerprint 2s / httpx 2s / vuln 24s) |
| 3 | 真实扫描 naabu 找到 2 端口,日志无 FK 错误 |
| 4 | playwright `qa-regression.spec.ts` 实跑 PASS |

---

## 2026-05-07 — v0.4.0 发布

### v0.4 智能扫描管线正式发布

里程碑 v0.4 五个核心目标全部实现并通过 E2E 验收（详见 `docs/active/review/v0.4-acceptance.md`）：

| Goal | 验证 |
| ---- | ---- |
| 1. 多目标类型（含 company） | `v0.4-company-flow.spec.ts` Step 3 |
| 2. FOFA 自动展开 | `v0.4-company-flow.spec.ts` Step 5/6（3 域名 + 3 IP 展开） |
| 3. 完整 8 阶段扫描管线 | `internal-scan-live.spec.ts`（21 findings，4 IP 全部覆盖） |
| 4. 智能服务指纹（Web + 非 Web） | `internal-scan-live.spec.ts` 验证 nginx/tomcat/grafana/redis/mysql 识别 |
| 5. 指纹驱动 Nuclei tags | `internal-scan-live.spec.ts` 13 个高危 finding |
| 6（新增）. Nuclei 分层扫描 | `scan-modal.spec.ts` + `scan-modal-real.spec.ts` |
| 7（新增）. Nuclei 速率防爆破 | `scan-modal-real.spec.ts` Worker 命令含 `-rlm 30 -c 3` |

### 发布前修复：nerva 命令构建 bug

`internal/worker/commands.go::BuildNervaCommand` 误用 nerva 标志：
- 原：`-w` 给 workers（错），`-T` 给 timeout（不存在的标志）
- 实际：`-W` 是 workers，`-w` 是 timeout（毫秒）
- 修复：`-W <workers> -w <timeout*1000>`（秒转毫秒）

修复前 nerva 全部失败 → redis/mysql 服务未被指纹识别 → Nuclei 仅扫 Web 服务。修复后 internal-scan-live 从 2/4 IP 命中变为 4/4 IP 命中（21 findings）。

### 本次发布前的最后一批改动

- **前端**：`TargetPage.tsx` 下拉新增 `company` 选项
- **后端**：`internal/search/fofa.go` 支持 `FOFA_BASE_URL` 环境变量覆盖（用于 E2E mock）
- **E2E 基础设施**：
  - 新增 `frontend/e2e/fixtures/fofa-mock.nginx.conf` — nginx 容器返回 FOFA 假数据
  - 新增 `frontend/e2e/tests/v0.4-company-flow.spec.ts` — Goals 1+2 验收
  - `docker-compose.e2e.yml` 新增 `fofa-mock` 服务，server 注入 `FOFA_BASE_URL=http://fofa-mock:8888`
- **文档**：
  - 新增 `docs/active/review/v0.4-acceptance.md` — v0.4 验收清单
  - `docs/design/v0.4-scan-pipeline.md` 状态从 `in_review` 改为 `accepted`，verification 改为 `passed`
  - `docs/current/plan.md` v0.4 升为 baseline
  - `docs/current/architecture.md` 增补 Company 目标流和 FOFA 自动展开说明
  - `README.md` 版本表新增 `v0.4.0`，功能清单合并 v0.4 章节
  - `wiki/SCHEMA.md` 里程碑 v0.4 改为 ✅ 已完成 + `v0.4.0` tag

---

## 2026-05-07

### Nuclei 分层扫描策略 + 速率防爆破

- **Nuclei 分层扫描**：新增 `PipelineConfig.NucleiScanDepth` 字段（`workflow`/`tags`/`both`）
  - `tags`（默认）：按 httpx 指纹精确匹配 Nuclei 模板（原行为）
  - `workflow`：使用 RBKD-SEC/templates workflow 串联指纹检测和漏洞利用
  - `both`：workflow + tags 双重检测，覆盖最全
- **RBKD-SEC/templates 集成**：`Dockerfile.worker-base` 在镜像构建阶段克隆到 `/opt/rbkd-templates`
- **Nuclei 速率防爆破**：
  - 新增 `nuclei_rate_limit_per_min`（`-rlm`）：每分钟请求数限制，扫描内网敏感目标时使用
  - 新增 `nuclei_concurrency`（`-c`）：并行模板/主机数（之前字段存在但未传入命令）
- **ScanModal UI**：Step 2 新增「Nuclei 扫描策略」三选一卡片 + 「分钟限速」「并发数」输入框
- **E2E 测试**：新增 `frontend/e2e/scan-modal.spec.ts` 和 `scan-modal-real.spec.ts`，覆盖 UI 交互、请求体、Worker 命令完整链路

**后端变更：**
- `internal/models/models.go` — `PipelineConfig` 新增 `NucleiScanDepth`、`NucleiRateLimitPerMinute`，`DefaultPipelineConfig` 默认 scan depth = "tags"
- `internal/worker/commands.go` — `BuildNucleiCommand` 函数签名重构：`(targetFile, profile string, rateLimit, rateLimitPerMin, concurrency int, tags []string, scanDepth string, workflowDir string)`，按 scanDepth 分支决定 `-w` / `-tags` / 二者并用
- `internal/workflow/pipeline.go` — 新增 `DefaultWorkflowDir = "/opt/rbkd-templates/workflows"` 常量，`runNucleiWeb` / `runNucleiNonWeb` 从 config 读取参数
- `internal/workflow/screenshot.go` — `BuildNucleiCommand` 调用更新（向后兼容，默认 "tags"）
- `internal/api/pipeline_handlers.go` — `buildConfigForMode` 处理 `NucleiScanDepth` 默认值

**前端变更：**
- `frontend/src/lib/api.ts` — `PipelineConfig` 接口新增 `nuclei_scan_depth` 和 `nuclei_rate_limit_per_min`，默认值同步
- `frontend/src/components/ScanModal.tsx` — 新增 `SCAN_DEPTH_OPTIONS` + 「Nuclei 扫描策略」面板；BASE_TOOL_FIELDS Nuclei 区域新增「分钟限速」字段，「并发数」最小值改为 1

**Docker 变更：**
- `Dockerfile.worker-base` — apk 添加 `git`，新增 `RUN git clone --depth 1 https://github.com/RBKD-SEC/templates.git /opt/rbkd-templates`

**E2E 测试基础设施：**
- `frontend/e2e/scan-modal.spec.ts` — 3 个 TC：UI 显示验证、参数修改验证、请求体验证
- `frontend/e2e/scan-modal-real.spec.ts` — 真实目标项目，验证 Worker 实际接收到 `-w /opt/rbkd-templates/workflows -c 3 -rlm 30 -rl 100`
- `frontend/playwright.e2e-minimal.config.ts` — 不依赖 global-setup 的最小配置（用于已运行的 Docker 环境）

**验证结果：**
Worker 实际执行的 Nuclei 命令（来自 anchor-worker 容器日志）：
```
nuclei -jsonl -l <targets-file>
       -severity critical,high,medium,low,info -timeout 10
       -w /opt/rbkd-templates/workflows
       -c 3 -rlm 30 -rl 100
       -t /data/nuclei/custom/bundles/current/templates
```

---

## 2026-05-05

### 搜索引擎集成 + 扫描流程重构完成

- **互联网搜索引擎页面**：新增全局 `/engines` 页面，集成 FOFA、Hunter、Quake 三大平台
- **API Key 全局配置**：`/engines/keys` 统一管理 FOFA Email+Key、Hunter Key、Quake Key，存储于 `engine_credentials` 表
- **FOFA 凭证迁移**：从项目级 `fofa_email`/`fofa_api_key` 迁移到全局表，Pipeline 自动读取
- **扫描模式重构**：从 4 种模式（quick/standard/deep/custom）简化为 2 种：`external`（外网）和 `internal`（内网）
- **ScanModal 两步式**：Step 1 选择模式 → Step 2 配置端口范围 + 各工具速度参数（rate/threads/timeout）
- **删除 ScanConfigPage**：所有扫描配置并入 ScanModal，不再保留独立配置页面
- **dnsx 集成**：dnsx CLI 替代 Go `resolver` 模块作为默认 DNS 解析方式
- **PipelineConfig 字段重构**：
  - `dns_concurrency`/`dns_timeout` → `dnsx_threads`/`dnsx_timeout`
  - `port_scan_timeout`/`port_scan_concurrency` → `naabu_timeout`/`naabu_threads`
  - `nerva_concurrency` → `nerva_workers`
  - 新增各工具 `rate_limit` 字段（subfinder/naabu/nerva/httpx）
- **Worker 命令构建器参数化**：所有 `BuildXxxCommand` 函数支持传入速度参数

**后端变更：**
- `internal/db/db.go` — Migration v9: `engine_credentials` 表 + FOFA 凭证迁移
- `internal/models/models.go` — `EngineCredential` 模型，`PipelineConfig` 扩展速度参数字段
- `internal/db/queries.go` — `GetEngineCredential` / `ListEngineCredentials` / `SaveEngineCredential` / `DeleteEngineCredential`
- `internal/search/hunter.go` / `quake.go` / `engine.go` — Hunter/Quake 客户端 + 统一接口
- `internal/api/engine_handlers.go` — `GET/POST/DELETE /engines/credentials`, `GET /engines/search`
- `internal/api/pipeline_handlers.go` — `buildConfigForMode` 重构为 external/internal，`handleCreateScan` 接收 `config`
- `internal/worker/commands.go` — 所有命令构建器支持速度参数
- `internal/workflow/pipeline.go` — dnsx 集成（`runDNSx`），调用命令构建器时传入 config 速度参数
- `internal/parser/dnsx.go` — dnsx JSONL 解析器
- `Dockerfile.worker-base` — 安装 dnsx

**前端变更：**
- `frontend/src/pages/EnginesPage.tsx` — 搜索引擎搜索页面
- `frontend/src/pages/EngineKeysPage.tsx` — API Key 配置页面
- `frontend/src/components/ScanModal.tsx` — 两步式扫描弹窗（模式选择 + 速度配置）
- `frontend/src/lib/api.ts` — `listEngineCredentials` / `saveEngineCredential` / `deleteEngineCredential` / `searchEngine`
- `frontend/src/components/Navbar.tsx` — 新增 "搜索引擎" 全局导航
- `frontend/src/App.tsx` — 注册 `/engines` 和 `/engines/keys` 路由
- `frontend/src/pages/ScanConfigPage.tsx` — 已删除

**验证：**
- `go build ./...` ✅ / `go test ./...` ✅
- `npx tsc --noEmit` ✅

---

## 2026-04-29

### v0.2 Phase 2: 项目管理功能修复与体验优化完成 ✅
- **项目管理入口修复**：Navbar 添加 "Projects" 导航项，App.tsx 注册 `/projects` 路由
- **项目创建**：ProjectPage 支持创建项目（名称、组织、目的、时间窗口、速率限制）
- **项目删除**：支持删除项目，二次确认对话框，级联删除所有关联数据（数据库 `ON DELETE CASCADE`）
- **项目选择**：点击项目卡片 → 设置 `currentProject` → 跳转 Dashboard，后续页面基于此项目操作
- **Dashboard 快捷入口**："当前项目: 未选择" → 可点击 "前往创建 →" 跳转到项目管理

**后端变更：**
- `internal/db/queries.go` — 新增 `DeleteProject(id)` 方法
- `internal/api/handlers.go` — 新增 `DELETE /projects/{id}` 路由 + `handleDeleteProject` handler（含审计日志）

**前端变更：**
- `frontend/src/lib/api.ts` — 新增 `deleteProject(id)` API 方法
- `frontend/src/pages/ProjectPage.tsx` — 删除按钮（hover 显示）+ 确认对话框 + 级联删除
- `frontend/src/components/Navbar.tsx` — 添加 "Projects" 导航入口
- `frontend/src/App.tsx` — 注册 `/projects` 路由
- `frontend/src/pages/DashboardPage.tsx` — "前往创建" 快捷链接

**验证：**
- `go build` ✅ / `go test` ✅ / `go vet` ✅
- `npx tsc --noEmit` ✅

**Tag:** `v0.2.0-p2`

---

## 2026-04-28

### v0.2 Phase 1: 容器化与远程 Worker 架构完成 ✅
- Docker 容器化：`Dockerfile.server` / `Dockerfile.worker` 多阶段构建
- `docker-compose.yml`：Server + Worker 分离，共享 `anchor-net` 网络
- `docker-rangefield/docker-compose.yml`：靶场环境接入 anchor-net
- `Makefile`：新增 `up` / `down` / `up-all` / `down-all` / `range-up` / `range-down` / `shell-server` / `shell-worker`
- 远程 Worker 注册/心跳/长轮询：`internal/api/worker_handlers.go`
- Worker 超时自动清理：`cleanupStaleWorkers()` goroutine，120s 阈值
- Ghost worker bug 修复：`internal/api/handlers.go` 过滤 offline worker，`internal/api/worker_handlers.go` 注册逻辑
- WorkersPage 前端修复：`frontend/src/pages/WorkersPage.tsx` 实时拉取 `/workers`，5s 轮询，状态指示灯
- Docker Compose 网络修复：worker 通过 service 名 `server:17421` 连接（非 localhost）
- 新增 Scope 确认流程：首次导入无 scope 规则时自动提示确认
- 目标导入扩展：逗号分隔展开、IP 连字符范围展开、自动类型推断

**相关文件：**
- `docker-compose.yml`
- `Dockerfile.server` / `Dockerfile.worker`
- `internal/api/handlers.go`
- `internal/api/worker_handlers.go`
- `internal/db/queries.go`
- `frontend/src/pages/WorkersPage.tsx`
- `Makefile`

**Tag:** `v0.2.0-p1`

---

## 2026-04-27

### M4 报告导出 + 端到端验收完成 ✅
- 新增 `internal/report/` 包：Markdown 报告（8 章节模板）+ JSON 数据包生成
- 新增 API：GET /projects/:id/reports/export.{md,json}（scope 校验）
- 新增前端 ReportsPage：Finding 列表预览、Markdown 实时渲染（Prism.js）、JSON 下载
- 新增 Finding 关联查询：ListFindingsWithEvidence（含 Evidence 列表）
- 修复：ListEvidenceByFinding NULL `created_by` Scan 崩溃（sql.NullString）
- 修复：Markdown 表格转义（`|` → `\|`）
- 修复：前端 XSS 风险（`dangerouslySetInnerHTML` → `dompurify` 净化）
- 端到端验收：9 个目标 → 86 条资产 → 人工确认 Finding → 报告导出
- 项目模块路径重命名：`github.com/P0m32Kun/Anchor`
- `go build` ✅ / `go test` ✅ / `go vet` ✅ / `npx tsc --noEmit` ✅

**Tag:** `v0.1.0-m4`

---

## 2026-04-26

### M2 资产发现完成 ✅
- 新增 4 张表：assets、ports、services、web_endpoints（含索引，向后兼容）
- 新增数据模型：Asset、Port、Service、WebEndpoint（含 JSON 列序列化）
- 新增 DB 查询方法：CreateAsset/GetAssetByNormalizedValue/UpdateAssetLastSeen/ListAssetsByProject、CreatePort/ListPortsByAsset/PortExists、CreateService/ListServicesByAsset、CreateWebEndpoint/ListWebEndpointsByAsset/ListWebEndpointsByProject/WebEndpointExists
- 新增解析器包 `internal/parser/`：Subfinder JSONL、httpx JSONL（连字符字段兼容）、Naabu JSONL/CSV 自动识别
- 新增资产归一包 `internal/asset/`：NormalizeDomain/NormalizeURL/NormalizeIP、Merger（MergeOrCreateAsset/CreatePortIfNotExists/CreateWebEndpointIfNotExists）
- 新增工作流包 `internal/workflow/`：AssetDiscoveryWorkflow（串行：Subfinder → 解析创建 domain Asset → httpx → 解析创建 WebEndpoint → Naabu → 解析创建 IP Asset + Port）
- 新增 API 端点：POST /projects/:id/workflows/asset-discovery、GET /projects/:id/assets、GET /projects/:id/web-endpoints、GET /assets/:id/ports、GET /assets/:id/services
- 新增前端 AssetPage（资产列表、WebEndpoint 列表、端口列表）+ TargetPage 资产发现入口
- Worker 新增 BuildHttpxCommand / BuildNaabuCommand
- 全部解析器含单元测试（正常 + 异常输入），全部通过 go test ./... / go vet ./...

### M1 目标与 Scope 增强完成 ✅
- 完成目标批量导入（TXT/CSV），支持拖拽上传，自动去重 + Scope Check
- 完成时间窗口校验（Scope Check 中 + handleRunTask 中 TOCTOU 防护）
- 完成速率限制配置（Project 表 `rate_limit` 列，Worker 自动映射工具参数）
- 完成执行计划预览增强（干运行返回时间窗口/速率限制/预估时间）
- Scope Check 新增时间窗口 + rate_limit >= 0 校验，新增 13 个单元测试
- 前端 ProjectPage 支持时间窗口和速率限制配置
- 前端 TargetPage 支持文件导入、导入统计展示、项目状态面板
- 修复前端 RunsPage TypeScript 编译错误

### M0 工程骨架完成 ✅
- 完成 SQLite schema（10 张表）
- 完成 Project / Target / ScopeRule CRUD API
- 完成 Scope Check 引擎（域名/URL/IP/CIDR 匹配 + 排除优先 + TOCTOU 防护）
- 完成 Worker subprocess runner（goroutine、workdir 隔离、超时、输出截断 100MB）
- 完成工具健康检查（binary path、version、DNS、network）
- 完成统一错误模型（7 种结构化错误码）
- 完成 HTTP API + SSE 实时推送
- 完成 Tauri 前端骨架（React/TS/Tailwind、Zustand、基础页面）
- 完成取消任务（SIGTERM → 5s → SIGKILL）
- 完成 ToolInvocation 持久化

### Agent 体系建立
- 创建 4 个专业 Agent：`frontend-dev`、`backend-dev`、`tech-advisor`、`qa-engineer`
- 安装 6 个新 skill：`tauri-v2`、`golang-pro`、`golang-testing`、`golang-performance`、`tailwind-design-system`、`rust-async-patterns`
- 创建 5 个 Chain 模板：`feature-dev`、`bug-fix`、`refactor`、`security-audit`、`arch-decision`
- 优化 Agent system prompt，内嵌 skill 自动加载规则

### 项目 Wiki 初始化
- 创建 `wiki/` 目录结构
- 创建 `SCHEMA.md`、`index.md`、`log.md`
- 创建 6 个 ADR 初始记录
- 创建前端/后端/API 约定文档

---

## 2026-05-01

### v0.3: 桌面可用性与可靠性完成 ✅
- **网络服务扫描**：Web 初筛工作流扩展支持非 Web 端口（Redis/MySQL/PostgreSQL/Elasticsearch/MongoDB/Memcached/MSSQL/Oracle）
  - 新增 `MapPortToTag` / `GroupPortsByTags` / `PortTarget`（`internal/nuclei/tagmapper.go`）
  - 新增 `runNetworkServiceScan()` 集成到 `WebScreeningWorkflow`（`internal/workflow/screenshot.go`）
  - 按服务标签分组执行 Nuclei，Finding 去重/评分/证据逻辑与 Web 端一致
- **CPE 指纹补充**：httpx 解析器从 CPE 字段提取 product name 作为 tech fallback
  - 解决 404/302 页面 tech 为空被跳过的问题（`internal/parser/httpx.go`）
- **httpx 增强**：`-follow-redirects` 参数（`internal/worker/worker.go`）
- **靶场修复**：Tomcat 靶场 Dockerfile 适配新版镜像（webapps.dist 复制、RemoteCIDRValve 放开、移除 LockOutRealm）

**相关文件：**
- `internal/nuclei/tagmapper.go` / `tagmapper_test.go`
- `internal/parser/httpx.go` / `httpx_test.go`
- `internal/workflow/screenshot.go`
- `internal/worker/worker.go`
- `docker-rangefield/apps/tomcat-vuln/Dockerfile`

**验证：**
- `go build` ✅ / `go vet` ✅ / `go test` ✅（新增 4 个单元测试全部通过）
- `npx tsc --noEmit` ✅

**Tag:** `v0.3.0`

---

## 更早

- 项目初始化（Tauri + Go 骨架）

## 2026-04-26 — M3 完成

**交付：**
- Nuclei 命令构建（BuildNucleiCommand：light/standard 策略、-jsonl 输出、文件输入、-rl 速率限制）
- Nuclei JSONL 解析器（internal/parser/nuclei.go），支持嵌套 info 对象提取，含单元测试
- Finding / Evidence 数据模型 + SQLite schema（含索引 dedup_key、status）
- Finding/Evidence DB 查询方法（CreateFinding/GetFindingByDedupKey/UpdateFindingEvidence/ListFindingsByProject/ListFindingsByStatus/CreateEvidence/ListEvidenceByFinding）
- HTTP 脱敏工具（internal/util/sanitizer.go）：Authorization/Cookie/Set-Cookie/X-Api-Key/Api-Key 正则替换
- Scoring 评分引擎（internal/scoring/scoring.go）：confidence/priority 规则评分，支持可解释原因列表
- Web 初筛工作流（internal/workflow/screenshot.go）：批量 Scope Check → Nuclei 扫描 → JSONL 解析 → dedup_key 去重 → Finding 创建/更新 → Evidence 保存（脱敏）
- API 端点：POST /projects/:id/workflows/web-screening、GET /projects/:id/findings、GET /findings/:id、PATCH /findings/:id/status、POST /findings/:id/evidence
- 前端 FindingsPage：列表（severity/confidence/priority/status 筛选）、详情弹窗（来源信息、Evidence 列表、状态变更、添加备注）
- App.tsx 添加 Findings 路由

**验证：**
- `go build` ✅ / `go test` 71 passed ✅ / `go vet` ✅
- `npx tsc --noEmit` ✅

**Tag:** `v0.1.0-m3`

---

## 2026-04-26 — M2 完成

**交付：**
- Subfinder/httpx/Naabu 工具解析器
- Asset/Port/Service/WebEndpoint 数据模型 + SQLite schema
- 资产归一（Normalizer + Merger）
- 资产发现工作流（串行：Subfinder → httpx → Naabu）
- API 端点（资产列表、WebEndpoint 列表、端口/服务查询）
- 前端 AssetPage

**关键修复：**
- httpx/Naabu 命令行参数过长（37590 个域名）→ 改为文件输入（`-l` / `-list`）

**验证：**
- `go build` ✅ / `go test` 68 passed ✅ / `go vet` ✅
- Subfinder 对 example.com 发现 37590 子域名
- Assets API 正常返回

**Tag:** `v0.1.0-m2`

## 2026-04-26 — M3 完成（含指纹驱动优化）

**交付：**
- Nuclei 集成（BuildNucleiCommand，支持 light/standard 策略）
- Nuclei JSONL 解析器（嵌套 info 对象提取，单行容错）
- Finding/Evidence 数据模型 + SQLite schema + CRUD
- 脱敏工具（SanitizeHTTPHeaders，Authorization/Cookie/Api-Key 等）
- Scoring 引擎（confidence/priority 规则评分，可解释原因列表）
- Web 初筛工作流（Scope Check → Nuclei → 解析 → dedup → Finding 创建）
- API 端点（工作流启动、Finding CRUD、状态变更、Evidence 添加）
- 前端 FindingsPage（列表、筛选、详情弹窗、状态变更）

**指纹驱动优化（M3 后期补充）：**
- 新增 `internal/nuclei/tagmapper.go`：httpx fingerprint → 精确 Nuclei tag 映射
- 按 tag 集合分组扫描：进程数 = 唯一 tag 集合数（不是 URL 数）
- 无指纹目标自动跳过，不浪费扫描资源
- 映射规则：Apache Druid → apache-druid（不含 apache），phpMyAdmin → phpmyadmin（不含 php）

**Code Review 修复：**
- RawArtifact 保存脱敏后数据 → 改为保存原始数据，Evidence.Excerpt 用脱敏版本
- 重复 Finding 评分未更新 → UpdateFindingEvidence 同步刷新评分
- PATCH 对不存在 ID 返回 200 → 增加存在性校验返回 404
- request/response 无大小限制 → 增加 10MB 上限
- httpx 命令行过长（37590 个参数）→ 改为文件输入 `-l hosts.txt`

**验证：**
- `go build` ✅ / `go test` 79 passed ✅ / `go vet` ✅
- 指纹分组验证：wordpress / nginx 分别生成独立任务，无指纹站跳过

**Tag:** `v0.1.0-m3`
