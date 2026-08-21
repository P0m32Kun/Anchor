# Decisions

Accepted ADRs record direction-setting choices that future work should not re-derive. They explain why; the current product, architecture, and plan documents show the operational baseline.

## Index

- [`0001-document-authority.md`](0001-document-authority.md): one documentation authority chain and explicit lifecycle tiers.
- [`0002-product-baseline.md`](0002-product-baseline.md): superseded local-friendly baseline, retained for history.
- [`0003-distributed-internet-asm-baseline.md`](0003-distributed-internet-asm-baseline.md): distributed internet attack-surface mapping for authorized adversary-emulation exercises.
- [`0004-deterministic-non-agent-product.md`](0004-deterministic-non-agent-product.md): deterministic execution without LLM, MCP, skill, or autonomous agent runtime.
- [`0005-semantic-scan-execution-adapters.md`](0005-semantic-scan-execution-adapters.md): semantic scan requests with SDK and process adapters, starting with a Naabu pilot.
- [`0006-nuclei-weak-auth-policy.md`](0006-nuclei-weak-auth-policy.md): two-high-one-weak coverage through policy-gated Nuclei and RBKD templates.
- [`0007-internal-mode-exit.md`](0007-internal-mode-exit.md): retire the dedicated internal scan scenario with an explicit 410-migrate compatibility exit.

Add a zero-padded ADR when a change alters product direction, architecture, a public contract, or a cross-file invariant. Update the affected current document in the same change. Supersede accepted ADRs with a new record rather than rewriting their decision.
