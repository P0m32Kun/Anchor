---
status: superseded
source_of_truth: false
owner: kun
last_updated: 2026-05-04
scope: scan-flow
superseded_by: docs/current/architecture.md
---

# Unified Scan Flow

> **Status: Superseded — Partially Implemented with Design Changes**
> This document describes the original 4-mode design (quick/standard/deep/custom). The implementation diverged:
> - Modes were reduced to 2: `external` and `internal`
> - ScanConfigPage was removed entirely; speed params moved into ScanModal Step 2
> - dnsx replaced the Go DNS resolver module
> - FOFA credentials moved from project-level to global `engine_credentials` table
> - Hunter and Quake search engines were added as a separate global feature (`/engines`)
>
> See `docs/current/architecture.md` for the current baseline.

Date: 2026-05-04
Scope: Backend + Frontend — merge two scan engines into one unified pipeline with clear mode selection

## Problem

Two disconnected scan execution paths exist:
- **Runs path** (`runs` table + `AssetDiscoveryWorkflow`): Only subfinder/httpx/naabu, no nuclei, no findings
- **Pipeline path** (`pipeline_runs` table + `Pipeline`): Full 9-stage pipeline with nuclei + findings, but not wired to frontend

Result: "Pipeline run not found" errors, empty findings, unclear scan progress, redundant config.

## Solution

Consolidate to a single Pipeline engine. Frontend gains a unified scan modal with 4 named modes. Backend adds mode support to pipeline runs.

## 1. Scan Modes

| Mode | ID | Description | PipelineConfig Override | Tool Chain |
|------|----|-------------|------------------------|------------|
| 快速侦察 | `quick` | 快速了解目标暴露面 | subfinder=off, fofa=off, cdn=off, nerva=off, nuclei=off | naabu → httpx |
| 标准扫描 | `standard` | 常规安全评估 | subfinder=on, fofa=off, cdn=off, nerva=on, nuclei=on (中高危) | subfinder → naabu → nerva → httpx → nuclei |
| 深度扫描 | `deep` | 全面安全审计 | all=on | fofa → subfinder → cdncheck → naabu → nerva → httpx → nuclei |
| 自定义 | `custom` | 高级用户手动配置 | user-defined | 用户自选 |

## 2. Backend Changes

### 2.1 New `mode` column on `pipeline_runs`

Migration v8: `ALTER TABLE pipeline_runs ADD COLUMN mode TEXT NOT NULL DEFAULT 'standard';`

### 2.2 New unified endpoint

`POST /projects/{id}/scan`
- Body: `{ mode: "quick"|"standard"|"deep"|"custom", name?: string }`
- Creates a `pipeline_runs` record with `mode` field
- Builds `PipelineConfig` based on mode (quick/standard/deep apply presets; custom reads from project config)
- Launches Pipeline in goroutine (same as current `handleRunPipeline`)
- Returns `202 Accepted` with `{ run_id, status, mode }`

### 2.3 Fix `createAndRunTask` to link tasks to run

In `pipeline.go:createAndRunTask`, set `RunID: &p.runID` on the ScanTask so tasks are queryable via `ListScanTasksByRun`.

### 2.4 Add `mode` field to `PipelineRun` model

```go
type PipelineRun struct {
    // ... existing fields
    Mode string `json:"mode" db:"mode"`
}
```

### 2.5 Update `ListPipelineRunsByProject` to support pagination

Add paginated variant matching the runs API pattern, so the frontend can use pipeline runs as the primary listing.

### 2.6 Deprecate old endpoints

Mark `POST /projects/{id}/runs` and `POST /projects/{id}/pipeline/run` as deprecated. Keep them working but the frontend will stop calling them.

## 3. Frontend Changes

### 3.1 New Scan Modal

Replace `CreateRunModal` in RunsPage with a new `ScanModal` component:
- 4 mode cards in a 2x2 grid
- Each card shows: icon, name, description, tool chain, estimated duration
- Target count display at bottom
- "Start Scan" button
- Calls `api.createScan(projectId, { mode })`

### 3.2 API Client

Add to `api.ts`:
```typescript
createScan(projectId: string, data: { mode: string; name?: string }): Promise<{ run_id: string; status: string; mode: string }>
listPipelineRuns(projectId: string, pagination?, signal?): Promise<PaginatedResponse<PipelineRun>>
getPipelineRun(projectId: string, runId: string): Promise<PipelineRun>
```

### 3.3 RunsPage Rewrite

- List from `pipeline_runs` instead of `runs`
- Show `mode` badge on each run (快速侦察/标准扫描/深度扫描/自定义)
- Click to expand: show stage progress + task list
- Fix the "Pipeline run not found" error by using consistent pipeline_runs API
- SSE: listen to `pipeline_stage_change` and `pipeline_complete` events

### 3.4 PipelineRun model update

```typescript
export interface PipelineRun {
  id: string;
  project_id: string;
  mode: string;
  status: string;
  stage?: string;
  error?: string;
  started_at: string;
  completed_at?: string;
  created_at: string;
}
```

### 3.5 ScanConfigPage Simplification

Remove tool enable/disable toggles (now handled by mode selection). Keep only:
- 并发数设置 (concurrency)
- 超时时间 (timeouts)
- 端口范围 (port range)
- nuclei 参数
- 排除规则 (scope exclusions)
- FOFA 凭证

## 4. Files to Modify

### Backend
| File | Changes |
|------|---------|
| `internal/db/db.go` | Migration v8: add `mode` to `pipeline_runs` |
| `internal/models/models.go` | Add `Mode` to `PipelineRun` struct |
| `internal/db/queries.go` | Update `CreatePipelineRun`, `ListPipelineRunsByProject` (pagination), add `CountPipelineRunsByProject` |
| `internal/workflow/pipeline.go` | Fix `createAndRunTask` to set `RunID` |
| `internal/api/pipeline_handlers.go` | New `handleCreateScan` handler, mode-based config builder |
| `internal/api/server.go` | Register `POST /projects/{id}/scan`, `GET /projects/{id}/runs` → pipeline_runs |

### Frontend
| File | Changes |
|------|---------|
| `frontend/src/lib/api.ts` | Add `createScan`, `listPipelineRuns`, `getPipelineRun`; update `Run` interface |
| `frontend/src/pages/RunsPage.tsx` | Rewrite: use pipeline_runs, new ScanModal, stage progress |
| `frontend/src/pages/ScanConfigPage.tsx` | Remove tool toggles, keep params only |

## 5. Out of Scope

- Merging `runs` and `pipeline_runs` tables (keep both, frontend just uses pipeline_runs)
- Removing `AssetDiscoveryWorkflow` (keep for backward compat)
- New tools or pipeline stages
- Page layout redesign (separate effort)
