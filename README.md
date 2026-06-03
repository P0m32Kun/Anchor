# Anchor — 目标中心自动化安全测试工作台

> 面向授权安全测试的目标中心工作台，通过编排成熟开源工具、强制范围校验、统一结果模型、人工验证队列和报告生成，减少安全人员在工具切换、数据整理、证据归档和报告交付上的重复劳动。

## 完整扫描流程

```
目标输入 → Scope Check → 资产发现 → Web 初筛 → 人工验证 → 报告导出
```

| 阶段            | 工具                       | 输出                                       |
| --------------- | -------------------------- | ------------------------------------------ |
| **Scope Check** | 自研引擎                   | ScopeDecision (allow/deny)                 |
| **资产发现**    | Subfinder → DNSx → CDNcheck → Naabu → nmap -sV → httpx | Asset / WebEndpoint / Port / Service |
| **Web 初筛**    | Nuclei（指纹驱动模板筛选） | Finding / Evidence                         |
| **人工验证**    | 前端队列                   | confirmed / false_positive / accepted_risk |
| **报告导出**    | 自研生成器                 | Markdown / JSON                            |

**核心设计**：指纹驱动 Nuclei 模板筛选 — httpx 识别的技术栈（WordPress/nginx/Apache Druid 等）精确映射到 Nuclei `-tags`，无指纹目标自动跳过，避免全量扫描。

**当前执行模型**：扫描由资产驱动引擎调度。目标先进入资产图，再按资产类型派生 `Work(资产 × 动作)`，工具输出继续回注新资产；Nuclei 命中写入 Findings。项目 scope 为 **仅排除**（未命中 exclude 默认允许）。前端「扫描执行」页展示的是运行观察台：引擎状态、扫描动作进度和 Work Items 明细；阶段只作为 UI 聚合标签，不代表固定线性流水线。设计说明见 [`docs/current/asset-driven-remediation-design.md`](docs/current/asset-driven-remediation-design.md)。

## 技术栈

| 层级          | 技术                                             |
| ------------- | ------------------------------------------------ |
| Web 前端      | React 18 + TypeScript + Tailwind CSS + Nginx     |
| 状态管理      | Zustand                                          |
| 本地/远程服务 | Go 1.26                                          |
| 数据库        | SQLite (WAL 模式)                                |
| 实时推送      | SSE (Server-Sent Events)                         |
| 部署          | Docker + docker-compose + install.sh             |

## 文档入口

当前文档入口统一收敛在 [`docs/README.md`](docs/README.md)。

- Agent 迭代指南：[`docs/current/agent-guide.md`](docs/current/agent-guide.md)
- 当前计划：[`docs/current/plan.md`](docs/current/plan.md)
- 当前架构：[`docs/current/architecture.md`](docs/current/architecture.md)
- 候选设计索引：[`docs/current/design/README.md`](docs/current/design/README.md)
- 历史归档：[`docs/archived/`](docs/archived/)

## 快速开始

### Docker 一键部署（推荐）

```bash
# 交互式安装向导（支持 Server / Worker / Server+Worker 三种模式）
bash install.sh
```

安装向导会自动：
1. 检测 Docker 环境
2. 选择部署模式
3. 配置端口和 API Token
4. 从阿里云 ACR 拉取预构建镜像（国内加速）
5. 启动容器并等待健康检查

完成后浏览器访问 `http://localhost` 即可使用。

**管理命令：**

```bash
bash install.sh status   # 查看状态
bash install.sh logs     # 查看日志
bash install.sh down     # 停止服务
```

### 本地开发

> 本地开发直接运行 Go 二进制，不经过 Docker。

**运行后端：**

```bash
# 启动 server
go run .
# 服务监听 :17421，数据目录 ~/.anchor

# 或本地 worker 连接本地 server
make run-worker
```

**运行前端：**

```bash
cd frontend
npm install
npm run dev
# 打开 http://localhost:5173
```

**构建：**

```bash
make build    # 构建 Go 后端
make test     # 运行测试
```

**本地 E2E 测试（使用新代码）：**

```bash
# 首次使用或工具版本更新时，构建 base 镜像（耗时较长，只需执行一次）
make build-base

# 日常测试迭代（约 10-30 秒）
make build-linux    # 交叉编译 Linux 二进制
make build-fast     # 快速构建 Docker 镜像
make e2e-local      # 启动 E2E 测试环境

# 运行测试
make test-e2e       # 运行 Playwright E2E 测试

# 停止环境
make e2e-local-down
```

**分层构建架构**：
- `anchor-server-base` / `anchor-worker-base`：预装运行时依赖，极少更新
- `anchor-server:local` / `anchor-worker:local`：基于 base，仅 COPY 二进制，构建速度快

## 部署架构

三个 Docker 镜像通过 docker-compose 编排：

