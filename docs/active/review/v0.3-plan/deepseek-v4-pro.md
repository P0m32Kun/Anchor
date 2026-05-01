# v0.3 Desktop Usability & Reliability — 技术架构评审

> **评审人**：DeepSeek V4 Pro（技术架构师视角）
>
> **评审日期**：2026-04-29
> **状态**：✅ 已完成（2026-05-01）
>
> **评审对象**：`docs/v0.3-plan.md` v1.1
>
> **评审方法**：基于文档 + 实际代码库全面审计（含路由、store、API 客户端、Go 后端、Tauri 配置、测试覆盖率、E2E 测试）

---

## 一、总体评价

| 维度 | 评分 | 说明 |
|------|------|------|
| 方案完整性 | 8/10 | 覆盖面广，Sprint 划分合理，边界场景定义详细 |
| 技术可行性 | 7/10 | 整体可行，但存在几个基础设施缺口 |
| 架构合理性 | 8/10 | 混合嵌套路由决策正确，交互规范细致 |
| 可维护性 | 7/10 | 测试策略清晰，但覆盖率基线太低 |
| 安全性 | 6/10 | Tauri 安全配置有严重遗漏 |
| 性能与扩展性 | 7/10 | Polling/SSE 策略合理，但 Context 方案有隐患 |
| **综合评分** | **7.2/10** | 良好方案，需修复 2 个阻塞项 + 5 个高优先级项后方可启动 |

### 简要结论

v0.3 方案整体质量良好，文档经过了细致的修订（v1.0 → v1.1），尤其是在组件现状评估、API 审计和 ProjectLayout 交互规范方面体现了扎实的代码审计工作。**核心风险不在于方案本身的设计，而在于几个基础设施缺口和一项被低估的工程成本**——Go 测试覆盖率仅 18.5%，workflow 模块（扫描流水线核心）覆盖率为 0%，而 Sprint 2.11 只分配了 2 小时做 workflow 测试。建议在 Sprint 0 启动前先修复阻塞项，并将部分高优项纳入 Sprint 0 范围。

---

## 二、逐条问题清单

### 🔴 阻塞级（BLOCKING）— 必须在 Sprint 0 启动前解决

#### B1. Vite 开发服务器未配置 API 代理

**现状**：
- Vite dev server 运行在 `http://localhost:1420`
- Go API server 运行在 `http://localhost:17421`
- `vite.config.ts` 中**无 proxy 配置**
- 前端 `config.ts` 中 `getApiBase()` 在两个分支（Tauri/Web）都返回 `http://localhost:17421`

**影响**：浏览器开发模式下，前端跨域请求 Go API（不同端口）。虽然后端有 `CORSMiddleware`，但：
1. Tauri WebView 中 CORS 行为与浏览器不同，依赖 CORS 不够可靠
2. 生产构建（`npm run build`）后，Tauri 加载的是静态文件，API 请求仍需到达 `localhost:17421`，这在桌面打包后依赖用户手动启动 Go 后端
3. 方案完全没有提及前后端联调的开发环境配置

**建议方案**：

```typescript
// vite.config.ts — 增加 proxy 配置
export default defineConfig(async () => ({
  plugins: [react()],
  clearScreen: false,
  server: {
    port: 1420,
    strictPort: true,
    watch: { ignored: ["**/src-tauri/**"] },
    // 新增：代理 API 请求到 Go 后端
    proxy: {
      "/api": {
        target: "http://localhost:17421",
        changeOrigin: true,
      },
    },
  },
}));
```

同时更新 `config.ts`：

```typescript
export function getApiBase(): string {
  const stored = localStorage.getItem(STORAGE_KEY);
  if (stored) return stored;

  const isTauri = !!(window as any).__TAURI__;
  if (isTauri) {
    return "http://localhost:17421";
  }

  // Web dev mode: use relative path (proxied by Vite)
  // Web production: same-origin or configured via localStorage
  return "";
}
```

**风险**：如不解决，Sprint 1-3 的所有前后端联调都只能在 Tauri 桌面端进行，严重影响开发效率。

---

#### B2. Tauri v2 安全配置缺失 — 无 Capability 文件

