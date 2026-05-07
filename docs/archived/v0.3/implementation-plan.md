---
status: review_material
source_of_truth: false
owner: kun
last_updated: 2026-05-04
scope: v0.3-review-material
---

# v0.3 Desktop Usability & Reliability — 多模型评审后实施方案

> **版本**：v2.0（基于多模型评审汇总修订）  
> **原始方案**：`docs/v0.3-plan.md` v1.1  
> **评审日期**：2026-04-29  
> **参与模型**：kimi-coding、glm-5.1、deepseek-v4-pro（mimo-v2.5-pro 调用失败，未参与）
> **状态**：✅ 已完成（2026-05-01）
> **文档角色**：评审汇总材料，不是当前仓库级计划

---

## 一、执行摘要

### 1.1 方案核心

v0.3 阶段定位为**"不堆新功能，把现有桌面工具打磨到稳定、顺手、可信赖"**。原始方案在路由重构（混合嵌套路由）、API 错误治理、三层测试策略、ProjectLayout 交互规范等方面体现了扎实的代码审计工作和务实的工程观。

### 1.2 各模型评分概览

| 模型 | 评分 | 关键结论 |
|------|------|----------|
| **kimi-coding** | **7.5/10** | 思路清晰、可执行性强，但安全维度几乎空白，Tauri 桌面端特殊性被低估，缺少数据迁移与降级策略 |
| **glm-5.1** | **7.5/10** | 高于平均水平的迭代计划，但 SSE 工作量被严重低估，Tauri 安全配置存在明显缺陷，项目切换数据一致性未设计 |
| **deepseek-v4-pro** | **7.2/10** | 良好方案，需修复 2 个阻塞项 + 5 个高优先级项后方可启动。揭示了工程资产状况的认知偏差（18.5% 覆盖率、workflow 0%） |
| **平均** | **7.4/10** | 方案在"做什么"层面清晰，但在"怎么做"的基础设施和安全基线上存在共识性缺口 |

### 1.3 评审核心结论

**三个模型一致认为**：
1. **Tauri 安全配置是硬伤**（CSP 为 null、无 capabilities、无权限配置），必须在 Sprint 0 修复
2. **SSE 前端未实现**，工作量被严重低估，不是"验证"而是"从零开发"
3. **Zustand Store 改造缺少具体设计方案**，需产出设计草案后才能开工
4. **Sprint 2 任务密度过高**，需拆分或延长工期
5. **缺少前端 ErrorBoundary**，未捕获渲染错误会导致整页白屏

**修正后总工期**：**19-25 天**（中位 22 天），比原估算 16-24.5 天增加约 2-3 天，主要用于 Tauri 安全基线、SSE 前端实现、工程健康检查。

---

## 二、共识问题清单（所有模型一致指出，必须解决）

### 🔴 阻塞级（BLOCKING）— Sprint 0 启动前必须完成

#### 【共识-B1】Tauri v2 安全配置严重缺失
- **来源**：glm-5.1、deepseek-v4-pro 均评为阻塞；kimi-coding 评为高
- **问题**：
  - `tauri.conf.json` 中 `"security": { "csp": null }` — CSP 完全禁用
  - `src-tauri/capabilities/` 目录为空 — Tauri v2 默认拒绝所有操作
  - `package.json` 依赖了 `@tauri-apps/plugin-shell`，但无对应 capability 授权
  - 无 `lib.rs` 文件（Tauri v2 移动端构建必需）
- **影响**：作为安全测试工具，CSP 为 null 是品牌信誉问题；报告导出等文件操作将静默失败
- **实施方案**：
  1. 创建 `src-tauri/capabilities/default.json`，至少包含 `core:default`、`shell:allow-open`
  2. 配置 CSP：`"default-src 'self'; connect-src 'self' http://localhost:17421; img-src 'self' data: blob:; style-src 'self' 'unsafe-inline'"`
  3. 迁移 `main.rs` 逻辑到 `lib.rs`，更新 `Cargo.toml` 添加 `[lib]` 段
  4. 安装 `tauri-plugin-dialog`（报告导出需要系统保存对话框）
- **负责人**：前端 + Tauri 配置
- **耗时**：1.5h
- **验收**：`npm run tauri dev` 正常启动，无权限报错；`shell:open` 可打开外部链接

#### 【共识-B2】SSE 前端完全未实现，工作量被低估
- **来源**：glm-5.1、deepseek-v4-pro 均评为阻塞；kimi-coding 评为中
- **问题**：
  - 后端 SSE 基础设施已就绪（`handleSSE` + `broadcastSSE`）
  - 前端零 SSE 代码：无 `EventSource`、无 `useSSE` hook
  - `RunsPage.tsx` 使用一次性 `useEffect` 拉取，无任何实时更新
  - 方案 Sprint 0.7 标注"验证 SSE 通信路径，1h" — 实际是**从零实现前端 SSE 客户端**
- **影响**：RunsPage 实时进度更新是核心功能，当前完全没有；Sprint 2.7 依赖 SSE 的 3h 估算严重不足
- **实施方案**：
  1. **Sprint 0.7** 改为"确认状态 + 产出 SSE 前端架构决策"（1h）
  2. **Sprint 2 新增任务**：实现 `useSSE` hook（4-6h），包含：
     - 自动重连（指数退避）
     - 心跳超时检测（后端需补充每 30s 发 `{"event":"ping"}`）
     - 页面可见性管理（切 tab 时断开/恢复）
     - 消息类型过滤（按 `project_id` 过滤事件）
     - SSE 断线降级到 `usePolling` 的 fallback 机制
  3. 后端 `broadcastSSE` 改造为 `broadcastProjectSSE`，支持按项目过滤（避免 Project A 事件推给 Project B 用户）
- **负责人**：前端 + 后端（SSE 过滤改造）
- **耗时**：Sprint 0: 1h（决策）+ Sprint 2: 4-6h（实现）
- **验收**：RunsPage 能实时接收任务进度更新；SSE 断线 3 次后自动降级到 polling

