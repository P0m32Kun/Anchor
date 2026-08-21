# FAQ

- Status: reference
- Updated: 2026-08-20

## Where is the current roadmap?

Only [`../current/plan.md`](../current/plan.md) is active.

## Is Anchor a fixed pipeline?

No. UI stages summarize progress. Asset type, attributes, profile, depth, and completion state drive work derivation. See [`../current/architecture.md`](../current/architecture.md).

## Is the scan engine behind a feature flag?

No repository-level enable flag is part of the current baseline. Confirm startup behavior in `main.go` and scan creation in `internal/api/pipeline_handlers.go`.

## Is Scope an allowlist?

The current product uses project targets plus exclusion behavior. Exact normalization and decision semantics live in `internal/scope/`, `internal/exclude/`, and their tests. Security-sensitive changes require denial-path tests.

## Why did a scan produce no findings?

Check run/work status, tool health, scope or exclusion decisions, selected profile, credentials for passive engines, parser output, and fingerprint-dependent Nuclei eligibility. Use API/UI diagnostics without logging credentials.

## Which deployment or test command should I use?

Use [`../runbooks/deployment.md`](../runbooks/deployment.md) for packaged deployment and [`../runbooks/e2e-testing.md`](../runbooks/e2e-testing.md) for the isolated test stack. Do not mix their Compose projects.

## Can we copy features from another open-source scanner?

Ideas can become proposals. Code, rules, templates, dictionaries, and assets require license and provenance review before import. See [`../proposals/scopesentry-capabilities.md`](../proposals/scopesentry-capabilities.md) for the current research example.
