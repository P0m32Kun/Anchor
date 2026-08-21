# 0001. Use one documentation authority chain

- Status: superseded by 0008
- Date: 2026-08-20

## Context

Anchor accumulated multiple agent guides, roadmaps, architecture snapshots, design trees, audit reports, and changelogs. Several were marked authoritative simultaneously, referenced deleted code or missing skills, and contradicted newer product work. Agents could choose a plausible but wrong document and implement against stale assumptions.

## Decision

Use one ordered authority chain: `AGENTS.md` → `docs/README.md` → product → architecture → plan → decisions. `docs/README.md` alone defines document lifecycle and navigation.

Keep only three repository-wide current documents. Put operational procedures, stable reference facts, unaccepted ideas, historical incidents, and superseded material in separate named tiers. `CLAUDE.md` is a thin pointer to `AGENTS.md`. Prose must not cache cheap implementation inventories such as route maps or file counts.

Automated documentation checks enforce required files, allowed live locations, working local links, retired path/skill references, and the absence of competing `source_of_truth` declarations.

## Consequences

Future changes have one obvious document to update and one active plan to consult. Historical reasoning remains available in archives without steering default implementation. Adding a new live document category or another authority source is a direction change and requires deliberate review.
