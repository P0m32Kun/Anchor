---
status: in_review
source_of_truth: false
owner: kun
last_updated: 2026-06-04
scope: src-bounty-workstation
---

# SRC Bounty Workstation Design

This document reviews and replaces the pasted Claude Code plan for turning Anchor into an SRC bounty workstation.

It is a proposal, not the current architecture baseline. Promote it only after implementation and acceptance evidence are complete.

## Verdict

The original direction is correct: Anchor should become an SRC workflow tool, not just a scanner.

The original plan is not ready for implementation as written. It over-expands into seven repositories, references files that do not exist in Anchor, introduces unnecessary storage paths, and puts high-risk/low-ROI work too early.

This revised plan keeps the goal but changes the implementation order:

1. First close the bounty workflow inside Anchor: program rules, candidate queue, submission pack.
2. Then add low-noise expansion through existing scan config, dict, and Spoor output.
3. Only after that add authenticated differential testing.
4. Keep POC, Pentest-Playbook, vuln-auto-pipeline, and finger as supporting work, not blockers.

## Problems In The Original Plan

| Original item | Problem | Replacement |
| --- | --- | --- |
| Phase 0 requires cleaning unrelated repositories | Not part of the product change; blocks work on unrelated git state | Do not include as a product phase. Handle dirty worktrees separately before editing each repo. |
| `internal/api/router.go` | Anchor has no router file. Routes are registered in `internal/api/server.go` and handlers live in separate `*_handlers.go` files. | Add routes in `server.go`; add `program_handlers.go`, `bounty_handlers.go`, `submission_handlers.go`. |
| Program Profile JSON store then SQLite | Anchor already uses SQLite migrations and `db.Queries`; a JSON v1 store creates a migration/deletion burden. | Use SQLite from the first implementation. |
| New Program Profile scope model duplicates project targets/scope | Anchor already has projects, targets, scope rules, excluded domains, and pipeline config. | Add a lightweight `src_programs` table and link it to project configuration. Keep scan targets and exclusions in existing tables. |
| Bounty Queue stores complex nested score object | Harder to query/sort in SQLite and unnecessary for v1. | Store numeric score columns and reason strings directly. |
| Submission Pack tries Markdown/PDF in v1 | PDF adds rendering/test cost and is not needed to submit SRC reports. | Markdown only in v1. Add PDF only after Markdown pack is stable. |
| `internal/scanengine/profiles/external.go` | That path does not exist. Existing profile code is under `internal/scanengine/core/`. | Extend `PipelineConfig` and `ProfileFromConfig`, not a new profile package. |
| High-value templates inside Anchor internal path | Built-in templates are currently managed through RBKD-templates / custom nuclei sources, not Go internal files. | Put high-signal workflows in RBKD-templates or as seeded custom source. Anchor only selects them. |
| finger metadata fields such as `bounty_priority` | `finger.json` follows wappalyzer-compatible schema. Non-standard metadata should not become a core dependency unless the parser supports it. | Keep finger focused on detection. Put bounty weighting in Anchor. |
| Credential vault stores plain map values | Sensitive credentials need encryption/redaction and strict output filtering. | Phase later. Store redacted metadata first; encrypted secret storage only when authenticated testing is implemented. |
| IDOR detection as a Nuclei workflow | IDOR is stateful and account-specific; a static nuclei workflow is insufficient. | Implement differential testing in Anchor using two credential contexts and safe read-only requests. |
| POC high-value directory in early milestone | Raises scope and compliance risk while not needed for first bounty workflow. | Defer. Use POC only for verified authorized cases and template hardening. |

## Target Product Shape

Anchor should add a new SRC workflow layer on top of the existing scan engine:

```text
Project + Program rules
  -> Authorized low-noise scan
  -> Findings / JS endpoints / exposed assets
  -> Bounty Queue ranking
  -> Manual verification
  -> Submission Pack
  -> Submitted / duplicate / accepted / paid tracking
```

The scan engine remains asset-driven. This design does not reintroduce a fixed linear pipeline.

## Data Model

### `src_programs`

Use this table for platform rules and bounty context. Do not duplicate targets.

Recommended fields:

