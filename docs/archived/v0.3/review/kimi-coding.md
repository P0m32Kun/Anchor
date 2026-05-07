# v0.3-plan 技术架构评审报告

> **评审视角**：Tauri + Go 全栈技术架构师
> **评审对象**：`docs/v0.3-plan.md`（v1.1，2026-04-29）
> **评审日期**：2026-04-29
> **状态**：✅ 已完成（2026-05-01）

---

## 一、总体评分与结论

**综合评分：7.5 / 10**

**结论**：
该方案在“产品化打磨”的目标定位上思路清晰，路由重构、API 治理、分层测试等核心决策均经过代码审计支撑，具备较强的可执行性。文档对现有组件成熟度的评估、对 Tailwind Token 的务实定位，以及对 agent-browser 的维护成本考量，体现了“不堆新功能、先稳底盘”的务实工程观。

**但存在三个结构性短板**：
1. **安全维度几乎空白**——作为安全扫描工具的桌面端，本地敏感数据（目标、漏洞、报告）的存储与传输缺少保护设计；
2. **Tauri 桌面端特殊性被低估**——跨平台 WebView 差异、IPC 权限、打包后 E2E 可行性等关键假设未经充分验证；
3. **缺少数据迁移与降级策略**——路由 Breaking Change 和 Store 改造对存量用户的影响未给出过渡方案。

如能在 Sprint 0 补充安全基线评估、验证 agent-browser 对 Tauri 打包产物的真实可测性，并在路由改造中加入 Legacy 重定向，方案可提升至 **8.5 分以上**。

---

## 二、逐条问题与风险

### 🔴 阻塞（Blocker）

**无当前即阻塞项**，但以下两项若在 Sprint 0 验证失败，将升级为阻塞：

#### B-1. agent-browser 对 Tauri 打包产物的 E2E 可测性未验证
- **风险描述**：方案将 agent-browser 作为 Sprint 4 核心验收手段，并声称其“支持 Tauri/Electron 桌面端”。但 Tauri v2 使用系统原生 WebView（macOS WKWebView / Windows WebView2 / Linux WebKitGTK），而非 Chromium 内核。agent-browser（若基于 Playwright/CDP）对 WKWebView 的 attach 需要 Safari Remote Inspector 协议或辅助功能（Accessibility）API，配置复杂度与 Electron 完全不同。若 Sprint 4 才发现无法自动化，验收将完全依赖手动，24 人天的投入缺乏自动化回归保护。
- **缓解**：Sprint 0 增加 1h 任务：用 agent-browser 对当前 Tauri dev 模式及打包后的 `.app`/`.exe` 跑一次最小可复现测试（打开窗口、截图、断言标题）。若失败，立即将 Sprint 4 的 E2E 降级为“浏览器环境验收 + Tauri 手工 smoke test”。

---

### 🟠 高（High）

#### H-1. 缺少本地敏感数据的保护设计（安全）
- **风险描述**：项目配置、扫描目标（可能包含内网资产）、漏洞 findings、报告均属于高敏感数据。方案中：
  - `currentProject` 持久化到 `localStorage`（WebView 级，明文存储于磁盘）；
  - 报告导出使用 Tauri `fs` plugin，但未提及导出路径的权限最小化；
  - 未评估 Go 后端本地数据库（如 SQLite）是否需要加密 at rest。
- **影响**：若工具部署在共享工作站上，其他用户可直接读取 WebView 的 localStorage 或 SQLite 文件获取扫描结果。
- **建议**：
  - 至少引入一次安全基线评估（Sprint 0，0.5h），明确数据分级；
  - 敏感配置（如 API Token、目标列表）避免存 `localStorage`，改用 Tauri `tauri-plugin-store`（基于 OS keychain/secure storage 的后端）或 Go 端加密存储；
  - 报告导出文件名做 `sanitize-filename` 处理，防止路径遍历。

#### H-2. 路由 Breaking Change 缺少迁移与降级策略（架构）
- **风险描述**：方案明确“删除当前 `/projects/:id/assets` 等 legacy 路由”，但未给出：
  - 已收藏书签、历史记录、localStorage 中旧路由的处理；
  - 用户从旧版本升级后首次打开应用的路由重定向；
  - 如果 Sprint 1 路由改造引入 regression，如何快速回滚（feature flag 或双轨运行）。