#### 【共识-B3】Vite 开发服务器未配置 API 代理
- **来源**：deepseek-v4-pro 评为阻塞；glm-5.1 在 M6 中提到
- **问题**：`vite.config.ts` 无 proxy 配置，浏览器开发模式下前端跨域请求 Go API
- **影响**：Sprint 1-3 的所有前后端联调只能在 Tauri 桌面端进行，严重影响开发效率
- **实施方案**：
  ```typescript
  // vite.config.ts
  export default defineConfig(async () => ({
    plugins: [react()],
    clearScreen: false,
    server: {
      port: 1420,
      strictPort: true,
      watch: { ignored: ["**/src-tauri/**"] },
      proxy: {
        "/api": {
          target: "http://localhost:17421",
          changeOrigin: true,
        },
      },
    },
  }));
  ```
  同时更新 `config.ts`：Web 模式下 `getApiBase()` 返回 `""`（使用相对路径）
- **负责人**：前端
- **耗时**：0.5h
- **验收**：浏览器 dev 模式（`npm run dev`）可直接调用 Go API，无 CORS 报错

---

### 🟠 高优先级（HIGH）— Sprint 0 完成，或 Sprint 1 启动前完成

#### 【共识-H1】Zustand Store 改造方案过于笼统
- **来源**：三个模型均评为高
- **问题**：
  - 当前 `store.ts` 是单一大 store（~70 行），无 loading/error 状态管理
  - 未说明 store 结构如何改造（单 store + 字段 vs 多 slice）
  - 未说明数据清理策略（切换项目时子资源是否清空）
  - 未说明缓存策略（tab 切换后是否重新 fetch）
- **实施方案**（推荐采用 **deepseek 的 slice 化方案**，kimi 和 glm 也倾向此方向）：
  ```typescript
  // store.ts — slice 化结构
  interface AppState {
    // 数据
    projects: Project[];
    targets: Target[];
    assets: Asset[];
    findings: Finding[];
    runs: Run[];
    // 加载/错误状态（按实体分离）
    loadState: {
      projects: { loading: boolean; error: string | null };
      targets: { loading: boolean; error: string | null };
      assets: { loading: boolean; error: string | null };
      findings: { loading: boolean; error: string | null };
      runs: { loading: boolean; error: string | null };
    };
    // 当前项目上下文
    currentProjectId: string | null;
    setCurrentProjectId: (id: string | null) => void;
  }
  ```
  **关键决策**（Sprint 0.6 产出）：
  1. 切换项目时**清空所有子资源 store**（targets, assets, findings, runs），避免显示旧数据
  2. 使用 Zustand `persist` 中间件持久化 `currentProjectId`（替代 localStorage 手动管理）
  3. 保持单 store + slice 模式（v0.3 不过度拆分为独立 hooks）
  4. 引入 `devtools` 中间件便于调试
- **负责人**：前端
- **耗时**：Sprint 0.6: 1h（设计草案）+ Sprint 2.4: 5-6h（实现）
- **验收**：切换项目时旧数据不残留；所有页面 loading/error 状态统一从 store 读取

#### 【共识-H2】React Context 方案存在性能隐患，建议改用 Zustand
- **来源**：kimi-coding、deepseek-v4-pro 评为中；glm-5.1 在 H1 中隐含
- **问题**：
  - 方案计划用 React Context 在 ProjectLayout 传递 project 信息
  - Context 值变化会导致**所有 consumer 重新渲染**（即使只用到部分字段）
  - 与 Zustand 并存造成状态管理分裂
- **冲突消解**：三个模型一致建议用 Zustand 替代 Context。
  - **kimi 建议**：项目上下文直接写入 Zustand store，`useParams()` 同步 `currentProjectId`
  - **deepseek 建议**：Context 只传 `projectId`，子页面自己获取数据（最简单）
  - **推荐方案**：采用 **deepseek 方案 A**（Context 只传 projectId）作为过渡，同时 Zustand store 管理 `currentProjectId`。这样性能最优，且与 Sprint 2.4 的 store 改造不冲突。
- **负责人**：前端
- **耗时**：Sprint 1.1: 包含在 ProjectLayout 实现中（不额外增加时间）
- **验收**：ProjectSwitcher 切换时，只有目标页面重新 fetch，非相关页面不重新渲染

#### 【共识-H3】Sprint 2 任务密度过高，必须拆分或延长
- **来源**：kimi-coding、glm-5.1、deepseek-v4-pro 均指出
- **问题**：
  - Sprint 2 原估算 5-7 天，实际需 7-9 天（33h → 41-48h）
  - 包含 API 治理、Store 改造、SSE 实现、5 个页面可靠性增强、Go 集成测试
  - 未预留 Bug 修复缓冲（若 Sprint 0 发现 P0 问题，Sprint 2 将被挤压）
- **实施方案**：
  - **方案 A（推荐）**：将 Sprint 2 拆分为 **Sprint 2a**（API 层 + Store 改造 + SSE，3-4 天）和 **Sprint 2b**（页面可靠性改造，4-5 天）
  - **方案 B**：延长 Sprint 2 至 8-9 天，并明确"若 Sprint 0 发现 >3 个 P0 bug，则 Sprint 2 仅做修复，不做新功能"
- **负责人**：项目负责人（排期调整）
- **验收**：每个 sub-sprint 有独立验收标准，不超期

#### 【共识-H4】缺少前端 ErrorBoundary，未捕获错误导致白屏
- **来源**：kimi-coding、glm-5.1、deepseek-v4-pro 均指出
- **问题**：任何未捕获的渲染错误（如 API 返回非预期格式导致 JS 报错）会使整个应用白屏
- **实施方案**：
  1. 在 App 根节点添加全局 ErrorBoundary，渲染 fallback UI + 重新加载按钮
  2. 在 ProjectLayout 添加独立 ErrorBoundary，隔离项目上下文错误
  3. 集成 Tauri `log` plugin，将前端错误日志写入本地文件
  ```typescript
  class ErrorBoundary extends Component<
    { fallback?: ReactNode; children: ReactNode },
    { hasError: boolean }
  > {
    state = { hasError: false };
    static getDerivedStateFromError() { return { hasError: true }; }
    componentDidCatch(error: Error, info: ErrorInfo) {
      console.error('[ErrorBoundary]', error, info.componentStack);
      // 可接入 Tauri log plugin
    }
    render() { /* fallback UI */ }
  }
  ```
