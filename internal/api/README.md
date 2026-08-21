# API package guide

This document explains where to inspect API behavior. It intentionally does not duplicate route or `Server` field inventories.

## Sources of truth

- `server.go`: `Server` dependencies, construction, middleware wiring, and every registered route.
- `middleware.go`: API and SSE authentication behavior.
- `*_handlers.go`: request decoding, response mapping, and delegation by domain.
- `handlers.go`: shared response helpers and core project/target handlers.
- `internal/errors/`: structured application error types.
- `*_test.go`: executable route, authorization, and response contracts.

Use CodeGraph for call paths and impact analysis when `.codegraph/` is present. Use `rg 'mux.Handle' internal/api/server.go` for the exact route set and `rg 's\.[A-Za-z]+' internal/api` for dependency consumers.

## Change rules

- Keep handlers focused on HTTP concerns; move reusable behavior into the owning domain package.
- Register a route and its authentication middleware in `Server.Register` in the same change as its handler.
- Add focused tests for method, path parameters, authorization, invalid input, domain failure, and success behavior that changed.
- Use shared JSON/error helpers. Streaming and file responses must still return the documented JSON error envelope before writing a successful body.
- Never log bearer tokens, SSE tokens, credentials, raw authorization headers, or secret configuration values.
- Update [`../../docs/reference/scan-api.md`](../../docs/reference/scan-api.md) only when the public scan contract changes. Update [`../../docs/reference/api-error-contract.md`](../../docs/reference/api-error-contract.md) only when the shared error contract changes.

The route registration and tests are the review checklist. A second hand-maintained route table would become a stale cache.