- **建议**：
  - 在 `App.tsx` 中保留一个 Sprint 的 Legacy 重定向层：
    ```tsx
    // App.tsx — 兼容层，Sprint 2 后移除
    <Route path="/targets" element={<Navigate to={`/projects/${lastProjectId}/targets`} replace />} />
    ```
  - 路由改造前，在 `localStorage` 中记录用户最后访问的项目 ID，用于重定向兜底。

#### H-3. Go 后端输入边界缺少安全视角（安全）
- **风险描述**：方案要求对 Scope Engine（CIDR、通配符）、Parser（目标导入）、Report Generator 做单元测试，但**缺少对输入攻击面的设计评审**：
  - 批量导入超大目标列表（10万+行）可能导致内存 DoS；
  - 通配符/正则解析存在 ReDoS 风险；
  - 报告生成若使用模板引擎（如 Go 的 `text/template`），存在 SSTI（Server-Side Template Injection）风险；
  - 文件上传导入未提及 MIME 校验和大小限制。
- **建议**：
  - Sprint 0 增加“输入安全边界”子任务（1h），为 Parser 和 Scope Engine 增加最大输入长度、最大递归深度限制；
  - 报告生成若用模板，务必使用 `html/template` 而非 `text/template`，且对模板变量做转义；
  - 在 `docs/api-error-contract.md` 中补充 413 Payload Too Large 的处理约定。

#### H-4. Sprint 2 任务密度过高，缺少缓冲（可维护性）
- **风险描述**：Sprint 2 承担 11 个任务（33h 估算，按 8h/天为 4.1 天），涵盖 API 治理、Store 改造、5 个页面的可靠性增强、Go 集成测试。此估算为纯编码时间，未包含联调、Bug 修复、Code Review、知识同步。一旦 Sprint 0 的 bug 盘点发现 P0 级问题，Sprint 2 将被迫挤压。
- **建议**：
  - 将 Go API 集成测试（2.10）和 workflow 状态机测试（2.11）移至 Sprint 3 或并行由另一位后端开发负责；
  - 或延长 Sprint 2 至 8-9 天，并明确“若 Sprint 0 发现 >3 个 P0 bug，则 Sprint 2 仅做修复，不做新功能”。

---

### 🟡 中（Medium）

#### M-1. SSE 技术路径未明确（技术可行性）
- **风险描述**：方案在多处提到“SSE 优先”，但缺少技术决策：前端 SSE 是通过浏览器 `EventSource` 直连 Go HTTP endpoint，还是通过 Tauri Rust 层的 Event/Channel 转发？
  - 若直连 Go：`EventSource` 不支持自定义 header（如 Auth），且 Tauri WebView 中的 CORS 行为与浏览器不同；
  - 若走 Tauri IPC：需要 Rust 层作为 bridge，增加复杂度，但更可靠。
- **建议**：Sprint 0.7 的验证任务应产出明确决策，并在 `docs/sse-status.md` 中给出架构图（前端 ← Tauri IPC ← Go / 前端 ← HTTP ← Go）。

#### M-2. ProjectLayout 的 React Context 不是最优解（架构合理性）
- **风险描述**：方案建议用 React Context 传递 project 信息。Context 在 ProjectLayout 层级会导致：
  - 任何子组件消费 Context 都会在其值变化时重新渲染（即使只用到部分字段）；
  - 与 Zustand 并存造成状态管理分裂（全局用 Zustand，项目上下文用 Context）。
- **建议**：项目上下文直接写入 Zustand store，通过 `useParams()` 在 ProjectLayout 的 `useEffect` 中同步 `currentProjectId`，子页面统一从 store 读取。这样_abort、loading、error 状态也能统一管理：
  ```ts
  // store/projectStore.ts
  interface ProjectState {
    currentProjectId: string | null;
    projectMap: Record<string, Project>;
    loading: boolean;
    error: string | null;
    setCurrentProjectId: (id: string) => void;
    // 自动触发 fetch，内部处理 AbortController
  }
  ```

#### M-3. 异常场景覆盖不完整（方案完整性）
- **风险描述**：ProjectLayout 交互规范覆盖了 404、删除、过期，但遗漏：
  - **项目加载超时**（后端阻塞或网络慢）：Skeleton 显示多久后应转为错误？
  - **后端 500 错误**：ProjectLayout 是否应阻止子页面渲染？
  - **权限降级**（用户失去项目访问权）：401/403 的处理；
  - **子页面在 ProjectLayout 加载失败时的行为**：Outlet 是否应无条件渲染？
