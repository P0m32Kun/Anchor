---
status: active
source_of_truth: true
owner: kun
last_updated: 2026-06-03
scope: runtime-baseline
---

# Current Architecture Baseline

This file describes the current repository baseline that agents should assume unless a task explicitly opts into an in-review design.

## System Shape

- Web frontend: React 18 + TypeScript, served by Nginx（静态文件 + `/api/` 反向代理到 Server）
- Go service: Server 提供 API、编排、Worker 对端点；Worker 执行外部安全工具
- Persistence: SQLite in WAL mode
- Realtime updates: SSE
- Scan execution: worker processes running external security tools (subfinder, dnsx, httpx, naabu, nmap, cdncheck, nuclei, spoor)
- Global engine credentials: FOFA/Hunter/Quake API keys stored in `engine_credentials` table, configured via `/engines/keys`
- Vulnerability dictionary: `finding_templates` table stores knowledge entries (title, severity, summary, remediation) matched against findings at report time; seeded from repo JSON (`is_builtin=1`) or created in UI (`is_builtin=0`)
- Report generation: synchronous Markdown export only; findings are aggregated by matched dictionary entry (`ReportSection`) before rendering

### 部署架构

三个 Docker 镜像，通过 docker-compose 编排：

| 镜像 | Dockerfile | 职责 |
|------|-----------|------|
| `anchor-server` | `Dockerfile.server` | Go API 服务（从 GitHub Release 下载预编译二进制） |
| `anchor-worker` | `Dockerfile.worker` | 安全工具 + Go Worker（预装所有安全扫描工具） |
| `anchor-frontend` | `Dockerfile.frontend` | Nginx 静态 serve React 构建产物 + `/api/` 反向代理 |

**三种部署模式**（通过 `install.sh` 交互式选择）：

| 模式 | compose 文件 | 适用场景 |
|------|-------------|---------|
| Server Only | `docker-compose.server.yml` | VPS 部署，Worker 远程连接 |
| Worker Only | `docker-compose.worker.yml` | 远程扫描节点，连接已有 Server |
| Server+Worker | `docker-compose.yml` | 同机完整部署（VPS/内网单机） |

**Compose 与镜像职责分离**（2026-06-03）：

| 用途 | compose 文件 | 镜像来源 |
|------|-------------|---------|
| **用户部署** | `docker-compose.yml` / `.server.yml` / `.worker.yml` | 仅 `image`（阿里云 ACR 三镜像），**无 `build`** |
| **E2E 自动化** | `docker-compose.e2e.yml` | 本地 `build`（`Dockerfile.*-fast`）+ 内嵌 rangefield + fofa-mock |
| **E2E 手动迭代** | `docker-compose.e2e-local.yml` | `Dockerfile.*-fast` → `anchor-*:local`；frontend 仍拉 ACR |

详细操作见 [`deployment.md`](deployment.md)（客户）与 [`e2e-testing.md`](e2e-testing.md)（开发者）。

**镜像分发**：
- GitHub Release：Go 二进制（`anchor-linux-amd64`、`anchor-linux-arm64`），tag 推送时由 CI 自动构建
- 阿里云 ACR：Docker 镜像（`crpi-wthv8jhah5ufmzlr.cn-hangzhou.personal.cr.aliyuncs.com/p0m32kun/`），Release 完成后 CI 自动推送；**用户侧只 pull，不在部署 compose 里 build**
- `install.sh` / `make up`：从 ACR 拉取三镜像后 `compose up -d`

**Docker 镜像 tag 策略**（2026-06-03）：
- `docker-push.yml` 在 Release workflow 成功后 checkout **触发 Release 的 tag/ref**（例如 `v0.2.0`），而非默认分支 HEAD
- 构建 server/worker 时传入 `RELEASE_VERSION=<tag>`，`Dockerfile.server` / `Dockerfile.worker` 从 `releases/download/<tag>/` 下载二进制；避免发布镜像内嵌 `latest` 资产与 tag 不一致
- 推送到 ACR 时同时打版本 tag（`anchor-server:v0.2.0`）与 `latest`；frontend 同步版本 tag（构建物来自 checkout 的源码树）
- 手动触发 `workflow_dispatch` 时需指定 `release_version` 输入（tag 或 `latest`）

