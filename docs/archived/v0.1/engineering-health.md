---
status: archived
source_of_truth: false
owner: kun
last_updated: 2026-05-07
scope: engineering-health
archive_reason: Sprint 0 基线数据，已过时
---

# 工程健康检查报告（Sprint 0.11）

> 生成日期：2026-04-29  
> 检查范围：Go 后端 + TypeScript 前端 + Tauri 桌面端  
> 基线版本：Sprint 0 代码库  
> ⚠️ 此文档已归档，数据基于 Sprint 0 代码库，仅供参考

---

## 1. 代码健康

### 1.1 文件规模

| 语言 | 文件类型 | 数量 | 备注 |
|------|----------|------|------|
| Go | `.go`（非测试） | 33 | 含 `main.go` + 14 个 internal 包 |
| Go | `_test.go` | 9 | 覆盖 5 个包 |
| TypeScript | `.ts` / `.tsx` | 23 | 8 页面 + 7 组件 + 3 lib + 2 入口 + 1 配置 + 2 其他 |

### 1.2 静态检查

| 检查项 | 命令 | 结果 | 状态 |
|--------|------|------|:----:|
| Go Vet | `go vet ./...` | 无问题 | ✅ |
| TypeScript 类型检查 | `tsc --noEmit` | **无法执行** — `node_modules` 未安装 | ⚠️ |

> **说明**：前端 `package.json` 已配置 `"typecheck": "tsc --noEmit"` 脚本，但当前工作树缺少 `node_modules`。在已安装依赖的环境中，历史构建通过 `tsc && vite build`，说明类型检查在 CI/构建流程中是有效的。

---

## 2. 测试覆盖

> 数据来源：`docs/coverage-baseline.md`（Sprint 0.8）

### 2.1 总体指标

| 指标 | 值 |
|------|-----|
| 总包数 | 15 |
| 有测试的包 | 5 |
| 零覆盖包 | 10 |
| **总语句覆盖率** | **18.5%** |
| 测试用例总数 | 161 |
| 测试状态 | ✅ 全部通过 |

### 2.2 各模块覆盖详情

| 包路径 | 语句覆盖率 | 状态 |
|--------|-----------|:----:|
| `internal/nuclei` | 96.8% | 🟢 优秀 |
| `internal/parser` | 90.0% | 🟢 优秀 |
| `internal/report` | 78.3% | 🟢 良好 |
| `internal/scope` | 68.4% | 🟢 良好 |
| `internal/asset` | 47.1% | 🟡 中等 |
| `internal/api` | 0.0% | 🔴 零覆盖 |
| `internal/db` | 0.0% | 🔴 零覆盖 |
| `internal/errors` | 0.0% | 🔴 零覆盖 |
| `internal/health` | 0.0% | 🔴 零覆盖 |
| `internal/models` | 0.0% | 🔴 零覆盖 |
| `internal/scoring` | 0.0% | 🔴 零覆盖 |
| `internal/util` | 0.0% | 🔴 零覆盖 |
| `internal/worker` | 0.0% | 🔴 零覆盖 |
| `internal/workflow` | 0.0% | 🔴 零覆盖 |
| `(root)` | 0.0% | 🔴 零覆盖 |

### 2.3 风险评级

- 🔴 **高**：`internal/api`（~50 个 handler）、`internal/workflow`（扫描编排）、`internal/db`（所有查询）
- 🟠 **中高**：`internal/worker`（分布式任务逻辑）
- 🟡 **中**：`internal/scoring`（漏洞评分计算）
- 🟢 **低**：`internal/errors`、`internal/util`、`internal/models`、`internal/health`（结构简单或纯类型定义）

---

## 3. 安全配置

### 3.1 Tauri CSP（内容安全策略）

**配置位置**：`src-tauri/tauri.conf.json`

```json
"csp": "default-src 'self'; connect-src 'self' http://localhost:17421; img-src 'self' data: blob:; style-src 'self' 'unsafe-inline'"
```

| 检查项 | 结果 | 状态 |
|--------|------|:----:|
| `default-src 'self'` | 限制默认资源加载为同源 | ✅ |
| `connect-src 'self' http://localhost:17421` | 仅允许连接本地后端 | ✅ |
| `img-src 'self' data: blob:` | 图片限制在同源 + data/blob | ✅ |
| `style-src 'self' 'unsafe-inline'` | 允许内联样式（Tailwind 需要） | ⚠️ 接受 |
| 无 `script-src 'unsafe-eval'` | 未放宽脚本执行限制 | ✅ |

