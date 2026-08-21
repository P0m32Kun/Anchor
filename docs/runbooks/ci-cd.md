# CI and release runbook

- Status: operational
- Updated: 2026-08-20

Workflow YAML under `.github/workflows/`, the `Makefile`, and release scripts are the source of truth. This runbook defines evidence and ordering, not a cached job inventory.

## Pull-request evidence

- Documentation validation runs for agent and Markdown changes.
- Go vet and tests cover backend changes.
- Frontend clean install, unit tests, typecheck, and build cover frontend changes.
- Docker-backed E2E runs when the changed behavior needs full-stack evidence; long suites may be a recorded pre-release check rather than every PR.

Local aggregate check:

```bash
scripts/pre-merge-check.sh
```

If the baseline is red, record the exact pre-existing failure and use a focused check to prove the changed path. Do not relabel a skipped job as passed.

## Release

1. Choose an exact revision and ensure required CI checks are green.
2. Run the production-image verification target defined by the `Makefile` or release workflow.
3. Record image digests, migration expectations, smoke results, and any unavailable checks.
4. Tag and publish only after the release-verification evidence is accepted.
5. Keep rollback images and the matching pre-upgrade data backup available.

Deployment steps are in [`deployment.md`](deployment.md); E2E environment steps are in [`e2e-testing.md`](e2e-testing.md).
