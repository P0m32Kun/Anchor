---
status: archived
source_of_truth: false
owner: kun
last_updated: 2026-05-07
scope: scout-report
archive_reason: Sprint 0 扫描报告，已过时
---

# Sprint 0 代码库全景扫描报告

> 扫描时间: 2026-04-29
> 项目: Anchor — 网络安全扫描平台  
> ⚠️ 此文档已归档，数据基于 Sprint 0 代码库，仅供参考

---

## 1. 路由结构

### 1.1 当前路由清单

| 路径 | 组件 | Navbar 链接 | 备注 |
|------|------|:-----------:|------|
| `/` | `DashboardPage` | ✅ | 主仪表盘，轮询 Worker 状态 |
| `/projects` | `ProjectPage` | ✅ | 项目列表与创建 |
| `/targets` | `TargetPage` | ✅ | 目标管理，使用 `useParams` 获取 projectId |
| `/assets` | `AssetPage` | ✅ | 资产清单，使用 `useParams` 获取 projectId |
| `/runs` | `RunsPage` | ✅ | 扫描运行管理，从 store 读取 currentProject |
| `/findings` | `FindingsPage` | ✅ | 安全发现，使用 `useParams` 获取 projectId |
| `/reports` | `ReportsPage` | ✅ | 报告导出，使用 `useParams` 获取 projectId |
| `/workers` | `WorkersPage` | ✅ | Worker 节点管理 |
| `/settings` | `SettingsPage` | ✅ | 应用配置（API Base、端口范围） |
| `/projects/:id` | `ProjectPage` | ❌ | **Legacy** — 渲染项目列表，非项目详情 |
| `/projects/:id/assets` | `AssetPage` | ❌ | **Legacy** — 同上 |
| `/projects/:id/findings` | `FindingsPage` | ❌ | **Legacy** — 同上 |
| `/projects/:id/reports` | `ReportsPage` | ❌ | **Legacy** — 同上 |

### 1.2 Legacy 路由问题

**核心矛盾**: 组件使用 `useParams<{ id: string }>()` 读取 `:id`，但 App.tsx 注释明确说明"params not fully handled"。

实际情况：
- `FindingsPage` (line 32): `const { id: projectId } = useParams<{ id: string }>()` — **仅在 Legacy 路由下能获取 projectId**
- `AssetPage` (line 35): `const { id } = useParams<{ id: string }>()` — 同上
- `ReportsPage` (line 14): `const { id } = useParams<{ id: string }>()` — 同上
- `TargetPage` (line 203): `const { id } = useParams<{ id: string }>()` — 同上
- `RunsPage` (line 29): 从 `useStore((s) => s.currentProject)` 获取 — **不依赖 useParams**
- `DashboardPage`: 直接调用 API，不依赖 projectId

**结论**: `/targets`, `/assets`, `/findings`, `/reports` 这 4 个一级路由在当前代码下**功能不完整**（`useParams` 返回 `undefined`），只有通过 `/projects/:id/*` 才能正常工作。

### 1.3 改造影响

- **需决定**: 项目上下文切换方式 — 全局 store `currentProject` vs URL params
- `RunsPage` 已使用 store 模式，可作参考
- `FindingsPage`/`AssetPage`/`ReportsPage`/`TargetPage` 需统一改造
- 建议移除 4 条 Legacy 路由，统一使用 `currentProject` store 模式

---

## 2. Zustand Store 现状

### 2.1 State 清单

| State 字段 | 类型 | 说明 |
|-----------|------|------|
| `projects` | `Project[]` | 项目列表 |
| `currentProject` | `Project \| null` | 当前选中项目 |
| `targets` | `Target[]` | 目标列表 |
| `tasks` | `ScanTask[]` | 扫描任务列表 |
| `assets` | `Asset[]` | 资产列表 |
| `webEndpoints` | `WebEndpoint[]` | Web 端点列表 |
| `ports` | `Record<string, Port[]>` | 资产端口映射 (key=assetId) |
| `services` | `Record<string, Service[]>` | 资产服务映射 (key=assetId) |
| `findings` | `Finding[]` | 安全发现列表 |
| `currentFinding` | `{ finding: Finding; evidence: Evidence[] } \| null` | 当前查看的 Finding |

