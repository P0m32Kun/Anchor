# 0004. Keep Anchor deterministic and non-agentic

- Status: accepted
- Date: 2026-08-20

## Context

ScopeSentry's current releases include MCP and skill integration, and the broader security-tool ecosystem increasingly presents LLM-driven agents as an automation layer. Anchor's product boundary is authorized internet attack-surface mapping with explicit operator control. Adding an LLM, MCP server, skill runtime, or autonomous planner would introduce a new trust boundary, nondeterministic execution, prompt/tool injection risk, and a product responsibility that is not required for the asset/work model.

## Decision

Anchor does not embed or depend on an LLM, MCP server, skill runtime, or autonomous agent planner. Work derivation, prioritization, tool selection, policy enforcement, and result normalization remain deterministic and auditable. Exposed AI/LLM services may be represented and assessed as ordinary internet assets using deterministic rules; Anchor never calls a model to plan, interpret, or execute a scan.

## Consequences

The product keeps a small, inspectable trust boundary and predictable replay behavior. ScopeSentry MCP/skill features are explicitly design input to exclude, not a capability to port. Future automation must be expressed as typed policy, state-machine, or rule changes with evidence, not hidden in an agent prompt.
