---
status: in_review
source_of_truth: false
owner: kun
last_updated: 2026-05-05
scope: custom-nuclei-template-management
verification: pending_implementation
---

# Custom Nuclei Template Management Plan

> Status: In Review  
> Audience: Claude Code implementation agent  
> Default rule: this is a candidate implementation plan, not the current architecture baseline. Current baseline remains `docs/current/architecture.md`.

## 1. Background

Anchor currently relies on Nuclei templates that are available inside the worker runtime. The official template repository is installed in the worker base image via `nuclei -update-templates`, and scan commands rely on Nuclei's default template lookup plus tag filtering.

The new requirement is to let Anchor manage only user-defined custom Nuclei templates:

- add/import custom templates,
- manage files and directories,
- delete custom templates,
- publish a versioned custom template bundle,
- synchronize that bundle to all workers,
- allow custom workflows to run with custom template paths.

Official ProjectDiscovery templates are out of scope for management. They should remain worker/runtime managed and not appear in the custom management UI.

## 2. Official Nuclei Usage Notes

Nuclei supports custom template and workflow paths directly:

```bash
nuclei -l targets.txt -t /path/to/templates/
nuclei -l targets.txt -t /path/to/template.yaml
nuclei -l targets.txt -w /path/to/workflow.yaml
```

Useful official references:

