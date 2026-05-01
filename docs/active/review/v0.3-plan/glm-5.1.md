# v0.3 执行计划 — 技术架构评审报告

> **评审模型**：GLM-5.1 (Technical Architect)
> **评审日期**：2026-04-29
> **状态**：✅ 已完成（2026-05-01）
> **评审对象**：`docs/v0.3-plan.md` v1.1
> **代码库快照**：Go 161 tests passing / `tsc --noEmit` 零错误 / `go vet` 无问题

---

## 整体评分：7.5 / 10

**结论**：这是一份 **高于平均水平** 的迭代计划。文档结构清晰、决策有据、验收标准可量化。最大的优点是"先审计再动手"的 Sprint 0 设计，以及基于代码审计修订的务实态度（v1.0→v1.1 的修订记录证明了这一点）。主要不足在于 **SSE 实现的工作量被低估**、**Zustand store 改造方案缺乏具体设计**、**Tauri 安全配置存在明显缺陷**，以及 **项目切换时的数据一致性问题未充分设计**。

以下按严重程度逐条分析。

---

## 一、阻塞级问题（Blocking）

### B1. Tauri 安全配置过于宽松，生产级应用不可接受

**现状**：`src-tauri/tauri.conf.json` 中：

```json
"security": {
  "csp": null
},
"bundle": {
  "icon": []
}
```

**问题**：
- **CSP 设为 null** 等于完全禁用 Content Security Policy。对于一个安全测试工具，这不仅是功能缺陷，更是品牌信誉问题——用户会质疑"一个做安全的产品自身都不安全"。
- **icon 数组为空**，Tauri 打包时会使用默认图标，桌面端体验极差（显示 Tauri 默认 logo 而非 Anchor 品牌）。
- **无 capabilities 配置**：Tauri v2 默认拒绝所有操作，但当前 `tauri.conf.json` 未引用任何 capability 文件。如果 v0.3 需要文件导出（报告下载），必须配置 `fs` 插件权限。

**建议**：

```json
{
  "security": {
    "csp": "default-src 'self'; img-src 'self' data: blob:; style-src 'self' 'unsafe-inline'; connect-src 'self' http://localhost:17421"
  }
}
```

同时：
1. 创建 `src-tauri/capabilities/default.json`，至少包含 `core:default`
2. 报告导出需要 `tauri-plugin-fs` → 对应 capability 中添加 `"fs:default"` 权限
3. Sprint 0 应追加一项 Tauri 安全基线审计

**风险等级**：不修复则无法发布正式版本。桌面端打包后 CSP 为 null，WebView 可被 XSS 利用（虽然攻击面有限，但作为安全工具不应留此口实）。

---

### B2. SSE 前端完全未实现，工作量被严重低估

**代码审计发现**：

- 后端 SSE 基础设施已就绪：`handleSSE`（handlers.go:907）+ `broadcastSSE`（handlers.go:938），支持连接建立、消息推送、客户端管理。
- **前端零 SSE 代码**：`frontend/src/` 下无 `EventSource`、无 `useSSE` hook、无任何 SSE 客户端逻辑。
- `RunsPage.tsx` 使用 `useEffect` + `api.listRuns()` 做 **一次性拉取**，无任何轮询或实时更新。

**计划中的描述**：
- Sprint 0.7 标注为"验证 SSE 通信路径"，耗时 1h。这不是"验证"，而是 **从零实现 SSE 前端客户端**。
- Sprint 2.3 的 `usePolling` hook（3h）作为 fallback，SSE 优先。但没有给 SSE 前端实现分配任何时间。

**影响**：
- RunsPage 实时进度更新是核心功能，当前完全没有。
- 如果 Sprint 0.7 发现前端确实没接入（已确认），那么需要新增 SSE 前端实现任务，估计 **4-6h**（含 `useSSE` hook、自动重连、心跳检测、降级到 polling 的 fallback）。

