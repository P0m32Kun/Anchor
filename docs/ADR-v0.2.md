# v0.2 架构决策记录（ADR）

> 本文档基于 v0.1 代码库现状，为第二阶段设计补齐关键架构决策。
> 状态：待确认

---

## 0. v0.1 现状摘要

| 组件 | 技术栈 | 现状 |
|------|--------|------|
| 前端 | React 18 + Vite + Tailwind v3 + Zustand + React Router v6 | 功能页集合 |
| 桌面壳 | Tauri v2（最小化，仅加载 WebView）| 无业务逻辑 |
| 后端核心 | Go 1.22，HTTP server `:17421` | 同进程 Worker |
| 数据库 | SQLite3（WAL 模式），`~/.anchor/anchor.db` | 已有 10+ 张表 |
| 查询层 | 手写 SQL（`internal/db/queries.go`）| 无 ORM |
| 工具执行 | `worker.Runner`，`os/exec` 直接调用 CLI | 单进程 |
| 事件推送 | SSE `/events` | 已有 |
| 文件存储 | `~/.anchor/workdirs/{project_id}/{task_id}/` | 原始输出 + 截图 |
| 工作流 | `AssetDiscoveryWorkflow`（subfinder→httpx→naabu）+ `WebScreeningWorkflow`（nuclei）| 硬编码串行 |

**关键结论**：v0.1 的核心架构（Go HTTP server + SQLite + SSE + 文件系统）**不需要推翻**，v0.2 是在此基础上的增量扩展。

---

## ADR-1：Worker 架构——单二进制、双模式

### 决策（已确认）

使用**同一个 Go 二进制文件**，通过命令行标志区分两种运行模式：

```
# 模式 A：核心服务器（默认，v0.1 已有的行为）
./anchor
  → 启动 HTTP server :17421
  → 管理 SQLite、项目数据、API
  → 协调 Worker

# 模式 B：Worker（v0.2 新增）
./anchor --worker --core-url http://192.168.1.10:17421 --token abc123
  → 不启动 API server
  → 连接核心服务器，拉取任务、执行、回传结果
  → 本地只保留临时 workdir，无持久数据库
```

### 本地 Worker（内网场景）

内网场景：带着笔记本进入客户内网，Worker 和桌面端都在这台笔记本上运行。

```
[桌面端 Tauri] --HTTP--> [核心服务器 :17421] --spawn--> [本地 Worker 进程]
                                               ↑___________↓
                                                通过 localhost HTTP 回连
```

核心服务器通过 `exec.Command` 启动本地 Worker 子进程，Worker 通过 `--core-url http://localhost:17421` 回连。无需 token（localhost 可信）。

### 远程 Worker（外网场景）

外网场景：桌面端在本地，Worker 部署在云主机上。

```
[桌面端 Tauri] --HTTP--> [核心服务器 :17421] <--HTTP 轮询-- [远程 Worker（云主机）]
```

远程 Worker 由用户在云主机上手动部署启动，通过 `--core-url` 指向桌面端的可达地址（需有公网 IP 或 VPN）。必须 token 注册。

### 不支持的场景

**NAT 穿透**：桌面端在家无公网 IP，Worker 在客户内网，双向不可达。v0.2 不支持，v0.3+ 考虑中继/打洞。

### 为什么不用 Tauri Sidecar？

Tauri Sidecar（`tauri-plugin-shell`）通过 stdin/stdout 通信，适合 Rust ↔ 外部进程的紧耦合场景。但我们的架构已经是**前端 ↔ Go HTTP server**的松耦合模式，Worker 也复用 HTTP 通信。引入 Sidecar 会增加一层不必要的复杂度（Rust 中转 stdin/stdout → 再 emit 到前端）。保持"Go 二进制自己管自己"更简洁。

### Worker 进程管理

v0.1 的 `worker.Runner` 已有进程管理能力（`procs map[string]*exec.Cmd`、`doneChs`）。v0.2 扩展为：