- **负责人**：前端
- **耗时**：1.5h（Sprint 1 或 Sprint 2）
- **验收**：手动制造一个渲染错误（如访问 undefined 属性），确认显示 fallback UI 而非白屏

#### 【共识-H5】项目切换数据一致性未设计
- **来源**：glm-5.1、deepseek-v4-pro 明确提到；kimi-coding 在 ProjectLayout 边界场景中隐含
- **问题**：
  - 方案说"筛选/排序/分页状态重置"，但没说 store 中的**数据本身**是否清空
  - 不清空 → 用户会在 Project B 页面临时看到 Project A 的数据
  - 清空 → 出现短暂的"空数据闪烁"（EmptyState → loading → 新数据）
- **实施方案**：
  1. 切换项目时**立即清空所有子资源 store**（targets, assets, findings, runs）
  2. ProjectLayout fetch 新项目信息期间显示**全页 Skeleton**（已计划在 Sprint 1.1）
  3. 子页面检测到 `currentProjectId` 变化后自动触发新 fetch
  4. 在 store 的 `setCurrentProject` action 中统一处理：
  ```typescript
  setCurrentProject: (projectId: string | null) => set({
    currentProjectId: projectId,
    targets: [],
    assets: [],
    findings: [],
    runs: [],
    // 保留 projects 列表（全局数据）
  })
  ```
- **负责人**：前端
- **耗时**：包含在 Sprint 2.4（Zustand 改造）中
- **验收**：切换项目后，旧项目数据不残留；新数据加载前有 Skeleton

#### 【共识-H6】Go 测试覆盖率基线过低，workflow 模块为 0%
- **来源**：deepseek-v4-pro 详细数据；glm-5.1 提到；kimi-coding 提到 Sprint 2 密度高
- **问题**：
  - Go 总覆盖率仅 **18.5%**
  - `internal/workflow/`（扫描流水线核心，900+ 行）覆盖率为 **0%**
  - Sprint 2.11 只分配了 2h 做 workflow 测试，完全不匹配复杂度
- **实施方案**：
  1. **Sprint 0.8** 增加：生成覆盖率基线报告（`go test -coverprofile=coverage.out ./...` → `go tool cover -func=coverage.out`）
  2. **调整 Sprint 4 目标**：核心模块（scope/parser/report）80%+，workflow 50%+，**整体 40%+**（而非原方案的 60% 整体）
  3. **Sprint 2.11** 上调至 **4-6h**，workflow 测试采用 mock `worker.Runner` 测试状态机转换（无需端到端运行工具）
  4. **Sprint 1.9** 增加 API handler 集成测试起步（至少 project CRUD）
- **负责人**：后端
- **耗时**：Sprint 0: +0.5h（基线报告）；Sprint 2: +2-4h（workflow 测试）；Sprint 4: +1-2h（查漏补缺）
- **验收**：产出 `docs/coverage-baseline.md`，记录各模块当前覆盖率

---

### 🟡 中优先级（MEDIUM）— Sprint 1-2 内解决

#### 【共识-M1】api.ts 错误处理增强（网络错误、blob 错误、超时、204、重试）
- **来源**：三个模型均详细提到（kimi H-3/M-9、glm H3/M5、deepseek M1）
- **问题**：
  - 无请求超时（长请求可能挂起）
  - 无 AbortController（组件卸载后请求继续）
  - 无 204 处理（`res.json()` 对空响应体抛异常）
  - 无 blob 响应处理（报告导出需要 `Content-Disposition`）
  - 无重试机制（网络瞬时故障无自动恢复）
  - `.catch(console.error)` 吞掉错误，用户看不到提示
