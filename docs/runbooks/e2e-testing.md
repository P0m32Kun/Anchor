# E2E testing runbook

- Status: operational
- Updated: 2026-08-20

The E2E stack is disposable and separate from packaged deployment. It must not use production credentials or user databases.

## Commands

The `Makefile` is the command source of truth:

```bash
make test-e2e-smoke
make test-e2e
make test-e2e-scan
make test-e2e-full
make test-e2e-local-down
```

Use the smallest target covering the changed behavior. Long scanner scenarios may require the prebuilt runtime base described by the corresponding Make targets.

## Before running

- Docker is available and the intended Compose project has no conflicting containers, ports, or networks.
- Frontend dependencies are installed from the repository's selected lockfile.
- Test tokens and credentials are fixtures, not production secrets.
- Targets are local deterministic fixtures unless a separately recorded authorization explicitly permits otherwise.
- Previous test data and browser state are removed through the stack's own teardown path.

## Evidence

Record the command, revision, Compose file/project, browser project, passed/failed/skipped counts, and first failure. A configured Playwright project is not evidence that it ran.

E2E authoring rules are in [`../conventions/testing.md`](../conventions/testing.md). Detailed fixture conventions may live in [`../../frontend/e2e/README.md`](../../frontend/e2e/README.md), but that file cannot override the repository testing convention.
