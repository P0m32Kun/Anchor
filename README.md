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
- **客户部署**：[`docs/current/deployment.md`](docs/current/deployment.md)
- **开发者 E2E**：[`docs/current/e2e-testing.md`](docs/current/e2e-testing.md)
- 候选设计索引：[`docs/current/design/README.md`](docs/current/design/README.md)
- 历史归档：[`docs/archived/`](docs/archived/)

## 快速开始

### 客户部署（生产环境）

```bash
bash install.sh   # 交互式向导：拉 ACR 镜像，无本地 build
```

详见 [`docs/current/deployment.md`](docs/current/deployment.md)。

### 本地开发

> 日常开发直接运行 Go 二进制，不经过 Docker。

```bash
go run .                    # Server :17421
cd frontend && npm run dev  # 前端 :5173
make build && make test     # 构建与单元测试
```

### 开发者 E2E 测试

```bash
make build-base && make build-fast   # 首次或工具版本变更后
make test-e2e                        # Playwright 快速套件
make test-e2e-scan                   # 长 pipeline（最长 30 分钟/spec）
```

详见 [`docs/current/e2e-testing.md`](docs/current/e2e-testing.md)。**勿**对客户环境使用 `docker-compose.e2e*.yml`。

## 目录结构

```
.
├── main.go                     # Go 服务入口（单二进制，server/worker 双模式）
├── go.mod / go.sum            # Go 模块
├── Makefile                    # 构建脚本 + Docker 生命周期命令
├── install.sh                  # 客户部署安装向导
├── docker-compose.yml               # 【部署】Server+Worker+Frontend（仅 ACR image）
├── docker-compose.server.yml        # 【部署】Server + Frontend
├── docker-compose.worker.yml        # 【部署】Worker only
├── docker-compose.e2e.yml           # 【E2E】Playwright + fast build + 内嵌靶场
├── docker-compose.e2e-local.yml     # 【E2E】手动迭代 + ACR frontend
├── docker-compose.release-verify.yml # 【验证】上线前镜像验收
├── Dockerfile.server           # 【发布】生产 Server（CI / release-verify）
├── Dockerfile.worker           # 【发布】生产 Worker
├── Dockerfile.frontend         # 【发布】生产 Frontend
├── Dockerfile.*-runtime-base   # 【E2E】预装运行时依赖（make build-base）
├── Dockerfile.*-fast           # 【E2E】快速 COPY 本地二进制
├── Dockerfile.compile          # 交叉编译 Linux 二进制
├── plan.md                     # 旧计划跳转页（非当前真相）
├── README.md                   # 本文件
├── docs/                       # 文档中心
│   ├── README.md              # 文档导航入口 + 维护规则
│   ├── current/               # 当前有效文档
│   │   ├── agent-guide.md     # Coding agent 迭代协议
│   │   ├── architecture.md    # 当前唯一架构基线
│   │   ├── plan.md            # 当前唯一仓库级计划
│   │   ├── deployment.md      # 客户部署指南
│   │   ├── e2e-testing.md     # 开发者 E2E 指南
│   │   ├── ci-cd-guide.md     # CI/CD 流程指南
│   │   ├── scan-api-guide.md  # 扫描 API 指南
│   │   ├── code-health-audit.md # 代码健康审计
│   │   ├── short-term-plan.md # 短期计划
│   │   ├── asset-driven-remediation-design.md
│   │   ├── design/            # 候选设计索引
│   │   ├── decisions/         # 决策索引
│   │   └── audit-reports/     # 审计报告
│   ├── active/                # 活跃评审材料
│   ├── design/                # 候选设计稿
│   ├── features/              # 功能专项文档
│   ├── plans/                 # 计划材料
│   ├── superpowers/           # 设计规格
│   ├── templates/             # 漏洞模板
│   ├── conventions/           # 编码规范
│   ├── pitfalls/              # 踩坑记录
│   ├── archived/              # 历史归档
│   ├── CHANGELOG.md           # 变更日志
│   ├── schema-migrations.md   # Schema 迁移策略
│   ├── api-error-contract.md  # API 错误约定
│   ├── refactoring-plan.md    # 重构想法（backlog）
│   └── functional-test.md     # 功能测试清单
├── internal/                   # Go 内部包（38 个子包）
│   ├── api/                   # HTTP API handlers（按 domain 拆分，48 个文件）
│   ├── asset/                 # 资产归一化与去重
│   ├── builtin/               # RBKD-SEC 内置资源同步（dict/templates/finger）
│   ├── cdn/                   # CDN 响应解析
│   ├── credentials/           # 凭证发现与策略引擎
│   ├── db/                    # SQLite + queries（按 domain 拆分）+ migrations（v1~v39）
│   ├── dictionary/            # 字典管理（ffuf 字典）
│   ├── errors/                # 结构化错误模型
│   ├── evaluator/             # 扫描评估引擎（规则 + 趋势分析 + 报告）
│   ├── exclude/               # 全局域名排除列表
│   ├── finding/               # Finding 持久化（NucleiPersister、Buffer）
│   ├── fingerprint/           # 指纹管理
│   ├── health/                # 工具健康检查
│   ├── httpxfp/               # httpx 指纹管理
│   ├── models/                # 数据模型（按 domain 拆分为 28 个文件）
│   ├── nuclei/                # Nuclei 自定义模板管理 + 指纹-Tag 映射
│   ├── parser/                # 工具输出解析器（共享 parseJSONLines 泛型骨架）
│   │   ├── common.go          # 泛型解析骨架 + 共享类型
│   │   ├── subfinder.go       # Subfinder JSONL 解析
│   │   ├── dnsx.go            # dnsx JSONL 解析
│   │   ├── httpx.go           # httpx JSONL 解析
│   │   ├── naabu.go           # Naabu 输出解析
│   │   ├── nmap.go            # Nmap 输出解析
│   │   ├── nuclei.go          # Nuclei JSONL 解析
│   │   ├── ffuf.go            # ffuf 输出解析
│   │   ├── gau.go             # gau JSONL 解析
│   │   └── katana.go          # Katana JSONL 解析
│   ├── passive/               # 被动搜索编排（FOFA/Hunter/Quake 并行调用）
│   ├── report/                # Markdown / JSON 报告生成
│   ├── resolve/               # DNS 解析
│   ├── safefs/                # 安全文件系统操作
│   ├── scanconfig/            # 扫描配置（PipelineConfig、nuclei tech 路由）
│   ├── scanengine/            # 资产驱动扫描引擎
│   │   ├── core/              # DiscoveryAsset、TaskAction、DeriveEligibleWorks
│   │   ├── work/              # Store (TryClaim/MarkDone/AllTerminal)
│   │   ├── queue/             # PriorityQueue + Fair/Staged 调度
│   │   ├── dedup/             # Run 级 normalized value 去重
│   │   ├── executor/          # ToolExecutor + 各工具 parser（httpx/katana/ffuf/spoor）
│   │   ├── stageagg/          # Stage 聚合（仅 UI 投影，不影响执行）
│   │   ├── pool/              # 通用池、httpx 候选、IP port 聚合、nuclei 分桶
│   │   ├── scheduler/         # ComputeLimits、IPThrottler、SeedBucketKey
│   │   ├── seed/              # 种子资产注入（被动搜索、边界过滤、垃圾过滤）
│   │   ├── domainpool/        # 域名批处理池
│   │   ├── recovery/          # orphan run 恢复
│   │   ├── engine.go          # ScanEngine 主循环
│   │   ├── engine_tier1.go    # Tier1 池化接线（DNS/CDN/Port/Subfinder）
│   │   └── engine_tier2.go    # Tier2 池化接线（httpx/nmap/nuclei）
│   ├── scope/                 # Scope Check 引擎
│   ├── scoring/               # Finding confidence/priority 评分
│   ├── search/                # 互联网搜索引擎客户端（共享 baseClient HTTP 基础）
│   │   ├── fofa.go            # FOFA API 客户端
│   │   ├── hunter.go          # Hunter API 客户端
│   │   ├── quake.go           # Quake API 客户端
│   │   └── engine.go          # 统一搜索引擎接口
│   ├── service/               # 服务层
│   ├── signal/                # 信号处理
│   ├── sources/               # 数据源管理（nuclei 源 bundle 同步）
│   ├── submission/            # 提交包管理
│   ├── toolguard/             # 外部工具执行白名单（binary + arg 安全检查）
│   ├── toolregistry/          # 工具注册表
│   ├── toolrun/               # 工具运行管理
│   ├── util/                  # 工具函数（脱敏、ID 生成、shutdown manager 等）
│   ├── watch/                 # 监控
│   ├── worker/                # Worker subprocess runner + 远程客户端 + 资源治理
│   └── workflows/             # 独立工作流（资产发现、Web 筛选、URL 去重）
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
└── docs/                       # 文档中心
    ├── README.md              # 文档导航入口 + 维护规则
    ├── current/               # 当前有效文档（agent-guide / architecture / plan / deployment / e2e-testing 等）
    ├── active/                # 活跃评审材料
    ├── design/                # 候选设计稿
    ├── features/              # 功能专项文档
    ├── templates/             # 漏洞模板
    ├── archived/              # 历史版本归档
    ├── conventions/           # 编码规范
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
| [dnsx](https://github.com/projectdiscovery/dnsx)           | DNS 解析             | v1.2+    |
| [httpx](https://github.com/projectdiscovery/httpx)         | Web 存活与指纹       | v1.3+    |
| [Naabu](https://github.com/projectdiscovery/naabu)         | 端口发现             | v2.1+    |
| [nmap](https://nmap.org/)                                  | 服务指纹识别 (-sV)   | v7.92+   |
| [cdncheck](https://github.com/projectdiscovery/cdncheck)   | CDN/WAF 过滤         | v1.2+    |
| [Nuclei](https://github.com/projectdiscovery/nuclei)       | 漏洞初筛             | v3.0+    |
| [Katana](https://github.com/projectdiscovery/katana)       | Web 爬虫（JS 端点发现） | v1.6+  |
| [ffuf](https://github.com/ffuf/ffuf)                       | 目录/参数爆破        | v2.1+    |
| [gau](https://github.com/lc/gau)                           | 历史 URL 枚举        | v2.2+    |
| [Spoor](https://github.com/P0m32Kun/Spoor)                 | JS 静态分析（路径/端点/密钥） | v0.2.0+ |

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
