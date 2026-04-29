# v0.3 Desktop Usability & Reliability — 代码审查报告

**审查日期**: 2026-04-29  
**审查范围**: 前端核心文件（API 层、状态管理、路由、SSE/Hooks、关键页面）+ 后端核心文件（Handlers、测试、Schema）  
**审查方法**: 静态代码分析 + E2E 测试覆盖检查 + 安全边界审查  

---

## 一、总体评分

| 维度 | 评分 | 说明 |
|------|------|------|
| 正确性 | 3/5 | 核心流程正确，但存在若干边界 case 和 cleanup 隐患 |
| 可读性 | 4/5 | 整体清晰，命名规范，但有重复代码和简略 E2E 测试 |
| 架构 | 3/5 | 分层合理，但存在绕过统一层、状态管理冗余、循环依赖风险 |
| 安全 | 3/5 | 输入校验覆盖不全，CORS 配置偏宽松，SSE 无客户端限流 |
| 性能 | 3/5 | useMemo 使用合理，但 Dashboard 存在请求风暴风险，SSE 重连逻辑有抖动 |

**综合评分: 3.2/5** — 代码健康度中等，有若干 **Important** 级别问题需要修复后方可进入生产环境。

---

## 二、正确性审查（3/5）

### ✅ 正确的设计
1. **Store 项目切换清空** — `setCurrentProjectId` 正确地清空了 `targets`、`assets`、`findings`、`runs` 等所有子资源，避免跨项目数据残留（`store.ts:66-82`）。
2. **AbortController 清理** — `FindingsPage` 和 `RunsPage` 的 `useEffect` 都正确地创建了 `AbortController` 并在 cleanup 中 `abort`，防止竞态请求（`FindingsPage.tsx:52`）。
3. **后端 SSE 项目隔离** — `handleProjectSSE` 按 `projectID` 分组管理客户端，`broadcastProjectSSE` 只向对应项目的客户端推送，不存在跨项目泄露（`handlers.go:933-988`）。
4. **SSE 断线重连** — `useSSE` 实现了指数退避（`retryDelay * 2`）、最大重试次数（默认 5 次）、心跳检测（默认 30s）、页面可见性管理，逻辑完整。

### ⚠️ 发现的问题

#### C-1: DashboardPage 轮询信号污染 — **Important**
**位置**: `frontend/src/pages/DashboardPage.tsx:79-87`
```tsx
const interval = setInterval(() => {
  if (!ctrl.signal.aborted) {
    fetchDashboardData(ctrl.signal);
  }
}, 10000);
```
**问题**: 所有轮询请求共用同一个 `AbortController.signal`。一旦组件卸载，`ctrl.abort()` 会同时取消所有进行中的请求，这本身是正确的。但更隐蔽的问题是：**如果某个请求因为信号 abort 而失败，interval 仍然继续运行，下一次 tick 发现 `signal.aborted === true` 就跳过 fetch，导致轮询永久停止**（虽然组件卸载时也会 clearInterval，但如果 abort 是由其他地方触发的，轮询就会死）。

**建议修复**:
```tsx
const interval = setInterval(() => {
  fetchDashboardData(new AbortController().signal);
}, 10000);
// cleanup 时统一 abort 当前正在进行的请求即可
```

#### C-2: useSSE scheduleReconnect ↔ connect 循环依赖 — **Important**
**位置**: `frontend/src/hooks/useSSE.ts:75-175`
```tsx
const scheduleReconnect = useCallback(() => { ... connect(); ... }, []);
const connect = useCallback(() => { ... scheduleReconnect(); ... }, [scheduleReconnect]);
```
**问题**: `scheduleReconnect` 依赖 `connect`，`connect` 又依赖 `scheduleReconnect`，形成循环依赖。虽然 `useCallback` 在首次渲染时能够解析（因为都是同一个 render cycle 内定义），但如果 deps 数组在未来被修改，可能导致无限重连循环或 stale closure。

**建议修复**: 将核心逻辑提取为不依赖 useCallback 的 plain functions，或者使用 ref 来存储回调引用，彻底打破循环依赖。