### 2.2 Actions 清单

| Action | 说明 |
|--------|------|
| `setProjects(p)` | 设置项目列表 |
| `setCurrentProject(p)` | 设置当前项目 |
| `setTargets(t)` | 设置目标列表 |
| `addTask(t)` | 追加任务 |
| `updateTask(t)` | 按 id 更新任务 |
| `setAssets(a)` | 设置资产列表 |
| `setWebEndpoints(w)` | 设置 Web 端点 |
| `setPorts(assetId, p)` | 设置某资产的端口 |
| `setServices(assetId, s)` | 设置某资产的服务 |
| `setFindings(f)` | 设置发现列表 |
| `setCurrentFinding(f)` | 设置当前发现详情 |

### 2.3 缺失评估

| 缺失项 | 严重度 | 说明 |
|--------|:------:|------|
| **Loading 状态** | 🔴 高 | 无全局 loading 指示器。各页面自己 `useState(false)` 管理，不共享 |
| **Error 状态** | 🔴 高 | 无统一错误存储。API 错误仅在页面级 catch，无全局通知 |
| **数据清理策略** | 🟡 中 | 切换项目时 `targets`/`assets`/`findings` 等不自动清空，可能显示旧数据 |
| **乐观更新** | 🟢 低 | 无乐观更新机制（可后续按需加） |
| **数据过期/缓存** | 🟡 中 | 无 stale-while-revalidate 或数据新鲜度管理，依赖页面 useEffect 手动刷新 |
| **分页/筛选状态** | 🟡 中 | 无分页状态管理，数据量大时有性能风险 |

### 2.4 各页面数据获取模式

| 页面 | 获取方式 | 说明 |
|------|---------|------|
| `DashboardPage` | 直接 `fetch()` | 绕过 store 和 api.ts，硬编码 `${API_BASE}/workers` |
| `ProjectPage` | `api.listProjects()` → `setProjects()` | ✅ 正确使用 store |
| `TargetPage` | `api.listTargets(projectId)` → 本地 `useState` | 混合模式 |
| `AssetPage` | `api.listAssets(projectId)` → `setAssets()` | 使用 store |
| `RunsPage` | `api.listRuns(projectId)` → 本地 `useState` | 混合模式 |
| `FindingsPage` | `api.listFindings(projectId)` → `setFindings()` | 使用 store |
| `ReportsPage` | `api.listFindings()` → 本地 `useState` | 全部本地状态 |
| `WorkersPage` | 直接 `fetch()` | 绕过 store 和 api.ts |
| `SettingsPage` | 无 API 调用 | 纯配置页 |

---

## 3. SSE（Server-Sent Events）现状

### 3.1 后端 SSE 实现

**文件**: `internal/api/handlers.go` (lines 905-945)

| 项目 | 详情 |
|------|------|
| 端点 | `GET /events` |
| 协议 | 标准 SSE (`text/event-stream`) |
| 客户端管理 | `map[string]chan []byte` (channel-based) |
| 初始事件 | `{"event":"connected"}` |
| 断线处理 | `r.Context().Done()` 自动清理 |

**广播事件清单**:

| 事件 | 触发点 | 数据 |
|------|--------|------|
| `connected` | 客户端连接时 | — |
| `task_update` | 任务完成/失败后 | `task_id` |
| `asset_discovery_complete` | 资产发现工作流结束 | `project_id`, `result` |
| `web_screening_complete` | Web 筛选工作流结束 | `project_id`, `result` |

### 3.2 前端 SSE 接入状态

**结论: ❌ 前端未接入 SSE**

搜索 `frontend/src/` 下 `EventSource`、`useSSE`、`server-sent` 等关键词，零匹配。

前端所有实时数据依赖 `setInterval` 轮询：
- `DashboardPage`: 5 秒轮询 `/workers`
- `WorkersPage`: 无轮询（仅加载时请求一次）

