---
status: active
source_of_truth: false
owner: kun
last_updated: 2026-05-26
scope: repository-slimdown-audit-handoff
audience: external-implementer (e.g. DeepSeek)
audit_baseline_commit: "(执行前由接手人填写，例如 main @ <sha>)"
---

# Anchor 全量瘦身 — 实现者交接说明

> 给接手执行的 Agent（DeepSeek 等）用。**先读完本文，再动手。**
>
> **任务性质**：以**审计与产出报告**为主；**默认不改业务代码**。经用户明确批准后，才可按 backlog 拆 PR 做删除/重构。

## 1. 任务一句话

对 Anchor 仓库做一次**全量瘦身审查**：清理工作区/垃圾文件策略、文档与计划治理、死代码与迁移收口、架构耦合评估、契约与安全回归、测试与 E2E 矩阵；产出 `docs/active/review/slimdown-*.md` 系列报告，并满足本文 **§8 全局验收**。

## 2. 必读文档（按顺序）

| 顺序 | 路径 | 用途 |
|------|------|------|
| 1 | **本文** | 执行顺序、产出物、验收、约束 |
| 2 | [`docs/current/plan.md`](../../current/plan.md) | 仓库级计划；勿与 backlog 提案冲突 |
| 3 | [`docs/current/code-health-audit.md`](../../current/code-health-audit.md) | 四阶段审计 checklist（T4/T3 复用） |
| 4 | [`docs/current/audit-reports/stage1-security-contracts.md`](../../current/audit-reports/stage1-security-contracts.md) | 2026-05-13 安全/契约结论 |
| 5 | [`docs/current/audit-reports/stage2-architecture.md`](../../current/audit-reports/stage2-architecture.md) | 2026-05-13 架构/耦合结论 |
| 6 | [`docs/current/audit-reports/stage3-testing.md`](../../current/audit-reports/stage3-testing.md) | 测试策略 |
| 7 | [`docs/current/audit-reports/stage4-frontend.md`](../../current/audit-reports/stage4-frontend.md) | 前端审计 |
| 8 | [`docs/README.md`](../../README.md) | 文档生命周期规则 |
| 9 | [`CLAUDE.md`](../../../CLAUDE.md) | 文档同步、禁止单位换算、E2E 门槛 |
| 10 | [`AGENTS.md`](../../../AGENTS.md) | Go 风格、测试要求 |
| 11 | [`internal/api/README.md`](../../../internal/api/README.md) | 改 API 前的字段反向索引 |

**不要当作执行计划**：[`docs/refactoring-plan.md`](../../refactoring-plan.md)（idea pool，含过时行数描述）。

## 3. 审计基线（执行前必须做）

1. 与用户确认 **审计基线 commit**（建议：已合并的 `main`，或用户指定的「feature 合并后」SHA）。
2. 在本文 front matter 填写 `audit_baseline_commit`。
3. 记录工作区状态：
   ```bash
   git rev-parse HEAD
   git status --short
   du -sh . src-tauri/target frontend/node_modules .codegraph 2>/dev/null
   ```
4. 若存在大量 **未合并 WIP**（tool registry、external scan、task live output 等），在 T2/T3 报告中单独开一节 **「WIP 阻塞项」**，**不得**在证据不足时删除相关文件。

### 3.1 已知现状快照（2026-05-26，供对照）

| 指标 | 约值 | 备注 |
|------|------|------|
| 工作区总大小 | ~3.3GB | 主要为 `src-tauri/target/` |
| `internal/` Go 源文件 | ~175 | 非测试 |
| `internal/` 测试文件 | ~64 | ~37% 文件比（高于 2026-05-13 基线 16%） |
| 明确垃圾 | `internal/scope/import.go.bak` | 应删除 |
| 未 ignore 工具目录 | `.codegraph/` 等 | 见 T0 |
| 上帝文件（行数） | `pipeline.go` ~540、`worker/server.go` ~478、`api/server.go` ~389 | T3 重点 |
| 阶段审计 | stage1–4 已有 | 做 **delta 复审**，勿全文重写 |

## 4. 已关闭的产品决定（禁止擅自改）

| 议题 | 决定 |
|------|------|
| 瘦身范围 | **审计优先**；实现需用户逐条批准 |
| 历史归档 | **不删** `docs/archived/` 正文（仅索引/标注 superseded） |
| Handler 结构 | **禁止** 为减行数把 handler 合并回单一 `handlers.go` |
| Pipeline DAG | **禁止** 用 YAML 描述整条 pipeline（见 tool-registry 设计） |
| CLI 单位 | **禁止** 代码内 `*1000` 等单位换算 |
| 验收门槛 | 产品行为以 **E2E** 为准；仅 build/typecheck **不算** 完成 |
| Git | **禁止** 擅自 `git commit` / `git push`，除非用户明确要求 |
| 语言 | 对话与报告用**中文**；代码/错误信息保持仓库现有风格 |