#### C-3: ProjectLayout 异步导航无 cleanup — **Medium**
**位置**: `frontend/src/components/ProjectLayout.tsx:42`
```tsx
catch((err) => {
  ...
  setTimeout(() => navigate("/projects"), 2000);
});
```
**问题**: `setTimeout` 在组件卸载后仍然执行 `navigate`，React Router v6 在卸载组件上调用 navigate 会抛出警告或导致意外行为。

**建议修复**:
```tsx
const timerRef = useRef<number>();
useEffect(() => {
  return () => clearTimeout(timerRef.current);
}, []);
// 在 catch 中: timerRef.current = setTimeout(...);
```

#### C-4: FindingsPage openDetail 无错误处理 — **Medium**
**位置**: `frontend/src/pages/FindingsPage.tsx:55-58`
```tsx
const openDetail = async (findingId: string) => {
  const data = await api.getFinding(findingId);
  setCurrentFinding(data);
  setDetailOpen(true);
};
```
**问题**: `api.getFinding` 失败时（如网络断开、finding 被删除），未捕获的 Promise rejection 会导致全局错误处理触发 Toast，但 `detailOpen` 状态不会更新，用户看到无反馈或错误 Toast。

**建议修复**:
```tsx
const openDetail = async (findingId: string) => {
  try {
    const data = await api.getFinding(findingId);
    setCurrentFinding(data);
    setDetailOpen(true);
  } catch (err) {
    toast("加载详情失败", "error");
  }
};
```

#### C-5: 后端 BatchUpdate 非原子操作 — **Important**
**位置**: `internal/api/retest_handlers.go:56-76`
```go
for _, id := range req.IDs {
  if err := s.queries.UpdateFindingStatus(id, ...); err != nil {
    writeError(w, http.StatusInternalServerError, ...)
    return
  }
}
```
**问题**: 循环中遇到第一个错误即返回，前面成功的更新不会回滚，造成**部分更新**。客户端收到 500，但部分 finding 状态已变更，数据不一致。

**建议修复**: 使用数据库事务包裹批量更新，或者至少在校验阶段先验证所有 ID 和 status 的合法性，再执行更新。

#### C-6: usePolling 在 enabled 切换时不会立即触发 — **Low**
**位置**: `frontend/src/hooks/usePolling.ts:65-76`
`executePoll` 的依赖为空数组，当 `enabled` 从 `false` 变为 `true` 时，`useEffect` 会执行 initial poll，这是正确的。但如果 `pollFn` 变化（如闭包捕获了新的参数），interval 中执行的仍然是旧的 `pollFnRef.current` 直到 interval 重置。目前 `interval` 变化才会重置 interval，但 `pollFn` 不在 deps 中，这是设计上的意图（使用 ref），不算 bug。

---

## 三、可读性审查（4/5）

### ✅ 优秀实践
1. **App.tsx 路由注册表** — 顶部注释表格完整列出了所有路由路径、组件、用途，维护得非常好（`App.tsx:1-30`）。
2. **命名规范** — `useSSE`、`usePolling`、`useRealtimeData` 命名清晰；`handleXxx` / `onXxx` 区分明确；状态变量 `xxxLoading` / `xxxError` 一致。
3. **Zustand store 结构** — slice 化虽然没有使用 slice pattern，但按资源分组（projects/targets/assets/findings/runs），易于理解。

### ⚠️ 发现的问题

#### R-1: api.ts fetchAPI / fetchBlob 代码重复 — **Medium**
**位置**: `frontend/src/lib/api.ts:42-109` 和 `111-148`
**问题**: 两个函数几乎完全一致（timeout、AbortController、signal 合并、错误分类、globalErrorHandler）。约 40 行重复逻辑。

**建议修复**: 提取共享的 `request` 基函数：
```tsx
async function request<T>(path: string, opts?: RequestInit & { timeout?: number }, parser: (res: Response) => Promise<T>): Promise<T> { ... }
```

