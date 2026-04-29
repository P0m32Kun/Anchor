
## Sprint 0.4+0.9+0.10 — 前端配置修复（已完成）

### 任务 A：补 typecheck 脚本
- ✅ `frontend/package.json` 增加 `"typecheck": "tsc --noEmit"`
- ✅ `npm run typecheck` 零错误（无需拆分 tsconfig）

### 任务 B：Tauri 安全基线配置
- ✅ `tauri.conf.json` 设置 CSP
- ✅ 创建 `capabilities/default.json`（含 core:default / shell:allow-open / dialog:default）
- ✅ 创建 `src/lib.rs` 并迁移应用逻辑（含 `#[cfg_attr(mobile, tauri::mobile_entry_point)]`）
- ✅ `src/main.rs` 改为薄包装层 `app_lib::run()`
- ✅ `Cargo.toml` 添加 `[lib]` 段和 `tauri-plugin-dialog` 依赖
- ✅ 安装 `@tauri-apps/plugin-dialog`（前端）和 `tauri-plugin-dialog`（Rust）
- ✅ `cargo check` 通过

### 任务 C：Vite proxy 配置
- ✅ `vite.config.ts` 添加 `/api` → `http://localhost:17421` proxy
- ✅ `config.ts` Web 模式返回 `""`（使用相对路径走 Vite proxy）

### 验收标准
- [x] `npm run typecheck` 零错误
- [x] `cargo check` 通过（Tauri 编译正常）
- [x] 浏览器 dev 模式（`npm run dev`）可通过 Vite proxy 调用 Go API 无 CORS 报错

---

## Sprint 0 — 代码库全景扫描（已完成）

### 产出
- ✅ 完整扫描报告写入 `docs/scout-report.md`

### 扫描覆盖范围
1. **路由结构** — 14 条路由，4 条 Legacy 路由存在 useParams 不一致问题
2. **Zustand Store** — 10 个 state + 10 个 action，缺少 loading/error 状态和数据清理策略
3. **SSE** — 后端已实现（4 种事件），前端未接入（0 匹配）
4. **组件成熟度** — 7 个基础组件均成熟，缺少 Modal/Dialog、DataTable 等
5. **主流程可运行性** — `make dev` + `npm run dev` 可一键启动，低风险
6. **Tailwind 配置** — 完整的深色主题 Token 体系，缺少 spacing/typography 自定义

### 关键发现
- 🔴 路由与项目上下文获取方式不统一（5 种模式）
- 🔴 SSE 前端完全未接入
- 🔴 Store 缺少 loading/error 全局状态
- 🟡 DashboardPage/WorkersPage 绕过 api.ts 直接 fetch

---

## Sprint 0.1 + 0.8 — 后端 API 审计 + Go 覆盖率基线（已完成）

### 任务 A：后端 API 错误格式审计
- ✅ 读取 `internal/errors/errors.go` — AppError 结构确认
- ✅ 读取 `internal/api/handlers.go` — writeError 统一实现确认
- ✅ 审计所有 blob 下载端点（archive、report MD/JSON、curl）
- ✅ 发现 Worker API 存在 3 处直接 `http.Error` 调用（`internal/worker/server.go`）
- ✅ 产出 `docs/api-error-contract.md`

#### 关键发现
| 发现 | 状态 |
|------|------|
| Core API 全部使用 `writeError`，无直接 `http.Error` | ✅ 符合预期 |
| Worker API 有 3 处 `http.Error`（L53/L257/L263） | ⚠️ 待修复 |
| Blob 端点错误时 Content-Type 正确切回 JSON | ✅ 正确 |
| `export.json` 成功/错误 Content-Type 相同 | ⚠️ 需文档化 |

### 任务 B：Go 测试覆盖率基线
- ✅ `go test ./...` — 161 例全部通过
- ✅ `go test -coverprofile=coverage.out ./...` — 覆盖率报告生成
- ✅ `go tool cover -func=coverage.out` — 总体覆盖率 **18.5%**
- ✅ `internal/scope/` 已有测试文件（`scope_test.go` + `import_test.go`，覆盖率 68.4%）
- ✅ 产出 `docs/coverage-baseline.md`

#### 覆盖率概况
| 包 | 覆盖率 | 状态 |
|----|--------|------|
| `internal/nuclei` | 96.8% | 🟢 |
| `internal/parser` | 90.0% | 🟢 |
| `internal/report` | 78.3% | 🟢 |
| `internal/scope` | 68.4% | 🟢 |
| `internal/asset` | 47.1% | 🟡 |
| `internal/api` | 0.0% | 🔴 |
| `internal/db` | 0.0% | 🔴 |
| `internal/workflow` | 0.0% | 🔴 |
| `internal/worker` | 0.0% | 🔴 |
| 其他零覆盖 | 0.0% | 🔴 |

#### 零覆盖模块重点
- **workflow** — 扫描编排核心，零覆盖，风险最高
- **api** — 全部 handler 无测试
- **db** — 全部查询方法无测试
- **worker** — 分布式任务执行无测试
