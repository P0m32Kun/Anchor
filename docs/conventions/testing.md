# Testing and delivery convention

- Status: normative
- Updated: 2026-08-20

Tests provide evidence for observable behavior. Select the lowest layer that can fail for the change, then add broader evidence when the behavior crosses boundaries.

## Layers

| Layer | Use | Allowed dependencies |
| --- | --- | --- |
| Go unit | Parsers, normalization, rules, scoring, pure state transitions | In-process fakes and temporary files |
| Go integration | SQLite queries/migrations, handlers, package composition | Temporary SQLite DB, real migrations, `httptest` |
| React unit | Components, stores, API client edge behavior | Vitest and DOM test utilities; mocked network boundary |
| E2E | Critical user journeys and deployment wiring | Playwright against the dedicated Docker E2E stack |
| Manual | Subjective UX, external-tool availability, or an environment not safely automated | Recorded steps and observed result |

Public DNS, arbitrary internet targets, installed scanner binaries, and developer machine state are not unit-test dependencies.

## Change flow

1. State the user or system behavior and an observable acceptance signal before implementation.
2. Reproduce a bug or add the smallest failing test when the change supports test-first work.
3. Implement the smallest coherent slice and keep focused tests green.
4. Run the broader relevant suite for the affected boundary.
5. Exercise a critical user-visible flow through E2E or record why manual acceptance remains.
6. Report exact commands, observed failures, skipped checks, and pre-existing baseline failures.

A build or typecheck is compilation evidence, not behavioral acceptance.

## E2E contract

- User actions that exist in the UI are performed through the UI.
- API calls may seed and clean fixtures or poll a long-running job; final business assertions return to the UI.
- Prefer state-based waits and visible assertions over fixed sleeps.
- Keep the production deployment and the E2E Compose stack separate.
- Use only targets and fixtures covered by explicit authorization. Local deterministic fixtures are the default.
- A workaround for a known product bug names the bug and the removal condition next to the test.

Commands and environment preparation live in [`../runbooks/e2e-testing.md`](../runbooks/e2e-testing.md). Manual acceptance format lives in [`../functional-test.md`](../functional-test.md).

## Completion evidence

- Core logic has focused coverage.
- Persistence or HTTP changes have integration coverage.
- User-visible critical behavior has E2E or recorded manual evidence.
- Security-sensitive execution changes test denial as well as success.
- Documentation changes pass `make check-docs`.
- Any unavailable check or existing red baseline is named; it is not silently replaced by weaker evidence.
