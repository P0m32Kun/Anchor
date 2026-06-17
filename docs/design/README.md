---
status: active
source_of_truth: false
owner: kun
last_updated: 2026-06-17
scope: proposal-directory
---

# Design Proposals

This directory stores in-review design material for upcoming or partial changes.

## Current Contents

- `batch-scan-scheduling/` — **draft** 批量扫描调度：资产池、Stage 调度接线、nuclei tech 分桶、分页与 orphan 恢复（2026-06-17）
- `scan-engine-convergence/` — **in-review** 扫描引擎收敛：统一 ScanEngine、移除旧 pipeline（2026-06-07）
- `hw-scan-optimization/` — **draft** 硬件扫描优化设计
- `tool-registry-and-artifact-store.md` — in-review proposal for tool YAML registry, unified execution, and large stdout handling
- `tool-registry-handoff-for-implementer.md` — handoff brief for external implementer (e.g. DeepSeek); start here when delegating
- `vuln-template-redesign.md` — 漏洞模板重设计
- `custom-nuclei-template-management.md` — **stale 2026-06-17**（40+ days without update, needs decision）
- `sse-token-and-sse-auth-testing.md` — **stale 2026-06-17**（no frontmatter, added stale marker）
- `src-bounty-workstation.md` — **superseded 2026-06-17**（SRC 能力已从代码库移除）
- `bounty-watch/` — **cancelled 2026-06-17**（随 SRC 收敛一并取消）
- `spoor-integration.md` — **in_review**（候选区，不标记 active）

## Rules

- Treat files here as proposals unless they are promoted into `docs/current/`.
- Historical design material lives in `docs/archived/`.
- If a design becomes obsolete, archive it or mark it `superseded` instead of leaving its status ambiguous.
