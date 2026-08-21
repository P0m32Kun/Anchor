# API contract convention

- Status: normative
- Updated: 2026-08-20

`internal/api/server.go` and route-level tests define the exact HTTP surface. `frontend/src/lib/` defines the consuming client behavior. Do not maintain a parallel endpoint inventory here.

## Rules

- JSON endpoints set `Content-Type: application/json`.
- Successful responses use the shape required by the handler's public test; there is no implicit global data wrapper.
- Errors use the shared application error envelope described in [`../reference/api-error-contract.md`](../reference/api-error-contract.md).
- Validate path parameters and request bodies before starting work or writing a success status.
- Protected routes use the registered authentication middleware. SSE uses its project-bound token flow.
- Field names, nullability, units, enum values, and time formats are public contract details and change with backend, frontend, and tests together.
- Pagination and filtering defaults must be explicit in tests when added or changed.
- File and stream endpoints must choose headers before writing the body and must not leak filesystem paths or credentials.

For scan requests and run observation, see [`../reference/scan-api.md`](../reference/scan-api.md).
