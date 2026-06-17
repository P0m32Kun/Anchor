# Anchor 文档中心

> 最后更新：2026-06-17
> 默认原则：任何 agent 在阅读历史材料前，先读本页和 `docs/current/`

## 先读这些

按默认顺序阅读：

1. [`../README.md`](../README.md)
2. [`current/agent-guide.md`](current/agent-guide.md)
3. [`current/plan.md`](current/plan.md)
4. [`current/architecture.md`](current/architecture.md)
5. [`current/design/README.md`](current/design/README.md)
6. [`current/decisions/README.md`](current/decisions/README.md)

## 当前有效文档

| 文档 | 角色 | 是否为当前真相 |
| --- | --- | --- |
| [`current/agent-guide.md`](current/agent-guide.md) | Coding agent 迭代协议 | Yes |
| [`current/plan.md`](current/plan.md) | 当前唯一仓库级计划 | Yes |
| [`current/architecture.md`](current/architecture.md) | 当前唯一架构基线 | Yes |
| [`current/deployment.md`](current/deployment.md) | **客户部署**（install.sh / ACR 镜像） | Yes |
| [`current/e2e-testing.md`](current/e2e-testing.md) | **开发者 E2E**（fast 镜像 / Playwright） | Yes |
| [`current/faq.md`](current/faq.md) | 常见问题解答 | Yes |
| [`current/ci-cd-guide.md`](current/ci-cd-guide.md) | CI/CD 流程（GitHub Actions） | Yes |
| [`current/scan-api-guide.md`](current/scan-api-guide.md) | Scan API 使用指南 | Yes |
| [`current/code-health-audit.md`](current/code-health-audit.md) | 工程健康审计 | Yes |
| [`current/asset-driven-remediation-design.md`](current/asset-driven-remediation-design.md) | 资产驱动扫描修复设计 | Yes |
| [`current/design/README.md`](current/design/README.md) | 当前候选设计索引 | Yes |
| [`current/decisions/README.md`](current/decisions/README.md) | 当前决策索引 | Yes |
| [`current/scan-history-comparison-qa-plan.md`](current/scan-history-comparison-qa-plan.md) | 扫描历史对比 QA 计划 | Yes |
| [`functional-test.md`](functional-test.md) | BDD 层手工验收 + 场景注册表 | Supporting |
| [`api-error-contract.md`](api-error-contract.md) | API 错误约定 | Supporting |
| [`schema-migrations.md`](schema-migrations.md) | Schema 迁移策略与版本历史 | Supporting |
| [`CHANGELOG.md`](CHANGELOG.md) | 项目变更日志 | Supporting |

### 当前审计报告

`current/audit-reports/` 存放分阶段审计报告：

| 报告 | 范围 |
| --- | --- |
| [`stage1-security-contracts.md`](current/audit-reports/stage1-security-contracts.md) | 安全合约审计 |
| [`stage2-architecture.md`](current/audit-reports/stage2-architecture.md) | 架构审计 |
| [`stage3-testing.md`](current/audit-reports/stage3-testing.md) | 测试审计 |
| [`stage4-frontend.md`](current/audit-reports/stage4-frontend.md) | 前端审计 |

## 验收与评审

`active/review/` 存放设计提案的验收记录：

| 文档 | 关联提案 |
| --- | --- |
| [`active/review/batch-scan-scheduling-acceptance.md`](active/review/batch-scan-scheduling-acceptance.md) | `design/batch-scan-scheduling/` |
| [`active/review/scan-engine-convergence-acceptance.md`](active/review/scan-engine-convergence-acceptance.md) | `design/scan-engine-convergence/` |
| [`active/review/external-scan-e0-e5.md`](active/review/external-scan-e0-e5.md) | `superpowers/plans/external-scan-pipeline` |
| [`active/review/bounty-watch-acceptance.md`](active/review/bounty-watch-acceptance.md) | `design/bounty-watch/`（已取消） |

## 候选方案与评审材料

这些文档可以参考，但默认不应当作当前真相：

