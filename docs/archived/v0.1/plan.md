---
archived: true
archived_at: "2026-04-29"
archived_by: doc-archivist
version: "v0.1"
original_path: "plan.md"
status: "completed"
reason: "v0.1 执行计划，所有里程碑已完成，被 docs/archived/v0.2/执行计划-v0.2.md 取代"
---

# 目标中心自动化安全测试工作台 — 开发执行计划

> PRD 基线：[设计.md](./设计.md)  
> 最后更新：2026-04-26

---

## 技术选型决策

| 领域 | 选项 A | 选项 B | 决策 | 理由 |
|------|--------|--------|------|------|
| 前端状态管理 | Zustand | Redux Toolkit | **Zustand** | MVP 状态不复杂，Zustand 更轻量，API 更简洁 |
| 实时推送 | SSE | WebSocket | **SSE** | MVP 只需服务端→客户端单向推送，SSE 更简单且基于 HTTP |
| 语法高亮 | Prism.js | highlight.js | **Prism.js** | 更轻量，MVP 只需几类高亮，不需要 highlight.js 的庞大生态 |

---

## 里程碑状态

| 里程碑 | 状态 | 目标日期 | 验收标准（详见设计.md §17） |
|--------|------|----------|----------------------------|
| M0 | 🟢 已完成 | 2026-04-26 | Tauri-Go 通信 + Scope Check + Subfinder 最小闭环 |
| M1 | 🟢 已完成 | 2026-04-26 | 目标输入 + Scope Check + 执行计划预览 |
| M2 | 🟢 已完成 | 2026-04-26 | Subfinder/httpx/Naabu + 资产归一 + RawArtifact |
| M3 | 🟢 已完成 | 2026-04-26 | Nuclei + Finding + confidence/priority 评分 |
| M4 | 🟢 已完成 | 2026-04-27 | 验证队列 + Markdown/JSON 报告导出 |

> 状态说明：🟡 进行中 / ⚪ 待开始 / 🟢 已完成 / 🔴 阻塞

---

## 当前 Sprint：M4 报告导出

### Sprint 目标

实现从 confirmed Finding 到可交付报告的最后一步：Markdown 报告生成 + JSON 数据导出。

### 前置条件

M0-M3 已完成。当前已有：
- Project / Target / Scope / Asset / WebEndpoint / Port 数据
- Finding（pending_review / confirmed / false_positive / accepted_risk / ignored）
- Evidence（request / response / note / raw_output）
- RawArtifact（原始工具输出）
- ToolInvocation（工具版本、命令参数）

### 任务清单

- [x] **1. 报告数据聚合**
  - [x] 1.1 按项目 ID 拉取 confirmed + accepted_risk Finding
  - [x] 1.2 关联 Asset / WebEndpoint / Evidence
  - [x] 1.3 聚合工具版本和任务摘要

- [x] **2. Markdown 报告生成**
  - [x] 2.1 报告模板定义（结构：概览/范围/方法/摘要/风险统计/漏洞详情/接受风险/附录）
  - [x] 2.2 漏洞详情渲染（资产/严重性/可信度/证据/复现摘要/修复建议）
  - [x] 2.3 风险统计图表（critical/high/medium/low 分布）
  - [x] 2.4 `GET /projects/:id/reports/export.md`

- [x] **3. JSON 导出**
  - [x] 3.1 JSON Schema 定义（project/scope/assets/findings/evidence/tool_invocations）
  - [x] 3.2 `GET /projects/:id/reports/export.json`
  - [ ] 3.3 与 DefectDojo 导入格式兼容（可选，预留字段）→ 延后到 v0.4

- [x] **4. 前端报告页面**
  - [x] 4.1 Reports 页面（报告大纲预览）
  - [ ] 4.2 漏洞顺序调整（拖拽排序）→ 延后到 v0.2
  - [x] 4.3 Markdown 预览 + 导出按钮

- [x] **5. MVP 端到端验收**
  - [x] 5.1 创建项目 → 导入目标 → 资产发现 → Web 初筛 → 人工确认 → 报告导出
  - [ ] 5.2 单项目 100 目标标准初筛稳定性测试 → 延后

### 验收标准

- Markdown 报告包含：项目摘要、测试范围、方法说明、漏洞列表（含证据）、修复建议、附录
- JSON 可被外部工具解析（字段完整、类型正确）
- 报告导出前提示敏感字段检查（redaction_status 标记）

### 风险与阻塞项

