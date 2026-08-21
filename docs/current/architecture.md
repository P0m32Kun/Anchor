# Architecture baseline

- Status: current
- Updated: 2026-08-20
- Basis: current source tree plus the internet-focused distributed target recorded in ADR 0003

This document separates observed implementation from target architecture. Code, tests, migrations, and configuration win for current behavior. Target sections describe approved direction and require implementation evidence before being presented as complete.

## Current implementation

Anchor is a Go application with a React frontend and SQLite persistence. The Go binary serves the API and coordinates local or remote worker execution. Nginx and Docker Compose provide packaged deployment.

```text
React UI
   │ HTTP + project-scoped SSE
   ▼
Go API ───── SQLite (WAL)
   │
   ├── scan engine: assets → eligible work → queues/pools → completion
   ├── domain services: scope, findings, reports, screenshots, watch
   └── tool execution: local runner or registered remote worker
                         │
                         ▼
              reviewed external scanner binaries
```

Current source-owned seams:

| Concern | Source of truth |
| --- | --- |
| Process startup and role selection | `main.go` |
| Routes, middleware wiring, and API dependencies | `internal/api/server.go` |
| SQLite opening and migrations | `internal/db/` |
| Asset and work types and eligibility | `internal/scanengine/core/` |
| Engine lifecycle and completion handling | `internal/scanengine/engine.go` and `engine_tier*.go` |
| Persistent work state | `internal/scanengine/work/` and DB queries |
| Queueing, fairness, pooling, and limits | `internal/scanengine/queue/`, `pool/`, and `scheduler/` |
| Tool definitions and execution policy | `tools/`, `internal/toolregistry/`, and `internal/toolguard/` |
| Tool output parsing | `internal/parser/` and `internal/scanengine/executor/` |
| Frontend routes and API client behavior | `frontend/src/` |

Do not maintain route counts, handler dependency tables, migration counts, or package trees in prose. Query the sources above.

## Current scan execution

1. The API validates a scan request, project targets, configuration, and scope.
2. Seeds enter `ScanEngine` as normalized discovery assets.
3. `core.DeriveEligibleWorks` derives actions from asset type, attributes, depth, and profile.
4. Deduplication and the persistent work store prevent duplicate work for a run.
5. Queues, pools, throttles, and concurrency limits decide when eligible work runs.
6. The local runner or a remote worker invokes a tool through the registry and guard.
7. Parsers turn output into assets, ports, services, endpoints, findings, evidence, and relations.
8. New or enriched assets are reconsidered until the engine reaches quiescence or cancellation.
9. Stage records and SSE are observation projections; they do not define fixed execution order.

Liveness and port-scan eligibility are under active stabilization. Derive exact semantics from `internal/scanengine/core/` and focused tests until that work is green.

## Legacy internal mode

The source tree still contains an `internal` scan preset and related UI, configuration, rules, and tests. It is an observed compatibility path, not part of the product target. Do not add features to it. Removal requires a separate change covering API compatibility, saved configuration, frontend behavior, tests, and migration/rejection semantics.

## Distributed internet target

The approved target is an internet mapping platform with explicit control-plane and execution-plane responsibilities:

```text
Web UI / API clients
          │ campaigns, internet assets, changes, findings, workers
          ▼
Control plane: scope + campaign state + work ownership + event projection
          │ durable task protocol and worker health
          ├───────────────┬────────────────┬───────────────
          ▼               ▼                ▼
      Worker A         Worker B          Worker N
  tool pools/guard  tool pools/guard  tool pools/guard
          │               │                │
          └──── asset, evidence, finding, and telemetry results ────┘
```

Target capabilities are:

- durable ownership and idempotent handoff of internet mapping work;
- worker registration, heartbeat, capacity, draining, cancellation, and restart recovery;
- organization/domain-driven passive collection with scoped DNS, CDN, port, service, HTTP, URL, fingerprint, and vulnerability follow-up;
- asset snapshots and diffs that explain additions, removals, and material changes;
- campaign-level progress and operator views independent of a fixed pipeline;
- backpressure, rate budgets, per-target low-impact policy, and evidence retention;
- tool-call, failure, retry, and provenance observability across workers.

### Semantic execution module

The target execution seam is a semantic request, not a command string:

```text
eligible work → ScanRequest → execution module → normalized result/evidence
                                  ├── SDK adapter
                                  └── guarded process adapter
```

`ScanRequest` carries the action, normalized targets, authorization context, rate and resource budgets, policy tier, template or rule selection, and an idempotency key. The execution module owns argument rendering, SDK option mapping, output parsing, cancellation, invocation provenance, and redaction. The scan engine and control plane do not learn a tool's CLI flags or SDK types.

The first SDK adapter is a Naabu pilot. It must prove result parity with the current process adapter, context cancellation, IPv4/IPv6 behavior, rate enforcement, resource accounting, and worker recovery before another adapter is promoted. Subfinder may follow. Nuclei remains an isolated process adapter initially because its SDK is actively changing and its template runtimes have a larger host attack surface. Nmap, FFUF, Katana, and Spoor remain process adapters until an SDK provides equivalent behavior and isolation evidence.

Remote workers receive semantic requests and return normalized result batches plus an execution receipt. The control plane never asks a worker to execute an arbitrary command assembled outside the registry. Workers repeat scope, exclusion, policy, and budget checks at the final execution point.

### Weak-authentication policy

The two-high-one-weak coverage contract treats high-risk service exposure, high-risk vulnerability evidence, and weak authentication as peer outputs. Weak-authentication discovery is routed through Nuclei tags and approved custom sources. Official `default-login` checks are excluded for services where attempts can lock accounts, including SSH; the RBKD custom template source supplies the approved replacement checks.

Standard campaigns may run anonymous, no-auth, empty-credential, and other explicitly safe checks. Account-locking checks require campaign authorization, protocol-specific attempt and target budgets, lockout/429/423 detection, cancellation, and redacted evidence. Successful credentials are never persisted as plaintext and are not reused for follow-on actions. Template source revision, digest, activation state, and task provenance are retained even though RBKD owns template content and maintenance.

The target does not yet choose PostgreSQL, Redis, another queue, a new identity model, or a multi-server consistency protocol. Those remain separate decisions backed by migration and failure evidence.

## Security invariants

- External execution passes authorization, internet scope, and global exclusions at the last practical boundary.
- Semantic requests pass registry, policy, and guard checks; process adapters never assemble commands through a shell, and SDK adapters receive only validated typed options.
- Worker registration, task polling, result submission, and SSE use authenticated, appropriately scoped credentials.
- Credentials and bearer tokens never appear in logs or diagnostic output.
- Raw artifacts, screenshots, reports, and asset snapshots stay rooted in approved data directories and safe filesystem helpers.
- Distributed retries are idempotent and cannot bypass scope or create uncontrolled duplicate findings.
- Nuclei templates and RBKD bundles are treated as versioned, provenance-bearing inputs. Unsigned or unapproved high-risk template content cannot be enabled by a worker.

## Promotion rule

A target capability becomes an implemented baseline only after code, focused tests, failure/restart evidence, and the owning current document agree. ScopeSentry research is design input, not runtime behavior.