- [Running Nuclei](https://docs.projectdiscovery.io/opensource/nuclei/running)
- [Template Workflows](https://docs.projectdiscovery.io/templates/workflows/overview)

For a distributed Anchor deployment, do not ask each worker to independently manage custom Git repositories. Server should own custom sources, publish a versioned bundle, and workers should synchronize that exact bundle.

## 3. Goals

1. Manage custom Nuclei template sources from Anchor Server.
2. Support Git repository import as the recommended long-term source.
3. Support upload import for zip/tar/file/folder-style workflows.
4. Let users browse, read, edit, create, and delete custom template files.
5. Validate custom templates and workflow references before publishing.
6. Publish immutable bundle versions.
7. Synchronize active custom bundle version to every worker.
8. Execute custom templates/workflows with explicit `-t` and `-w` paths.
9. Record the custom bundle version used by scan tasks for reproducibility.

## 4. Non-Goals

- Do not manage official ProjectDiscovery templates in Anchor UI.
- Do not delete, edit, or version official templates.
- Do not require users to store secrets for private repositories in this milestone unless already supported by the project.
- Do not change the existing official-template fingerprint-driven scan behavior except where needed to run additional custom template tasks.

## 5. Recommended User Workflow

Long-term custom templates should live in a separate GitHub repository, for example:

```text
anchor-nuclei-templates/
  templates/
    web/
    network/
    cves/
    exposures/
  workflows/
    springboot-workflow.yaml
    wordpress-workflow.yaml
  payloads/
  README.md
```

Anchor imports that repository as one custom source. Users may also upload ad-hoc templates or archives for quick testing. Uploaded files should still be represented as custom sources.

Private Git repository support is out of scope for this milestone. Recommended sources are public HTTPS repositories. If a user requires a private repository, they should mirror it as a public read-only repository, or wait for the credential-storage milestone. The Git import API may accept a `branch` field but should not accept a token/secret field in this milestone.

Custom workflow paths should be relative to the source root:

```yaml
id: springboot-workflow
info:
  name: Spring Boot Custom Workflow
  author: kun
  severity: info

workflows:
  - template: templates/web/springboot-detect.yaml
    subtemplates:
      - template: templates/cves/CVE-xxxx-yyyy.yaml
```

Anchor should execute that workflow with an absolute path inside the synchronized worker bundle:

```bash
nuclei -jsonl -l targets.txt \
  -w /data/nuclei/custom/current/sources/{source_id}/workflows/springboot-workflow.yaml
```

## 6. Storage Layout

Server-side storage under `ANCHOR_DATA_DIR`:

```text
{dataDir}/nuclei/custom/
  sources/
    {source_id}/
      source.json
      files/
        templates/
        workflows/
        payloads/
  bundles/
    {bundle_version}/
      manifest.json
      sources/
        {source_id}/...
    {bundle_version}.tar.gz
  current -> bundles/{bundle_version}
```

Worker-side storage:

```text
{dataDir}/nuclei/custom/
  bundles/
    {bundle_version}/
      manifest.json
      sources/
        {source_id}/...
  current -> bundles/{bundle_version}
```

The worker should switch `current` atomically after a successful download and validation.

## 7. Database Plan

Add table `nuclei_custom_sources`:

```sql
CREATE TABLE IF NOT EXISTS nuclei_custom_sources (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  type TEXT NOT NULL, -- git, upload, file
  uri TEXT,
  branch TEXT,
  enabled INTEGER NOT NULL DEFAULT 1,
  routing_policy TEXT NOT NULL DEFAULT 'manual',
  status TEXT NOT NULL DEFAULT 'draft',
  last_sync_at DATETIME,
  last_validate_at DATETIME,
  last_error TEXT,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL
);
```

Add table `nuclei_custom_bundles`:

```sql
CREATE TABLE IF NOT EXISTS nuclei_custom_bundles (
  version TEXT PRIMARY KEY,
  manifest_json TEXT NOT NULL,
  archive_path TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'active',
  created_at DATETIME NOT NULL,
  activated_at DATETIME
);
```

Reuse `worker_nodes.template_versions` to store worker-reported custom bundle version. Prefer JSON:

```json
{
  "nuclei_custom": "sha256:..."
}
```

Decision: this milestone adds a dedicated nullable column `nuclei_custom_bundle_version TEXT` to `scan_tasks` (or to the equivalent custom-nuclei task table introduced in Phase 4). This is a Phase 1 schema deliverable so that Phase 4 can record reproducibility metadata without a follow-up migration. The column stores the exact bundle version (e.g. `sha256:abc...`) that was synchronized to the worker when the task ran. NULL means the task did not use custom templates.

## 8. Backend API Plan

Custom source management:

```text
GET    /nuclei/custom/sources
POST   /nuclei/custom/sources/git
POST   /nuclei/custom/sources/upload
POST   /nuclei/custom/sources/{id}/refresh
PATCH  /nuclei/custom/sources/{id}
DELETE /nuclei/custom/sources/{id}
```

File management:

```text
GET    /nuclei/custom/sources/{id}/files
GET    /nuclei/custom/sources/{id}/files/{path...}
PUT    /nuclei/custom/sources/{id}/files/{path...}
DELETE /nuclei/custom/sources/{id}/files/{path...}
```

Validation and publishing:

```text
POST   /nuclei/custom/sources/{id}/validate
POST   /nuclei/custom/validate
POST   /nuclei/custom/publish
GET    /nuclei/custom/manifest
GET    /nuclei/custom/bundles/{version}.tar.gz
```

Suggested response for `GET /nuclei/custom/manifest`:

```json
{
  "active_version": "sha256:abc",
  "download_url": "/nuclei/custom/bundles/sha256-abc.tar.gz",
  "sources": [
    {
      "id": "src_1",
      "name": "anchor-nuclei-templates",
      "type": "git",
      "enabled": true
    }
  ],
  "created_at": "2026-05-05T00:00:00Z"
}
```

## 9. Validation Rules

Validation must run before publishing a bundle.

Path safety:

- reject absolute paths inside user-managed file paths,
- reject `..` traversal,
- reject symlinks that escape the source root,
- normalize paths with `filepath.Clean`,
- verify every resolved path remains inside the expected source root.

Allowed files:

- always allow `.yaml` and `.yml`,
- allow payload/resource files under `payloads/` if needed by templates,
- reject executable files by default,
- cap upload archive size and extracted file count.

Nuclei validation:

- run Nuclei template validation against custom source files where supported,
- capture validation output and store it in `last_error`,
- publishing must fail if enabled sources contain invalid templates.

Workflow reference validation:

- parse workflow YAML,
- find `template:` references and nested `subtemplates`,
- require relative paths,
- reject references outside the same source root,
- fail validation if a referenced template is missing.

## 10. Worker Synchronization Plan

Worker lifecycle:

1. Worker starts.
2. Worker registers with server.
3. Worker requests `GET /nuclei/custom/manifest`.
4. If `active_version` differs from local version, worker downloads bundle tarball.
5. Worker extracts to a temporary directory.
6. Worker validates expected manifest exists.
7. Worker atomically moves it into `bundles/{version}` and switches `current`.
8. Worker heartbeat reports `template_versions`.

Heartbeat payload should include template version:

```json
{
  "status": "idle",
  "capabilities": ["subfinder", "naabu", "httpx", "nuclei"],
  "template_versions": {
    "nuclei_custom": "sha256:abc"
  }
}
```

Server should persist this into `worker_nodes.template_versions`.

Dispatch behavior:

- For custom template tasks, prefer workers whose `nuclei_custom` version equals the task's required bundle version.
- If no synced worker exists, either trigger sync and retry or fail with a clear "custom templates not synced" error.
- Do not silently run custom scans on a worker with the wrong bundle version.

## 11. Scan Integration Plan

Official templates:

- keep existing fingerprint-driven `-tags` behavior,
- continue using official templates from Nuclei's default runtime location,
- do not expose official template files in custom management.

Custom templates:

- run as additional Nuclei tasks,
- pass explicit custom paths with `-t` and `-w`,
- do not depend on official tag mapping unless a custom source opts into it later.

Add scan config fields:

```json
{
  "enable_custom_nuclei": true,
  "custom_nuclei_refs": [
    {
      "source_id": "src_1",
      "kind": "template",
      "path": "templates/web/springboot-detect.yaml"
    },
    {
      "source_id": "src_1",
      "kind": "directory",
      "path": "templates/web/"
    },
    {
      "source_id": "src_1",
      "kind": "workflow",
      "path": "workflows/springboot-workflow.yaml"
    }
  ]
}
```

Refs execution strategy (TBD in Phase 4):

The `custom_nuclei_refs` array supports three execution shapes — one nuclei task per ref, one task per `source_id`, or one task for all refs. The trade-off is failure-isolation granularity vs. dispatch overhead. This decision is deferred to Phase 4 when scan integration is implemented; Phase 1–3 do not depend on it. Default lean: one nuclei task per ref, for clean per-ref attribution of findings, errors, and bundle versions.

Command examples:

```bash
nuclei -jsonl -l targets.txt \
  -t /data/nuclei/custom/current/sources/src_1/templates/web/springboot-detect.yaml
```

```bash
nuclei -jsonl -l targets.txt \
  -t /data/nuclei/custom/current/sources/src_1/templates/web/
```

```bash
nuclei -jsonl -l targets.txt \
  -w /data/nuclei/custom/current/sources/src_1/workflows/springboot-workflow.yaml
```

The command builder should avoid mixing `-w` and unrelated `-tags` filters unless intentionally supported. Custom workflow execution should be explicit and predictable.

## 12. Frontend Plan

Add a "Custom Nuclei Templates" management page.

Required UI:

- source list with status, type, enabled state, last validation time, last error,
- add Git source,
- upload template/archive,
- refresh Git source,
- enable/disable source,
- delete source,
- file tree,
- file viewer/editor for `.yaml/.yml`,
- create file/folder,
- delete file,
- validate source,
- publish bundle,
- worker sync status showing current custom bundle version.

Editing should create draft changes. Published bundles should be immutable.

## 13. Implementation Sequence

### Phase 1: Backend Custom Source Management

1. Add DB migrations and query methods.
2. Add models for custom source, bundle, manifest, file tree, validation result.
3. Implement filesystem service for source root operations.
4. Implement path safety helpers with focused tests.
5. Implement Git import and upload import.
6. Implement file CRUD APIs.
7. Implement source list and delete APIs.

### Phase 2: Validation and Publishing

1. Implement source-level validation.
2. Implement workflow reference validation.
3. Implement `POST /nuclei/custom/publish`.
4. Generate immutable bundle directory and `.tar.gz`.
5. Generate manifest with source metadata and content hash.
6. Add tests for invalid templates, invalid paths, missing workflow refs, and successful publish.

### Phase 3: Worker Sync

1. Add worker-side template sync component.
2. Fetch manifest at startup and periodically or during heartbeat.
3. Download and extract bundle.
4. Atomically switch `current`.
5. Report `template_versions` in heartbeat.
6. Persist worker template version on server.
7. Add tests for version comparison and failed download behavior.

### Phase 4: Scan Integration

1. Extend scan config for custom refs.
2. Add custom Nuclei command builder for `-t` and `-w`.
3. Create additional custom nuclei tasks after existing official nuclei stage.
4. Require matching worker bundle version for custom tasks.
5. Record bundle version used by custom tasks. Depends on the dedicated `nuclei_custom_bundle_version` column added in Phase 1 (see §7). Phase 4 must not introduce that schema change itself.
6. Add command builder and workflow pipeline tests.

Phase 4 also resolves the refs execution strategy noted in §11 (per-ref task vs. merged task). The default lean is one task per ref.

### Phase 5: Frontend

1. Add API client methods.
2. Add custom template management route/page.
3. Add source list, import dialogs, file tree, editor, validation, publish actions.
4. Add worker sync status display.
5. Add frontend typecheck and E2E coverage.

## 14. CodeGraph / Change Protocol

Before editing any function, class, or method, follow repository instructions:

```bash
codegraph status
codegraph impact <symbol>
```

Known high-risk area:

- `BuildNucleiCommand` currently has CRITICAL upstream impact because it feeds the official Nuclei scan path.

Prefer adding a new custom command builder instead of widening `BuildNucleiCommand` immediately. If `BuildNucleiCommand` must change, warn the user before editing and add focused regression tests for the official scan path.

## 15. Acceptance Criteria

### Backend API

- User can create a Git custom source.
- User can upload a custom template file or archive.
- User can list custom sources.
- User can list files under a source.
- User can read, edit, create, and delete files under a source.
- User can delete a custom source.
- Path traversal attempts are rejected.
- Symlink escape attempts are rejected.
- Invalid workflow references are reported with file path and reason.
- Invalid custom templates prevent publishing.
- Valid sources can be published into an immutable bundle.
- `GET /nuclei/custom/manifest` returns the active bundle version.
- `GET /nuclei/custom/bundles/{version}.tar.gz` downloads the active bundle.

### Worker Sync

- A worker with no custom bundle downloads the active bundle on startup.
- A worker with an old custom bundle updates to the active version.
- A worker reports `template_versions.nuclei_custom` in heartbeat.
- Server persists the worker custom template version.
- Failed bundle download does not corrupt the existing `current` bundle.
- Worker never switches `current` until extraction succeeds.

### Scan Execution

- Official nuclei scanning still works with existing fingerprint-driven `-tags` behavior.
- A selected custom template file runs with `-t`.
- A selected custom template directory runs with `-t`.
- A selected custom workflow runs with `-w`.
- Custom workflow relative template references resolve on the worker.
- Custom task refuses to run on a worker with the wrong bundle version.
- Findings produced by custom templates are parsed through the existing Nuclei parser.
- The custom bundle version used for a task is recorded or visible in task metadata/logs.

### Frontend

- User can add a Git source from the UI.
- User can upload a custom template/archive from the UI.
- User can browse the source file tree.
- User can edit and delete custom files.
- User can validate a source and see errors.
- User can publish a bundle.
- User can see worker custom bundle sync status.
- Official templates are not shown as managed files.

### Regression

- Existing official Nuclei command builder tests still pass.
- Existing pipeline tests still pass.
- No official-template management UI or API is introduced.
- Deleting a custom source does not affect already-published immutable bundles used by running tasks.

## 16. Suggested Validation Commands

Use the smallest meaningful validation first:

```bash
go test ./internal/nuclei/... ./internal/worker/... ./internal/api/... ./internal/workflow/...
go test ./...
cd frontend && npm run typecheck
cd frontend && npm run test:e2e
```

If E2E cannot run because Docker, browsers, or external tools are missing, report the exact blocker and list remaining verification.

## 17. Implementation Notes

- Keep official and custom template concerns separate in naming and storage.
- Prefer immutable bundle versions over mutable shared directories.
- Prefer atomic directory/symlink switches for worker safety.
- Prefer explicit `-t` and `-w` custom paths over relying on Nuclei default paths.
- Avoid broad refactors in the first implementation.
- Keep user-uploaded files under the custom source root only.
- Treat custom templates as untrusted input until validated.
