# ScopeSentry-inspired internet mapping design input

- Status: proposal
- Updated: 2026-08-20
- Authority: design input for the distributed internet platform; implementation still requires phase acceptance

## Position

Anchor is absorbing ScopeSentry's useful design ideas as part of its distributed internet attack-surface mapping direction. Capabilities are selectively reimplemented behind Anchor's asset/work, tool registry, parser, scope, and evidence seams. ScopeSentry source code, vendored modules, and unverified rule/data collections are not imported.

ScopeSentry's AGPL-3.0 and commercial terms require separate license and provenance review for every third-party artifact. Original research is preserved under [`../archived/2026-08-document-reset/docs/design/scopesentry-integration/`](../archived/2026-08-document-reset/docs/design/scopesentry-integration/).

## Adopted principles

- Distributed workers need explicit identity, health, capacity, task ownership, cancellation, and restart semantics.
- Internet asset mapping needs snapshots, diffs, source provenance, and change history, not only the latest row.
- Passive collection and low-impact policies reduce unnecessary active traffic while preserving scoped follow-up.
- Operator information architecture should move quickly from campaign status to worker state, asset evidence, changes, and findings.
- Focused rule packs improve signal only when their noise budget, update path, and license are explicit.
- External tools remain reviewed processes behind a registry and guard; no interpreted plugin runtime is required.

## Candidate slices

| Slice | Anchor seam | Required proof |
| --- | --- | --- |
| Worker/node governance | worker API, task store, scheduler | heartbeat/offline/restart integration scenarios |
| Internet asset snapshots and diff | asset models, migrations, report/UI queries | deterministic add/remove/change fixtures |
| Passive source expansion | `internal/search/`, `internal/passive/`, seed injection | provenance, scope, rate, and failure isolation |
| Takeover/sensitive/URL rule packs | parser/rules/finding pipeline | license inventory, noise budget, fixture corpus |
| Low-impact active policies | scan config, tool registry, invocation audit | exact flags and observable enforcement |
| Campaign-oriented UI | React pages, API, SSE | loading/empty/error/live-update acceptance |

## Excluded product path

ScopeSentry-inspired work must not recreate a dedicated internal-network scanning mode. Reusable network primitives remain valid only when they support authorized internet assets.

## Unresolved choices

This design input does not decide PostgreSQL versus another store, Redis versus another queue, RBAC, tenant isolation, notification integrations, or the final multi-server consistency protocol. The distributed execution contract must establish requirements and failure evidence before those choices are recorded in a new ADR.
