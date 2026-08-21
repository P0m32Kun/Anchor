---
status: in_review
source_of_truth: false
owner: kun
last_updated: 2026-05-26
scope: tool-registry-artifact-store
verification: pending_implementation
---

# 工具注册表与执行结果存储 — 设计方案

> **状态**: In Review（提案，非当前架构基线）  
> **受众**: 实现 Agent / Reviewer  
> **基线**: 仍以 [`docs/current/architecture.md`](../current/architecture.md) 为准；落地并验收后，在本文件标记 `accepted` 并更新 architecture 的「工具执行」小节。

## 1. 背景与问题

Anchor 当前将 **CLI 拼装**（`internal/worker/commands.go` 的 `Build*Command`）、**安全门禁**（`internal/toolguard/allowlist.go`）、**管线编排**（`internal/workflow/pipeline_*.go`）和 **输出解析**（`internal/parser`）分散在多处。

典型调用链：

```text
pipeline.runNaabu
  → worker.BuildNaabuCommand(hostFile, portRange, rate, threads, timeout)
  → createAndRunTask(ctx, "naabu", args)
  → runner.Run → RawArtifact(stdout)
  → readTaskStdout → os.ReadFile 全量 []byte
  → parser.ParseNaabuOutput
```

带来的维护成本：

| 变更类型 | 当前需改动 |
|----------|------------|
| 调整 naabu 默认 `-rate` | `commands.go` + 可能多处 `PipelineConfig` 传参 |
| 新增二进制到 allowlist | `allowlist.go` + `commands.go` |
| 替换端口扫描实现 | `commands.go` + `parser` + 所有 `runNaabu` 调用 + allowlist |
| 大体积 stdout（nuclei/ffuf） | 全量读入内存；API `handleGetArtifactContent` 亦全量返回 |

