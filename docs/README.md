# Anchor documentation

This file is the only documentation index and the only place that defines document authority.

## Authority chain

Read these documents in order:

1. [`../AGENTS.md`](../AGENTS.md): repository working contract for coding agents.
2. [`current/product.md`](current/product.md): who Anchor serves, the value it provides, and what is outside its current scope.
3. [`current/architecture.md`](current/architecture.md): architecture observed in the current source tree.
4. [`current/plan.md`](current/plan.md): the only active repository-wide work queue.
5. [`decisions/`](decisions/): accepted rationale for direction-setting choices.

An accepted decision that changes the baseline must update the affected current document in the same change. If the chain conflicts, repair it before implementation. No proposal, runbook, reference, incident note, or archive may override it.

## Live document map

| Location | Purpose | Authority |
| --- | --- | --- |
| [`current/`](current/) | Product, implemented architecture, and active plan | Canonical for its named subject |
| [`decisions/`](decisions/) | Durable accepted decisions and rationale | Binding when reflected in current docs |
| [`conventions/`](conventions/) | Coding, API, frontend, and testing rules | Normative within their subject |
| [`functional-test.md`](functional-test.md) | Manual and E2E acceptance inventory | Verification reference |
| [`runbooks/`](runbooks/) | Deployment, CI/CD, and E2E operations | Operational procedure, not product direction |
| [`reference/`](reference/) | Stable API, error, migration, and FAQ facts | Reference, subordinate to code |
| [`proposals/`](proposals/) | Unaccepted options and external-project research | Non-authoritative |
| [`pitfalls/`](pitfalls/) | Historical failure patterns | Non-normative experience |
| [`archived/`](archived/) | Superseded snapshots retained for archaeology | Historical only |

The root [`CHANGELOG.md`](../CHANGELOG.md) is the only changelog. The `.cursor/` tree is local tooling support and never a product or architecture source.

## Lifecycle

- **Current**: a fact in one of the three files under `current/`.
- **Accepted**: a decision in `decisions/`, reflected into current docs when it changes them.
- **Proposal**: an option under `proposals/`; implementation requires explicit approval and, when direction changes, an ADR.
- **Reference**: a lookup that must follow code and configuration.
- **Historical**: material under `archived/` or `pitfalls/`; useful for context, never for default implementation guidance.

Do not create a second roadmap, architecture baseline, agent guide, changelog, or document index. Extend the owning current document or archive the superseded material.

## Maintenance

- Store reasons and stable contracts, not inventories that can be read cheaply from source.
- Link to the owning document instead of copying its rule.
- Move replaced material to `archived/`; do not leave contradictory siblings live.
- Run `make check-docs` after changing Markdown, `AGENTS.md`, or `CLAUDE.md`.