- **建议**：补充“加载超时”和“后端 500”两条边界场景，建议 ProjectLayout 在 error 状态下渲染全页 ErrorBoundary，不渲染 Outlet。

#### M-4. 报告导出的 Tauri 权限与流式生成缺失（技术可行性）
- **风险描述**：报告导出涉及 Tauri `fs:write` 权限，但方案未提及 `capabilities/default.json` 的权限配置。此外，若 findings 数量巨大（如 10,000+），一次性生成报告到内存再写入可能导致内存溢出。
- **建议**：
  - 在 Sprint 1 确认 `src-tauri/capabilities/default.json` 已授予 `fs:default` 或最小化路径权限；
  - Go 端报告生成采用流式写入（`io.Writer` 接口），前端通过 Tauri 的 `writeFile` 分块写入，或让 Go 直接写入用户选定路径后返回路径。

#### M-5. 缺少桌面端系统级集成设计（方案完整性）
- **风险描述**：作为桌面应用，方案未涉及系统级集成：
  - 全局快捷键（如 Cmd+Shift+N 新建项目）；
  - 系统托盘/菜单栏状态（扫描中显示托盘图标进度）；
  - 文件拖放导入目标列表；
  - 外部链接打开方式（Tauri `shell:open` 权限）。
- **建议**：虽不在 v0.3 范围，但应在文档“不在本阶段范围”中明确列出，并评估哪些交互必须依赖系统级能力（如拖放导入可显著提升 Target 导入体验）。

#### M-6. Workers 离线检测机制未定义（架构合理性）
- **风险描述**：文档仅提及“WorkersPage 可靠性（状态刷新/注册/离线检测）”，但没有定义 Worker 模型：
  - Worker 是独立进程、Goroutine、还是外部二进制？
  - 通信协议（HTTP、gRPC、stdin/stdout）？
  - “离线”判定标准是心跳超时（如 3 次心跳丢失）还是 TCP 连接断开？
- **建议**：Sprint 0 增加 0.5h 任务产出 `docs/worker-architecture.md`，明确心跳协议和超时阈值，否则 Sprint 2.8 的“离线检测”无法实施。

#### M-7. 设计 Token 方案的版本兼容性风险（可维护性）
- **风险描述**：方案基于 `tailwind.config.js` 的 `theme.extend.colors`，这是 Tailwind v3 模式。若项目已升级或计划升级到 Tailwind v4，v4 采用 CSS-first 配置（`@theme`），`tailwind.config.js` 将被废弃。
- **建议**：Sprint 1.3 开始前，确认当前 Tailwind 版本。若已是 v4，应直接采用 `@theme` 注册颜色 token；若仍为 v3，应在文档中标注“v4 迁移时需重构 Token 层”。

#### M-8. 缺少前端错误边界与崩溃收集（可维护性）
- **风险描述**：方案提到“全局空状态/loading 检查”，但未涉及 React Error Boundary。任何未捕获的渲染错误（如 `project` 为 undefined 时访问 `project.name`）会导致整页白屏，这是桌面应用最糟糕的体验。
- **建议**：
  - 在 ProjectLayout 和 Dashboard 层级包裹 `<ErrorBoundary>`；
  - 集成 Tauri `log` plugin，将前端错误日志写入本地文件，便于用户报告问题。

#### M-9. Zustand Store 竞态条件未充分讨论（架构合理性）
- **风险描述**：Sprint 2.4 要求“Zustand store 改造（加载状态、错误状态、请求取消）”，但未给出模式。典型的竞态场景：用户在 RunsPage 快速切换筛选条件，触发多次 `fetchRuns()`，若前一次请求后返回，会覆盖新请求的结果。
- **建议**：在 store 的异步 action 中采用“stamp”或 AbortController 模式：
  ```ts
  const fetchRuns = async (projectId: string, signal: AbortSignal) => {
    set({ loading: true, error: null });
    try {
      const data = await api.getRuns(projectId, { signal });
      if (signal.aborted) return; // 丢弃过期请求
      set({ runs: data, loading: false });
    } catch (e) {
      if (signal.aborted) return;
      set({ error: e.message, loading: false });
    }
  };
  ```