**现状**：
- `src-tauri/capabilities/` 目录**为空**（无任何 capability 文件）
- `tauri.conf.json` 中 `"security": { "csp": null }` — CSP 被完全禁用
- `package.json` 依赖了 `@tauri-apps/plugin-shell`，但 Rust 端 `main.rs` 中 `.plugin(tauri_plugin_shell::init())` **没有对应的 capability 授权**
- 无 `lib.rs` 文件（Tauri v2 移动端构建必需）

**影响**：
1. `tauri-plugin-shell` 在运行时会静默失败（Tauri v2 默认拒绝所有权限）
2. CSP 为 null 意味着 XSS 攻击无防护
3. 未来如需添加其他插件（fs、dialog、http），同样会因为缺少 capability 而失败
4. 无法支持移动端构建

**建议方案**：

1. 创建 `src-tauri/capabilities/default.json`：

```json
{
  "$schema": "../gen/schemas/desktop-schema.json",
  "identifier": "default",
  "windows": ["main"],
  "permissions": [
    "core:default",
    "shell:allow-open"
  ]
}
```

2. 更新 `tauri.conf.json`，添加 capabilities 引用并设置合理的 CSP：

```json
{
  "app": {
    "security": {
      "csp": "default-src 'self'; connect-src 'self' http://localhost:17421; img-src 'self' data:",
      "capabilities": ["default"]
    }
  }
}
```

3. 将 `main.rs` 中的应用逻辑迁移到 `lib.rs`：

```rust
// src-tauri/src/main.rs
#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]
fn main() {
    app_lib::run();
}

// src-tauri/src/lib.rs
#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
```

4. 更新 `Cargo.toml` 添加 `[lib]` 段：

```toml
[lib]
name = "app_lib"
crate-type = ["staticlib", "cdylib", "rlib"]
```

**风险**：如不解决，Sprint 4 的 Tauri 打包 smoke test 会因为权限问题而失败，且难以排查（Tauri v2 的权限失败是静默的，不报错，只返回 undefined/null）。

---

### 🟠 高优先级（HIGH）— 应在 Sprint 0 内解决

#### H1. Go 测试覆盖率仅 18.5%，workflow 模块为 0%

**现状**（基于实际覆盖率数据）：

| 模块 | 路径 | 覆盖率 | 方案预期 |
|------|------|--------|----------|
| Scope Engine | `internal/scope/` | 有测试文件 | Sprint 0-1 补测试 ✅ |
| Report Generator | `internal/report/` | 有测试文件 | Sprint 1 补测试 ✅ |
| Parser | `internal/parser/` | 有测试文件 | Sprint 1 补测试 ✅ |
| **Workflow** | `internal/workflow/` | **0.0%** | Sprint 2 — 仅 2h |
| **Screenshot Workflow** | `internal/workflow/screenshot.go` | **0.0%** | 未安排 |
| Asset Merger | `internal/asset/` | 无测试文件 | 未安排 |
| API Handlers | `internal/api/` | **0.0%** | Sprint 2 |
| DB Queries | `internal/db/` | 无测试文件 | 未安排 |
| **总计** | | **18.5%** | 目标 >60%（Sprint 4） |

**问题**：
1. `workflow/discovery.go`（587 行）和 `workflow/screenshot.go`（300+ 行）是**扫描流水线的核心**，0% 覆盖率意味着任何改动都是盲飞
2. 方案在 Sprint 2.11 给 workflow 测试只分配了 **2 小时**，完全不匹配这个模块的复杂度
3. `asset/merger.go` 和 `asset/normalizer.go`（资产去重/合并逻辑）也没有测试
4. API handlers 的集成测试依赖 httptest，目前完全没有

**建议**：
- **上调测试优先级**：workflow 测试至少应占 1 天（4-6h），且应从 Sprint 1 开始而非 Sprint 2
- **明确测试范围**：workflow 测试不需要端到端运行工具，可以通过 mock `worker.Runner` 来测试状态机转换逻辑
- **Sprint 0 增加覆盖率基线任务**：`go test ./... -cover -coverprofile=coverage.out` → 生成覆盖率报告 → 记录各模块当前基线
- **调整目标**：考虑到 workflow/API 的复杂度，Sprint 4 的 60% 覆盖率目标应该分解为：核心模块（scope/parser/report）80%+，workflow 50%+，整体 40%+