| Field | Notes |
| --- | --- |
| `id` | Anchor ID |
| `project_id` | One project can have zero or one active SRC profile in v1 |
| `name` | Program name, for example `360SRC` or company name |
| `platform` | `360src`, `butian`, `hackerone`, `bugcrowd`, `other` |
| `program_url` | Platform URL |
| `rules_url` | Rules / scope policy URL |
| `allow_automation` | Whether automation is allowed |
| `allow_dir_brute` | Whether low-noise path brute force is allowed |
| `allow_weak_password` | Default false; requires explicit authorization |
| `allow_authenticated_test` | Default false |
| `max_rps` | Program-level safe rate limit |
| `max_concurrency` | Program-level concurrency limit |
| `preferred_vuln_types` | JSON array, for ranking only |
| `payout_hint` | JSON object, optional, for sorting hints |
| `notes` | Manual notes |
| `created_at`, `updated_at` | Timestamps |

Scope stays in existing project targets, scope rules, and excluded domains. Program rules constrain scan config; they do not replace scope evaluation.

### `bounty_candidates`

Use this table as a ranking and tracking layer over findings or API endpoints.

Recommended fields:

| Field | Notes |
| --- | --- |
| `id` | Anchor ID |
| `project_id` | Required |
| `program_id` | Optional until program is configured |
| `finding_id` | Nullable; used for Nuclei/Spoor secret findings |
| `endpoint_id` | Nullable; future API endpoint queue linkage |
| `source_kind` | `finding`, `api_endpoint`, `asset`, `manual` |
| `title` | Candidate title snapshot |
| `vuln_type` | Normalized type, e.g. `rce`, `ssrf`, `idor`, `secret_leak` |
| `severity` | Snapshot from finding or scoring |
| `confidence` | Snapshot from finding or scoring |
| `value_score` | 0-100 |
| `impact_score` | 0-30 |
| `novelty_score` | 0-20 |
| `repro_score` | 0-20 |
| `scope_score` | 0-15 |
| `safety_score` | 0-15 |
| `duplicate_risk` | `low`, `medium`, `high`, `unknown` |
| `ranking_reason` | Human-readable reason |
| `verify_status` | `pending`, `verifying`, `confirmed`, `false_positive`, `not_applicable` |
| `submission_status` | `not_ready`, `ready`, `submitted`, `duplicate`, `accepted`, `rejected`, `paid` |
| `submission_url` | Optional |
| `submission_id` | Optional |
| `paid_amount` | Optional integer amount |
| `notes` | Manual notes |
| `created_at`, `updated_at` | Timestamps |

Add a unique partial dedup rule in code: do not create duplicate candidates for the same project + finding or project + endpoint.

### `submission_packs`

Use this table to snapshot generated report text and checklist state.

Recommended fields:

| Field | Notes |
| --- | --- |
| `id` | Anchor ID |
| `candidate_id` | Required |
| `format` | `markdown` in v1 |
| `template` | `generic`, `360src`, `hackerone`, `bugcrowd` |
| `content` | Generated Markdown |
| `checklist_json` | Checklist result |
| `redaction_status` | `raw`, `reviewed`, `redacted` |
| `created_at`, `updated_at` | Timestamps |

Do not add PDF in v1.

## Backend Implementation Plan

### PR1: Program Rules

Files:

- `internal/models/src_program.go`
- `internal/db/v29.go` or next migration number
- `internal/db/queries_src_program.go`
- `internal/api/program_handlers.go`
- `internal/api/server.go`
- `internal/api/README.md`

Routes:

```text
POST   /projects/{id}/src-program
GET    /projects/{id}/src-program
PUT    /projects/{id}/src-program
DELETE /projects/{id}/src-program
```

Do not add global `/api/v1/programs` routes. Anchor currently uses project-scoped routes without `/api/v1`.

Acceptance:

- Can create, read, update, delete one program profile per project.
- Program `max_rps` and `max_concurrency` are shown to scan creation code, but PR1 does not need to enforce them yet.
- Tests cover CRUD and project ownership validation.

### PR2: Bounty Queue

Files:

- `internal/models/bounty_candidate.go`
- `internal/db/v30.go` or next migration number
- `internal/db/queries_bounty_candidate.go`
- `internal/bounty/scorer.go`
- `internal/bounty/duplicate_risk.go`
- `internal/api/bounty_handlers.go`
- `internal/api/server.go`
- `frontend/src/pages/BountyQueuePage.tsx`

Routes:

```text
GET   /projects/{id}/bounty-candidates
POST  /projects/{id}/bounty-candidates/refresh
GET   /bounty-candidates/{id}
PATCH /bounty-candidates/{id}
```