**Worker 部署网络**：
- **出站为主**：Worker 通过 `--core-url` 连接 Server（长轮询拉任务、心跳、上报结果），无需 Worker 公网 IP
- **Server 入站 Worker（可选）**：运行中任务的实时 stdout/stderr 由 Server 代理到 Worker 注册的 `endpoint`（`proxyWorkerTaskOutput`）；若不需要 UI 实时日志，可不暴露 Worker HTTP
- **同机 compose**：`docker-compose.yml` 中 Worker 使用服务名 `worker` 作为 endpoint；Server 与 Worker 使用**独立** named volume（`anchor-server-data` / `anchor-worker-data`），避免 SQLite 与 Worker 本地状态互相覆盖

**多 Worker 任务分配**（`internal/worker/dispatcher.go`）：

Server 侧 `Runner.Run` 将 ScanTask 派发到远程 Worker HTTP `POST /tasks`。调度策略：

1. **最少负载优先**：`load = DB running tasks + server in-flight 计数`（避免并发 dispatch 竞态）
2. **同负载轮询**：负载相同时 round-robin，新注册 worker（load=0）与空闲 peer 平等参与
3. **容量上限**：尊重 `worker_nodes.max_concurrency`（默认 10），满载 worker 跳过
4. **故障转移**：不可达时 `MarkWorkerOffline` 并尝试下一个 worker
5. **Worker 侧并发**：`WorkerServer` 尊重 `ANCHOR_WORKER_MAX_CONCURRENCY`（默认 10）；满载 HTTP 503，Server 改派而不标记 offline
6. **UI 负载**：`GET /workers` 返回 `running_tasks` / `max_concurrency`；Workers 页展示 `N / M`

本地双 worker 测试：`docker-compose.e2e-multi-worker.override.yml` 或 `make test-e2e-multi-worker-up`。验收：`GET /runs/{runId}/tasks` 按 `worker_id` 分组；E2E：`multi-worker-dispatch.spec.ts`。

**API Token**：安装时写入 `.env` 的 `ANCHOR_API_TOKEN`；轮换时修改 `.env` 并重启 compose。`install.sh` 不将完整 Token 打印到终端（仅提示已写入 `.env`）。

**多平台支持**：
- Docker 镜像支持 `linux/amd64`（VPS/PC）和 `linux/arm64`（Mac M1/M2）
- 通过 GitHub Actions 使用 QEMU + Docker Buildx 构建多平台镜像
- 用户拉取时自动选择匹配的平台架构

**前端 API 配置**：Nginx 反向代理 `/api/` → `server:17421`，前端默认 `apiBase="/api"`，无需手动配置 API 地址。

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

扫描模式由前端 `ScanModal` 选择两种 preset：**外网**、**内网**。Run 摘要 API：`GET /projects/{id}/pipeline/runs/{runId}/summary`（新发现数 + 阶段覆盖率）。资产血缘 API：`GET /assets/{id}/lineage?run_id=`。部署 preset 默认值：`GET /scan/defaults`。

各工具的速率限制、并发线程、超时参数在 ScanModal Step 2「高级选项」折叠面板中配置，通过 `POST /projects/{id}/scan` 的 `config` 字段传递。端口范围支持 top100 / top1000 / high-risk / full / custom 五种预设。

引擎凭证在全局 `engine_credentials` 表配置（`/engines/keys`）。被动搜索在 `runPassiveSearch` 中并行调用 FOFA、Hunter、Quake（fail-soft，单引擎失败不阻断）。Engines 页手动搜索仍保留，供扫描外调研。

### 目标输入与资产驱动