#### R-2: importTargets 绕过统一 API 层 — **Medium**
**位置**: `frontend/src/lib/api.ts:254-274`
**问题**: `importTargets` 直接调用原生 `fetch`，自己处理错误逻辑，没有复用 `fetchAPI` 的统一错误分类和 global error handler。如果后端返回非 JSON 错误，行为与 `fetchAPI` 不一致。

**建议修复**: 使用 `fetchAPI` 并支持 `FormData` body（当前 `fetchAPI` 硬编码了 `"Content-Type": "application/json"` 的 header，需要调整以支持覆盖）。

#### R-3: LegacyRouteGuard console.warn 每次渲染触发 — **Low**
**位置**: `frontend/src/App.tsx:32-34`
```tsx
useEffect(() => {
  console.warn(`[Deprecation] Accessed legacy route: ${location.pathname}`);
}, [location]);
```
**问题**: 虽然使用了 `useEffect`，但 `location` 作为依赖，在每次路由参数变化时都会触发。这实际上不是性能问题，但如果用户频繁访问 legacy 路由，console 会被刷满。

**建议**: 如预期 Sprint 1.11 移除，可加 TODO 和 issue 链接。

#### R-4: FindingsPage.e2e.md 过于简略 — **Medium**
**位置**: `frontend/e2e/tests/FindingsPage.e2e.md`
**问题**: 相比 `RunsPage.e2e.md`（226 行）和 `DashboardPage.e2e.md`（详细多 TC），`FindingsPage.e2e.md` 只有 30 行，没有覆盖：筛选交互、批量操作、详情弹窗、状态变更、复测按钮。

**建议**: 参照 RunsPage 的详细程度，补充筛选、批量操作、详情弹窗的 E2E 步骤。

---

## 四、架构审查（3/5）

### ✅ 好的设计
1. **SSE → Polling 降级策略** — `useRealtimeData` 将 SSE 和 Polling 封装为统一接口，`isLive` / `isPolling` / `isLoading` 状态清晰，上层组件无需关心底层传输方式。
2. **ErrorBoundary 包裹 Routes** — 页面级错误被捕获，不会导致整个应用白屏（`App.tsx:131`）。
3. **后端 Handler 分层** — 每个资源有独立的 handler，错误统一通过 `writeError` / `writeJSON` 输出。

### ⚠️ 发现的问题

#### A-1: FindingsPage 状态更新后全量刷新 — **Important**
**位置**: `frontend/src/pages/FindingsPage.tsx:61-79`
```tsx
const changeStatus = async (findingId: string, status: string) => {
  await api.updateFindingStatus(findingId, status);
  ...
  if (projectId) {
    const updated = await api.listFindings(projectId, undefined);
    setFindings(updated ?? []);
  }
};
```
**问题**: 单条 finding 状态更新后，重新拉取**整个项目的所有 findings**。在 findings 数量大时（如 1000+），这是明显的性能瓶颈。

**建议修复**:
- 乐观更新：先本地修改 finding 状态，再发请求，失败时回滚
- 或至少只更新单条：`setFindings(prev => prev.map(f => f.id === findingId ? { ...f, status } : f))`

#### A-2: DashboardPage 跨项目聚合请求风暴 — **Important**
**位置**: `frontend/src/pages/DashboardPage.tsx:35-55`
```tsx
const runsPromises = validProjects.map((p) => api.listRuns(p.id, signal).catch(() => []));
const findingsPromises = validProjects.map((p) => api.listFindings(p.id, "pending_review", signal).catch(() => []));
```
**问题**: 每 10 秒，Dashboard 会并发发起 `2 * N` 个请求（N = 项目数）。如果用户有 20 个项目，每 10 秒就是 40 个 API 请求。没有服务端聚合接口，这是 N+1 查询的前端版本。

**建议修复**: 后端提供 `/dashboard` 或 `/stats` 聚合接口，一次请求返回所有项目的统计概览。

#### A-3: Store 存在未使用的状态字段 — **Low**
**位置**: `frontend/src/lib/store.ts:23-26`
```tsx
workersLoading: boolean;
workersError: string | null;
reportsLoading: boolean;
reportsError: string | null;
```
**问题**: 这些字段在 store 中定义了 setter，但审查范围内没有任何页面使用它们。这是死代码，增加维护负担。