Candidate generation:

- Start with findings only.
- Include `nuclei` high/critical findings.
- Include `spoor` secret findings.
- Include medium findings only when source rule type is high-signal, such as exposed config, unauthenticated API docs, cloud key, actuator, file read.
- Exclude pure `info` and low-confidence banner/version findings from candidate generation.

Scoring v1:

```text
base = severity_weight + confidence_weight
impact = type_weight
novelty = source_weight
repro = evidence_weight
scope = target_weight
safety = safe_method_weight
duplicate_penalty = duplicate_risk_weight
value_score = clamp(base + impact + novelty + repro + scope + safety - duplicate_penalty)
```

Duplicate risk v1:

- High: common public nuclei CVE template on popular core target.
- Medium: known template with request/response evidence but no business impact.
- Low: JS-discovered endpoint, secret, or manual finding with target-specific evidence.
- Unknown: insufficient data.

Acceptance:

- Refreshing candidates is idempotent.
- Candidates sort by `value_score desc`, then `duplicate_risk`, then `created_at`.
- A scan with seeded Nuclei/Spoor findings produces a deterministic top list.

### PR3: Submission Pack

Files:

- `internal/models/submission_pack.go`
- `internal/db/v31.go` or next migration number
- `internal/db/queries_submission_pack.go`
- `internal/submission/pack.go`
- `internal/submission/checklist.go`
- `internal/api/submission_handlers.go`
- `internal/api/server.go`
- `frontend/src/pages/SubmissionPackPage.tsx`

Routes:

```text
POST /bounty-candidates/{id}/submission-pack
GET  /bounty-candidates/{id}/submission-pack
```

Markdown sections:

```markdown
# {{title}}

## 基本信息
- 漏洞类型
- 影响等级
- 影响资产
- SRC 项目

## 漏洞描述

## 复现步骤

## 请求与响应

## 影响说明

## 边界说明

## 修复建议

## 审核前检查清单
```

Checklist:

- Scope confirmed.
- Evidence exists.
- Reproduction steps are present.
- Sensitive data is redacted or minimized.
- No destructive action is required.
- Not pure banner/version/info.
- Third-party domain risk reviewed.
- Manual verification status is `confirmed` before submission status can become `ready`.

Acceptance:

- Pack generation works for a candidate backed by one finding and evidence.
- Pack generation refuses `false_positive` candidates.
- Generated content contains sanitized request/response excerpts, not raw unbounded artifacts.

### PR4: Low-Noise SRC Scan Mode

This is the first cross-repo dependency, but Anchor should not wait for all dict changes.

Anchor changes:

- Extend `models.PipelineConfig` with a named preset or mode value such as `src_low_noise`.
- Use existing `EnableFfuf`, `EnableKatana`, `EnableNuclei`, `NucleiRequireFingerprint`, and rate fields rather than creating a new scan profile package.
- In `core.ProfileFromConfig`, keep the external profile but enable ffuf only when config explicitly allows it.
- Add UI preset in `ScanModal`.

dict changes:

- Add small, bounded wordlists under existing categories, not a new incompatible layout:
  - `path/swagger.txt`
  - `path/actuator.txt`
  - `path/admin-panel-low-noise.txt`
  - `path/debug-low-noise.txt`
  - `path/sensitive-files-low-noise.txt`
- Add `path/src-low-noise.txt` as a curated aggregate.

Nuclei workflow:

- Put `bounty-high-signal` in RBKD-templates or a custom nuclei source.
- Anchor selects the workflow; Anchor should not store Nuclei YAML under `internal/`.

Acceptance:

- External scan default remains conservative.
- `src_low_noise` enables only bounded ffuf dictionaries and high-signal Nuclei workflows.
- Program rule `allow_dir_brute=false` prevents ffuf even if the user selects `src_low_noise`.

### PR5: Spoor Endpoint Enrichment

Do not block PR1-PR4 on Spoor.

Spoor changes:

- Extend existing `Finding` output rather than inventing a separate incompatible envelope.
- Add optional fields only:
  - `method`
  - `params`
  - `sensitivity`
  - `tags`
  - `requires_auth_hint`
- Keep backward compatibility with Anchor's current parser.

Anchor changes:

- Update `internal/scanengine/executor/spoor.go` to read optional fields.
- Initially create bounty candidates directly from high-sensitivity Spoor endpoints.
- Add a dedicated `api_endpoint_candidates` table only if endpoint workflow needs more state than `bounty_candidates` can represent.