## 5. 执行顺序（六轨 T0–T6，必须按序）

每轨完成后产出对应 Markdown，路径固定见 §6。单轨内可并行只读扫描，但**交付物按轨提交**。

```text
T0 工作区卫生 → T1 文档治理 → T2 死代码/迁移收口 → T3 架构耦合
    → T4 契约与安全回归 → T5 测试与 E2E 矩阵 → T6 基线回写
```

---

### T0 — 工作区与仓库卫生（0.5 工日）

**目标**：列清可清理项与 `.gitignore` 缺口；**可不执行删除**，但须给出 clean 脚本 spec。

**执行清单**：

- [ ] T0.1 扫描：`.bak`、`.tmp`、`*.log`、根目录 `image*.png`、`qa.md`、`/plan.md`（根目录跳转）
- [ ] T0.2 体积分解：`du -sh` 顶层目录；标注已在 `.gitignore` 与未覆盖项
- [ ] T0.3 对照 `.gitignore`：`.codegraph/`、`.antigravitycli/`、`.deepseek/`、`.DS_Store` 等
- [ ] T0.4 检查 untracked 是否含敏感信息（如 `*.json` 凭证）
- [ ] T0.5 建议 `scripts/clean-local.sh` 内容（`cargo clean`、`frontend` 缓存、可选 `workdirs/`）

**产出物**：[`slimdown-t0-workspace.md`](slimdown-t0-workspace.md)

**表格列**：路径 | 类型(build/cache/scratch/secret?) | 建议动作(删/ignore/保留) | 是否应入库

---

### T1 — 文档与计划瘦身（1 工日）

**目标**：单一真相源；降低 Agent 误读旧方案概率。

**执行清单**：

- [ ] T1.1 校验 `docs/**` 中带 `status:` / `source_of_truth:` 的 front matter 是否准确
- [ ] T1.2 标出与 **当前实现冲突** 的文档（重点：`docs/superpowers/*`、`docs/design/tool-registry-*`、`refactoring-plan.md` 过时描述）
- [ ] T1.3 从 stage1–4 提炼 **≤2 页**「瘦身阻塞 Top N」摘要（不复制全文）
- [ ] T1.4 Agent 技能：`.claude/skills/gitnexus` 已删 vs `.agents/skills/gitnexus-*` 重复 — 给出单一来源建议
- [ ] T1.5 建议在 `docs/current/plan.md` 增加 workstream 行（文案由你起草，**改文件需用户批准**）

**产出物**：[`slimdown-t1-docs.md`](slimdown-t1-docs.md)

---

### T2 — 死代码、迁移收口（2 工日）

**前置**：基线 commit 上 WIP 已合并或范围已冻结。

**执行清单**：

- [ ] T2.1 `go test ./...` 与 `staticcheck`（若环境有）记录结果
- [ ] T2.2 allowlist（`internal/toolguard`）vs `tools/*.yaml` vs 实际 `toolrun.Invoke` / `Build*` 调用 — 双份/孤儿清单
- [ ] T2.3 **迁移四角表**：`urlfinder` / `gau` / `passive` / `katana` — 代码、DB enum、前端类型、文档 四向一致
- [ ] T2.4 每个删除候选：`codegraph_callers` 或 `rg` 证明 **callers=0**（附命令输出摘要）
- [ ] T2.5 列出「可删 / 须保留 / 阻塞于 WIP」三类

**产出物**：[`slimdown-t2-dead-code.md`](slimdown-t2-dead-code.md)

**禁止**：无 callers=0 证据的删除建议。

---

### T3 — 架构耦合与简洁度（3 工日）

**方法**：以 [`stage2-architecture.md`](../../current/audit-reports/stage2-architecture.md) 为底稿做 **delta**（基线 commit 至今变更）。

**重点文件**：

- `internal/workflow/pipeline.go`、`pipeline_*.go`
- `internal/worker/server.go`、`dispatcher.go`
- `internal/api/*_handlers.go`（`retest`、`pipeline`、`asset`、`task_output`）
- `internal/toolrun/`、`internal/toolregistry/`

**评审维度**（每条 1–5 分 + 一句理由）：

| 维度 | 检查问题 |
|------|----------|
| 单一职责 | 函数 >80 行是否应拆 |
| 依赖方向 | handler 是否含事务/聚合业务 |
| 重复执行路径 | 是否仍有平行 `exec.Command` 与 `toolrun.Invoke` |
| 可测试性 | CLI 是否可 mock |
| 配置单位 | 是否存在禁止的单位换算 |

