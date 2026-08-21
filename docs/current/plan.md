# Current plan

- Status: current
- Updated: 2026-08-20
- Theme: restore a trustworthy base, then evolve Anchor into a deterministic distributed internet attack-surface mapping platform

This is the only active repository-wide plan. Each ticket below is a bounded implementation entry point; its executor still enters the normal change-gate workflow. No ticket is complete from compilation alone: the listed acceptance evidence is required.

## Baseline evidence at restart

The working tree contains substantial uncommitted user changes; they are preserved. The following read-only checks were observed before this plan was accepted:

- `go vet ./...`: passed.
- `go test ./...`: 2492 passed, 9 failed, 10 skipped. Failures cover CIDR asset schema/model agreement, resolver invalid-domain behavior, and liveness/port-scan eligibility semantics.
- `make check-docs`: passed.
- Frontend `tsc --noEmit`: passed.
- Frontend unit tests: could not start because the installed Vitest 4.1.9 package expects a Vite export absent from the installed Vite 5.0.0 package.

These are restart evidence, not completion claims. P0 exits only after the full checks are green or an unavailable environment is explicitly recorded as acceptance debt.

## Phase 0: restore a trustworthy base

### P0-1 — Go data and eligibility baseline

- Goal: make CIDR persistence, DNS resolver error semantics, liveness, and port-scan eligibility agree across schema, models, code, and fixtures.
- Owned paths: `internal/db/`, `internal/models/`, `internal/resolve/`, `internal/scanengine/core/`, `internal/scanengine/`, and their focused tests.
- Entry condition: current nine Go test failures are reproduced on the ticket revision.
- Acceptance: temporary SQLite migrations accept the supported internet asset types; invalid resolver inputs fail deterministically without public-DNS dependence; empty, cancelled, and mixed-validity cases are covered; CDN/HTTPX eligibility tests pass.
- Verification: focused package tests, `go test ./...`, then `go vet ./...`.
- Unlocks: P0-3 and all feature tickets.

### P0-2 — Deterministic frontend toolchain

- Goal: select one package manager and one lockfile, remove the Vitest/Vite incompatibility, and make install, typecheck, unit tests, and build reproducible.
- Owned paths: `frontend/package.json`, the selected `frontend/*lock.yaml` or `frontend/*lock.json`, `frontend/pnpm-workspace.yaml` when applicable, `.github/workflows/`, `Makefile`, and `scripts/pre-merge-check.sh`.
- Entry condition: P0-1 is green and the existing frontend dependency state is captured.
- Acceptance: clean dependency installation from the selected lockfile; `typecheck`, unit tests, and production build all run in CI and locally using the same command.
- Verification: frozen install, `npm run typecheck`, `npm run test:unit`, and `npm run build` (or the equivalent selected package-manager commands).
- Unlocks: P0-3 and UI tickets.

### P0-3 — Named internet-mapping smoke evidence

- Goal: prove one bounded server/worker internet-mapping campaign with scope denial, worker health, cancellation, asset output, finding/evidence lineage, and cleanup.
- Owned paths: `docker-compose.e2e*.yml`, `frontend/e2e/`, `docs/functional-test.md`, and the smallest required test fixtures.
- Entry condition: P0-1 and P0-2 pass.
- Acceptance: the named Docker-backed scenario runs against controlled fixtures; if Docker or a required fixture is unavailable, the exact debt and environment are recorded without claiming success.
- Verification: the named Playwright/E2E smoke command plus artifact and database inspection.
- Unlocks: Phase 1 and Phase 2.

## Phase 1: retire the dedicated internal scenario

### P1-1 — Explicit internal-mode compatibility exit

- Goal: reject new `internal` requests, provide explicit handling for saved configurations, remove the dedicated UI choice and specialized branches, and retain only reusable internet primitives.
- Owned paths: `internal/api/`, `internal/models/`, `internal/db/`, `internal/scanengine/`, `frontend/src/`, `frontend/e2e/`, migrations, and compatibility tests.
- Entry condition: P0-3 is accepted and all legacy `internal` references are inventoried.
- Acceptance: new requests receive a documented compatibility error; saved configurations are rejected or explicitly migrated without silently widening scope; UI, API, migration, unit, and E2E behavior agree.
- Verification: focused API/database/frontend tests, migration test with an old configuration, and the named E2E path.
- Unlocks: new internet-only capability work.

