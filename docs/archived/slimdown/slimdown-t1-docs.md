---
status: archived
source_of_truth: false
owner: kun
audit_date: 2026-05-26
audit_baseline_commit: "main @ 7399a5f"
scope: document-governance
archived_date: "2026-06-17"
archive_reason: "slimdown series completed, moved from active/review/"
---

# T1 报告：文档与计划瘦身

> 审计日期：2026-05-26 | 基线 commit：`7399a5f`

---

## 1. Front Matter 校验

### 1.1 状态准确性核查

| 文档 | 当前 status | 应改为 | 理由 |
|------|-------------|--------|------|
| `docs/archived/v0.4/cyberos-ui-redesign.md` | `in-review` | `superseded` | 位于 `archived/` 下但状态是 `in-review`，v0.4 已结束 |
| `docs/archived/v0.4/visionos-glassmorphism-redesign.md` | `in-review` | `superseded` | 同上，两个设计在 v0.4 中被搁置 |
| `docs/archived/v0.4/acceptance.md` | `accepted` | `superseded` | v0.4 验收标准，但 `source_of_truth: true` 不应指向归档；应由 `plan.md` 的「Active Workstreams」表管理 |
| `docs/design/tool-registry-handoff-for-implementer.md` | `active` | `in_review` | 设计文档 `tool-registry-and-artifact-store.md` 是 `in_review`，其 handoff 不应高一级 |

### 1.2 归档文档状态合规性

`docs/archived/` 下所有文档均应有 `source_of_truth: false`。抽查通过：
- v0.1 全部 ✅（archived / superseded / completed）
- v0.2 全部 ✅（completed / archived）
- v0.3 全部 ✅（completed / review_material）
- v0.4 两处违规 ❌（见 1.1）

### 1.3 `docs/superpowers/` 目录未纳入文档治理

```
docs/superpowers/
  plans/
    2026-05-19-builtin-assets.md（tracked? → 已入库）
    2026-05-25-external-scan-pipeline.md（untracked）
  specs/
    2026-05-19-builtin-assets-design.md（tracked）
    2026-05-25-external-scan-pipeline-design.md（untracked）
```

`docs/superpowers/` **不在 `docs/README.md` 的导航树中**，文件有 front matter 但 `docs/README.md` 未索引。两个 `2026-05-25-*` 文件处于 untracked 状态。

---

## 2. 与当前实现冲突的文档

| 文档 | 冲突内容 | 风险 |
|------|---------|------|
| `docs/refactoring-plan.md` | 声称阶段 1-6 已完成，但描述的行数、文件关系与当前架构不符 | Medium — 前台标注 `status: backlog`, `source_of_truth: false`，但实现者可能误读 |
| `docs/archived/v0.4/acceptance.md` | 标注为 v0.4 验收标准且 `source_of_truth: true`，但 `plan.md` 已明确 v0.4 为「Accepted」状态 | Low — 内容本身准确，但不应继续作为 truth source |
| `docs/design/vuln-template-redesign.md` | 提到 `status: accepted` 升级计划（§2815-2616），但 `plan.md` 中 `vuln-template-redesign` 未出现在 Active Workstreams | Medium — 设计已冻结但 plan 未跟踪 |

**结论**：无影响实施的重大冲突。主要问题是 `refactoring-plan.md` 的行数/文件描述过时，和 `vuln-template-redesign.md` 的 plan 跟踪缺失。

---

## 3. 瘦身阻塞 Top N 摘要（从 stage1-4 提炼）

| # | 阻塞项 | 来源 | 等级 | 影响 T? |
|---|--------|------|------|---------|
| 1 | **worker 全包 0% 覆盖率** | stage3 | Critical | T5 — 无测试保护，死代码删除风险高 |
| 2 | **workflow/discovery.go 编排 0% 测试** | stage3 | Critical | T2/T3 — 编排路径无回归 |
| 3 | **Project 后端缺失 `start_time`/`end_time`** | stage1 | Critical | T4 — scope 时间窗口控制完全失效 |
| 4 | **worker 注册无去重 + 超时任务未重分配** | stage1 | High | T3 — ghost worker 风险 |
| 5 | **handler 内直接开事务（retest_handlers）** | stage2 | High | T3 — 分层违规 |
| 6 | **api.ts 976L 上帝文件 + 35 空 catch** | stage4 | High | T1 — 文档化，不阻塞其他轨 |
| 7 | **CDN IP 未过 scope** | stage1 | Medium | T4 — 边界绕过风险 |

---

## 4. Agent 技能去重

### 4.1 现状

在 `7399a5f` 中已删除 `.claude/skills/gitnexus/`（6 个 SKILL.md 文件），但：

| 文件 | 存在状态 | 说明 |
|------|----------|------|
| `.claude/agents/semble-search.md` | ✅ 存在 | Claude 端 Agent 配置 |
| `.cursor/agents/semble-search.md` | ✅ 存在 | Cursor 端 Agent 配置（内容基本相同） |
| `.reasonix/agents/` | ❌ 不存在 | 当初 commit 称「迁移至 .reasonix/agents/」但目录未创建 |

### 4.2 建议

1. **删除 `.cursor/agents/`** — 与 `.claude/agents/semble-search.md` 内容重复
2. **不重建 `.reasonix/agents/`** — 当前无跨平台技能库需求，单一来源即可
3. **技能托管策略** — 建议在 `CLAUDE.md` 中直接定义技能引用，不依赖单独的文件目录

---

## 5. 建议在 `plan.md` 中新增的 Workstream

建议在 `docs/current/plan.md` 的「Active Workstreams」表新增一行（**需用户批准后修改**）：

```
| Slimdown 2026 | Active | Yes | docs/active/review/slimdown-2026-handoff-for-implementer.md |
```

同时将「Code health audit」状态从 `Active` 改为 `Accepted`（已出阶段 1-4 报告，且被 Slimdown 覆盖）。

---

## 6. 建议修复清单（改文件需批准）

| 优先级 | 文件 | 修改内容 |
|--------|------|----------|
| P0 | `docs/archived/v0.4/cyberos-ui-redesign.md` | `status: superseded` |
| P0 | `docs/archived/v0.4/visionos-glassmorphism-redesign.md` | `status: superseded` |
| P0 | `docs/archived/v0.4/acceptance.md` | `status: superseded`, `source_of_truth: false` |
| P1 | `docs/design/tool-registry-handoff-for-implementer.md` | `status: in_review` |
| P1 | `docs/README.md` | 增补 `docs/superpowers/` 导航索引 |
| P1 | `docs/current/plan.md` | 增补 Slimdown workstream，标记 Code health audit 为 Accepted |
| P2 | 删除 `.cursor/agents/` | 与 `.claude/agents/` 重复 |