**需要实现**:
1. SSE 连接管理 hook（自动重连、心跳）
2. 事件分发机制 → 更新 Zustand store
3. 替换 `DashboardPage` 的轮询逻辑

---

## 4. 组件成熟度

### 4.1 组件清单

| 组件 | 文件 | 行数 | 导出 | 功能 | 成熟度 |
|------|------|:----:|------|------|:------:|
| `Button` | `Button.tsx` | 75 | `Button` | 变体按钮（primary/secondary/ghost/danger）、loading 状态、forwardRef | ✅ 成熟 |
| `Card` | `Card.tsx` | 68 | `Card`, `CardHeader`, `CardTitle`, `CardDescription` | 玻璃拟态卡片容器、hover 效果 | ✅ 成熟 |
| `Badge` | `Badge.tsx` | 146 | `Badge`, `SeverityBadge`, `StatusBadge` | 语义化徽章、严重度/状态映射 | ✅ 成熟 |
| `Toast` | `Toast.tsx` | 104 | `ToastProvider`, `useToast` | 全局通知（success/warning/error）、自动消失 | ✅ 成熟 |
| `Navbar` | `Navbar.tsx` | 71 | `Navbar` | 顶部导航栏、活动链接高亮 | ✅ 成熟 |
| `EmptyState` | `EmptyState.tsx` | 51 | `EmptyState` | 空状态占位、可配操作按钮 | ✅ 成熟 |
| `Skeleton` | `Skeleton.tsx` | 32 | `Skeleton`, `SkeletonCard`, `SkeletonList` | 加载骨架屏 | ✅ 成熟 |

### 4.2 缺失组件评估

| 组件 | 优先级 | 说明 |
|------|:------:|------|
| **Modal/Dialog** | 🔴 高 | FindingsPage 详情弹窗、ProjectPage 删除确认都用内联 JSX，需统一组件 |
| **Table** | 🟡 中 | 资产列表、Finding 列表等使用 `<table>` 手写，无排序/分页 |
| **DataTable** | 🟡 中 | 可排序、可筛选的数据表格（含分页） |
| **Select/Dropdown** | 🟡 中 | 状态筛选、工具选择等使用原生 `<select>` |
| **Tabs** | 🟢 低 | 暂无 Tab 切换场景，可后续按需 |
| **SSE Hook** | 🔴 高 | `useSSE` 连接管理（对应 §3） |

### 4.3 页面组件概览

| 页面 | 行数 | 复杂度 | 主要功能 |
|------|:----:|:------:|---------|
| `TargetPage` | 428 | 🔴 高 | 目标 CRUD + 文件导入 + Scope 规则 + DryRun |
| `AssetPage` | 337 | 🟡 中 | 资产列表 + 端口/服务/端点详情面板 |
| `FindingsPage` | 289 | 🟡 中 | Finding 列表 + 详情 + 状态变更 + 证据 |
| `RunsPage` | 229 | 🟡 中 | Run 列表 + 创建 + Task 展开 |
| `ProjectPage` | 231 | 🟢 低 | 项目列表 + 创建 + 删除 |
| `ReportsPage` | 235 | 🟡 中 | Finding 摘要 + Markdown 预览 + 导出 |
| `DashboardPage` | 146 | 🟢 低 | 状态概览卡片（静态为主） |
| `WorkersPage` | 129 | 🟢 低 | Worker 列表 |
| `SettingsPage` | 168 | 🟢 低 | API Base 配置 + 端口范围预设 |

---

## 5. 主流程可运行性

### 5.1 后端依赖

| 项目 | 详情 |
|------|------|
| Go 版本 | `go 1.26` |
| 外部依赖 | `github.com/mattn/go-sqlite3`（唯一第三方依赖） |
| 数据库 | SQLite（`~/.anchor/`） |
| 入口 | `main.go` — 支持 server / worker 双模式 |

**风险评估**: ✅ 低风险
- `go-sqlite3` 需要 CGO，确保 `CGO_ENABLED=1`
- 无其他外部依赖，启动不需要外部服务

### 5.2 前端依赖