High-signal endpoint tags:

- `admin`
- `role`
- `user`
- `order`
- `export`
- `download`
- `upload`
- `token`
- `secret`
- `config`
- `debug`
- `swagger`
- `graphql`
- `actuator`

Acceptance:

- Old Spoor JSONL still parses.
- New endpoint fields improve candidate ranking.
- No authenticated or state-changing tests are performed in this PR.

### PR6: Authenticated Differential Testing

This is explicitly not part of the first month unless PR1-PR5 finish early.

Constraints:

- Only implement after Program Profile supports `allow_authenticated_test`.
- Default to read-only methods.
- Do not store secrets in plain text.
- Redact credentials from logs, tool invocations, artifacts, SSE, and reports.

Implementation direction:

- `credential_profiles`: encrypted credential references and redacted labels.
- `auth_contexts`: named roles such as `user_a`, `user_b`, `admin_readonly`.
- `differential_tests`: request templates and results.
- No static Nuclei IDOR workflow as the primary mechanism.

Acceptance:

- Two-account read-only IDOR check can compare status code, body shape, and sensitive field presence.
- Unsafe methods require explicit per-program and per-test confirmation.
- Credentials never appear in artifacts or generated packs.

## Frontend Plan

Add only the minimum UI needed per milestone.

PR1:

- Project settings section: SRC Program.

PR2:

- `BountyQueuePage`: table/list sorted by score.
- Candidate detail drawer reusing existing finding detail/evidence UI.

PR3:

- Submission pack preview page.
- Checklist editor.
- Copy Markdown button.

PR4:

- `ScanModal` preset: `SRC Low Noise`.
- Show disabled reason when program rules forbid ffuf or automation.

Avoid building a complex Kanban until the data model is proven useful.

## Multi-Repository Scope

| Repository | Role | Timing |
| --- | --- | --- |
| Anchor | Core implementation | PR1-PR6 |
| dict | Low-noise wordlists | PR4 |
| Spoor | Endpoint enrichment | PR5 |
| finger | More accurate detection only | Optional; do not add bounty metadata as required input |
| POC | Verified authorized PoC hardening | Later; not in the first month |
| Pentest-Playbook | Document manual SRC workflow | Parallel after PR3 |
| vuln-auto-pipeline | Update design docs after Anchor model stabilizes | Later |
| pi-audit-agent | Feeds custom findings/templates into Anchor | Separate workflow, not a dependency |

## Thirty-Day Plan

| Days | Deliverable | Scope |
| --- | --- | --- |
| 1-4 | Program rules | Anchor only |
| 5-10 | Bounty Queue v1 | Anchor only |
| 11-15 | Submission Pack v1 | Anchor only |
| 16-20 | SRC Low Noise preset | Anchor + dict + template source |
| 21-25 | Spoor endpoint enrichment | Spoor + Anchor parser |
| 26-30 | Polish, tests, real target dry-run | No new major features |

Authenticated differential testing is moved to the next cycle unless the first five deliverables are complete and verified.

## Acceptance Path

Use one controlled test project:

1. Add SRC Program profile with automation allowed, directory brute allowed, low rate limits.
2. Add scoped test domain or local rangefield target.
3. Run `src_low_noise` scan.
4. Confirm that findings create bounty candidates.
5. Mark one candidate confirmed.
6. Generate submission pack.
7. Verify generated report has scope, impact, evidence, safe boundary statement, and checklist.

Minimum automated tests:

- DB migration tests for new tables.
- Query tests for CRUD and idempotent refresh.
- Scorer table-driven tests.
- API handler tests for project ownership and invalid state transitions.
- Frontend smoke tests for queue rendering if existing frontend test harness supports it.

## Non-Goals For The First Cycle

- No SaaS multi-tenant billing.
- No PDF export.
- No ML ranking.
- No full credential vault.
- No destructive authenticated testing.
- No broad weak-password testing.
- No large public PoC expansion.
- No rewrite of the scan engine.
- No replacement of existing Project/Target/Scope models.

## Promotion Criteria

This proposal can be promoted into `docs/current/plan.md` only when:

- PR1-PR3 are accepted or explicitly scoped as the next milestone.
- Routes and data models are implemented and tested.
- `docs/current/architecture.md` is updated with the accepted SRC workflow layer.
- Any cross-repo dependency is documented with exact contract changes.