**建议**: 移除或注释说明未来用途。

#### A-4: App.tsx 全局错误处理器清理函数不一致 — **Low**
**位置**: `frontend/src/App.tsx:88-91`
```tsx
return () => {
  setGlobalErrorHandler(() => {});
};
```
**问题**: 使用了匿名空函数而不是 `clearGlobalErrorHandler()`。虽然效果相同，但增加了与清理函数的不一致性。

**建议**: 改为 `clearGlobalErrorHandler()`。

---

## 五、安全审查（3/5）

### ✅ 正面的安全实践
1. **后端输入校验** — `handlePatchFindingStatus` 和 `handleAddEvidence` 都有 `validStatuses` / `validTypes` 白名单校验（`handlers.go:1264-1274`）。
2. **命令参数脱敏** — `redactArgs` 函数在日志中隐藏包含 `key`/`token`/`secret`/`password` 的参数（`handlers.go:1082-1094`）。
3. **CORS 白名单** — 只允许已知的前端开发域名，不允许通配符 `*`。

### ⚠️ 发现的问题

#### S-1: CORS 配置不完整 — **Important**
**位置**: `internal/api/handlers.go:215-230`
```go
w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
```
**问题**: 
- 缺少 `Authorization` header（如果未来添加 JWT/API Key 认证会失败）
- 缺少 `Access-Control-Allow-Credentials: true`（如果前端需要携带 cookie）
- 缺少 `Access-Control-Max-Age`（每次预检请求都走完整处理链，浪费资源）
- `DELETE` / `PATCH` 方法已注册，但 CORS 允许方法中没有 `DELETE`

**建议修复**:
```go
w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
w.Header().Set("Access-Control-Max-Age", "86400")
```

#### S-2: handleCreateProject 缺少必填校验 — **Important**
**位置**: `internal/api/handlers.go:231-280`
**问题**: `req.Name` 没有校验是否为空字符串，可以创建 name 为 `""` 的项目。同样 `Organization`、`Purpose` 等字段如果传入超长字符串（如 1MB），可能导致数据库性能问题或日志膨胀。

**建议修复**:
```go
if strings.TrimSpace(req.Name) == "" {
  writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "name is required"))
  return
}
if len(req.Name) > 200 {
  writeError(...)
}
```

#### S-3: SSE 客户端无数量限制 — **Medium**
**位置**: `internal/api/handlers.go:933-988`
**问题**: `sseClients[projectID][clientID] = ch` 没有限制单个 project 的客户端数量。恶意用户或脚本可以为同一个项目打开无限 SSE 连接，每个连接占用一个 goroutine + channel buffer，可能导致内存泄漏和 goroutine 爆炸（DoS）。

**建议修复**:
```go
s.mu.Lock()
if len(s.sseClients[projectID]) >= 100 {
  s.mu.Unlock()
  writeError(w, http.StatusServiceUnavailable, errors.New(errors.ErrRateLimit, "too many SSE connections"))
  return
}
```

#### S-4: handleBatchUpdateFindingStatus 缺少校验 — **Important**
**位置**: `internal/api/retest_handlers.go:56-76`
**问题**: 
- 没有校验 `req.IDs` 是否为空
- 没有校验 `req.Status` 是否合法（可以传入任意字符串注入数据库）
- 没有校验 `req.IDs` 的长度上限（传入 10 万个 ID 会导致长时间占用连接）

**建议修复**:
```go
if len(req.IDs) == 0 { ... }
if len(req.IDs) > 1000 { ... }
if !validStatuses[req.Status] { ... }
```

#### S-5: Markdown 报告 XSS 风险 — **Medium**
**位置**: `internal/api/handlers.go:1354-1365`
**问题**: `report.GenerateMarkdown(data)` 如果包含用户可控输入（如 finding title、evidence excerpt），且前端直接渲染 Markdown 为 HTML，存在 XSS 风险。当前审查范围没有看到前端 Markdown 渲染组件，但如果未来在 ReportsPage 中渲染，需要考虑。