```
[浏览器] --HTTP--> [Nginx :80] --/api/--> [Server :17421] <--HTTP 长轮询-- [Worker]
                           |
                     [React 静态文件]
```

- **Frontend**：Nginx 静态 serve React 构建产物 + `/api/` 反向代理到 Server
- **Server**：纯管理面，负责 API、任务调度、数据持久化
- **Worker**：预装所有安全工具，通过长轮询拉取任务执行

**三种部署模式：**

| 模式 | 适用场景 |
|------|---------|
| Server Only | VPS 部署，Worker 远程连接 |
| Worker Only | 远程扫描节点，连接已有 Server |
| Server+Worker | 本地开发/测试，完整功能 |

Worker **不需要公网 IP**，只要 outbound 能访问 Server 即可（任务拉取、心跳、结果上报均为 Worker → Server）。

**Server 是否需要访问 Worker HTTP？** 任务调度与结果回传不依赖 Server 入站连 Worker。若要在 UI 中实时查看远端 Worker 上**仍在运行**任务的 stdout/stderr，Server 会按 Worker 注册时上报的 `endpoint` 反向请求 Worker HTTP（见 `internal/api/task_output_handlers.go`）。因此：

- 仅跑扫描、不看实时日志：Worker 只需能访问 Server，无需对 Server 暴露端口。
- 需要实时任务输出：Worker 的 `endpoint` 必须对 Server 可达（内网 IP/主机名、Docker 服务名 `worker`、或 Tailscale 等）。

**Docker 镜像 tag 策略**（与 GitHub Release 对齐）：

- 推送 `v*` tag 后：`release.yml` 上传 `anchor-linux-{amd64,arm64}`；`docker-push.yml` 在 Release 成功后 checkout **该 tag**（非 main HEAD），并以同一版本构建镜像。
- ACR 镜像同时打 `anchor-*:<tag>` 与 `anchor-*:latest`；`Dockerfile.server` / `Dockerfile.worker` 通过 `RELEASE_VERSION` build-arg 从 GitHub Release 下载对应 tag 的二进制（CI 中不会默认拉 `latest` 资产）。
- 本地 `install.sh` 默认拉 `:latest`；要锁定版本可改 compose/`ANCHOR_REGISTRY` 镜像 tag 为 `v0.x.x`。

**API Token 轮换**：Token 保存在项目根目录 `.env` 的 `ANCHOR_API_TOKEN`。轮换时编辑 `.env` 后执行 `bash install.sh restart`（或对应 compose 重启）。安装向导不会在终端打印完整 Token，请从 `.env` 读取或自行记录。

## 目录结构

