# Desktop Bug Triage — Sprint 0.2 主流程阻断点

> 测试时间：2026-04-29  
> 测试环境：macOS + Chrome (agent-browser) + Go 1.26 + Vite dev server  
> 后端启动：`go run .` (port 17421)  
> 前端启动：`cd frontend && npm run dev` (port 1420)

---

## 优先级定义

| 优先级 | 定义 | 示例 |
|--------|------|------|
| **P0** | 主流程中断，功能完全不可用 | 页面白屏、API 全挂、数据损坏 |
| **P1** | 关键页面/操作不可用，用户体验严重受损 | 页面空白、状态错误、操作无响应 |
| **P2** | 交互不顺，有 workaround | 错误提示缺失、空态误导、需手动刷新 |
| **P3** | 视觉/文案细节问题 | 间距、文案、颜色不一致 |

---

## Bug 列表

| # | 优先级 | 页面/模块 | 描述 | 复现步骤 | 状态 |
|---|--------|-----------|------|----------|------|
| 1 | **P0** | 全局 / API 层 | **Vite Proxy 与前端 API 路径完全错位** — `vite.config.ts` 只代理 `/api/*` 到后端，但前端所有 API 调用使用的是 `/projects`、`/workers`、`/runs` 等路径（无 `/api` 前缀），后端路由也未注册 `/api/...` 前缀。导致 Web dev 模式下**所有 API 请求全部 404**，应用完全不可用。 | 1. 清空 localStorage，刷新页面<br>2. 打开 Network 面板<br>3. 观察所有 fetch 请求返回 200 (HTML) 而非 JSON | 🔴 未修复 |
| 2 | **P1** | Dashboard | **Dashboard 从不显示当前项目** — 代码中未从 Zustand store 读取 `currentProject`，无论是否已选项目，始终显示 "前往创建 →"。 | 1. 进入 Projects 页面，创建或点击一个项目<br>2. 跳转到 Dashboard<br>3. 观察 "当前项目" 仍显示 "前往创建 →" | 🔴 未修复 |
| 3 | **P1** | 全局 / Store | **`currentProject` 不持久化** — Zustand store 未使用 persist 中间件，`currentProject` 在页面刷新后丢失，导致每次刷新都需要重新选择项目。 | 1. 在 Projects 页面点击一个项目<br>2. 刷新页面<br>3. 进入 Targets / Runs / Assets 页面，提示 "请先选择一个项目" | 🔴 未修复 |
| 4 | **P1** | Targets | **TargetPage 缺少项目级路由** — `App.tsx` 有 `/projects/:id/assets`、`/projects/:id/findings`、`/projects/:id/reports`，但**没有** `/projects/:id/targets` 或 `/projects/:id/runs`。TargetPage 只能通过 `/targets` 访问，完全依赖 Zustand 中的 `currentProject`。 | 1. 直接访问 `http://localhost:1420/projects/{id}/targets`<br>2. 404 或显示 Dashboard 内容（取决于路由匹配） | 🔴 未修复 |
| 5 | **P1** | Targets | **添加目标时 Scope 确认响应未处理** — 当项目没有配置 Scope 规则时，后端 `POST /projects/{id}/targets` 返回 `{needs_scope_confirmation: true, suggested_rule: {...}}`，但前端 `addTarget` 期望返回 `Target` 对象，直接将其加入 targets 列表（实际无效果，且目标未真正创建）。 | 1. 创建新项目（不添加 Scope 规则）<br>2. 在 Targets 页面输入域名点击 "添加"<br>3. 观察目标列表无变化，无错误提示 | 🔴 未修复 |
| 6 | **P1** | Findings | **Findings 页面无项目时展示误导性空表** — 访问 `/findings`（无项目 ID）时，`useParams` 得到 `undefined`，不发起 API 请求，但页面仍渲染表头和 "暂无 Finding"，用户无法知道需要先选项目。 | 1. 直接访问 `http://localhost:1420/findings`<br>2. 观察页面显示完整表格和 "暂无 Finding" | 🔴 未修复 |
| 7 | **P1** | Reports | **Reports 页面允许无项目 ID 访问** — 路由 `/reports` 无 `:id` 参数，但代码写 `const projectId = id!`，实际为 `undefined`，导致 API 调用 `/projects/undefined/reports/export.md`，返回 404 或空内容，页面显示 "0 个确认漏洞" 等空态。 | 1. 直接访问 `http://localhost:1420/reports`<br>2. 观察页面无错误提示，仅显示空报告 | 🔴 未修复 |
| 8 | **P2** | 全局 | **所有 API 调用静默失败** — 几乎每个页面都用 `.catch(console.error)` 或 `.catch(() => {})` 吞掉异常，用户看不到任何网络错误提示，只能看到永远为空的数据。 | 1. 停止后端服务<br>2. 刷新任意页面<br>3. 观察 Console 有错误，但 UI 只显示 "暂无"、"0 次"、"加载中..." 等 | 🟡 未修复 |
| 9 | **P2** | Dashboard | **后端断开时 Dashboard 不提示连接失败** — Workers 请求失败时 `catch {}` 直接忽略，`onlineWorkers` 保持为 0，显示 "在线 Worker 0" 而非 "连接失败"。 | 1. 启动应用后停止后端<br>2. 观察 Dashboard "在线 Worker" 从 N 变为 0，无任何错误提示 | 🟡 未修复 |
| 10 | **P2** | Runs | **Runs 页面空态信息矛盾** — 无项目时显示 "共 0 次" 和 "请先选择一个项目" 同时出现，前者暗示有数据只是为空，后者说明无法加载。 | 1. 直接访问 `/runs`（不选项目）<br>2. 观察 "共 0 次" 和 "请先选择一个项目" 并列显示 | 🟡 未修复 |
| 11 | **P2** | Projects | **创建项目后不会自动选中** — `handleCreate` 只调用 `setProjects([p, ...projects])`，不调用 `setCurrentProject(p)`，用户创建后需手动点击项目卡片才能继续操作。 | 1. 在 Projects 页面创建新项目<br>2. 创建成功后页面仍留在 Projects，未自动跳转或选中 | 🟡 未修复 |
| 12 | **P2** | Settings | **Server 地址在 Web 模式下为空** — `config.ts` Web 模式返回 `""`（相对路径），但输入框 placeholder 为 `http://localhost:17421`，用户初次进入 Settings 看到空白输入框，无法直观知道当前是否已连接。 | 1. 清空 localStorage 后进入 Settings<br>2. 观察 Server 地址输入框为空 | 🟡 未修复 |
| 13 | **P3** | Backend | **空集合返回 `null` 而非 `[]`** — `ListTargetsByProject`、`ListFindingsByProject` 等查询在没有记录时返回 Go 的 `nil` slice，JSON 序列化为 `null` 而非 `[]`。前端虽用 `?? []` 兜底，但 API 契约不一致。 | 1. `curl /projects/{id}/targets`（无目标的项目）<br>2. 观察返回 `null` | 🟡 未修复 |
| 14 | **P3** | Projects | **datetime-local 输入框在某些浏览器中渲染为多个 spinbutton** — 在测试浏览器中，`<input type="datetime-local">` 被拆分为年/月/日/时/分五个独立 spinbutton，占用大量空间且体验不佳。 | 1. 打开 Projects 页面<br>2. 观察开始/结束时间输入区域 | 🟡 未修复 |