> **结论**：CSP 配置合理，符合桌面端 + 本地后端的部署模式。`'unsafe-inline'` 为 Tailwind CSS 必需。

### 3.2 Tauri Capabilities（权限能力）

**配置位置**：`src-tauri/capabilities/default.json`

```json
{
  "identifier": "default",
  "windows": ["main"],
  "permissions": ["core:default", "shell:allow-open", "dialog:default"]
}
```

| 权限 | 用途 | 状态 |
|------|------|:----:|
| `core:default` | Tauri 核心默认能力 | ✅ |
| `shell:allow-open` | 允许打开外部链接/程序 | ✅ 已最小化 |
| `dialog:default` | 文件对话框 | ✅ |
| 无 `fs:write` / `fs:read` 全局权限 | 未授予文件系统全局写权限 | ✅ |

> **结论**：能力清单遵循最小权限原则，未授予不必要的文件系统或网络权限。

### 3.3 CORS 配置

**配置位置**：`internal/api/handlers.go:195`

```go
func CORSMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
        // ...
    })
}
```

| 检查项 | 当前值 | 建议 | 状态 |
|--------|--------|------|:----:|
| `Access-Control-Allow-Origin` | `*`（通配符） | 生产环境应限制为 `http://localhost:17421` 或 Tauri 的 `tauri://localhost` | ⚠️ |
| `Access-Control-Allow-Methods` | `GET, POST, PATCH, OPTIONS` | 若需删除操作，应补充 `DELETE` | 🟡 |
| `Access-Control-Allow-Headers` | `Content-Type` | 若使用自定义头（如 `X-Request-ID`），需补充 | 🟡 |

> **结论**：当前 CORS 配置适合开发调试，但生产环境存在过度放宽风险。考虑到这是桌面端应用（非公网暴露），风险可控，但仍建议收紧。

---

## 4. 依赖状态

### 4.1 Go 依赖

| 依赖 | 版本 | 用途 |
|------|------|------|
| Go | 1.26 | 语言版本 |
| `github.com/mattn/go-sqlite3` | v1.14.22 | SQLite 驱动（CGO） |

> **观察**：Go 端依赖极度精简，仅有一个第三方库。无 HTTP 框架（使用标准库 `net/http`），无 ORM（手写 SQL）。

### 4.2 npm 依赖（关键）

| 依赖 | 版本 | 用途 |
|------|------|------|
| `react` | ^18.2.0 | UI 框架 |
| `react-dom` | ^18.2.0 | DOM 渲染 |
| `react-router-dom` | ^6.22.0 | 路由 |
| `zustand` | ^4.5.0 | 状态管理 |
| `tailwindcss` | ^3.4.0 | CSS 框架 |
| `@tauri-apps/api` | ^2.0.0 | Tauri 前端 API |
| `@tauri-apps/plugin-dialog` | ^2.7.0 | 对话框插件 |
| `@tauri-apps/plugin-shell` | ^2.0.0 | Shell 插件 |
| `typescript` | ^5.3.0 | 类型系统 |
| `vite` | ^5.0.0 | 构建工具 |

> **观察**：前端技术栈统一且现代，Tauri v2 插件版本与 API 版本对齐。无冗余 UI 库（如未引入 Radix、Headless UI 等）。

---

## 5. 架构健康

### 5.1 路由结构

**当前状态**：Flat 路由 + Legacy 嵌套路由并存（来源：`docs/scout-report.md`）

| 路由模式 | 路径示例 | 组件获取 projectId 方式 |
|----------|----------|------------------------|
| Flat（主要） | `/targets`, `/assets` | `useStore((s) => s.currentProject)` 或 `useParams().id`（fallback） |
| Legacy（注释标注） | `/projects/:id/assets` | `useParams<{ id: string }>()` |

**问题**：
- Legacy 路由组件虽读取 `useParams().id`，但 App.tsx 注释明确说明 "params not fully handled"。
- `RunsPage` 完全依赖 Zustand store，不使用 `useParams`，导致直接访问 `/projects/:id/runs` 会丢失项目上下文。
- 无 `<ProjectLayout>` 嵌套布局，每个页面自行处理 `projectId` 获取逻辑。

**健康评级**：🟡 **需改进** — v0.3 计划已决策迁移到混合嵌套路由，尚未实施。

### 5.2 状态管理

**方案**：Zustand（`frontend/src/lib/store.ts`）

