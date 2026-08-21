# 阶段四：前端代码健康审计报告

> 审计日期：2026-05-13
> 审计范围：`frontend/src/` 下全部 52 个 TS/TSX 文件
> 工具链状态：`npx tsc --noEmit` 通过
> 审计原则：只审计，不改代码

---

## 执行摘要

| 指标 | 数值 | 健康阈值 | 状态 |
|------|------|----------|------|
| 前端源文件 | 52 | — | — |
| 测试文件 | 4 (8%) | >30% | 严重不足 |
| 超 500 行文件 | 5 | <3 | 超标 |
| api.ts 行数 | 976 | <300 | 严重超标 |
| store.ts 行数 | 196 | <150 | 略超 |
| useState 总调用 | 168 | — | 偏高 |
| useStore 总调用 | 53 | — | 偏高 |
| 空 catch 块 | 35+ | 0 | 严重 |

**最严重的 3 个问题**：
1. **api.ts 976 行上帝文件** — 41 个类型定义 + 58 个 API 方法 + 请求基础设施全部挤在一个文件
2. **35+ 个空 catch 块** — 静默吞掉所有 API 错误，用户看不到失败反馈，调试困难
3. **4 个测试文件覆盖 8%** — 仅测了 APIError、Button、ScanModal ffuf 门控、RunsPage SSE reducer，页面渲染零覆盖

---

## 4.1 api.ts 拆分方案

### 4.1.1 当前状况

`frontend/src/lib/api.ts` (976L) 包含以下职责：

| 职责 | 行数范围 | 内容 |
|------|----------|------|
| HTTP 基础设施 | 1-191 | APIError、request、fetchAPI、fetchBlob、错误处理、全局 handler |
| 类型定义 | 198-975 | 41 个 interface/type 定义 |
| API 方法 | 386-682 | `api` 对象，58 个方法 |
| 常量 | 844-899 | DEFAULT_PIPELINE_CONFIG、DEFAULT_HIGH_RISK_PORTS、TP_PRESET 等 |

### 4.1.2 Domain 分组

按后端 API 路由分组，58 个方法可分为 10 个 domain：

| Domain | 方法数 | 对应后端路由前缀 | 使用文件数 |
|--------|--------|------------------|------------|
| projects | 4 | `/projects` | 4 (ProjectPage, TargetPage, DashboardPage, ProjectLayout) |
| targets | 2 | `/projects/{id}/targets` | 2 (TargetPage, RunsPage) |
| scope | 2 | `/scope-rules` | 1 (TargetPage) |
| scans / runs | 6 | `/scan*`, `/runs`, `/tasks` | 2 (RunsPage, DashboardPage) |
| findings | 5 | `/findings` | 2 (FindingsPage, DashboardPage) |
| assets | 5 | `/projects/{id}/assets`, `/assets` | 2 (AssetPage, DashboardPage) |
| reports | 4 | `/reports`, `/runs/{id}/report` | 2 (ReportsPage, RunsPage) |
| workers | 2 | `/workers` | 1 (WorkersPage) |
| engines | 4 | `/engines` | 2 (EnginesPage, EngineKeysPage) |
| nuclei-custom | 10 | `/nuclei/custom` | 1 (TemplatesPage) |
| dictionaries | 6 | `/dictionaries` | 1 (DictionariesPage) |
| httpx-fingerprints | 6 | `/httpx/fingerprints` | 1 (HttpxFingerprintsPage) |
| finding-templates | 5 | `/finding-templates` | 1 (VulnTemplatesPage) |
| health / tool-templates | 3 | `/health`, `/tool-templates` | 2 (DashboardPage, WorkersPage) |

### 4.1.3 建议拆分结构

```
frontend/src/lib/api/
  client.ts          (~120L)  HTTP 基础设施：APIError, request, fetchAPI, fetchBlob
  types.ts           (~200L)  共享类型：PaginatedResponse, PaginationParams, PAGE_ALL
  projects.ts        (~60L)   Project 类型 + 方法
  targets.ts         (~40L)   Target, ScopeRule, ScopeConfirmationResponse 等
  scans.ts           (~80L)   ScanTask, PipelineRun, PipelineRunStage, Run 等
  findings.ts        (~70L)   Finding, Evidence, Report 等
  assets.ts          (~70L)   Asset, WebEndpoint, Port, Service, ServicePort 等
  workers.ts         (~30L)   WorkerNode, ToolHealth 等
  engines.ts         (~60L)   EngineCredential, SearchResult 等
  nuclei-custom.ts   (~120L)  NucleiCustomSource, NucleiCustomFileEntry 等
  dictionaries.ts    (~70L)   Dictionary 类型 + 方法
  httpx-fp.ts        (~70L)   HttpxFingerprint 类型 + 方法
  finding-templates.ts (~60L) FindingTemplate 类型 + 方法
  pipeline-config.ts (~40L)   PipelineConfig, DEFAULT_PIPELINE_CONFIG, DEFAULT_HIGH_RISK_PORTS
  index.ts           (~30L)   统一 re-export
```

