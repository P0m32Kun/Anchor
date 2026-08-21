---
last_updated: 2026-06-17
---

# Anchor 文档状态审计报告

> 审计日期: 2026-06-17
> 审计范围: docs/ 下全部 126 个 .md 文件
> 审计人: researcher (kanban task t_3e7fea6a)

---

## 一、Frontmatter 状态标记与实际位置不一致

### 1.1 设计文档状态与 README 声明冲突

| 文件 | Frontmatter 状态 | docs/design/README.md 声明 | 建议操作 |
|---|---|---|---|
| `docs/design/src-bounty-workstation.md` | `implemented` | "superseded 2026-06-17" | 更新 frontmatter 为 `status: superseded`，与 README 一致 |

### 1.2 design/ 目录中 status=active 但应在 current/ 的文档

| 文件 | 状态 | 说明 | 建议操作 |
|---|---|---|---|
| `docs/design/spoor-integration.md` | `active, sot=true` | 标记为 source_of_truth 且 active，但放在 design/（候选方案区）而非 current/（当前真相区） | 迁移到 `docs/current/` 或降级为 `in_review` |

### 1.3 docs/current/ 中缺少 frontmatter 的关键文档

以下文件在 `docs/current/`（当前真相区）但完全没有 frontmatter，违反维护规则第 4 条「新增文档时优先补状态头」：

| 文件 | 建议操作 |
|---|---|
| `docs/current/asset-driven-remediation-design.md` | 添加 `status: accepted` (plan.md 中已列为 Accepted) |
| `docs/current/ci-cd-guide.md` | 添加 `status: active` |
| `docs/current/deployment.md` | 添加 `status: active` (docs/README.md 已列为 Yes) |
| `docs/current/e2e-testing.md` | 添加 `status: active` (docs/README.md 已列为 Yes) |
| `docs/current/scan-api-guide.md` | 添加 `status: active` |
| `docs/current/scan-history-comparison-qa-plan.md` | 添加 `status: active` (docs/README.md 已列为 Yes) |
| `docs/current/short-term-plan.md` | 添加 `status: active` 或 `draft` |
| `docs/current/audit-reports/stage1-security-contracts.md` | 添加 frontmatter |
| `docs/current/audit-reports/stage2-architecture.md` | 添加 frontmatter |
| `docs/current/audit-reports/stage4-frontend.md` | 添加 frontmatter |

### 1.4 conventions/ 全部缺少 frontmatter

| 文件 | 建议操作 |
|---|---|
| `docs/conventions/api-contracts.md` | 添加 `status: active` |
| `docs/conventions/backend.md` | 添加 `status: active` |
| `docs/conventions/frontend.md` | 添加 `status: active` |
| `docs/conventions/testing.md` | 添加 `status: active` |
| `docs/conventions/testing-workflow.md` | 添加 `status: active` |

### 1.5 其他缺少 frontmatter 的非归档文件

| 文件 | 建议操作 |
|---|---|
| `docs/functional-test.md` | 添加 `status: active` |
| `docs/features/exclude-domains.md` | 添加 frontmatter |
| `docs/schema-migrations.md` | 添加 `status: active` (docs/README.md 已列为 Supporting) |
| `docs/design/sse-token-and-sse-auth-testing.md` | 添加 `status: in_review` 或归档 |
| `docs/plans/2026-06-03-remove-legacy-pipeline.md` | 添加 frontmatter，确认是否已执行 |

---

## 二、docs/superpowers/ 未纳入导航树的文件

`docs/README.md` 仅有一行泛指链接 `[superpowers/](superpowers/) | Mixed`，未列出任何子文件。以下 9 个文件全部未被导航树直接索引：

### 2.1 有外部引用（已融入架构/评审）

| 文件 | 状态 | 引用方 | 建议操作 |
|---|---|---|---|
| `specs/2026-05-19-builtin-assets-design.md` | `approved` | architecture.md, slimdown-t1 | 保留，考虑加入导航树 |
| `specs/2026-05-29-asset-driven-scan-engine-design.md` | `approved` | architecture.md, plan.md | 保留，考虑加入导航树 |
| `specs/2026-06-17-anchor-core-slimdown-design.md` | `accepted, sot=true` | design/README.md | 保留，应加入导航树 |
| `plans/2026-05-19-builtin-assets.md` | (无 FM) | slimdown-t1 | 添加 frontmatter |
| `plans/2026-05-25-external-scan-pipeline.md` | (无 FM) | slimdown-t1, external-scan-e0-e5 | 添加 frontmatter |

### 2.2 无外部引用（孤立文件）

| 文件 | 状态 | 建议操作 |
|---|---|---|
| `plans/2026-05-29-asset-driven-scan-engine.md` | (无 FM) | 确认是否已实现；若已实现则归档或标记 superseded |
| `plans/2026-06-01-scan-evaluator.md` | (无 FM) | 确认是否已实现；若已实现则归档或标记 superseded |
| `specs/2026-05-25-external-scan-pipeline-design.md` | `in_review` | 确认是否仍在评审；若已决定则更新状态 |
| `specs/2026-06-01-scan-evaluator-design.md` | (无 FM, 正文写"已批准") | 添加 frontmatter `status: approved`，确认实现状态 |

---

## 三、重复/冲突的文档

### 3.1 计划文档三重存在

