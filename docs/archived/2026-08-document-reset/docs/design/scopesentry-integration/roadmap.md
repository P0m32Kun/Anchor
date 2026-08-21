---
status: draft
source_of_truth: false
owner: kun
last_updated: 2026-08-20
scope: distributed-asm-evolution
verification: pending_implementation
---

# ScopeSentry 整合与分布式演进 — 里程碑

> Status: Draft
> 依赖：[`design.md`](design.md)（本文任务引用其章节编号）。
> 原则：每个里程碑独立可验收、可发布；E2E 全绿是进入下一里程碑的硬门槛。

---

## M1 — 存储与通信演进（外壳改造）

**目标**：消除分布式天花板（SQLite 单写者），收敛通信单通道，实现断点续扫。

### 任务

| # | 任务 | 要点 |
| --- | --- | --- |
| M1-1 | PostgreSQL 迁移基线 | 以 v44 schema 为基线合并重写为 Postgres v1 迁移；驱动换 pgx；`docs/schema-migrations.md` 增补 |
| M1-2 | queries 层平移 | 112 个 queries 文件方言适配；既有 queries 测试平移并全绿 |
| M1-3 | work 队列原子抢占平移 | TryClaim 改 `SELECT ... FOR UPDATE SKIP LOCKED` |
| M1-4 | 通信收敛单通道 | 删除 Dispatcher 推模式；保留长轮询（间隔可配，默认 5s）+ 30s 心跳；取消信号经轮询响应下发 |
| M1-5 | 断点续扫 | server 重启：running work 归还 pending 重入队；orphan 标 failed 逻辑删除 |
| M1-6 | 部署与 CI | docker-compose 增加 postgres；CI 起 Postgres service 跑全量测试；E2E 基础套件适配 |

### 验收

- [ ] 多 worker（≥2）E2E：任务分配、故障转移（kill 一个 worker 任务被改派）
- [ ] 断点续扫 E2E：扫描中重启 server，pending work 恢复执行，run 最终 completed
- [ ] 既有 163 个 Go 测试 + 37 个 Playwright E2E 在 Postgres 后端下全绿
- [ ] 压测参考：1000 目标、2 worker，无 SQLite lock 报错，tool call 批量率不回退

---

## M2 — POC 库对接与指纹路由验证

**目标**：owner 独立 POC 维护库成为唯一 POC 源，指纹路由链路验证。

### 任务

| # | 任务 | 要点 |
| --- | --- | --- |
| M2-1 | builtin 同步源切换 | 模板 bundle 指向 owner POC 仓库；同步格式（nuclei YAML + tech→tags 元数据）定义并在 owner 库 README 说明 |
| M2-2 | 路由回归测试 | tech→tags 映射对 owner 库标签体系回归；`_default: skip`（无指纹不跑）保持 |
| M2-3 | 同步可靠性 | 同步失败降级（沿用旧 bundle）+ 告警事件 |

### 验收

- [ ] owner 库新增 POC → 60s 内 worker 可用，无重发版
- [ ] 靶场 E2E：已知指纹资产命中对应 POC；无指纹资产不触发 nuclei

---

## M3 — 扫描广度模块（借鉴重实现）

**目标**：补齐测绘平台必备的发现类别。**全程遵守 D5：不复制 ScopeSentry 代码。**

### 任务（按优先级）

| # | 任务 | design.md |
| --- | --- | --- |
| M3-1 | 测绘引擎补齐 shodan/censys/zoomeye + 蜜罐标记回写 attrs | §5.0 |
| M3-2 | findings category 字段 + 迁移（nuclei/sensitive/takeover/dir） | §6 |
| M3-3 | 敏感信息检测 SCAN_SENSITIVE（规则预编译 + AC 预筛 + 并行规则组；规则源许可评审先行） | §5.1 |
| M3-4 | 子域接管 CHECK_TAKEOVER（CNAME 指纹表 + 统一 client 验证） | §5.2 |
| M3-5 | 目录扫描成洞（ffuf → findings(category=dir) 阈值规则） | §5.3 |
| M3-6 | URL 去重 uro 语义 Go 实现（进派生队列前生效） | §5.5 |
| M3-7 | 通知渠道 dingtalk/feishu/wecom/telegram + category 路由 | §5.4 |

### 验收

- [ ] 每模块：单元测试 + 靶场 E2E + findings 进人工验证队列
- [ ] 敏感信息/接管规则数据源许可清单评审通过（R1）
- [ ] 派生规则测试：动作门控（Precondition）覆盖新 action，不破坏 Fair/Staged 调度

---

## M4 — UI 重做与策略引擎完善

**目标**：按 ScopeSentry 信息架构重做视图层，低影响策略引擎完整落地。

### 任务

| # | 任务 | 要点 |
| --- | --- | --- |
| M4-1 | 视觉基线 | 从 ScopeSentry 公开演示提取风格基线（色板/密度/组件形态），写 design tokens |
| M4-2 | 统一资产浏览页 | 多类型资产视图 + 高级筛选（对齐 ScopeSentry 资产管理页） |
| M4-3 | 资产 diff / 变更历史页 | 暴露已有 asset_changes/snapshots；对接 signal 告警 |
| M4-4 | 任务页融合 | ScopeSentry 式进度汇总 + 保留 Work 明细观察台 |
| M4-5 | 新增发现页 | 敏感信息 / 接管 / 目录发现（复用 Findings 队列组件） |
| M4-6 | 引擎页扩展 | 6 引擎 + 蜜罐标记 + 配额展示 |
| M4-7 | 策略引擎 | 速率档位、端口升级策略、被动优先 Precondition（§3.5 全表） |
| M4-8 | （可选）指纹库数据扩充 | 许可明确的数据源；与 tech→tags 路由兼容 |

### 验收

- [ ] 前端 vitest + Playwright 全绿；新页面覆盖对应 API
- [ ] 策略 E2E：gentle 档位下 per-IP QPS 符合限制；CDN/蜜罐资产零主动端口扫描
- [ ] 双跑对比：同一目标集 vs 旧 Anchor（扫描覆盖 + tool call 数 + 耗时）报告归档

---

## 里程碑外（Backlog，不承诺）

- 多用户 / RBAC（O3）
- SQLite 嵌入式单机模式（O2）
- gRPC 通道（O1）
- 资产关系图可视化（asset_relations 血缘已有后端）
- 持续监控 watch 的产品化（watch/signal 后端已有大半）
