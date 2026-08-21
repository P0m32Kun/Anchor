# 0008. Keep AGENTS.md as the sole repository Agent contract

- Status: accepted
- Date: 2026-08-21

## Context

ADR 0001 established `AGENTS.md` as the repository working contract while retaining a thin `CLAUDE.md` pointer for Claude-specific discovery. The project no longer needs a second agent entrypoint, and keeping one adds a required file and a compatibility check without adding independent guidance.

## Decision

Remove the repository-root `CLAUDE.md` pointer. `AGENTS.md` is the only repository-wide Agent contract, and documentation checks must require and validate `AGENTS.md` without requiring or validating `CLAUDE.md`.

## Consequences

There is one maintained Agent instruction entrypoint and no compatibility shim to keep synchronized. Claude integrations that rely on automatic `CLAUDE.md` discovery will not receive a repository-specific pointer; agents must follow `AGENTS.md` through the repository's documented authority chain.