**产出物**：[`slimdown-t3-architecture.md`](slimdown-t3-architecture.md) + P0/P1/P2 backlog（**仅瘦身级**，不大重构）

---

### T4 — 契约、安全、pitfall 回归（2 工日）

**复用**：[`code-health-audit.md`](../../current/code-health-audit.md) 阶段一 checklist。

**额外关注（相对 2026-05-13）**：

- `internal/api/task_output_handlers.go` 与前端 `useTaskLiveOutput.ts`、`api.ts`
- `pipeline_handlers.go` 外网 preset 字段
- 7 条 `docs/pitfalls/2026042*.md` 逐项：仍成立 / 已修复 / 回归

**产出物**：[`slimdown-t4-contracts-security.md`](slimdown-t4-contracts-security.md)

**表格列**：后端字段 | 前端字段 | 差异 | 风险(Critical/High/...) | 建议

---

### T5 — 测试与 E2E 矩阵（2 工日）

**执行清单**：

- [ ] T5.1 更新测试比例（Go/TS 文件数、关键包覆盖率印象）
- [ ] T5.2 新包零测试清单：`toolregistry`、`toolrun`、`cdn/parse` 等
- [ ] T5.3 定义 **瘦身回归 E2E 最小集**（≤5 条用户流，写明 spec 路径或需新建）
- [ ] T5.4 记录运行命令与预期（环境不足则标 BLOCKED）

**建议 E2E 最小集（可调整，须在报告中写明）**：

1. 创建 Target（公司/域名任一）
2. 触发外网或默认 scan
3. 任务列表可见且状态推进
4. Task live output（若基线已实现）可拉取
5. Run 结束可在 Runs 页查看结果

**产出物**：[`slimdown-t5-test-matrix.md`](slimdown-t5-test-matrix.md)

---

### T6 — 度量回写（0.5 工日）

**执行清单**：

- [ ] T6.1 起草 `code-health-audit.md` 基线表更新稿（**提交需用户批准**）
- [ ] T6.2 起草 `plan.md` Slimdown workstream 状态（Accepted / 下一迭代链接）
- [ ] T6.3 撰写 [`slimdown-2026-summary.md`](slimdown-2026-summary.md)：一页总览 + 是否满足 §8

**产出物**：[`slimdown-2026-summary.md`](slimdown-2026-summary.md)

---

## 6. 产出物清单（全部要有）

| 文件 | 轨道 |
|------|------|
| `docs/active/review/slimdown-t0-workspace.md` | T0 |
| `docs/active/review/slimdown-t1-docs.md` | T1 |
| `docs/active/review/slimdown-t2-dead-code.md` | T2 |
| `docs/active/review/slimdown-t3-architecture.md` | T3 |
| `docs/active/review/slimdown-t4-contracts-security.md` | T4 |
| `docs/active/review/slimdown-t5-test-matrix.md` | T5 |
| `docs/active/review/slimdown-2026-summary.md` | T6 总览 |

可选（用户批准后）：`scripts/clean-local.sh`、对 `docs/current/*.md` 的小幅同步 PR。

---

## 7. 工具与命令

```bash
# 环境自检
go vet ./internal/...
go test ./...                    # 记录 pass/fail 与包级失败
cd frontend && npm run typecheck

# 体积与垃圾
du -sh . src-tauri/target frontend/node_modules .codegraph 2>/dev/null
find . -name '*.bak' -o -name '.DS_Store' 2>/dev/null | head -50

# 契约（示例）
rg 'json:"' internal/models/*.go
# 对照 frontend/src/lib/api.ts

# 结构（优先 CodeGraph MCP，无则 rg）
# codegraph_search / codegraph_callers / codegraph_impact
rg 'internal/db' internal/api/*.go
rg 'exec\.Command' internal/
```

改 API 前必读：`internal/api/README.md` 字段反向索引。

---

## 8. Definition of Done（全局验收）

### 8.1 审计阶段完成（默认目标）

全部满足才算 **瘦身审计完成**：

