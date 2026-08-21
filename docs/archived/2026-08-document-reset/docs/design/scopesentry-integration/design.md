---
status: draft
source_of_truth: false
owner: kun
last_updated: 2026-08-20
scope: distributed-asm-evolution
verification: pending_implementation
---

# ScopeSentry 整合与分布式演进 — 主设计

> Status: Draft
> Audience: 实施者（coding agent / 开发者）
> 默认规则：提案，不是当前架构基线。当前基线仍以 `docs/current/architecture.md` 为准。
> 阅读顺序：本目录 [`README.md`](README.md)（已定决策）→ 本文 → [`roadmap.md`](roadmap.md)。

---

## 1. 背景与动机

### 1.1 定位转变

Anchor 当前是「面向授权渗透测试的目标中心工作台」。本提案将其演进为
「企业级**分布式攻击面测绘平台**」：多目标大规模持续测绘、多 worker 水平扩展、
被动优先降低对目标的主动探测影响、资产变化持续监控。

### 1.2 ScopeSentry 调研结论（2026-08-20，代码级走读）

ScopeSentry（API 后端 + 扫描节点 + Vue UI，Redis 解耦 + MongoDB 存储）调研结论：

**可取（精华）**

| 能力 | 说明 |
| --- | --- |
| 分布式模型 | 多节点经 Redis List 抢占目标、节点心跳/热更新配置、任务级取消 |
| UI 信息架构 | 资产管理/diff/changelog、任务进度、节点管理、测绘引擎查询页 |
| 扫描广度 | 子域接管、敏感信息、URL 安全、被动扫描、目录扫描成洞等独立模块 |
| 规则库积累 | Web 指纹（AC 自动机三路预筛）、敏感信息正则、接管 CNAME 指纹 |
| 低影响策略意识 | CDN/WAF 检测跳过端口扫描、去重三级缓存 |

**不可取（糟粕）**

| 问题 | 位置 |
| --- | --- |
| 14 级固定流水线拓扑刚性，模块强耦合、无法按阶段扩缩容 | `modules/manage.go` |
| 插件系统（yaegi 解释执行慢一个量级、Clone 装配开销） | `internal/plugins` |
| 库内嵌导致版本绑架：nuclei fork 落后 8 个 minor（v3.3.6 vs v3.11.1） | `libs/nuclei` 等 4 个 fork |
| 性能热点：指纹匹配每候选新建 http.Client + 重编译正则；wayback 串行逐条验证；nuclei 每批全量重载模板 | `webfingerprint/core.go` 等 |
| 默认全量 POC 裸跑（指纹联动可选） | `nuclei.go` |
| 扫描端无测绘引擎 API 集成（被动收集弱） | — |
| 工程问题：固定 sleep 堆叠、WaitGroup 计数 bug、索引冗余/缺失 | 详见调研记录 |

**许可证**：AGPL-3.0 + 商用附加条款（组织使用需商业授权）。
→ 本项目为 MIT。**只参考其公开设计思路与数据格式，不复制任何代码**（见 §2 D5）。

### 1.3 Anchor 现状评估

**强项（直接作为基座复用，约 60% 工作量）**

| 组件 | 价值 |
| --- | --- |
| `internal/scanengine/core/` | 资产驱动派生模型（资产类型 × ActionRule(MaxDepth+Precondition) → Work），规避固定流水线的拓扑刚性 |
| `internal/scanengine/queue/ + scheduler/` | 三层优先队列 + Fair/Staged 调度 + ComputeLimits 动态并发 + IPThrottler（低影响策略引擎雏形） |
| `internal/scanengine/pool/` 分层批量 | DNS 100/批、nmap 1IP×全端口、nuclei 按指纹 tag 分桶；实测 tool call ~5791 → ~150-300（20 倍） |
| `internal/parser/` | 泛型 JSONL 解析骨架 + 9 工具解析器，零依赖 |
| `internal/toolregistry/ + tools/*.yaml` | 声明式工具定义：新工具 = 一个 YAML + 一个解析器（即「固定工具链」的低成本扩展形态） |
| `internal/search/` | FOFA/Hunter/Quake 归一化 client + company 三维展开 + 边界/垃圾过滤 |
| `internal/scope/ + exclude/ + toolguard/` | TOCTOU 复检、全局排除、执行白名单 |
| 指纹驱动 nuclei 路由 | tech→tags（YAML 可配 `_default: skip`），无指纹自动跳过 |
| 测试资产 | Go 测试占 37%、37 个 Playwright E2E、内嵌靶场 |