**建议**：
1. 将 Sprint 0.7 从"验证"改为"确认状态 + 设计方案"，Sprint 2 新增一项"实现 SSE 前端客户端"（4-6h）。
2. SSE hook 设计应包含：
   - 自动重连（指数退避，与 usePolling 一致）
   - 心跳超时检测（后端当前未实现心跳，需补充）
   - 页面可见性管理（切 tab 时断开/恢复）
   - 消息类型过滤（按 project_id 过滤事件）

```typescript
// 建议 SSE hook 签名
function useSSE<T>(
  url: string,
  options?: {
    eventType?: string;        // 只监听特定 event type
    filter?: (data: T) => boolean;
    retryInterval?: number;    // 重连间隔，默认指数退避
    enabled?: boolean;         // 页面可见性控制
  }
): { data: T | null; error: Error | null; status: 'connecting' | 'connected' | 'disconnected' }
```

---

## 二、高级问题（High）

### H1. Zustand store 改造方案过于笼统

**现状**：`store.ts` 是一个单一大 store（~70 行），包含所有实体（projects, targets, tasks, assets, findings...），但 **完全没有 loading/error 状态管理**。每个页面自行 `useState` 管理 loading。

**计划描述**："分层管理——数据加载状态放 store，表单提交状态放页面"。

**问题**：
1. 未说明 store 结构如何改造。是保持单 store + 增加 loading/error 字段？还是拆分为多个 slice（如 `useProjectStore`、`useRunStore`）？
2. 未说明数据清理策略。切换项目时，`targets`、`assets`、`findings` 等列表数据是否清空？如果不清空，用户会短暂看到旧项目的数据。
3. 未说明缓存策略。同一项目内切换 tab 再切回来，是重新 fetch 还是使用 store 缓存？
4. 未说明是否引入 `zustand/middleware` 的 `persist` 或 `devtools`。

**建议**：Sprint 0 应产出一份 store 设计草案（不是"确认现状"），至少包含：

```typescript
// 建议的 slice 化结构
interface ProjectSlice {
  projects: Project[];
  currentProject: Project | null;
  loading: boolean;
  error: string | null;
  // ...
}

interface RunsSlice {
  runs: Run[];
  loading: boolean;
  error: string | null;
  // SSE 连接状态
  // ...
}

// 使用 Zustand 的 slice pattern
const useStore = create<ProjectSlice & RunsSlice & ...>()(
  devtools(
    persist(
      (...a) => ({
        ...createProjectSlice(...a),
        ...createRunsSlice(...a),
      }),
      { name: 'anchor-store', partialize: (state) => ({ currentProject: state.currentProject }) }
    )
  )
);
```

**关键决策点**（需在 Sprint 0 敲定）：
- 切换项目时是否清空所有子资源 store 数据？（建议：**是**，清空后重新 fetch）
- 是否使用 Zustand `persist` 中间件持久化 `currentProject`？（当前方案说 localStorage 手动管理，与 `config.ts` 的 `STORAGE_KEY` 模式一致，但这与 Zustand persist 功能重复）
- Store 是否按 feature 拆分为独立 hooks？（建议 v0.3 先不拆，保持单 store + slice，避免过度设计）

---

### H2. 后端 CORS 配置存在安全隐患

**现状**：`CORSMiddleware` 设置 `Access-Control-Allow-Origin: *`。

**风险评估**：
- 在 Tauri 桌面端场景下，请求实际上走 WebView → localhost HTTP，CORS 意义有限。
- 但如果用户在浏览器中直接访问（开发模式或独立 Web UI），`*` 意味着任何恶意网站都可以向 `localhost:17421` 发请求。
- 对于安全测试工具，攻击者可通过恶意页面触发扫描、获取 findings 数据、甚至修改 scope 规则。

