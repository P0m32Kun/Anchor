---
status: active
source_of_truth: true
owner: kun
last_updated: 2026-06-12
scope: system-reference
---

# Anchor 系统参考手册

> 本文件是 Anchor 项目的完整系统参考手册。
> **目标**：coding agent 无需阅读源码，仅凭本文即可了解功能位置、增改入口。
>
> 本文件是 `docs/current/architecture.md` 的补充 —— 架构基线讲"系统长什么样"，
> 本手册讲"代码在哪、怎么改"。

---

## 目录

1. [Phase 1: 项目骨架与包体系总览](#phase-1-项目骨架与包体系总览)
   - [技术栈](#技术栈)
   - [项目入口与启动流程](#项目入口与启动流程)
   - [internal 包体系全表](#internal-包体系全表)
   - [包依赖层次](#包依赖层次)
   - [前端目录结构](#前端目录结构)
   - [环境变量一览](#环境变量一览)
2. [Phase 2: 工具执行全链路](#phase-2-工具执行全链路)
3. [Phase 3: 扫描引擎深度](#phase-3-扫描引擎深度)
4. [Phase 4: 数据模型与 API 对照](#phase-4-数据模型与-api-对照)
5. [Phase 5: 新增功能标准操作指南](#phase-5-新增功能标准操作指南)

---

# Phase 1: 项目骨架与包体系总览

## 技术栈

| 层 | 技术 | 版本 |
|----|------|------|
| 后端语言 | Go | 1.26 |
| 前端框架 | React | 18 |
| 前端语言 | TypeScript | — |
| 数据库 | SQLite | WAL 模式 |
| 构建工具 | Make | 3.81 |
| 容器 | Docker / docker-compose | — |
| 实时推送 | SSE（Server-Sent Events） | — |
| JWT 鉴权 | golang-jwt/jwt/v5 | 5.3.1 |
| 系统监控 | gopsutil/v3 | 3.24.5 |
| 无头浏览器 | chromedp | 0.15.1 |
| 配置格式 | YAML / JSON | — |

## 项目入口与启动流程

**入口文件**: `main.go`

```text
main()
├── flag 解析: -worker, -core-url
├── ANCHOR_DATA_DIR 环境变量 / ~/.anchor
│
├── 非 worker 模式 → runServer(dataDir)
│   ├── db.Open(dataDir)           → SQLite 连接 + 自动迁移
│   ├── queries := db.New(sqliteDB)
│   ├── 种子化 finding_templates   (vuln-templates.json)
│   ├── server := api.NewServer(queries, sqliteDB, dataDir)
│   ├── server.Register(mux)        → 注册所有 HTTP 路由
│   └── http.ListenAndServe(":17421", CORSMiddleware)
│
└── worker 模式 → runWorker(dataDir, coreURL)
    ├── builtin.SyncAll()           → 同步 RBKD 内置资源
    ├── ws := worker.NewWorkerServer(dataDir, coreURL, apiToken)
    ├── ws.Register(mux)
    ├── 随机端口监听 0.0.0.0:0
    ├── 若有 coreURL → 创建 RemoteClient
    │   ├── Register("remote-worker")     指数退避重试 5 次
    │   ├── StartHeartbeat(30s)
    │   ├── StartSourceSync(60s)
    │   └── StartPolling()
    └── http.Serve(listener, mux)
```

**Docker 模式**: 通过 `install.sh` 交互式选择部署模式（Server Only / Worker Only / Server+Worker），使用 docker-compose 编排三个镜像（anchor-server、anchor-worker、anchor-frontend）。

## internal 包体系全表

> 按分层组织：**基础设施 → 数据层 → 业务逻辑 → API 层 → 执行引擎 → Worker → 工具**

### 基础设施层

| 包 | 路径 | 职责 | 关键文件 | 依赖 |
|----|------|------|----------|------|
| `errors` | `internal/errors/` | 统一错误定义，全项目共享的错误类型 | `errors.go` | 无 |
| `util` | `internal/util/` | 工具函数：ID 生成（ULID）、URL 解析、Sanitizer、优雅关停 | `id.go`, `url.go`, `sanitizer.go`, `shutdown.go` | 无 |
| `safefs` | `internal/safefs/` | 安全的文件系统操作（路径校验、沙箱读写） | `safefs.go` | 无 |
| `cdn` | `internal/cdn/` | CDN 检测：解析 CDN 服务商数据，判断 IP 是否属于已知 CDN | `parse.go` | 无 |
| `health` | `internal/health/` | 工具健康检查：检测本地已安装的安全工具版本可用性 | `health.go` | `toolguard` |
| `asset` | `internal/asset/` | 资产合并(Merger)与标准化(Normalizer) | `merger.go`, `normalizer.go`, `state.go` | `models` |

### 数据层

| 包 | 路径 | 职责 | 关键文件 | 依赖 |
|----|------|------|----------|------|
| `db` | `internal/db/` | SQLite 数据库操作。包括：连接管理、自动迁移（v1→v44+）、查询方法、种子数据 | `db.go`, `queries*.go`, `v*.go`, `buffer.go` | `models`, `sqlite3` |
| `models` | `internal/models/` | 所有业务模型结构体定义。纯数据结构，无方法（除 JSON 序列化） | `*.go` 共 29 个文件 | 无 |

> **models 包文件索引**（29 个文件）：
> `asset.go`, `asset_change.go`, `asset_relation.go`,
> `cdn.go`, `certificate.go`, `common.go`, `dashboard.go`,
> `dictionary.go`, `dns.go`, `engine.go`, `exclude.go`,
> `finding.go`, `finding_template.go`, `fingerprint.go`,
> `health.go`, `httpx_fingerprint.go`, `notification.go`,
> `nuclei_custom.go`, `project.go`, `scan.go`, `scan_work.go`,
> `scope.go`, `signal.go`, `target.go`, `tool_call_log.go`,
> `watch.go`, `worker.go`

### 业务逻辑层

| 包 | 路径 | 职责 | 关键文件 | 依赖 |
|----|------|------|----------|------|
| `service` | `internal/service/` | 业务逻辑层。接口定义 + 实现，负责 API handler 与 DB 之间的编排 | `service.go`, `interfaces.go`, `project.go`, `target.go`, `finding.go`, `adapter.go` | `models`, `db` |
| `scope` | `internal/scope/` | 项目范围规则引擎。检查目标是否在范围内，支持 import/export | `scope.go`, `import.go` | `models`, `db` |
| `exclude` | `internal/exclude/` | 全局域名排除管理。内置默认排除列表 + 用户自定义 | `exclude.go`, `defaults.go` | `models`, `db` |
| `scoring` | `internal/scoring/` | 漏洞评分计算（CVSS 等） | `scoring.go` | `models` |
| `finding` | `internal/finding/` | Nuclei 发现结果持久化（NucleiPersister） | `nuclei.go` | `models`, `db` |
| `signal` | `internal/signal/` | 信号处理 + Webhook 通知 | `signal.go`, `webhook.go` | `models`, `db` |
| `notify` | `internal/notify/` | 通知发送（钉钉/企微/飞书等） | `notify.go` | — |
| `watch` | `internal/watch/` | 文件/目录变更监听 | `watcher.go` | — |
| `resolve` | `internal/resolve/` | DNS 解析器 | `resolver.go` | — |
| `fingerprint` | `internal/fingerprint/` | Nmap 服务指纹识别 | `nmap.go` | — |
| `screenshot` | `internal/screenshot/` | 网页截图（基于 chromedp） | `capture.go`, `manager.go` | `chromedp` |
| `nuclei` | `internal/nuclei/` | Nuclei 相关：模板标签映射(tagmapper)、自定义模板管理(custom/) | `tagmapper.go`, `custom/` | `models`, `db` |
| `dictionary` | `internal/dictionary/` | 字典管理（ffuf 等工具的字典词库） | `manager.go`, `layout.go`, `seed.go` | `models`, `db` |
| `httpxfp` | `internal/httpxfp/` | HTTPX 指纹管理 | `manager.go`, `layout.go`, `seed.go` | `models`, `db` |

### 被动搜索层

| 包 | 路径 | 职责 | 关键文件 | 依赖 |
|----|------|------|----------|------|
| `search` | `internal/search/` | 被动搜索引擎统一接口 + 实现：FOFA、Hunter、Quake | `engine.go`, `fofa.go`, `hunter.go`, `quake.go` | `models` |
| `passive` | `internal/passive/` | crt.sh 证书透明度日志查询 | `crt.go` | — |

### 工具注册与执行层

| 包 | 路径 | 职责 | 关键文件 | 依赖 |
|----|------|------|----------|------|
| `toolregistry` | `internal/toolregistry/` | 工具注册表：YAML schema → ToolDef，参数编译 → RenderCmdline | `types.go`, `schema.go`, `validate.go`, `embed.go` | 无 |
| `toolguard` | `internal/toolguard/` | 工具安全门禁：二进制白名单 + 参数 shell 元字符校验 | `allowlist.go` | 无 |
| `toolrun` | `internal/toolrun/` | 工具执行统一入口：Invoke() 串联 registry → guard → Run → artifact | `invoke.go`, `artifact.go` | `toolregistry`, `toolguard`, `models`, `util` |
| `parser` | `internal/parser/` | 安全工具 stdout 解析器，每个工具一个文件。输入 stdout bytes，输出结构化结果 | `common.go`, `nuclei.go`, `httpx.go`, `naabu.go`, `ffuf.go`, `katana.go`, `subfinder.go`, `dnsx.go`, `nmap.go`, `gau.go` | `models` |

### 扫描执行层

| 包 | 路径 | 职责 | 关键文件 | 依赖 |
|----|------|------|----------|------|
| `scanengine` | `internal/scanengine/` | 资产驱动扫描引擎主控。详见 [Phase 3](#phase-3-扫描引擎深度) | `engine.go`, `engine_tier1.go`, `engine_tier2.go` | 多包依赖 |
| `scanengine/core` | `internal/scanengine/core/` | 核心类型：DiscoveryAsset、TaskAction、AssetAttrs、DeriveEligibleWorks 规则 | `task.go`, `rules.go`, `asset.go`, `attrs.go`, `preconditions.go`, `profile_config.go` | `models`, `toolregistry` |
| `scanengine/executor` | `internal/scanengine/executor/` | Executor 接口 + ToolExecutor 实现（httpx/nuclei/katana/ffuf/spoor） | `executor.go`, `ffuf.go`, `katana.go`, `spoor.go` | `toolrun`, `parser`, `models` |
| `scanengine/pool` | `internal/scanengine/pool/` | 通用池化 + httpx/port/nuclei 分桶 | `pool.go`, `ip_port_agg.go`, `nuclei_buckets.go`, `probe_target.go` | `models`, `scanconfig` |
| `scanengine/queue` | `internal/scanengine/queue/` | 优先级队列：PopFairStaged、StageRank | `fair.go`, `priority.go`, `stage_rank.go` | — |
| `scanengine/scheduler` | `internal/scanengine/scheduler/` | 调度器：并发计算、IP 限流、种子分桶 | `limits.go`, `ip_throttle.go`, `bucket.go` | — |
| `scanengine/work` | `internal/scanengine/work/` | WorkItem 存储：TryClaim/MarkDone/AllTerminal | `store.go` | `models`, `db` |
| `scanengine/dedup` | `internal/scanengine/dedup/` | Run 级资产去重 | `run_dedup.go` | — |
| `scanengine/recovery` | `internal/scanengine/recovery/` | Server 重启后孤儿 Run 恢复 | `orphan.go` | `db` |
| `scanengine/seed` | `internal/scanengine/seed/` | 种子资产扩展、被动搜索编排、边界过滤 | `convert.go`, `expand.go`, `passive_search.go`, `passive.go`, `boundary_filter.go`, `junk_filter.go` | `search`, `passive`, `models` |
| `scanengine/stageagg` | `internal/scanengine/stageagg/` | Work → Stage 投影（仅 UI 展示，不影响执行逻辑） | `aggregator.go` | `models` |
| `scanengine/domainpool` | `internal/scanengine/domainpool/` | 域名池化（DNS/CDN 批量） | `pool.go` | — |

### 扫描配置层

| 包 | 路径 | 职责 | 关键文件 | 依赖 |
|----|------|------|----------|------|
| `scanconfig` | `internal/scanconfig/` | 扫描配置管理：Profile 配置、文件读取、Nuclei 路由规则 | `config.go`, `apply.go`, `files.go`, `nuclei_routing.go`, `fallbacks.go` | `models` |

### Worker 层

| 包 | 路径 | 职责 | 关键文件 | 依赖 |
|----|------|------|----------|------|
| `worker` | `internal/worker/` | 扫描任务执行引擎。包括：本地 Runner 执行、远程 Worker Server/Client、命令构建、资源治理 | `worker.go`, `server.go`, `remote_client.go`, `dispatcher.go`, `commands.go`, `task_output.go`, `concurrency.go`, `resource_governor.go` | `models`, `db`, `toolguard` |

### 报告与内置资源层

| 包 | 路径 | 职责 | 关键文件 | 依赖 |
|----|------|------|----------|------|
| `report` | `internal/report/` | 报告导出：Markdown 格式，按字典模板聚合 | `report.go`, `markdown.go`, `aggregate_run.go` | `models`, `db` |
| `builtin` | `internal/builtin/` | RBKD 内置资源同步：Git clone/pull dict/templates/finger | `sync.go`, `config.go`, `nuclei_symlink.go` | — |

### API 层

| 包 | 路径 | 职责 | 关键文件 | 依赖 |
|----|------|------|----------|------|
| `api` | `internal/api/` | HTTP API 层。Server 结构体 + 所有 handler + 中间件 + SSE | `server.go`, `*_handlers.go`（31 个 handler 文件） | 多包依赖，见下文 |

> **API handler 文件总览**见 `internal/api/README.md`，此处仅录顶层包依赖：
> - `queries` 字段被 16 个 handler 文件依赖（几乎全包）
> - `dataDir` 被 8 个 handler 文件依赖
> - `scopeEng` 被 3 个 handler 文件依赖
> - 任务分发四件套(`taskQueue`/`taskResults`/`sseClients`/`mu`)被 run/worker/sse 依赖
> - 其余字段各被 1 个 handler 文件依赖

## 包依赖层次

> 注意：以下箭头方向为**依赖方向**，箭头指向被依赖方。

```text
                             ┌──────────┐
                             │   api    │  ← HTTP 层
                             └────┬─────┘
                                  │
                    ┌─────────────┼─────────────┐
                    │             │             │
              ┌─────┴─────┐ ┌────┴────┐ ┌──────┴──────┐
              │  service  │ │  scan   │ │  worker     │  ← 业务/执行层
              └─────┬─────┘ │ engine  │ └──────┬──────┘
                    │       └────┬─────┘        │
                    │            │              │
              ┌─────┴─────┐ ┌────┴────┐ ┌──────┴──────┐
              │    db     │ │ executor│ │ toolrun     │  ← 数据/工具层
              └─────┬─────┘ └────┬────┘ └──────┬──────┘
                    │            │             │
              ┌─────┴─────┐     │      ┌──────┴──────┐
              │  models   │     │      │ toolregistry │  ← 模型/定义层
              └───────────┘     │      ├─────────────┤
                                │      │ toolguard    │
                                │      └──────┬──────┘
                                │             │
                           ┌────┴────┐  ┌────┴────┐
                           │  parser │  │   util  │    ← 工具库
                           └─────────┘  └─────────┘

  ┌─────────────────────────────────────────────────────┐
  │                     main.go                         │  ← 入口
  │    runServer() → api.NewServer() → Register()       │
  │    runWorker() → worker.NewWorkerServer()           │
  └─────────────────────────────────────────────────────┘
```

### 关键依赖链示例

| 场景 | 调用链 |
|------|--------|
| 项目 CRUD | `HTTP POST /projects` → `project_handlers.go` → `projectSvc.Create()` → `db.queries.Create()` → `models.Project` |
| 启动扫描 | `HTTP POST /projects/{id}/scan` → `pipeline_handlers.go` → `scanengine.NewEngine()` → `engine.Run()` → 循环 tick |
| 工具执行 | `engine.tick()` → `executor.ToolExecutor.Execute()` → `toolrun.Invoke()` → `toolguard.Validate()` → `toolregistry.RenderCmdline()` → `worker.Runner.Run()` |
| 输出解析 | `executor.onWorkComplete()` → `parser.ParseNucleiOutput()` → 写入 `scan_work_items` + `findings` |

## 前端目录结构

> `frontend/src/` 下的主要目录和职责：

| 目录 | 职责 | 关键文件 |
|------|------|----------|
| `pages/` | 页面级组件，每个功能一个页面 | `RunsPage.tsx`, `ProjectsPage.tsx`, `FindingsPage.tsx` |
| `components/` | 可复用 UI 组件 | 表单、表格、弹窗等 |
| `hooks/` | 自定义 React Hooks | API 调用、SSE 订阅、状态管理 |
| `lib/` | 工具函数、API 客户端 | API 请求封装 |
| `test/` | 前端测试 | E2E 测试（Playwright） |

## 环境变量一览

> 所有 `ANCHOR_*` 环境变量及其作用：

| 变量 | 默认值 | 作用域 | 说明 |
|------|--------|--------|------|
| `ANCHOR_DATA_DIR` | `~/.anchor` | Server/Worker | 数据存储目录（DB WAL + worker 工作目录） |
| `ANCHOR_PORT` | `17421` | Server | API 监听端口 |
| `ANCHOR_API_TOKEN` | — | Server/Worker | API 鉴权 Token |
| `ANCHOR_CORE_URL` | — | Worker | Server 连接地址（远程 Worker 模式） |
| `ANCHOR_WORKER_HOST` | `127.0.0.1` | Worker | Worker 对外公布的地址 |
| `ANCHOR_WORKER_MAX_CONCURRENCY` | `10` | Worker | Worker 最大并发任务数 |
| `ANCHOR_TEMPLATES_SEED` | `docs/templates/vuln-templates.json` | Server | 漏洞模板种子文件路径 |
| `ANCHOR_BUILTIN_SYNC` | `on` | Server/Worker | 内置资源自动同步开关（`on`/`off`） |
| `ANCHOR_GOVERNOR_MEM_THRESHOLD` | — | Worker | 资源治理内存阈值百分比 |
| `ANCHOR_GOVERNOR_CPU_THRESHOLD` | — | Worker | 资源治理 CPU 阈值百分比 |
| `ANCHOR_SCAN_ENGINE` | — | Server | 启用资产驱动 ScanEngine（`1` 启用） |

---

# Phase 2: 工具执行全链路

> 本章覆盖从「扫描引擎决定执行一个工具」到「工具输出被解析入库」的完整数据流。
> 这条链路是 Anchor 的核心价值所在 —— 安全工具的编排与输出处理。

## 链路总览

```text
ScanEngine.tick()
  → executor.ToolExecutor.Execute()
     → 1. actionToToolID()       —— Action 枚举 → 工具 ID
     → 2. toolrun.Invoke()
        → 3. registry.Render()   —— YAML 定义 → argv
        → 4. toolguard.Validate() —— 二进制 + 参数白名单
        → 5. runner.Run()        —— exec.Command 执行
        → 6. 读 stdout artifact
     → 7. 写 ToolCallLog
  → onWorkComplete()
     → 8. parser.ParseXXX()      —— stdout 解析
     → 9. 新资产注入引擎循环
```

## 第 1 层：工具注册表（toolregistry）

**包路径**：`internal/toolregistry/`
**核心类型**：

| 类型 | 文件 | 说明 |
|------|------|------|
| `ToolDef` | `types.go` | YAML 工具定义的结构化表示。含 ID/Binary/Parameters/Literals/Presets |
| `ParamDef` | `types.go` | 参数定义：类型 + 标志 + 默认值 |
| `Registry` | `types.go` | 编译后的工具集合，提供 `Render(id, params)` → argv |
| `RenderParams` | `schema.go` | `map[string]interface{}` |

**关键操作**：

```go
// 加载
reg := toolregistry.DefaultRegistry()  // 读 embed tools/*.yaml

// 渲染命令
argv, err := reg.Render("nuclei", toolregistry.RenderParams{
    "target_file": "/tmp/targets.txt",
    "profile":     "standard",
    "tags":        []string{"cve", "rce"},
})
// → ["nuclei", "-l", "/tmp/targets.txt", "-jsonl", "-severity", "critical,high,medium", "-timeout", "5", "-tags", "cve,rce"]
```

**参数类型**：

| 类型 | 说明 | 示例产出 |
|------|------|----------|
| `string` | 字符串 | `-flag value` |
| `int` | 整型（>0 才渲染） | `-flag 100` |
| `string_list` | 逗号拼接 | `-tags cve,rce` |
| `path` | 文件路径 | `-l /tmp/hosts.txt` |
| `enum` | 枚举值。支持 `ValueFlags`（value→一组 flag）和 `Preset`（命名预设） | `-severity critical,high` |

**工具定义文件**：`tools/*.yaml`（13 个文件）

| YAML 文件 | Tool ID | Binary | 输出格式 |
|-----------|---------|--------|----------|
| `nuclei.yaml` | nuclei | nuclei | jsonl |
| `httpx.yaml` | httpx | httpx | jsonl |
| `naabu.yaml` | naabu | naabu | jsonl |
| `subfinder.yaml` | subfinder | subfinder | jsonl |
| `dnsx.yaml` | dnsx | dnsx | jsonl |
| `cdncheck.yaml` | cdncheck | cdncheck | jsonl |
| `nmap_alive.yaml` | nmap_alive | nmap | greppable |
| `nmap_service.yaml` | nmap_service | nmap | xml |
| `ffuf.yaml` | ffuf | ffuf | jsonl |
| `katana.yaml` | katana | katana | jsonl |
| `gau.yaml` | gau | gau | jsonl |
| `spoor.yaml` | spoor | spoor | jsonl |

**Override 机制**：`ANCHOR_TOOLS_DIR` 环境变量指向的目录中若有同名 `.yaml` 文件，会覆盖嵌入式定义。用于开发测试而不重建二进制。

## 第 2 层：工具安全门禁（toolguard）

**包路径**：`internal/toolguard/`
**核心文件**：`allowlist.go`
**核心类型**：`Allowlist`

```go
// 创建
allowlist := toolguard.NewAllowlistFromBinaries(reg.Binaries())

// 校验
err := allowlist.Validate(binary, args)
// 失败条件：
//   1. binary 的 basename 不在白名单中
//   2. 任一参数含 shell 元字符：; | & > < ` $ ( ) { } [ ] \n \r
```

**默认白名单**：subfinder, dnsx, httpx, naabu, nmap, nuclei, cdncheck, ffuf, gau, katana, spoor, chromium, google-chrome, git, sh, bash

**接入点**：`toolrun.Invoke()` 内部自动调用；旧版 `worker.Runner.Run()` 也调用。

## 第 3 层：工具执行统一入口（toolrun.Invoke）

**包路径**：`internal/toolrun/`
**核心文件**：`invoke.go`, `artifact.go`

**Invoke 流程**（8 步，对应 [链路总览](#链路总览)）：

1. **查注册表**：`reg.Get(toolID)` → 找不到报错
2. **渲染命令行**：`reg.Render(toolID, params)` → argv
3. **追加额外参数**：拼接 `ExtraArgs`（如 nuclei workflow 路径）
4. **白名单校验**：`allowlist.Validate(binary, args[1:])`
5. **创建 ScanTask**：写 `scan_tasks` 表，含 command_template
6. **执行**：`runner.Run(ctx, taskID)` 阻塞等待完成
7. **读 stdout**：从 `RawArtifact` 文件读取 stdout 字节
8. **返回**：`InvokeResult{Task, Stdout, Err}`

```go
type InvokeInput struct {
    ProjectID string
    RunID     *string
    TaskID    string                   // 空则自动生成
    ToolID    string                   // tools/*.yaml 的 id
    Params    toolregistry.RenderParams
    ExtraArgs []string                 // 追加的 argv token
}

type InvokeResult struct {
    Task   *models.ScanTask
    Stdout []byte
    Err    error
}
```

> **重要**：`InvokeResult.Err` 非 nil 时 `Stdout` 可能仍有内容（部分输出的工具）。调用方需同时检查两者。

## 第 4 层：Worker 执行器

存在两条执行路径：

### 路径 A：新路径（ScanEngine 使用，推荐）

`internal/scanengine/executor/executor.go` → `toolrun.Invoke()` → `worker.Runner.Run()`

**Action → ToolID 映射**（`actionToToolID` 函数）：

| Action（core.TaskAction） | Tool ID | 工具 |
|---------------------------|---------|------|
| `ActionHTTPXFingerprint` | httpx | HTTP 探活 + 指纹 |
| `ActionNucleiScan` | nuclei | 漏洞扫描 |
| `ActionPortScan` | naabu | 端口扫描 |
| `ActionServiceFingerprint` | nmap_service | 服务指纹 |
| `ActionKatanaCrawl` | katana | 爬虫 |
| `ActionFFUFBrute` | ffuf | 目录爆破 |
| `ActionSubdomainEnum` | subfinder | 子域枚举 |
| `ActionDNSResolve` | dnsx | DNS 解析 |
| `ActionCDNCheck` | cdncheck | CDN 检测 |
| `ActionSpoorScan` | spoor | JS 静态分析 |

**执行过程**：Executor 在调用 `toolrun.Invoke` 前后记录 `ToolCallLog`（开始时间、参数 JSON、完成时间、状态、输出摘要、错误信息）。

### 路径 B：旧路径（维护模式，不推荐用于新代码）

`internal/worker/commands.go` 中的 `Build*Command` 函数族。

这些函数保留作为 golden test 对照和 `discovery.go`/`screenshot.go` 两个旧工作流使用。
**新代码（ScanEngine 池化批量执行）走路径 A。**

### Worker 并发模型

| 组件 | 文件 | 说明 |
|------|------|------|
| `Runner` | `worker.go` | 核心执行器。从 DB 读 command_template → `exec.Command` → 写 artifact → 更新状态 |
| `WorkerServer` | `server.go` | 远程 Worker HTTP 服务。接收 `POST /tasks` 执行，支持实时日志推送 |
| `RemoteClient` | `remote_client.go` | 远程 Worker 客户端。长轮询拉任务、心跳、同步内置资源 |
| `Dispatcher` | `dispatcher.go` | Server 侧任务分配。最少负载优先 → 轮询 → 容量上限 → 故障转移 |
| `ResourceGovernor` | `resource_governor.go` | 系统资源阈值控制 |
| `ConcurrencyLimiter` | `concurrency.go` | 任务并发数限制 |
| `TaskOutput` | `task_output.go` | 任务实时输出流式读取（SSE） |

**Task 生命周期**：`TaskCreated → TaskRunning → TaskCompleted / TaskFailed / TaskCancelled`

## 第 5 层：输出解析器（parser）

**包路径**：`internal/parser/`
**核心文件**：`common.go` + 每个工具一个解析文件

| 文件 | 解析函数 | 输入格式 | 输出类型 |
|------|----------|----------|----------|
| `nuclei.go` | `ParseNuclei` | JSONL | `[]NucleiResult` |
| `httpx.go` | `ParseHTTPX` | JSONL | `[]HTTPXResult` |
| `naabu.go` | `ParseNaabu` | JSONL | `[]NaabuResult` |
| `ffuf.go` | `ParseFFUF` | JSONL | `[]FFUFResult` |
| `katana.go` | `ParseKatana` | JSONL | `[]KatanaResult` |
| `subfinder.go` | `ParseSubfinder` | JSONL | `[]SubfinderResult` |
| `dnsx.go` | `ParseDNSX` | JSONL | `[]DNSXResult` |
| `nmap.go` | `ParseNmap` | XML | `[]NmapResult` |
| `gau.go` | `ParseGau` | JSONL | `[]GauResult` |

**解析机制**（`parseJSONLines` 泛型函数位于 `common.go`）：
- 用 `bufio.Reader.ReadBytes('\n')` 逐行读取（非 `bufio.Scanner`，避免 64 KiB 行限制）
- 每行通过 `json.Unmarshal` 解码到对应结构体
- 跳过空行，记录 `ParseError`
- 单行最大 16 MiB 硬限制（`maxJSONLineBytes`），超限则跳过并记录错误

## 全链路数据流示例

以 **httpx 指纹扫描** 为例：

```text
1. DeriveEligibleWorks(AssetHTTPService)
   → ActionHTTPXFingerprint 被派生

2. engine.tick()
   → pq.PopFairStaged() 获取 work item
   → executor.Execute(ctx, work, params)

3. ToolExecutor.Execute()
   → actionToToolID("http_fingerprint") → "httpx"
   → toolrun.Invoke(ctx, queries, runner, reg, InvokeInput{
         ToolID: "httpx",
         Params: {"target_file": "/tmp/hosts.txt", "tech_detect": true},
     })

4. toolrun.Invoke()
   → reg.Render("httpx", params) → [httpx -l /tmp/hosts.txt -json -td -silent -threads 50]
   → toolguard.Validate("httpx", ["-l", ...])
   → queries.CreateScanTask(&ScanTask{CommandTemplate: "httpx -l ..."})
   → runner.Run(ctx, taskID)
      → exec.Command("httpx", "-l", "/tmp/hosts.txt", ...)
      → 写 stdout 到 artifact 文件
   → 读 artifact → InvokeResult.Stdout = stdout bytes

5. onWorkComplete()
   → ParseHttpxOutput(stdout)
      → 解析 JSONL → 新资产 (AssetHTTPService, AssetHTTPPath)
      → attrs.Technologies = ["React", "nginx"]
      → web endpoints 写入 DB

6. 新资产注入 assetDepth 和 pq，引擎循环继续
```

## 关键设计原则

1. **注册表驱动**：所有 CLI 参数定义在 `tools/*.yaml`，不硬编码在代码中。
2. **安全门禁**：每个 `exec.Command` 调用都经 `toolguard` 校验二进制路径和参数。
3. **路径 A 优先**：新功能使用 `toolrun.Invoke` 路径，不要把 `Build*Command` 函数用于新代码。
4. **16 MiB 保护**：JSONL 解析器单行硬限制，防 OOM。
5. **记录审计**：每次工具调用写 `tool_call_logs` 表（参数、耗时、输出摘要、状态）。

---

# Phase 3: 扫描引擎深度

> 本章覆盖 `internal/scanengine/` 全部子包的职责、核心类型、关键数据流。
> ScanEngine 是 Anchor 的扫描执行核心，采用资产驱动模型（非管线阶段模型）。

## 核心概念

| 概念 | 说明 |
|------|------|
| DiscoveryAsset | 引擎内部 DTO：资产图中的节点。含 Type/Value/Depth/Attrs |
| Work (ScanWorkItem) | 调度最小单元：`(run_id, asset_id, action)` 唯一 |
| ActionRule | 动作启用条件：`Enabled + MaxDepth + Precondition` |
| EngineState | Run 级状态：`running` → `wind_down` → `stopped` |
| Profile | 扫描模式（internal/external/url_only），决定哪些 ActionRule 生效 |

## 资产类型与派生规则

### 资产类型（`core.AssetType`）

| 类型 | 常量值 | 说明 |
|------|--------|------|
| 子域 | `SUBDOMAIN` | 域名级资产，如 `api.example.com` |
| IP | `IP` | 单 IP 地址 |
| CIDR | `CIDR` | IP 段，如 `172.30.0.0/24` |
| IP:Port | `IP_PORT` | 开放端口 |
| HTTP 服务 | `HTTP_SERVICE` | URL 级 Web 服务 |
| HTTP 路径 | `HTTP_PATH` | 爬虫发现的子路径 |
| JS URL | `JS_URL` | Katana/Spoor 发现的 JS 端点 |

### 动作类型（`core.TaskAction`）

| Action | 说明 | 对应 Tool ID | Stage |
|--------|------|-------------|-------|
| `PASSIVE_SEARCH` | 被动搜索引擎 | subfinder | search |
| `PASSIVE_CERT` | crt.sh 证书查询 | crt | passive_cert |
| `PASSIVE_URL` | Gau 历史 URL | gau | passive_url |
| `SUBDOMAIN_ENUM` | 子域枚举 | subfinder | subdomain |
| `DNS_RESOLVE` | DNS 解析 | dnsx | resolve |
| `CDN_CHECK` | CDN 检测 | cdncheck | cdn_filter |
| `PORT_SCAN` | 端口扫描 | naabu | portscan |
| `SERVICE_FINGERPRINT` | 服务指纹 | nmap_service | fingerprint |
| `HTTPX_FINGERPRINT` | HTTP 探活+指纹 | httpx | httpx |
| `KATANA_CRAWL` | 爬虫 | katana | crawl |
| `FFUF_BRUTE` | 目录爆破 | ffuf | ffuf |
| `NUCLEI_SCAN` | 漏洞扫描 | nuclei | vuln |
| `SPOOR_SCAN` | JS 静态分析 | spoor | crawl |

### 派生规则（`core.DeriveEligibleWorks`）

**外网 Profile** 的规则表 —— 资产类型到动作的自动映射：

| 资产类型 | 派生的动作 | 前置条件 | 最大深度 |
|---------|-----------|----------|----------|
| SUBDOMAIN | 子域枚举, DNS 解析, CDN 检测 | 类型匹配 | 0（仅种子） |
| IP | DNS 解析（A/AAAA/CNAME）, CDN 检测 | IP 且存活 | -1（无限） |
| 非 CDN IP | 端口扫描 | 存活且非 CDN | 2 |
| IP_PORT | 服务指纹 | — | 2 |
| HTTP_SERVICE | HTTP 指纹, 爬虫(1层), 目录爆破(1层), Nuclei(指纹门控) | 类型匹配 | 2 |
| HTTP_PATH | 爬虫(1层), JS 分析(1层), Nuclei | — | 2 |

**内网 Profile** 与外部基本相同，但：
- 不做 CDN 过滤（不派生 CDN_CHECK）
- 所有 IP 直通端口扫描

## 子包详解

### `core/` — 核心类型

| 文件 | 职责 |
|------|------|
| `asset.go` | DiscoveryAsset 定义、AssetType 枚举、目标分类 `ClassifySeedTarget`、值标准化 |
| `task.go` | TaskAction 枚举、ActionToTool/ActionToStage 映射 |
| `rules.go` | ActionRule 结构、Profile 接口、Profile 实现（internal/external）、DeriveEligibleWorks 函数 |
| `attrs.go` | AssetAttrs 结构（StatusCode/Fingerprinted/Technologies 等） |
| `preconditions.go` | 前置条件函数（isSubdomain/isIP/isHTTPAndFingerprinted 等） |
| `profile_config.go` | ProfileFromConfig — PipelineConfig → Profile 适配 |

### `work/` — WorkItem 存储

**核心文件**：`store.go`
**核心类型**：`Store`

| 方法 | 说明 |
|------|------|
| `TryClaim(ctx, workItem)` | 尝试领取一个 work（CAS，避免重复调度） |
| `MarkDone(workID)` | 标记 work 完成 |
| `AllTerminal(runID)` | 检查 run 的所有 work 是否终态 |

### `queue/` — 优先级队列

| 文件 | 职责 |
|------|------|
| `priority.go` | PriorityQueue 基本操作（Push/Pop/Len） |
| `fair.go` | PopFair / PopFairStaged — 公平调度器，按 bucket 分组限流 |
| `stage_rank.go` | StageRank 枚举 + ActionToStageRank 映射 |

**StageRank 执行顺序**（低值优先）：

```text
Discovery(10) → Subdomain(20) → Resolve(30) → CDN(40) → Port(50)
→ Service(60) → Web(70) → Crawl(80) → Brute(90) → Vuln(100)
```

**调度策略**（`PopFairStaged`）：
1. 找出队列中最低 stage rank（最早阶段）
2. 在该阶段内，按 bucket 公平选择（bucket 内 inflight 最少的优先）
3. 支持 tier 内优先级（high/med/low）

### `pool/` — 批量池化

**核心思想**：将同质输入批量打包为单次 CLI 调用，而非 per-asset 一条 work。

| 组件 | 文件 | 批量策略 |
|------|------|----------|
| `Pool` | `pool.go` | 通用池。缓冲 members → 达到 BatchSize 或 FlushTimeout 时 flush 为批 |
| `IPPortAggregator` | `ip_port_agg.go` | IP:Port 聚合。同 IP 的多个端口合成一次 nmap 调用 |
| `NucleiTagBuckets` | `nuclei_buckets.go` | Nuclei 按 tech 分桶。同 tech 的 URL 合并为一批 nuclei |
| `ProbeTarget` | `probe_target.go` | 探活候选集管理 |

**三层批量**：

| 层级 | Action | 池 | 典型 batch |
|------|--------|----|------------|
| Tier1 | DNS / CDN / Subfinder | `Pool` / `domainpool` | 50–100 行/CLI |
| Tier2 | httpx / nmap / nuclei | `httpPool` / `IPPortAggregator` / `NucleiTagBuckets` | 20–100 URL |
| Tier3 | katana / ffuf / spoor | 1 asset/work（不池化） | 1 |

### `scheduler/` — 调度器

| 文件 | 职责 |
|------|------|
| `limits.go` | `ComputeLimits(seedCount)` 动态计算并发上限。Base=8, +2/seed, 上限50 |
| `ip_throttle.go` | IPThrottler — 同 IP 限流（防止对同一目标发太多请求） |
| `bucket.go` | SeedBucketKey — 种子分桶，负载分散 |

### `seed/` — 种子资产扩展

| 文件 | 职责 |
|------|------|
| `convert.go` | 统一转换为 DiscoveryAsset（含类型推断） |
| `expand.go` | 扩展种子目标（一个 company → 多个 domain/IP） |
| `passive_search.go` | 编排被动搜索（FOFA/Hunter/Quake） |
| `passive.go` | 被动阶段主流程 |
| `boundary_filter.go` | 边界过滤（避免扫到外部未授权目标） |
| `junk_filter.go` | 垃圾域名过滤（`junk_keywords.go` 中的列表） |

### `dedup/` — Run 级去重

**核心文件**：`run_dedup.go`
- 维护 run 级别的已处理资产标准化值集合
- 防止同一资产被多次注入引擎

### `executor/` — 工具执行

| 文件 | 职责 |
|------|------|
| `executor.go` | Executor 接口 + ToolExecutor 实现。Execute()：action → toolID → toolrun.Invoke + ToolCallLog |
| `ffuf.go` | FFUF 输出解析（生成新 HTTP_PATH 资产） |
| `katana.go` | Katana 输出解析（生成 URL/JS 端点） |
| `spoor.go` | Spoor JS 分析输出解析 |

### `stageagg/` — UI Stage 投影

**核心文件**：`aggregator.go`
- 将 Work → Stage 分组，写入 `pipeline_run_stages` 表
- 通过回调推 SSE 事件
- **仅用于 UI 进度条展示，不影响执行逻辑**

### `recovery/` — 孤儿 Run 恢复

**核心文件**：`orphan.go`
- `RecoverOrphanRuns()` — Server 重启时将状态为 `running`/`wind_down` 的 run 标 `failed`

### `domainpool/` — 域名池

**核心文件**：`pool.go`
- 域名级批量池（DNS/CDN 批量处理）
- 与通用 Pool 类似，但针对域名分桶

## 引擎主循环

```text
Run(targets)
│
├── 1. initTier1Pools()  创建各池（hostPool/cdnPool/portPool/domainPool）
├── 2. initTier2Pools()  创建 httpx/http/nuclei 池
├── 3. SeedInitialTargets(targets)  → 注入种子资产
├── 4. processNewAsset(asset)       → DeriveEligibleWorks → Push
│
├── 主调度循环（tick 每 2s）：
│   ├── 检查收敛条件
│   │   ├── idleTimeout 无新资产 → wind_down
│   │   ├── wind_down 且队列空且全 work 终态 → stopped
│   │   └── absoluteTimeout 超时 → cancelled
│   │
│   ├── PopFairStaged() → work item
│   ├── 分配 sem 信号量（并发控制）
│   ├── go executeWork(ctx, work)
│   │   ├── executor.Execute()
│   │   │   ├── actionToToolID → toolrun.Invoke
│   │   │   └── log ToolCall（开始→完成）
│   │   └── onWorkComplete()
│   │       ├── parser.ParseXXX(stdout)
│   │       ├── 新资产 → processNewAsset（递归）
│   │       └── attrs 更新
│   │
│   └── 处理 Pool flush（Tier1/Tier2）
│       └── 批次完成 → onWorkComplete（同上）
│
└── 5. finalizeRun() → flush 所有池 → drain 队列 → 更新 DB 状态
```

## 收敛机制

| 条件 | 行为 |
|------|------|
| `time.Since(lastNewAsset) > idleTimeout`（默认 5min） | `running` → `wind_down` |
| `wind_down` + 队列空 + 全 Work 终态 | `wind_down` → `stopped` |
| wind_down 期间 | 仅允许 Nuclei 和 httpx 执行 |
| `AbsoluteTimeout == 0`（默认） | 不设硬超时 |
| Server 重启 | `recovery.RecoverOrphanRuns` 将 orphan run 标 failed |

## 并发控制

```text
ComputeLimits(seedCount) → GlobalMax（上限 50）
                          → PerBucketMax = 1
                          → ActiveBuckets = 初始 3，每 30s 增加 3
```

## 完整 tick 时序

```text
ticker (2s)
  ├── processPoolFlushes()      处理 Tier1/Tier2 池 flush
  ├── checkConvergence()        检查引擎状态
  ├── limits = ComputeLimits()  计算并发上限
  ├── while sem 有空闲：
  │     work = pq.PopFairStaged()
  │     if no work → break
  │     sem <- struct{}         占用信号量
  │     go executeWorkConcurrent(work)
  │          → executor.Execute()
  │          → onWorkComplete()
  │          → sem <- struct{}  释放
  └── updateMetrics()           更新 DB 指标
```

---

# Phase 4: 数据模型与 API 对照

> 本章列出核心模型结构体及其关联的 API 路径，帮助定位增改点。

## 模型 → DB 表 → API 映射

### 项目

| 模型 | 文件 | DB 表 | 主要字段 |
|------|------|-------|---------|
| `Project` | `project.go` | `projects` | ID, Name, Organization, Purpose, RateLimit, PortRange, DefaultProfile, ScopeBoundaryMode |

**对应 API**：

| 方法 | 路径 | Handler |
|------|------|---------|
| POST | `/projects` | `project_handlers.go` |
| GET | `/projects` | `project_handlers.go` |
| GET | `/projects/{id}` | `project_handlers.go` |
| DELETE | `/projects/{id}` | `project_handlers.go` |

**Service**：`projectSvc`（`internal/service/project.go`）

### 目标

| 模型 | 文件 | DB 表 | 主要字段 |
|------|------|-------|---------|
| `Target` | `target.go` | `targets` | ID, ProjectID, Type(domain/url/ip/cidr/company), Value, Source, Status |

**对应 API**：

| 方法 | 路径 | Handler |
|------|------|---------|
| POST | `/projects/{id}/targets` | `target_handlers.go` |
| GET | `/projects/{id}/targets` | `target_handlers.go` |
| DELETE | `/projects/{id}/targets/{targetId}` | `target_handlers.go` |
| POST | `/projects/{id}/targets/import` | `target_handlers.go` |

**Service**：`targetSvc`（`internal/service/target.go`）

### 资产

| 模型 | 文件 | DB 表 | 主要字段 |
|------|------|-------|---------|
| `Asset` | `asset.go` | `assets` | ID, ProjectID, Type(domain/ip/cidr/url), Value, NormalizedValue, SourceTools, Tags |
| `Port` | `asset.go` | `asset_ports` | ID, AssetID, Port, Protocol, State |
| `WebEndpoint` | `asset.go` | `web_endpoints` | ID, AssetID, URL, Scheme, StatusCode, Title, Technologies |
| `ServicePort` | — | `service_ports` | ID, AssetID, Port, Protocol, ServiceName |

**对应 API**：

| 方法 | 路径 | Handler |
|------|------|---------|
| GET | `/projects/{id}/assets` | `asset_handlers.go` |
| GET | `/projects/{id}/web-endpoints` | `asset_handlers.go` |
| GET | `/projects/{id}/service-ports` | `asset_handlers.go` |
| GET | `/assets/{id}/lineage` | `asset_handlers.go` |

### 资产变更

| 模型 | 文件 | DB 表 | 主要字段 |
|------|------|-------|---------|
| `AssetChange` | `asset_change.go` | `asset_changes` | ID, AssetID, Field, OldValue, NewValue, DetectedAt |

**对应 API**：

| 方法 | 路径 | Handler |
|------|------|---------|
| GET | `/projects/{id}/asset-changes` | `asset_change_handlers.go` |

### 资产关系

| 模型 | 文件 | DB 表 | 主要字段 |
|------|------|-------|---------|
| `AssetRelation` | `asset_relation.go` | `asset_relations` | ID, SourceAssetID, TargetAssetID, RelationType, Source |

**用途**：表达 domain→ip(belongs_to)、port→service(runs)、cidr→ip(contains) 等关系。

### 扫描运行

| 模型 | 文件 | DB 表 | 主要字段 |
|------|------|-------|---------|
| `PipelineRun` | `scan.go` | `pipeline_runs` | ID, ProjectID, Status, EngineState, Config JSON, StartedAt, CompletedAt |
| `ScanWorkItem` | `scan_work.go` | `scan_work_items` | ID, RunID, AssetID, Action, Status, TaskID, BatchMode, BucketKey |
| `ScanTask` | `scan.go` | `scan_tasks` | ID, ProjectID, RunID, Tool, CommandTemplate, Status, ExitCode |

**对应 API**：

| 方法 | 路径 | Handler |
|------|------|---------|
| POST | `/projects/{id}/scan` | `pipeline_handlers.go` |
| GET | `/pipeline/runs` | `pipeline_handlers.go` |
| GET | `/projects/{id}/pipeline/runs/{runId}/metrics` | `scan_metrics_handlers.go` |
| GET | `/projects/{id}/pipeline/runs/{runId}/works` | `scan_work_handlers.go` |
| GET | `/projects/{id}/pipeline/runs/{runId}/tool-calls` | `scan_work_handlers.go` |
| GET | `/runs/{runId}/tasks` | `run_handlers.go` |
| POST | `/pipeline/runs/{runId}/cancel` | `pipeline_handlers.go` |

### 扫描配置（PipelineConfig）

存储为 `projects.pipeline_config` JSON 字段，结构体定义在 `models/scan.go`。

| 配置维度 | 字段 | 示例值 |
|---------|------|--------|
| Nuclei 层数 | `nuclei_scan_depth` | `"tags"` / `"workflow"` / `"both"` |
| Nuclei 速率 | `nuclei_rate_limit` / `nuclei_rate_limit_per_min` / `nuclei_concurrency` | 100 / 0 / 25 |
| FFUF 档次 | `ffuf_tier` | `"off"` / `"small"` / `"medium"` |
| 端口范围 | `port_range` | `"top100"` / `"top1000"` / `"full"` / `"high-risk"` / 自定义 |
| 指纹门控 | `nuclei_require_fingerprint` | true / false |
| 并发参数 | `rate_limit` / `threads` / `timeout` | 150 / 50 / 30 |

### Finding（发现/漏洞）

| 模型 | 文件 | DB 表 | 主要字段 |
|------|------|-------|---------|
| `Finding` | `finding.go` | `findings` | ID, ProjectID, SourceTool, TemplateID, Name, Severity(critical/high/medium/low/info), Status(new/pending_review/confirmed/false_positive/accepted_risk/ignored/reported), DedupKey, RawRequest, RawResponse |
| `Evidence` | `finding.go` | `evidences` | ID, FindingID, Type, Content, Path |
| `FindingTemplate` | `finding_template.go` | `finding_templates` | ID, Title, Severity, Summary, Remediation, CVE, CVSS |

**对应 API**：

| 方法 | 路径 | Handler |
|------|------|---------|
| GET | `/projects/{id}/findings` | `finding_handlers.go` |
| GET | `/findings/{id}` | `finding_handlers.go` |
| PATCH | `/findings/{id}` | `finding_handlers.go` |
| POST | `/findings/{id}/evidence` | `finding_handlers.go` |
| PATCH | `/findings/batch-status` | `finding_handlers.go` |
| GET | `/findings/{id}/curl` | `finding_handlers.go` |
| GET | `/finding-templates` | `finding_template_handlers.go` |
| POST | `/findings/{id}/retest` | `retest_handlers.go` |

**Service**：`findingSvc`（`internal/service/finding.go`）

### 范围规则（Scope）

| 模型 | 文件 | DB 表 | 主要字段 |
|------|------|-------|---------|
| `ScopeRule` | `scope.go` | `scope_rules` | ID, ProjectID, Action(include/exclude), Type, Value |

**对应 API**：

| 方法 | 路径 | Handler |
|------|------|---------|
| GET | `/scope-rules` | `scope_handlers.go` |
| POST | `/scope-rules` | `scope_handlers.go` |
| DELETE | `/scope-rules/{id}` | `scope_handlers.go` |
| POST | `/scope-rules/parse` | `scope_handlers.go` |

### Worker 节点

| 模型 | 文件 | DB 表 | 主要字段 |
|------|------|-------|---------|
| `WorkerNode` | `worker.go` | `worker_nodes` | ID, Endpoint, Status(online/offline), MaxConcurrency, RunningTasks, LastHeartbeat |

**对应 API**：

| 方法 | 路径 | Handler |
|------|------|---------|
| GET | `/workers` | `worker_handlers.go` |
| POST | `/workers/register` | `worker_handlers.go` |
| POST | `/workers/{id}/heartbeat` | `worker_handlers.go` |
| GET | `/workers/{id}/tasks/poll` | `worker_handlers.go` |
| POST | `/tasks/{id}/result` | `worker_handlers.go` |

### 其他模型

| 模型 | 文件 | DB 表 | 用途 |
|------|------|-------|------|
| `Dictionary` | `dictionary.go` | `dictionaries` | ffuf 字典词库（含 builtin） |
| `HttpxFingerprint` | `httpx_fingerprint.go` | `httpx_fingerprints` | HTTPX 指纹库（含 builtin） |
| `NucleiCustomSource` | `nuclei_custom.go` | `nuclei_custom_sources` | Nuclei 模板源管理 |
| `EngineCredential` | `engine.go` | `engine_credentials` | FOFA/Hunter/Quake API 凭证 |
| `ExcludedDomain` | `exclude.go` | `excluded_domains` | 全局排除域名列表（含 builtin） |
| `Notification` | `notification.go` | `notifications` | 通知记录 |
| `ToolCallLog` | `tool_call_log.go` | `tool_call_logs` | 工具调用审计日志 |
| `Signal` | `signal.go` | `signals` | 扫描信号/事件 |
| `Watch` | `watch.go` | `watches` | 资产变更监听 |
| `HealthCheck` | `health.go` | — | 健康检查状态（不持久化） |

## 核心 API 调用路径（请求 → 响应）

### 1. 创建项目 → 添加目标 → 启动扫描

```text
POST /projects
  → project_handlers.handleCreateProject
  → projectSvc.Create(&Project{Name, Organization, Purpose})
  → db.queries.CreateProject
  Response: 201 {id: "proj_xxx"}

POST /projects/{id}/targets
  → target_handlers.handleCreateTarget
  → targetSvc.Create(&Target{Type: "domain", Value: "example.com"})
  → db.queries.CreateTarget
  Response: 201 {id: "tgt_xxx"}

POST /projects/{id}/scan
  → pipeline_handlers.handleStartScan
  → scanengine.New(queries, runner, tools, ...).Run(targets)
  → 异步返回 run_id
  Response: 202 {run_id: "run_xxx"}
```

### 2. 查看扫描进度

```text
GET /projects/{id}/events (SSE)
  → sse.handleProjectSSE
  → 实时推送 stage_change / work_complete / finding_new 事件

GET /projects/{id}/pipeline/runs/{runId}/metrics
  → scan_metrics_handlers
  → db.queries.GetPipelineRunMetrics
  Response: {engine_state, total_works, completed_works, ...}

GET /projects/{id}/pipeline/runs/{runId}/works?page=1&page_size=50
  → scan_work_handlers
  → db.queries.ListScanWorkItems
  Response: {items: [...], total, page, page_size}
```

### 3. 查看与管理发现结果

```text
GET /projects/{id}/findings?severity=high&status=new
  → finding_handlers.handleListFindings
  → findingSvc.ListByProject
  → db.queries.ListFindings
  Response: {items: [...Finding], total, page, page_size}

PATCH /findings/{id}
  → finding_handlers.handlePatchFinding
  → findingSvc.UpdateStatus("confirmed")
  Response: 200 {id, status: "confirmed"}

PATCH /findings/batch-status
  → finding_handlers.handleBatchStatusUpdate
  → findingSvc.BatchUpdateStatus
  Response: 200 {affected: N}
```

### 4. 导出报告

```text
GET /projects/{id}/reports/export.md
  → report_handlers.handleExportReport
  → db.queries → report.GenerateMarkdown(finding模板聚合)
  Response: text/markdown 文件
```

---

# Phase 5: 新增功能标准操作指南

> 本章提供了 6 种常见新增功能场景的**精确文件路径与步骤**。
> coding agent 按照步骤直接定位到对应文件即可增改。

## 1. 加一个新安全工具（完整流程）

以添加一个新的安全扫描工具 `snewtool` 为例：

```text
步骤 1: 定义工具注册信息
  文件: tools/snewtool.yaml
  内容: ID/binary/parameters/literals/presets
  → 参考: tools/nuclei.yaml 或 tools/ffuf.yaml

步骤 2: 添加输出解析器
  文件: internal/parser/snewtool.go
  内容: 解析函数 ParseSnewtool(stdout []byte) → []SnewtoolResult
  → 如果输出是 JSONL，可以直接用 parseJSONLines 泛型函数
  → 参考: internal/parser/nuclei.go

步骤 3: 注册 Executor（新路径）
  文件: internal/scanengine/executor/executor.go
  改动: actionToToolID() 添加映射
  → 若需新的 Action 类型，先在 core/task.go 添加 TaskAction 枚举

步骤 4: 添加派生规则（如果需要引擎自动调度）
  文件: internal/scanengine/core/rules.go
  改动: Profile.Rules() 添加 ActionRule{Action, Enabled, MaxDepth, Precondition}

步骤 5: 添加配置字段
  文件: internal/models/scan.go (PipelineConfig 添加新字段)
  文件: internal/scanconfig/config.go (配置默认值)

步骤 6: 更新 worker 旧版命令（维护模式）
  文件: internal/worker/commands.go
  添加: BuildSnewtoolCommand(...) 函数

步骤 7: 更新 toolguard 白名单（若为新二进制）
  文件: internal/toolguard/allowlist.go
  改动: NewAllowlist() 添加 binary 名
  或 toolguard.NewAllowlistFromBinaries(reg.Binaries()) 自动包含

步骤 8: 数据模型
  - 需要新模型: 在 internal/models/ 添加文件
  - 需要新查询: 在 internal/db/queries_*.go 添加方法
  - 需要新表: 见「加 DB 迁移」

步骤 9: 前端
  - 新配置选项: frontend/src/pages/ 下的 Scans/Settings 页面
  - 新展示页: 新增 pages/ 页面 + hooks/ API 调用
```

## 2. 加一条新 API

```text
步骤 1: 定义数据模型
  文件: internal/models/xxx.go
  结构体: 定义请求/响应结构体

步骤 2: 添加 DB 查询方法
  文件: internal/db/queries_xxx.go
  方法: CRUD 操作（ListByXxx / GetXxx / CreateXxx / UpdateXxx / DeleteXxx）
  测试: queries_xxx_test.go

步骤 3: 添加 Service 层（可选，当业务逻辑复杂时）
  文件: internal/service/xxx.go
  接口: internal/service/interfaces.go 定义新接口（方便 mock）
  适配器: internal/service/adapter.go 注册到 server

步骤 4: 添加 Handler
  文件: internal/api/xxx_handlers.go
  路由: server.go 的 Register() 函数注册新路径
  → 新文件需要在 server.go 的 Server struct 注释中说明依赖

步骤 5: 同步更新文档
  - internal/api/README.md: Handler 文件总览表 + 字段反向索引
  - docs/current/system-reference.md: 对应章节

步骤 6: 前端（若有）
  - hooks/ 添加 API 调用
  - pages/ 或 components/ 添加 UI
  - 类型定义: lib/ 或 components/ 下
```

## 3. 加一个 DB 迁移

DB 自动迁移系统位于 `internal/db/`，每次新增 `vN.go` 文件 + 在 `db.go` 的 `migrate()` 函数中注册。

```text
步骤 1: 创建迁移文件
  文件: internal/db/v45.go  （版本号 = 当前最新 + 1）
  签名: func migrateV45(db *sql.DB) error
  内容: CREATE TABLE / ALTER TABLE / 数据迁移 SQL

步骤 2: 注册迁移
  文件: internal/db/db.go → migrate() 函数
  添加:
    if version < 45 {
        if err := migrateV45(db); err != nil { ... }
        if _, err := db.Exec("PRAGMA user_version = 45"); err != nil { ... }
        version = 45
    }

步骤 3: 添加 DB 查询方法（若需要操作新表）
  文件: internal/db/queries_xxx.go
  或新建 queries_xxx.go 然后在 queries.go 的 Queries struct 中注册

步骤 4: 测试
  文件: internal/db/queries_xxx_test.go
  或在已有测试文件添加

注意事项:
  - 迁移必须是幂等的（可重入）
  - 使用 CREATE TABLE IF NOT EXISTS
  - ALTER TABLE 前检查列是否存在（SELECT pragma_table_info）
  - 迁移 v1→v44 已有实现，v44 为当前最新
```

## 4. 加一个新的扫描动作

扫描动作 = `TaskAction` 枚举 + `DeriveEligibleWorks` 规则 + Executor 实现。

```text
步骤 1: 添加 Action 枚举
  文件: internal/scanengine/core/task.go
  添加: const ActionMyNewScan TaskAction = "MY_NEW_SCAN"
  映射:
    ActionToTool[ActionMyNewScan] = "tool_id"
    ActionToStage[ActionMyNewScan] = "stage_name"

步骤 2: 添加 StageRank（确定执行优先级）
  文件: internal/scanengine/queue/stage_rank.go
  添加常量（如 StageMyNew StageRank = 95）+ ActionToStageRank 映射

步骤 3: 添加派生规则
  文件: internal/scanengine/core/rules.go
  在 Profile.Rules() 中添加:
    {Action: ActionMyNewScan, Enabled: true, MaxDepth: 2, Precondition: isHTTPService}

步骤 4: 添加 Executor 映射
  文件: internal/scanengine/executor/executor.go
  在 actionToToolID() 中添加:
    case string(core.ActionMyNewScan): return "tool_id"

步骤 5: 添加输出解析（若需）
  文件: internal/scanengine/executor/xxx.go
  或在 internal/parser/ 添加通用解析器
```

## 5. 加一个新的被动搜索引擎

```text
步骤 1: 实现 Engine 接口
  文件: internal/search/xxx.go（如 fofa.go / hunter.go / quake.go）
  实现:
    type XXXEngine struct { ... }
    func (e *XXXEngine) Search(ctx, query, page) → []SearchResult, error

步骤 2: 注册到引擎工厂
  文件: internal/search/engine.go
  在 NewEngine 或 engine 映射中添加新引擎

步骤 3: 添加到被动搜索编排
  文件: internal/scanengine/seed/passive_search.go
  在 runPassiveSearch() 中并行调用新引擎

步骤 4: API 凭证管理（若需 API Key）
  文件: internal/api/engine_handlers.go
  db 表: engine_credentials（已存在，按 engine 名称区分）

步骤 5: 前端（若需）
  在 Engines 设置页面添加配置项
```

## 6. 代码修改通用规则

| 修改类型 | 必须检查 | 必须更新 |
|---------|---------|---------|
| 改 handler | `internal/api/README.md` 的反向索引 | 同文件的 Handler 表 + 字段索引；`server.go` 字段注释 |
| 改 Server 字段 | `internal/api/README.md` 字段反向索引 | server.go 注释 + API README 反向索引 |
| 新路由 | `server.go` Register 函数 | `internal/api/README.md` |
| 新 model | `internal/models/` 现有相似模型 | 当前文档 Phase 4 对应表 |
| 新扫描工具 | `tools/*.yaml` 现有定义 | 当前文档 Phase 2 工具表 |
| 加环境变量 | `main.go` / `server.go` 读取点 | 当前文档 Phase 1 环境变量表 |

> 完整约束见项目 `CLAUDE.md`「文档同步约束」小节。
