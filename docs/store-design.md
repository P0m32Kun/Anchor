# Zustand Store 设计草案

> **文档版本**：v1.0  
> **生成日期**：2026-04-29  
> **对应 Sprint**：0.6  
> **状态**：设计评审中，Sprint 1 启动前确认

---

## 一、当前 Store 问题诊断

基于 Scout 报告和代码审计：

| 问题 | 影响 | 位置 |
|------|------|------|
| `setCurrentProject` 只更新自身，不清空子资源 | 切换项目后旧数据残留 | `store.ts: setCurrentProject` |
| 无 `loading`/`error` 状态 | 各页面重复实现，不一致 | 各 Page 组件 |
| 无 `AbortController` 管理 | 切换页面/项目时产生竞态 | 各 Page `useEffect` |
| `DashboardPage`/`WorkersPage` 直接 `fetch` | 绕过 api.ts 和 store，数据不共享 | `DashboardPage.tsx`, `WorkersPage.tsx` |
| Store 为单一大文件 | 逻辑混杂，难维护 | `store.ts` |

---

## 二、目标 Slice 化结构

```typescript
// store/slices/projectSlice.ts
interface ProjectSlice {
  projects: Project[];
  currentProjectId: string | null;
  loading: boolean;
  error: string | null;
  fetchProjects: (signal?: AbortSignal) => Promise<void>;
  setCurrentProjectId: (id: string | null) => void;
}

// store/slices/targetSlice.ts
interface TargetSlice {
  targets: Target[];
  loading: boolean;
  error: string | null;
  fetchTargets: (projectId: string, signal?: AbortSignal) => Promise<void>;
  clearTargets: () => void;
}

// store/slices/runSlice.ts
interface RunSlice {
  runs: Run[];
  loading: boolean;
  error: string | null;
  fetchRuns: (projectId: string, signal?: AbortSignal) => Promise<void>;
  clearRuns: () => void;
}

// store/slices/assetSlice.ts
interface AssetSlice {
  assets: Asset[];
  webEndpoints: WebEndpoint[];
  loading: boolean;
  error: string | null;
  fetchAssets: (projectId: string, signal?: AbortSignal) => Promise<void>;
  clearAssets: () => void;
}

// store/slices/findingSlice.ts
interface FindingSlice {
  findings: Finding[];
  currentFinding: { finding: Finding; evidence: Evidence[] } | null;
  loading: boolean;
  error: string | null;
  fetchFindings: (projectId: string, signal?: AbortSignal) => Promise<void>;
  clearFindings: () => void;
}

// store/slices/workerSlice.ts
interface WorkerSlice {
  workers: Worker[];
  loading: boolean;
  error: string | null;
  fetchWorkers: (signal?: AbortSignal) => Promise<void>;
}

// store/slices/dashboardSlice.ts
interface DashboardSlice {
  stats: DashboardStats | null;
  loading: boolean;
  error: string | null;
  fetchStats: (signal?: AbortSignal) => Promise<void>;
}
```

### 合并后的 AppState

```typescript
interface AppState extends
  ProjectSlice,
  TargetSlice,
  RunSlice,
  AssetSlice,
  FindingSlice,
  WorkerSlice,
  DashboardSlice {}
```

---

## 三、数据清理策略

`setCurrentProjectId(id)` 时触发 `clearProjectResources()`：

**清空**：
- `targets`
- `runs`
- `assets`
- `webEndpoints`
- `findings`
- `currentFinding`

**保留**：
- `projects`（全局项目列表）
- `currentProjectId`（新值）
- `workers`（全局 Worker 列表，不依赖项目）
- `dashboard.stats`（全局统计）

```typescript
setCurrentProjectId: (id) => {
  set({
    currentProjectId: id,
    targets: [],
    runs: [],
    assets: [],
    webEndpoints: [],
    findings: [],
    currentFinding: null,
    // loading/error 状态也重置
  });
  if (id) {
    get().fetchTargets(id);
    get().fetchRuns(id);
    get().fetchAssets(id);
    get().fetchFindings(id);
  }
},
```

