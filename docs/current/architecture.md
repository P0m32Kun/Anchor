---
status: active
source_of_truth: true
owner: kun
last_updated: 2026-06-01
scope: runtime-baseline
---

# Current Architecture Baseline

This file describes the current repository baseline that agents should assume unless a task explicitly opts into an in-review design.

## System Shape

- Desktop client: Tauri 2.x shell hosting a React 18 + TypeScript frontend
- Local/remote service: Go application providing API, orchestration, and worker-facing endpoints
- Persistence: SQLite in WAL mode
- Realtime updates: SSE
- Scan execution: worker processes running external security tools (subfinder, dnsx, httpx, naabu, nmap, cdncheck, nuclei, spoor)
- Global engine credentials: FOFA/Hunter/Quake API keys stored in `engine_credentials` table, configured via `/engines/keys`
- Vulnerability dictionary: `finding_templates` table stores knowledge entries (title, severity, summary, remediation) matched against findings at report time; seeded from repo JSON (`is_builtin=1`) or created in UI (`is_builtin=0`)
- Report generation: synchronous Markdown export only; findings are aggregated by matched dictionary entry (`ReportSection`) before rendering

### 执行模型：资产驱动（非管线阶段）

**扫描执行是资产驱动模型，不是管线阶段模型。** 不存在固定的 P1→P2→P3→P4→P5 执行顺序。

核心循环：

```
发现资产 → DeriveEligibleWorks(资产类型) → 派生 Work(资产×动作) → 执行工具 → 输出解析 → 发现新资产 → 循环
```

关键组件（`internal/scanengine/`）：

| 组件 | 职责 |
|------|------|
| `core/rules.go` | `DeriveEligibleWorks()` — 根据资产类型 + Profile 规则派生 Work |
| `core/task.go` | `TaskAction` 枚举 + `ActionToTool` 映射 |
| `engine.go` | 主循环：`processNewAsset` → `tick` → `executeWork` → `onWorkComplete` → `processNewAsset` |
| `executor/` | 工具调用 + 输出解析器（httpx, katana, ffuf, nuclei, spoor） |
| `stageagg/` | **仅用于 UI 投影**（SSE 进度展示），不影响执行顺序 |

资产类型 → 动作派生规则（`core/rules.go`）：

| 资产类型 | 派生的动作 |
|---------|-----------|
| `AssetSubdomain` | SubdomainEnum, DNSResolve, CDNCheck |
| `AssetIP` | DNSResolve, CDNCheck, PortScan（alive 且非 CDN） |
| `AssetIPPort` | ServiceFingerprint |
| `AssetHTTPService` | HTTPXFingerprint, KatanaCrawl, FFUFBrute, NucleiScan |
| `AssetHTTPPath` | KatanaCrawl, NucleiScan, SpoorScan |

收敛机制：`idle_timeout`（无新资产）→ `wind_down`（仅允许 Nuclei/httpx）→ `stopped`。

通过 `ANCHOR_SCAN_ENGINE=1` 环境变量启用。详见 `docs/superpowers/specs/2026-05-29-asset-driven-scan-engine-design.md`。

## Baseline Workflow

产品叙事：

`目标输入 -> Scope Check -> 资产发现 -> Web 初筛 -> 人工验证 -> 报告导出`

### 扫描配置（Profile）

> **注意：以下是扫描配置（Profile），决定哪些工具可用和默认参数，不是执行顺序。**
> 实际执行由资产驱动引擎按 `DeriveEligibleWorks()` 规则自动调度，见上方「执行模型」。

**内网 (`internal`)** — 以主动发现为主，Profile 规则允许全部动作。

**外网 (`external`)** — `DefaultExternalPipelineConfig()` + `buildConfigForMode("external")` 提供默认参数：