- **实施方案**：
  ```typescript
  async function fetchAPI<T>(
    path: string,
    opts?: RequestInit & { timeout?: number; signal?: AbortSignal }
  ): Promise<T> {
    const { timeout = 30000, signal: extSignal, ...fetchOpts } = opts || {};
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), timeout);
    if (extSignal) {
      extSignal.addEventListener("abort", () => controller.abort());
    }
    try {
      const res = await fetch(`${API_BASE}${path}`, {
        ...fetchOpts,
        signal: controller.signal,
        headers: {
          ...(fetchOpts.method !== 'GET' ? { "Content-Type": "application/json" } : {}),
          ...fetchOpts.headers,
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
  同时增加 `fetchBlob()` 函数处理报告导出，全局错误拦截触发 Toast 而非静默吞掉。
- **负责人**：前端
- **耗时**：4-5h（Sprint 2.1）
- **验收**：网络断开时显示 Toast 提示；组件卸载后请求被取消；报告导出错误时正确解析 JSON 错误体

#### 【共识-M2】路由 Breaking Change 缺少迁移与降级策略
- **来源**：kimi-coding 评为高；deepseek 在遗漏项中提到
- **问题**：删除旧路由后，已收藏书签、历史记录、localStorage 中旧路由无法访问
- **实施方案**：
  ```tsx
  // App.tsx — 兼容层，Sprint 2 后移除
  function LegacyRouteGuard() {
    const location = useLocation();
    const lastProjectId = localStorage.getItem('lastProjectId');
    useEffect(() => {
      console.warn(`[Deprecation] Accessed legacy route: ${location.pathname}`);
    }, [location]);
    if (!lastProjectId) return <Navigate to="/projects" replace />;
    const legacyMap: Record<string, string> = {
      '/targets': `/projects/${lastProjectId}/targets`,
      '/assets': `/projects/${lastProjectId}/assets`,
      '/runs': `/projects/${lastProjectId}/runs`,
      '/findings': `/projects/${lastProjectId}/findings`,
      '/reports': `/projects/${lastProjectId}/reports`,
    };
    const redirectTo = legacyMap[location.pathname];
    return redirectTo ? <Navigate to={redirectTo} replace /> : <Navigate to="/projects" replace />;
  }
  ```
- **负责人**：前端
- **耗时**：1h（Sprint 1.1）
- **验收**：访问 `/targets` 自动重定向到 `/projects/{lastProjectId}/targets`

#### 【共识-M3】Tailwind 配置与方案描述不符
- **来源**：glm-5.1、deepseek-v4-pro 均指出
- **问题**：`tailwind.config.js` 已有完整设计体系（surface/brand/text/accent/glass/radius/shadow），方案 Sprint 1.3 仍说"补充 theme.extend.colors"
- **实施方案**：
  - Sprint 1.3 改为**"审计并统一 class 命名规范"**
  - 找出页面中的硬编码颜色（如 `text-zinc-100`、`bg-yellow-500/15`），统一替换为语义 Token
  - 产出 `docs/design-tokens.md` 映射表（deepseek H5 建议）
- **负责人**：前端
- **耗时**：0.5h（原 1.5h → 0.5h，节省 1h）
- **验收**：无裸 `bg-zinc-*`、`text-yellow-*` 等硬编码颜色

#### 【共识-M4】报告导出在 Tauri 模式下的实现路径未明确
- **来源**：glm-5.1、deepseek-v4-pro 均指出
- **问题**：
  - 浏览器模式：`<a download>` 触发下载
  - Tauri 模式：需要 `tauri-plugin-dialog` 的 `save()` API 弹出系统保存对话框
  - 当前无 `fs` 或 `dialog` 插件配置
- **实施方案**：
  1. 安装 `tauri-plugin-dialog`：`cargo add tauri-plugin-dialog` + `npm add @tauri-apps/plugin-dialog`
  2. 在 `capabilities/default.json` 添加 `"dialog:default"`
  3. 前端判断 Tauri 环境时使用 `save()` dialog，浏览器环境用 `<a download>` fallback
  4. Blob 端点错误约定：错误时返回 `application/json` + `{"error": {"code": "...", "message": "..."}}`
- **负责人**：前端 + Tauri 配置
- **耗时**：2h（Sprint 2.9）
- **验收**：Tauri 模式下导出报告弹出系统保存对话框；错误时正确显示 Toast

#### 【共识-M5】缺少"后端未运行"的前端检测与引导
- **来源**：glm-5.1、deepseek-v4-pro 均指出
- **问题**：用户首次打开 Tauri 桌面端，Go 后端未启动时，页面空白或永远 loading
- **实施方案**：
  1. 添加 `api.healthCheck()` 方法，调用 `/health` endpoint（如不存在则新增）
  2. App 初始化时执行 health check，失败时显示全屏引导："后端服务未启动，请确认 Anchor 服务正在运行" + 重试按钮
  3. 前端检测到连续 3 次网络错误时，显示"后端服务异常"提示 + 重启指引
- **负责人**：前端 + 后端（新增 `/health` endpoint）
- **耗时**：2h（Sprint 2）
- **验收**：关闭 Go 后端后刷新前端，显示明确的错误引导而非白屏

#### 【共识-M6】设计 Token 映射表文档缺失
- **来源**：deepseek-v4-pro 评为高；glm-5.1 在 M2 中提到
- **问题**：当前已使用大量自定义 class（`bg-surface`、`text-text-primary`、`text-accent-green` 等），但无文档记录，新人看不懂
- **实施方案**：产出 `docs/design-tokens.md`：
  ```markdown
  # Design Tokens
  ## 颜色系统
  | Token | 用途 | 示例 |
  |-------|------|------|
  | `bg-surface` | 页面主背景 | Dashboard、Settings |
  | `bg-surface-elevated` | 卡片/面板背景 | Card 组件 |
  | `text-text-primary` | 主文本 | 标题、正文 |
  | `text-text-secondary` | 次要文本 | 描述、标签 |
  | `text-accent-green` | 成功状态 | 扫描完成 Badge |
  ## 使用原则
  - 禁止直接使用 hex 颜色值
  ```
- **负责人**：前端
- **耗时**：0.5h（Sprint 1.3）
- **验收**：文档覆盖所有自定义 class 的语义和用途

---

## 三、差异化建议清单（各模型独特视角，可选采纳）

### kimi-coding 独特建议

| 建议 | 优先级 | 说明 |
|------|--------|------|
| **可靠性阶段优先解决"不可恢复错误"** | 高 | 将请求取消 + AbortController、导出失败恢复提前到 Sprint 1（而非 Sprint 2），数据完整性 bug 比交互一致性更致命 |
| **Tauri "本地优先"特性未被充分利用** | 低 | Target 导入的即时格式校验、Scope 规则本地预览可下沉到前端，减少 HTTP 往返。作为后续优化方向记录 |
| **agent-browser 价值在视觉回归** | 中 | 若 agent-browser 无法 attach Tauri 打包产物，将其重新定位为"视觉回归基线 + 可访问性快照" |
| **每引入一个新依赖需安全必要性评审** | 中 | 作为安全扫描工具，建议将"依赖引入安全评审"显性化 |
| **数据备份与恢复机制** | 高 | SQLite/localStorage 损坏时用户如何恢复？建议至少提供导出/导入全量数据功能 |
| **多窗口状态同步** | 中 | Tauri 支持多窗口，Zustand 状态隔离在各 WebView 中，需防止数据竞争 |

### glm-5.1 独特建议

| 建议 | 优先级 | 说明 |
|------|--------|------|
| **后端 CORS 配置安全隐患** | 高 | 当前 `Access-Control-Allow-Origin: *`，恶意网站可向 localhost:17421 发请求。建议限制为 `http://localhost:1420`、`http://localhost:5173`、`tauri://localhost` |
| **fetchJSON 错误处理静默失败** | 高 | `.catch(console.error)` 不会显示给用户。建议全局错误拦截触发 Toast |
| **后端 `broadcastSSE` 全局广播问题** | 中 | 所有 SSE 客户端收到所有事件。建议客户端注册时携带 project_id，支持按项目过滤 |
| **Go 后端缺少请求级 context 传播** | 低 | 客户端断开后数据库操作仍继续。建议为关键 handler 添加 `r.Context()` 传播 |
| **并发扫描任务资源限制** | 中 | 前端限制同时 running 任务数，后端超限返回 429 |
| **将 component audit 改为 component registry** | 低 | 在代码中创建 `components/index.ts` JSDoc 或 `COMPONENTS.md` 作为活文档 |

