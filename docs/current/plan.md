---
status: active
source_of_truth: true
owner: kun
last_updated: 2026-05-07
scope: repository-wide
---

# Current Plan

This is the only repository-wide implementation plan that agents should treat as current by default.

## Current Baseline

- The last clearly completed milestone is `v0.4` ("smart scan pipeline + multi-target-type + Nuclei layered scanning").
- v0.4 acceptance criteria and test mapping live in [`../active/review/v0.4-acceptance.md`](../active/review/v0.4-acceptance.md).
- If a document conflicts with this file, treat this file as the planning baseline unless the task explicitly says otherwise.

## Active Workstreams

| Workstream | Status | Source of truth? | Notes |
| --- | --- | --- | --- |
| Documentation governance | Active | Yes | Consolidate navigation, mark document lifecycle, reduce agent confusion |
| Current architecture baseline | Active | Yes | Defined in [`architecture.md`](architecture.md) |
| v0.4 scan pipeline | Accepted | Yes | Verified by `docs/active/review/v0.4-acceptance.md`; design doc archived at `docs/archived/v0.4/scan-pipeline.md` (superseded 2026-05-12, see `architecture.md` for current baseline) |
| Database migration notes for v0.4 | Accepted | Yes | Tied to v0.4; superseded plans now archived |
| Large-scale refactoring | Backlog | No | See [`../refactoring-plan.md`](../refactoring-plan.md) as an idea pool, not an approved release plan |

## Rules For New Work

1. Promote only one repository-wide plan at a time.
2. Keep proposal documents in `docs/design/` until they are accepted and verified.
3. Move completed milestone plans to `docs/archived/`.
4. If a task needs a short-lived task plan, keep it scoped to that task and link back here instead of creating another competing top-level roadmap.

## Exit Criteria For Promoting A Proposal

A proposal can replace this plan only when all of the following are true:

- its scope is approved,
- its implementation status is clear,
- the related E2E acceptance path is defined,
- superseded plans are explicitly marked.