| 文档/目录 | 状态 | 说明 |
| --- | --- | --- |
| [`design/`](design/) | Mixed | 新设计提案区（详见 [`design/README.md`](design/README.md)） |
| [`superpowers/`](superpowers/) | Mixed | 外网扫描 pipeline 等 superpowers 设计文档；部分已归档 |
| [`refactoring-plan.md`](refactoring-plan.md) | Backlog | 重构想法和拆分路线 |

## 归档

- 历史版本文档统一放在 [`archived/`](archived/)
- 详细归档说明见 [`archived/README.md`](archived/README.md)
- 归档内容包括：v0.1–v0.4 版本文档、slimdown review 系列、废弃 plans

## 实施计划

[`plans/`](plans/) 存放待执行或进行中的实施计划：

- [`plans/2026-06-03-remove-legacy-pipeline.md`](plans/2026-06-03-remove-legacy-pipeline.md) — 清除旧流水线扫描模式

## 功能文档

[`features/`](features/) 存放已实现功能的说明文档：

- [`features/exclude-domains.md`](features/exclude-domains.md) — 全局域名排除列表

## 知识库（mempalace）

[`mempalace/`](mempalace/) 存放跨项目的决策、约定和反模式知识：

| 文档 | 类型 |
| --- | --- |
| [`decision-watch-ffuf-default.md`](mempalace/decision-watch-ffuf-default.md) | Decision |
| [`decision-mempalace-autosave-architecture.md`](mempalace/decision-mempalace-autosave-architecture.md) | Decision |
| [`convention-design-status-lifecycle.md`](mempalace/convention-design-status-lifecycle.md) | Convention |
| [`anti-pattern-sqlite-memory-pool.md`](mempalace/anti-pattern-sqlite-memory-pool.md) | Anti-pattern |

## 通用参考

| 位置 | 说明 |
| --- | --- |
| [`conventions/`](conventions/) | 编码规范（含 [`testing-workflow.md`](conventions/testing-workflow.md) 测试工作流、[`backend.md`](conventions/backend.md)、[`frontend.md`](conventions/frontend.md)、[`api-contracts.md`](conventions/api-contracts.md)、[`testing.md`](conventions/testing.md)） |
| [`pitfalls/`](pitfalls/) | 踩坑记录（详见 [`pitfalls/README.md`](pitfalls/README.md)） |

## 待处理

[`inbox/`](inbox/) 存放待分类的文档：

- [`inbox/audit-anchor-docs-status.md`](inbox/audit-anchor-docs-status.md) — 文档状态审计报告

## 维护规则

1. 任意时刻只保留一份仓库级 plan 和一份仓库级 architecture。
2. 新设计先进入 `docs/design/`，只有被提升后才能改写 `docs/current/`。
3. 阶段结束后，把计划、设计和评审材料移入 `docs/archived/` 或明确标成 `archived` / `superseded`。
4. 新增文档时优先补状态头，而不是再新增一个平行入口。
5. 面向 agent 的流程、验证门槛和本地 artifact 规则统一维护在 [`current/agent-guide.md`](current/agent-guide.md)。

### 版本号规则

项目采用三位版本号 `vMAJOR.MINOR.PATCH`（当前 `v0.4.0`），递增规则如下：

| 位置 | 递增条件 | 示例 |
|------|---------|------|
| **PATCH**（最后一位） | 小更新：bug 修复、文档同步、测试补充、配置调整 | `v0.2.1` → `v0.2.2` |
| **MINOR**（中间一位） | 一般更新：新功能、新模块、API 变更、UI 改进 | `v0.2.x` → `v0.3.0` |
| **MAJOR**（第一位） | 较大重构和更新：架构变更、数据模型重构、破坏性变更 | `v0.x.x` → `v1.0.0` |

- MINOR 或 MAJOR 递增时，低位归零。
- 当前处于 `v0.x.x` 阶段，MAJOR=0 表示尚未正式稳定发布；首次稳定版为 `v1.0.0`。
