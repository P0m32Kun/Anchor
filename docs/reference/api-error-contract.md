# API error contract

- Status: reference
- Updated: 2026-08-20

JSON API failures use the shared application error shape:

```json
{
  "error": {
    "code": "BAD_REQUEST",
    "message": "human-readable summary",
    "detail": "optional diagnostic detail"
  }
}
```

`internal/errors/` owns error codes and values. The shared response helper in `internal/api/handlers.go` owns serialization. Handler tests own status-code mappings for each route.

## Rules

- `code` is stable and machine-readable; changing it is a public contract change.
- `message` is safe for users and contains no credential or sensitive raw tool output.
- `detail` is optional and must also be safe to expose.
- JSON errors set `Content-Type: application/json`, including failures from endpoints whose success body is a file or stream when headers have not yet been committed.
- Worker-only file transport may have a different transport error, but the server API must translate it before exposing it to the frontend.

Do not copy a static error-code table into documentation. Query `internal/errors/` and its consumers for the current set.
