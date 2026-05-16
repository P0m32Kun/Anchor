# Changelog

## [0.0.3.0] - 2026-05-16

### Added
- Tool execution allowlist (`internal/toolguard/Allowlist`)
- Binary basename whitelist: only `subfinder`, `dnsx`, `httpx`, `naabu`, `nmap`, `nuclei`, `cdncheck`, `git`, `sh`, `bash` allowed
- Shell metacharacter rejection in all arguments (`;|&><`$(){}[]\n\r`)
- Wired into all 5 `exec.Command` call sites: `worker.go`, `server.go`, `health.go`, `cdn/detector.go`, `nuclei/custom/git.go`
- `Allowlist.Allow(name)` for runtime extension (custom tool registration)

### Tests
- `TestAllowlist_*` — allowed/rejected binaries, path traversal, shell meta detection, normal args pass-through

## [0.0.2.0] - 2026-05-16

### Added
- Resource governor (`ResourceGovernor`) with static memory/CPU thresholds
- `Acquire(ctx)` gates task execution: memory blocks via polling, CPU sleeps fixed delay
- `ResourceSampler` interface with gopsutil-backed default implementation
- Thresholds configured via `ANCHOR_GOVERNOR_*` environment variables
- Wired into `Runner.Run` (API server) and `WorkerServer.executeTask` (remote worker)

### Tests
- `TestResourceGovernor_*` — fake sampler unit tests (block/release/cancel/fail-open/env parsing)
- `TestGopsutilSamplerSmoke` / `TestRealGovernorHappyPath` — real system integration tests

### Dependencies
- Added `github.com/shirou/gopsutil/v3` for cross-platform system metrics

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