## Phase 2: semantic execution and distributed contract

### P2-1 — Deep semantic execution module

- Goal: replace the CLI-shaped `RenderParams` seam with a small `ScanRequest` interface containing action, normalized targets, policy, budgets, template/rule selection, and idempotency context.
- Owned paths: new `internal/scanner/` or equivalent execution module, `internal/scanengine/executor/`, `internal/toolrun/`, `internal/toolregistry/`, `internal/models/`, and focused tests.
- Entry condition: P0-1 and P1-1 are complete.
- Acceptance: the scan engine submits semantic requests; process execution remains behaviorally compatible; parsing, redaction, invocation provenance, cancellation, and scope checks are hidden behind the module interface.
- Verification: adapter contract fixtures, cancellation and empty-input tests, process-adapter parity tests, and focused Go tests.
- Unlocks: P2-2 and P2-3.

### P2-2 — Naabu SDK adapter pilot

- Goal: add a Naabu SDK adapter while retaining the process adapter as a fallback selected by worker capability and policy.
- Owned paths: `internal/scanner/`, `internal/scanengine/executor/`, `internal/worker/`, `go.mod`, `go.sum`, tool capability metadata, and adapter tests.
- Entry condition: P2-1 contract is accepted.
- Acceptance: SDK and process adapters produce equivalent normalized ports/services on the same fixtures; context cancellation, rate limits, resource accounting, IPv4/IPv6 behavior, privilege requirements, and worker restart behavior are observable.
- Verification: local adapter parity suite, controlled network fixtures, race test, and worker integration test. No promotion based on compile success alone.
- Unlocks: Subfinder SDK evaluation and semantic remote requests.

### P2-3 — Semantic worker protocol and leases

- Goal: make worker tasks carry semantic requests and establish durable ownership, lease expiry, idempotent result submission, retry, cancellation, drain, and offline recovery.
- Owned paths: `internal/api/worker_handlers.go`, `internal/api/server.go`, `internal/worker/remote_client.go`, `internal/worker/dispatcher.go`, `internal/db/queries_scan.go`, `internal/scanengine/work/`, models, and new migrations.
- Entry condition: P2-1 is accepted; P2-2 supplies one production adapter and one fallback adapter.
- Acceptance: a task can be claimed once, renewed, safely retried after lease expiry, submitted more than once without duplicate findings, cancelled, drained, and recovered after worker or server restart.
- Verification: temporary SQLite integration tests, multi-worker Docker scenario, fault injection for lost heartbeat/result replay, and read-only review of scope enforcement.
- Unlocks: horizontal worker scaling and Phase 3 capability packs.

## Phase 3: internet mapping depth and two-high-one-weak coverage

### P3-1 — Nuclei and RBKD policy enforcement

- Goal: route weak-auth checks through Nuclei tags and approved RBKD templates; exclude official `default-login` for high lockout-risk services such as SSH; make template inputs versioned and provenance-bearing.
- Owned paths: `internal/nuclei/`, `internal/scanengine/`, `internal/worker/commands.go`, `tools/nuclei.yaml`, `internal/builtin/`, `internal/models/nuclei_custom.go`, `internal/db/queries_nuclei.go`, worker Dockerfiles, and related UI/tests.
- Entry condition: P2-3 semantic request and lease contract are accepted.
- Acceptance: standard campaigns run safe anonymous/no-auth checks; protected services never receive excluded tags; RBKD revision/digest and activation state are stored with task provenance; account-locking checks require explicit policy and stop on lockout signals; plaintext credentials never enter logs or findings.
- Verification: tag-routing fixtures, policy denial tests, bundle revision tests, Docker worker smoke, and redaction tests.
- Unlocks: complete two-high-one-weak coverage claim.

### P3-2 — High-risk vulnerability prioritization