```
.
├── main.go                     # Go 服务入口（单二进制，server/worker 双模式）
├── go.mod / go.sum            # Go 模块
├── Makefile                    # 构建脚本 + Docker 生命周期命令
├── docker-compose.yml          # Server + Worker + 网络编排（发布版）
├── docker-compose.e2e-local.yml # 本地 E2E 测试环境
├── Dockerfile.server           # Server 镜像（发布版，从 GitHub Release 下载）
├── Dockerfile.worker           # Worker 镜像（发布版，含安全工具）
├── Dockerfile.server-runtime-base # Server 运行时 base 镜像
├── Dockerfile.worker-runtime-base # Worker 运行时 base 镜像（含安全工具）
├── Dockerfile.server-fast      # Server 快速构建（基于 base）
├── Dockerfile.worker-fast      # Worker 快速构建（基于 base）
├── Dockerfile.compile          # 交叉编译 Linux 二进制
├── plan.md                     # 旧计划跳转页（非当前真相）
├── README.md                   # 本文件
├── docs/                       # 文档中心
│   ├── README.md              # 文档导航入口
│   ├── current/               # 当前唯一有效的计划/架构入口
│   ├── design/                # 候选设计稿
│   ├── archived/              # 历史归档
│   └── 部署指南.md             # 部署指南
├── internal/                   # Go 内部包
│   ├── api/                   # HTTP API handlers（已按 domain 拆分）
│   ├── asset/                 # 资产归一化与去重
│   ├── db/                    # SQLite + queries（按 domain 拆分）+ migrations（v1~v13 独立文件）
│   ├── errors/                # 结构化错误模型
│   ├── health/                # 工具健康检查
│   ├── models/                # 数据模型（按 domain 拆分为 14 个文件）
│   ├── nuclei/                # Nuclei 指纹-Tag 映射
│   ├── parser/                # 工具输出解析器（共享 parseJSONLines 泛型骨架）
│   │   ├── common.go          # 泛型解析骨架 + 共享类型
│   │   ├── subfinder.go       # Subfinder JSONL 解析
│   │   ├── dnsx.go            # dnsx JSONL 解析
│   │   ├── httpx.go           # httpx JSONL 解析
│   │   ├── naabu.go           # Naabu 输出解析
│   │   ├── nmap.go            # Nmap 输出解析
│   │   └── nuclei.go          # Nuclei JSONL 解析
│   ├── search/                # 互联网搜索引擎客户端（共享 baseClient HTTP 基础）
│   │   ├── fofa.go            # FOFA API 客户端
│   │   ├── hunter.go          # Hunter API 客户端
│   │   ├── quake.go           # Quake API 客户端
│   │   └── engine.go          # 统一搜索引擎接口
│   ├── report/                # Markdown / JSON 报告生成
│   ├── scope/                 # Scope Check 引擎
│   ├── scoring/               # Finding confidence/priority 评分
│   ├── toolguard/             # 外部工具执行白名单（binary + arg 安全检查）
│   ├── util/                  # 工具函数（脱敏、ID 生成、shutdown manager 等）
│   ├── worker/                # Worker subprocess runner + 远程客户端 + 资源治理
│   ├── workflows/             # 独立工作流（资产发现、Web 筛选）
│   │   ├── discovery.go       # AssetDiscoveryWorkflow
│   │   ├── screenshot.go      # WebScreeningWorkflow
│   │   └── dedup.go           # URL 去重工具
│   └── scanengine/            # 资产驱动扫描引擎
├── frontend/                   # React 前端
│   ├── src/
│   │   ├── lib/              # API 客户端 + Zustand store
│   │   ├── hooks/            # 共享 Hooks（useResource 通用数据加载）
│   │   ├── components/       # 共享 UI 组件
│   │   ├── pages/            # 页面组件
│   │   └── App.tsx           # 路由与布局
│   └── package.json
├── nginx.conf                  # Nginx 反向代理配置（/api/ → server:17421）
├── docker-rangefield/          # 靶场环境（用于测试）
│   ├── docker-compose.yml
│   └── README.md
└── docs/                       # 项目文档
    ├── current/               # 当前有效文档（agent-guide / architecture / plan）
    ├── archived/              # 历史版本归档
    ├── conventions/           # 编码约定
    ├── pitfalls/              # 踩坑记录
    └── CHANGELOG.md           # 变更日志
```

## 功能清单

### M0: 工程骨架 ✅

- [x] SQLite schema + 迁移（14 张表）
- [x] Scope Check 引擎（域名/URL/IP/CIDR 匹配 + 排除优先 + TOCTOU 防护）
- [x] Worker subprocess runner（goroutine、workdir 隔离、超时、SIGTERM→SIGKILL、100MB 截断）
- [x] HTTP API + SSE 实时推送
- [x] React 前端骨架

### M1: 目标与 Scope 增强 ✅

- [x] 目标批量导入（TXT/CSV），支持逗号展开、IP 连字符范围、自动类型推断
- [x] 首次导入自动提示 Scope 确认
- [x] 时间窗口校验
- [x] 速率限制配置
- [x] 执行计划预览增强

### M2: 资产发现 ✅

- [x] Subfinder 子域名枚举 → 解析 → Asset(domain)
- [x] httpx 存活探测 + 指纹采集 → WebEndpoint + Asset(url)
- [x] Naabu 端口扫描 → Asset(ip) + Port
- [x] 资产归一化（normalized_value 去重）
- [x] CIDR 展开支持（Scope Check）

### M3: Nuclei 初筛 ✅

- [x] **指纹驱动模板筛选** — httpx technologies → 精确 Nuclei `-tags`
- [x] 按 Tag 分组扫描 — 进程数 = 唯一 tag 集合数
- [x] 无指纹目标自动跳过
- [x] Nuclei JSONL 解析 → Finding 去重（dedup_key）
- [x] confidence/priority 规则评分引擎
- [x] Evidence 保存（request/response 脱敏）
- [x] Finding 验证队列 UI

### M4: 报告导出 ✅

- [x] Markdown 报告生成（8 章节模板）
- [x] JSON 数据导出
- [x] 前端报告预览页面
- [x] 端到端验收通过

### v0.2 Phase 1: 容器化与远程 Worker ✅

- [x] Docker 镜像（server / worker）
- [x] Docker Compose 编排
- [x] 远程 Worker 注册/心跳/长轮询
- [x] Worker 超时自动清理
- [x] WorkersPage 实时列表

### v0.2 Phase 2: 项目管理与体验修复 ✅

- [x] 项目创建/删除/级联清理
- [x] 导航修复（Projects 入口、Legacy 路由清理）
- [x] Dashboard 快捷入口
- [x] 首次导入自动提示 Scope 确认

### v0.3: 可用性与可靠性 ✅

