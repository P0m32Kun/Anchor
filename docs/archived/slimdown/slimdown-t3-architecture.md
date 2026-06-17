---
status: archived
source_of_truth: false
owner: kun
audit_date: 2026-05-26
audit_baseline_commit: "main @ 7399a5f"
scope: architecture-coupling
archived_date: "2026-06-17"
archive_reason: "slimdown series completed, moved from active/review/"
---

# T3 报告：架构耦合与简洁度（Delta 复审）

> 审计日期：2026-05-26 | 基线：stage2-architecture.md (2026-05-13) 的 Delta

---

## 0. 变更范围（2026-05-13 至今新增/大改的文件）

| 文件 | 变更类型 | 行数 | 说明 |
|------|----------|------|------|
| `internal/toolrun/invoke.go` | **新增** | 178 | 统一工具执行入口 |
| `internal/toolregistry/schema.go` | **新增** | 296 | 工具注册表：YAML 解析 + Render |
| `internal/toolregistry/types.go` | **新增** | — | 类型定义 |
| `internal/toolregistry/embed.go` | **新增** | — | 嵌入 YAML |
| `internal/toolregistry/validate.go` | **新增** | — | 校验 |
| `internal/api/task_output_handlers.go` | **新增** | ~100 | 任务实时输出 |
| `internal/cdn/parse.go` + `parse_test.go` | **新增** | — | CDN 解析逻辑 |
| `internal/workflow/pipeline_tool.go` | **大幅重构** | **771** | 吸收了所有工具调用，从~400→771 |
| `internal/workflow/pipeline.go` | 重构 | 540 | 新增 tools 字段、toolregistry 集成 |
| `internal/workflow/pipeline_ffuf.go` | **新增** | 78 | ffuf 专用逻辑 |
| `internal/workflow/pipeline_cdn_portscan.go` | **新增** | 15 | CDN portscan 阶段 |
| `internal/workflow/pipeline_crawl.go` | 修改 | 64 | 改用 toolrun.Invoke |
| `internal/workflow/pipeline_passive.go` | 重构 | 344 | 加入 gau 集成 |

---

## 1. 评审维度评分

### 1.1 单一职责

| 文件 | 分数 | 理由 |
|------|------|------|
| `toolrun/invoke.go` | **4/5** | 职责清晰：Invoke 只做一件事（编排 argv → 执行 → 结果）。~178 行合理 |
| `toolregistry/schema.go` | **4/5** | Render 方法含部分后处理逻辑（mode 条件），但仍聚焦于 argv 构建 |
| `toolregistry/validate.go` | **3/5** | 与 schema.go 中的 Render 有少量重叠（参数校验在两者间分布） |
| `pipeline_tool.go` | **2/5** | 771 行，包含 12 种工具的调用函数（每个 ~30-60 行）+ 公共逻辑。**新上帝文件** |
| `pipeline.go` | **3/5** | 540 行，`NewPipeline` + `Run` + flushers + 各种辅助。职责多但内聚性可接受 |
| `task_output_handlers.go` | **5/5** | 单一 handler + 1 个辅助函数 + 1 个 proxy 函数，恰好 3 个函数，~100 行 |

### 1.2 依赖方向

| 检查项 | 分数 | 理由 |
|--------|------|------|
| `toolrun → toolguard + toolregistry` | **4/5** | 依赖方向正确：`toolrun` 调用 `toolguard` 做校验、`toolregistry` 做渲染。无反向依赖 |
| `toolregistry` 被谁依赖 | **3/5** | `toolun` + `workflow` 都依赖 `toolregistry`，但 `toolregistry` 不依赖任何业务包（仅有 `fmt` / `sort` / `strings`）— **正确的底层位置** |
| `workflow → toolrun + toolregistry` | **4/5** | 依赖方向正确：workflow 编排层调用 toolrun，不直接操作 worker |
| `api/task_output_handlers.go → worker` | **3/5** | 直接 import `internal/worker` 读 `ReadTaskOutput`，但这属于执行层的基础设施调用，可接受 |

