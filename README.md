# Anchor — 目标中心自动化安全测试工作台

> 面向授权安全测试的目标中心工作台，通过编排成熟开源工具、强制范围校验、统一结果模型、人工验证队列和报告生成，减少安全人员在工具切换、数据整理、证据归档和报告交付上的重复劳动。

## 完整扫描流程

```
目标输入 → Scope Check → 资产发现 → Web 初筛 → 人工验证 → 报告导出
```

| 阶段 | 工具 | 输出 |
|------|------|------|
| **Scope Check** | 自研引擎 | ScopeDecision (allow/deny) |
| **资产发现** | Subfinder → httpx → Naabu | Asset / WebEndpoint / Port |
| **Web 初筛** | Nuclei（指纹驱动模板筛选） | Finding / Evidence |
| **人工验证** | 前端队列 | confirmed / false_positive / accepted_risk |
| **报告导出** | 自研生成器 | Markdown / JSON |

**核心设计**：指纹驱动 Nuclei 模板筛选 — httpx 识别的技术栈（WordPress/nginx/Apache Druid 等）精确映射到 Nuclei `-tags`，无指纹目标自动跳过，避免全量扫描。

## 技术栈

| 层级 | 技术 |
|------|------|
| 桌面客户端 | Tauri 2.x + React 18 + TypeScript + Tailwind CSS |
| 状态管理 | Zustand |
| 本地/远程服务 | Go 1.26 |
| 数据库 | SQLite (WAL 模式) |
| 实时推送 | SSE (Server-Sent Events) |
| 语法高亮 | Prism.js |

## 快速开始

### 依赖

- Go 1.26+
- Node.js 18+
- 外部安全工具
  ```bash
  go install -v github.com/projectdiscovery/subfinder/v2/cmd/subfinder@latest
  go install -v github.com/projectdiscovery/httpx/cmd/httpx@latest
  go install -v github.com/projectdiscovery/naabu/v2/cmd/naabu@latest
  go install -v github.com/projectdiscovery/nuclei/v3/cmd/nuclei@latest
  ```

### Docker 一键部署（推荐）

```bash
# 启动 Server + Worker + 靶场（可选）
make up-all

# 停止所有服务
make down-all

# 查看状态
docker compose ps
```

Server 监听 `:17421`，Worker 自动注册并拉取任务。

### 本地开发

**运行后端：**
```bash
go run main.go
# 服务监听 :17421，数据目录 ~/.anchor
```

**运行前端：**
```bash
cd frontend
npm install
npm run dev
# 打开 http://localhost:1420
```

**构建：**
```bash
make build    # 构建 Go 后端
make test     # 运行测试
tauri build   # 构建 Tauri 桌面应用
```

## Worker 架构

Worker 完全容器化，通过 Docker Compose 编排：

```
[客户端] --HTTP--> [Server :17421] <--HTTP 长轮询-- [Worker 容器]
```

- **Server**：纯管理面，负责 API、任务调度、数据持久化
- **Worker**：主动注册到 Server，通过长轮询拉取任务，执行安全工具（subfinder/httpx/naabu/nuclei）

Worker **不需要公网 IP**，只要 outbound 能访问 Server 即可。

**典型场景：**
- 公网 VPS 部署 Server，家庭 WiFi 笔记本运行 Worker
- 多云 Worker 统一接入中心 Server
- 单机开发：`make up-all` 一键启动完整环境

## 目录结构

