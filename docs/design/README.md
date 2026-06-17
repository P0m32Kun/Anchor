---
status: active
source_of_truth: false
owner: kun
last_updated: 2026-06-04
scope: proposal-directory
---

# Design Proposals

This directory stores in-review design material for upcoming or partial changes.

## Current Contents

- `batch-scan-scheduling/` — **draft** 批量扫描调度：资产池、Stage 调度接线、nuclei tech 分桶、分页与 orphan 恢复（2026-06-17）
- `custom-nuclei-template-management.md` — in-review proposal for managing custom Nuclei templates
- `src-bounty-workstation.md` — **superseded 2026-06-17**（SRC 能力已从代码库移除，见 `docs/superpowers/specs/2026-06-17-anchor-core-slimdown-design.md`）
- `bounty-watch/` — **cancelled 2026-06-17**（随 SRC 收敛一并取消）
- `tool-registry-and-artifact-store.md` — in-review proposal for tool YAML registry, unified execution, and large stdout handling
- `tool-registry-handoff-for-implementer.md` — handoff brief for external implementer (e.g. DeepSeek); start here when delegating

## Rules

- Treat files here as proposals unless they are promoted into `docs/current/`.
- Historical design material lives in `docs/archived/`.
- If a design becomes obsolete, archive it or mark it `superseded` instead of leaving its status ambiguous.