---

#### H2. Tailwind 配置现状与方案描述不符

**现状**：
- `tailwind.config.js` **已经存在且非常完善**，包含：
  - 完整的 `theme.extend.colors`（surface、brand、text-*、accent-*、glass-* 五组色系）
  - 自定义 `borderRadius`（apple-*）
  - 自定义 `boxShadow`（apple-* + glow-*）
  - 自定义 `animation` + `keyframes`（fadeIn、slideUp、slideDown、shimmer、pulse-slow）
  - 自定义 `fontFamily`（sans、mono）

**矛盾点**：
方案 Sprint 1.3 写道："审计现有 Tailwind 自定义 class，**补** `tailwind.config.js` theme.extend.colors"——但 config 已经齐全，不需要"补"。

方案待评审问题 #6 写道："只做 Tailwind theme.extend.colors 官方化"——已经在做了。

**建议**：
- 将 Sprint 1.3 从"补配置"改为**"审计并统一 class 命名规范"**
- 当前代码中可能混用了自定义 class（如 `text-accent-green`）和 Tailwind 原生工具类，需要统一
- 确认所有自定义 class 都有对应的 `tailwind.config.js` 注册项（当前已注册的覆盖了主要色系，不需要额外工作）
- **减少 Sprint 1.3 耗时**：从 1.5h 减少到 0.5h（仅确认不需要额外注册的 class 是否存在）

---

#### H3. React Context 方案的性能隐患

**问题**：方案提出用 React Context 在 ProjectLayout 中共享 project 数据：

> "fetch 项目信息后通过 React Context 传给子页面"

**风险分析**：
1. React Context 的值变化会导致**所有 consumer 组件重新渲染**，无论它们是否使用了变化的部分
2. RunsPage 使用 polling（每 5s 刷新）— 如果 polling 更新了 Context 中的某个字段（如 `lastScannedAt`），会导致 FindingsPage、AssetPage 等所有子页面也重新渲染
3. 项目切换时，`currentProject` 变化会导致整个 `<Outlet>` 子树重新挂载——方案已明确"重置筛选/排序/分页状态"，但未考虑渲染性能影响

**建议方案**（三选一）：

**方案 A（推荐）：Context 只传 projectId，不传完整 project 对象**

```tsx
// ProjectLayout 只提供 projectId
const ProjectContext = createContext<string | null>(null);

// 每个子页面根据 projectId 自己获取数据
function TargetsPage() {
  const projectId = useContext(ProjectContext);
  // 用 useQuery 或 useEffect + fetch 获取 targets
}
```

优点：无性能问题，子页面独立管理自身数据
缺点：每个页面都要写 `useEffect` + fetch（但这是常态）

**方案 B：Zustand store 管理 project 数据，用 selector 避免不必要渲染**

```typescript
// store 中增加
interface ProjectData {
  project: Project | null;
  loading: boolean;
  error: string | null;
}

// 组件中使用 selector
const projectName = useStore(s => s.currentProject?.name);
```

优点：数据集中管理，selector 实现细粒度订阅
缺点：store 变大，需要管理清理逻辑

**方案 C：保持 Context，但拆分为两个 Context（静态数据 + 动态数据）**

```tsx
const ProjectIdContext = createContext<string>();      // 不变
const ProjectDataContext = createContext<Project>();   // 可能变化
```

优点：ProjectSwitcher 切换时只触发 ProjectIdContext 变化
缺点：复杂度增加

**推荐方案 A**——对桌面工具来说最简单、最可预测。

---

#### H4. SSE 前端接入状态未知 — Sprint 2.7 的关键依赖

**现状**：
- 后端有完整的 SSE 基础设施：`GET /events` endpoint + `broadcastSSE()` + channel-based fan-out
- SSE 事件在 `handleRunTask`（行 1063）和 `handleCancelTask`（行 1142）中广播
- 前端：**方案标注"需验证"（Sprint 0.7）**