- Goal: combine high-risk service exposure, Nuclei evidence, asset criticality, confidence, and a versioned KEV input into deterministic work and reporting priority.
- Owned paths: `internal/finding/`, `internal/models/finding.go`, database queries/migrations, scan profiles, and report/UI projections.
- Entry condition: P3-1 provides stable template provenance and findings.
- Acceptance: priority is reproducible from stored inputs; KEV refresh failure is visible; high-risk findings retain evidence lineage and do not bypass scope or deduplication.
- Verification: fixture corpus, persistence round trips, ranking property tests, and report/API acceptance.
- Unlocks: operator triage views and scheduled remeasurement.

### P3-3 — Modern passive/API/cloud asset expansion

- Goal: add high-signal 2026 internet techniques behind existing asset/work seams: passive source provenance, cloud/ASN discovery, IPv6-aware probing, OpenAPI/GraphQL/API documentation discovery, JavaScript endpoint/secret extraction, WebSocket/gRPC evidence, takeover, URL-security, and fingerprint rule packs.
- Owned paths: `internal/search/`, `internal/passive/`, `internal/scanengine/seed/`, `internal/parser/`, `tools/`, asset models/migrations, and focused fixtures.
- Entry condition: P2-3 and P3-2 are accepted.
- Acceptance: every source has scope, rate, provenance, license, noise budget, failure isolation, and normalized output; active follow-up is policy-gated and visible in invocation records.
- Verification: deterministic source fixtures, scope-denial tests, parser corpus, persistence tests, and one user-visible campaign acceptance.
- Unlocks: Phase 4 information architecture.

## Phase 4: operator information architecture

### P4-1 — Campaign, worker, asset, change, finding, and evidence views

- Goal: present the ScopeSentry-inspired information architecture through Anchor's React/API/SSE seams without reintroducing a fixed pipeline.
- Owned paths: `frontend/src/pages/`, `frontend/src/components/`, `frontend/src/lib/api.ts`, server read models and routes, and Playwright fixtures.
- Entry condition: P2-3 and the first P3 capability are accepted.
- Acceptance: loading, empty, error, authorization, cancellation, live-update, long-running, and coverage-gap states are visible; pages link findings to assets, changes, evidence, worker, task, and tool provenance.
- Verification: focused unit tests, typecheck/build, and Docker-backed Playwright scenarios.
- Unlocks: user acceptance of the campaign workflow.

## Phase 5: measured scale and resilience

### P5-1 — Storage, queue, identity, and multi-server promotion

- Goal: choose and implement durable infrastructure only after lease/load/failure evidence justifies it.
- Owned paths: `internal/db/`, task transport adapters, deployment manifests, migrations, observability, and recovery tests.
- Entry condition: P2-3, P3-3, and measured load/failure data are accepted.
- Acceptance: horizontal workers, backpressure, metrics, migration/rollback, identity scope, and multi-server consistency are verified in a disposable environment.
- Verification: staged load test, restart/failure matrix, migration rehearsal, security review, and human operational acceptance.
- Unlocks: production-grade distributed claim.

## Scheduling rules

- Do not reintroduce a dedicated internal-network product scenario.
- Do not add a fixed pipeline where asset/work derivation can express the behavior.
- Do not add an interpreted plugin runtime, MCP/skill runtime, LLM integration, or autonomous agent planner.
- Do not import ScopeSentry source code or data without separate license approval.
- Do not label target architecture as implemented without observed code, focused tests, and failure/restart evidence.
- Do not enable high lockout-risk authentication templates through a generic default tag; route them through the approved custom-source policy.
- Do not promote an SDK adapter until it proves parity, cancellation, budgets, resource behavior, provenance, and recovery against the process adapter.
- Each phase or cross-layer increment requires its own change card and, for direction changes, an ADR.

Detailed design input is in [`../proposals/scopesentry-capabilities.md`](../proposals/scopesentry-capabilities.md); durable direction is in [`../decisions/0003-distributed-internet-asm-baseline.md`](../decisions/0003-distributed-internet-asm-baseline.md), [`../decisions/0004-deterministic-non-agent-product.md`](../decisions/0004-deterministic-non-agent-product.md), [`../decisions/0005-semantic-scan-execution-adapters.md`](../decisions/0005-semantic-scan-execution-adapters.md), and [`../decisions/0006-nuclei-weak-auth-policy.md`](../decisions/0006-nuclei-weak-auth-policy.md).
