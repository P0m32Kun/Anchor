# 0002. Keep Anchor local-friendly and operator-focused

- Status: superseded by 0003
- Date: 2026-08-20

## Context

Recent product notes described Anchor both as a personal/small-team authorized-testing workbench and as a future enterprise distributed attack-surface platform. The latter implied PostgreSQL, multi-server operation, RBAC, continuous monitoring, and a broader operational burden. Those directions conflict and would lead agents to perform a platform rewrite while the existing baseline still needs stabilization.

## Decision

Anchor currently serves an individual operator or small team performing authorized, time-bounded security testing. Keep the Go and React application, SQLite persistence, HTTP/SSE model, and reviewed external-tool execution approach as the baseline.

Use other open-source projects as sources of capability and UI ideas, subject to license and provenance review. Do not treat them as architecture templates. Enterprise identity, continuous ASM, PostgreSQL, generic plugins, and multi-server consistency remain outside the current plan until a later accepted ADR changes this decision.

## Consequences

Near-term work prioritizes restoring a green, reproducible baseline and improving signal, evidence, and setup. Distributed or enterprise features require explicit product evidence, migration design, operational acceptance, and a superseding decision rather than arriving through an unrelated refactor.
