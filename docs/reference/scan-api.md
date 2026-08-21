# Scan API reference

- Status: reference
- Updated: 2026-08-20

## Start a scan

```http
POST /projects/{id}/scan
Authorization: Bearer <token>
Content-Type: application/json
```

The supported product mode is `external`. The request may provide a nested `config` object. Configuration fields belong inside `config`; exact fields, defaults, enum values, and validation live in `internal/scanconfig/`, `internal/api/pipeline_handlers.go`, the frontend scan modal, and their tests.

The retired `internal` mode is rejected (P1-1): `POST` with `mode=internal` returns `410 INTERNAL_MODE_REMOVED` unless `?migrate=external` is supplied, in which case a saved internal-shaped config is explicitly migrated to the internet baseline. New clients must use `external`.

An accepted response identifies the run. It does not mean tools finished successfully.

## Observe a scan

- Run detail and status are exposed below `/projects/{id}/pipeline/runs/{runId}`.
- Work-item and tool-call endpoints explain what the asset-driven engine attempted.
- Project SSE provides live projections and uses the project-bound token flow.
- Cancellation has a dedicated run endpoint registered in `internal/api/server.go`.

The exact route list is `Server.Register`; do not infer it from historical docs. Stage names are UI/compatibility projections and do not define fixed execution order.

## Contract changes

A change to scan fields, defaults, route shape, status values, or SSE payload requires coordinated backend tests, frontend types/client behavior, and the relevant user-flow evidence. Follow [`../conventions/api-contracts.md`](../conventions/api-contracts.md).