```
.
├── main.go                     # Go 服务入口（单二进制，server/worker 双模式）
├── go.mod / go.sum            # Go 模块
├── Makefile                    # 构建脚本 + Docker 生命周期命令
├── docker-compose.yml          # Server + Worker + 网络编排
├── Dockerfile.server           # Server 镜像（纯管理面）
├── Dockerfile.worker           # Worker 镜像（含安全工具）
├── 设计.md                      # PRD（产品需求文档）
├── plan.md                     # 开发执行计划与进度
├── README.md                   # 本文件
├── docs/                       # 技术文档
│   ├── API.md                 # API 参考
│   ├── ARCHITECTURE.md        # 架构说明
│   ├── ADR-v0.2.md            # v0.2 架构决策
│   └── 部署指南.md             # 部署指南
├── internal/                   # Go 内部包
│   ├── api/                   # HTTP API handlers
│   ├── asset/                 # 资产归一化与去重
│   ├── db/                    # SQLite schema + queries
│   ├── errors/                # 结构化错误模型
│   ├── health/                # 工具健康检查
│   ├── models/                # 数据模型
│   ├── nuclei/                # Nuclei 指纹-Tag 映射
│   ├── parser/                # 工具输出解析器
│   │   ├── subfinder.go       # Subfinder JSONL 解析
│   │   ├── httpx.go           # httpx JSONL 解析
│   │   ├── naabu.go           # Naabu 输出解析
│   │   └── nuclei.go          # Nuclei JSONL 解析
│   ├── report/                # Markdown / JSON 报告生成
│   ├── scope/                 # Scope Check 引擎
│   ├── scoring/               # Finding confidence/priority 评分
│   ├── util/                  # 工具函数（脱敏、ID 生成等）
│   ├── worker/                # Worker subprocess runner + 远程客户端
│   └── workflow/              # 工作流编排
│       ├── discovery.go       # 资产发现工作流
│       └── screenshot.go      # Web 初筛工作流
├── frontend/                   # Tauri + React 前端
│   ├── src/
│   │   ├── lib/              # API 客户端 + Zustand store
│   │   ├── components/       # 共享 UI 组件
│   │   ├── pages/            # 页面组件
│   │   └── App.tsx           # 路由与布局
│   └── package.json
├── src-tauri/                  # Tauri 配置
├── docker-rangefield/          # 靶场环境（用于测试）
│   ├── docker-compose.yml
│   └── README.md
└── wiki/                       # 项目知识库
    ├── SCHEMA.md              # AI 指令文件（必读）
    ├── decisions/             # 架构决策记录 (ADR)
    ├── conventions/           # 编码约定
    ├── pitfalls/              # 踩坑记录
    └── log.md                 # 变更日志
```

## 功能清单

### M0: 工程骨架 ✅

- [x] SQLite schema + 迁移（14 张表）
- [x] Scope Check 引擎（域名/URL/IP/CIDR 匹配 + 排除优先 + TOCTOU 防护）
- [x] Worker subprocess runner（goroutine、workdir 隔离、超时、SIGTERM→SIGKILL、100MB 截断）
- [x] HTTP API + SSE 实时推送
- [x] Tauri 前端骨架

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

### v0.3: 桌面可用性与可靠性 ✅

- [x] 扫描入口统一（TargetPage Subfinder 按钮 → Runs 导航）
- [x] 路由统一（`/projects/:projectId/*` 嵌套路由）
- [x] 网络服务扫描（非 Web 端口：Redis/MySQL/PostgreSQL/Elasticsearch/MongoDB/Memcached/MSSQL/Oracle）
- [x] CPE 指纹补充（404/302 页面 tech fallback）
- [x] httpx `-follow-redirects`
- [x] 靶场环境修复（Tomcat 弱口令/Manager 访问）
- [x] Go 单元测试覆盖（tagmapper、httpx parser、parser 包）

## 外部工具依赖

| 工具 | 用途 | 最低版本 |
|------|------|----------|
| [Subfinder](https://github.com/projectdiscovery/subfinder) | 子域名枚举 | v2.6+ |
| [httpx](https://github.com/projectdiscovery/httpx) | Web 存活与指纹 | v1.3+ |
| [Naabu](https://github.com/projectdiscovery/naabu) | 端口发现 | v2.1+ |
| [Nuclei](https://github.com/projectdiscovery/nuclei) | 漏洞初筛 | v3.0+ |
| [Nmap](https://nmap.org/) | 深度服务识别（可选） | v7.92+ |

## 版本

| Tag | 说明 |
|-----|------|
| `v0.1.0-m0` | 工程骨架 |
| `v0.1.0-m1` | 目标与 Scope 增强 |
| `v0.1.0-m2` | 资产发现 |
| `v0.1.0-m3` | Nuclei 初筛 |
| `v0.1.0-m4` | 报告导出 |
| `v0.2.0-p1` | Docker 容器化 + 远程 Worker 架构 |
| `v0.2.0-p2` | 项目管理与体验修复 |
| `v0.3.0` | 桌面可用性与可靠性 |

## 许可

MIT License