**预估拆分后总行数**：~120 + 200 + 60 + 40 + 80 + 70 + 70 + 30 + 60 + 120 + 70 + 70 + 60 + 40 + 30 = ~1120L（含 index.ts 和少量重复 import）

### 4.1.4 跨 Domain 共享类型

| 共享类型 | 使用 Domain | 建议位置 |
|----------|-------------|----------|
| `PaginatedResponse<T>` | 全部列表接口 | `types.ts` |
| `PaginationParams` / `PAGE_ALL` | 全部列表接口 | `types.ts` |
| `PipelineConfig` | scans, ScanModal | `pipeline-config.ts` |
| `ScanTask` | scans, assets | `scans.ts` (被 assets 引用) |

### 4.1.5 拆分风险与迁移成本

| 风险 | 等级 | 说明 |
|------|------|------|
| Import 路径变更 | Low | 17 个文件需要更新 import，可用 IDE 批量重构 |
| 循环引用 | Low | 当前无循环引用，拆分后需确保 types.ts 不 import 任何 domain 文件 |
| 类型丢失 | Low | 所有类型通过 index.ts re-export，外部使用方式不变 |
| 测试依赖 | Low | 4 个测试文件均 import api.ts，拆分后改为 `import { X } from "../lib/api"` 即可 |

**建议迁移步骤**：
1. 创建 `api/` 目录和 `client.ts`、`types.ts`
2. 逐个 domain 迁移（每次一个 PR），从最少依赖的 domain 开始（workers → engines → ...）
3. 最后迁移 `api` 对象方法，删除原 `api.ts`
4. 保留 `frontend/src/lib/api.ts` 作为 `export * from "./api"` 的兼容层（可选，1 个版本后移除）

---

## 4.2 大页面拆分方案

### 4.2.1 文件规模概览

| 文件 | 行数 | useState 数 | useStore 数 | 风险等级 |
|------|------|-------------|-------------|----------|
| `RunsPage.tsx` | 833 | 14 | 6 | High |
| `TemplatesPage.tsx` | 803 | 20 | 0 | High |
| `TargetPage.tsx` | 715 | 17 | 4 | High |
| `ScanModal.tsx` | 634 | 8 | 0 | Medium |
| `FindingsPage.tsx` | 589 | 11 | 8 | Medium |
| `DictionariesPage.tsx` | 536 | 13 | 0 | Medium |

### 4.2.2 RunsPage.tsx (833L) 拆分方案

**当前职责**：
- 扫描运行列表展示 + 选择
- 流水线阶段详情（SSE + 轮询双模式实时更新）
- 任务列表（按阶段分组）
- 报告生成/下载/删除
- 扫描启动（调用 ScanModal）
- 取消扫描

**建议拆分**：

```
frontend/src/pages/runs/
  RunsPage.tsx              (~250L)  页面壳：加载数据、布局、SSE 连接管理
  RunList.tsx               (~180L)  运行列表卡片
  RunListItem.tsx           (~120L)  单个运行卡片（含状态图标、模式 badge）
  PipelineDetail.tsx        (~200L)  右侧流水线详情面板
  StageTimeline.tsx         (~150L)  阶段时间线
  StageTaskList.tsx         (~100L)  阶段内任务列表
  ReportButton.tsx          (~130L)  报告操作按钮（已内联在 RunsPage 中）
  hooks/
    useRunPolling.ts        (~80L)   轮询逻辑抽离
    useRunSSE.ts            (~60L)   SSE 事件处理抽离
  utils.ts                  (~50L)   mergeStageEvent, formatDurationMs 等
```

**拆分优先级**：P1 — RunsPage 是核心页面，改动频率高

### 4.2.3 TemplatesPage.tsx (803L) 拆分方案

**当前职责**：
- 模板源列表（Table + 展开文件树）
- Git 导入 Modal
- 文件上传 Modal
- 文件编辑器 Modal
- Manifest 展示
- 文件树构建/展开逻辑

