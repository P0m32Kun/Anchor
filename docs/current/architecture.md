---
status: active
source_of_truth: true
owner: kun
last_updated: 2026-05-19
scope: runtime-baseline
---

# Current Architecture Baseline

This file describes the current repository baseline that agents should assume unless a task explicitly opts into an in-review design.

## System Shape

- Desktop client: Tauri 2.x shell hosting a React 18 + TypeScript frontend
- Local/remote service: Go application providing API, orchestration, and worker-facing endpoints
- Persistence: SQLite in WAL mode
- Realtime updates: SSE
- Scan execution: worker processes running external security tools (subfinder, dnsx, httpx, naabu, nmap, cdncheck, nuclei)
- Pipeline configuration: mode-driven (`external`/`internal`) tool selection, per-tool speed params (rate limit, threads, timeout), port range presets
- Global engine credentials: FOFA/Hunter/Quake API keys stored in `engine_credentials` table, configured via `/engines/keys`
- Vulnerability dictionary: `finding_templates` table stores knowledge entries (title, severity, summary, remediation) matched against findings at report time; seeded from repo JSON (`is_builtin=1`) or created in UI (`is_builtin=0`)
- Report generation: synchronous Markdown export only; findings are aggregated by matched dictionary entry (`ReportSection`) before rendering

## Baseline Workflow

The stable product narrative remains:

`目标输入 -> Scope Check -> 资产发现 -> Web 初筛 -> 人工验证 -> 报告导出`

实际执行管线（当前已实现）：

```
目标导入 → 分类 → (FOFA/Subfinder) → DNSx 解析 → CDN 过滤 → Naabu 端口扫描 → nmap -sV 服务指纹 → httpx Web 探活 → Nuclei 漏洞扫描
```

扫描模式由前端 `ScanModal` 选择：

- **外网扫描 (`external`)**：启用全部工具链（FOFA → Subfinder → DNSx → CDNCheck → Naabu → nmap -sV → HTTPX → Nuclei）
- **内网扫描 (`internal`)**：仅启用 Naabu → nmap -sV → HTTPX → Nuclei

各工具的速率限制、并发线程、超时参数在 `ScanModal` Step 2 中配置，通过 `POST /projects/{id}/scan` 的 `config` 字段传递。端口范围支持 top100 / top1000 / high-risk / full / custom 五种预设。

FOFA 凭证不再绑定到项目，而是从全局 `engine_credentials` 表读取。Hunter 和 Quake 通过独立的 `/engines/search` API 调用，结果统一为 `SearchResult` 格式。

### 多目标类型与 Company 目标自动展开

`PipelineConfig.runFlow` 按 `Target.Type` 分流到不同入口：

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

1. **二进制白名单** — 只允许预定义的工具名（`subfinder`, `dnsx`, `httpx`, `naabu`, `nmap`, `nuclei`, `cdncheck`, `git`, `sh`, `bash`）。检查基于 `filepath.Base`，因此 `/tmp/evil` 即使伪装成允许的名字也会被拒绝（basename 不在列表中），而 `/usr/local/bin/nuclei` 会被接受（basename `nuclei` 在白名单中）。
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

## What Is Not Baseline Yet

- `docs/refactoring-plan.md` is a backlog/refactor inventory, not the current product architecture.
- `docs/design/custom-nuclei-template-management.md` is an in-review design for custom Nuclei template management.

## How To Use This File

- Use this file for repo-level orientation.
- Use the implementation and tests to answer behavior questions.
- Use `docs/current/design/README.md` only when a task explicitly targets a proposal or review stream.

## Documentation Contract

If architecture changes materially, update this file first or in the same change set. Proposal documents should explain the delta from this baseline instead of redefining the entire system from scratch.