---

## 详细分析

### Bug #1 — P0 API 路径错位（最严重）

**根因**：
- `vite.config.ts` 配置了 `proxy: { "/api": { target: "http://localhost:17421" } }`
- `frontend/src/lib/api.ts` 中所有 API 路径为 `/projects`、`/workers`、`/targets` 等（无前缀）
- `frontend/src/lib/config.ts` Web 模式返回 `""`，导致请求发到 `http://localhost:1420/projects`
- Vite dev server 对 `/projects` 做 SPA fallback（返回 `index.html`），而非代理到后端
- 后端 `internal/api/handlers.go` 注册的路由也无 `/api` 前缀

**结果**：`fetch("/projects")` 拿到 HTML，`res.json()` 抛异常，`.catch(console.error)` 静默吞掉，UI 永远显示空状态。

**修复建议（二选一）**：
1. **前端方案**：`api.ts` 中所有路径加 `/api` 前缀（如 `/api/projects`），同时 Vite proxy 保持 `/api` → `17421`。
2. **后端方案**：`Register()` 中所有路由加 `/api` 前缀，前端保持现状。

**推荐方案 1**（前端改路径），因为后端路由已被其他客户端（Worker、CLI、测试）使用，改后端影响面更大。

---

### Bug #2 — P1 Dashboard 不显示当前项目

**代码位置**：`frontend/src/pages/DashboardPage.tsx`

当前实现：
```tsx
<div className="text-xs text-text-tertiary mb-1">当前项目</div>
<button onClick={() => navigate("/projects")}>
  前往创建 →
</button>
```

完全未使用 `useStore((state) => state.currentProject)`。应改为：
```tsx
const currentProject = useStore((state) => state.currentProject);
// ...
<div>
  {currentProject ? currentProject.name : "前往创建 →"}
</div>
```

---

### Bug #3 — P1 currentProject 不持久化

**代码位置**：`frontend/src/lib/store.ts`

当前使用基础 Zustand，无 persist 中间件。应添加：
```ts
import { persist } from 'zustand/middleware';

export const useStore = create(
  persist<AppState>(
    (set) => ({...}),
    { name: 'anchor-store' }
  )
);
```

或至少对 `currentProject` 单独持久化到 `localStorage`。

---

### Bug #4 — P1 TargetPage 缺少项目级路由

**代码位置**：`frontend/src/App.tsx`

当前路由表：
```tsx
<Route path="/targets" element={<TargetPage />} />
<Route path="/projects/:id/assets" element={<AssetPage />} />
<Route path="/projects/:id/findings" element={<FindingsPage />} />
<Route path="/projects/:id/reports" element={<ReportsPage />} />
```