**建议拆分**：

```
frontend/src/pages/templates/
  TemplatesPage.tsx         (~180L)  页面壳 + Tab 切换
  SourceList.tsx            (~200L)  模板源表格
  FileTree.tsx              (~150L)  文件树渲染
  SourceActions.tsx         (~80L)   操作按钮组（刷新/验证/启用/删除）
  modals/
    GitImportModal.tsx      (~80L)
    UploadModal.tsx         (~70L)
    FileEditorModal.tsx     (~60L)
  hooks/
    useFileTree.ts          (~60L)   buildFileTree, flattenTree, toggleFolder
  utils.ts                  (~40L)   statusBadge, typeIcon
```

**拆分优先级**：P2 — 功能稳定，改动频率中等

### 4.2.4 TargetPage.tsx (715L) 拆分方案

**当前职责**：
- 项目信息展示（ProjectInfo）
- 目标列表（Table）
- 添加单个目标表单
- 批量文件导入（FileImport + 拖拽）
- Scope 规则管理
- Dry Run 结果展示
- Scope 确认 Dialog

**建议拆分**：

```
frontend/src/pages/target/
  TargetPage.tsx            (~200L)  页面壳 + 布局
  ProjectInfo.tsx           (~50L)   已独立为子组件，可外提
  TargetList.tsx            (~120L)  目标列表
  AddTargetForm.tsx         (~60L)   添加目标表单
  FileImport.tsx            (~120L)  已独立为子组件，可外提
  ScopeRules.tsx            (~100L)  Scope 规则展示
  AddScopeRuleForm.tsx      (~50L)   添加 Scope 规则表单
  DryRunResult.tsx          (~120L)  Dry Run 结果面板
  modals/
    ScopeConfirmModal.tsx   (~40L)
    DryRunConfirmModal.tsx  (~30L)
```

**拆分优先级**：P1 — 核心流程第一步，改动频率高

### 4.2.5 ScanModal.tsx (634L) 拆分方案

**当前职责**：
- 扫描模式选择（外网/内网）
- 端口范围配置（-tp 预设 / -p 自定义）
- Nuclei 扫描策略
- Ffuf 配置（字典选择门控）
- 性能调优参数

**建议拆分**：

```
frontend/src/components/scan-modal/
  ScanModal.tsx             (~180L)  Modal 壳 + 步骤切换
  ModeSelector.tsx          (~100L)  外网/内网模式选择
  PortConfig.tsx            (~150L)  端口范围配置
  NucleiStrategy.tsx        (~80L)   扫描策略选择
  FfufConfig.tsx            (~120L)  Ffuf 配置（含字典门控）
  PerformanceTuning.tsx     (~100L)  性能参数调优
  hooks/
    useStoredConfig.ts      (~50L)   localStorage 读写
  utils.ts                  (~60L)   decodePortRange, updateConfig 等
```

**拆分优先级**：P2 — 功能稳定，但配置项多

### 4.2.6 可复用模式提取

以下模式在多个页面重复，建议提取为共享组件/hook：

| 重复模式 | 出现文件 | 建议提取 |
|----------|----------|----------|
| 数据加载 + loading + error | 全部页面 | `useResource` 已存在，但各页面仍手写大量 boilerplate |
| 表格 + 分页 | FindingsPage, AssetPage, DictionariesPage | `DataTable` 组件 |
| 空状态 | 全部页面 | `EmptyState` 已存在 |
| 确认删除 Dialog | TemplatesPage, DictionariesPage, HttpxFingerprintsPage | `useConfirmDelete` hook |
| 文件上传 + 拖拽 | TargetPage, DictionariesPage, HttpxFingerprintsPage | `FileDropZone` 组件 |
| 表单 Modal（创建/编辑）| 多个页面 | `CrudModal` 模式 |

---

## 4.3 状态管理审查

### 4.3.1 store.ts 当前状态

`frontend/src/lib/store.ts` (196L) 使用 Zustand + persist，包含：