**弱项（需改造，约 20% 工作量）**

| 问题 | 方案 |
| --- | --- |
| SQLite WAL 单写者 = 分布式天花板 | → PostgreSQL（§3.1） |
| 通信双通道冗余（推 + 长轮询） | → 收敛单通道（§3.2） |
| 崩溃恢复 = 全部标 failed（认输式） | → pending work 重入队断点续扫（§3.3） |
| 通知仅 Webhook | → 补钉钉/飞书/企微/Telegram（§5.5） |
| 无用户体系（单 token） | → 开放问题 O3 |

### 1.4 复用比例结论

**~60% Anchor 已验收代码 + ~20% 外壳改造 + ~20% 借鉴 ScopeSentry 的重新实现。**
不从零重写，不直接二开 ScopeSentry（许可证 + 架构包袱双重否决）。

---

## 2. 已定决策（owner 2026-08-20 确认）

| # | 决策 | 理由 |
| --- | --- | --- |
| D1 | **不做插件系统**。固定工具链；扩展 = toolregistry YAML + 新解析器 + 重编译 | 降低复杂度；Anchor 模式已被验证，且 toolguard 白名单天然覆盖 |
| D2 | **不接 LLM**。POC 由 owner 独立维护库供给，经 `internal/builtin` 同步机制对接，保持 nuclei 兼容格式 | 已有成熟的 POC 运营流程 |
| D3 | **纯代码 + 规则 + API + 工具**。不引入解释器运行时（yaegi/wasm/gRPC sidecar 均不考虑） | 与 D1 一致的复杂度约束 |
| D4 | **UI 参考 ScopeSentry** 信息架构与视觉设计；前端保留 React18+TS+Tailwind+Zustand 组件体系 | UI 是 ScopeSentry 公认优点；重写视图层而非组件底座 |
| D5 | **合规红线**：ScopeSentry 为 AGPL-3.0 + 商用附加条款。只参考公开文档描述的思路、行为与数据格式；**不复制其源码、不 vendored 其模块**。规则/字典等数据逐源确认许可后再引入 | 版权合规 |
| D6 | **进程调用 + 批量池化**的 tool 执行模式保留，不改为库内嵌 | 池化已摊薄进程开销；避免 ScopeSentry 式 fork 版本绑架 |

---

## 3. 目标架构

### 3.1 总览

```
                    ┌────────────────────────────────────────┐
                    │  Frontend (React, 参考 ScopeSentry 设计) │
                    └───────────────┬────────────────────────┘
                                    │ HTTP/SSE
                    ┌───────────────▼────────────────────────┐
                    │  Server（单实例起步，可多实例）            │
                    │  · API + SSE 推送                       │
                    │  · scanengine：资产图/派生规则/公平调度    │
                    │  · dispatcher：最少负载定向派发（保留）     │
                    │  · 被动收集编排（FOFA/Hunter/Quake/...）   │
                    └───────┬──────────────────────┬─────────┘
                            │ PostgreSQL           │ 单通道任务协议（HTTP）
                    ┌───────▼───────┐     ┌────────▼─────────┐
                    │ PostgreSQL    │     │ Worker × N（无状态）│
                    │ 资产图/Work/   │     │ ToolExecutor      │
                    │ findings/审计  │     │ parser + toolguard│
                    └───────────────┘     │ ResourceGovernor  │
                                          └──────────────────┘
```

部署形态保持四容器：frontend / server / worker / postgres。
**不引入 Redis、消息队列**（见 3.4）。

### 3.2 存储层：SQLite → PostgreSQL

