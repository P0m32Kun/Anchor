# Go 测试覆盖率基线报告

> 生成日期：2026-04-29  
> Sprint：0.8  
> 总语句覆盖率：**18.5%**

---

## 1. 总体概况

| 指标 | 值 |
|------|-----|
| 总包数 | 15 |
| 有测试的包 | 5 |
| 零覆盖包 | 10 |
| 总语句覆盖率 | **18.5%** |
| 测试用例总数 | 161 |
| 测试状态 | ✅ 全部通过 |

---

## 2. 各模块覆盖率表格

| 包路径 | 语句覆盖率 | 测试文件 | 状态 |
|--------|-----------|----------|------|
| `internal/nuclei` | **96.8%** | `tagmapper_test.go` | 🟢 优秀 |
| `internal/parser` | **90.0%** | `nuclei_test.go`, `naabu_test.go`, `subfinder_test.go`, `httpx_test.go` | 🟢 优秀 |
| `internal/report` | **78.3%** | `report_test.go` | 🟢 良好 |
| `internal/scope` | **68.4%** | `scope_test.go`, `import_test.go` | 🟢 良好 |
| `internal/asset` | **47.1%** | `normalizer_test.go` | 🟡 中等 |
| `internal/api` | **0.0%** | ❌ 无 | 🔴 零覆盖 |
| `internal/db` | **0.0%** | ❌ 无 | 🔴 零覆盖 |
| `internal/errors` | **0.0%** | ❌ 无 | 🔴 零覆盖 |
| `internal/health` | **0.0%** | ❌ 无 | 🔴 零覆盖 |
| `internal/models` | **0.0%** | ❌ 无 | 🔴 零覆盖 |
| `internal/scoring` | **0.0%** | ❌ 无 | 🔴 零覆盖 |
| `internal/util` | **0.0%** | ❌ 无 | 🔴 零覆盖 |
| `internal/worker` | **0.0%** | ❌ 无 | 🔴 零覆盖 |
| `internal/workflow` | **0.0%** | ❌ 无 | 🔴 零覆盖 |
| `(root)` | **0.0%** | ❌ 无 | 🔴 零覆盖 |

---

## 3. 零覆盖模块列表（重点关注）

### 🔴 核心业务逻辑（优先补测）

| 模块 | 说明 | 风险等级 |
|------|------|----------|
| `internal/workflow` | 扫描编排（discovery、screenshot） | **高** — 核心扫描流程无测试 |
| `internal/api` | 所有 HTTP handler（~50 个 handler） | **高** — API 行为无回归保护 |
| `internal/db` | 所有数据库查询方法 | **高** — 数据层无测试 |
| `internal/worker` | Worker 服务端与任务执行 | **中高** — 分布式任务逻辑无测试 |

### 🟡 基础设施/工具类

| 模块 | 说明 | 风险等级 |
|------|------|----------|
| `internal/errors` | AppError 结构体与构造函数 | 低 — 结构简单，但建议补测 |
| `internal/util` | ID 生成、路径处理等工具 | 低 |
| `internal/models` | 数据模型定义 | 低 — 多为类型定义 |
| `internal/health` | 健康检查逻辑 | 低 |
| `internal/scoring` | 漏洞评分计算 | 中 — 业务逻辑建议覆盖 |

---

## 4. 测试框架状态

### 4.1 已有测试文件清单

```
internal/asset/normalizer_test.go     → asset 包
internal/nuclei/tagmapper_test.go     → nuclei 包
internal/parser/nuclei_test.go        → parser 包
internal/parser/naabu_test.go         → parser 包
internal/parser/subfinder_test.go     → parser 包
internal/parser/httpx_test.go         → parser 包
internal/report/report_test.go        → report 包
internal/scope/scope_test.go          → scope 包
internal/scope/import_test.go         → scope 包
```

### 4.2 测试模式观察

- **表驱动测试**：parser、nuclei、asset 包已使用标准的 Go table-driven 测试模式
- **子测试 (`t.Run`)**：parser 包广泛使用
- **无接口/mock 测试**：db 层无测试，api 层无测试，尚未建立 mock 基础设施
- **无并发测试**：`-race` 检测尚未在 CI 中常规运行

### 4.3 关于 `internal/scope` 测试状态

任务要求"为 `internal/scope/` 编写第一个测试文件（如尚未有）"，经确认：
- ✅ `internal/scope/scope_test.go` 已存在（7.8K）
- ✅ `internal/scope/import_test.go` 已存在（14.2K）
- 覆盖率 68.4%，已有较完善的测试覆盖，无需新增首个测试文件

---

## 5. 补测优先级建议

### Phase 1（Sprint 0.9–1.0）—— 核心流程
1. `internal/workflow` — 至少覆盖 `discovery.go` 的主流程
2. `internal/errors` — 快速补齐（<30 分钟），覆盖 `New`、`Newf`、`WithDetail`、`Error()`
3. `internal/api` — 从 `writeError`/`writeJSON` 等纯函数开始，再逐步覆盖 handler

### Phase 2（Sprint 1.1–1.2）—— 数据层 + Worker
4. `internal/db` — 引入 `sqlmock` 或 SQLite in-memory 测试
5. `internal/worker` — 测试任务执行状态机与 HTTP 服务端点

### Phase 3（后续）—— 工具类
6. `internal/scoring`、`internal/util`、`internal/health`

---

## 6. 运行命令备忘

```bash
# 运行全部测试
go test ./...

# 生成覆盖率报告
go test -coverprofile=coverage.out ./...

# 查看函数级覆盖率
go tool cover -func=coverage.out

# 查看 HTML 覆盖率（浏览器）
go tool cover -html=coverage.out

# 带 race 检测运行
go test -race ./...
```