### deepseek-v4-pro 独特建议

| 建议 | 优先级 | 说明 |
|------|--------|------|
| **工程健康检查任务** | 高 | Sprint 0 增加 `docs/engineering-health.md`，记录覆盖率、lint 状态、依赖版本、安全配置等基线指标 |
| **SSE + Polling 协作模式明确定义** | 中 | 产出 `useRealtimeData` hook：SSE 优先，断线 3 次后降级到 polling，页面不可见时断开 SSE |
| **SQLite Schema 迁移策略** | 中 | 短期在 `db.go` 中用 `PRAGMA user_version` + `ALTER TABLE IF NOT EXISTS`；长期引入 golang-migrate |
| **Scope 确认页面数据流设计** | 中 | 明确 Dry Run 结果展示、approve 操作逻辑、Target 列表联动更新 |
| **Findings 批量操作事务性** | 低 | 批量修改 100 个 findings 时部分成功/失败的处理、乐观更新策略 |
| **Tauri 打包产物与 Go 后端发布关系** | 中 | 明确 v0.3 阶段桌面应用启动方式（sidecar vs 独立服务） |

---

## 四、冲突消解

### 冲突 1：React Context vs Zustand 管理 project 上下文

| 模型 | 建议 |
|------|------|
| 原始方案 | React Context 传递完整 project 对象 |
| kimi-coding | Zustand 替代 Context，`useParams()` 同步 `currentProjectId` |
| deepseek-v4-pro | Context 只传 `projectId`（最简单），或 Zustand selector（最精细） |

**推荐选择**：采用 **deepseek 方案 A** — ProjectLayout 的 Context 只传 `projectId`（字符串，不变引用，无渲染问题），子页面通过 Zustand store 读取 `currentProjectId` 并自行 fetch 数据。

**理由**：
- 性能最优：projectId 不变，不会触发不必要的重新渲染
- 与 Sprint 2.4 的 Zustand 改造完全兼容
- 实现最简单，不引入额外的 Context provider 嵌套

### 冲突 2：SSE 前端实现路径

| 模型 | 建议 |
|------|------|
| 原始方案 | Sprint 0.7 "验证 SSE 通信路径，1h" |
| glm-5.1 | Sprint 0.7 改为"确认状态 + 设计方案"，Sprint 2 新增 SSE 实现（4-6h） |
| deepseek-v4-pro | Sprint 0.7 产出决策，Sprint 2.7 时间上调至 5-6h |

**推荐选择**：采用 **glm-5.1 建议**。

**理由**：
- 前端确实零 SSE 代码，需要从零实现而非简单验证
- `useSSE` hook + 自动重连 + 心跳检测 + 降级 fallback 是完整功能，4-6h 合理
- 后端 `broadcastSSE` 改造为项目级过滤可并行进行

### 冲突 3：设计 Token 方案程度

| 模型 | 建议 |
|------|------|
| 原始方案 | 补充 `tailwind.config.js` theme.extend.colors |
| glm-5.1 | 配置已齐全，改为审计硬编码颜色并统一替换 |
| deepseek-v4-pro | 确认 Tailwind 版本（v3 vs v4），产出 Token 映射表文档 |

**推荐选择**：采用 **glm-5.1 + deepseek 结合** — Sprint 1.3 改为"审计硬编码颜色 + 产出 `docs/design-tokens.md`"，不修改 `tailwind.config.js`。

**理由**：
- 代码审计已确认配置齐全，无需重复工作
- 硬编码颜色替换有实际价值（如 RunsPage 中的 `bg-yellow-500/15`）
- Token 映射表文档解决新人上手问题

---

## 五、修订后的 Sprint 计划

### Sprint 0：盘点与基线（2-2.5 天）← 原 1.5-2 天

| # | 任务 | 产出 | 耗时 | 变更 |
|---|------|------|------|------|
| 0.1 | 审计后端 API 错误格式（确认 blob 端点边界 + Tauri 文件保存方案） | `docs/api-error-contract.md` | 0.5h | 扩展范围 |
| 0.2 | 跑一遍主流程，记录所有阻断点 | `tasks/desktop-bugs.md` 初版 | 3h | — |
| 0.3 | 确认路由方案 C 的影响范围 + 产出 Legacy 重定向方案 | 更新"影响范围"章节 | 1h | +Legacy 方案 |
| 0.4 | 补 `npm run typecheck` 脚本，验证零错误 | package.json 更新 | 0.5h | -0.5h |
| 0.5 | 编写 10 条主流程验收用例 | `docs/acceptance-criteria.md` | 1.5h | — |
| **0.6** | **产出 Zustand Store 设计草案（slice 化结构 + 数据清理策略）** | store 设计草案 | 1.5h | **+0.5h** |
| **0.7** | **确认 SSE 前端状态 + 产出 SSE 架构决策（HTTP 直连 vs Tauri IPC）** | `docs/sse-status.md` | 1.5h | **+0.5h** |
| 0.8 | 搭建 Go 测试框架 + 生成覆盖率基线报告 | 第一个测试文件 + `docs/coverage-baseline.md` | 2h | +0.5h |
| **0.9** | **Tauri 安全基线配置（CSP + capabilities + lib.rs 迁移）** | `src-tauri/capabilities/default.json` + 安全基线 | 1.5h | **新增** |
| **0.10** | **Vite proxy 配置（API 代理）** | `vite.config.ts` 更新 | 0.5h | **新增** |
| **0.11** | **工程健康检查（产出 `docs/engineering-health.md`）** | 工程健康基线文档 | 0.5h | **新增（deepseek 建议）** |