| 项目 | 详情 |
|------|------|
| Node 版本 | 未指定（建议 ≥ 18） |
| 构建工具 | Vite 5 |
| 框架 | React 18 + TypeScript 5 |
| 样式 | Tailwind CSS 3.4 + PostCSS |
| 桌面 | Tauri v2（可选） |
| 路由 | react-router-dom 6.22 |
| 状态 | zustand 4.5 |

**风险评估**: ✅ 低风险
- 所有依赖版本明确、主流
- 无锁文件冲突风险

### 5.3 启动方式

| 命令 | 用途 | 说明 |
|------|------|------|
| `make dev` | 后端开发 | `go run .` (端口 17421) |
| `make dev-web` | 前端 Web 开发 | `docker compose up -d` + Vite dev server |
| `make dev-desktop` | 桌面应用开发 | `docker compose up -d` + Tauri dev |
| `make up` | 启动 Docker 服务 | docker compose (server + worker) |
| `make run` | 编译运行后端 | `go build -o bin/anchor .` |

**一键启动评估**: ⚠️ 部分可用
- `make dev` 可单独启动后端（需要 CGO）
- `make dev-web` 需要 Docker + `docker compose up -d`（会启动 PostgreSQL/worker 等服务）
- 纯本地开发：`make dev`（后端）+ `cd frontend && npm run dev`（前端）即可
- Tauri 桌面：需要 Rust 工具链

### 5.4 Docker 配置

| 文件 | 用途 |
|------|------|
| `docker-compose.yml` | 主服务（server + worker） |
| `docker-rangefield/docker-compose.yml` | 靶场环境 |
| `Dockerfile.server` | 服务端镜像 |
| `Dockerfile.worker` | Worker 镜像 |

---

## 6. Tailwind 配置现状

### 6.1 设计 Token 体系

**文件**: `frontend/tailwind.config.js`

| 分类 | Token | 详情 |
|------|-------|------|
| **色彩** | `surface` | 4 级深色层级 (`#0B0E14` → `#30363D`) |
| | `brand` | primary/secondary/success/warning/danger/purple |
| | `text` | primary/secondary/tertiary/quaternary |
| | `accent` | Apple 风格鲜艳色（blue/green/red/orange/yellow/purple/teal） |
| | `glass` | 边框/背景/悬停/激活（半透明） |
| **圆角** | `borderRadius` | apple/apple-sm/apple-md/apple-lg/apple-xl (8-20px) |
| **阴影** | `boxShadow` | apple-sm/apple/apple-md/apple-lg + glow 系列（带色彩光晕） |
| **动画** | `animation` | fade-in/slide-up/slide-down/shimmer/pulse-slow |
| **字体** | `fontFamily` | SF Pro 系统字体栈 + SF Mono 等宽字体 |
| **模糊** | `backdropBlur` | glass: 40px |
| **缓动** | `transitionTimingFunction` | apple: cubic-bezier(0.32, 0.72, 0, 1) |

### 6.2 评估

| 方面 | 状态 | 说明 |
|------|:----:|------|
| 色彩体系 | ✅ 完整 | 深色主题 + 4 级 surface + 语义色 + glass 透明体系 |
| 间距体系 | ⚠️ 缺失 | 未定义自定义 spacing scale，使用 Tailwind 默认 |
| 字体大小 | ⚠️ 缺失 | 未定义 type scale，使用 Tailwind 默认 |
| 组件预设 | ❌ 缺失 | 无 `@layer components` 组件级样式预设（CSS 中有 `.liquid-glass` 等） |
| 暗色切换 | ✅ 配置 | `darkMode: "class"`，但当前仅深色模式 |
| 响应式断点 | ⚠️ 默认 | 使用 Tailwind 默认断点，未针对安全工具场景定制 |

**CSS 自定义样式** (`index.css`):
- `.liquid-glass` / `.liquid-glass-hover` / `.nav-glass-dark` 等实用类
- 自定义滚动条样式
- 按钮样式 `.btn-dark-*` 系列
- 全局背景渐变效果

---

## 7. 后端 API 路由全景

### 7.1 API 端点清单