**建议**：
- 开发模式：限制为 `http://localhost:1420`（Vite dev server）和 `http://localhost:5173`（备用端口）。
- 生产模式（Tauri）：限制为 `tauri://localhost`（Tauri v2 的 origin scheme）。
- 或者：Tauri 模式下完全跳过 CORS（请求通过 IPC 而非 HTTP），只在独立 Web 模式启用 CORS。

```go
func CORSMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        origin := r.Header.Get("Origin")
        allowed := map[string]bool{
            "http://localhost:1420": true,
            "http://localhost:5173": true,
            "tauri://localhost":     true,
        }
        if allowed[origin] {
            w.Header().Set("Access-Control-Allow-Origin", origin)
        }
        // ...
    })
}
```

---

### H3. `fetchJSON` 的错误处理存在静默失败风险

**现状**（`api.ts`）：

```typescript
async function fetchJSON<T>(path: string, opts?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    headers: { "Content-Type": "application/json" },
    ...opts,
  });
  if (!res.ok) {
    const data = await res.json().catch(() => null);
    const message = data?.error?.message || `${res.status}: ${await res.text()}`;
    throw new APIError(message, data?.error?.code);
  }
  return res.json();
}
```

**问题**：
1. 非 `ok` 响应时，先尝试 `res.json()`，失败后 fallback 到 `res.text()`。但 `res.text()` 此时也会消耗 response body，且如果 `res.json()` 已经消耗了 body（`catch` 并非总能阻止），`res.text()` 可能返回空字符串。
2. `Content-Type` header 硬编码为 `application/json`，对 GET 请求无害但对 blob 下载（报告导出）会出错。
3. 无 AbortController、无超时、无重试。
4. `api.listRuns(projectId).then(...).catch(console.error)` — **所有 API 调用的错误都被静默吞掉**（`console.error` 不会显示给用户）。

**建议**（Sprint 2.1 应包含）：
1. 重构 `fetchJSON` 为 `fetchAPI`，区分 JSON 和 blob 模式。
2. 全局错误拦截：网络错误和 5xx 错误应触发全局 Toast，而不是被每个页面分别 `catch(console.error)`。
3. 为所有 API 调用添加 AbortController 支持。

```typescript
// 建议的错误处理增强
async function fetchAPI<T>(path: string, opts?: RequestInit & { timeout?: number }): Promise<T> {
  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), opts?.timeout ?? 30000);

  try {
    const res = await fetch(`${API_BASE}${path}`, {
      ...opts,
      signal: controller.signal,
      headers: {
        ...(opts?.method !== 'GET' ? { "Content-Type": "application/json" } : {}),
        ...opts?.headers,
      },
    });

    if (res.status === 204) return null as T;

    if (!res.ok) {
      const text = await res.text();
      try {
        const data = JSON.parse(text);
        throw new APIError(data?.error?.message ?? `请求失败 (${res.status})`, data?.error?.code);
      } catch (e) {
        if (e instanceof APIError) throw e;
        throw new APIError(`服务器返回非 JSON 响应 (${res.status})`, "NON_JSON_RESPONSE");
      }
    }

    return res.json();
  } catch (err) {
    if (err instanceof DOMException && err.name === 'AbortError') {
      throw new APIError('请求超时，请检查网络连接', 'TIMEOUT');
    }
    if (err instanceof TypeError) {
      throw new APIError('网络连接失败，请检查后端服务是否运行', 'NETWORK_ERROR');
    }
    throw err;
  } finally {
    clearTimeout(timeoutId);
  }
}
```

---

### H4. Sprint 2 工期估算偏乐观

**分析**：Sprint 2 包含 11 个任务，估计 5-7 天：