### 1.3 重复执行路径

| 路径 | 分数 | 理由 |
|------|------|------|
| `exec.Command` 是否仍有平行路径 | **3/5** | toolrun 已统一 pipeline 工具调用，但以下地方仍有直 exec：<br>- `internal/cdn/detector.go:27,57` — cdncheck 直 exec（未走 toolrun）<br>- `internal/health/health.go:93,141` — 健康检查，不同领域<br>- `internal/worker/server.go:167` — 远程 worker，执行层自身<br>- `internal/worker/worker.go:206` — 本地 runner，执行层自身<br>- `internal/nuclei/custom/git.go:59` — git 操作<br>- `internal/builtin/sync.go:50-68` — git clone |
| CDN 检测是否应走 toolrun | 悬而未决 | `cdn/detector.go` 在 pipeline 中通过 `cdnDet.Detect(ctx, ip)` 调用，内部直 exec cdncheck。这与 pipeline 其余工具调用不一致 |

### 1.4 可测试性

| 组件 | 分数 | 理由 |
|------|------|------|
| `toolrun.Invoke` | **4/5** | 使用 `TaskRunner` 接口（可 mock）和 `ScanTaskDB` 接口（可 mock）。注入友好 |
| `toolregistry.Render` | **4/5** | 纯函数：输入 `(id, params)` → 输出 `(argv, error)`。极容易测 |
| `Pipeline` | **2/5** | 创建时需要 4 个真实依赖（`*db.Queries`, `*worker.Runner`, `*scope.Engine`, `dataDir string`），不是接口注入。`NewPipeline` 内部 new 了 `resolve.NewResolver()` 和 `cdn.NewDetector()`，无法在构造时 mock |
| `task_output_handlers.go` | **3/5** | handler 依赖 `s.queries`、`s.dataDir`、`task.WorkerID` 分支逻辑。可通过 httptest + mock queries 测试 |

### 1.5 配置单位

| 检查 | 分数 | 理由 |
|------|------|------|
| 是否存在禁止的单位换算 | **5/5** | ✅ 未发现代码内 `*1000` 等单位换算。工具参数通过 `toolregistry.RenderParams` 传递并保持工具原生单位 |
| `tools/*.yaml` 中的参数单位 | **5/5** | 以 flags 形式保持工具原生语义 |

---

## 2. 重点文件评估

### 2.1 `pipeline_tool.go` (771 行) — 新上帝文件

**问题**：这个文件吸收了旧版 `worker.Build*` 函数的全部工具调用逻辑后膨胀到 771 行。12 个 `run*Tool` 函数（runSubfinder, runDNSx, runHttpx 等）结构高度重复：

```
func (p *Pipeline) runXxx(ctx, input) (result, error) {
    // 1. 构建 InvokeInput
    // 2. 调用 toolrun.Invoke
    // 3. 解析 stdout
    // 4. 返回结果
}
```

**建议**：
- P0（瘦身级）：将每个 `run*Tool` 拆到独立文件`pipeline_tool_subfinder.go`、`pipeline_tool_dnsx.go` 等（预估每文件 30-60 行，共 12 个文件）
- 或者：提取公共模式到 `toolrun` 包，减少 pipeline_tool.go 的 boilerplate

### 2.2 `toolrun` + `toolregistry` 新增分层评价

**正面**：
- `toolregistry` 作为底层声明层（仅依赖标准库），位置正确
- `toolrun` 作为执行编排层，接口抽象完善（`TaskRunner` + `ScanTaskDB` 可 mock）
- 与 `toolguard` 的衔接干净

**改进点**：
- `toolregistry.Registry` 的 `HighRiskPorts` 常量（296 行大字符串）与 `worker/commands.go` 的 `HighRiskPorts` 重复。如 `pipeline_tool.go:624` 注释所写："Phase 3" 应移除旧版常量

### 2.3 `task_output_handlers.go` 新增 handler

