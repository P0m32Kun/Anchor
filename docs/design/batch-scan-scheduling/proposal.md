---
status: draft
source_of_truth: false
owner: kun
created: 2026-06-17
scope: batch-scan-scheduling
supersedes_partial:
  - docs/design/hw-scan-optimization/design.md  # AD-3/AD-5 接线与批量 work 未落地部分
related_incidents:
  - run id-1781616036504596506-2  # 1067 domain → 6597 work / 5791 tool calls
parent_workstreams:
  - docs/design/scan-engine-convergence/design.md
  - docs/design/hw-scan-optimization/design.md
---

# 批量扫描调度 — Proposal

## 问题

2026-06-16 外网扫描复盘（项目 `id-1781586450778379925-8`，run `id-1781616036504596506-2`）：

| 指标 | 实测 |
|------|------|
| 输入目标 | 1067 domain |
| Work items | 6597 |
| 工具调用 | 5791 |
| 并发 | BatchSize=5 固定 |
| 前端单次轮询 payload | works 2.2MB + tool-calls 5.5MB |

根因不是「目标太多」，而是 **调度粒度 = 1 资产 × 1 动作 = 1 次 CLI**，且 [`hw-scan-optimization/design.md`](../hw-scan-optimization/design.md) 中 Stage Rank、公平调度、IP 节流、domainpool 等 **代码已写、测试已有，但未接入 `engine.go`**。

## 目标

1. **按工具特性分三层批量**（大池 / 同质分组 / 单点），把 1000 级目标的 CLI 调用降到 ~150–200 次量级（nuclei/nmap 因精准度要求节省幅度较小）。
2. **阶段门控 + 阶段内池化**：一次 Run 内 DNS→CDN→Port→Web→Vuln 顺序不变；各阶段内部批量执行。
3. **Nuclei 按 httpx tech 分桶路由**，禁止「宽 tag 全模板」批量（策略 A 弃用）。
4. **Nmap 按 IP 聚合 open ports**，不做全局大池、不维持 1 port = 1 次 nmap。
5. **不设扫描时间硬上限**；Run 以 queue 排空 + resume 为正常结束；补齐 orphan run 恢复。
6. **前端/API 分页**：轮询 metrics/summary，明细 lazy load。

## 非目标

- 引入外部 DAG 框架（继续 ScanEngine + 资产门控）
- 修改 CLI 工具参数单位语义
- 要求用户手动分多次 Run 完成各阶段（阶段在引擎内自动完成）

## 与现有设计的关系

| 文档 | 关系 |
|------|------|
| `scan-engine-convergence` | G2 asset_state、G3 high-value 门控 — nuclei 分桶的前置依赖 |
| `hw-scan-optimization` | AD-3 Stage Rank、AD-5 调度 — **本提案落地其未接线部分** |
| `custom-nuclei-template-management` | RoutingPolicy — 与 AD-B5 tech 路由表协同 |

## 交付物

| 文件 | 用途 |
|------|------|
| [design.md](./design.md) | 架构决策与数据流 |
| [spec.md](./spec.md) | 可验收 REQ |
| [tasks.md](./tasks.md) | 分阶段实施清单与文件映射 |

## 批准门禁

- [ ] 用户确认三分法批量模型（Tier 1/2/3）
- [ ] 用户确认 nuclei 弃用策略 A，默认 tech 路由 + 无 tech skip（低噪音）
- [ ] 用户确认不设 AbsoluteTimeout 默认值

批准后进入 Implement：按 [tasks.md](./tasks.md) P0→P3 顺序，每 PR ≤400 行。
