# Changelog

## [0.0.1.0] - 2026-05-16

### Added
- Finding batch insert buffer (`FindingBuffer`) with capacity/timeout dual-trigger flush
- `BatchInsertFindings` using transaction-wrapped individual INSERTs (avoids SQLite param limit)
- Centralized shutdown manager (`util.Manager`) with LIFO cleanup order
- Pipeline-level in-memory dedup via `seenDedupKeys` (eliminates N+1 `GetFindingByDedupKey` queries)
- Deferred evidence creation after buffer flush (guarantees FK constraint safety)

### Tests
- `TestBatchInsertFindings` — batch insert with nullable FK fields
- `TestFindingBuffer_AddAndFlush` / `FlushOnCapacity` / `Dedup` / `CloseFlushes`
- `TestManager_RegisterAndShutdown` / `ShutdownIdempotent` / `HandlerError` / `Empty`

### Fixed
- Removed stale `TestIsSlowScanStage` (referenced deleted `slow_scan.go`)