**评价**：
- 职责单一，代码干净
- `proxyWorkerTaskOutput` 创建独立的 `http.Client{}`（无超时/连接池复用），应在 Server 层共享一个 HTTP client

### 2.4 CDN 检测路径不一致

`cdn/detector.go` 作为 pipeline 的一部分（`Pipeline.cdnDet`），绕过 toolrun 直接 `exec.CommandContext`。这意味着 CDN 检测不走：
- toolguard allowlist 校验
- toolrun 的统一 stdout 捕获
- toolrun 的 artifact 管理

**修复建议**：CDN 检测的 `Detect()` 内部改为调用 `toolrun.Invoke`（已在 pipeline_tool.go:448 实现了 cdncheck 的 toolrun 路径，但 pipeline_flow.go 可能走的旧路径）。

---

## 3. 上帝文件对比（vs 2026-05-13 基线）

| 文件 | 2026-05-13 | 2026-05-26 | 变化 | 建议 |
|------|-----------|-----------|------|------|
| `worker/server.go` | 730 | ~580 | ↓ 减少（重构？） | 仍高于阈值 |
| `pipeline.go` | ~540 | 540 | → 持平 | 可接受 |
| **`pipeline_tool.go`** | ~400 | **771** | **↑ +371** | **P0 — 应拆分为 12 个文件** |
| `pipeline_flow.go` | — | 380 | 新增 | 可接受 |
| `pipeline_passive.go` | — | 344 | 新增 | 可接受 |

---

## 4. 存量问题（stage2 报告未修复项）

| ID | 问题 | 等级 | 当前状态 |
|----|------|------|----------|
| A-01 | `retest_handlers.go` 直接开事务 | High | **未修复** — 仍在 handler 内 `s.rawDB.Begin()` |
| A-02 | `asset_handlers.go` 200+ 行聚合逻辑在 handler | Medium | **未修复** — 未下沉到 Service |
| A-03 | `pipeline_handlers.go` buildConfigForMode 78 行业务逻辑 | Medium | **未修复** — 仍在 handler 内 |
| A-04 | Service 层仅 3 个，缺 PipelineService/AssetService/ReportService | Medium | **未修复** |
| A-05 | worker/server.go 仍高于 500 行 | Medium | **未完全修复** — 有改善但仍需拆分 |
| A-06 | `api -> db` 3 处直接依赖 | Medium | **未修复** |

---

## 5. 建议 P0/P1/P2 Backlog（仅瘦身级）

| 等级 | 项 | 预估工作量 | 文件 |
|------|-----|-----------|------|
| **P0** | 拆分 `pipeline_tool.go` 为 12 个工具文件 | 1 工日 | `workflow/pipeline_tool_*.go` |
| **P1** | 统一 CDN 检测路径（走 toolrun.Invoke） | 0.5 工日 | `cdn/detector.go` + `pipeline_flow.go` |
| **P1** | 消除 `toolregistry.HighRiskPorts` 与 `worker/commands.go` 重复 | 0.5 工日 | `toolregistry/schema.go` + `worker/commands.go` |
| **P2** | 为 `NewPipeline` 注入接口（Resolver/CDNDetector）而非直接 new | 1 工日 | `workflow/pipeline.go` |
| **P2** | 共享 HTTP client 到 Server 结构体 | 0.5 工日 | `api/task_output_handlers.go` |

**不在此范围**（不属瘦身级）：
- 不拆分 `worker/server.go`（与 2026-05-13 stage2 一致的结论：这是更大的重构）
- 不新增 Service 层（P1 项，保持现有结构）
- 不合并 handler 文件（项目政策禁止）

---

## 6. 风险分级汇总

| 等级 | 数量 | 说明 |
|------|------|------|
| High | 1 | `pipeline_tool.go` 771 行的新上帝文件 |
| Medium | 4 | CDN 执行路径不一致、Port 常量重复、NewPipeline 硬依赖、handler 事务 |
| Low | 2 | 共享 HTTP client、存量 stage2 问题 |