- **本地 Worker**：核心服务器通过 `runner.spawnLocalWorker(workerID)` 启动子进程，记录在 `procs` 中。崩溃后自动重启（3 次 backoff）。
- **远程 Worker**：Worker 进程自主管理，核心服务器通过心跳超时检测存活状态。
- 异常退出：核心服务器检测到后，前端 SSE 推送事件。

---

## ADR-2：通信协议——HTTP REST + 长轮询

### 决策（已确认）

Worker ↔ 核心服务器之间使用 **HTTP REST + 长轮询**（long polling）进行双向通信。

**本地 Worker** 通过 localhost HTTP，无需认证。
**远程 Worker** 通过 HTTP + `Authorization: Bearer {token}`。

### 接口设计

**Worker 拉取任务（长轮询）**

```http
GET /workers/{worker_id}/tasks/poll?timeout=30s
Authorization: Bearer {token}  # 远程 Worker 必填，本地 Worker 可选

# 核心服务器 hold 住请求，直到有任务或超时
# 响应 200：{ "task_id": "...", "type": "scan", "payload": {...} }
# 响应 204：无任务，Worker 立即重试
```

**Worker 上报心跳**

```http
POST /workers/{worker_id}/heartbeat
Authorization: Bearer {token}  # 远程 Worker 必填

{ "status": "idle|busy", "capabilities": [...], "tool_versions": {...} }
```

**Worker 上报任务结果**

```http
POST /tasks/{task_id}/result
Authorization: Bearer {token}  # 远程 Worker 必填

{
  "status": "completed|failed|partial_success",
  "steps": [...],
  "artifacts": [...],
  "screenshot_paths": ["screenshots/abc.png"]
}
```

**核心服务器获取 Worker 文件（截图等）**

```http
GET /workers/{worker_id}/files/{file_path}
Authorization: Bearer {token}  # 远程 Worker 必填

# 返回二进制文件流
```

### 为什么不用 WebSocket / gRPC？

| 方案 | 问题 |
|------|------|
| WebSocket | 双向实时性好，但防火墙/代理可能阻断 WebSocket 升级；重连逻辑复杂 |
| gRPC | 内网可能不通（HTTP/2 严格）；增加 protobuf 依赖；调试困难 |
| HTTP 短轮询 | Worker 频繁请求，浪费资源 |
| **HTTP 长轮询** | ✅ 兼容所有代理和防火墙；无额外依赖；实现简单；Worker 侧只需标准 HTTP client |

### 安全模型

- **本地 Worker**：localhost 通信，无需 token，依赖操作系统进程隔离
- **远程 Worker**：
  - Token 注册：用户桌面端生成 token → 复制到远程 Worker 启动参数
  - Token 传输：`Authorization: Bearer {token}`
  - 传输加密：v0.2 默认 HTTP，建议远程 Worker 走 VPN/隧道；v0.3 考虑 mTLS
- Worker 白名单：核心服务器只下发 tool 在白名单中的任务
- 命令参数审计：Worker 上报的参数经过脱敏后写入 audit_log

---

## ADR-3：任务执行模型——Run / ScanTask / Step 三层

### 决策

明确三层结构：

```
ToolTemplate（预设：选哪些工具 + 速率）
    ↓ 用户选择模板 + 目标，点击"运行"
Run（一次扫描执行，对应 v0.1 的 ScanPlan）
    ↓ 根据模板生成
ScanTask（每个工具一个任务）
    ↓ 实际执行
Step（任务内的执行阶段）
```

### 与 v0.1 的映射

| v0.1 | v0.2 | 说明 |
|------|------|------|
| `ScanPlan` | `Run`（保留表名 scan_plans，扩展字段） | 一次扫描执行 |
| `ScanTask` | `ScanTask`（保留，增加 worker_id 必填、step 关联） | 单个工具的执行实例 |
| — | `ScanStep`（新增表） | 任务执行阶段 |
| — | `ToolTemplate`（新增表） | 预设模板 |