| 风险 | 级别 | 应对措施 | 状态 |
|------|------|----------|------|
| Markdown 模板维护成本高 | 低 | MVP 用硬编码模板，v0.2 引入模板系统 | ⚪ 待评估 |
| JSON 字段与外部平台不兼容 | 低 | 预留 DefectDojo 字段，v0.4 做适配 | ⚪ 待评估 |
| 大量 Finding 时报告生成慢 | 中 | 分页生成，前端流式展示 | 🟡 待验证 |

---

## v0.2 路线图

### Phase 1: 容器化与远程 Worker ✅

| 任务 | 状态 | 文件 |
|------|------|------|
| Docker 镜像构建（server / worker） | ✅ | `Dockerfile.server`, `Dockerfile.worker` |
| Docker Compose 编排 | ✅ | `docker-compose.yml`, `docker-rangefield/docker-compose.yml` |
| 远程 Worker 注册/心跳/长轮询 | ✅ | `internal/api/worker_handlers.go`, `internal/worker/remote_client.go` |
| Worker 超时自动清理 | ✅ | `internal/api/handlers.go` `cleanupStaleWorkers()` |
| WorkersPage 实时列表 | ✅ | `frontend/src/pages/WorkersPage.tsx` |
| Makefile 容器化命令 | ✅ | `Makefile` |

### Phase 2: 模板管理（进行中）

- [ ] Server 端模板仓库管理（上传/版本/校验）
- [ ] Worker 模板更新指令下发（通过 poll 通道）
- [ ] 模板版本同步与差异检测

### Phase 3: 任务调度增强

- [ ] Worker 能力上报（工具版本、并发限制、网络环境）
- [ ] 多 Worker 任务负载均衡
- [ ] 任务优先级队列

---

## Sprint 日志

### 2026-04-26

- 完成：1.1–1.5 SQLite schema 全部表结构 + 迁移
- 完成：2.1/2.3 Project / Target / ScopeRule CRUD API
- 完成：3.1–3.3/3.6–3.8 Scope Check 引擎（域名/URL/IP/CIDR 匹配 + TOCTOU + 13 个单元测试）
- 完成：4.1/4.2/4.3/4.5/4.6 Worker subprocess runner（goroutine、workdir、超时、输出收集、100MB 截断）
- 完成：5.1/5.3/5.5 工具健康检查（binary path、version、DNS、network、workdir）
- 完成：6.1/6.3 统一错误模型定义 + 分级日志
- 完成：7.1–7.4 Tauri 前端骨架（React/TS/Tailwind、Zustand、基础页面、HTTP API 通信）
- 完成：go build 通过 / go test 13 passed
- 阻塞：8.x M0 闭环验收（需 Tauri 与 Go 后端联调运行验证）
- 风险：脱敏过滤器（6.2）和 SIGTERM→SIGKILL 取消控制（4.4）尚未实现，不影响最小闭环
- 下一步：联调验证最小闭环，然后进入 M1

### 2026-04-26 — Critical 修复

**Code Review 发现 4 个 Critical 问题，已修复：**

- ✅ **C1** Worker `Run()` 增加 TOCTOU Scope Check（执行前校验 ScopeDecision freshness，Scope 变更则重新评估，失败标记 `scope_denied`）
- ✅ **C2** API 路由修复（`//` 双斜杠 → Go 1.22 `{id}` wildcard，`cancel`/`artifacts` 端点恢复正常）
- ✅ **C3** 结构化错误模型（新建 `internal/errors/errors.go`，7 种错误码 + HTTP 状态码映射，API 统一返回 `{"error": {"code": "...", "message": "..."}}`）
- ✅ **C4** 线程安全 ID（`internal/util/id.go` 使用 `atomic.Int64`，所有包统一替换 `util.GenerateID()`）

**验证：** `go build` ✅ / `go test` 13 passed ✅ / `go vet` ✅

**遗留（非阻塞，M1 同步做）：**
- ✅ ToolInvocation 持久化（已修复）
- ✅ 取消任务 PID 追踪（已修复）
- ✅ 后台 goroutine context 超时（已修复）

### 2026-04-26 — M0 联调验证（最小闭环）

**验证流程：创建项目 → 添加目标 → Scope Check → Subfinder → 展示结果**