**问题**：
- Sprint 2.7（RunsPage 可靠性）明确依赖 SSE："Runs 页优先用 SSE（如 Sprint 0.7 确认已通）"
- 如果 Sprint 0.7 发现前端未接入 SSE，Sprint 2.7 的 3h 估算可能需要翻倍（实现 EventSource 订阅 + 重连 + 状态同步）
- 即使前端已接入 SSE，也需要确认：重连机制、SSE 断线时的降级策略、EventSource 与 AbortController 的配合

**建议**：
- **Sprint 0.7 产出应包含决策**：
  - 如果已接入 → 确认 EventSource 实现质量，产出 `docs/sse-status.md`
  - 如果未接入 → 评估接入工作量，Sprint 2.7 时间上调至 5-6h
- **SSE + Polling 的协作模式应明确定义**：
  - SSE 连接建立 → 暂停 polling
  - SSE 断线 → 指数退避重连（最多 3 次）→ 降级到 polling
  - 页面不可见 → 断开 SSE（节省连接），恢复时重新建立

---

#### H5. 缺失的设计 Token 映射表文档

**问题**：方案中待评审问题 #6 提到"只做 Tailwind theme.extend.colors 官方化"，但方案的 Sprint 1.3 和待评审问题都没有提到**需要建立一份 Token 映射表文档**。

**现状**：当前代码中已在使用自定义 class，如：
- `bg-surface` / `bg-surface-elevated` / `bg-surface-elevated-2`
- `text-text-primary` / `text-text-secondary`
- `text-accent-green` / `text-brand-primary` / `text-brand-danger`
- `border-glass-border` / `border-glass-border-light`
- `shadow-glow-blue` / `shadow-apple`

这些 class 覆盖了 7 个维度（surface/brand/text/accent/glass/shadow/radius），但**没有任何文档记录**。新人看不懂这些 class 的含义，PR review 时也无法判断何种场景用哪个 token。

**建议**：在 Sprint 1.3（0.5h 即可）产出 `docs/design-tokens.md`，格式如下：

```markdown
# Design Tokens

## 颜色系统

| Token | 用途 | 示例 |
|-------|------|------|
| `bg-surface` | 页面主背景 | Dashboard、Settings 背景 |
| `bg-surface-elevated` | 卡片/面板背景 | Card 组件 |
| `text-text-primary` | 主文本 | 标题、正文 |
| `text-text-secondary` | 次要文本 | 描述、标签 |
| `text-accent-green` | 成功/通过状态 | 扫描完成 Badge |
| `text-brand-danger` | 危险/错误状态 | 漏洞严重级别 Badge |
| ... | ... | ... |

## 使用原则
- 禁止直接使用 hex 颜色值
- 禁止混用 surface 层级（如 elevation-3 的卡片放在 elevation-2 的 panel 中无意义）
```

---

### 🟡 中优先级（MEDIUM）— Sprint 1-2 内解决

#### M1. API 客户端（api.ts）缺少基础能力

**现状**（`frontend/src/lib/api.ts`，310 行）：
- `fetchJSON<T>()` 函数：基础 fetch 包装，有 `APIError` 类
- **无请求超时**：长请求（扫描启动、报告生成）可能挂起无限久
- **无 AbortController 集成**：组件卸载后请求继续执行
- **无 204 处理**：`res.json()` 对空响应体抛异常
- **无 blob 响应处理**：报告导出需要 `Content-Disposition` → blob 下载
- **无重试机制**：网络瞬时故障无自动恢复
- **无并发请求去重**：快速点击可能发送重复请求

**与方案的对齐**：方案 §3.1-3.3 正确识别了这些问题，但实现细节需要补充。

**建议**：