| 方法 | 路径 | Handler | 说明 |
|------|------|---------|------|
| POST | `/projects` | `handleCreateProject` | 创建项目 |
| GET | `/projects` | `handleListProjects` | 项目列表 |
| GET | `/projects/{id}` | `handleGetProject` | 获取项目详情 |
| DELETE | `/projects/{id}` | `handleDeleteProject` | 删除项目 |
| POST | `/projects/{id}/targets` | `handleCreateTarget` | 创建目标 |
| POST | `/projects/{id}/targets/import` | `handleImportTargets` | 批量导入目标 |
| GET | `/projects/{id}/targets` | `handleListTargets` | 目标列表 |
| POST | `/projects/{id}/runs` | `handleCreateRun` | 创建运行 |
| GET | `/projects/{id}/runs` | `handleListRuns` | 运行列表 |
| GET | `/runs/{id}` | `handleGetRun` | 运行详情 |
| GET | `/runs/{id}/tasks` | `handleGetRunTasks` | 运行任务列表 |
| POST | `/projects/{id}/workflows/asset-discovery` | `handleStartAssetDiscovery` | 启动资产发现 |
| GET | `/projects/{id}/assets` | `handleListAssetsFiltered` | 资产列表（带筛选） |
| GET | `/projects/{id}/web-endpoints` | `handleListWebEndpointsByProject` | Web 端点列表 |
| GET | `/assets/{id}/ports` | `handleListPorts` | 端口列表 |
| GET | `/assets/{id}/services` | `handleListServices` | 服务列表 |
| POST | `/projects/{id}/workflows/web-screening` | `handleStartWebScreening` | 启动 Web 筛选 |
| GET | `/projects/{id}/findings` | `handleListFindings` | Finding 列表 |
| GET | `/findings/{id}` | `handleGetFinding` | Finding 详情 |
| PATCH | `/findings/{id}/status` | `handlePatchFindingStatus` | 更新 Finding 状态 |
| POST | `/findings/{id}/evidence` | `handleAddEvidence` | 添加证据 |
| POST | `/findings/{id}/retest` | `handleRetestFinding` | 复测 Finding |
| GET | `/findings/{id}/retests` | `handleListRetests` | 复测历史 |
| PATCH | `/findings/batch-status` | `handleBatchUpdateFindingStatus` | 批量更新状态 |
| GET | `/findings/{id}/curl` | `handleGetFindingCurl` | 获取 cURL 命令 |
| POST | `/scope-rules` | `handleCreateScopeRule` | 创建范围规则 |
| GET | `/scope-rules` | `handleListScopeRules` | 范围规则列表 |
| POST | `/projects/{id}/scope-rules/batch` | `handleBatchCreateScopeRules` | 批量创建规则 |
| POST | `/scan-plans` | `handleCreateScanPlan` | 创建扫描计划 |
| POST | `/scan-plans/{id}/approve` | `handleApprovePlan` | 批准扫描计划 |
| POST | `/scan-plans/dry-run` | `handleDryRun` | 试运行 |
| GET | `/scan-tasks/{id}` | `handleGetTask` | 任务详情 |
| POST | `/scan-tasks/{id}/cancel` | `handleCancelTask` | 取消任务 |
| POST | `/tasks/run` | `handleRunTask` | 运行任务 |
| GET | `/tasks/{id}/artifacts` | `handleListArtifacts` | 任务产物 |
| GET | `/health/tools` | `handleToolHealth` | 工具健康状态 |
| POST | `/health/check` | `handleHealthCheck` | 健康检查 |
| GET | `/events` | `handleSSE` | SSE 事件流 |
| GET | `/projects/{id}/reports/export.md` | `handleExportReportMD` | 导出 Markdown |
| GET | `/projects/{id}/reports/export.json` | `handleExportReportJSON` | 导出 JSON |
| POST | `/projects/{id}/archive` | `handleCreateArchive` | 创建归档 |
| GET | `/projects/{id}/archive/download` | `handleDownloadArchive` | 下载归档 |
| GET | `/tool-templates` | `handleListToolTemplates` | 工具模板列表 |
| GET | `/tool-templates/{id}` | `handleGetToolTemplate` | 工具模板详情 |
| GET | `/workers` | `handleListWorkers` | Worker 列表 |
| POST | `/workers/register` | `handleRegisterWorker` | Worker 注册 |
| POST | `/workers/{id}/heartbeat` | `handleWorkerHeartbeat` | Worker 心跳 |
| GET | `/workers/{id}/tasks/poll` | `handlePollTasks` | Worker 轮询任务 |