**Sprint 0 新增交付物**：
- `docs/api-error-contract.md`（含 blob 边界 + Tauri 文件保存约定）
- `tasks/desktop-bugs.md`
- `docs/acceptance-criteria.md`
- `docs/sse-status.md`（含 SSE 架构决策）
- `docs/coverage-baseline.md`
- `docs/engineering-health.md`
- `docs/design-tokens.md`（Token 映射表，Sprint 1.3 前置产出）
- `npm run typecheck` 零错误
- `go test ./...` scope 模块有测试覆盖
- Tauri 安全配置完成（CSP + capabilities）
- Vite proxy 配置完成

**启动门控**（Sprint 0 完成后必须全部通过）：
- [ ] Tauri `npm run tauri dev` 正常启动，无权限报错
- [ ] Vite proxy 配置完成，浏览器 dev 模式可联调
- [ ] SSE 架构决策文档产出（明确 HTTP 直连 vs Tauri IPC）
- [ ] Zustand Store 设计草案评审通过
- [ ] Go 覆盖率基线报告产出
- [ ] `npm run typecheck` 零错误

---

### Sprint 1：设计系统与页面骨架（3-4 天）← 原 2.5-4 天

| # | 任务 | 依赖 | 耗时 | 变更 |
|---|------|------|------|------|
| 1.1 | 实现 ProjectLayout + 嵌套路由（Context 只传 projectId + Zustand 管理） | 决策 1 + 交互规范 | 5h | 方案优化 |
| 1.2 | Navbar 重构（全局项 + 项目项 + Project Switcher） | 1.1 | 3h | — |
| 1.3 | 审计硬编码颜色值，产出 `docs/design-tokens.md` | 无 | 0.5h | -1h |
| 1.4 | 新增 Input、Select、Table 组件（含 a11y 基线） | 1.3 | 4h | +a11y |
| 1.5 | 新增 Modal、ConfirmDialog 组件（焦点陷阱 + aria 标签） | 1.3 | 3h | +a11y |
| 1.6 | 已有组件微调（Badge run status 映射，排查原生 alert/confirm） | 1.3 | 1h | — |
| 1.7 | 统一页面布局结构（标题区、操作区、筛选区、内容区 + 面包屑） | 1.4, 1.5 | 4h | — |
| 1.8 | 去掉原生 alert/confirm，替换为 Toast + Modal | 1.5 | 2h | — |
| **1.9** | **添加 React ErrorBoundary（全局 + ProjectLayout 层级）** | 无 | 1.5h | **新增** |
| 1.10 | Go 测试补充：scope engine、report generator、parser 核心路径 | Sprint 0.8 | 4h | — |
| **1.11** | **Legacy 路由重定向兼容层** | 1.1 | 1h | **新增** |

**Sprint 1 验收**：
- [ ] 所有页面可通过新路由结构访问
- [ ] 旧路由（`/targets` 等）自动重定向到新路由
- [ ] Project Switcher 可切换项目，切换后旧数据不残留
- [ ] ProjectLayout 5 个边界场景处理正确
- [ ] 所有页面使用统一组件和布局
- [ ] 无原生 alert/confirm
- [ ] ErrorBoundary 兜底生效（手动测试）
- [ ] `npm run typecheck` 零错误
- [ ] `go test ./...` scope/report/parser 有测试覆盖

---

### Sprint 2a：API 层 + Store 改造 + SSE（3-4 天）← 原 Sprint 2 前半

| # | 任务 | 依赖 | 耗时 |
|---|------|------|------|
| 2a.1 | api.ts 重构（fetchAPI：超时、AbortController、204、blob、重试、全局错误拦截） | Sprint 0.1 | 4-5h |
| 2a.2 | 请求生命周期管理（AbortController 封装、防重复点击、竞态取消） | 2a.1 | 3h |
| 2a.3 | Zustand store 改造（slice 化 + loading/error 状态 + 数据清理策略 + persist/devtools） | 2a.2, Sprint 0.6 | 5-6h |
| 2a.4 | SSE 前端实现（`useSSE` hook：自动重连、心跳、可见性管理、project 过滤） | Sprint 0.7 | 4-6h |
| 2a.5 | 后端 SSE 改造（`broadcastProjectSSE` + 心跳 + 丢弃日志） | 2a.4 | 2h |
| 2a.6 | `usePolling` hook（Workers 自动刷新 + 退避 + 页面可见性） | 2a.2 | 2h |
| 2a.7 | usePolling + SSE 协作模式（`useRealtimeData`：SSE 优先，断线降级 polling） | 2a.4, 2a.6 | 2h |
| 2a.8 | Go API 集成测试（project CRUD、target import、scan start/cancel） | 无 | 4h |
| 2a.9 | Go workflow 状态机测试（mock Runner，测试状态转换） | Sprint 0.8 | 4-6h |

**Sprint 2a 验收**：
- [ ] api.ts 重构完成，所有调用方适配
- [ ] Zustand store slice 化完成，切换项目数据清空
- [ ] `useSSE` hook 可用，自动重连 + 心跳正常
- [ ] SSE 断线 3 次后自动降级到 polling
- [ ] 后端 `broadcastProjectSSE` 按项目过滤
- [ ] `go test ./...` 通过，workflow 有测试覆盖

---

### Sprint 2b：页面可靠性改造（4-5 天）← 原 Sprint 2 后半

| # | 任务 | 依赖 | 耗时 |
|---|------|------|------|
| 2b.1 | ProjectPage 可靠性（创建/编辑/删除操作、错误处理、confirm） | 2a.3 | 3h |
| 2b.2 | TargetPage 可靠性（导入/批量导入/Dry Run/Scope 确认） | 2a.3 | 4h |
| 2b.3 | RunsPage 可靠性（启动/取消/重试/状态刷新，SSE 优先） | 2a.4, 2a.7 | 3h |
| 2b.4 | WorkersPage 可靠性（状态刷新/注册/离线检测） | 2a.6 | 2h |
| 2b.5 | ReportsPage 可靠性（导出前检查/格式选择/Tauri save dialog + blob 下载） | 2a.1, 2a.3 | 3h |
| 2b.6 | 后端未运行检测 + 全屏引导 | 2a.1 | 2h |
| 2b.7 | CORS 配置修复（限制 origin 为 localhost + tauri://localhost） | 2a.1 | 1h |

