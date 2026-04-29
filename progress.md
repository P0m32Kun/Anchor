# Progress

## Status
Active Development — v0.2 Phase 1 Complete

## Recently Completed
- [x] Docker containerization (server + worker split, Dockerfiles, compose)
- [x] Remote worker architecture (outbound-only HTTP, long-polling task pull)
- [x] Ghost worker cleanup (heartbeat timeout goroutine, 120s stale threshold)
- [x] WorkersPage API integration (real-time fetch + 5s polling)
- [x] Docker Compose networking fix (worker → server via service DNS name)
- [x] Scope confirmation flow (auto-suggest scope rules on first target import)
- [x] Target import expansion (comma-separated, IP hyphen ranges, auto type-detect)
- [x] CIDR expansion in Scope Check (subnet include/exclude support)

## In Progress
- [ ] v0.2 Phase 2: Nuclei template management via server

## Upcoming
- [ ] v0.2 Phase 2: Server-managed nuclei template distribution to workers
- [ ] v0.2 Phase 3: Worker capability reporting + multi-worker task routing

## Known Issues
- Worker DNS resolution intermittently fails on Docker Desktop (macOS) but recovers after restart
- No graceful shutdown for `cleanupStaleWorkers` goroutine (non-critical for production)

## Files Changed (Latest Sprint)
- `docker-compose.yml` — server/worker split with anchor-net bridge
- `Dockerfile.server` / `Dockerfile.worker` — multi-stage builds
- `internal/api/handlers.go` — worker registration, heartbeat, poll, cleanup
- `internal/api/worker_handlers.go` — remote worker lifecycle
- `internal/db/queries.go` — worker_nodes CRUD
- `frontend/src/pages/WorkersPage.tsx` — remote worker list UI
- `Makefile` — up-all / down-all / range-up / range-down commands