| 任务 | 估计 | 实际可能 | 原因 |
|------|------|----------|------|
| 2.1 api.ts 增强 | 3h | 4-5h | 需要重构 fetchJSON + 所有调用方适配 |
| 2.2 AbortController | 3h | 2-3h | — |
| 2.3 usePolling + SSE | 3h | **6-8h** | SSE 前端从零实现（见 B2） |
| 2.4 Zustand 改造 | 3h | **5-6h** | 所有页面的 store 调用都要适配 |
| 2.5-2.9 五个页面改造 | 15h | **18-22h** | 每个 3-4h 只够改现有逻辑，加上新的 loading/error/abort 状态，可能翻倍 |
| 2.10 API 集成测试 | 4h | 4h | — |
| 2.11 Workflow 测试 | 2h | 2h | — |
| **合计** | **33h** | **41-48h** | **约 6-8 个工作日** |

**建议**：
- 将 Sprint 2 拆分为 Sprint 2a（API 层 + Store 改造 + SSE）和 Sprint 2b（页面改造）。
- 或者将部分页面改造下放到 Sprint 3（Sprint 3 本身偏体验优化，可以承担更多功能工作）。

---

## 三、中级问题（Medium）

### M1. 项目切换数据一致性未设计

**场景**：用户在 Project A 的 Findings 页面，通过 Navbar Project Switcher 切换到 Project B。

**问题**：
- 计划说"子页面的筛选/排序/分页状态重置"，但没说 store 中的 **数据本身** 是否清空。
- 如果不清空，用户会在 Project B 的页面临时看到 Project A 的 findings（直到新请求返回）。
- 如果清空，会出现短暂的"空数据闪烁"（EmptyState → loading → 新数据）。

**建议**：
1. 切换项目时，立即清空所有子资源 store（targets, assets, findings, runs）。
2. ProjectLayout fetch 新项目信息期间显示全页 Skeleton（计划中已提到，很好）。
3. 子页面检测到 `currentProject` 变化后自动触发新 fetch。
4. 可选优化：在 store 中记录每个项目的 lastFetchTime，如果在 N 秒内切回，使用缓存。

```typescript
// 在 Zustand store 中
setCurrentProject: (project) => set({
  currentProject: project,
  // 清空所有子资源
  targets: [],
  assets: [],
  webEndpoints: [],
  ports: {},
  services: {},
  findings: [],
  currentFinding: null,
  runs: [],
  tasks: [],
})
```

---

### M2. 设计 Token 方案与现状重复

**现状**：`tailwind.config.js` 已有完整的设计体系：
- `surface.*`（4 层暗色表面）
- `brand.*`（6 种品牌色）
- `text-*`（4 层文字色）
- `accent.*`（9 种强调色 + glow 变体）
- `glass.*`（6 种玻璃态色值）
- `rounded-apple*`（5 种圆角）
- `shadow-apple*` + `shadow-glow-*`

**计划描述**："审计现有自定义 class，补充 `tailwind.config.js` 的 `theme.extend.colors`"。

**问题**：这不是"补充"，而是 **已经完成了**。Tailwind 配置中已有完整的颜色体系。Sprint 1.3 分配的 1.5h 可能是浪费时间。

**建议**：
- Sprint 1.3 改为"审计页面中的硬编码颜色值（如 `text-zinc-100`、`bg-yellow-500/15`），统一替换为语义 Token"。
- 在 RunsPage 中发现大量硬编码颜色：`bg-yellow-500/15 text-yellow-300`、`bg-blue-500/15 text-blue-300` 等——这些应该用 `brand.warning`、`brand.primary` 的变体替代。
- 产出一份"硬编码颜色 → 语义 Token"的映射表，而非修改 `tailwind.config.js`。

---

### M3. 报告导出在 Tauri 模式下的实现路径未明确

**问题**：
- 计划提到"blob 下载失败时检查 Content-Type"，但没有说 Tauri 模式下文件如何保存。
- 浏览器模式：`window.open(blobUrl)` 或 `<a download>` 触发下载。
- Tauri 模式：需要 `tauri-plugin-fs` 或 `tauri-plugin-dialog` 的 `save()` API，弹出系统保存对话框。