缺少：
```tsx
<Route path="/projects/:id/targets" element={<TargetPage />} />
<Route path="/projects/:id/runs" element={<RunsPage />} />
```

TargetPage 已使用 `useParams` 读取 `id` 并回退到 `currentProject`，只需在路由表中注册即可。

---

### Bug #5 — P1 Scope 确认未处理

**代码位置**：`frontend/src/pages/TargetPage.tsx` → `addTarget`

后端响应（无 Scope 规则时）：
```json
{
  "message": "当前项目未设置授权范围，是否将此目标自动加入授权范围？",
  "needs_scope_confirmation": true,
  "suggested_rule": { "action": "include", "type": "domain", "value": "example.com" }
}
```

前端代码：
```tsx
const t = await api.createTarget(projectId, { type: targetType, value: targetValue });
setTargets([...targets, t]);
```

`t` 实际为 confirmation 对象，不是 Target。应检查响应并提示用户确认加入 Scope。

---

### Bug #8 — P2 全局静默失败

**影响面**：几乎所有 API 调用

示例：
```tsx
// DashboardPage.tsx
api.listProjects().then((data) => setProjects(data ?? [])).catch(console.error);

// TargetPage.tsx
api.listTargets(projectId).then((data) => setTargets(data ?? [])).catch(console.error);

// AssetPage.tsx
api.listAssets(projectId).then((data) => setAssets(data ?? [])).catch(console.error);
```

**建议**：在 `lib/api.ts` 的 `fetchJSON` 中统一处理网络错误，或添加全局 Toast 通知。至少应在页面级别添加 `error` state 并渲染错误提示。

---

## 测试截图

| 页面 | 截图路径 | 说明 |
|------|----------|------|
| Dashboard | `/tmp/dashboard.png` | 正常加载，但显示 "前往创建 →" |
| Assets (有项目) | `/tmp/assets-page.png` | 通过 `/projects/:id/assets` 访问正常 |
| Findings (有项目) | `/tmp/findings-page.png` | 通过 `/projects/:id/findings` 访问正常 |
| Reports (有项目) | `/tmp/reports-page.png` | 通过 `/projects/:id/reports` 访问正常 |
| Settings | `/tmp/settings-page.png` | 正常加载，Server 地址为空 |

---

## 后端日志摘要

启动日志：
```
2026/04/29 14:01:34 [server] worker id-... marked offline on startup
2026/04/29 14:01:34 anchor server listening on :17421
2026/04/29 14:01:34 data dir: /Users/kun/.anchor
```

未发现启动错误或 panic。

---

## 测试结论

| 检查项 | 结果 | 说明 |
|--------|------|------|
| Dashboard 页面可正常打开 | ⚠️ 部分通过 | 页面打开，但不显示当前项目 |
| Projects 页面可打开，能创建新项目 | ✅ 通过 | 创建功能正常 |
| Targets 页面可打开，能导入目标 | ❌ 失败 | Web 模式下 API 全挂；需手动设置 API_BASE |
| Scope 确认（Dry Run） | ⚠️ 部分通过 | Dry Run API 正常，但前端 Scope 确认交互未实现 |
| Runs 页面可打开，能启动扫描 | ❌ 失败 | 无项目级路由，依赖 Store 状态 |
| 扫描过程中 Runs 页面状态更新 | ⚠️ 未测试 | 无法启动扫描 |
| Assets 页面可打开，显示资产 | ⚠️ 部分通过 | 通过直接 URL `/projects/:id/assets` 可访问 |
| Findings 页面可打开，显示漏洞 | ⚠️ 部分通过 | 通过直接 URL `/projects/:id/findings` 可访问 |
| Reports 页面可打开，能导出报告 | ⚠️ 部分通过 | 通过直接 URL `/projects/:id/reports` 可访问，但导出按钮未测试 |
| Workers 页面可打开，显示 Worker 状态 | ✅ 通过 | 正常显示，后端断开时有错误提示 |
| Settings 页面可打开，配置可保存 | ✅ 通过 | 保存并刷新功能正常 |
| 项目切换功能 | ❌ 失败 | Dashboard 不显示项目，Store 不持久化 |
| 网络断开时页面表现 | ⚠️ 部分通过 | Workers 页面有错误提示，Dashboard 静默显示 0 |
| 空数据时页面表现（EmptyState） | ⚠️ 部分通过 | 空状态存在，但与错误状态混淆 |
| 错误状态表现（API 500 等） | ❌ 失败 | 全局静默失败，无用户可见错误提示 |

**总体评估**：
- **P0 阻断**：1 个（API 路径错位导致 Web 模式完全不可用）
- **P1 严重**：6 个（项目上下文丢失、路由缺失、Scope 确认未处理、空态误导）
- **P2 一般**：5 个（静默失败、空态矛盾、创建不自动选中、配置显示不清）
- **P3 细节**：2 个（API 返回 null、datetime 渲染异常）

**主流程状态**：🔴 **阻断** — 在修复 P0 API 路径问题之前，Web dev 模式无法完成任何涉及 API 的操作。