#### M-10. 状态机定义缺失（方案完整性）
- **风险描述**：Runs、Workers、Findings 均有状态流转（如 running → completed → failed），但文档中没有状态转换图或非法转换的防护。前后端状态机不一致是常见 bug 源（如前端允许“重试”时后端已处于不可重试状态）。
- **建议**：在 Sprint 2 中为 Workflow 和 Runs 产出最小状态机文档（如 Mermaid 图），并在 Go 层用显式状态转换函数（而非直接赋值）实现。

---

### 🟢 低（Low）

#### L-1. ESLint 完全不引入可能导致代码风格漂移
- **风险描述**：方案以“tsc strict 已够”为由暂不引入 ESLint，但缺少替代方案（如 Biome、OXC、或严格的 Prettier 配置）。多人协作时，风格不一致会累积技术债。
- **建议**：若暂不引入 ESLint，至少配置 `.prettierrc` 强制基础格式，并在 Sprint 4 重新评估引入 `typescript-eslint`（仅开启 recommended 规则，配置成本 <30min）。

#### L-2. 组件封装粒度可能过度（可维护性）
- **风险描述**：新增 Input、Select、Table、Modal、ConfirmDialog 5 个基础组件。若项目中已有直接使用原生 `<input>` + Tailwind class 的表单，全部迁移到新组件是一次性大改。且自研 Select/Table 的功能（搜索、多选、横向滚动、排序）很容易越做越重。
- **建议**：评估是否直接引入 Headless UI（Radix UI）或 shadcn/ui 的底层组件（无样式逻辑，仅交互和可访问性），再套 Tailwind 皮肤。这样比自己实现 Select/Modal 的 WAI-ARIA 行为更稳健。

#### L-3. 性能基线缺失（性能）
- **风险描述**：作为可靠性阶段，未建立性能基线：首屏加载时间、大数据量表格渲染帧率、内存占用（WebView  notoriously 容易内存泄漏）。
- **建议**：Sprint 4 增加一条“性能快照”任务：用 Tauri DevTools Performance 面板记录 Dashboard 和 FindingsPage 在 1000/10000 条数据下的渲染时间，作为后续优化的 benchmark。

#### L-4. Bug 分级中的“越权”归类为 P0 但未解释（安全性）
- **风险描述**：Bug 分级将“越权”列为 P0，但当前是单用户桌面应用，越权模型未定义。是防止其他系统用户读取数据？还是防止 Worker 进程越权？
- **建议**：在 Bug 文档中补充一句：“P0 越权指 Scope 规则误判导致扫描非授权目标，或本地数据被其他系统用户访问。”

#### L-5. `noUnusedLocals` 开启可能导致开发体验下降
- **风险描述**：`tsconfig.json` 开启 `noUnusedLocals` 和 `noUnusedParameters`，在快速迭代时（如临时注释一段代码）会频繁触发编译错误，打断开发流。
- **建议**：开发模式（`tsconfig.app.json`）可关闭 `noUnusedLocals`，仅在生产构建/CI 的 `typecheck` 脚本中开启。或配置 Vite 插件延迟报错。

---

## 三、遗漏的关键点

以下关键点在文档中完全未被提及，建议在 Sprint 0 或后续补充：

| 遗漏项 | 重要性 | 说明 |
|--------|--------|------|
| **数据备份与恢复** | 高 | 若 SQLite 或 localStorage 损坏（崩溃、断电），用户如何恢复项目？至少应有导出/导入全量数据的机制。 |
| **多窗口状态同步** | 中 | Tauri 支持多窗口。若用户打开两个窗口操作同一项目，Zustand 状态隔离在各自 WebView 中，如何防止数据竞争？ |
| **自动更新（Updater）** | 中 | Tauri 内置 updater。v0.3 作为“可信赖”阶段，应考虑更新机制，至少预留 `tauri.conf.json` 的 updater 配置。 |
| **国际化（i18n）架构预留** | 低 | 虽不在本阶段，但文案硬编码越深，后续 i18n 成本越高。建议用简单常量对象（`const t = { createProject: '创建项目' }`）而非裸字符串，作为零成本预留。 |
| **黑暗模式预留** | 低 | 当前 Tailwind class 硬编码颜色（如 `text-accent-green`）不支持黑暗模式。若未来需要，需全局重构。建议在 Sprint 1 将颜色类抽象为语义名（`text-success`），即使当前只映射到一个色值。 |
| **Go 依赖漏洞扫描** | 中 | 安全工具的供应链安全至关重要。建议在 CI 或 Sprint 4 引入 `govulncheck` 扫描 Go 依赖。 |

---

## 四、具体优化建议（附代码/配置示例）

