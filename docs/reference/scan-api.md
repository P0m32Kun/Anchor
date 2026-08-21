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

The current source may still accept `internal`. It is a legacy compatibility value scheduled for removal, not a product mode for new clients. Do not add fields, UI, documentation, or tests that expand it. Its final rejection or migration behavior requires the dedicated compatibility change listed in the current plan.

An accepted response identifies the run. It does not mean tools finished successfully.

## Observe a scan

- Run detail and status are exposed below `/projects/{id}/pipeline/runs/{runId}`.
- Work-item and tool-call endpoints explain what the asset-driven engine attempted.
- Project SSE provides live projections and uses the project-bound token flow.
- Cancellation has a dedicated run endpoint registered in `internal/api/server.go`.

The exact route list is `Server.Register`; do not infer it from historical docs. Stage names are UI/compatibility projections and do not define fixed execution order.

## Contract changes

A change to scan fields, defaults, route shape, status values, or SSE payload requires coordinated backend tests, frontend types/client behavior, and the relevant user-flow evidence. Follow [`../conventions/api-contracts.md`](../conventions/api-contracts.md).
