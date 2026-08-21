# Anchor Agent Contract

## Product boundary

Anchor is a distributed internet attack-surface mapping platform for authorized adversary-emulation exercises. It orchestrates external scanners across worker capacity, enforces scope, normalizes assets and findings, tracks changes over time, exposes live progress, and supports human verification and reporting.

The current implementation uses Go, React + TypeScript, SQLite, HTTP/SSE, and a server/worker deployment. Distributed execution, persistent task ownership, asset snapshots/diffs, passive collection, and resilient worker recovery are product goals; their storage, queue, identity, and multi-server designs must be decided deliberately. Do not mistake current single-server limitations for the product boundary.

## Read order

Use this authority chain for repository work:

1. `AGENTS.md` for working rules.
2. `docs/README.md` for document authority and navigation.
3. `docs/current/product.md` for product scope.
4. `docs/current/architecture.md` for implemented architecture.
5. `docs/current/plan.md` for prioritized work.
6. `docs/decisions/` for accepted direction-setting decisions.

Proposals, references, runbooks, pitfalls, and archives never override that chain. If two live documents conflict, stop and repair the higher-level source instead of choosing whichever text is more convenient.

## Navigation

- If `.codegraph/` exists, use CodeGraph first for architecture, call paths, symbol sources, and impact analysis.
- Use `rg` or `rg --files` for exact text, paths, and file listings.
- Treat code, tests, migrations, configuration, and command help as the source of truth for implementation details. Do not rely on cached file counts, route tables, or directory trees in prose.

## Engineering contract

- Preserve authorization and scope checks around every external-tool execution path.
- Keep HTTP handlers thin; put domain behavior in focused packages or services.
- Pass `context.Context` first where an operation can block, perform I/O, or be cancelled.
- Return contextual errors; avoid panics and hidden global state in production code.
- Keep external tool units identical to the upstream CLI or API. Name units explicitly at boundaries.
- Test empty input, cancellation, concurrent access, limits, and persistence failure where relevant.

## Verification

Choose checks by changed behavior, not by habit:

- Go logic: focused package tests, then the broader relevant Go test set.
- Database behavior: temporary SQLite database with real migrations.
- React logic: focused unit tests and typecheck.
- User-visible critical flows: the relevant Docker-backed Playwright or manual functional scenario.
- Documentation: `make check-docs`.

Build or typecheck proves only compilation. Completion claims must name the commands actually run and separate observed results from configured capability. If the repository baseline is already red, report the pre-existing failures and prove that the changed path did not add new ones.

## Documentation changes

- Update only the canonical document that owns the changed fact, plus links that point to it.
- Record a direction-setting decision in `docs/decisions/`; do not bury it in a plan or proposal.
- Keep implementation inventories discoverable from code. `internal/api/server.go` owns routes and `Server` dependencies; `internal/api/README.md` explains how to navigate them but does not duplicate them.
- Put operational procedures in `docs/runbooks/`, stable facts in `docs/reference/`, conventions in `docs/conventions/`, unaccepted ideas in `docs/proposals/`, and superseded material in `docs/archived/`.
- Run `make check-docs` after changing Markdown or agent guidance.

Use Chinese for collaboration unless the user asks for another language. Follow only skills available in the current host; a missing optional skill must not block normal repository work.