| 状态类别 | 字段 | 是否应全局 | 建议 |
|----------|------|------------|------|
| 项目 | projects, currentProjectId, currentProject | 是 | 保留在 store |
| 目标 | targets, targetsLoading, targetsError | 否 | 下沉到 TargetPage 本地 state |
| 任务 | tasks | 否 | 下沉到 RunsPage 本地 state |
| 资产 | assets, assetsLoading, assetsError | 否 | 下沉到 AssetPage 本地 state |
| Web 端点 | webEndpoints | 否 | 下沉到 AssetPage 本地 state |
| 端口/服务 | ports, services, servicePorts | 否 | 下沉到 AssetPage 本地 state |
| 发现 | findings, findingsLoading, findingsError | 否 | 下沉到 FindingsPage 本地 state |
| 当前发现 | currentFinding | 否 | 下沉到 FindingsPage 本地 state |
| 运行 | runs, runsLoading, runsError | 否 | 下沉到 RunsPage 本地 state |
| Worker | workersLoading, workersError | 否 | 下沉到 WorkersPage 本地 state |
| 报告 | reportsLoading, reportsError | 否 | 下沉到 ReportsPage 本地 state |
| UI | sidebarCollapsed | 是 | 保留在 store |
| 状态历史 | findingStatusHistory | 否 | 下沉到 FindingsPage 本地 state |

### 4.3.2 全局状态膨胀问题

**当前问题**：
- store.ts 管理了 10+ 个 domain 的数据，任何页面切换都会触发不相关的 re-render
- `setCurrentProjectId` 会一次性清空 15 个字段，逻辑耦合严重
- 大量 `loading`/`error` 状态重复模式

**风险等级**：Medium — 当前页面数不多，性能影响有限，但维护成本高

### 4.3.3 缺少的全局状态

| 状态 | 当前位置 | 建议 | 风险 |
|------|----------|------|------|
| Worker 在线状态 | WorkersPage 本地 | 应提升到 store（Dashboard 需要展示 online_workers） | Low |
| SSE 连接状态 | 各页面各自管理 | 应统一（Dashboard、RunsPage 都依赖 SSE） | Medium |
| API Token | `config.ts` | 当前在 localStorage 直接读写，可考虑纳入 store | Low |

### 4.3.4 状态管理规范建议

```
全局状态（Zustand store）：
  - currentProjectId / currentProject
  - projects（列表，用于导航）
  - sidebarCollapsed
  - apiToken（可选）
  - sseConnectionStatus（可选）

页面级状态（useState / useReducer）：
  - 所有列表数据（targets, assets, findings, runs 等）
  - 所有 loading / error 状态
  - 表单状态
  - Modal 开关状态

跨页面共享数据（React Query / SWR 推荐）：
  - 考虑引入 TanStack Query 替代手动 fetch + loading + error 管理
  - 自动缓存、自动重试、自动去重
```

---

## 4.4 前端测试路线图

### 4.4.1 当前测试状况

| 测试文件 | 覆盖内容 | 价值评估 |
|----------|----------|----------|
| `api.test.ts` (31L) | APIError retryable 属性、PAGE_ALL 常量 | Low — 纯工具函数测试 |
| `Button.test.tsx` (28L) | Button 渲染、loading、disabled | Low — UI 组件基础行为 |
| `ScanModal.test.tsx` (64L) | ffuf 字典门控（Fix 3 回归测试） | High — 业务逻辑验证 |
| `RunsPage.test.tsx` (74L) | mergeStageEvent SSE reducer | High — 核心数据流验证 |

**覆盖率**：约 8%（4 个测试文件 / 52 个源文件）
**严重缺口**：15 个页面组件零测试、api.ts 请求逻辑零测试、store.ts 零测试

### 4.4.2 测试优先级矩阵

| 优先级 | 测试目标 | 预估工作量 | 业务价值 |
|--------|----------|------------|----------|
| P0 | api.ts 请求/错误处理单元测试 | 1 工日 | High — 所有 API 调用的基础 |
| P0 | store.ts 状态变更单元测试 | 0.5 工日 | High — 核心状态管理 |
| P1 | TargetPage 渲染 + 添加目标 | 1 工日 | High — 核心流程入口 |
| P1 | RunsPage 渲染 + 列表交互 | 1 工日 | High — 核心流程观察 |
| P1 | ScanModal 完整配置流程 | 1 工日 | High — 扫描启动配置 |
| P2 | FindingsPage 筛选 + 状态变更 | 0.5 工日 | Medium |
| P2 | DashboardPage 统计展示 | 0.5 工日 | Medium |
| P2 | TemplatesPage CRUD 操作 | 0.5 工日 | Medium |
| P3 | 各表单验证逻辑 | 1 工日 | Low — 后端也有校验 |

### 4.4.3 测试基础设施建议

当前使用 Vitest + @testing-library/react，基础设施已就绪。建议补充：