| 配置维度 | 内容 | 涉及工具 |
|---------|------|----------|
| 被动资产 | 搜索引擎 + 证书/历史 URL | FOFA、Hunter、Quake、`passive_cert`（crt.sh）、`passive_url`（gau） |
| 解析降噪 | 子域 + DNS + CDN | Subfinder（默认 passive）、dnsx、cdncheck |
| 受限主动 | 默认 top100、降 Naabu 速率 | nmap alive → Naabu → nmap -sV；`skip_portscan_on_cdn_host` 为 true 时不扫 CDN IP |
| Web 扩面 | 探活 + 爬虫/目录 | httpx → Katana（`-jc` JS 端点）→ ffuf（`ffuf_tier` small/medium/off）→ Spoor（JS 静态分析）→ httpx_2 |
| 精 POC | 指纹驱动 | Nuclei workflow 默认；`nuclei_require_fingerprint` 为 true 时无指纹跳过 |

```
company → passive_search(FOFA+Hunter+Quake) → domain/ip 分流 → 各自进入资产驱动循环
domain  → subfinder/DNS/CDN → 端口 → httpx/Katana/ffuf/Spoor/Nuclei（由资产类型自动派生）
url     → httpx/Katana/ffuf/Spoor/Nuclei（由资产类型自动派生）
```

扫描模式由前端 `ScanModal` 选择；外网模式加载 `DEFAULT_EXTERNAL_PIPELINE_CONFIG`（`port_range: top100`、`nuclei_scan_depth: workflow` 等）。

各工具的速率限制、并发线程、超时参数在 `ScanModal` Step 2 中配置，通过 `POST /projects/{id}/scan` 的 `config` 字段传递。端口范围支持 top100 / top1000 / high-risk / full / custom 五种预设。

引擎凭证在全局 `engine_credentials` 表配置（`/engines/keys`）。被动搜索在 `runPassiveSearch` 中并行调用 FOFA、Hunter、Quake（fail-soft，单引擎失败不阻断）。Engines 页手动搜索仍保留，供扫描外调研。

### 多目标类型与 Company 目标自动展开

> **注意：以下是 Legacy Pipeline 的目标分流逻辑（`PipelineConfig.runFlow`）。**
> 在资产驱动引擎下，所有目标统一作为 `AssetSubdomain` 种子资产注入，由 `DeriveEligibleWorks()` 自动派生后续动作。

Legacy Pipeline 按 `Target.Type` 分流：

| 目标类型 | 入口 |
| --- | --- |
| `domain` | Subfinder → DNSx → CDN → Naabu → nmap -sV → httpx/Nuclei |
| `ip` | CDNCheck → Naabu → nmap -sV → httpx/Nuclei |
| `cidr` | Naabu → nmap -sV → httpx/Nuclei |
| `url` | httpx → Nuclei（仅 Web） |
| `company` | FOFA `org/cert/title` 三维搜索 → 展开为新 Target（domain/ip）→ 路由到对应 flow |

Company 目标在 `runCompanyFlow` 中调用 FOFA：每个查询返回的资产被去重后作为 `source="fofa"` 的新 Target 写入 DB，再分别进入 domain/ip flow。`FOFA_BASE_URL` 环境变量可覆盖默认 `https://fofa.info` 用于 E2E mock。

### Nuclei 分层扫描策略

`PipelineConfig.NucleiScanDepth` 控制 Nuclei 扫描方式，用户在 ScanModal Step 2 通过「Nuclei 扫描策略」面板选择：

| 模式 | 命令行 | 适用场景 |
| ---- | ------ | -------- |
| `tags`（默认） | `-tags <fingerprint-tags>` | 广度扫描，按 httpx 指纹精确匹配模板 |
| `workflow` | `-w /opt/rbkd-templates/workflows` | 精确扫描，使用预定义 workflow 串联指纹检测和漏洞利用 |
| `both` | `-w ... -tags ...` | 综合扫描，workflow + tags 双重检测，覆盖最全 |