**建议**: 在生成 Markdown 时对用户输入进行 HTML 转义，或前端使用安全的 Markdown 渲染器（如 `react-markdown` + `rehype-sanitize`）。

---

## 六、性能审查（3/5）

### ✅ 正面的性能实践
1. **useMemo 使用合理** — `FindingsPage` 的 `filteredFindings`、`DashboardPage` 的 `stats`/`recentRuns`/`recentFindings` 都使用了 `useMemo`，避免不必要的重计算。
2. **Debounced 搜索** — `FindingsPage` 的 keyword 输入有 300ms debounce，减少过滤频率。
3. **SSE 页面可见性管理** — `useSSE` 在页面 hidden 时主动关闭连接，节省资源。

### ⚠️ 发现的问题

#### P-1: DashboardPage 请求风暴 — **Critical（架构已提及，性能角度重申）**
每 10 秒 `2 * N` 个并发请求，无聚合接口。项目数 > 10 时即可感知延迟。

#### P-2: FindingsPage 筛选全量内存过滤 — **Medium**
**位置**: `frontend/src/pages/FindingsPage.tsx:105-122`
```tsx
const filteredFindings = useMemo(() => { ... }, [findings, statusFilter, severityFilter, debouncedKeyword]);
```
**问题**: 虽然用了 `useMemo`，但 `findings` 是全量加载的。如果项目有 5000+ findings，全量加载 + 客户端过滤会消耗大量内存和 CPU。

**建议**: 后端支持分页和服务器端筛选，前端改为分页展示。

#### P-3: useRealtimeData 的 polling fallback 抖动 — **Medium**
**位置**: `frontend/src/hooks/useRealtimeData.ts:66-70`
```tsx
const pollingEnabled = sseFailed && status === "error";
```
**问题**: 当 SSE 短暂断开（如网络抖动），`status` 变为 `error`，`sseFailed` 设为 `true`，polling 立即启动。如果 SSE 在 2 秒后重连成功，`status` 变为 `open`，polling 立即停止。这种快速切换会导致：
- 请求数量突增（SSE + Polling 同时存在短暂重叠）
- UI 状态快速跳变（"轮询中" ↔ "实时连接"）

**建议**: 增加 polling 启动的延迟缓冲，如 SSE 失败 3 秒后才启动 polling：
```tsx
const [pollingEnabled, setPollingEnabled] = useState(false);
useEffect(() => {
  if (sseFailed && status === "error") {
    const timer = setTimeout(() => setPollingEnabled(true), 3000);
    return () => clearTimeout(timer);
  }
  setPollingEnabled(false);
}, [sseFailed, status]);
```

#### P-4: usePolling interval 变化时无平滑过渡 — **Low**
**位置**: `frontend/src/hooks/usePolling.ts:65-76`
**问题**: 当 `enabled` 或 `interval` 变化时，旧的 interval 被 `clearInterval`，新的 interval 立即创建，但上一个 tick 可能刚好被清除，导致两次 poll 的间隔短于预期。

**建议**: 使用 `setTimeout` 递归代替 `setInterval`，每次执行完再调度下一次，避免 interval 叠加。

---

## 七、测试覆盖审查

### 后端测试
| 文件 | 覆盖情况 | 评价 |
|------|---------|------|
| `handlers_test.go` | CreateProject, ListProjects, GetProject, DeleteProject, ImportTargets | **不足**: 缺少 SSE、Findings CRUD、BatchUpdate、CORS、HealthCheck、错误路径测试 |
| `workflow_test.go` | extractValues, joinArgs, parseInt, writeHostsFile, buildNaabuArgs, computeDedupKey, truncateString, writeTargetsFile | **良好**: 表驱动测试，命名清晰，覆盖辅助函数 |
| `db.go` | 无独立测试 | **需补充**: Schema 迁移测试（升级/降级路径）|