参考 [CyberStrikeAI](https://github.com/Ed1s0nZ/CyberStrikeAI) 的 `tools/*.yaml` 与「大结果附件化」思路，但 **不** 引入其 AI 编排、100+ 工具广度或 C2/WebShell 能力。

## 2. 目标

1. **执行与编排解耦**：管线只表达「跑哪个工具、输入是什么」；argv 由统一层根据声明生成。
2. **单一事实来源**：每个 Worker 子进程的 binary、参数 schema、输出格式在 **工具注册表** 中声明；allowlist 由注册表派生，避免双份维护。
3. **可替换工具**：同类能力（如端口扫描）可通过换 `tool_id` + 换 parser 完成，不必在编排层复制 `Build*` 逻辑。
4. **大结果可控**：stdout/stderr 始终以 `RawArtifact` 落盘；超过阈值时编排层默认 **流式/分页读取**，不把整文件载入内存。
5. **行为等价迁移**：首版迁移不改变现有外网/内网 preset 的可观测行为（命令行、stage、解析结果一致），仅重构结构。

## 2.1 已决事项（2026-05-26）

| # | 议题 | 决定 |
|---|------|------|
| 1 | Nuclei 声明方式 | **全 YAML**（与 naabu 相同）；不保留 Go `builder`；迁移时 **移除** 已无用的 `-stats -si 30` |
| 2 | Ad-hoc 手调命令 | **无场景**；废弃 `POST /tasks/run` 的任意 `command` 字符串；扫描任务 **仅** 经 `toolrun.Invoke(tool_id, params)` |
| 2b | 是否仍限制可调用工具 | **是** — registry 定义「管线能点名哪些 `tool_id`」；allowlist 在 Worker `exec` 前做最后一道二进制校验（见 §9.1） |
| 3 | Registry 版本入库 | **不要**；历史复现依赖 `ScanTask.CommandTemplate`；任务/产物 **保留期 ≤30 天**（与现有清理策略对齐，本方案不引入永久审计库） |
| 4 | 执行位置 | **仅 HTTP/API 类步骤可在 Server 进程**（FOFA、Hunter、Quake、crt.sh）；**一切 CLI 子进程必须在 Worker**（含 `gau`，从 `internal/passive` 迁出） |

## 3. 非目标

- **不把整条 pipeline 写成 YAML**（P1–P5 顺序、CDN 跳过、company 展开、Nuclei workflow 策略仍留在 Go）。
- **不服务通用 AI Agent 随意选工具**（无「100 工具目录」；注册表仅包含 Anchor 管线已用或明确计划使用的工具）。
- **不在 YAML 中做单位换算**（字段语义与上游 CLI 一致，遵守项目「禁止 `*1000`」约束）。
- **首版不改造** FOFA/Hunter/Quake/crt.sh 等 HTTP 被动源（继续 `internal/passive` + `recordPassiveTask`）。
- **首版不要求** 运行时热加载（可选 Phase 2）；默认 embed + 可选目录覆盖。
- **不替代** 自定义 Nuclei 源管理（见 [`custom-nuclei-template-management.md`](custom-nuclei-template-management.md)）。

## 4. 设计原则

1. **四层分离**（见 §5）：声明 → 执行 → 解析 → 编排；依赖单向向下。
2. **显式编排**：`pipeline_flow.go` 保留可读的业务分支；禁止用隐式 DAG 配置替代。
3. **Fail closed**：未在注册表中的 `tool_id`、未知参数、allowlist 失败 → 任务失败并写 stage 错误，不静默降级。
4. **可复现**：`ScanTask.CommandTemplate` 继续保存最终 argv 字符串；注册表版本号写入 task 元数据（可选字段，见 §8）。
5. **与 Worker 模型一致**：子进程仍由现有 `Runner` / `WorkerServer` 执行；注册表只负责 **生成 `[]string` args**。

## 5. 架构分层

```text
┌─────────────────────────────────────────────────────────────┐
│ L4 编排  internal/workflow/pipeline_*.go                   │
│     runPortScan(ctx, hostFile)  // 业务：CDN 过滤、stage 事件  │
└───────────────────────────┬─────────────────────────────────┘
                            │ Invoke(toolID, InvokeInput)
┌───────────────────────────▼─────────────────────────────────┐
│ L3 解析  internal/parser (+ internal/cdn, fingerprint)     │
│     Parse(toolID, io.Reader) → 领域类型                       │
└───────────────────────────┬─────────────────────────────────┘
                            │ ArtifactRef (path, size, format)
┌───────────────────────────▼─────────────────────────────────┐
│ L2 执行  internal/toolrun (新包)                            │
│     Resolve argv → allowlist → 创建 ScanTask → Runner.Run     │
│     OpenStdout(ref) → io.Reader / 分页 API                    │
└───────────────────────────┬─────────────────────────────────┘
                            │ 读取 ToolDef
┌───────────────────────────▼─────────────────────────────────┐
│ L1 声明  tools/*.yaml + internal/toolregistry (新包)         │
│     binary, parameters, presets, output, timeouts             │
└─────────────────────────────────────────────────────────────┘
```

### 5.1 L1 — 工具注册表 (`toolregistry`)

**职责**

- 加载并校验 `ToolDef` 列表（见 §6）。
- 提供 `Get(id)`, `List()`, `Binaries()`（供 allowlist 同步）。
- 暴露 `RegistryVersion`（内容 hash 或 semver），供审计。

**加载顺序**

1. `//go:embed tools/*.yaml`（默认、可复现构建）
2. 若 `ANCHOR_TOOLS_DIR` 非空，则 **覆盖/追加** 同名 `id`（开发调参；生产可选关闭）

**不在注册表内的执行**

| 类型 | 示例 | 处理方式 |
|------|------|----------|
| HTTP 被动 API | FOFA, Hunter, Quake, crt.sh | Server 内 HTTP 客户端 + `recordPassiveTask`；**不** `exec` |
| CLI（含 gau） | subfinder, naabu, gau, … | **仅** `toolrun.Invoke` → Worker `exec` |
| 复合逻辑 | Nuclei 按 tag/workflow 多轮 | L4 循环 `toolrun.Invoke("nuclei", params)`；argv 全由 `tools/nuclei.yaml` 生成 |
| Ad-hoc 用户命令 | 原 `POST /tasks/run` | **移除或改为** `tool_id` + `params`；无任意 command 入口 |

### 5.2 L2 — 工具运行时 (`toolrun`)

**核心 API（Go 接口，示意）**

```go
// InvokeInput 由 L4 构造；不嵌入 PipelineConfig 全量字段。
type InvokeInput struct {
    ProjectID string
    RunID     *string
    TaskID    string // 可选，空则生成
    ToolID    string
    Params    map[string]any // 由 registry 校验
}

type InvokeResult struct {
    Task      *models.ScanTask
    Stdout    ArtifactHandle // 非空时表示可读
    Err       error
}

func Invoke(ctx context.Context, runner TaskRunner, reg *toolregistry.Registry, in InvokeInput) (InvokeResult, error)

func OpenArtifact(h ArtifactHandle) (io.ReadCloser, error)
func ReadArtifactRange(h ArtifactHandle, offset, limit int64) ([]byte, error)
```

**职责**

1. 根据 `ToolDef` + `Params` 生成 `[]string`（等价于今日 `Build*Command`）。
2. 调用 `toolguard.Allowlist.Validate(binary, args[1:])`。
3. 创建 `ScanTask`（`Tool` 字段 = `tool_id`，`CommandTemplate` = 拼接后的命令行）。
4. 调用现有 `runner.Run(ctx, taskID)`。
5. 返回 `ArtifactHandle`（path + size + `output_format`），**默认不** `ReadFile` 全量。

**与 `createAndRunTask` 关系**

- `pipeline.createAndRunTask` 退化为对 `toolrun.Invoke` 的薄封装（迁移期可保留旧签名，内部转发）。
- 最终删除 `worker.Build*Command` 的对外调用（函数可保留为 registry 单元测试的黄金对照，或删除）。

### 5.3 L3 — 解析器（现有 `parser`，小改）

**约定**

- 每个 `tool_id` 在 `ToolDef` 中声明 `output.format`: `jsonl` | `xml` | `greppable` | `text`。
- L4 调用 `toolrun.OpenArtifact` 得到 `io.Reader`，传入现有 `Parse*`。
- 新增工具时：**必须** 新增或复用 parser；注册表 **不** 承担结构化解析。

**Parser 注册（可选 Phase 2）**

```go
var parsers = map[string]func(io.Reader) (any, error)
```

首版可继续显式 `parser.ParseNaabuOutput(r)`，不强制 map。

### 5.4 L4 — 管线编排（现有 workflow，语义不变）

**改造方式**

- 将 `worker.BuildXxx(...)` 替换为 `toolrun.Invoke(..., ToolID: "xxx", Params: {...})`。
- **保留** 所有业务逻辑：如 `ipsForPortScan`、`SkipPortscanOnCDNHost`、`NucleiScanDepth` 分支、`customWorkflowPaths`。

**示例（概念）**

```go
// 之前
task, stdout, err := p.createAndRunTask(ctx, "naabu",
    worker.BuildNaabuCommand(hostFile, p.config.PortRange, ...))

// 之后
res, err := toolrun.Invoke(ctx, p.runner, p.tools, toolrun.InvokeInput{
    ProjectID: p.projectID,
    RunID:     &p.runID,
    ToolID:    "naabu",
    Params: map[string]any{
        "hostFile":  hostFile,
        "portRange": p.config.PortRange,
        "rate":      p.config.NaabuRate,
        "threads":   p.config.NaabuThreads,
        "timeout":   p.config.NaabuTimeout,
    },
})
ports := parser.ParseNaabuOutput(res.Stdout.Reader()) // 或 Open + defer Close
```

## 6. 工具声明格式（YAML）

文件位置：**仓库根** `tools/<id>.yaml`（与 `internal/builtin` 的 dict/templates 无关）。

### 6.1 顶层字段

| 字段 | 必填 | 说明 |
|------|------|------|
| `id` | 是 | 稳定标识，与 `ScanTask.Tool`、stage 展示一致 |
| `binary` | 是 | 可执行文件名（allowlist basename） |
| `description` | 否 | 人类可读说明 |
| `output.format` | 是 | `jsonl` / `xml` / `greppable` / `text` |
| `output.artifact_type` | 否 | 默认 `stdout`；与 `models.ArtifactStdout` 对齐 |
| `timeout_default_sec` | 否 | 覆盖 `defaultToolTimeout` 表 |
| `parameters` | 是 | 见 §6.2 |
| `presets` | 否 | 命名参数变换（如 `portRange`） |
| `literals` | 否 | 固定追加的 flag（如 `-json`） |

### 6.2 参数类型

| type | 说明 |
|------|------|
| `string` | 直接作为 flag 值或 positional |
| `int` | `>0` 时才追加（与今日 Build* 行为一致） |
| `string_list` | 逗号连接或 repeated flag（按工具定义） |
| `path` | 主机文件路径；校验无 shell 元字符 |
| `enum` | 限定取值 |

每个参数可选：

- `flag` / `flags`：CLI 标志
- `position`：无 flag 时的位置参数索引
- `required`：默认 false
- `when`：简单条件（如 `scanDepth == workflow`）— Phase 2；首版复杂条件仍由 L4 传不同 `tool_id` 或省略参数

### 6.3 Presets（端口范围等）

将 `BuildNaabuCommand` 内 `switch portRange` 迁入声明：

```yaml
id: naabu
binary: naabu
output:
  format: jsonl
literals:
  - ["-json"]
  - ["-list"]  # 实际由 parameter hostFile 提供 path
parameters:
  hostFile: { type: path, required: true, flag: "-list" }
  rate:       { type: int, flag: "-rate" }
  threads:    { type: int, flag: "-c" }
  timeout:    { type: int, flag: "-timeout" }  # 毫秒，与 naabu CLI 一致
  portRange:
    type: enum
    preset: port_range_naabu  # 引用 presets 块
presets:
  port_range_naabu:
    top100: []                    # 无额外 flag
    top1000: [["-tp", "1000"]]
    full: [["-tp", "full"]]
    high-risk: [["-p", "<HighRiskPorts 常量>"]]  # 见 §6.5
    default: [["-p", "{{value}}"]] # custom 列表
```

`HighRiskPorts` 等 **长常量** 保留在 Go（`toolregistry` 或 `worker` 包常量），YAML 通过 `preset_ref: HighRiskPorts` 引用，避免 YAML 内嵌 2KB 字符串。

### 6.4 首版纳入注册表的工具清单

| tool_id | 替代今日 | 备注 |
|---------|----------|------|
| `subfinder` | `BuildSubfinderCommand` | 含 passive / `-pc` |
| `dnsx` | `BuildDNSxCommand` | |
| `cdncheck` | `BuildCDNCheckCommand` | |
| `nmap_alive` | `BuildNmapAliveCommand` | greppable |
| `nmap_service` | `BuildNmapServiceScanCommand` | xml；ports 为 int list |
| `naabu` | `BuildNaabuCommand` | presets |
| `httpx` | `BuildHttpxCommand` | 含 `-cff` |
| `katana` | `BuildKatanaCommand` | |
| `ffuf` | `BuildFfufCommand` | |
| `nuclei` | `BuildNucleiCommand` | 多模式见 §6.6 |
| `nuclei_custom` | `BuildNucleiCustomCommand` | 合并进 `nuclei` 参数（`customTemplatesDir` / `customWorkflowsDir`）或单独 `tool_id` |
| `gau` | `passive.RunGau`（Server 本地 exec） | Phase 2 迁入；**必须**走 Worker |

**不纳入 registry**

- `git` / `sh` / `bash`：builtin sync 与 Worker 内部脚本；列入 allowlist **系统扩展**，不进 `tools/*.yaml`。

### 6.5 Nuclei（全 YAML，已决）

- 单文件 `tools/nuclei.yaml`：`profile`、`scanDepth`、`workflowDir`、`templatePath`、`tags`、`rateLimit` 等用 `parameters` + `when` 表达（与 `BuildNucleiCommand` 逻辑等价）。
- **刻意不迁移** `-stats -si 30`：代码库中已无 `idleOutputTimeout` 消费方，该 flag 仅增加 stderr 噪音。
- **留在 L4 的**：按 endpoint/tag/workflow 的 **多次 Invoke** 循环；不在 YAML 里描述 DAG。

### 6.6 变体与 tool_id

| 策略 | 适用 |
|------|------|
| 单 `tool_id` + 多参数 | naabu, httpx, nuclei, gau |
| 多 `tool_id` | CLI 形状完全不同（如未来 `rustscan` vs `naabu`） |

替换工具时：新增 `tools/rustscan.yaml` + `parser.ParseRustscan` + L4 将 `naabu` 改为 `rustscan`（或通过 preset 配置 `port_scan_tool: naabu`）。

## 7. 大结果存储与读取

### 7.1 阈值

| 常量 | 建议值 | 行为 |
|------|--------|------|
| `ArtifactInlineMax` | 256 KiB | L4 解析可一次性读入内存 |
| `ArtifactPageDefault` | 64 KiB | API 默认页大小 |
| `ArtifactPageMax` | 1 MiB | API 单页上限 |

现有 `maxEvidenceSize`（10MB）用于 finding evidence，与本方案独立。

### 7.2 管线侧

- `toolrun.Invoke` 返回的 `ArtifactHandle` 带 `Size`。
- `Size <= ArtifactInlineMax`：`parser` 可用 `io.ReadAll`。
- `Size > ArtifactInlineMax`：**必须** `bufio.Scanner` 按行（jsonl）或 `xml.Decoder` 流式解析；禁止 `readTaskStdout` 式全量读。

### 7.3 API 侧

扩展 `GET /api/.../artifacts/content`（或新路由）：

```http
GET /tasks/{taskId}/artifacts/{artifactId}/content?offset=0&limit=65536
```

- 响应：`200` + `Content-Range` 或 JSON `{ data, offset, total, truncated }`。
- 无 `offset/limit` 且 `size > ArtifactInlineMax` → `413` 或强制分页，避免 OOM。
- 现有全量 `handleGetArtifactContent` 行为在迁移期对 **小文件** 保持兼容。

### 7.4 前端

- Runs / Task 详情：大 stdout **默认摘要**（行数、前 N KB），「加载更多」走分页 API。
- 首版可仅后端就绪，前端仍限制展示 256KB（与阈值对齐）。

## 8. 数据模型与保留期

| 项 | 变更 |
|----|------|
| `scan_tasks` | **不新增** `tool_registry_version`；复现靠 `command_template` |
| `raw_artifacts` | 无 schema 变更；已有 `size`, `sha256` |
| `Pipeline` struct | 注入 `*toolregistry.Registry`（Server 启动时构造一次） |
| 保留期 | 任务与 stdout 产物 **≤30 天**；本方案不扩展长期归档（与产品数据生命周期一致） |

**不新增** `tool_executions` 表；`ScanTask` + `RawArtifact` 已足够。

## 9. Allowlist 与调用边界

### 9.1 为何全自动仍要「可调用工具」控制

无手调 API 后，入口只剩管线/L4 代码，但仍需要 registry + allowlist：

1. **编排白名单**：L4 只能 `Invoke` 注册表里存在的 `tool_id`；防止实现错误写成 `Invoke("nmap", …)` 时拼成 `nc` 或随意二进制。
2. **Worker 最后一道门**：`exec` 前 `toolguard.Validate`；即便 DB/代码被改坏，未声明二进制仍失败。
3. **参数 schema**：registry 校验 `Params` 类型与必填项，避免把路径/端口拼进错误 flag。
4. **运维面一致**：Worker 镜像只需安装 registry 列出的工具；health check 与 `tools/*.yaml` 对齐。
5. **未来扩展**：若增加「按模板重跑单 stage」，仍复用同一 `tool_id` 集合，无需新开任意 exec 口子。

Registry = **产品允许使用的扫描能力目录**；allowlist = **进程执行层硬约束**。二者冗余是故意的（纵深防御）。

### 9.2 集成方式

```text
Server 启动:
  reg := toolregistry.Load()
  // Server 不用于扫描 CLI exec；registry 供 pipeline / toolrun 生成 argv

Worker 启动:
  reg := toolregistry.Load()  // 同 embed 内容
  allowlist := toolguard.NewAllowlistFromRegistry(reg.Binaries())
  + systemBinaries: git, sh, bash  // 非扫描
```

- `toolguard.NewAllowlist()` 保留，用于测试。
- 新增扫描能力 = 新增 `tools/<id>.yaml` + parser +（可选）L4 stage；**禁止** 只改 allowlist。
- **Server 禁止** `exec.Command` 扫描类 CLI（`gau` 迁移后删除 `passive.RunGau` 的本地 exec）。

## 10. 与现有模块的边界

| 模块 | 关系 |
|------|------|
| `internal/worker/commands.go` | 迁移后删除或仅留测试对照；逻辑进 registry builder |
| `internal/worker/server.go` | 仍 `exec`；入参仍为 `[]string` |
| `internal/models/engine.go` | `PipelineConfig` 不变；字段映射到 `Params` |
| `internal/api/task_handlers.go` | 移除或替换 `POST /tasks/run` 任意 `command`；无手调场景 |
| `internal/passive/gau.go` | Phase 2 删除 Server exec；改 `toolrun.Invoke("gau", …)` |
| `internal/workflow/pipeline_stage.go` | stage id 不变 |
| `docs/current/architecture.md` | 验收后增补「Tool Registry」小节 |

## 11. 实施阶段

### Phase 0 — 设计评审

- [x] 四项开放问题已关闭（§2.1）。
- 产出：认可的 `tools/*.yaml` 示例（`naabu.yaml`, `httpx.yaml`, `nuclei.yaml`）。

### Phase 1 — Registry + 等价 argv（无行为变更）

1. 新增 `internal/toolregistry`：解析 YAML、校验、`embed`。
2. 为 §6.4 全部工具编写 YAML；**黄金测试**：`registry.Render` 与 `Build*` 一致，**nuclei 除外** `-stats/-si` 两 flag（ intentional 删除）。
3. `toolguard.NewAllowlistFromRegistry`。
4. **不删** `Build*`；pipeline 仍走旧路径。

**验收**：`go test ./internal/toolregistry/...` 全绿；argv 黄金矩阵覆盖所有 preset。

### Phase 2 — toolrun + pipeline 切换 + gau 上 Worker

1. 新增 `internal/toolrun`（**强制** dispatch Worker；Server 不执行 registry 内 CLI）。
2. `createAndRunTask` → `toolrun.Invoke`；删除 `passive.RunGau` 的 Server `exec`。
3. 删除 pipeline 对 `worker.Build*` 的直接引用；废弃 `POST /tasks/run` 任意 command（删路由或 410）。
4. `Pipeline` 构造时注入 `Registry`。

**验收**：`go test ./internal/workflow/...`；现有 integration tests 无回归；外网 preset E2E（`external-scan-flow.spec.ts`）通过；gau stage 的 artifact 来自 Worker stdout。

### Phase 3 — 大结果

1. 流式 parser 调用路径；废弃 `readTaskStdout` 全量读。
2. 分页 API + 单元测试。
3. 前端 Task 详情按需加载（可 follow-up issue）。

**验收**：构造 >256KB 假 stdout integration test；内存不随文件线性增长（可用 `testing.AllocsPerRun` 粗测）。

### Phase 4 — 可选增强

- `ANCHOR_TOOLS_DIR` 热覆盖。
- `port_scan_tool` 配置化切换 naabu/rustscan。
- MCP 只读暴露 `list_tools` / `get_task_output`（独立提案）。
- 任务/artifact 30 天 TTL 清理 job（若尚未统一实现）。

## 12. 测试策略

| 层级 | 内容 |
|------|------|
| 单元 | YAML 解析错误、非法参数、preset 展开、allowlist 派生 |
| 黄金 | 每个 tool_id：旧 `Build*` vs `registry.Render` argv 完全一致 |
| 集成 | 现有 `pipeline_*_test.go`；mock runner 不写真实 subprocess |
| E2E | 外网 ScanModal preset 跑通；Runs 页 stage 与迁移前一致 |

## 13. 风险与缓解

| 风险 | 缓解 |
|------|------|
| YAML 与 Go 双源漂移 | Phase 1 黄金测试；CI 禁止新增 `Build*` 无 registry 对应 |
| Nuclei 行为回归 | `tools/nuclei.yaml` + `when` 规则单元测试 + 专项 integration test |
| gau 迁 Worker 后远程 Worker 未装 gau | health check / 部署文档列出 registry 二进制；stage fail-soft 或明确报错 |
| 过度抽象 | 非目标明确禁止 DAG YAML；复杂逻辑留 L4 |
| 安全：YAML 注入路径 | `path` 类型走 toolguard 元字符检查；禁止 shell 拼接 |
| 迁移周期长 | 分 Phase；Phase 1 可合并独立 PR |

## 14. 成功标准（Definition of Done）

1. 所有 §6.4 工具仅通过 `toolregistry` 生成 argv。
2. `toolguard` 与 registry 二进制集合一致（除显式系统扩展）。
3. 无 `readTaskStdout` 全量读 >256KB 文件的调用路径。
4. `docs/current/architecture.md` 已更新；本文件 `status: accepted`。
5. `docs/active/review/` 有简短验收记录（命令 + E2E 证据）。

## 15. 开放问题

无（见 §2.1）。

---

## 附录 A — 目录结构（拟新增）

```text
tools/
  naabu.yaml
  httpx.yaml
  nuclei.yaml
  ...
internal/toolregistry/
  registry.go      # Load, Get, Render
  schema.go        # ToolDef types
  validate.go
  embed.go
  registry_test.go
  golden_test.go
internal/toolrun/
  invoke.go
  artifact.go
  invoke_test.go
```

## 附录 B — `InvokeInput.Params` 与 `PipelineConfig` 映射（节选）

| tool_id | Param key | PipelineConfig 字段 |
|---------|-----------|---------------------|
| naabu | rate | NaabuRate |
| naabu | portRange | PortRange |
| httpx | rateLimit | HttpxRateLimit |
| subfinder | mode | SubfinderMode |
| nuclei | scanDepth | NucleiScanDepth |
| nuclei | rateLimit | NucleiRateLimit |

完整映射表在 Phase 1 实现时于 `toolregistry` 包注释或 `tools/README.md` 维护。