**Sprint 2b 验收**：
- [ ] 创建项目 → 导入目标 → Scope 确认 → 启动扫描 可连贯完成
- [ ] 所有页面在网络断开时显示错误提示而非白屏
- [ ] 后端未启动时显示全屏引导
- [ ] Workers/Runs 自动刷新不造成请求风暴
- [ ] 导出报告前有 findings 状态检查，Tauri 下弹出系统保存对话框
- [ ] 重复点击提交按钮不会发多次请求

---

### Sprint 3：漏洞/资产/报告体验（4-5 天）← 原 4-6 天

| # | 任务 | 依赖 | 耗时 | 变更 |
|---|------|------|------|------|
| 3.1 | FindingsPage 筛选（严重级别/状态/工具/关键词） | Sprint 2b | 3h | — |
| 3.2 | FindingsPage 详情（证据展示、代码片段高亮） | Sprint 2b | 3h | — |
| 3.3 | Findings 状态修改（confirmed/false_positive/accepted_risk + 批量操作） | 3.2 | 3h | — |
| 3.4 | AssetPage 筛选与搜索（类型/端口/技术栈） | Sprint 2b | 3h | — |
| 3.5 | AssetPage 详情（端口列表、关联 findings） | 3.4 | 2h | — |
| 3.6 | ReportsPage 报告预览（Markdown 渲染） | Sprint 2b | 3h | — |
| 3.7 | DashboardPage 优化（跨项目统计、最近任务、待处理 findings） | Sprint 2b | 3h | — |
| 3.8 | 全局空状态检查 | 3.1-3.7 | 2h | — |
| 3.9 | 全局 loading 状态检查 | 3.1-3.7 | 2h | — |
| **3.10** | **SQLite Schema 迁移策略（PRAGMA user_version + ALTER TABLE）** | 无 | 1.5h | **新增** |

**Sprint 3 验收**：
- [ ] Findings 可按严重级别/状态筛选，状态修改即时生效
- [ ] Assets 可按类型筛选，详情页展示关联数据
- [ ] Reports 导出前显示 findings 摘要
- [ ] 所有页面空状态有引导而非空白
- [ ] 所有数据加载有 Skeleton 而非闪烁

---

### Sprint 4：测试自动化与回归（3-4 天）← 原 3-5 天

| # | 任务 | 依赖 | 耗时 | 变更 |
|---|------|------|------|------|
| 4.1 | agent-browser E2E 主流程用例（创建→导入→扫描→结果→报告） | Sprint 2b, 3 | 4h | — |
| 4.2 | agent-browser E2E 异常路径用例 | 4.1 | 3h | — |
| 4.3 | Go 测试查漏补缺（边缘 case、覆盖率报告） | Sprint 1.10, 2a.9 | 4-5h | +1-2h |
| 4.4 | 集中修复 P0/P1 bug | tasks/desktop-bugs.md | 4h | — |
| 4.5 | 集中修复 P2 bug | 4.4 | 3h | — |
| 4.6 | P3 视觉细节清理 | 4.5 | 2h | — |
| 4.7 | 编写合并前检查脚本 | 无 | 1h | — |
| 4.8 | 全流程 smoke test（agent-browser 浏览器 + Tauri 打包桌面端） | 4.1-4.6 | 2h | — |
| **4.9** | **验证 agent-browser 对 Tauri 打包产物的可测性** | 4.8 | 1h | **新增（kimi 阻塞项验证）** |

**Sprint 4 验收**：
- [ ] 10 条主流程验收用例全部 pass
- [ ] `go test ./...` 通过，核心模块覆盖率 >80%，workflow >50%，整体 >40%
- [ ] `go vet ./...` 通过
- [ ] `npm run typecheck` 零错误
- [ ] `npm run build` 成功
- [ ] P0/P1 bug 清零
- [ ] agent-browser smoke test 通过
- [ ] Tauri 打包后桌面 smoke test 通过
- [ ] agent-browser 对 Tauri 打包产物可测性验证完成（如不可行，记录降级方案）

---

## 六、风险缓解措施

| 风险 | 缓解措施 | 责任人 |
|------|----------|--------|
| 路由改造影响范围大，破坏已有功能 | Sprint 0 梳理影响范围；Sprint 1 保留 Legacy 重定向兼容层；Sprint 1 验收要求所有旧路由可访问 | 前端 |
| Tauri 安全配置缺失导致打包失败 | Sprint 0.9 强制完成；Sprint 0 启动门控包含 Tauri dev 正常启动 | 前端 |
| SSE 前端从零实现延期 | Sprint 0.7 产出明确架构决策；Sprint 2a 单独排期；若延期，Sprint 2b RunsPage 先用 polling | 前端 |
| Sprint 2 任务密度过高 | 拆分为 Sprint 2a + 2b，各自独立验收；明确缓冲策略 | 项目负责人 |
| Zustand store 改造牵连所有页面 | 先加 loading/error 状态，不改现有 setter 签名；Sprint 0.6 产出设计草案评审通过后再开工 | 前端 |
| agent-browser 对 Tauri 打包产物不可测 | Sprint 4.9 验证；如不可行，降级为"浏览器环境验收 + Tauri 手工 smoke test" | QA |
| 后端 API 错误格式不统一（blob 边界） | Sprint 0.1 确认 blob 端点错误约定；产出 `docs/api-error-contract.md` | 后端 |
| Go 测试覆盖率基线过低 | Sprint 0.8 产出基线报告；调整 Sprint 4 目标为整体 40%+；workflow 测试提前到 Sprint 2a | 后端 |
| 项目切换数据不一致 | Sprint 0.6 设计草案明确数据清理策略；Sprint 2a.3 实现清空逻辑 | 前端 |
| 类型错误治理超预期 | tsconfig 已开 strict，Sprint 0.4 验证零错误（当前已零错误） | 前端 |