**当前 `tauri.conf.json`** 没有 `fs` 或 `dialog` 插件配置，也没有 capabilities 文件。

**建议**：
1. Sprint 0.1（API 错误格式审计）应扩展为"API 交互模式审计"，同时确认 blob 端点的错误处理和 Tauri 文件保存方案。
2. 安装 `tauri-plugin-dialog`（提供 `save()` 和 `open()` 对话框）：
   ```bash
   cd src-tauri && cargo add tauri-plugin-dialog
   cd ../frontend && npm add @tauri-apps/plugin-dialog
   ```
3. 在 `capabilities/default.json` 添加 `"dialog:default"`。
4. 前端判断 Tauri 环境时使用 `save()` dialog，浏览器环境用 `<a download>` fallback。

---

### M4. 后端 `broadcastSSE` 是全局广播，缺少项目级过滤

**现状**：

```go
func (s *Server) broadcastSSE(data map[string]interface{}) {
    b, _ := json.Marshal(data)
    for _, ch := range s.sseClients {
        select {
        case ch <- b:
        default:  // 消息丢弃，无日志
        }
    }
}
```

**问题**：
1. 所有 SSE 客户端收到所有事件。如果用户同时打开了多个项目（不同 tab 或不同窗口），Project A 的扫描事件会推给正在看 Project B 的用户。
2. `default` 分支静默丢弃消息——在高频事件（如任务进度）场景下，客户端可能丢失关键状态更新而不知情。
3. 无心跳机制——如果后端长时间不发事件，连接可能被中间层（代理/浏览器）超时关闭。

**建议**：
1. SSE 客户端注册时携带 project_id，`broadcastSSE` 支持按项目过滤。
2. 添加心跳（每 30s 发一次 `{"event":"ping"}`）。
3. 被丢弃的消息应记录日志。

```go
type sseClient struct {
    id        string
    projectID string  // 可为空（全局订阅）
    ch        chan []byte
}

func (s *Server) broadcastProjectSSE(projectID string, data map[string]interface{}) {
    b, _ := json.Marshal(data)
    for _, client := range s.sseClients {
        if client.projectID != "" && client.projectID != projectID {
            continue
        }
        select {
        case client.ch <- b:
        default:
            log.Printf("[SSE] client %s channel full, dropping message", client.id)
        }
    }
}
```

---

### M5. 前端 `useEffect` 依赖项不完整，可能导致数据不同步

**在 `RunsPage.tsx` 中**：

```typescript
useEffect(() => {
  if (!projectId) return;
  api.listRuns(projectId).then((data) => setRuns(data ?? [])).catch(console.error);
  api.listToolTemplates().then((data) => setTemplates(data ?? [])).catch(console.error);
}, [projectId]);  // ← ESLint react-hooks/exhaustive-deps 会警告缺失 api 依赖
```

**问题**：
1. `api` 不在依赖数组中（虽然 `api` 是模块级常量，不会变，但 ESLint 规则会报警）。
2. 未处理竞态：如果 `projectId` 快速切换（A → B），A 的请求可能晚于 B 返回，导致显示错误数据。
3. `.catch(console.error)` 吞掉错误，用户看不到任何提示。

**建议**：
1. 使用 `AbortController` 取消前一次请求。
2. 错误应通过 Toast 或 store 展示给用户。
3. Sprint 2 应统一封装为 `useFetch` hook 或利用 Zustand 的 `async actions`。

---

### M6. Go 后端缺少请求级 context 传播

**现状**：`handleCreateProject`、`handleListProjects` 等 handler 接收 `http.ResponseWriter, *http.Request`，但大部分 handler 没有使用 `r.Context()`。数据库查询使用 `s.queries.XXX()` 无 context 参数。

**问题**：
- 客户端断开连接后，handler 中的数据库操作仍会继续执行（浪费资源）。
- 无法设置请求级超时。
- `handleSSE` 正确使用了 `r.Context().Done()`，但其他 handler 未做到。