```typescript
// 增加超时和取消支持
async function fetchJSON<T>(
  path: string,
  opts?: RequestInit & { timeout?: number; signal?: AbortSignal }
): Promise<T> {
  const { timeout = 30000, signal: extSignal, ...fetchOpts } = opts || {};

  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), timeout);

  // 合并外部 signal
  if (extSignal) {
    extSignal.addEventListener("abort", () => controller.abort());
  }

  try {
    const res = await fetch(`${API_BASE}${path}`, {
      headers: { "Content-Type": "application/json" },
      signal: controller.signal,
      ...fetchOpts,
    });

    // 204 No Content
    if (res.status === 204) return null as T;

    if (!res.ok) {
      const data = await res.json().catch(() => null);
      const message = data?.error?.message || `${res.status}: ${res.statusText}`;
      throw new APIError(message, data?.error?.code);
    }

    return res.json();
  } catch (err) {
    if (err instanceof DOMException && err.name === "AbortError") {
      throw new APIError("请求已取消", "CANCELLED");
    }
    if (err instanceof TypeError && err.message === "Failed to fetch") {
      throw new APIError("网络连接失败，请检查网络", "NETWORK_ERROR");
    }
    throw err;
  } finally {
    clearTimeout(timeoutId);
  }
}

// 新增 blob 下载支持
async function fetchBlob(
  path: string,
  opts?: RequestInit & { timeout?: number }
): Promise<{ blob: Blob; filename: string }> {
  const { timeout = 60000, ...fetchOpts } = opts || {};
  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), timeout);

  try {
    const res = await fetch(`${API_BASE}${path}`, {
      signal: controller.signal,
      ...fetchOpts,
    });

    if (!res.ok) {
      // blob 端点也可能返回 JSON 错误
      const contentType = res.headers.get("content-type") || "";
      if (contentType.includes("application/json")) {
        const data = await res.json();
        throw new APIError(
          data?.error?.message || "导出失败",
          data?.error?.code
        );
      }
      throw new APIError(`导出失败: ${res.status}`, "EXPORT_ERROR");
    }

    const disposition = res.headers.get("content-disposition") || "";
    const filename =
      disposition.match(/filename="?([^"]+)"?/)?.[1] || "download";

    return { blob: await res.blob(), filename };
  } finally {
    clearTimeout(timeoutId);
  }
}
```

---

#### M2. Zustand Store 缺少加载/错误状态

**现状**（`frontend/src/lib/store.ts`）：
- 只有数据字段 + setter，无 loading/error 状态
- `addTask`/`updateTask` 只能追加/更新，无法表示"正在加载任务列表"
- 无 per-entity 的状态区分（全局 task 列表 vs 单个 task 详情）

**建议**：方案说"分层管理"是对的，但需要明确定义——哪些放 store，哪些放组件：

```typescript
// store.ts — 增加的 loading/error 状态
interface AppState {
  // 数据
  projects: Project[];
  // 加载/错误状态（按实体分离）
  loadState: {
    projects: { loading: boolean; error: string | null };
    targets: { loading: boolean; error: string | null };
    assets: { loading: boolean; error: string | null };
    tasks: { loading: boolean; error: string | null };
    findings: { loading: boolean; error: string | null };
  };
  setLoadState: (
    entity: keyof AppState["loadState"],
    state: Partial<{ loading: boolean; error: string | null }>
  ) => void;

  // 当前项目（用于 ProjectLayout）
  currentProjectId: string | null;
  setCurrentProjectId: (id: string | null) => void;
}
```

**组件端**（不放入 store）：
- 表单提交状态（创建/编辑/删除的 loading）
- 单项操作的错误提示（如"目标导入失败：xxx"）
- 这些属于 UI 局部状态，用 `useState` 即可

---

#### M3. SQLite Schema 迁移策略缺失

**问题**：方案没有提及数据库 schema 变更策略。当前使用 SQLite（`internal/db/db.go`），v0.3 改造可能涉及：
- 新增字段（如 scope rules 的批量操作记录）
- 索引优化（查询性能）

**风险**：没有迁移机制，每次 schema 变更只能：
1. 删库重建（开发阶段可接受，但已积累的测试数据丢失）
2. 手动 `ALTER TABLE`（容易遗漏）

**建议**：
- **短期（Sprint 0-2）**：在 `db.go` 的 `Open()` 中增加 `PRAGMA user_version` 检查，简单的 `ALTER TABLE IF NOT EXISTS` 即可
- **长期（v0.4+）**：引入 golang-migrate 或类似工具
- **Sprint 0**：记录当前 schema 版本，作为基线

---

#### M4. 报告导出 Blob 边界约定需明确定义

**现状**：
- 后端有两个报告导出端点：`GET /projects/{id}/reports/export.md` 和 `GET /projects/{id}/reports/export.json`
- 方案 Sprint 0.1 标注"主要确认 blob 边界"

