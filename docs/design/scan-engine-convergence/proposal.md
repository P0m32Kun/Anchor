---
status: accepted
source_of_truth: false
owner: kun
created: 2026-06-07
scope: scan-engine-convergence
related_plan: docs/current/plan.md
---

# Scan Engine 收敛 — Proposal

## 动机

Anchor 已完成资产驱动 ScanEngine（`internal/scanengine/`），方向正确，但存在三类结构性问题导致「功能齐全、护网难用」：

1. **外网 Tier-0 入口不完整**：企业名（company）是护网最常见入口，应经 FOFA/Hunter/Quake 展开为 domain/ip/url；当前 `seed/expand.go` 仅接 FOFA，与 architecture 文档承诺不一致；前端已移除显式 company 入口。
2. **资产图谱未落库**：运行时 `DiscoveryAsset.ParentID` 存在，DB 无 `asset_relations`；`discovery_depth` 列从未写入。
3. **执行面过宽**：外网默认 katana/ffuf 全开；legacy `internal/workflows/` 与 ScanEngine 双轨并行。

本变更 **不扩展新工具**，而是收敛结构，使 Anchor 成为护网可直接使用的「攻击面扩展引擎」。

## 范围

### 包含

| 阶段 | 内容 |
|------|------|
| G0 | 企业名 → FOFA/Hunter/Quake 并行展开；读 `EnablePassiveSearch`；恢复前端 company 入口 |
| G1 | `asset_relations` 表 + lineage API；company→seed 血缘 |
| G2 | `asset_state` 持久化（Fingerprinted/Alive/CDN/Technologies） |
| G3 | 外网 Profile 收敛；high-value 门控 katana/spoor/ffuf |
| G4 | 删除 legacy workflows；统一扫描入口 |
| G5 | preconditions 提取；为后续 YAML 规则铺路（P2，可选） |

### 不包含

- 新工具集成（subfinder 变种、新 passive 源等）
- 完整 YAML 规则引擎 UI
- ScanEngine 重写为 DAG 框架
- 合并 ports/services/web_endpoints 为单表（Phase 6B，后续再议）
- Neo4j / 外部图数据库

## 影响

| 模块 | 变更类型 |
|------|----------|
| `internal/scanengine/seed/` | 重构 expand；新增 passive_search |
| `internal/db/` | v33 迁移 asset_relations；asset_state |
| `internal/api/` | lineage API；删 workflow handlers |
| `internal/workflows/` | **删除** |
| `frontend/src/pages/TargetPage.tsx` | 恢复 company 选项 |
| `docs/current/architecture.md` | 被动搜索路径、黄金链、删过时描述 |
| `docs/functional-test.md` | 新增 BDD 场景 |

## 风险

| 风险 | 缓解 |
|------|------|
| Hunter/Quake API 配额消耗 | `PassiveSearchResultLimit` + fail-soft |
| 删 legacy workflow 影响现有用户 | AssetPage 改调 ScanModal；E2E 覆盖 |
| asset_relations 迁移 | 增量迁移，不影响现有 assets 表 |
| 外网默认关 katana 行为变化 | ScanModal 高级面板可显式开启；文档说明 |

## 调研结论（Research 阶段产出）

### 已确认正确（保留）

- 事件驱动循环：`processNewAsset → DeriveEligibleWorks → execute → onWorkComplete`
- Work 状态机：`ScanWorkItem` + running/wind_down/stopped
- Orchestrator 模型：`toolrun.Invoke` + Worker 外部 CLI
- `TargetTypeCompany` + `scope.DetectType` 后端支持；auto 模式可隐式识别企业名

### 已确认缺口

| 项 | 现状 |
|----|------|
| company 展开 | 仅 FOFA（`seed/expand.go`） |
| `EnablePassiveSearch` | expand 未读 |
| 前端 company | TargetPage 无选项；E2E `v0.4-company-flow.spec.ts` 已 skip |
| Hunter/Quake | 客户端存在，未接入 seed |
| 血缘 | ParentID 内存 only |
| legacy workflow | `workflows/asset-discovery` 仍注册；AssetPage 仍调用 |

## 批准状态

- [x] 用户批准 proposal（2026-06-07）
- [x] spec.md REQ 验收信号已评审
- [x] tasks.md 已确认 PR 拆分

## 参考

- 对话收敛清单（2026-06-07）
- [`docs/current/architecture.md`](../../current/architecture.md) 执行模型
- [`docs/current/asset-driven-remediation-design.md`](../../current/asset-driven-remediation-design.md)
- [`docs/design/scan-engine-convergence/spec.md`](spec.md)
- [`docs/design/scan-engine-convergence/design.md`](design.md)
- [`docs/design/scan-engine-convergence/tasks.md`](tasks.md)