- **理由**：SQLite 单写者是多 worker 高并发的根本瓶颈；Postgres 的
  `SELECT ... FOR UPDATE SKIP LOCKED` 原生支持 work 队列原子抢占（Anchor 的
  work TryClaim 模式直接平移）。
- **迁移策略**：不逐条平移 44 个历史迁移。以 v44 schema 为基线**合并重写为
  Postgres v1 迁移**；`docs/schema-migrations.md` 增补说明。SQLite 历史迁移
  目录归档保留（演进史即需求记录）。
- **表设计平移清单**（结构不变，仅方言适配）：projects / targets / scope_rules /
  scope_decisions / scan_tasks / tool_invocations / raw_artifacts / audit_logs /
  tool_health / assets(UNIQUE project+normalized_value) / ports / services /
  web_endpoints / findings(UNIQUE project+dedup_key) / evidence / worker_nodes /
  runs / screenshots / scan_work_items / pipeline_run_stages / asset_relations /
  asset_state / asset_changes / asset_snapshots / watch/signal/notification /
  retest_runs。
- **驱动**：`pgx`（纯 Go，弃 CGO 依赖 go-sqlite3）。
- **嵌入式单机模式（SQLite）**：不保留（开放问题 O2，默认否）。

### 3.3 通信与调度

- **收敛单通道**：移除「Dispatcher 主动推 + RemoteClient 长轮询」双通道冗余，
  保留**拉模式长轮询**为主通道（兼容 worker 位于 NAT 后的部署；30s 心跳不变）。
  推模式代码删除，`Runner.Run` 本地回退执行保留（单机部署形态：server 进程内
  直接执行 work，跳过网络）。
- **调度算法保留**：最少负载优先（DB running + 内存 inflight 双计数）→ 同负载
  round-robin → 容量 503 改派 → offline 故障转移（现 `dispatcher.go` 逻辑平移）。
- **断点续扫**：替换 orphan 恢复的「全部标 failed」。Server 重启时：
  running work → 归还 pending（重入队）；pending work → 保持；run 状态不变。
  幂等性由工具侧原子输出 + assets/findings 唯一键保证。
- **任务取消**：长轮询响应携带 cancel 信号位；worker 侧 SIGTERM 链路已有。

### 3.4 Redis / 消息队列：不引入

测绘规模（万级目标、百级并发 tool call）下 Postgres SKIP LOCKED 队列吞吐充足；
少一个状态组件 = 部署与运维复杂度显著下降。若未来单 server 成为瓶颈，
优先拆分「被动收集 / 编排 / API」为多 server 实例（均无状态，共享 Postgres），
仍无需消息队列。

### 3.5 低影响策略引擎（渐进增强）

在现有 IPThrottler + ComputeLimits 基础上补齐：

| 策略 | 现状 | 增量 |
| --- | --- | --- |
| per-IP 限流 | 已有 | — |
| CDN/WAF 排除 | cdncheck 已有（IP 级） | 子域级 CNAME 链路排除 |
| 蜜罐排除 | 无 | 测绘引擎 API 的蜜罐标记回写 asset attrs，命中跳过主动扫描 |
| 被动优先级 | 部分 | 派生规则 Precondition 增加「被动数据充分则降级验证」（如引擎已返回 port/service 则跳过 naabu） |
| 时间窗/速率档位 | Scope 配置已有时间窗口 | 速率档位（gentle/normal/aggressive）映射各工具参数 |
| 端口策略 | 高危端口预设已有 | 资产分级：默认 top 端口，高价值/未知资产升全端口 |

---

## 4. 工具链基线（D1/D6 落地）

固定 11 工具，全部进程调用 + Dockerfile.worker 版本锁定 + Renovate 升级 PR + E2E 验证：

| 工具 | 用途 | 状态 |
| --- | --- | --- |
| subfinder | 子域名枚举 | 沿用 |
| dnsx | DNS 解析 | 沿用 |
| cdncheck | CDN/WAF 判断 | 沿用 |
| naabu | 端口扫描 | 沿用（ScopeSentry 用 rustscan，已被本路线否决） |
| nmap -sV | 服务指纹 | 沿用 |
| httpx | Web 存活 + 指纹 | 沿用（`-cff` 自定义指纹机制已有） |
| nuclei | 漏洞初筛 | 沿用（POC 源 = owner 库，见 §5.0） |
| katana | 爬虫 | 沿用（否决 rad/gospider 路线） |
| ffuf | 目录/参数爆破 | 沿用 |
| gau | 历史 URL | 沿用 |
| Spoor | JS 静态分析 | 沿用 |