| 步骤 | 操作 | 结果 |
|------|------|------|
| 1 | `POST /projects` 创建项目 | ✅ `id-1777200034357892000-1` |
| 2 | `POST /projects/:id/targets` 添加域名目标 | ✅ `example.com` |
| 3 | `POST /scope-rules` 添加 include 规则 | ✅ `example.com` |
| 4 | `POST /scan-plans/dry-run` 干运行 | ✅ `decision: allow, reason: 命中包含规则` |
| 5 | `POST /tasks/run` 启动 Subfinder | ✅ Task 创建，自动分配 plan_id（FK 修复后） |
| 6 | Worker TOCTOU Scope Check | ✅ 通过，任务进入 `running` |
| 7 | Subfinder 执行 | ✅ 进程运行约 40s，找到 22293 子域名 |
| 8 | 任务完成 | ✅ `status: completed, exit_code: 0` |
| 9 | RawArtifact 保存 | ✅ stdout 1.8MB，SHA256 校验，`redaction_status: unchecked` |
| 10 | `GET /tasks/:id/artifacts` 查看结果 | ✅ JSON 数组，包含路径和校验值 |
| 11 | 前端页面加载 | ✅ Vite dev server `localhost:1420` 正常 |
| 12 | CORS 跨域 | ✅ 前端 Origin `localhost:1420` 可访问 API |

**联调中发现并修复的问题：**

| 问题 | 原因 | 修复 |
|------|------|------|
| 创建任务 FK 约束失败 | `plan_id` 为空字符串违反外键 | handleRunTask 自动创建默认 ScanPlan |
| 获取任务报错 | `worker_id` NULL 无法扫描到 `string` | models.ScanTask.WorkerID 改为 `*string` |
| Scope Check 时间解析失败 | SQLite datetime 格式与 Go RFC3339 不匹配 | GetMaxScopeRuleUpdatedAt 多格式解析 fallback |

**结论：M0 最小闭环验证通过。**

### 2026-04-26 — Major 遗留项修复

**1. ToolInvocation 持久化** ✅
- `internal/db/queries.go`: 新增 `CreateToolInvocation` + `UpdateToolInvocation`
- `internal/worker/worker.go`: `Run()` 启动时写入 ToolInvocation，完成后更新 `finished_at` + `exit_code`

**2. 取消任务 PID 追踪** ✅
- Runner 新增 `doneChs map[string]chan struct{}` + `sync.RWMutex`
- `Run()` 启动进程后注册到 `procs` 和 `doneChs`，`cmd.Wait()` 完成后清理
- `Cancel()` 发送 SIGTERM，监听 `doneCh` 等待退出，5s 后 SIGKILL
- **评审发现竞态**: `Cancel()` 原实现调用 `cmd.Process.Wait()`，与 `Run()` 中的 `cmd.Wait()` 竞争。修复：取消独立 Wait，改为共享 `doneCh`。
- `handleCancelTask()` 同时调用 `worker.Cancel(id)` 终止实际子进程

**3. Context 超时** ✅
- `internal/api/handlers.go`: 新增 `defaultToolTimeout()` 映射（Subfinder/httpx 300s、Naabu/Nmap 600s、Nuclei 1800s、默认 300s）
- `handleRunTask()` goroutine 使用 `context.WithTimeout()` 替代 `context.Background()`
- 超时后任务状态更新为 `failed, exit_code=-1`

**验证:** `go build` ✅ / `go test` 13 passed ✅ / `go vet` ✅

### 2026-04-26 — M1 联调验证（目标与 Scope 增强）

**验证流程：批量导入 → 时间窗口 → 速率限制 → 干运行增强**

| 功能 | 操作 | 结果 |
|------|------|------|
| TXT 导入 | `POST /projects/:id/targets/import` 上传 TXT | ✅ 5 imported, 1 denied (10.0.0.0/8) |
| CSV 导入 | `POST /projects/:id/targets/import` 上传 CSV | ✅ 2 imported, 1 duplicate, 1 denied |
| 时间窗口（通过） | 创建项目 start=2026-04-01, end=2026-12-31 | ✅ 目标允许通过 |
| 时间窗口（拒绝） | 创建过期项目 start=2025-01-01, end=2025-12-31 | ✅ 干运行返回 `decision: deny, reason: 不在测试时间窗口内` |
| 速率限制（存储） | 创建项目 rate_limit=50 | ✅ Project 字段正确保存 |
| 速率限制（拒绝） | 创建项目 rate_limit=-1 | ✅ 400 Bad Request `rate_limit must be >= 0` |
| 干运行增强 | 包含 time_window_valid + rate_limit + estimated_seconds | ✅ 信息完整 |

**Code Review 发现并修复的问题：**

| 问题 | 级别 | 修复 |
|------|------|------|
| 前端文件上传字段名与后端不匹配 | Critical | `api.ts` 字段名 `"targets_file"` → `"file"` |
| `denied_targets` 类型不匹配（无 reason） | Critical | 新增 `DeniedTarget` 结构体，后端填充拒绝原因 |
| `ValidateBeforeRun` 时间窗口 TOCTOU 缺失 | Major | 获取 Project，时间窗口过期或 rate_limit<0 时强制重新 Check |
| `handleRunTask` 缺少 rate_limit 校验 | Major | 新增 `project.RateLimit < 0` 返回 400 |