### 4.1 路由兼容层（缓解 H-2）

在 `App.tsx` 中保留一个 Sprint 周期的 legacy 重定向：

```tsx
import { Navigate, useLocation } from 'react-router-dom';

function LegacyRouteGuard() {
  const location = useLocation();
  const lastProjectId = localStorage.getItem('lastProjectId');
  
  // 记录旧路由访问到日志，用于评估何时可安全移除兼容层
  useEffect(() => {
    console.warn(`[Deprecation] Accessed legacy route: ${location.pathname}`);
  }, [location]);

  if (!lastProjectId) {
    return <Navigate to="/projects" replace />;
  }

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

// 在路由表末尾添加
<Route path="/targets" element={<LegacyRouteGuard />} />
<Route path="/assets" element={<LegacyRouteGuard />} />
// ... 其他 legacy 路由
```

### 4.2 Zustand + AbortController 的竞态安全模式（缓解 M-9）

```ts
// store/utils.ts
export function withAbort<T>(
  set: (fn: (state: any) => any) => void,
  get: () => any,
  key: string // store 中保存 abortController 的 key
) {
  return async (fetcher: (signal: AbortSignal) => Promise<T>, onSuccess: (data: T) => void) => {
    // 取消前一次请求
    const prev = get()[key] as AbortController | undefined;
    prev?.abort();

    const ctrl = new AbortController();
    set((state) => ({ ...state, [key]: ctrl, loading: true, error: null }));

    try {
      const data = await fetcher(ctrl.signal);
      if (ctrl.signal.aborted) return;
      set((state) => ({ ...state, [key]: undefined, ...onSuccess(data), loading: false }));
    } catch (err: any) {
      if (ctrl.signal.aborted) return;
      set((state) => ({ ...state, [key]: undefined, error: err.message, loading: false }));
    }
  };
}
```

### 4.3 Tauri 权限最小化配置（缓解 H-1, M-4）

```json
// src-tauri/capabilities/default.json
{
  "$schema": "../gen/schemas/desktop-schema.json",
  "identifier": "default",
  "windows": ["main"],
  "permissions": [
    "core:default",
    "fs:allow-write-file",
    {
      "identifier": "fs:scope",
      "allow": [{ "path": "$DOWNLOAD/*" }, { "path": "$DOCUMENT/{app-name}/*" }]
    },
    "dialog:default",
    "shell:default"
  ]
}
```

> 注意：避免使用 `fs:allow-read` 读取任意路径，导出时通过 Tauri `dialog:save` 让用户选择路径，再写入。

### 4.4 输入安全边界（缓解 H-3）

```go
// internal/parser/parser.go
const (
    MaxTargetLines    = 100_000
    MaxTargetLineLen  = 2_048
    MaxTotalInputSize = 10 * 1024 * 1024 // 10 MB
)

func ParseTargets(reader io.Reader) ([]Target, error) {
    lr := io.LimitReader(reader, MaxTotalInputSize)
    scanner := bufio.NewScanner(lr)
    scanner.Buffer(make([]byte, 1024), MaxTargetLineLen)

    var targets []Target
    for scanner.Scan() {
        if len(targets) >= MaxTargetLines {
            return nil, fmt.Errorf("target count exceeds %d", MaxTargetLines)
        }
        // parse...
    }
    return targets, nil
}
```

### 4.5 React Error Boundary 兜底（缓解 M-8）

```tsx
// components/ErrorBoundary.tsx
import { Component, ReactNode } from 'react';

interface Props { fallback?: ReactNode; children: ReactNode; }
interface State { hasError: boolean; error?: Error; }

export class ErrorBoundary extends Component<Props, State> {
  state: State = { hasError: false };
  
  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, info: React.ErrorInfo) {
    // 可接入 Tauri log plugin
    console.error('[ErrorBoundary]', error, info.componentStack);
  }

  render() {
    if (this.state.hasError) {
      return this.props.fallback || (
        <div className="p-8 text-center">
          <h2 className="text-lg font-semibold text-destructive">页面出现错误</h2>
          <p className="text-muted-foreground mt-2">请尝试刷新或返回首页</p>
          <button onClick={() => window.location.href = '/'} className="mt-4 btn-primary">
            返回首页
          </button>
        </div>
      );
    }
    return this.props.children;
  }
}
```

在 `ProjectLayout` 和 `Dashboard` 的根节点包裹 `<ErrorBoundary>`。