**明确不引入**：rustscan（naabu 替代）、rad（katana headless 替代）、gospider
（katana 替代）、uro/Python 运行时（URL 去重 Go 内实现，§5.6）、ksubdomain
（dnsx + 池化满足；pcap 依赖增加部署复杂度）。

### 4.1 POC 库对接（D2）

- `internal/builtin` 的模板 bundle 同步源指向 **owner 独立维护的 POC 仓库**
  （nuclei 兼容 YAML + tech→tags 元数据）。
- 指纹路由（`scanconfig/nuclei_routing.go`）继续作为唯一筛选入口；
  `_default: skip` 语义保持 —— **无指纹不跑 POC** 是不可妥协的默认。
- owner 库新增 POC 后由同步机制（60s 轮询已有）热更新，无需重发版。

---

## 5. 借鉴 ScopeSentry 的能力（重新实现，遵守 D5）

> 每项标注：借鉴点（公开行为）→ Anchor 落点 → 顺带修正的已知低效模式。

### 5.0 测绘引擎补齐（P4，先做——被动数据的入口）

- 借鉴：ScopeSentry 支持 FOFA/Hunter/Quake/ZoomEye/Shodan/Censys/0.zone 等多引擎。
- 落点：`internal/search/` 增加 shodan.go / censys.go / zoomeye.go，
  复用 baseClient 与 SearchResult 归一化模型；引擎能力矩阵（哪些引擎支持
  org 展开/蜜罐标记/历史数据）写入 seed 编排的引擎选择逻辑。
- 修正：引擎结果直接回写 asset attrs（含蜜罐标记），供 §3.5 被动优先级使用。

### 5.1 敏感信息检测（P1）

- 借鉴：对 HTTP 响应体/URL 做规则化敏感信息匹配（分块 + overlap + 规则引擎 +
  AHOCorasick 关键词预筛 + 可选 secret detector 验证）。
- 落点：新 TaskAction `SCAN_SENSITIVE`，作用于 HTTP_SERVICE / HTTP_PATH 资产；
  katana/gau 产出的响应与 JS（Spoor 密钥检测已有，规则层面合并）统一送入。
  规则存 DB（YAML 导入），**正则预编译缓存 + 指纹级关键词预筛**。
- 修正：ScopeSentry 该模块逐块 × 逐规则串行 + 无规则并行 —— 我们实现时
  worker pool 并行规则组；命中写入 findings（category=sensitive）。

### 5.2 子域接管检测（P2）

- 借鉴：CNAME 指向已知可接管服务（CNAME 指纹库）→ 主动验证 → 告警。
- 落点：dnsx 解析结果已含 CNAME；新 TaskAction `CHECK_TAKEOVER` 作用于
  已解析 SUBDOMAIN 资产；CNAME 指纹库存 DB（YAML 导入，使用公开可核实的
  接管指纹数据源，逐源确认许可）；验证请求复用统一 http client。
- 修正：ScopeSentry 用朴素三层 Contains 匹配 —— 我们用预编译指纹表 +
  AC 预筛（与 5.1 共用基础设施）。

### 5.3 目录扫描成洞（P3）

- 借鉴：目录扫描结果独立成「发现」而非仅资产。
- 落点：ffuf 输出已回注资产；增加 findings(category=dir) 生成规则
  （状态码/长度/字典项敏感度阈值可配），命中进人工验证队列。

### 5.4 通知渠道（P5）

- 落点：`internal/notify` 增加 dingtalk / feishu / wecom / telegram；
  沿用现有事件模型（signal.new / scan.completed），按 finding category 路由。

### 5.5 URL 去重 uro 语义（P6）

