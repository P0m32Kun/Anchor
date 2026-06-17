---
status: active
source_of_truth: true
owner: kun
audit_date: 2026-05-26
audit_baseline_commit: "main @ 7399a5f"
scope: testing-e2e-matrix
---

# T5 报告：测试与 E2E 矩阵

> 审计日期：2026-05-26 | Delta 基线：stage3-testing.md (2026-05-13)

---

## 1. 覆盖指标对比（2026-05-13 → 2026-05-26）

| 指标 | 2026-05-13 | 2026-05-26 | 变化 | 健康阈值 | 状态 |
|------|-----------|-----------|------|----------|------|
| Go 源文件 | 172 | 175 | +3 | — | — |
| Go 测试文件 | 28 (16%) | **64 (37%)** | **+36 (+21pp)** | >40% | 大幅改善 |
| 前端源文件 | 52 | 56 | +4 | — | — |
| 前端测试文件 | 4 (8%) | 4 (7%) | 0 | >30% | ❌ 未改善 |
| 0% 覆盖率的包 | 11 | **3** (errors, health, passive) | **-8** | 0 | ❌ 仍有 3 个 |

### 包级覆盖率变迁

| 包 | 2026-05-13 | 2026-05-26 | delta | 状态 |
|----|-----------|-----------|-------|------|
| api | 12.5% | **19.8%** | +7.3pp | ✅ 改善 |
| worker | **0%** | **12.6%** | +12.6pp | ✅ 改善 |
| workflow | 8.9% | **15.3%** | +6.4pp | ✅ 改善 |
| scope | 70.7% | **85.8%** | +15.1pp | ✅ 显著改善 |
| cdn | **0%** | **50.0%** | +50pp | ✅ 显著改善 |
| db | 25.8% | **42.1%** | +16.3pp | ✅ 显著改善 |
| service | 14.1% | **34.3%** | +20.2pp | ✅ 显著改善 |
| **NEW** toolregistry | — | **66.9%** | — | ✅ 新建包即高覆盖 |
| **NEW** toolrun | — | **53.4%** | — | ✅ 新建包即高覆盖 |
| **NEW** parser | 75.5% (*) | **73.3%** | — | → 持平（加 gau/katana parser 后略降） |

> (*) 2026-05-13 的 parser 仅含 subfinder/httpx/nuclei，今日加了 gau/katana

---

## 2. 新增测试文件清单（自 2026-05-13 以来）

| 测试文件 | 包 | 说明 |
|---------|---|------|
| `internal/toolregistry/registry_test.go` | toolregistry | 14 个 golden test（vs old Build* 命令）|
| `internal/toolrun/invoke_test.go` | toolrun | Invoke 单元测试 |
| `internal/cdn/parse_test.go` | cdn | CDN 解析逻辑 |
| `internal/parser/katana_test.go` | parser | Katana 输出解析 |
| `internal/parser/gau.go` | parser | GAU 输出解析（虽然非 test 文件）|
| `internal/worker/commands_katana_test.go` | worker | Katana 命令构建 |
| `internal/worker/task_output_test.go` | worker | 实时输出读取 |
| `internal/workflow/pipeline_cdn_portscan_test.go` | workflow | CDN portscan 阶段 |
| `internal/workflow/pipeline_external_integration_test.go` | workflow | 外网扫描集成测试 |
| `internal/workflow/pipeline_ffuf_test.go` | workflow | ffuf 逻辑 |
| `internal/workflow/pipeline_passive_test.go` | workflow | 被动扫描逻辑 |
| `internal/search/fofa_mock_test.go` | search | FOFA mock |

---

## 3. 新建包零测试清单

| 包 | 覆盖率 | 测试文件 | 状态 |
|---|--------|---------|------|
| `internal/toolregistry` | **66.9%** | `registry_test.go` | ✅ 已覆盖 |
| `internal/toolrun` | **53.4%** | `invoke_test.go` | ✅ 已覆盖 |
| `internal/cdn` | **50.0%** | `parse_test.go`, `detector_test.go` | ✅ 已覆盖 |
| `internal/passive` | **0%** | 无 | ❌ `crt.go` 无测试（仅 1 个文件） |
| `internal/health` | **0%** | 无 | ❌（健康检查，风险低） |
| `internal/errors` | **0%** | 无 | ❌（错误类型定义，风险低） |

---

## 4. 瘦身回归 E2E 最小集

### 建议 5 条用户流

| # | 流 | 前置条件 | 预期结果 | 命令 |
|---|-----|---------|---------|------|
| **F1** | 创建 Target（公司/域名） | 项目已存在 | Target 出现在列表 | `POST /projects/{id}/targets` + DB 验证 |
| **F2** | 触发外网 scan | 项目有 domain target | PipelineRun 创建 + stages 推进 | `POST /projects/{id}/pipeline/run` 含 body `{"mode":"external"}` |
| **F3** | 任务列表可见且状态推进 | F2 正在运行 | `GET /pipeline/runs/{runId}/stages` 返回非空 stages | curl 轮询 |
| **F4** | Task live output 可拉取 | F2 有 running task | `GET /tasks/{id}/output?stream=stdout` 返回 content | curl 检查 content 非空 |
| **F5** | Run 结束后 Runs 页可见结果 | F2 已完成 | `GET /projects/{id}/runs` 返回结果列表 | curl 返回 runs 含 status=completed |

### 环境需求

| 依赖 | 说明 |
|------|------|
| Docker compose | `worker` + `rangefield` 容器 |
| SQLite DB | 已 migrate |
| FOFA API Key | 可选（F5 可不依赖外部 API） |
| Worker 容器 | 运行外网工具 |

### BLOCKED 原因

如果环境不满足上述条件，F2-F5 无法通过。在审计环境中仅能验证 F1（CRUD + DB 直查）。

### 可在审计环境运行的验证

```bash
# F1 — 创建 Target（需先有项目）
go test -run TestCreateTarget ./internal/api/ -v

# F4 — Task output 路径（单元测试验证）
go test -run TestReadTaskOutput ./internal/worker/ -v

# 全部可用单元测试通过
go test ./internal/... | grep -c '^ok'
```

---

## 5. 记录结果

| 命令 | 结果 |
|------|------|
| `go vet ./internal/...` | ✅ 通过 |
| `npm --prefix frontend run typecheck` | ✅ 通过 |
| `go test ./internal/...` | ✅ 29/29 包全绿 |
| `go test -cover ./internal/...` | 最低包 0%（errors/health/passive），最高 100%（toolguard） |

---

## 6. 建议

| 优先级 | 项 | 说明 |
|--------|----|------|
| P2 | 为 `internal/passive/crt.go` 添加测试 | 唯一缺测的新代码 |
| P3 | 将 F2-F5 编写为 Go 集成测试（httptest + mock worker） | 当前无可运行的全栈 E2E 脚本 |