1. **MSW (Mock Service Worker)** — 统一 mock API 响应，替代各测试文件中的 `vi.mock`
2. **测试工具函数** — `renderWithStore()`、`renderWithRouter()` 减少 boilerplate
3. **测试数据工厂** — 用 faker 或手写 factory 生成测试数据

### 4.4.4 前端测试提升计划

**Milestone 1（立即，1 工日）**：
- [ ] 补充 api.ts 请求逻辑测试（request 超时、错误分类、fetchBlob）
- [ ] 补充 store.ts 核心状态变更测试（setCurrentProjectId 清空逻辑）
- [ ] 补充 ScanModal 端口配置测试（decodePortRange 边界值）

**Milestone 2（1 周内，3 工日）**：
- [ ] TargetPage 冒烟测试（渲染、添加目标、文件导入）
- [ ] RunsPage 冒烟测试（渲染、选择运行、阶段展示）
- [ ] FindingsPage 冒烟测试（渲染、筛选、状态变更）
- [ ] 目标：测试文件数达到 12+（覆盖率约 20%）

**Milestone 3（2 周内，3 工日）**：
- [ ] DashboardPage 测试
- [ ] TemplatesPage / DictionariesPage CRUD 测试
- [ ] AssetPage 测试
- [ ] 引入 MSW 统一 mock
- [ ] 目标：测试文件数达到 20+（覆盖率约 30%）

---

## 附录 A：空 catch 块详细清单

以下文件存在空 catch 块（静默吞掉错误，用户无反馈）：

| 文件 | 空 catch 数 | 风险等级 | 修复建议 |
|------|-------------|----------|----------|
| `TemplatesPage.tsx` | 12 | High | 每个 catch 至少 toast 错误信息 |
| `HttpxFingerprintsPage.tsx` | 5 | High | 同上 |
| `DictionariesPage.tsx` | 6 | High | 同上 |
| `VulnTemplatesPage.tsx` | 5 | High | 同上 |
| `FindingsPage.tsx` | 1 | Medium | 同上 |
| `WorkersPage.tsx` | 1 | Medium | 同上 |

**总计：35 个空 catch 块分布在 6 个文件中**

修复模式：
```tsx
// 当前（坏）
} catch (err: any) {}

// 建议（好）
} catch (err: any) {
  toast(err.message || "操作失败", "error");
}
```

---

## 附录 B：前端文件规模完整排名

| 排名 | 文件 | 行数 | 类型 |
|------|------|------|------|
| 1 | `src/lib/api.ts` | 976 | 库 |
| 2 | `src/pages/RunsPage.tsx` | 833 | 页面 |
| 3 | `src/pages/TemplatesPage.tsx` | 803 | 页面 |
| 4 | `src/pages/TargetPage.tsx` | 715 | 页面 |
| 5 | `src/components/ScanModal.tsx` | 634 | 组件 |
| 6 | `src/pages/FindingsPage.tsx` | 589 | 页面 |
| 7 | `src/pages/DictionariesPage.tsx` | 536 | 页面 |
| 8 | `src/pages/AssetPage.tsx` | 507 | 页面 |
| 9 | `src/pages/HttpxFingerprintsPage.tsx` | 480 | 页面 |
| 10 | `src/pages/VulnTemplatesPage.tsx` | 476 | 页面 |
| 11 | `src/pages/ReportsPage.tsx` | 473 | 页面 |

---

## 附录 C：拆分优先级总表

| 优先级 | 项 | 预估工作量 | 影响范围 |
|--------|-----|------------|----------|
| P0 | 修复 35 个空 catch 块 | 0.5 工日 | 6 个文件 |
| P0 | api.ts 拆分为 domain 文件 | 2 工日 | 17 个 import 文件 |
| P1 | RunsPage 拆分 | 1.5 工日 | 1 个文件 |
| P1 | TargetPage 拆分 | 1.5 工日 | 1 个文件 |
| P1 | 补充 api.ts + store.ts 测试 | 1 工日 | 新增 2-3 个测试文件 |
| P2 | ScanModal 拆分 | 1 工日 | 1 个文件 |
| P2 | TemplatesPage 拆分 | 1 工日 | 1 个文件 |
| P2 | 状态管理精简（store.ts 下沉） | 2 工日 | 10 个文件 |
| P2 | 补充核心页面冒烟测试 | 3 工日 | 新增 6-8 个测试文件 |
| P3 | 提取共享组件（DataTable, FileDropZone 等） | 2 工日 | 多个文件 |

---

*报告生成时间：2026-05-13*
*审计人：Claude Opus 4.7*
