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
| M2 | ⚪ 待开始 | — | Subfinder/httpx/Naabu + 资产归一 + RawArtifact |
| M3 | ⚪ 待开始 | — | Nuclei + Finding + confidence/priority 评分 |
| M4 | ⚪ 待开始 | — | 验证队列 + Markdown/JSON 报告导出 |

> 状态说明：🟡 进行中 / ⚪ 待开始 / 🟢 已完成 / 🔴 阻塞

---

## 当前 Sprint：M1 目标与 Scope 增强

### Sprint 目标

增强目标输入能力和 Scope 校验维度：支持批量导入（TXT/CSV）、时间窗口校验、速率限制配置、执行计划预览增强。

### 任务清单

- [x] **1. SQLite schema 定义与迁移**
  - [x] 1.1 Project / Target / ScopeRule 表
  - [x] 1.2 ScanPlan / ScanTask / ToolInvocation 表
  - [x] 1.3 Asset / Port / Service / WebEndpoint 表
  - [x] 1.4 Finding / Evidence / RawArtifact / FindingRevision 表
  - [x] 1.5 ScopeDecision / AuditLog 表

- [x] **2. Project / Target / Scope API**
  - [x] 2.1 项目 CRUD
  - [x] 2.2 目标输入与导入（TXT/CSV）
  - [x] 2.3 ScopeRule 配置（include/exclude）

- [x] **3. Scope Check 引擎**
  - [x] 3.1 域名匹配（精确 + 子域名包含 + 通配 + 排除优先）
  - [x] 3.2 URL 前缀匹配
  - [x] 3.3 IP/CIDR 匹配
  - [x] 3.4 时间窗口校验
  - [x] 3.5 速率限制可配置性校验
  - [x] 3.6 ScopeDecision 持久化
  - [x] 3.7 TOCTOU 执行前重校验
  - [x] 3.8 单元测试（边界用例）

- [x] **4. Worker subprocess runner**
  - [x] 4.1 goroutine 内 Worker 实现
  - [x] 4.2 独立 workdir 管理（`<data_dir>/workdirs/<project_id>/<task_id>/`）
  - [x] 4.3 超时控制（per-task，策略默认值）
  - [x] 4.4 取消控制（SIGTERM→SIGKILL）
  - [x] 4.5 输出收集（stdout/stderr/exit code）
  - [x] 4.6 输出大小硬上限（100 MB 截断）

- [x] **5. 工具健康检查**
  - [x] 5.1 binary path + version 采集
  - [ ] 5.2 Nuclei template 验证（`nuclei -validate`）
  - [x] 5.3 DNS 解析可用性检查
  - [ ] 5.4 代理可达性检查（若用户配置）
  - [x] 5.5 writable workdir / network availability

- [x] **6. 统一错误模型 + 日志基础设施**
  - [x] 6.1 7 种错误类型定义（ScopeDeniedError / ToolNotFoundError / ToolTimeoutError / ToolExecutionError / ParseError / TruncationWarning / WorkdirError）
  - [ ] 6.2 脱敏过滤器（Authorization / Cookie / API Key）
  - [x] 6.3 分级日志（debug / info / warn / error）

- [x] **7. Tauri 桌面壳**
  - [x] 7.1 Tauri 2.x 项目初始化
  - [x] 7.2 React + TypeScript + Tailwind + shadcn/ui 配置
  - [x] 7.3 Go sidecar / HTTP 通信层（MVP 用 HTTP API）
  - [x] 7.4 基础路由与布局（Project / Target / Plan / Runs / Findings / Reports）

- [x] **8. M0 最小闭环验收**
  - [x] 8.1 创建项目
  - [x] 8.2 添加目标（域名）
  - [x] 8.3 通过 Scope Check
  - [x] 8.4 调一次 Subfinder
  - [x] 8.5 保存 RawArtifact
  - [x] 8.6 Tauri UI 展示结果

### 风险与阻塞项

| 风险 | 级别 | 应对措施 | 状态 |
|------|------|----------|------|
| Tauri sidecar 调用 Go 二进制在 Windows 路径处理有坑 | 中 | M0 优先在 macOS 验证，Windows 后续适配 | 🔴 待验证 |
| 外部工具版本差异导致解析失败 | 中 | 健康检查采集版本号，解析器按版本兼容 | 🟡 已识别 |
| SQLite WAL 模式在 Tauri 资源目录的权限问题 | 低 | 明确 data_dir 路径，初始化时检查可写 | 🟡 已识别 |

### Sprint 验收标准

> 能创建项目 → 添加目标 → 通过 Scope Check → 调一次 Subfinder → 保存 RawArtifact → Tauri UI 展示结果

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
- 开始日期：—
- 结束日期：—
- 验收结果：□

### M3：Nuclei 初筛
- 开始日期：—
- 结束日期：—
- 验收结果：□

### M4：人工验证与报告
- 开始日期：—
- 结束日期：—
- 验收结果：□
