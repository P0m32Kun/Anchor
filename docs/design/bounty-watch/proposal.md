---
status: proposed
source_of_truth: false
owner: kun
created: 2026-06-13
scope: bounty-watch
related_plan: docs/current/plan.md
---

# Bounty Watch — Proposal

## 动机

用户从赏金平台领取目标（IP / IP 段 / 域名 / 企业名），交给 Anchor 做**持续攻击面监控**：发现关联、隐蔽、边缘资产，并汇聚指纹、敏感信息、漏洞等高价值信号；人工或外部 Agent 只处理 inbox 顶部内容。

Anchor 已完成 Scan Engine 收敛（资产驱动、lineage、外网 Profile），但产品形态仍是**一次性护网扫描**，与上述 workflow 存在四类结构性差距：

1. **Scope 边界**：赏金需要 `*.target.com` + OOS 限定扫描范围；护网通常不需要。现有 scope 在引擎内**始终**做 exclude 检查，且 **include 规则不参与扩面边界**，无法表达赏金 in-scope 模型。应改为**项目级可选**，默认关闭。
2. **持续监控**：仅有手动 `POST /scan` + 30min 绝对超时，无周期 tick、无 run 间 diff、无 change feed。
3. **边缘发现被 Profile 压住**：外网默认 katana/ffuf 关；Spoor 绑定 katana 开关，敏感信息检测外网 baseline 实际未启用。
4. **高价值信号分散**：findings / endpoints / assets / BountyCandidate 各管一段，缺少统一 inbox 与增量 API。

本变更 **不引入外部 Agent 编排**，不改造 exploit/PoC 链；只把 Anchor 演进为「可选 scope 边界 + 赏金 Profile + 信号 inbox + 监控 tick」的资产监控底座。

## 范围

### 包含

| 阶段 | 内容 | 优先级 |
|------|------|--------|
| BW0 | 可选 Scope 边界（项目开关；include + exclude；seed/work/finding 三道门） | P0 |
| BW1 | `bounty` 扫描 Preset + Spoor 独立开关 | P0 |
| BW2 | Signal Inbox（高价值信号汇聚 + 评分 + API） | P0 |
| BW3 | Delta API + 二次 run 跳过稳定资产 | P1 |
| BW4 | Watch Mode（周期 passive tick + change SSE） | P1 |
| BW5 | 边缘发现补强（gau/crt 深化、AssetJSURL 接线） | P2 |

### 不包含

- Hermes / Pi Agent 集成协议（用户自行实现）
- PoC 链、报告提交、赏金平台 API 对接
- YAML 规则引擎 / DAG 可视化编排
- 新 passive 源（Shodan、Censys 等）— 后续独立 proposal
- Neo4j / 外部图数据库
- 合并 ports/services/web_endpoints 单表（沿用 asset_relations 方案）

## Scope 边界设计原则（用户确认）

| 模式 | 适用 | 行为 |
|------|------|------|
| **关闭**（默认） | 护网 / 无 scope 限制的 recon | 扫描扩面**不**做 include 边界；全局 `exclude-domains.txt` 仍生效；项目 exclude 规则可选是否参与（见 design.md） |
| **开启** | 赏金 program | 用户配置 include（如 `*.target.com`、CIDR）+ exclude（OOS）；seed 注入、Work 派生、Finding 入库**三道过滤**；out-of-scope 资产可标记 `scope_status=out` 但不扩面 |

开关位置：**项目设置**（非 ScanModal 单次），持久化到 `projects` 表；ScanModal 只读展示当前 scope 模式。

## 影响模块

| 模块 | 变更类型 |
|------|----------|
| `internal/models/project.go` | 新增 `ScopeBoundaryMode` 字段 |
| `internal/db/v*.go` | 迁移 projects 列 |
| `internal/scope/` | `CheckInScope` / include 边界评估 |
| `internal/scanengine/seed/` | seed 过滤 |
| `internal/scanengine/engine.go` | work 门控；读 project scope 模式 |
| `internal/finding/` | finding 入库 scope 标记 |
| `internal/models/engine.go` | `DefaultBountyPipelineConfig` |
| `internal/scanengine/core/profile_config.go` | Spoor 独立开关 |
| `internal/bounty/` | Signal scorer 扩展 |
| `internal/api/` | inbox API、delta API、watch API、project scope 设置 |
| `frontend/` | 项目 Scope 设置页、Inbox 页、Watch 配置 |
| `docs/current/architecture.md` | 新章节（accept 后） |
| `docs/functional-test.md` | 场景注册表 |

## 风险与缓解

| 风险 | 缓解 |
|------|------|
| Scope 开启后漏扫 in-scope 子域 | include 规则支持 `*.domain` + 精确域；单元测试覆盖 wildcard |
| Scope 关闭时护网行为变化 | 默认 `off`；E2E 护网路径回归 |
| Watch tick 消耗 FOFA 配额 | passive-only tick + 可配置 interval + fail-soft |
| Inbox 噪声 | 评分阈值 + scope 加权 + 用户 dismiss |
| 分阶段交付中间态不可用 | 每阶段独立验收（见 tasks.md）；BW0+BW1 可先交付 MVP |

## 与现有计划关系

- **Scan Engine 收敛**：已完成；本 proposal **扩展** Profile/preset，不重写 engine 循环。
- **short-term-plan.md**：指纹 tag 映射（G2）与 BW1 Nuclei 精准度互补；不冲突。
- **BountyCandidate 现有代码**：BW2 复用 scorer，扩展 signal 来源，不另起炉灶。

## 交付策略

按 `tasks.md` 分 6 个阶段（BW0–BW5）增量交付：**每阶段完成后跑该阶段验收（unit + integration + E2E + functional-test 勾选）**，全部阶段完成后更新 `docs/active/review/bounty-watch-acceptance.md` 并 promote 到 architecture baseline。