Workflow 模板来自 [RBKD-SEC/RBKD-templates](https://github.com/RBKD-SEC/RBKD-templates)，由启动时 Git 同步落盘到 `/opt/rbkd-templates`（镜像 build-time clone 作离线兜底）。

### 团队内置资源

三个 RBKD-SEC public 仓库在 **Server + Worker** 启动时自动同步，注册为 DB 只读行；用户自定义资源仍走现有 UI CRUD。详细设计见 `docs/superpowers/specs/2026-05-19-builtin-assets-design.md`。

**仓库与磁盘路径**

| 仓库 | 默认路径 | Seed 产物 |
|------|----------|-----------|
| [RBKD-SEC/dict](https://github.com/RBKD-SEC/dict) | `/opt/dict` | 多条 `builtin:` + 相对路径字典行 |
| [RBKD-SEC/RBKD-templates](https://github.com/RBKD-SEC/RBKD-templates) | `/opt/rbkd-templates` | `builtin:rbkd-templates` nuclei 源行 |
| [RBKD-SEC/finger](https://github.com/RBKD-SEC/finger) | `/opt/finger/finger.json` | `builtin:rbkd-finger` httpx 指纹行 |

`internal/builtin.SyncAll()` 负责 clone/pull（`ANCHOR_BUILTIN_SYNC=off` 跳过；失败 fail-soft，保留上次落盘）。环境变量见 `internal/builtin/config.go`。

**启动顺序**

```text
Server NewServer():
  1. builtin.SyncAll()
  2. dictMgr.SeedBuiltin(ANCHOR_BUILTIN_DICT_ROOT)
  3. httpxFpMgr.SeedBuiltin(/opt/finger)
  4. nucleiCustomMgr.SeedBuiltin()   // 仅 DB 行，不 clone 到 dataDir

Worker runWorker():
  1. builtin.SyncAll()
  2. syncSources() 时按 DB enabled 管理 RBKD symlink
  3. bundle sync 仅用户自定义 nuclei 源（跳过 builtin=1）
```

**双轨 UI**

| | 团队内置 (`builtin=1`) | 我的自定义 |
|--|------------------------|------------|
| 列表 | 标签「内置」+ commit | 现有 CRUD |
| 编辑/删除 | 禁止 | 允许 |
| 启用开关 | `PATCH .../enabled` | 字典/指纹走常规 PATCH；nuclei 源走 `PATCH /sources/{id}` |

扫描侧仅 `enabled=1` 的资源参与：ffuf 字典下拉、httpx `-cff` 合并、nuclei workflow 路径。

**Worker RBKD symlink**

内置 nuclei 源 **不走 bundle**。Worker 在 `syncSources()` 中对 `builtin:rbkd-templates` 调用 `builtin.ApplyRBKDNucleiSymlink`：

| `enabled` | 行为 |
|-----------|------|
| `true` | 创建/刷新 `~/nuclei-templates/RBKD-templates` → `/opt/rbkd-templates` |
| `false` | **移除** symlink（若存在）；不删除 `/opt/rbkd-templates` |

禁用内置 = tags 与 workflow 均不加载 RBKD 树。因 nuclei `-tags` 搜索整个模板根，仅靠 DB 不列入 workflow 路径不足以排除 tags，必须通过不创建 symlink 实现。

**Nuclei tags 搜索范围**

`-tags` 在 `~/nuclei-templates/` **全树**生效（官方模板 + RBKD 子目录 + 用户 bundle 源）。RBKD 模板与官方模板同等参与 tag 匹配；内置禁用时无 symlink，RBKD 不参与 tags。workflow 模式仍走 `customWorkflowPaths()` → `{install_path}/workflows/{tag}.yaml`。

### Nuclei 速率与并发控制

`PipelineConfig` 暴露三个 Nuclei 速率字段，用户在 ScanModal Step 2 → Nuclei 区域配置：

| 字段 | Nuclei flag | 默认 | 用途 |
| ---- | ----------- | ---- | ---- |
| `nuclei_rate_limit` | `-rl` | 100 rps | 每秒请求数（常规限速） |
| `nuclei_rate_limit_per_min` | `-rlm` | 0（禁用） | 每分钟请求数（防止账号锁定/告警） |
| `nuclei_concurrency` | `-c` | 25 | 并行模板/主机数 |

扫描内网敏感目标（认证页面、ICS/SCADA、网络设备）时，建议将 `nuclei_rate_limit_per_min` 设为 30 以下、`nuclei_concurrency` 压到 1-5，避免触发账号锁定。

### 资源治理（ResourceGovernor）

`internal/worker/resource_governor.go` 提供系统级内存/CPU 阈值控制，避免长扫把本机拖死。`Runner.Run`（API 服务器侧任务入口）与 `WorkerServer.executeTask`（远端 worker 侧任务执行）在启动子进程之前都会调用 `governor.Acquire(ctx)`：

- 内存使用率 ≥ `MemoryThresholdPct` → 按 `MemoryPollInterval` 节奏轮询直到水位回落，相当于新任务排队。
- CPU 使用率 ≥ `CPUThresholdPct` → 一次性 `time.Sleep(CPUDelay)` 后放行，相当于入队速率减半。
- 采样失败（gopsutil 报错）→ fail-open,放行任务,避免误阻塞。
- `ctx` 取消时立即返回 `ctx.Err()`,任务被标记失败。

阈值通过环境变量配置，单位与上游工具一致（百分比即百分比，毫秒即毫秒，**代码内不做单位转换**）：

| 变量 | 默认 | 含义 |
| ---- | ---- | ---- |
| `ANCHOR_GOVERNOR_ENABLED` | `true` | 关掉则 `Acquire` 直接放行 |
| `ANCHOR_GOVERNOR_MEM_PCT` | `85` | 内存阈值百分比 (0-100) |
| `ANCHOR_GOVERNOR_CPU_PCT` | `80` | CPU 阈值百分比 (0-100) |
| `ANCHOR_GOVERNOR_POLL_MS` | `1000` | 内存阻塞时的轮询间隔(毫秒) |
| `ANCHOR_GOVERNOR_CPU_DELAY_MS` | `500` | CPU 超阈值时的固定延迟(毫秒) |

系统级指标采样依赖 `github.com/shirou/gopsutil/v3`。`ResourceSampler` 接口允许测试时注入 fake 实现。

### 漏洞辞典（FindingTemplate）

`finding_templates` 表已从「字段覆盖工具」升级为「漏洞辞典」。每个词条代表一类已知漏洞，包含 title、severity、summary、remediation 四个可覆盖字段，以及一个 `match_keys` 字符串数组用于匹配 findings。

**匹配逻辑（两级精确匹配）**

`GetFindingTemplateForFinding(sourceTool, sourceRuleID, title)` 按以下优先级查找启用的词条：

1. **Tier 1 — source_ruleID 匹配**：遍历该工具的全部启用词条，检查 `match_keys` 中是否精确包含 `finding.SourceRuleID`。
2. **Tier 2 — title 匹配**：若 Tier 1 未命中，检查 `match_keys` 中是否精确包含 `finding.Title`（兜底）。

一个词条可挂多个 `match_keys`（chip 输入），因此同一漏洞类型可以覆盖不同工具报告的不同 ruleID 或 title。

**来源与版本管理**

| 字段 | 含义 |
|------|------|
| `is_builtin` | `true` = 来自仓库种子 JSON；`false` = UI 创建 |
| `user_modified` | `true` = 内置词条被本地编辑过，阻止自动覆盖 |
| `builtin_payload` | 最新上游版本的 JSON，用于「上游有更新」提示和「接受上游」操作 |

**报告渲染时的字段覆盖**

模板字段非空时优先使用模板值，空时自动回退到 finding 自身值：

| 字段 | 回退行为 |
|------|---------|
| title | 模板空 → 使用 `finding.Title` |
| severity | 模板空 → 使用 `finding.Severity` |
| summary | 模板空 → 使用 `finding.Summary` |
| remediation | 模板空 → 使用 `finding.Remediation` |

### 报告渲染（按词条聚合 Sections）

报告不再按单个 finding 平铺渲染，而是先按词条聚合为 `ReportSection`：

- **命中词条**：同一 `FindingTemplate` 下的所有 findings 合并为一个 Section，标题和描述只出现一次，受影响资产以表格形式列出多行。
- **未命中**：每个 finding 单独成一个 Section，标题使用原始 finding 值，描述和修复建议区域提示「该漏洞类型尚未在辞典中维护」。

Sections 按 severity 倒序排列；同级 severity 时命中词条排在未命中前面。

报告生成是同步的：前端点击「导出报告」直接调用 `handleExportReportMD`，后端即时生成 Markdown 并返回下载。异步报告流程（Report 模型、后台生成、状态轮询）已完全删除。

### 工具执行白名单（Allowlist）

`internal/toolguard/allowlist.go` 提供外部二进制执行的集中管控。所有 `exec.Command` / `exec.CommandContext` 调用点在创建子进程之前都经过 `Allowlist.Validate(binary, args)` 检查：

1. **二进制白名单** — 只允许预定义的工具名（`subfinder`, `dnsx`, `httpx`, `naabu`, `nmap`, `nuclei`, `cdncheck`, `spoor`, `git`, `sh`, `bash`）。检查基于 `filepath.Base`，因此 `/tmp/evil` 即使伪装成允许的名字也会被拒绝（basename 不在列表中），而 `/usr/local/bin/nuclei` 会被接受（basename `nuclei` 在白名单中）。
2. **参数安全检查** — 拒绝任何包含 shell 元字符（`;|&><`$(){}[]\n\r`）的参数。`exec.Command` 本身已规避 shell 注入，这层检查是纵深防御：万一参数在未来被拼接到 shell 字符串中，元字符不会穿透。

接入点覆盖全部 5 个 `exec.Command` 调用文件：

| 文件 | 检查位置 |
|------|----------|
| `internal/worker/worker.go` | `Runner.Run` 本地回退执行前 |
| `internal/worker/server.go` | `WorkerServer.executeTask` 子进程启动前 |
| `internal/health/health.go` | `getVersion` / `getNucleiTemplatePath` 调用前 |
| `internal/cdn/detector.go` | `CheckIP` / `FilterCDNIPs` 调用前 |
| `internal/nuclei/custom/git.go` | `ExecCloner.Clone` 调用前 |

`Allowlist.Allow(name)` 支持运行时扩展（测试和自定义工具注册）。新增工具时强制走注册流程：先 `Allow()` 再执行。

## 资产驱动扫描引擎（ScanEngine）

**状态**：已实现，通过 `ANCHOR_SCAN_ENGINE=1` 环境变量启用。**这是当前唯一的扫描执行模型。**

### 核心概念

扫描执行为 **资产图 + Work(资产×动作) + 属性门控 + 收敛状态机**。不存在固定阶段顺序。

| 概念 | 说明 |
|------|------|
| DiscoveryAsset | 发现层 DTO，含 Type/Value/Depth/Attrs |
| Work (ScanWorkItem) | 调度真相：`(run_id, asset_id, action)` 唯一 |
| ActionRule | 动作启用条件：Enabled + MaxDepth + Precondition |
| EngineState | run 级状态：`running` → `wind_down` → `stopped` |

### 包结构

```
internal/scanengine/
  core/           DiscoveryAsset, TaskAction, AssetAttrs, DeriveEligibleWorks
  work/           Store (TryClaim/MarkDone/AllTerminal)
  queue/          PriorityQueue (high/medium/low)
  dedup/          RunDedup (run-level normalized value dedup)
  executor/       Executor interface + ToolExecutor + httpx/nuclei/katana/ffuf/spoor parsers
  stageagg/       Aggregator (Work → Stage projection)
  engine.go       ScanEngine main loop
```

### 深度控制

- `MaxDiscoveryDepth = 2`（全局默认）
- katana/ffuf `MaxDepth = 1`（仅 depth ≤ 1 执行）
- 子域枚举 `MaxDepth = 1`

### 执行与同步

- `BatchSize` 同时作为最大在途 Work 数和并发信号量容量，避免 scheduler 一次性放出过多子进程。
- `PriorityQueue` 对 `WorkID` 做去重；同一 Work 在被 `Pop()` 之前不会重复入队。
- `Run()` 在取消上下文时会先等待所有 in-flight work 完成，再把引擎状态切到 `stopped`。
- `stageagg.Aggregator` 用互斥锁串行化 stage 投影更新，避免并行 completion 回调互相踩写。
- `executor.Executor` 通过接口注入，便于测试时替换真实工具执行器。

### 收敛规则

| 条件 | 行为 |
|------|------|
| `time.Since(lastNewAsset) > idleTimeout` | → `wind_down` |
| `wind_down` + 队列空 + 全 Work 终态 | → `stopped` |
| wind_down 期间 | 仅允许 Nuclei、httpx |

### 双轨可观测性

- **真相**：`scan_work_items` + `scan_tasks` + run metrics
- **UI 投影**：`pipeline_run_stages` + `pipeline_stage_change` SSE（由 `stageagg.Aggregator` 生成，仅用于前端进度展示，**不影响执行逻辑**）
- Stage 是 UI 分组标签（crawl/vuln/httpx 等），不是执行阶段；同一 stage 可多轮 `running`

### API

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/projects/{id}/pipeline/runs/{runId}/metrics` | 引擎状态 + Work 计数 |
| GET | `/projects/{id}/pipeline/runs/{runId}/works` | Work 列表 |
| GET | `/assets/{id}/works?run_id=` | 单资产 Work 时间线 |

### 全局域名排除列表

**状态**：已实现，默认启用。

为防止爬虫在目标网站中发现外部链接时将公共服务域名误判为目标资产，系统提供了全局域名排除列表。

#### 工作原理

| 组件 | 说明 |
|------|------|
| `internal/exclude/defaults.go` | 内置默认排除域名列表（github.com, apache.org, w3.org 等） |
| `internal/exclude/exclude.go` | 排除管理器，支持内存缓存和域名变更回调 |
| `excluded_domains` 表 | 持久化存储，区分内置（builtin=1）和用户自定义（builtin=0） |
| `internal/api/exclude_handlers.go` | REST API 接口 |
| `internal/scanengine/engine.go` | 在 `processNewAsset` 中集成过滤 |

#### 过滤时机

域名排除检查在以下时机执行：

1. **scanengine.processNewAsset**: 每当发现新资产时，检查其域名是否在排除列表中
2. **支持 URL 解析**: 对于 URL 类型的资产，自动提取域名进行检查
3. **子域名匹配**: `example.com` 会匹配 `api.example.com`、`sub.example.com` 等

#### API 接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/excluded-domains` | 查看所有排除域名（内置 + 自定义） |
| GET | `/excluded-domains/defaults` | 查看内置默认列表 |
| POST | `/excluded-domains` | 添加自定义域名 |
| POST | `/excluded-domains/batch` | 批量添加域名 |
| DELETE | `/excluded-domains/{domain}` | 删除自定义域名（内置不可删） |
| POST | `/excluded-domains/reset` | 重置为默认列表 |
| GET | `/excluded-domains/check?domain=` | 检查域名是否被排除 |

#### 数据库

```sql
CREATE TABLE excluded_domains (
    id TEXT PRIMARY KEY,
    domain TEXT NOT NULL UNIQUE,
    reason TEXT NOT NULL DEFAULT '',
    builtin INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

#### 启动顺序

```text
Server NewServer():
  1. SeedDefaultExcludedDomains()  // 种子化内置域名
  2. LoadCustomExcludedDomains()   // 加载用户自定义域名到内存
```

详细文档见 `docs/features/exclude-domains.md`。

## What Is Not Baseline Yet

- `docs/refactoring-plan.md` is a backlog/refactor inventory, not the current product architecture.
- `docs/design/custom-nuclei-template-management.md` is an in-review design for custom Nuclei template management.

## How To Use This File

- Use this file for repo-level orientation.
- Use the implementation and tests to answer behavior questions.
- Use `docs/current/design/README.md` only when a task explicitly targets a proposal or review stream.

## Documentation Contract

If architecture changes materially, update this file first or in the same change set. Proposal documents should explain the delta from this baseline instead of redefining the entire system from scratch.