### Step 模型

```sql
CREATE TABLE scan_steps (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL REFERENCES scan_tasks(id) ON DELETE CASCADE,
    name TEXT NOT NULL CHECK(name IN ('scope_check','prepare_input','run_tool','collect_artifacts','parse_output','normalize_result','score_result','cleanup')),
    status TEXT DEFAULT 'pending' CHECK(status IN ('pending','running','completed','failed','skipped')),
    started_at DATETIME,
    finished_at DATETIME,
    error_code TEXT,
    error_summary TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### Run 的生成逻辑

用户选择「外网标准初筛」模板 → 系统生成一个 Run → 根据模板中的工具列表生成 4 个 ScanTask（Subfinder、Naabu、httpx、Nuclei）→ 任务按依赖关系串行/并行执行

依赖关系：
- Subfinder 无依赖（直接从目标域名开始）
- Naabu 依赖 Subfinder 的输出（域名→IP）
- httpx 依赖 Naabu 的输出（IP+端口）
- Nuclei 依赖 httpx 的输出（存活 URL）

### 为什么不用 DAG 调度器？

v0.2 的工具依赖是固定的线性链，不需要通用 DAG。硬编码依赖关系更简单可靠。如果未来需要复杂 DAG（v0.3+），再引入调度器。

---

## ADR-4：截图数据流——Worker 截图、核心服务器收集

### 决策

```
[Worker 端]
  Rod 截图 → 保存到 Worker 本地 {workdir}/screenshots/{asset_id}.png
  任务完成时 → 上报 screenshot_paths 列表

[核心服务器]
  收到任务结果 → 通过 HTTP 从 Worker 拉取截图文件
  → 保存到 ~/.anchor/projects/{project_id}/screenshots/
  → 生成缩略图（前端展示用）
  → SQLite 记录 screenshot 元数据

[前端]
  通过 API 获取缩略图 BASE64 或文件 URL
  点击放大时获取原图