所有目标统一作为 `AssetSubdomain` 种子资产注入，由 `DeriveEligibleWorks()` 自动派生后续动作。Company 目标通过被动搜索引擎（FOFA/Hunter/Quake）展开为 domain/ip 子目标后注入。

### Nuclei 分层扫描策略

`PipelineConfig.NucleiScanDepth` 控制 Nuclei 扫描方式，用户在 ScanModal Step 2 通过「Nuclei 扫描策略」面板选择：

| 模式 | 命令行 | 适用场景 |
| ---- | ------ | -------- |
| `tags`（默认） | `-tags <fingerprint-tags>` | 广度扫描，按 httpx 指纹精确匹配模板 |
| `workflow` | `-w /opt/rbkd-templates/workflows` | 精确扫描，使用预定义 workflow 串联指纹检测和漏洞利用 |
| `both` | `-w ... -tags ...` | 综合扫描，workflow + tags 双重检测，覆盖最全 |

Workflow 模板来自 [RBKD-SEC/RBKD-templates](https://github.com/RBKD-SEC/RBKD-templates)，由启动时 Git 同步落盘到 `/opt/rbkd-templates`（镜像 build-time clone 作离线兜底）。

### Nuclei Code Templates（代码模板）

Worker 执行 Nuclei 时默认启用 `-code` 和 `-dut=false` 标志（`internal/worker/commands.go`）：

| 标志 | 作用 |
|------|------|
| `-code` | 启用 Nuclei 代码模板执行，允许运行 `.code.yaml` 模板（内置 JavaScript/Go 逻辑的自定义模板） |
| `-dut=false` | 禁用模板签名验证/更新检查，允许使用未签名的 RBKD 模板 |

这两个标志使 RBKD-SEC 团队编写的代码模板（含复杂检测逻辑的自定义模板）能够正常运行，而不被 Nuclei 的默认签名验证阻止。代码模板在模板仓库（RBKD-templates）中以 `.code.yaml` 后缀标识。

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
  2. syncSources() 管理 RBKD symlink（仅 builtin 行）
```

**Templates UI（Nuclei）**

| 能力 | 说明 |
|------|------|
| 列表 | 仅 `builtin:rbkd-templates` |
| 启用开关 | `PATCH .../enabled` |
| 模板更新 | `ANCHOR_BUILTIN_SYNC` + `/opt/rbkd-templates` 挂载；无 UI git/upload/publish |

扫描侧仅 `enabled=1` 的资源参与：ffuf 字典下拉、httpx `-cff` 合并、nuclei workflow 路径。

**Worker RBKD symlink**

Worker 在 `syncSources()` 中对 `builtin:rbkd-templates` 调用 `builtin.ApplyRBKDNucleiSymlink`：

| `enabled` | 行为 |
|-----------|------|
| `true` | 创建/刷新 `~/nuclei-templates/RBKD-templates` → `/opt/rbkd-templates` |
| `false` | **移除** symlink（若存在）；不删除 `/opt/rbkd-templates` |

禁用内置 = tags 与 workflow 均不加载 RBKD 树。因 nuclei `-tags` 搜索整个模板根，仅靠 DB 不列入 workflow 路径不足以排除 tags，必须通过不创建 symlink 实现。

**Nuclei tags 搜索范围**

`-tags` 在 `~/nuclei-templates/` **全树**生效（官方模板 + RBKD 子目录）。RBKD 模板与官方模板同等参与 tag 匹配；内置禁用时无 symlink，RBKD 不参与 tags。workflow 模式仍走 `{install_path}/workflows/{tag}.yaml`（RBKD-templates symlink）。

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

- `ComputeLimits(seedCount)` 驱动全局并发；`BatchSize` 仅作测试 override。
- `PopFairStaged` + `StageRank` 保证 DNS/Port 先于 httpx/nuclei。
- `PriorityQueue` 对 `WorkID` 去重；batch work 与单 asset work 共用队列。
- `Run()` 在取消上下文时会先等待所有 in-flight work 完成，再把引擎状态切到 `stopped`。
- `stageagg.Aggregator` 用互斥锁串行化 stage 投影更新，避免并行 completion 回调互相踩写。
- `executor.Executor` 通过接口注入，便于测试时替换真实工具执行器。

### 收敛规则

| 条件 | 行为 |
|------|------|
| `time.Since(lastNewAsset) > idleTimeout` | → `wind_down` |
| `wind_down` + 队列空 + 全 Work 终态 | → `stopped` |
| wind_down 期间 | 仅允许 Nuclei、httpx |
| `AbsoluteTimeout == 0`（默认） | 不设硬超时；长 run 正常 |
| Server 重启 | `recovery.RecoverOrphanRuns` 将 orphan run 标 failed |

### 批量调度（BatchWork + Pool）

**状态**：P0–P2 已实现（2026-06-17，`docs/design/batch-scan-scheduling/`）。

扫描工具按 **输入同质性** 分三层批量，而非 per-asset 一条 work：

| 层级 | Action | 机制 | 典型 batch |
|------|--------|------|------------|
| Tier1 | DNS / CDN / Port / Subfinder | `pool.Pool` / `domainpool` | 50–100 行/CLI |
| Tier2 | httpx / nmap / nuclei | `httpPool` / `IPPortAggregator` / `NucleiTagBuckets` | 20–100 URL 或 1 IP×ports |
| Tier3 | katana / ffuf / spoor / 单点 nuclei | 仍 1 URL/work | 1 |

**BatchWorkItem**（`scan_work_items` v39+）：`batch_mode=1` 时 `input_file` + `member_asset_ids` JSON 表示一批 CLI；`asset_id` 为 `batch:{workID}`。flush 时写入 DB 并入 `PriorityQueue`。

**阶段调度**（取代旧 ClassifyAction 优先级）：

```
PopFairStaged → 高 stage pending 时低 stage 不 pop
ComputeLimits(seedCount) → 动态 sem 宽度
IPThrottler → executeWork 前按 host/IP 限流
```

**Nuclei**：httpx technologies → `scan.config.yaml` 的 `nuclei_tech_routing` 分桶；`noise_level=low` 无 tech 则 skip；禁止宽 tag 混批。

**Run 结束**：`finalizeRun` flush 所有 Pool → `drainUntilQuiescent` 排空队列。

包结构补充：

```
internal/scanengine/
  pool/           通用池、httpx 候选、IP port 聚合、nuclei 分桶
  queue/fair.go   PopFair / PopFairStaged
  scheduler/      ComputeLimits、IPThrottler、SeedBucketKey
  recovery/       orphan run 恢复
  engine_tier1.go / engine_tier2.go  池化接线
internal/scanconfig/nuclei_routing.go  tech 路由表
```

**1067 domain 外网预期**：tool call ~150–300（对比改造前 ~5791）；work item <1000。

### 双轨可观测性

- **真相**：`scan_work_items` + `scan_tasks` + run metrics
- **UI 投影**：`pipeline_run_stages` + `pipeline_stage_change` SSE（由 `stageagg.Aggregator` 生成，仅用于前端进度展示，**不影响执行逻辑**）
- Stage 是 UI 分组标签（crawl/vuln/httpx 等），不是执行阶段；同一 stage 可多轮 `running`
- **前端展示**：`frontend/src/pages/RunsPage.tsx` 的运行详情是「运行观察台」。主视图展示引擎状态、Work 计数、扫描动作进度和 Work Items 明细；不再把 stage 渲染为严格线性 Pipeline 时间线。

### API

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/projects/{id}/pipeline/runs/{runId}/metrics` | 引擎状态 + Work 计数 |
| GET | `/projects/{id}/pipeline/runs/{runId}/works` | Work 列表（`page`/`page_size` 分页，默认 50） |
| GET | `/projects/{id}/pipeline/runs/{runId}/tool-calls` | 工具调用日志（分页） |
| GET | `/assets/{id}/works?run_id=` | 单资产 Work 时间线 |

`scan_work_items.task_id` 关联 `scan_tasks.id`，前端 Work 日志优先跳转 task 详情。

### 项目 Scope（排除规则 only）

**状态**：2026-06-03 起全局 **exclusion-only**（见 [`asset-driven-remediation-design.md`](asset-driven-remediation-design.md)）。

| 层级 | 行为 |
|------|------|
| 无 exclude 命中 | **允许**（含发现链上的新子域/IP/URL） |
| 命中 `scope_rules.action=exclude` | **拒绝**派生 work / FilterTargets 过滤 |
| `scope_rules.action=include` | **已废弃**，不参与评估 |

ScanEngine 在 `processNewAsset` 与 seed 路径同时检查：全局 `exclude.Manager` + `scope.Engine.IsExcludedForProject`。

`PipelineConfig`（`EnableHttpx`、`EnableNuclei` 等）通过 `core.ProfileFromConfig` 控制 work 派生；Nuclei stdout 经 `internal/finding.NucleiPersister` 写入 Findings + Evidence。

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

1. **scanengine.processNewAsset**: 全局排除列表 + 项目 exclude 规则
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

## Docker 构建与部署

用户生产环境只消费 ACR 上的 `anchor-server`、`anchor-worker`、`anchor-frontend`（见上文 compose 表）。**客户部署操作见 [`deployment.md`](deployment.md)；开发者 E2E 见 [`e2e-testing.md`](e2e-testing.md)。**

下列 **runtime-base / fast** 镜像仅供开发者本地 E2E 快速迭代；**生产发布**使用独立的 `Dockerfile.server` / `Dockerfile.worker`（`debian:bookworm-slim` 单阶段，经 `docker/install-anchor-binary.sh` 注入 Release 二进制），不出现在客户部署 compose 中。

### 镜像构建策略（开发 E2E / CI 发布）

**E2E 快速构建**（`Dockerfile.*-fast`，基于 runtime-base）：

```
┌─────────────────────────────────────────────────────────┐
│  应用层（频繁变化，快速构建 10-30 秒）                    │
│  ┌─────────────────┐  ┌─────────────────┐              │
│  │ anchor-server   │  │ anchor-worker   │              │
│  │ :local          │  │ :local          │              │
│  │ COPY bin/anchor │  │ COPY bin/anchor │              │
│  └────────┬────────┘  └────────┬────────┘              │
├───────────┼─────────────────────┼───────────────────────┤
│  Base 层（很少变化，预构建缓存）                          │
│  ┌────────┴────────┐  ┌────────┴────────┐              │
│  │ anchor-server-  │  │ anchor-worker-  │              │
│  │ base            │  │ base            │              │
│  │ + 系统依赖      │  │ + 安全工具×10   │              │
│  └─────────────────┘  └─────────────────┘              │
└─────────────────────────────────────────────────────────┘
```

**生产发布镜像**（CI `docker-push.yml` / `make release-verify`）：

| 镜像 | Dockerfile | 构建方式 |
|------|-----------|---------|
| `anchor-server` | `Dockerfile.server` | `debian:bookworm-slim` + `install-anchor-binary.sh` |
| `anchor-worker` | `Dockerfile.worker` | 同上 + 内联安装安全工具 |
| `anchor-frontend` | `Dockerfile.frontend` | Nginx + React 构建产物 |

**E2E 快速镜像清单**：

| 镜像 | Dockerfile | 职责 |
|------|-----------|------|
| `anchor-server-base` | `Dockerfile.server-runtime-base` | Server 运行时依赖 |
| `anchor-worker-base` | `Dockerfile.worker-runtime-base` | Worker 运行时 + 安全工具 |
| `anchor-server:local` | `Dockerfile.server-fast` | 基于 base，COPY 本地二进制 |
| `anchor-worker:local` | `Dockerfile.worker-fast` | 基于 base，COPY 本地二进制 |

**优势**：
- 测试迭代快：只复制二进制，10-30 秒完成构建
- 节省带宽：base 镜像可预构建并缓存
- 测试/发布一致性：使用相同的构建流程

### 多平台构建

**支持平台**：
- `linux/amd64` (x86_64)：VPS、PC 用户
- `linux/arm64` (aarch64)：Mac M1/M2 用户

**构建方式**：
- GitHub Actions 使用 QEMU 模拟 + Docker Buildx 构建多平台镜像
- 构建命令：`docker buildx build --platform linux/amd64,linux/arm64 -t <image> --push .`
- 用户拉取时自动选择匹配的平台架构

### CI/CD 流程

**触发条件**：
- `ci.yml`：push/PR 到 `main` 时跑 `go test`/`go vet` 与前端 typecheck/unit/build（见 [`ci-cd-guide.md`](ci-cd-guide.md)）
- `release-verify.yml`：**tag 推送前**手动触发，本地等价 `make release-verify`；用生产 Dockerfile 构建候选镜像并按用户 compose 路径验收
- `release.yml`：tag 推送时触发，构建 Go 二进制并上传到 GitHub Release
- `docker-push.yml`：Release 完成后自动触发，或通过 `workflow_dispatch` 手动触发

**上线前验证**（`RELEASE_VERSION=local`）：
- `Dockerfile.server` / `Dockerfile.worker` 通过 `docker/install-anchor-binary.sh` 安装二进制：`local` 用 `bin/anchor-linux-*`，否则 curl GitHub Release（`docker-push.yml` 构建前 `mkdir -p bin`）
- `docker-compose.release-verify.yml` 镜像结构与 `docker-compose.yml` 一致，端口/网络隔离（默认 frontend `:18080`、API `:17422`）

**构建流程**（`docker-push.yml`）：
1. 解析 `RELEASE_VERSION`（Release workflow 的 `head_branch` = tag，或 `workflow_dispatch` 输入）
2. Checkout **该 tag/ref**（保证 Dockerfile 内源码种子与发布版本一致）
3. 设置 QEMU + Docker Buildx，登录阿里云 ACR
4. 构建并推送多平台镜像；server/worker 传入 `--build-arg RELEASE_VERSION=<tag>`

**镜像标签**：
- `v0.x.x`：与 GitHub Release tag 对齐的权威版本标签
- `latest`：每次成功 docker-push 同时更新，供 `install.sh` 默认拉取

### 环境变量与配置

**Server 镜像**：
- `ANCHOR_DATA_DIR=/data`：数据存储目录
- `ANCHOR_PORT=17421`：API 服务端口
- `ANCHOR_TEMPLATES_SEED=/app/templates/vuln-templates.json`：漏洞模板种子文件

**Worker 镜像**：
- `ANCHOR_DATA_DIR=/data`：数据存储目录
- `ANCHOR_CORE_URL=""`：Server 连接地址（启动时配置）

**安全工具版本**（在 Dockerfile.worker 中集中管理）：
- subfinder: 2.14.0
- naabu: 2.6.1
- httpx: 1.9.0
- nuclei: 3.8.0
- cdncheck: 1.2.38
- dnsx: 1.2.3
- katana: 1.6.1
- ffuf: 2.1.0
- gau: 2.2.4
- spoor: 0.2.0

## What Is Not Baseline Yet

- `docs/refactoring-plan.md` is a backlog/refactor inventory, not the current product architecture.
- `docs/design/custom-nuclei-template-management.md` is an in-review design for custom Nuclei template management.

## How To Use This File

- Use this file for repo-level orientation.
- Use the implementation and tests to answer behavior questions.
- Use `docs/current/design/README.md` only when a task explicitly targets a proposal or review stream.

## Documentation Contract

If architecture changes materially, update this file first or in the same change set. Proposal documents should explain the delta from this baseline instead of redefining the entire system from scratch.
