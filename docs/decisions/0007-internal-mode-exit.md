# 0007. Retire the dedicated internal scan scenario (P1-1)

- Status: accepted
- Date: 2026-08-21

## Context

Dedicated internal-network scanning is not a product scenario (see ADR 0003 and
`docs/current/product.md#product-boundaries`). The source tree nevertheless
retained a legacy `internal` scan preset, a dedicated UI choice, saved
configurations, and a specialized execution branch. Leaving it in place risks
treating it as a supported surface, silently widening scope, and splitting the
Internet-only boundary.

## Decision

Remove the dedicated internal scan scenario with an explicit compatibility exit:

- New `POST /projects/{id}/scan` requests with `mode=internal` are rejected with
  `410 INTERNAL_MODE_REMOVED` and a documented error message. No field, UI,
  documentation, or test expands the internal mode.
- A saved internal-shaped project `pipeline_config` is either rejected or
  migrated only on an explicit act: the caller passes `?migrate=external`, which
  rewrites the stored config to the conservative Internet baseline. The server
  never silently rewrites scope.
- The frontend offers the Internet scan mode only and rewrites a stored
  `internal` mode to `external`; the internal card and branch are removed.
- Migration 46 converts legacy `internal` tool templates to `external` and
  annotates them. Historical `pipeline_runs` with `mode='internal'` are retained
  as immutable audit rows and rendered with a legacy badge.
- The scan engine retains only the reusable Internet profile; the internal
  profile (`DefaultInternalProfile`) and internal preset are deleted.

## Consequences

New and existing clients get a clear, permanent compatibility signal instead of
ambiguous behavior. Operators can migrate saved configurations deliberately.
The Internet-only boundary is enforced uniformly at the API, UI, schema, and
execution layers. Breaking change for any client still issuing `internal`
scans: they must either stop or explicitly migrate with `?migrate=external`.