**建议**：v0.3 不要求全量改造，但 Sprint 2.10（API 集成测试）应考虑：
1. 为关键 handler（扫描启动、报告导出）添加 context 传播。
2. 在 `db/queries.go` 层面添加 `XXXCtx(ctx context.Context, ...)` 变体（非 v0.3 必须，但应在风险清单中记录）。

---

### M7. 无"后端未运行"的前端检测与引导

**场景**：用户首次打开 Tauri 桌面端，Go 后端服务尚未启动。

**现状**：
- `config.ts` 硬编码 `http://localhost:17421`。
- 所有 API 调用会失败，页面 `catch(console.error)` 静默处理。
- 用户看到空白页面或永远 loading，不知道发生了什么。

**建议**：
1. 添加 `api.healthCheck()` 方法，调用 `/health/tools` 或新的 `/health` endpoint。
2. App 初始化时执行 health check，失败时显示全屏引导："后端服务未启动，请确认 Anchor 服务正在运行" + 重试按钮。
3. 可选：Tauri 模式下自动检测并启动后端进程（sidecar），但这可能超出 v0.3 范围。

---

## 四、低级问题（Low）

### L1. `tsc --noEmit` 已零错误，Sprint 0.4 可能是零产出

**现状**：当前 `tsc --noEmit` 已通过且零错误。Sprint 0.4 "修复现有类型错误"可能不需要做任何事。

**建议**：Sprint 0.4 改为"补 `typecheck` 脚本到 package.json + 验证零错误"，0.5h 足够。省下的时间给 B2（SSE 审计）。

---

### L2. ESLint 暂不引入的决策是正确的

**赞同**：当前阶段 tsc strict 已提供足够的静态检查。ESLint 的维护成本（配置、规则调优、插件兼容）在 v0.3 阶段不划算。

**建议**：在"不在本阶段范围"中明确列出 ESLint，避免后续评审重复讨论。

---

### L3. agent-browser E2E 决策合理，但长期风险需记录

**赞同**：agent-browser 匹配当前阶段需求。但需记录：当项目需要 CI 自动化回归时（如 v0.5+），需要重新评估 Playwright。

---

### L4. 测试覆盖率 >60% 的目标缺少基线

**现状**：Go 已有 161 tests passing，但无覆盖率报告。

**建议**：Sprint 0.8 搭建测试框架时，顺带跑一次 `go test -coverprofile=coverage.out ./...`，记录基线覆盖率。Sprint 4 验收时对比。

---

### L5. 面包屑导航的实现细节缺失

**计划提到面包屑**："Dashboard > Projects > acme-scan > Targets"，但未说明：
1. 面包屑数据从哪来？（路由配置 vs. store vs. 硬编码）
2. 是否作为独立组件？
3. 在移动端/窄窗口下的处理方式。

**建议**：面包屑可作为 Sprint 1.7（统一页面布局）的一部分实现，不需要单独设计。

---

### L6. `darkMode: "class"` 在 tailwind.config.js 中配置但未使用

**现状**：`tailwind.config.js` 有 `darkMode: "class"`，但：
- 所有页面使用硬编码暗色（`bg-zinc-900/50`、`text-zinc-100`）。
- 无 `dark:` 前缀的 class。
- 计划明确排除"深色/浅色主题切换"。

**建议**：v0.3 不需要改动，但应记录：当前实际上是一个 "dark-only" 主题。`darkMode: "class"` 配置是为未来主题切换预留的，v0.3 不依赖它。

---

## 五、遗漏与补充

### O1. 窗口状态持久化

Tauri 窗口关闭后重开，应恢复上次的窗口位置和大小。当前 `tauri.conf.json` 只配置了初始尺寸 1280x800，没有 position 记忆。