---

## 五、独特见解 / 差异化视角

### 5.1 “可靠性阶段”应优先解决“不可恢复错误”，而非“交互一致性”

文档将 UI 统一（设计 Token、组件补齐）放在 Sprint 1，核心流程可靠性放在 Sprint 2。但从桌面应用的视角，**用户最不可原谅的体验是丢数据和无法退出**，而非按钮圆角不一致。建议将以下两项提前到 Sprint 1：
- 请求取消 + AbortController（防止页面切换后回调 setState 导致崩溃）；
- 导出/写入操作的失败恢复（报告写到一半失败，是否留下半份文件？）。

交互一致性可以边做边改，但数据完整性的 bug 一旦释放，用户信任难以挽回。

### 5.2 Tauri 的“本地优先”特性未被充分利用

Tauri 相比 Electron 的最大优势是**极低的资源占用和原生的系统集成能力**。但方案中的 Tauri 仅被当作“打包浏览器”使用，所有业务逻辑仍放在 Go HTTP API 中。实际上，部分非敏感逻辑（如目标格式校验、Scope 规则的本地预览、报告模板的客户端渲染）可以下沉到 Tauri 的 Rust 层或前端，减少 IPC/HTTP 往返延迟。

**具体机会点**：
- Target 导入的即时格式校验（纯前端即可，无需发到 Go）；
- Scope 规则的本地预览（Dry Run 结果如果能在前端基于已下载的 Scope 规则本地计算，响应更快）。

这并非 v0.3 必做项，但应在文档中标注为“后续架构优化方向”。

### 5.3 agent-browser 的真正价值不在 E2E，而在“视觉回归”

如果 agent-browser 最终只能跑在浏览器环境（无法 attach Tauri 打包产物），它的价值不应被否定——对于 Tailwind + React 应用，**跨浏览器/跨操作系统 的像素级视觉回归**比功能回归更脆弱。建议将 agent-browser 的用途重新定位为：
- **视觉回归基线**：每个 Sprint 对关键页面截图，与基线对比，捕捉 WebView 渲染差异；
- **可访问性快照**：确保新增组件没有破坏 ARIA 结构。
这比强行用它做端到端功能测试更符合桌面工具的实际需求。

### 5.4 “不堆新功能”是最好的安全策略

作为安全扫描工具，每行新增代码都是潜在的攻击面。v0.3 明确排除“新增扫描工具集成”是明智的。但建议将这一原则显性化：**在可靠性阶段，每引入一个新依赖（包括 Tauri plugin、npm package），都需经过安全必要性评审**。例如，若为了 Select 组件引入 `react-select`，其额外的依赖树会增加供应链攻击面；自研或基于 Radix UI 的 headless 模式更安全。

---

## 六、评审行动清单（供参考）

| 优先级 | 行动项 | 建议执行 Sprint |
|--------|--------|-----------------|
| 高 | 验证 agent-browser 对 Tauri 打包产物的可测性 | Sprint 0 |
| 高 | 补充本地敏感数据保护评估（localStorage/SQLite/fs） | Sprint 0 |
| 高 | 增加路由 Legacy 重定向兼容层 | Sprint 1 |
| 高 | 为 Parser/Scope Engine 增加输入安全边界（大小限制、ReDoS 防护） | Sprint 0-1 |
| 高 | 调整 Sprint 2 密度：将 Go 集成测试移至 Sprint 3 或并行化 | Sprint 0 规划 |
| 中 | 明确 SSE 技术路径（HTTP 直连 vs Tauri IPC） | Sprint 0 |
| 中 | 用 Zustand 替代 React Context 管理 project 上下文 | Sprint 1 |
| 中 | 补充 ProjectLayout 超时/500 边界场景 | Sprint 1 |
| 中 | 配置 Tauri 最小化 fs 权限 | Sprint 1 |
| 中 | 产出 Worker 心跳/离线检测协议文档 | Sprint 0 |
| 中 | 引入 React Error Boundary + Tauri log plugin | Sprint 2 |
| 低 | 确认 Tailwind 版本（v3 vs v4），调整 Token 方案 | Sprint 1 前 |
| 低 | 建立性能基线（大数据量渲染、内存） | Sprint 4 |
| 低 | 预留 i18n/黑暗模式架构（语义化颜色命名） | Sprint 1 |

---

*本评审由技术架构师视角独立完成，所有代码示例可直接用于项目参考。*