---

## 七、验收标准

### 整体验收（v0.3 阶段结束）

1. **主流程连贯性**：新用户不看文档，能完成"创建项目 → 导入目标 → Scope 确认 → 启动扫描 → 查看结果 → 导出报告"完整链路
2. **错误处理**：网络断开、后端未启动、404 项目、空数据等场景均有明确提示，不白屏
3. **路由稳定性**：新路由结构稳定，旧路由自动重定向，ProjectLayout 5 个边界场景处理正确
4. **实时更新**：RunsPage 能实时接收任务进度（SSE 或 polling），WorkersPage 自动刷新不造成请求风暴
5. **测试基线**：`go test ./...` 通过，核心模块覆盖率 >80%，workflow >50%；`npm run typecheck` 零错误
6. **Tauri 打包**：桌面端可正常安装运行，报告导出弹出系统保存对话框，CSP 配置正确
7. **Bug 清零**：P0/P1 bug 清零，P2  bug 清零 80% 以上
8. **文档完备**：`docs/api-error-contract.md`、`docs/acceptance-criteria.md`、`docs/sse-status.md`、`docs/design-tokens.md`、`docs/engineering-health.md` 全部产出

---

## 八、时间估算

| Sprint | 原估算 | 修订后 | 变更原因 |
|--------|--------|--------|----------|
| Sprint 0 | 1.5-2 天 | **2-2.5 天** | 增加 Tauri 安全配置、Vite proxy、工程健康检查、Zustand 设计草案 |
| Sprint 1 | 2.5-4 天 | **3-4 天** | 增加 ErrorBoundary、Legacy 重定向、a11y 基线 |
| Sprint 2a | —（原 Sprint 2: 5-7 天） | **3-4 天** | 拆分自原 Sprint 2，增加 SSE 实现、workflow 测试 |
| Sprint 2b | —（原 Sprint 2: 5-7 天） | **4-5 天** | 拆分自原 Sprint 2，增加后端未运行检测、CORS 修复 |
| Sprint 3 | 4-6 天 | **4-5 天** | 增加 SQLite 迁移策略 |
| Sprint 4 | 3-5 天 | **3-4 天** | 增加 Tauri E2E 可测性验证 |
| **总计** | **16-24.5 天** | **19.5-24.5 天** | 中位 **22 天**，比原估算增加约 2 天 |

---

## 九、各模型关键发现交叉对照表

| 发现项 | kimi-coding | glm-5.1 | deepseek-v4-pro | 优先级 | 状态 |
|--------|:-----------:|:-------:|:---------------:|:------:|:----:|
| Tauri CSP null / 无 capabilities | — | ✅ 阻塞 | ✅ 阻塞 | 阻塞 | 待办 |
| SSE 前端未实现 | — | ✅ 阻塞 | ✅ 阻塞 | 阻塞 | 待办 |
| Vite proxy 缺失 | — | — | ✅ 阻塞 | 阻塞 | 待办 |
| Zustand Store 改造方案笼统 | ✅ 高 | ✅ 高 | ✅ 高 | 高 | 待办 |
| React Context 性能隐患 | ✅ 中 | — | ✅ 高 | 高 | 待办 |
| Sprint 2 密度过高 | ✅ 高 | ✅ 高 | ✅ 高 | 高 | 待办 |
| 缺少 ErrorBoundary | ✅ 中 | ✅ 遗漏 | ✅ 遗漏 | 高 | 待办 |
| 项目切换数据一致性 | — | ✅ 中 | ✅ 中 | 高 | 待办 |
| Go 覆盖率 18.5% / workflow 0% | — | — | ✅ 高 | 高 | 待办 |
| Tailwind 配置与方案不符 | — | ✅ 中 | ✅ 高 | 中 | 待办 |
| api.ts 错误处理增强 | ✅ 高 | ✅ 高 | ✅ 中 | 中 | 待办 |
| 路由 Breaking Change 迁移 | ✅ 高 | — | — | 中 | 待办 |
| 报告导出 Tauri 路径 | — | ✅ 中 | ✅ 中 | 中 | 待办 |
| 后端未运行检测 | — | ✅ 中 | ✅ 中 | 中 | 待办 |
| 设计 Token 映射表 | — | — | ✅ 高 | 中 | 待办 |
| SSE 全局广播过滤 | — | ✅ 中 | — | 中 | 待办 |
| CORS * 安全隐患 | — | ✅ 高 | — | 中 | 待办 |
| fetchJSON 静默失败 | — | ✅ 高 | — | 中 | 待办 |
| 后端 context 传播 | — | ✅ 中 | — | 低 | 待办 |
| 并发扫描资源限制 | — | ✅ 遗漏 | — | 低 | 待办 |
| 数据备份与恢复 | ✅ 遗漏 | — | — | 高 | 可选 |
| Tauri 本地优先利用 | ✅ 独特 | — | — | 低 | 可选 |
| 工程健康检查 | — | — | ✅ 独特 | 高 | 已采纳 |
| SQLite 迁移策略 | — | — | ✅ 中 | 中 | 已采纳 |

---

## 十、下一步行动

1. **审阅本实施方案**，确认优先级、范围和 Sprint 拆分方式
2. **更新原始设计文档**（`docs/v0.3-plan.md`）为 v2.0，合并本方案的修订内容
3. **按启动门控清单**逐项完成 Sprint 0 前置任务
4. **确认 Tauri 安全配置的优先级**（是否作为独立 hotfix 提前完成）
5. **安排 Zustand Store 设计草案评审会**（Sprint 0.6 产出后）
6. **确认 SSE 架构决策**（HTTP 直连 vs Tauri IPC）

---

*本实施方案由 planner agent 基于 kimi-coding、glm-5.1、deepseek-v4-pro 三个模型的独立评审意见汇总生成。mimo-v2.5-pro 模型调用失败，未参与本次评审。*