**建议**：v0.3 不需要实现，但应记录在技术债清单中。可使用 `tauri-plugin-window-state`（官方插件）在后续版本实现。

---

### O2. 并发扫描任务的资源限制

**场景**：用户在同一项目中启动多个扫描任务。

**问题**：计划未提及前端是否有并发任务数限制。如果用户快速点击"新建扫描"5次，后端会尝试同时启动 5 个工作流，可能导致系统资源耗尽。

**建议**：Sprint 2.7（RunsPage 可靠性）应包含：
1. 前端限制：同时只能有 N 个 running 任务时禁用"新建扫描"按钮。
2. 后端限制：在 handler 中检查当前项目 running 任务数，超限返回 429。
3. 或者更简单：扫描启动按钮在已有一个 running 任务时显示为"队列中"状态。

---

### O3. API 版本化

**现状**：所有路由为 `/projects`、`/runs/{id}` 等，无版本前缀。

**影响**：当前是 v0.x 阶段，API 可能频繁变更。如果用户升级应用后 API 不兼容，会出现难以排查的错误。

**建议**：v0.3 不需要添加 API 版本化（/api/v1/...），但应在"不在本阶段范围"中记录，v1.0 之前需考虑。

---

### O4. 错误边界（React ErrorBoundary）

**计划未提及**：如果页面渲染抛出异常（如 API 返回了非预期格式的数据导致 JS 报错），当前没有 React ErrorBoundary，整个应用会白屏。

**建议**：
1. 在 App 层添加全局 ErrorBoundary，渲染"出了点问题"的 fallback UI + 重新加载按钮。
2. 在 ProjectLayout 添加独立的 ErrorBoundary，隔离项目上下文错误。
3. 估计耗时 1-2h，可加入 Sprint 1。

```typescript
// 最小化 ErrorBoundary
class ErrorBoundary extends React.Component<
  { fallback?: React.ReactNode; children: React.ReactNode },
  { hasError: boolean }
> {
  state = { hasError: false };
  static getDerivedStateFromError() { return { hasError: true }; }
  render() {
    if (this.state.hasError) {
      return this.props.fallback ?? (
        <div className="p-8 text-center">
          <p className="text-red-400">页面渲染出错</p>
          <button onClick={() => this.setState({ hasError: false })}>重试</button>
        </div>
      );
    }
    return this.props.children;
  }
}
```

---

### O5. 无障碍（Accessibility）基本要求

安全测试工具的用户群体中可能有依赖键盘导航或屏幕阅读器的用户。计划仅提及"键盘焦点管理"（2.4），但未覆盖：

1. **ARIA 标签**：状态徽章（Badge）需要 `aria-label`，不只是视觉颜色区分。
2. **焦点管理**：Modal 打开后焦点应 trap 在 Modal 内，关闭后恢复到触发元素。
3. **Live Region**：Toast 应使用 `aria-live="polite"`，错误提示使用 `role="alert"`。
4. **表格可访问性**：Table 组件需要 `<thead>`/`<tbody>`、`scope` 属性。

**建议**：v0.3 不需要完美无障碍，但新增的 5 个基础组件（Input、Select、Table、Modal、ConfirmDialog）应满足 WCAG 2.1 AA 的基本要求。在组件 spec 中追加 a11y checklist。

---

### O6. Go 后端日志规范

**现状**：handler 中混用 `log.Printf` 和 `fmt.Sprintf`，无结构化日志。

**建议**：v0.3 不需要引入 `slog` 或 `zerolog`，但应在 handler 中统一使用 `log.Printf` 并添加 `[handler]` 前缀，便于 grep 和调试。可作为代码审查的附带检查。

---

## 六、独特见解

### U1. "Tauri 嵌入式 Go 后端"架构的隐藏风险

当前架构是 Tauri WebView → HTTP localhost → Go server。这意味着 Go 后端是一个独立进程。**Tauri 不负责管理 Go 进程的生命周期**。