**问题**：blob 端点的错误处理与 JSON API 完全不同：

| 场景 | JSON API 响应 | Blob API 应响应 |
|------|--------------|----------------|
| 项目不存在 | `{"error":{"code":"NOT_FOUND","message":"..."}}` | `{"error":...}` 或 404 + 空体？ |
| 无 findings | `{"error":{"code":"BAD_REQUEST","message":"..."}}` | 同 JSON |
| 报告生成失败 | 500 + error JSON | 500 + error JSON |

**关键问题**：方案没有明确 **blob 端点在错误时应该返回什么**——返回 JSON（前端需要检查 Content-Type）还是返回 HTTP 错误码 + 空体？

**建议**（在 `docs/api-error-contract.md` 中明确）：

```markdown
## Blob 端点错误约定

所有 blob 下载端点（report export, archive download）在错误时：
- 返回 `application/json` Content-Type
- 响应体格式与普通 API 一致：`{"error": {"code": "...", "message": "..."}}`
- HTTP 状态码与普通 API 一致（404/400/500）

前端处理逻辑：
1. 检查 `response.ok`
2. 如果不 ok，检查 `Content-Type`：
   - `application/json` → 解析错误 JSON → 抛出 APIError
   - 其他 → 抛出通用错误
3. 如果 ok，读取 blob
```

---

#### M5. `usePolling` Hook 与 SSE 的协作模式未定义

**问题**：方案提出"统一用 `usePolling(fetchFn, intervalMs)` hook"，同时又提出"Runs 页优先用 SSE"。这两个模式如何共存？

- 如果 RunsPage 使用 SSE，它就不需要 `usePolling`
- 如果 SSE 断线，RunsPage 需要降级到 `usePolling`
- `usePolling` 的"页面不可见时暂停"与 SSE 的"断开节省连接"逻辑互斥

**建议**：明确定义 SSE + Polling 的协作语义：

```typescript
// hooks/useRealtimeData.ts
function useRealtimeData(options: {
  sseUrl?: string;           // SSE endpoint (如 "/api/events")
  pollFn?: () => Promise<T>; // 降级用 polling
  interval?: number;         // 降级 polling interval
}) {
  // 优先 SSE，断线 3 次后退避到 polling
  // 页面不可见 → 断开 SSE，恢复时重连
  // 返回 { data, error, isLive } (isLive 表示当前是 SSE 还是 polling)
}

// WorkersPage：只需 polling
usePolling(fetchWorkers, 5000);

// RunsPage：SSE 优先 + polling 降级
useRealtimeData({
  sseUrl: "/api/events",
  pollFn: fetchRuns,
  interval: 5000,
});
```

---

#### M6. Vite proxy 配置缺失后的 API 联调路径不明确

与 B1 相关但侧重开发体验：当前项目开发时，前端开发者需要：
1. 先启动 Go 后端（`go run .` 或 `make dev`）
2. 再启动前端（`npm run dev`）
3. 还需要确认 API 能通

方案中的 Sprint 规划没有提及**每次 Sprint 的验收环境是什么**——是在 Tauri 桌面端验收还是在浏览器验收？

**建议**：在每个 Sprint 的验收标准中明确：
```markdown
- [ ] 浏览器 dev 模式（`npm run dev` + `go run .`）通过
- [ ] Tauri 桌面端（`npm run tauri dev`）通过
```

---

### 🟢 低优先级（LOW）— Sprint 3-4 或后续

#### L1. 无结构化日志方案

对于一个"Usability & Reliability"阶段，如果出 bug 只能靠 `console.log` 和 `fmt.Printf` 排查，效率会很低。建议在 Sprint 2-3 中：
- 前端：加 `loglevel` 或类似轻量库，区分 `debug`/`info`/`warn`/`error`
- 后端：已有的 `log.Printf` 保持，但加 request ID 追踪（middleware 注入 `X-Request-ID`）

#### L2. Makefile 缺少 `go vet` 步骤

方案 §5.4 合并前检查清单包含 `go vet ./...`，但 Makefile 的 `test` target 只运行 `go test ./...`。建议：

