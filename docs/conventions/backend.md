# Backend convention

- Status: normative
- Updated: 2026-08-20

## Boundaries

- HTTP decoding and response writing stay in `internal/api/`.
- SQLite access and migrations stay in `internal/db/`.
- Asset-to-work behavior stays in `internal/scanengine/`; UI stage aggregation does not control execution.
- External commands pass through the tool registry/guard and a runner boundary.
- Parsers consume captured tool output without owning process lifecycle.

## Go rules

- Run `gofmt`; use `goimports` when imports change and it is available.
- Put `context.Context` first on blocking, I/O, or cancellable operations.
- Wrap errors with actionable context and preserve the cause with `%w`.
- Keep production dependencies explicit; avoid mutable package globals.
- Protect shared state with clear ownership or synchronization and test cancellation and shutdown.
- Scan nullable SQLite columns through `sql.Null*` or equivalent types.
- Keep external CLI/API units unchanged and encode the unit in names or boundary documentation.
- Treat user-controlled paths, command arguments, output, and report content as untrusted.

Verification follows [`testing.md`](testing.md). The implemented subsystem map is in [`../current/architecture.md`](../current/architecture.md).