如果 Go 进程崩溃：
1. 前端所有 API 调用失败。
2. SSE 连接断开。
3. 没有任何"自动重启"机制。

**建议**：
- Tauri sidecar 模式（`tauri-plugin-shell`）可以管理子进程生命周期，但需要改造。
- 更简单的方案：前端检测到连续 N 次网络错误时，显示"后端服务异常"的全屏提示 + 重启按钮。
- 这与 M7（后端未运行检测）是同一个问题的不同阶段，可以统一处理。

### U2. Scope 确认页面的安全关键性

Scope 确认是安全测试工具的 **关键安全边界**——确认错误的 scope 可能导致对未授权目标的扫描（法律风险）。计划中将 Scope 作为独立页面的决策是正确的。

但应进一步考虑：
1. Scope 确认是否需要 **二次确认**（如输入项目名称确认）？
2. Scope 变更是否需要 **审计日志**？
3. 管理员是否有强制 override scope 的能力？

v0.3 可以不做这些，但应在架构文档中记录 Scope 确认是 **安全关键路径**，后续版本需加强。

### U3. 建议将 "component audit" 改为 "component registry"

与其在文档中列表"已有组件"和"需新增组件"，建议在代码中创建 `components/index.ts` 的 JSDoc 或单独的 `COMPONENTS.md`，作为活文档。每次新增或修改组件时更新，而非在计划文档中维护静态列表（容易过时）。

---

## 七、Sprint 节奏调整建议

基于以上分析，建议调整如下：

| Sprint | 原计划 | 建议调整 |
|--------|--------|----------|
| Sprint 0 | 1.5-2 天 | **2-2.5 天**：增加 Tauri 安全基线审计（B1）、SSE 状态确认改为方案设计（B2）、Zustand store 设计草案（H1）、Go 测试覆盖率基线（L4） |
| Sprint 1 | 2.5-4 天 | **3-4 天**：增加 ErrorBoundary（O4）、组件 a11y 基线（O5）。Token 审计改为"硬编码颜色替换"（M2），可能省时间 |
| Sprint 2 | 5-7 天 | **7-9 天**：增加 SSE 前端实现（B2，4-6h）、api.ts 重构工作量增加（H3）。或拆分为 2a/2b |
| Sprint 3 | 4-6 天 | **4-5 天**：可从 Sprint 2 接收部分页面改造工作 |
| Sprint 4 | 3-5 天 | **3-4 天**：无需大调整 |
| **总计** | **16-24.5 天** | **19-24.5 天**（中位 22 天） |

---

## 八、总结

### 做得好的地方
1. ✅ Sprint 0 "先审计再动手"的设计思路，避免盲目开发
2. ✅ 基于代码审计修订计划（v1.0→v1.1），文档有生命力
3. ✅ 路由方案 C 的 ProjectLayout 交互规范（5 个边界场景），比大多数设计文档都细致
4. ✅ 测试策略分层合理（Go 单元 → API 集成 → 类型检查 → E2E）
5. ✅ "不在本阶段范围"明确定义，防止 scope creep
6. ✅ 三层测试策略和 agent-browser 的选择是务实决策

### 需要改进的地方
1. ❌ SSE 前端从零实现，被当作"验证"而非"开发"
2. ❌ Tauri 安全配置（CSP、capabilities）需要在 Sprint 0 就修复
3. ❌ Zustand store 改造需要更具体的设计方案
4. ❌ 前端错误处理链（API 错误 → 全局提示 → 用户操作）需要端到端设计
5. ❌ 项目切换时的数据一致性需要明确策略

### 最终建议
本计划在"做什么"层面非常清晰，但在"怎么做"的某些关键路径上仍需补充。建议在 Sprint 0 开工前，优先解决 B1（Tauri 安全）和 B2（SSE 方案），产出 H1（Store 设计草案），然后才开始 Sprint 1 的开发工作。
