# 0003. Make distributed internet attack-surface mapping the product baseline

- Status: accepted
- Date: 2026-08-20

## Context

The previous documentation reset incorrectly narrowed Anchor to a local-friendly tool, while the actual direction is a distributed platform for internet attack-surface mapping during authorized adversary-emulation exercises. The project is actively absorbing useful ScopeSentry design ideas. A further clarification removed the dedicated internal-network scenario because another tool now owns that use case.

The repository already contains an asset-driven scan engine, remote-worker API paths, passive search integrations, pooling, resource governance, findings/evidence, and a web observation surface. It is therefore an internet mapping platform base with distributed gaps, not a local tool whose current storage and process topology define the final product.

## Decision

Anchor's product baseline is distributed internet attack-surface mapping for authorized adversary-emulation exercises. The target includes coordinated workers, durable work ownership, internet asset snapshots and diffs, passive-first collection, low-impact active follow-up, campaign/worker/asset/finding observability, and resilient recovery.

Dedicated internal-network scanning is outside the product. Existing `internal` presets and specialized paths are legacy behavior to retire through a separately approved compatibility-aware change. Reusable IP, CIDR, liveness, port, and service primitives remain when they support authorized internet mapping.

ScopeSentry contributes design input only: information architecture, node/task governance, asset change views, passive mapping breadth, focused rule packs, and low-impact execution ideas may be selectively reimplemented behind Anchor's seams. Its code, vendored modules, and unverified data are not copied. License and provenance review is mandatory.

The current SQLite, HTTP/SSE, server/worker implementation is the starting point. PostgreSQL, Redis or another queue, RBAC, tenant isolation, notification integrations, and multi-server consistency remain separate decisions until requirements, failure modes, migration strategy, and acceptance evidence justify them.

## Consequences

The roadmap must retire the dedicated internal scenario, establish a distributed execution contract, deepen internet mapping, and improve operator information architecture. Documentation must distinguish observed implementation from target architecture. Future agents must neither extend the legacy internal mode nor introduce distributed infrastructure solely by assumption.