### 前端 E2E 测试
| 页面 | 测试文件 | 评价 |
|------|---------|------|
| App.tsx | `App.e2e.md` | 覆盖导航和健康检查 |
| DashboardPage | `DashboardPage.e2e.md` | **良好**: 6 个 TC，覆盖空状态、统计、最近活动 |
| RunsPage | `RunsPage.e2e.md` | **良好**: 覆盖 SSE、轮询、取消确认、空状态引导 |
| FindingsPage | `FindingsPage.e2e.md` | **严重不足**: 仅 30 行，无筛选、批量操作、详情弹窗覆盖 |
| ProjectLayout | `ProjectLayout.e2e.md` | **良好**: 覆盖嵌套路由和错误跳转 |

### 测试缺失清单
- [ ] **SSE 集成测试**（前后端都没有）
- [ ] **CORS Preflight 测试**
- [ ] **Findings 批量操作 E2E**
- [ ] **Dashboard 请求性能/压力测试**
- [ ] **Race condition 测试** (`go test -race`)

---

## 八、修复建议汇总（按优先级）

### 🔴 Critical / 阻塞合并
| # | 问题 | 文件 | 修复建议 |
|---|------|------|---------|
| 1 | DashboardPage 轮询信号污染，可能导致轮询永久停止 | `DashboardPage.tsx` | 每次轮询使用新的 `AbortController` |
| 2 | BatchUpdate 非原子操作，部分更新导致数据不一致 | `retest_handlers.go` | 使用数据库事务包裹批量更新 |
| 3 | SSE 无客户端数量限制，存在 DoS 风险 | `handlers.go` | 单项目限制 100 个并发连接 |

### 🟠 Important / 强烈建议
| # | 问题 | 文件 | 修复建议 |
|---|------|------|---------|
| 4 | FindingsPage 状态更新后全量刷新 | `FindingsPage.tsx` | 乐观更新或本地 patch |
| 5 | DashboardPage 跨项目请求风暴 | `DashboardPage.tsx` + 后端 | 后端提供聚合统计接口 |
| 6 | handleCreateProject 缺少输入校验 | `handlers.go` | 增加 name 非空和长度限制 |
| 7 | BatchUpdate 缺少 status/ids 校验 | `retest_handlers.go` | 增加白名单和长度上限校验 |
| 8 | CORS 配置不完整 | `handlers.go` | 补充 Headers、Methods、Max-Age |
| 9 | useSSE 循环依赖风险 | `useSSE.ts` | 使用 ref 打破循环或提取 plain functions |
| 10 | FindingsPage.e2e.md 过于简略 | `e2e/tests/FindingsPage.e2e.md` | 补充筛选、批量操作、详情弹窗测试 |

### 🟡 Medium / 建议优化
| # | 问题 | 文件 | 修复建议 |
|---|------|------|---------|
| 11 | ProjectLayout setTimeout 无 cleanup | `ProjectLayout.tsx` | 使用 ref + cleanup |
| 12 | FindingsPage openDetail 无 try/catch | `FindingsPage.tsx` | 添加错误处理 |
| 13 | api.ts fetchAPI/fetchBlob 重复代码 | `api.ts` | 提取 request 基函数 |
| 14 | importTargets 绕过统一 API 层 | `api.ts` | 复用 fetchAPI（支持 FormData）|
| 15 | useRealtimeData polling 抖动 | `useRealtimeData.ts` | 增加 3s 启动缓冲 |
| 16 | Store 未使用的 workers/reports 状态 | `store.ts` | 移除死代码 |

---

## 九、结论

v0.3 的 Desktop Usability & Reliability 实现**核心功能正确**，SSE + Polling 降级、项目隔离、AbortController 清理、Store 切换清空等关键机制都到位。但在以下方面需要加强：

1. **边界条件处理** — Dashboard 轮询信号、BatchUpdate 原子性、SSE 客户端限流
2. **输入校验完整性** — 后端多个 handler 缺少参数校验
3. **性能优化** — Dashboard 聚合接口、Findings 乐观更新
4. **测试覆盖** — FindingsPage E2E、SSE 集成测试、Race 检测

**建议在修复所有 🔴 Critical 和 🟠 Important 级别问题后，方可合并到主分支。**

---

*报告生成完毕。如有疑问或需要针对某一问题的详细修复代码，请随时告知。*