**验证：** `go build` ✅ / `go test` 52 passed ✅ / `go vet` ✅ / `tsc --noEmit` ✅

---

## 历史 Sprint

（Sprint 结束后归档到此）

---

## M0–M4 执行概要

（详细验收标准见设计.md §17，此处仅记录实际起止日期和验收结果）

### M0：工程骨架
- 开始日期：2026-04-26
- 结束日期：2026-04-26
- Git tag：`v0.1.0-m0`
- 验收结果：✅ 最小闭环验证通过
- 关键交付：SQLite schema、Scope Check 引擎、Worker subprocess runner、HTTP API + SSE、Tauri 前端骨架
- 代码统计：~30 个 Go 文件 + 13 个前端文件
- 测试覆盖：13 个单元测试（Scope Check 引擎）

### M1：目标与 Scope
- 开始日期：2026-04-26
- 结束日期：2026-04-26
- Git tag：`v0.1.0-m1`
- 验收结果：✅ 联调验证通过
- 关键交付：TXT/CSV 批量导入、时间窗口校验、速率限制配置、干运行增强
- 代码统计：+4 新文件（import.go, import_test.go, progress.md 等）
- 测试覆盖：52 个单元测试（新增 39 个，含导入解析 + 时间窗口）

### M2：资产发现
- 开始日期：2026-04-26
- 结束日期：2026-04-26
- Git tag：`v0.1.0-m2`
- 验收结果：✅ Subfinder/httpx/Naabu 联调通过，资产归一化正确
- 关键交付：Discovery 工作流、资产合并器、RawArtifact 保存、WebEndpoint 解析
- 代码统计：+6 新文件（discovery.go, merger.go, parser/*.go 等）
- 测试覆盖：13 个 parser 单元测试

### M3：Nuclei 初筛
- 开始日期：2026-04-26
- 结束日期：2026-04-26
- Git tag：`v0.1.0-m3`
- 验收结果：✅ Nuclei 输出解析 + Finding 生成 + 评分体系
- 关键交付：Nuclei parser、Finding 模型、confidence/priority 评分、Tag Mapper
- 代码统计：+4 新文件（nuclei.go, scoring.go, tagmapper.go 等）
- 测试覆盖：15 个单元测试（含 tagmapper_test.go）

### M4：人工验证与报告
- 开始日期：2026-04-27
- 结束日期：2026-04-27
- Git tag：`v0.1.0-m4`
- 验收结果：✅ 端到端验收通过（9 目标 → 86 资产 → 报告导出）
- 关键交付：report 包（Aggregate/Markdown/JSON）、ReportsPage 前端、export.md/.json API
- 代码统计：+5 新文件（report/*.go, ReportsPage.tsx）
- 测试覆盖：15 个 report 单元测试
- 端到端：创建项目 → 导入 9 目标 → 资产发现 → Nuclei 扫描 → 人工确认 → Markdown/JSON 报告导出
- Bug 修复：Evidence `created_by` NULL 处理（ListEvidenceByFinding Scan 错误）

---

## v0.2 路线图

> 目标：从单机 MVP 过渡到可分布式部署的生产就绪扫描平台

### Phase 1: 容器化与远程 Worker ✅
- Docker 镜像构建（server / worker 双镜像）
- Docker Compose 编排（anchor-net 统一网络）
- Worker 注册 / 心跳 / 长轮询机制
- 幽灵 Worker 自动清理（心跳超时标记 offline）
- 部署场景支持：内网 Docker、公网 VPS、家庭 WiFi 笔记本

### Phase 2: 模板管理（进行中）
- Server 端模板仓库管理（上传 / 版本 / 分类）
- Worker 模板更新指令下发（通过 poll channel）
- 模板版本同步与校验（SHA256 / 签名）

### Phase 3: 任务调度增强
- Worker 能力上报（工具版本、并发限制、网络画像）
- 多 Worker 任务负载均衡（最少连接 / 轮询）
- 任务优先级队列（紧急扫描优先）

### Phase 4: 可观测性与运维
- Worker 日志流回传（SSE / WebSocket）
- 任务执行链路追踪（task_id → run_id → worker_id）
-  metrics 接口（Prometheus 格式）

### 风险与阻塞项
| 风险 | 级别 | 应对措施 |
|------|------|----------|
| Worker 在内网无公网 IP，Server 无法主动推送 | 低 | 长轮询 + 指令队列已解决 |
| 大量 Worker 同时轮询导致 Server 负载高 | 中 | v0.2 Phase 3 引入 WebSocket 替代长轮询 |
| Nuclei 模板体积大，分发慢 | 中 | 增量更新 + diff 同步 |