```makefile
check:
	go vet ./...
	go test ./... -race -count=1
```

#### L3. E2E 测试目前是烟雾测试，无断言能力

现有的 10 个 `.e2e.md` 文件（如 `App.e2e.md`）使用 `agent-browser screenshot` + 人工判断。对于验收测试这可以接受，但方案 Sprint 4 提出的"10 条主流程验收用例全部 pass"需要一个**自动化的 pass/fail 判定机制**。

建议在 `docs/acceptance-criteria.md` 中为每条用例定义明确的 pass 条件（如 URL 匹配、特定文本出现、特定元素可见），让 agent-browser 能自动判断。

#### L4. 无主题切换方案

当前 `tailwind.config.js` 中 `darkMode: "class"` 且所有颜色都是暗色主题（`#0B0E14` 等）。方案"不在本阶段范围"中明确排除了深色/浅色主题切换，但需注意：如果后续要支持亮色主题，当前的 CSS 变量体系（都在 `:root` 下定义，没有 `.dark` scope）需要重构。建议至少将当前的颜色定义标注为"dark-only"。

---

## 三、方案遗漏的关键考虑

### 3.1 Tauri 打包产物与 Go 后端的发布关系

方案未说明桌面应用如何获取 Go 后端：
- **当前模式**：用户手动启动 Go 后端 → 再打开 Tauri 桌面应用
- **预期模式**：Sidecar 模式（Go binary 作为 Tauri sidecar 随应用分发）vs 独立服务模式

这对 Sprint 4 的"Tauri 打包后桌面 smoke test"至关重要——如果打包后的 `.app` 不带 Go 后端，smoke test 需要通过外部方式启动后端。

**建议**：在方案中明确 v0.3 阶段的桌面应用启动方式，即使是"用户先启动 `./anchor server`，再打开 `.app`"也需要文档化。

### 3.2 Scope 确认页面的数据流设计

方案将 Scope 确认独立为 `/projects/:id/scope` 页面，但未说明：
- Scope 确认页面展示的是 Dry Run 结果还是当前 Scope Rules？
- Dry Run 的"approve"操作是创建 Scope Rules 还是直接标记 Target 为 allowed？
- 如果用户在 Scope 页面做了 allow/deny 后，回到 Targets 页面，Target 列表是否需要更新？

这些在 Sprint 2.6 实现前需要明确。

### 3.3 Worker 离线检测的具体实现

方案 Sprint 2.8 提到"Worker 离线检测"，但当前 Worker 架构是：
- Worker 向 Core 注册（`POST /workers/register`）
- Worker 每 30s 发送心跳（`POST /workers/{id}/heartbeat`）
- Core 的 SSE 广播 worker 状态变化

**问题**：前端如何判断 worker 离线？是通过 SSE 事件，还是通过 polling workers 列表并检查 `last_heartbeat` 字段？如果是后者，`last_heartbeat` 字段是否已在 API 响应中？

### 3.4 Findings 批量操作的事务性

方案 Sprint 3.3 提出 Findings 状态修改（confirmed/false_positive/accepted_risk + 批量操作），但未考虑：
- 批量修改 100 个 findings 时，部分成功部分失败如何处理？
- 是否需要乐观更新（前端先改 UI，后端确认后最终一致）？
- 批量操作的 API 端点是否已存在？

### 3.5 并发编辑冲突

方案聚焦单用户桌面工具，但如果有多个 Worker 同时扫描同一个项目：
- 两个 Worker 同时发现同一资产，后端 merger 去重逻辑是否足够？
- 用户在查看 Findings 页面时，Worker 正在写入新 finding，前端如何更新？

---

## 四、方案亮点（值得保留的设计决策）

1. **决策 1（混合嵌套路由）**：正确选择。对齐了后端 API 的 project-scoped 设计，解决了当前路由无法感知项目上下文的核心问题。

2. **决策 2（agent-browser E2E）**：务实选择。当前阶段不需要 Playwright 的重量级测试矩阵，agent-browser 的截图+快照模式匹配验收测试需求。

3. **§3 工作流设计**：API 错误处理增强（§3.1-3.4）覆盖全面，对 `fetchJSON` 的增强需求分析准确。后端 `AppError` 结构已统一（确认代码审计后）。

