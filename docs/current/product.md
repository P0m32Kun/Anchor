# Product baseline

- Status: current
- Updated: 2026-08-20

## Purpose

Anchor is a distributed internet attack-surface mapping platform for authorized adversary-emulation exercises. It coordinates discovery and verification across worker capacity, turns heterogeneous tool output into a queryable internet asset graph and finding set, and makes change, evidence, and execution state visible to operators.

The core loop is:

```text
authorized internet scope → passive collection → scoped active mapping → asset graph → prioritized work → findings/evidence → change-aware report
```

The platform optimizes for internet coverage, signal, safe execution, repeatability, and operator control. It is not a generic command launcher and not a copy of another scanner's implementation.

## Users and operating model

- Primary users: red teams, blue teams, and security operators running authorized internet-facing exercises or exposure assessments.
- Typical use: organization/domain-based internet asset discovery, external exposure mapping, repeated measurement, and campaign support.
- Execution: a control plane coordinates one or more workers; workers run reviewed external tools under scope and resource controls.
- Observation: the web UI exposes campaign progress, workers, assets, changes, findings, evidence, and reports.
- Safety: authorization, internet scope, exclusions, rate limits, cancellation, and provenance are first-class behavior.
- Automation model: execution is deterministic and policy-driven. Anchor does not contain an LLM, MCP server, skill runtime, or autonomous agent planner.

## Current implementation base

- Asset-driven work derivation rather than a fixed stage pipeline.
- Domain, URL, IP, CIDR, and company-oriented target ingestion.
- Passive search, DNS/CDN/port/Web discovery, tool execution, parsers, deduplication, findings, evidence, screenshots, and reports.
- Persistent work state, server/worker HTTP paths, React UI, project-scoped SSE, and SQLite persistence.

These are observed source capabilities, not a claim that the distributed target is complete. [`plan.md`](plan.md) owns stabilization and evolution gates.

## Direction absorbed from ScopeSentry

Anchor selectively reimplements useful ideas behind its own asset/work, registry, parser, scope, and evidence seams:

- internet asset inventory, snapshots, diffs, and change history;
- worker health, task ownership, cancellation, capacity, and restart recovery;
- passive-first collection and low-impact scoped active follow-up;
- subdomain takeover, sensitive-information, URL-security, and fingerprint rule packs;
- campaign, worker, asset, change, finding, and evidence information architecture.

ScopeSentry's fixed pipeline, interpreted plugin runtime, MCP/skill integration, embedded tool forks, and implementation code are not adopted.

## Product boundaries

- Dedicated internal-network scanning is not a product scenario. The legacy `internal` preset, API value, UI choice, and specialized execution path were retired by the P1-1 compatibility exit: new `internal` scan requests receive `410 INTERNAL_MODE_REMOVED`, saved internal configurations are rejected or explicitly migrated (never silently widened), and the UI exposes the internet scan mode only.
- All internet targets require explicit authorization and scope/exclusion enforcement.
- External tools remain reviewed implementations behind a semantic execution module and registry/guard. A tool may use an in-process SDK or a guarded process adapter; callers do not depend on CLI arguments.
- The first-class exposure triad is high-risk service/port exposure, high-risk vulnerability evidence, and weak authentication. Weak authentication is discovered through Nuclei routing and approved custom template sources, not an unbounded password engine.
- For authentication checks, Anchor may run safe anonymous/default checks in a standard campaign. Checks that can lock accounts require explicit campaign authorization, protocol budgets, lockout detection, and redacted evidence. SSH and similar services exclude the official Nuclei `default-login` tag; the RBKD custom template source owns the check content and maintenance.
- RBKD templates, dictionaries, and fingerprints are external maintained inputs. Anchor records source, revision or digest, activation state, and scan provenance; it does not own their detection content.
- Third-party code, rules, templates, dictionaries, and UI assets require provenance and license review. ScopeSentry is AGPL-3.0 with commercial terms; its code is not copied or vendored.
- Target AI/LLM services may be modeled as ordinary internet assets and assessed with deterministic rules, but Anchor never calls an LLM to plan, interpret, or execute work.
- SQLite, task transport, identity, and multi-server consistency are implementation decisions under evaluation, not permanent constraints.
- RBAC, tenant isolation, notification integrations, and continuous scheduling semantics require explicit product acceptance and ADRs; they are not inferred automatically from “distributed.”

See [`../proposals/scopesentry-capabilities.md`](../proposals/scopesentry-capabilities.md) for capability design input and [`../decisions/0003-distributed-internet-asm-baseline.md`](../decisions/0003-distributed-internet-asm-baseline.md), [`../decisions/0004-deterministic-non-agent-product.md`](../decisions/0004-deterministic-non-agent-product.md), [`../decisions/0005-semantic-scan-execution-adapters.md`](../decisions/0005-semantic-scan-execution-adapters.md), [`../decisions/0006-nuclei-weak-auth-policy.md`](../decisions/0006-nuclei-weak-auth-policy.md), and [`../decisions/0007-internal-mode-exit.md`](../decisions/0007-internal-mode-exit.md) for durable direction.