| 文件 | 状态 | 说明 |
|---|---|---|
| `docs/current/plan.md` | `active, sot=true` | **唯一仓库级计划**（维护规则第 1 条） |
| `docs/current/short-term-plan.md` | (无 FM) | 2026-06-12 生成的短期聚焦计划，与 plan.md 可能产生平行入口 |
| `docs/refactoring-plan.md` | `backlog, sot=false` | 重构想法池，plan.md 已声明"not an approved release plan" |

**建议**: short-term-plan.md 应明确标注 `status: draft` 并在 plan.md 中链接为子计划，或合并进 plan.md 避免平行入口。

### 3.2 测试文档双文件（合理拆分，无冲突）

| 文件 | 职责 |
|---|---|
| `docs/conventions/testing.md` | 测试分层约定（金字塔、编写规范） |
| `docs/conventions/testing-workflow.md` | 开发与测试工作流（SDD→BDD→TDD） |

两个文件互相引用，职责清晰，**不是重复**。但均缺少 frontmatter。

### 3.3 设计文档目录分散

设计文档分布在三个位置：
- `docs/design/` — 候选方案区
- `docs/current/design/` — 当前候选设计索引
- `docs/superpowers/specs/` — superpowers 生成的设计

`docs/design/README.md` 和 `docs/current/design/README.md` 各自维护索引，但 `superpowers/specs/` 未被任何索引覆盖。

---

## 四、过时但未标记 superseded 的设计文档

### 4.1 design/ 中状态异常的文档

| 文件 | 当前状态 | 最后更新 | 问题 | 建议操作 |
|---|---|---|---|---|
| `docs/design/src-bounty-workstation.md` | `implemented` | 2026-06-04 | README 已声明"superseded 2026-06-17"但 frontmatter 仍为 implemented | 更新为 `superseded` |
| `docs/design/bounty-watch/*.md` (4 files) | `proposed` | (无) | README 声明"cancelled 2026-06-17"但 frontmatter 仍为 proposed | 更新为 `cancelled` |
| `docs/design/custom-nuclei-template-management.md` | `in_review` | 2026-05-05 | 40+ 天未更新，仍为 in_review | 确认是否仍在评审；若已决定则更新状态 |
| `docs/design/vuln-template-redesign.md` | `in_review` | 2026-05-19 | 28+ 天未更新 | 确认状态（plan.md 中有对应的 active plan） |
| `docs/design/tool-registry-and-artifact-store.md` | `in_review` | 2026-05-26 | 21+ 天未更新 | 确认是否仍在评审 |
| `docs/design/tool-registry-handoff-for-implementer.md` | `in_review` | 2026-05-26 | 21+ 天未更新 | 同上 |
| `docs/design/sse-token-and-sse-auth-testing.md` | (无 FM) | (无) | 无 frontmatter，状态不明 | 添加 frontmatter 或归档 |

### 4.2 active/review/ 中可能过时的验收文档

| 文件 | 状态 | 说明 | 建议操作 |
|---|---|---|---|
| `docs/active/review/bounty-watch-acceptance.md` | `draft` | bounty-watch 已取消 | 归档到 `docs/archived/` |
| `docs/active/review/external-scan-e0-e5.md` | (无 FM) | 关联的 external-scan-pipeline 设计仍为 in_review | 确认状态 |

---

## 五、docs/design/ 中 accepted 但未出现在 plan.md Active Workstreams 的项

| 文件 | 状态 | plan.md 中是否存在 | 建议操作 |
|---|---|---|---|
| `docs/design/scan-engine-convergence/` (4 files: design, proposal, spec, tasks) | `accepted` | **否** — plan.md 的 "Asset-driven scan engine" 引用的是 `internal/scanengine/` 而非此设计目录 | 确认此设计是否已被 Asset-driven scan engine workstream 吸收；若是则标记 superseded 并归档 |
| `docs/design/hw-scan-optimization/design.md` | `accepted` | **否** — plan.md 中无对应 workstream | 添加到 plan.md Active Workstreams 或确认是否为独立已完成项 |

---

## 六、汇总统计

| 类别 | 数量 |
|---|---|
| 总文件数 | 126 |
| 有完整 frontmatter 的 | 68 (54%) |
| 缺少 frontmatter 的 | 58 (46%) |
| 状态与位置不一致的 | 13 |
| 孤立/未导航的 superpowers 文件 | 9 |
| accepted 但不在 plan.md 的设计 | 5 (目录级) |
| 过时未标记的 | 7+ |
| 需要 frontmatter 补全的 | 45+ |

---

## 七、优先修复建议

1. **P0 — 状态冲突**: 更新 `src-bounty-workstation.md` 和 `bounty-watch/` 的 frontmatter 与 README 一致
2. **P0 — 位置冲突**: 决定 `spoor-integration.md` 归属（current/ 还是 design/）
3. **P1 — 缺失 frontmatter**: 为 `docs/current/` 和 `docs/conventions/` 下所有文件补充 frontmatter
4. **P1 — accepted 设计归档**: 确认 `scan-engine-convergence/` 和 `hw-scan-optimization/` 的当前状态
5. **P2 — superpowers 导航**: 为 superpowers/ 下的活跃文件建立索引入口
6. **P2 — 计划文档去重**: 明确 short-term-plan.md 与 plan.md 的关系
7. **P3 — 批量补 frontmatter**: 覆盖 pitfalls/、mempalace/、features/ 等目录