| ID | 验收项 | 证据 |
|----|--------|------|
| G1 | 7 份 `slimdown-*.md` 已创建且互相引用一致 | 文件列表 + summary 链接 |
| G2 | 每份报告含：**风险等级**、**可执行建议**、**阻塞项** 三节 | 人工抽查 |
| G3 | T2 删除建议 **0 条** 无 callers 证据 | t2 表格 |
| G4 | T4 Critical/High 契约差异已列全，无静默遗漏 | t4 表格 |
| G5 | T5 含可运行的 E2E 命令或 BLOCKED 原因 | t5 |
| G6 | `slimdown-2026-summary.md` 含：建议 PR 拆分（≤400 行/PR）、是否建议进入实现阶段 | summary |
| G7 | 基线 commit SHA 已写入 summary | summary 头部 |
| G8 | `go vet ./internal/...` 在审计基线上已执行并记录结果 | summary 或 t5 |
| G9 | 未擅自 commit（或 commit 仅含 review 文档且用户要求） | git log |
| G10 | 中文报告，技术术语可保留英文 | 全文 |

### 8.2 实现阶段完成（仅用户批准后）

| ID | 验收项 | 证据 |
|----|--------|------|
| I1 | 本地 clean 后工作区 **<500MB**（可不含 `node_modules`） | `du -sh` |
| I2 | 无 `.bak`、ignore 覆盖 T0 表 100% | t0 勾选 |
| I3 | 已批准删除项落地且 `go test ./...` 绿 | CI |
| I4 | 瘦身 E2E 最小集 **5/5** 绿 | playwright 报告 |
| I5 | 动过 `server.go` 则 `internal/api/README.md` 已同步 | diff |
| I6 | `code-health-audit.md` 基线日期与指标已更新 | diff |

---

## 9. 建议 PR 粒度（实现阶段，非审计必须）

| PR | 内容 |
|----|------|
| PR-A | T0：删 `.bak` + `.gitignore` + `scripts/clean-local.sh` |
| PR-B | T2 已批准死代码删除 + allowlist/registry 对齐 |
| PR-C | T3 P0 项（如 `retest_handlers` 事务下沉，单点） |
| PR-D | T1 文档 front matter / plan workstream |
| PR-E | T6 基线表更新 |

每 PR 独立可 review、可回滚；**不要** 与 feature WIP 混 PR。

---

## 10. 常见陷阱

- 在 WIP 未合并时删除 `urlfinder` / `passive/gau` 引用 — 易断构建。
- 把 `refactoring-plan.md` 当当前计划执行大重构。
- 仅跑 `go build` 宣称瘦身完成 — 违反项目 E2E 文化。
- 删除 `docs/archived/`「减负」— 非体积问题，且破坏历史追溯。
- T3 提出「合并所有 pipeline 为一个文件」— 与瘦身目标相反。

---

## 11. 风险分级（写入报告时统一使用）

| 等级 | 定义 | 示例 |
|------|------|------|
| Critical | 数据丢失/越权/安全绕过 | scope check 绕过 |
| High | 功能不可用/数据损坏 | 契约不一致导致 UI 崩溃 |
| Medium | 可维护性/分层问题 | handler 内开事务 |
| Low | 风格/小幅重复 | <20 行重复 |

---

## 12. 复制给 DeepSeek 的启动 Prompt（可直接粘贴）

```markdown
你在仓库 Anchor 执行「全量瘦身审计」，路径以本机 checkout 为准。

先只读、不要改业务代码，按顺序阅读：
1. docs/active/review/slimdown-2026-handoff-for-implementer.md（全文）
2. docs/current/code-health-audit.md
3. docs/current/audit-reports/stage1-security-contracts.md ～ stage4-frontend.md
4. CLAUDE.md、AGENTS.md、docs/current/plan.md

与用户确认审计基线 commit 后，按 T0→T6 顺序产出：
- docs/active/review/slimdown-t0-workspace.md
- docs/active/review/slimdown-t1-docs.md
- docs/active/review/slimdown-t2-dead-code.md
- docs/active/review/slimdown-t3-architecture.md
- docs/active/review/slimdown-t4-contracts-security.md
- docs/active/review/slimdown-t5-test-matrix.md
- docs/active/review/slimdown-2026-summary.md

约束：
- 默认只写报告，不删代码、不 commit（除非用户明确要求只提交 review 文档）。
- T2 删除建议必须有 callers=0 证据。
- 不做大重构；不把 handler 合并回单文件。
- 中文写报告；跑 go vet / go test 并记录结果。

完成后回复：基线 SHA、7 份报告路径、§8.1 逐项是否满足、是否建议进入实现阶段及 PR-A～E 建议顺序。
```

---

## 13. 交接检查表（给用户）

- [ ] 已指定审计基线 commit（main 或合并后分支）
- [ ] 已告知 DeepSeek：当前是否有未合并 WIP
- [ ] 已粘贴 §12 启动 Prompt
- [ ] 期望交付：7 份 `slimdown-*.md` + 可选 `clean-local.sh`
- [ ] 实现阶段需另开指令，引用本文件 §8.2 与 §9