4. **ProjectLayout 交互规范**：5 个边界场景（404 项目、项目切换、加载中 Skeleton、项目删除、时间窗口过期）考虑周到，这是容易遗漏的细节。

5. **风险与缓解表（§六）**：诚实且有针对性，没有粉饰风险。特别是"agent-browser E2E 维护成本高"和"Tauri WebView 与浏览器不一致"的认识很务实。

6. **明确的排除范围（§七）**：清晰定义了 v0.3 的边界，避免了范围蔓延。

---

## 五、Sprint 耗时修正建议

基于代码审计发现，建议调整以下 Sprint 耗时：

| 任务 | 原估算 | 建议 | 原因 |
|------|--------|------|------|
| Sprint 0.1（API 审计） | 0.5h | 保持 | 后端已统一，仅确认 blob 边界 ✅ |
| Sprint 0.4（typecheck） | 1h | 0.5h | tsconfig 已开 strict，现有错误应不多 |
| Sprint 0.8（Go 测试框架） | 1.5h | 2h | 覆盖率基线报告（新增需求）+ 第一个测试文件 |
| Sprint 1.3（Tailwind 审计） | 1.5h | 0.5h | 配置已齐全，仅需确认命名规范 |
| Sprint 2.11（workflow 测试） | 2h | 4-6h | 0% 覆盖率的 900+ 行核心代码 |
| Sprint 4.3（Go 测试查漏） | 3h | 4-5h | 18.5% → 40%+ 需要显著工作量 |
| **新增：Tauri capability** | — | 1.5h | B1 阻塞项，Sprint 0 必须完成 |
| **新增：Vite proxy** | — | 0.5h | B2 阻塞项，Sprint 0 必须完成 |

修正后总耗时：**约 18.5-28 天**（中位 23 天，比原估算增加约 3 天）

---

## 六、整体评分

| 维度 | 评分 | 评语 |
|------|------|------|
| 方案完整性 | 8/10 | 覆盖主流程 + 边界场景 + 异常路径，少数数据流细节待补充 |
| 技术可行性 | 7/10 | 2 个阻塞项需修复，之后即可行 |
| 架构合理性 | 8/10 | 路由决策、交互规范、分层管理均合理 |
| 可维护性 | 7/10 | 测试策略清晰但覆盖率基线太低 |
| 安全性 | 6/10 | Tauri 安全配置缺失是硬伤 |
| 性能与扩展性 | 7/10 | Polling/SSE 设计合理，Context 方案需优化 |
| **综合** | **7.2/10** | |

### 最终建议

**可以启动，但 Sprint 0 范围需扩大**。当前 Sprint 0 预计 1.5-2 天应扩展为 2.5-3 天，以容纳 2 个阻塞项 + 5 个高优先级项的初步处理。Sprint 2 需要为 workflow 测试预留更多时间。

**启动条件**：
- [ ] B1: Vite proxy 配置完成
- [ ] B2: Tauri capability 文件创建 + lib.rs 迁移
- [ ] H1: 生成覆盖率基线报告（`go tool cover -func=coverage.out`）
- [ ] H2: 确认 Tailwind config 命名规范，更新 Sprint 1.3 任务描述
- [ ] H3: 确定 ProjectLayout 的 context 实现方案（推荐方案 A）
- [ ] H4/H5: Sprint 0 产出中增加设计 token 文档 + SSE 接入决策

---

> **差异化视角**：本评审的核心贡献不在于发现了更多 bug，而在于**揭示了一个系统性风险——方案对自己的工程资产状况存在认知偏差**。方案基于代码审计后做了很好的修订（v1.0 → v1.1），但 18.5% 的 Go 测试覆盖率（workflow 0%）和 Tauri 安全配置的空缺是两个硬性信号，表明项目的"可测试性基础设施"和"安全基线"比方案假设的更薄弱。建议在 Sprint 0 增加"工程健康检查"任务，产出 `docs/engineering-health.md`，记录当前的覆盖率、lint 状态、依赖版本、安全配置等基线指标，为后续 Sprint 的"可靠性"目标提供量化参照。