- 借鉴：URL 归一化去重（参数名集合哈希、低价值路径剔除）以削减下游扫描量。
- 落点：`internal/workflows/` URL 去重流程升级为 uro 语义的 Go 实现
  （无 Python 依赖）；去重发生在 URL 进入派生队列之前。

### 5.6 Web 指纹库增强（P7，可选）

- 借鉴：ScopeSentry 指纹规则的三路预筛（title/header/body AC 自动机）。
- 落点：httpx `-cff` 已支持自定义指纹；扩充指纹库数据（仅采用许可明确的
  公开指纹数据源），保持与 tech→tags 路由兼容。

---

## 6. 数据模型演进

- **findings 统一类别化**：新增 `category` 枚举（nuclei / sensitive / takeover /
  dir / ...），敏感信息与接管**不建新表**，复用 findings + evidence + 评分/
  验证队列链路（与 ScopeSentry「每类结果一张表」对比，这是刻意的简化）。
- **asset attrs 扩展**：honeypot(引擎标记) / passive_source(数据来自哪个引擎) /
  scan_tier(端口策略档位)，支撑 §3.5 策略引擎。
- **断点续扫**：`scan_work_items` schema 不变，语义扩展见 §3.3。
- **资产 diff/snapshot**：已有 asset_changes / snapshots 基础，UI 重做后完整暴露。

---

## 7. UI 重做方案（D4）

前端底子保留（React18 + TS + Tailwind + Zustand + 自建组件 + 虚拟滚动），
视图层按 ScopeSentry 信息架构重做：

| ScopeSentry 页面 | Anchor 现状 | 动作 |
| --- | --- | --- |
| 资产管理（HTTP/Other/子域/URL/漏洞 统一视图 + 筛选 DSL） | AssetsPage 基础版 | 重做：统一资产浏览 + 高级筛选 |
| 资产 diff / 变更历史 | AssetChanges/Snapshots 后端已有 | 重做前端暴露 |
| 任务列表 + 实时进度 | RunsPage（Work 明细观察台） | 融合：保留 Work 明细，外层加 ScopeSentry 式进度汇总 |
| 节点管理 | WorkersPage（实时负载） | 视觉对齐 |
| 测绘引擎查询 | EnginesPage（3 引擎） | 扩展至 6 引擎 + 蜜罐标记展示 |
| 敏感信息/接管/目录发现页面 | 无 | 新增（复用 Findings 队列组件） |
| 通知渠道配置 | NotificationChannelsPage 基础版 | 扩展渠道 |

视觉规范（色板/间距/组件密度）实施时从 ScopeSentry-UI 公开演示页提取风格基线，
**不复制其前端代码**（D5）。

---

## 8. 风险与开放问题

| # | 类型 | 内容 | 处置 |
| --- | --- | --- | --- |
| R1 | 合规 | ScopeSentry AGPL-3.0 + 商用条款；指纹/敏感信息/接管指纹等**数据**的来源许可需逐源确认 | D5 原则执行；数据源清单在 M3 实施前评审 |
| R2 | 工程 | Postgres 迁移体量大（112 queries 文件） | 以 v44 为基线合并重写；pgx + 既有 queries 测试平移 |
| R3 | 工程 | 通信收敛到纯拉模式可能增大任务派发延迟 | 长轮询间隔可调（默认 5s）；延迟敏感场景开放 O1 |
| R4 | 运维 | worker 大输出内存治理在 Postgres 下的行为 | ResourceGovernor 已有阈值机制，M1 E2E 验证 |
| O1 | 开放 | 拉模式 vs gRPC 双向流 | 默认拉模式；瓶颈实测后再议 |
| O2 | 开放 | 是否保留 SQLite 嵌入式单机模式 | 默认不保留；若有离线交付需求再评估（driver 抽象预留） |
| O3 | 开放 | 多用户/权限体系 | v1 保持单 token；企业化需求明确后再设计 |

---

## 9. 与当前架构基线的关系

本提案**不修改** `docs/current/architecture.md`。各里程碑完成验收后，
按维护规则将对应章节提升进 current（architecture.md 增量更新 + CHANGELOG），
本目录随后标记 superseded 并归档。
