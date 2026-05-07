---
status: active
source_of_truth: true
owner: kun
last_updated: 2026-05-04
scope: decision-index
---

# Current Decisions

There is no curated current ADR set yet.

## Temporary Rule

- Treat [`../architecture.md`](../architecture.md) as the binding repository-level decision summary.
- Treat files under `docs/archived/` and `docs/archived/*/decisions/` as historical context unless a task explicitly says to revive or migrate one of them.

## When To Add A Current ADR

Create a file in this directory when a decision:

- changes the baseline architecture,
- affects multiple subsystems,
- needs a stable rationale that should outlive one task or one branch.