---

## 8. 架构总览

```
┌──────────────────────────────────────────────┐
│                   Frontend                    │
│  React 18 + Vite + Tailwind + Zustand        │
│  Tauri v2 (desktop) / Browser (web)          │
│                                              │
│  pages/        components/     lib/          │
│  ├── Dashboard ├── Button      ├── api.ts    │
│  ├── Project   ├── Card        ├── store.ts  │
│  ├── Target    ├── Badge       ├── config.ts │
│  ├── Asset     ├── Toast                    │
│  ├── Runs      ├── Navbar                    │
│  ├── Findings  ├── EmptyState               │
│  ├── Reports   └── Skeleton                  │
│  ├── Workers                                 │
│  └── Settings                                │
└─────────────────┬────────────────────────────┘
                  │ HTTP REST + SSE (未接入)
                  ▼
┌──────────────────────────────────────────────┐
│                Backend (Go)                   │
│  main.go → api.Server → db.Queries (SQLite)  │
│                                              │
│  internal/                                   │
│  ├── api/       → HTTP handlers + SSE        │
│  ├── db/        → SQLite queries             │
│  ├── worker/    → Task runner                │
│  ├── workflow/  → Asset discovery / Web      │
│  ├── scope/     → Scope engine               │
│  ├── health/    → Tool health checker        │
│  ├── models/    → Data models                │
│  ├── nuclei/    → Nuclei integration         │
│  ├── report/    → Report generation          │
│  └── parser/    → Output parsers             │
└─────────────────┬────────────────────────────┘
                  │
                  ▼
┌──────────────────────────────────────────────┐
│              Worker (Go/Docker)               │
│  扫描工具: naabu, httpx, nuclei 等           │
│  通过 HTTP 注册到 Core，轮询任务             │
└──────────────────────────────────────────────┘
```

---

## 9. 关键发现与建议

### 🔴 高优先级

1. **路由与项目上下文不一致** — 5 个页面获取 projectId 方式不同（useParams vs store vs 无），需统一
2. **SSE 前端未接入** — 后端已实现完整 SSE，前端仍用轮询
3. **Store 缺少 Loading/Error 状态** — 各页面重复实现，无统一模式

### 🟡 中优先级

4. **数据清理** — 切换项目时旧数据残留
5. **DashboardPage / WorkersPage 绕过 api.ts** — 直接 fetch，不一致
6. **Modal 组件缺失** — 多页面内联弹窗逻辑

### 🟢 低优先级

7. **Tailwind 间距/字体 Token** — 当前用默认值，可后续定制
8. **分页** — 当前数据量小，暂无性能问题
9. **Tauri 桌面模式** — 配置已有但非核心路径

---

## 10. 文件索引

| 文件路径 | 行数 | 说明 |
|---------|:----:|------|
| `frontend/src/App.tsx` | 65 | 路由定义 |
| `frontend/src/lib/store.ts` | 55 | Zustand store |
| `frontend/src/lib/api.ts` | 313 | API 客户端 + 类型定义 |
| `frontend/src/lib/config.ts` | 24 | API Base 配置 |
| `frontend/src/main.tsx` | ~65 | 入口 + ErrorBoundary |
| `frontend/tailwind.config.js` | ~150 | Tailwind 设计 Token |
| `frontend/src/index.css` | ~250 | 全局样式 + 组件类 |
| `internal/api/handlers.go` | 1326 | 所有 HTTP handler |
| `main.go` | ~130 | 程序入口（server/worker 双模式） |
| `Makefile` | ~85 | 构建/开发/部署命令 |
| `go.mod` | 5 | Go 模块定义 |
| `frontend/package.json` | ~30 | 前端依赖 |