---

## 四、AbortController 集成

每个 async action 内部管理 AbortController：

```typescript
fetchTargets: async (projectId, signal) => {
  // 如果有前一次请求的 abort 方法，调用它
  const prevAbort = get().targetAbort;
  prevAbort?.();

  const controller = new AbortController();
  set({ targetAbort: () => controller.abort(), loading: true, error: null });

  try {
    const data = await api.listTargets(projectId, { signal: controller.signal });
    if (signal?.aborted) return; // 外部也取消了
    set({ targets: data, loading: false });
  } catch (err) {
    if (controller.signal.aborted) return;
    set({ error: err instanceof APIError ? err.message : '加载失败', loading: false });
  }
},
```

**关键规则**：
1. 发起新请求前，取消上一次请求
2. 组件 unmount 时，外部传入的 AbortSignal 触发取消
3. 请求返回后检查 signal.aborted，避免设置过期数据

---

## 五、Persist 策略

只持久化 `currentProjectId`：

```typescript
import { persist } from 'zustand/middleware';

export const useStore = create(
  persist<AppState>(
    (set, get) => ({
      // ... all slices
    }),
    {
      name: 'anchor-store',
      partialize: (state) => ({ currentProjectId: state.currentProjectId }),
    }
  )
);
```

**不持久化**：
- 所有列表数据（targets, assets, findings 等）— 避免数据过期
- loading/error 状态 — 刷新后应重置
- workers/dashboard 数据 — 全局数据，刷新后重新获取

---

## 六、DashboardPage / WorkersPage 迁移方案

### DashboardPage 迁移

**当前**：直接 `fetch()` + `setInterval` 轮询
```typescript
// 当前代码
useEffect(() => {
  const fetchWorkers = () => fetch(`${API_BASE}/workers`).then(...);
  fetchWorkers();
  const interval = setInterval(fetchWorkers, 5000);
  return () => clearInterval(interval);
}, []);
```

**目标**：使用 store 的 dashboard slice
```typescript
// 新代码
const stats = useStore(s => s.stats);
const loading = useStore(s => s.dashboardLoading);

useEffect(() => {
  fetchStats();
}, []);
```

### WorkersPage 迁移

**当前**：直接 `fetch()`
**目标**：使用 store 的 worker slice
```typescript
const workers = useStore(s => s.workers);
const loading = useStore(s => s.workerLoading);

useEffect(() => {
  fetchWorkers();
}, []);
```

**轮询策略**：SSE 接入后，DashboardPage/WorkersPage 优先使用 SSE，SSE 断线时回退到 `usePolling`

---

## 七、实施顺序

| 步骤 | 内容 | Sprint |
|------|------|--------|
| 1 | 创建 slice 文件结构 | Sprint 2.4 |
| 2 | 实现 ProjectSlice（含 persist） | Sprint 2.4 |
| 3 | 实现 TargetSlice + AssetSlice + FindingSlice | Sprint 2.4 |
| 4 | 实现 RunSlice + WorkerSlice + DashboardSlice | Sprint 2.4 |
| 5 | 迁移 DashboardPage / WorkersPage 到 store | Sprint 2.4 |
| 6 | 各页面统一使用 store 的 loading/error 状态 | Sprint 2b |
| 7 | 删除页面级的 `useState` loading/error | Sprint 2b |

---

## 八、风险与缓解

| 风险 | 缓解 |
|------|------|
| Store 改造牵连所有页面 | 先加新 slice，不改现有 state/action 签名；逐步迁移 |
| Persist 导致旧数据格式不兼容 | `partialize` 只持久化简单字段（currentProjectId），不持久化复杂对象 |
| AbortController 竞态处理不当 | 每个 slice 独立管理，单元测试覆盖竞态场景 |