- [x] 扫描入口统一（TargetPage Subfinder 按钮 → Runs 导航）
- [x] 路由统一（`/projects/:projectId/*` 嵌套路由）
- [x] 扫描模式选择（外网/内网）+ 速度参数配置面板
- [x] 高危端口预设（115 个攻击面端口，覆盖 Redis/ES/MongoDB/K8s/Ollama 等）
- [x] dnsx DNS 解析替代 Go resolver
- [x] 互联网搜索引擎页面（FOFA/Hunter/Quake 统一搜索 + API Key 配置）
- [x] FOFA 凭证全局化（engine_credentials 表）
- [x] nmap -sV 服务指纹识别 + cdncheck CDN 过滤集成
- [x] 网络服务扫描（非 Web 端口：Redis/MySQL/PostgreSQL/Elasticsearch/MongoDB/Memcached/MSSQL/Oracle）
- [x] CPE 指纹补充（404/302 页面 tech fallback）
- [x] httpx `-follow-redirects`
- [x] 靶场环境修复（Tomcat 弱口令/Manager 访问）
- [x] Go 单元测试覆盖（tagmapper、httpx parser、parser 包、naabu args）
- [x] E2E 测试：ScanConfigPage + high-risk pipeline 端到端验证

### v0.4: 智能扫描能力 ✅

- [x] **多目标类型导入**：domain / ip / cidr / url / **company**（新增）
- [x] **Company 目标自动展开**：FOFA `org/cert/title` 三维搜索 → 展开为 domain/ip 子目标 → 路由到对应 flow
- [x] **资产驱动扫描引擎**：目标 → 资产图 → Work(资产×动作) → 工具执行 → 新资产回注循环；阶段仅用于 UI 聚合展示
- [x] **智能服务指纹**：nmap -sV 识别 Web + 非 Web 服务，不依赖端口号
- [x] **指纹驱动 Nuclei tags**：服务类型精确映射到 Nuclei `-tags`
- [x] **Nuclei 分层扫描**：tags / workflow / both 三选一（含 RBKD-SEC/templates 集成）
- [x] **Nuclei 速率防爆破**：`-rlm`（每分钟限速）+ `-c`（并发）防止账号锁定
- [x] **互联网搜索引擎页面**：FOFA / Hunter / Quake 统一搜索 + 全局 API Key 配置
- [x] **E2E 验收**：5 个 v0.4 目标全部通过测试覆盖（见 `docs/active/review/v0.4-acceptance.md`）

### v0.4.x: 稳定性与治理增量（已合并）

- [x] **Findings 批量写入缓冲**：`FindingBuffer` 容量/超时双触发 flush，消除 N+1 `GetFindingByDedupKey` 查询
- [x] **资源治理**：`ResourceGovernor` 静态内存/CPU 阈值，内存超阈值轮询阻塞、CPU 超阈值 sleep 延迟
- [x] **工具执行白名单**：`toolguard.Allowlist` 二进制 basename 白名单 + shell 元字符拒绝，覆盖全部 5 个 `exec.Command` 调用点

## 外部工具依赖

| 工具                                                       | 用途                 | 最低版本 |
| ---------------------------------------------------------- | -------------------- | -------- |
| [Subfinder](https://github.com/projectdiscovery/subfinder) | 子域名枚举           | v2.6+    |
| [dnsx](https://github.com/projectdiscovery/dnsx)           | DNS 解析             | latest   |
| [httpx](https://github.com/projectdiscovery/httpx)         | Web 存活与指纹       | v1.3+    |
| [Naabu](https://github.com/projectdiscovery/naabu)         | 端口发现             | v2.1+    |
| [nmap](https://nmap.org/)                                | 服务指纹识别 (-sV)   | system   |
| [cdncheck](https://github.com/projectdiscovery/cdncheck)   | CDN/WAF 过滤         | latest   |
| [Nuclei](https://github.com/projectdiscovery/nuclei)       | 漏洞初筛             | v3.0+    |
| [Spoor](https://github.com/P0m32Kun/Spoor)                 | JS 静态分析（路径/端点/密钥） | v0.2.0+ |
| [Nmap](https://nmap.org/)                                  | 深度服务识别（可选） | v7.92+   |

## 版本

| Tag         | 说明                             |
| ----------- | -------------------------------- |
| `v0.1.0-m0` | 工程骨架                         |
| `v0.1.0-m1` | 目标与 Scope 增强                |
| `v0.1.0-m2` | 资产发现                         |
| `v0.1.0-m3` | Nuclei 初筛                      |
| `v0.1.0-m4` | 报告导出                         |
| `v0.2.0-p1` | Docker 容器化 + 远程 Worker 架构 |
| `v0.2.0-p2` | 项目管理与体验修复               |
| `v0.3.0`    | 可用性与可靠性                   |
| `v0.4.0`    | 智能扫描管线 + Nuclei 分层扫描   |

## 许可

MIT License