```

### 截图触发时机

- **自动截图**：ToolTemplate 中勾选"截图"时，httpx 发现 Web 资产后自动 Rod 截图
- **手动截图**：资产列表中点击"截图"按钮，对指定 URL 触发一次截图任务

### 缩略图策略

Worker 端只生成**原图**（1920×1080 或 viewport 大小）。缩略图（200px 宽）由核心服务器生成，使用 Go 的 `image` 标准库或 `github.com/nfnt/resize`。

### 数据库存储

```sql
CREATE TABLE screenshots (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    asset_id TEXT REFERENCES assets(id) ON DELETE SET NULL,
    task_id TEXT REFERENCES scan_tasks(id) ON DELETE SET NULL,
    url TEXT NOT NULL,
    original_path TEXT NOT NULL,      -- ~/.anchor/projects/{pid}/screenshots/{id}_orig.png
    thumbnail_path TEXT NOT NULL,     -- ~/.anchor/projects/{pid}/screenshots/{id}_thumb.png
    width INTEGER,
    height INTEGER,
    taken_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

---

## ADR-5：NetworkProfile 拆分——动态画像 vs 静态配置

### 决策

将原设计中混杂的 NetworkProfile 拆分为两个实体：

```sql
-- 实体 A：WorkerNetworkProfile（Worker 级，动态，健康检查上报）
CREATE TABLE worker_network_profiles (
    id TEXT PRIMARY KEY,
    worker_id TEXT NOT NULL REFERENCES worker_nodes(id) ON DELETE CASCADE,
    type TEXT NOT NULL CHECK(type IN ('external','internal','restricted','lab')),
    egress_ip TEXT,
    dns_ok BOOLEAN,
    proxy_enabled BOOLEAN,
    proxy_ok BOOLEAN,
    target_network_reachable BOOLEAN,
    platform TEXT,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 实体 B：ProjectScanConfig（项目级，静态，用户配置）
CREATE TABLE project_scan_configs (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL UNIQUE REFERENCES projects(id) ON DELETE CASCADE,
    authorized_networks TEXT,        -- JSON ["10.0.0.0/8", "192.168.1.0/24"]
    business_tags TEXT,              -- JSON ["生产区", "OA系统"]
    environment TEXT DEFAULT 'unknown' CHECK(environment IN ('production','staging','test','unknown')),
    max_concurrency INTEGER DEFAULT 10,
    allowed_scan_windows TEXT,       -- JSON [{"start":"22:00","end":"06:00"}]
    screenshot_enabled BOOLEAN DEFAULT FALSE,
    directory_bruteforce_enabled BOOLEAN DEFAULT FALSE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### 为什么拆分？

原设计把"Worker 当前能访问什么网络"和"这个项目允许扫什么网段"混在一起。前者是**运行时动态信息**（Worker 的网络环境），后者是**授权静态配置**（项目的合法范围）。混在一个表里有几个问题：

1. 一个 Worker 可能被多个项目使用（虽然 v0.2 先不处理，但架构应该支持）
2. `authorized_networks` 应该跟着项目走，而不是跟着 Worker 走
3. `max_concurrency` 应该是项目配置，但执行时由 ToolTemplate 覆盖

---

## ADR-6：数据模型补充

### 新增表清单

| 表名 | 用途 | v0.2 必须？ |
|------|------|-----------|
| `worker_nodes` | Worker 注册信息 | ✅ M1 |
| `worker_health_checks` | 工具健康检查结果 | ✅ M1 |
| `worker_network_profiles` | Worker 网络画像 | ✅ M2 |
| `project_scan_configs` | 项目扫描配置 | ✅ M2 |
| `scan_steps` | 任务执行步骤 | ✅ M1 |
| `tool_templates` | 工具模板预设 | ✅ M3 |
| `retest_runs` | 复测记录 | ✅ M4 |
| `saved_views` | 前端筛选视图 | 🟡 很值得做 |
| `screenshots` | 截图元数据 | ✅ M3 |

### 新增字段（现有表）

| 表 | 新增字段 | 说明 |
|---|---------|------|
| `scan_tasks` | `steps_json TEXT` | 存储步骤状态摘要（冗余，方便列表查询） |
| `scan_tasks` | `tool_template_id TEXT` | 关联使用的模板 |
| `assets` | `first_seen DATETIME` | 首次发现时间 |
| `assets` | `last_seen DATETIME` | 最后发现时间 |
| `assets` | `screenshot_id TEXT` | 关联最新截图 |
| `findings` | `raw_request TEXT` | 原始请求（复测时需要） |
| `findings` | `raw_response TEXT` | 原始响应 |
| `findings` | `matched_template TEXT` | 匹配的 Nuclei 模板 ID |

---

## ADR-7：v0.1 → v0.2 迁移策略

### 迁移方式

SQLite 迁移通过 `internal/db/db.go` 中的 `migrate()` 函数扩展：

```go
func migrate(db *sql.DB) error {
    // v0.1 基础 schema（已有）
    if err := migrateV01(db); err != nil { return err }
    // v0.2 增量 schema（新增）
    if err := migrateV02(db); err != nil { return err }
    return nil
}
```

### 具体变更

1. **新增表**：`worker_nodes`, `worker_health_checks`, `worker_network_profiles`, `project_scan_configs`, `scan_steps`, `tool_templates`, `retest_runs`, `saved_views`, `screenshots`
2. **新增列**：`scan_tasks.worker_id`（已有，但 v0.1 为 NULL，v0.2 本地 Worker 默认填本地 worker ID）
3. **新增列**：`assets.first_seen`, `assets.last_seen`，默认值为 `created_at`
4. **数据填充**：
   - `tool_templates`：INSERT 4 个内置模板
   - `worker_nodes`：INSERT 一条本地 Worker 记录（id=`local`，mode=`local`）
   - `scan_tasks.worker_id`：UPDATE 所有已有任务，设置 worker_id=`local`
5. **零数据丢失**：所有变更都是 ADD，无 DROP/ALTER DELETE

---

## ADR-8：资产增量对比机制

### 决策

通过 `first_seen` + `last_seen` + 自然键匹配实现增量对比。

### 算法

```
新扫描结果 AssetList
现有资产库 AssetDB

对于每个新资产 a:
  自然键 = a.host + ":" + a.port（或 URL）
  如果在 AssetDB 中找到匹配:
    更新 last_seen = now
    标记为 "existing"
  否则:
    插入新记录，first_seen = now, last_seen = now
    标记为 "new"

对于 AssetDB 中 last_seen < 本次扫描开始时间 的资产:
  标记为 "disappeared"
```

自然键选择：
- IP + Port → `{ip}:{port}`
- URL → 标准化后的 URL（去协议、去尾部斜杠）
- Domain → 小写域名

---

## 需要你确认的问题

## 已确认决策

| 问题 | 用户选择 | 影响 |
|------|---------|------|
| 1. Worker 部署范围 | **外网用远程 Worker，内网用本地 Worker**。放弃 NAT 穿透 | 保留远程 Worker 的 token/心跳/长轮询，但明确不支持 NAT 穿透场景 |
| 2. Chromium 部署 | **A**：启动时自动下载，无网环境截图不可用，Worker 页面提醒 | 截图是可选功能，不阻塞核心流程 |
| 3. 本地 Worker 启动 | **C**：默认自动启动，可设置中关闭改为手动 | 零配置开箱即用，高级用户可控 |

### 问题 1 的架构影响

Worker 架构区分两种模式：

| | 本地 Worker | 远程 Worker |
|--|------------|------------|
| 场景 | 内网（笔记本带进现场） | 外网（云主机） |
| 启动 | 核心服务器自动 spawn | 用户手动部署启动 |
| 通信 | localhost HTTP | HTTP 长轮询 |
| 认证 | 无需 token | Bearer token |
| 心跳 | 按需健康检查 | 30s 心跳 |
| 崩溃恢复 | 核心服务器自动重启 | 前端提示，用户手动处理 |
| 文件回传 | localhost HTTP | HTTP 文件下载 |

---

## 确认后下一步

以上 3 个问题确认后，我会更新本文档为「已确认」状态，并生成：

1. `internal/db/migrate_v02.go` — v0.2 数据库迁移
2. `internal/models/models_v02.go` — 新增数据模型
3. `docs/API-v0.2.md` — Worker 通信 API 详细规范

---

## 附录：v0.2 整体架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                        桌面端（Tauri）                           │
│  ┌─────────────────┐  HTTP  ┌─────────────────────────────────┐ │
│  │ React 前端       │◄──────►│ Go 核心服务器 :17421             │ │
│  │ (Dashboard 等)   │  REST  │  • SQLite 数据库                 │ │
│  └─────────────────┘  SSE   │  • 项目/资产/Finding 管理         │ │
│                              │  • Worker 协调                   │ │
│                              │  • 文件收集（截图等）              │ │
│                              └──────────────┬──────────────────┘ │
└─────────────────────────────────────────────┼────────────────────┘
                                              │
                    ┌─────────────────────────┼─────────────────────────┐
                    │ spawn                   │ HTTP 长轮询              │
                    ▼                         ▼                         ▼
           ┌─────────────────┐      ┌─────────────────┐      ┌─────────────────┐
           │ 本地 Worker      │      │ 远程 Worker A   │      │ 远程 Worker B   │
           │ (子进程)         │      │ (客户内网)      │      │ (云主机)        │
           │ 同机器，localhost│      │ 网络可达桌面端   │      │ 公网 IP         │
           └─────────────────┘      └─────────────────┘      └─────────────────┘
```
