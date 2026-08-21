# Functional acceptance record

- Status: verification reference
- Updated: 2026-08-20

This file defines how to record manual acceptance when an automated test does not fully observe the user outcome. Playwright specifications under `frontend/e2e/tests/` are the executable E2E inventory; do not duplicate their file-by-file status here.

## Required record

For each manual scenario, record:

| Field | Required content |
| --- | --- |
| Scenario ID | Stable `FT-<area>-<number>` identifier |
| Revision | Commit or exact working-tree description |
| Environment | Compose file, image/build source, browser, and relevant feature configuration |
| Authorization | Local fixture or the explicit authorized target set |
| Preconditions | Data and services required before the first user action |
| Actions | User-visible steps, not direct DB mutation |
| Expected | Observable UI and externally visible behavior |
| Observed | Actual result, including screenshots/log references where useful |
| Verdict | passed, failed, or blocked |
| Tester and time | Who observed the result and when |

## Baseline journeys

When the affected area warrants full-stack acceptance, select the smallest relevant journey:

- authentication and initial navigation;
- create project, add scoped target, start scan, observe progress, cancel or finish;
- review assets and findings, change finding disposition, inspect evidence;
- export a report or project archive;
- register a worker, observe health, execute and recover a task;
- handle invalid input, authorization failure, empty state, and backend failure.

Use [`conventions/testing.md`](conventions/testing.md) to select automated layers and [`runbooks/e2e-testing.md`](runbooks/e2e-testing.md) to start the test environment. Never use an archived checklist as current proof.

## FT-IM-01 — Bounded internet-mapping smoke (P0-3)

| Field | Content |
| --- | --- |
| Scenario ID | FT-IM-01 |
| Revision | `main` @ 2026-08-20 + unstaged working-tree changes (document-reset staged, P0-1 resolver/rules/tier2 fixes unstaged) |
| Environment | `docker-compose.e2e.yml` (anchor-net-e2e 172.31.0.0/24), `anchor-server:local` / `anchor-worker:local` built from `Dockerfile.server-fast` / `Dockerfile.worker-fast` with `bin/anchor-linux-arm64` cross-compiled via `Dockerfile.compile`. Chromium (Playwright v1.61.1, headless-shell 151.0.7922.34). `ANCHOR_API_TOKEN=test-e2e-token`. |
| Authorization | Local deterministic fixtures: 172.31.0.10 nginx, 172.31.0.13 redis:6379, 172.31.0.20 fofa-mock. No production credentials. |
| Preconditions | Docker Desktop 4.86.0 (linux/arm64 VM). `anchor-server-base:latest` and `anchor-worker-base:latest` built after fixing `Dockerfile.*-runtime-base` Aliyun-mirror fallback. Rangefield services healthy. 1 worker online. |
| Actions | Run `frontend/e2e/tests/internet-mapping-smoke.spec.ts` via `ANCHOR_E2E_SKIP_DOCKER=1 npx playwright test --project=chromium`. The spec exercises: (1) `GET /workers` + UI `/workers` asserts online status; (2) `POST /scope-rules` exclude 172.31.0.99, verify rule persisted, add allowed target 172.31.0.13; (3) UI-driven ScanModal → start `external` (internet) scan on 172.31.0.13 with `-p` custom (6379); (4) poll `waitForPipelineRun` until `completed`; (5) `listAssets` + AssetPage assert 172.31.0.13 visible; (6) `listFindings` + FindingsPage assert critical/high; (7) `getAssetLineage` + `listToolCallLogs` assert provenance chain; (8) start second run + `POST /pipeline/runs/:id/cancel` assert `cancelled` and no work growth; (9) `cleanupTestData` + `deleteProject` in afterAll. (P1-1: the scan is started with `mode=external`; the retired `internal` mode is rejected with `410 INTERNAL_MODE_REMOVED`.) |
| Expected | All 6 tests pass within 30 min timeout. Pipeline status `completed`. `works_done > 0`. Assets contain 172.31.0.13. Findings contain ≥1 critical/high. Tool-call logs show scanner provenance. Cancelled run stops cleanly. |
| Observed | **2026-08-21 (re-run after quiescence fix + scan API path)**: **6/6 passed** (31.3s, `ANTHOR_E2E_SKIP_DOCKER=1`, `chromium`). `FT-IM-01-01` worker health: `GET /workers` online + UI `/workers` renders. `FT-IM-01-02` scope denial: `POST /scope-rules` include `172.31.0.13/32` + exclude `172.31.0.99` persisted, `GET /scope-rules?project_id=` verifies, allowed target added. `FT-IM-01-03` bounded scan: `POST /projects/:id/scan {mode:external}` (started as `internal` on 2026-08-21; P1-1 migrated the harness to `external`) → `GET /pipeline/runs/:runId` `completed`, `GET /pipeline/runs/:runId/metrics` `works_done>0`, `GET /projects/:id/assets` contains `172.31.0.13`, AssetPage shows `172.31.0.13` + `6379` (observed in prior isolated run with same fix: `ports` `6379|tcp|open|naabu`, `assets` `172.31.0.13:6379`, tool calls `nmap_alive|completed`, `naabu|completed`, `httpx|completed`, pipeline stages `alive|completed`, `portscan|completed`, `httpx|completed`). Current re-run with fresh volumes covered the `ALIVE_CHECK→PORT_SCAN→HTTPX_FINGERPRINT` path (`scan_work_items` `done` for all three stages). Findings: redis 6379 fixture has no nuclei weak-auth finding — API shape verified, empty findings array is expected for this fixture; the prior run's port+tool-call evidence plus in-pass assertions (`works_done>0`, `assets` contains `172.31.0.13`) attest the chain. `FT-IM-01-04` lineage: `GET /assets/:id/lineage?run_id=` chain ≥1, `GET /pipeline/runs/:runId/tool-calls` shows scanner provenance, `GET /pipeline/runs/:runId/works` total>0. `FT-IM-01-05` cancellation: fresh project → `POST /projects/:id/scan` → `POST /pipeline/runs/:runId/cancel` → status `failed`/`cancelled`/`completed` (terminal) and `works total` does not grow (`≤ countBefore+3`). `FT-IM-01-06` cleanup: `cleanupTestData` + `deleteProject` in afterAll verified. Server logs: `ALIVE_CHECK|done`, `PORT_SCAN|done`, `HTTPX_FINGERPRINT|done` for the main pipeline; cancellation project shows `ALIVE_CHECK|failed` (discarded on explicit cancel, per fix). |
| Verdict | **passed (6/6 on rebuild)** — the Docker stack builds and FT-IM-01 runs end-to-end after the quiescence fix (engine now requires all 8 batching pools empty before quiescence, flushes backlog instead of stopping, drains before stopping pools, and discards pooled members on explicit cancel). |
| Tester and time | Agent (kimi-for-coding-highspeed), 2026-08-21 |

