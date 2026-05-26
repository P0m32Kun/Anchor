---
status: active
source_of_truth: true
owner: kun
audit_date: 2026-05-26
audit_baseline_commit: "main @ 7399a5f"
scope: slimdown-summary
---

# Slimdown 2026 — 全量瘦身审计总览

> 审计日期：2026-05-26 | 基线 commit：`7399a5f`（`main`）
> 审计方式：只读 + 报告产出，默认不改代码。用户已批准的执行项见下。

---

## 0. 基线状态

| 指标 | 值 |
|------|-----|
| 工作区总大小 | ~3.3G（`src-tauri/target/` 占 3.0G） |
| Go 源文件 | 175 |
| Go 测试文件 | 64（37%）|
| 前端源文件 | 56 |
| 前端测试文件 | 4（7%）|
| `go vet ./internal/...` | ✅ 通过 |
| `go test ./internal/...` | ✅ 29/29 全绿 |
| 基线 commit | `7399a5f` |

---

## 1. 产出物清单

| # | 文件 | 轨道 | 状态 |
|---|------|------|------|
| ✅ | [`slimdown-t0-workspace.md`](slimdown-t0-workspace.md) | T0 | 已产出 |
| ✅ | [`slimdown-t1-docs.md`](slimdown-t1-docs.md) | T1 | 已产出 |
| ✅ | [`slimdown-t2-dead-code.md`](slimdown-t2-dead-code.md) | T2 | 已产出 |
| ✅ | [`slimdown-t3-architecture.md`](slimdown-t3-architecture.md) | T3 | 已产出 |
| ✅ | [`slimdown-t4-contracts-security.md`](slimdown-t4-contracts-security.md) | T4 | 已产出 |
| ✅ | [`slimdown-t5-test-matrix.md`](slimdown-t5-test-matrix.md) | T5 | 已产出 |
| ✅ | **本文** | T6 | 已产出 |

---

## 2. Definition of Done — 审计阶段（G1–G10）

| ID | 验收项 | 证据 | 状态 |
|----|--------|------|------|
| G1 | 7 份 `slimdown-*.md` 已创建且互相引用一致 | 上表 + 本文件链接 | ✅ |
| G2 | 每份报告含：风险等级、可执行建议、阻塞项三节 | 抽查 T0–T5 均包含 | ✅ |
| G3 | T2 删除建议 0 条无 callers 证据 | T2 表格（`StageURLFinder` 证据充足） | ✅ |
| G4 | T4 Critical/High 契约差异已列全，无静默遗漏 | 5 个存量问题 + 新增端点 | ✅ |
| G5 | T5 含可运行的 E2E 命令或 BLOCKED 原因 | 5 条用户流 + 审计环境可运行的单元测试 | ✅ |
| G6 | summary 含建议 PR 拆分（≤400 行/PR），含是否建议进入实现阶段 | 见 §3、§4 | ✅ |
| G7 | 基线 commit SHA 已写入 summary | 本文 front matter `7399a5f` | ✅ |
| G8 | `go vet ./internal/...` 已执行并记录结果 | §0 + T2 | ✅ |
| G9 | 未擅自 commit（或 commit 仅含 review 文档且用户要求） | 用户要求推送的除外 | ✅ |
| G10 | 中文报告 | 全文中文 | ✅ |

**审计阶段完成度：10/10 ✅ 全部满足**

---

## 3. 是否建议进入实现阶段

### ✅ 建议进入实现阶段

理由：
1. 审计发现明确，且执行修复的风险可控
2. 7 条 pitfall 全部无回归，证明现有修复稳定
3. 测试覆盖率已大幅改善（worker 0%→12.6%，workflow 8.9%→15.3%），重构安全性高于基线

### 实施前需注意

- T4 的 5 个 Critical/High 存量问题**不属瘦身范围**，不应在瘦身 PR 中混入修复
- `pipeline_tool.go` 的拆分是最大架构改动，建议拆为独立 PR

---

## 4. 建议 PR 拆分

| PR | 内容 | 预估行数 | 风险 |
|----|------|---------|------|
| **PR-A** | ~~T0：删 `.bak` + `.gitignore` + `scripts/clean-local.sh`~~ | ~50 | Low — **已在审计阶段执行**（`7399a5f`） |
| **PR-B** | T1 文档 front matter 修复（4 个文件 status 修正 + plan.md workstream 更新） | ~30 | Low |
| **PR-C** | T3 P0：拆分 `pipeline_tool.go`（771L→12 个工具文件） | +160/–0 | Med — 无逻辑变更，纯拆分 |
| **PR-D** | T3 P1：统一 CDN 检测路径走 toolrun + 消除 HighRiskPorts 重复 | ~30 | Med |
| **PR-E** | T6：`code-health-audit.md` 基线表更新 + `plan.md` Slimdown workstream | ~20 | Low |

**建议执行顺序**：PR-B → PR-C → PR-D → PR-E

每 PR ≤400 行变更（PR-C 为拆分纯新增，无逻辑变更）。

---

## 5. 各轨道核心建议汇总

| 轨道 | P0 级 | P1 级 | P2 级 |
|------|-------|-------|-------|
| T0 | ✅ 已执行（.gitignore + 删 E2E 截图 + 删 .bak） | — | — |
| T1 | 修正 3 个 archived 文档 front matter | plan.md workstream 增补 | 删除 `.cursor/agents/` |
| T2 | 无（urlfinder 代码已删） | — | 标注 `SlowScanToolURLFinder` Deprecated |
| T3 | 拆分 `pipeline_tool.go` | 统一 CDN 检测路径 | NewPipeline 接口注入 |
| T4 | 记录 5 个存量问题（不改代码） | — | CDN 检测接 toolguard |
| T5 | — | 为 pasive/crt.go 补测试 | E2E 最小集脚本化 |

---

## 6. `code-health-audit.md` 基线表更新稿

> 以下为建议更新内容，**需用户批准后修改** `docs/current/code-health-audit.md`

| 指标 | 2026-05-13 | 2026-05-26 | 健康阈值 |
|------|-----------|-----------|----------|
| Go 源文件 | 172 | 175 | — |
| TSX/TS 源文件 | 52 | 56 | — |
| Go 测试文件 | 28 (16%) | **64 (37%)** | >40% |
| 前端测试文件 | 4 (8%) | 4 (7%) | >30% |
| Go 超 400 行文件 | 13 | **14**（`pipeline_tool.go` 771L 新增）| <5 |
| Go 超 800 行文件 | 1 | 1 | 0 |
| DB 迁移版本 | 20 | 20 | v0.x 略高 |
| 裸 SQL 字符串 | 182 | ~286 | — |
| api 直接 import db | 4 → 3 | 3 | 0 |
| api 直接 import models | 17 | 15 | 逼近 0 |

---

## 7. 建议在 `plan.md` 新增的 Workstream

```
| Slimdown 2026 | Active | Yes | docs/active/review/slimdown-2026-handoff-for-implementer.md |
```

同时更新：Code health audit 状态从 `Active` → `Accepted`（已被 Slimdown 覆盖）。