| 维度 | 现状 | 评估 |
|------|------|------|
| Store 结构 | 单文件、扁平化，含 11 个状态字段 + 11 个 setter | 🟢 简洁，适合当前规模 |
| 跨页面状态 | `currentProject` 作为全局上下文，各页面自行订阅 | 🟡 有效但无嵌套路由配合时易漂移 |
| 持久化 | 无 `persist` 中间件，刷新页面后 `currentProject` 丢失 | 🔴 需改进（Sprint 1 计划） |
| 类型安全 | 完整的 `AppState` interface，TypeScript 类型覆盖 | 🟢 |

### 5.3 API 错误格式

**方案**：统一 `AppError` JSON 结构（来源：`docs/api-error-contract.md`）

| 维度 | 现状 | 评估 |
|------|------|------|
| Core API 错误格式 | 100% 使用 `writeError`，统一返回 `{"error": {"code", "message", "detail"}}` | 🟢 |
| Blob 端点错误 | 错误时切回 `application/json`，与 Core API 一致 | 🟢 |
| Worker API 错误 | `internal/worker/server.go` 中 3 处使用 `http.Error` 返回纯文本 | ⚠️ 与 Core API 契约不一致 |
| 前端错误处理 | `api.ts` 中 `fetchJSON` 统一捕获并抛出 `APIError`，部分页面有局部 `try/catch` | 🟡 无全局错误边界 |

---

## 6. 待办项（按优先级分级）

### 🔴 P0 — 阻塞或高安全风险

| # | 待办项 | 说明 | 建议 Sprint |
|---|--------|------|------------|
| P0-1 | API 层零测试 | `internal/api` 50+ handler 无任何测试，任何重构都无回归保护 | 0.9–1.0 |
| P0-2 | 数据库层零测试 | `internal/db` 所有查询方法无测试 | 0.9–1.0 |
| P0-3 | 扫描编排零测试 | `internal/workflow` 是核心业务逻辑，无任何测试 | 0.9–1.0 |

### 🟠 P1 — 严重影响可用性或可维护性

| # | 待办项 | 说明 | 建议 Sprint |
|---|--------|------|------------|
| P1-1 | 路由结构混乱 | Flat + Legacy 并存，`RunsPage` 不读 params，切换项目时上下文易丢失 | 1.0 |
| P1-2 | `currentProject` 无持久化 | 刷新页面后项目上下文丢失，用户需重新选择 | 1.0 |
| P1-3 | 无全局错误边界 | 网络断开或 API 异常时可能白屏（AC-009） | 1.0 |
| P1-4 | Worker API 错误格式不一致 | `http.Error` 纯文本与 Core API JSON 契约不统一 | 1.1 |

### 🟡 P2 — 中等优先级改进

| # | 待办项 | 说明 | 建议 Sprint |
|---|--------|------|------------|
| P2-1 | 总测试覆盖率 < 20% | 161 个测试用例集中在 parser/nuclei/report/scope，核心业务缺失 | 1.0–1.2 |
| P2-2 | `go test -race` 未常规运行 | 并发相关代码（`Server.mu`、`sseClients`、`taskQueue`）无 race 检测 | 1.1 |
| P2-3 | `node_modules` 环境缺失 | 当前工作树无法运行 `tsc --noEmit`，需确认 CI 中类型检查是否启用 | 0.11 |
| P2-4 | CORS `Allow-Origin: *` | 生产环境应收紧为白名单 | 1.2 |

### 🟢 P3 — 优化与债务

| # | 待办项 | 说明 | 建议 Sprint |
|---|--------|------|------------|
| P3-1 | `writeError` 无日志关联 | 生产环境难以关联请求 ID 与错误响应 | 后续 |
| P3-2 | `handleExportReportJSON` 双重 Content-Type 歧义 | 成功与错误均为 `application/json`，需前端检查 `Content-Disposition` | 后续 |
| P3-3 | 部分页面使用 `alert()` 报错 | TargetPage、RunsPage 等处使用浏览器 `alert`，应替换为统一 Toast | 1.1 |
| P3-4 | `tail -f` 式日志散落 | `log.Printf` 直接输出，未使用结构化日志 | 后续 |

---

## 附录：检查命令备忘

```bash
# Go 静态检查
go vet ./...

# Go 测试（全量）
go test ./...

# Go 覆盖率
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out

# Go Race 检测
go test -race ./...

# 前端类型检查（需先 npm install）
cd frontend && npm run typecheck

# 前端构建
cd frontend && npm run build

# Tauri 安全审计
cd src-tauri && cargo audit
```

---

*文档版本：v1.0*  
*基线代码版本：Sprint 0（2026-04-29）*