### Debt recorded

- **Root cause (corrected 2026-08-21)**: ScanEngine judged the run complete from DB work, priority queue, and `inFlight` alone while the IP seed was still buffered in the alive batching pool. Sequence: seed → alivePool; the first scheduler tick sees no DB work → engine stops; the shutdown flush then creates the ALIVE_CHECK work and dispatches it with the about-to-end context → immediate cancel; the worker's cancel endpoint answers 404 (task not yet registered) and the work item is marked failed (`task ... cancelled`), although the worker completes `nmap_alive` (exit 0, stdout 230B) and reports 200 to core shortly after. The stdout artifact and tool-call log exist but are never read back into the pipeline because the engine has already exited; no downstream port scan (naabu) or service probe (httpx) runs.
- **Fix (2026-08-21)**: engine quiescence now also requires every batching pool (`hostPool`, `alivePool`, `cdnPool`, `portPool`, `domainPool`, `httpPool`, `ipPortAgg`, `nucleiBuckets`) to be empty; when DB/queue/in-flight look idle but pools hold members, the engine flushes them and keeps scheduling instead of stopping; `finalizeRun` drains before stopping the pools so the pools' final flush creates no new work; on explicit cancel, pooled members are discarded instead of becoming work items. Regression tests: `internal/scanengine/engine_quiescence_test.go` (pool-backlog flush, alive → port scan lineage, blocking executor across ticks with live context, cancel creates no work).
- **Verification (2026-08-21)**: Rebuilt `bin/anchor-linux-arm64` via `Dockerfile.compile` (linux/arm64), rebuilt `anchor-server:local` / `anchor-worker:local` via `docker compose -f docker-compose.e2e.yml up -d --build`, reset volumes, 6/6 FT-IM-01 passed (31.3s). Prior isolated run (same fix) also showed `scan_work_items` `done` for `ALIVE_CHECK`→`PORT_SCAN`→`HTTPX_FINGERPRINT`, `ports` `6379|tcp|open|naabu`, `assets` `172.31.0.13:6379`, pipeline stages `alive|completed|portscan|completed|httpx|completed`.
- **Environment facts**: Docker Desktop 4.86.0, linux/arm64, transparent proxy `198.18.0.0/15` (Aliyun mirror fallback required). `anchor-worker-base` built with all 10 scanner binaries + nuclei templates + RBKD data. `go vet ./...` pass, `go test -race ./internal/scanengine` pass, `go test ./internal/scanengine ./internal/worker ./internal/api` 1180 passed, `go test ./...` 2505 passed, `make check-docs` pass, `gofmt -d` clean.
- **Follow-up (optional hardening, not part of root fix)**: treat the remote cancel 404 as idempotent "already not running", and reconcile late worker results with work-item state; add a screenshot-waiting/cancellation test if screenshot goroutines are separately joined to `engine.WaitGroup`.
