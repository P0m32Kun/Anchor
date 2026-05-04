# Unified Scan Flow — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Merge two disconnected scan engines into a single unified Pipeline with 4 named scan modes, a unified frontend modal, and proper run tracking.

**Architecture:** All scans go through the Pipeline engine. A new `POST /projects/{id}/scan` endpoint accepts a `mode` parameter, builds the appropriate `PipelineConfig` preset, and launches the pipeline. Frontend lists from `pipeline_runs` instead of `runs`.

**Tech Stack:** Go (backend), React + TypeScript (frontend), SQLite, SSE

**Spec:** `docs/superpowers/specs/2026-05-04-unified-scan-flow.md`

---

### Task 1: Database Migration — Add `mode` to `pipeline_runs`

**Files:**
- Modify: `internal/db/db.go`
- Modify: `internal/models/models.go`

- [ ] **Step 1: Add migration v8**

In `internal/db/db.go`, find the v7 migration (around line 795) and add v8 after it:

```go
// v8: add mode to pipeline_runs
if _, err := db.Exec(`ALTER TABLE pipeline_runs ADD COLUMN mode TEXT NOT NULL DEFAULT 'standard'`); err != nil {
    // Column may already exist, ignore error
    log.Printf("migration v8 (mode column): %v", err)
}
```

- [ ] **Step 2: Add Mode field to PipelineRun model**

In `internal/models/models.go`, update the `PipelineRun` struct (around line 823):

```go
type PipelineRun struct {
    ID          string     `json:"id" db:"id"`
    ProjectID   string     `json:"project_id" db:"project_id"`
    Mode        string     `json:"mode" db:"mode"`
    Status      string     `json:"status" db:"status"`
    Stage       string     `json:"stage,omitempty" db:"stage"`
    Error       string     `json:"error,omitempty" db:"error"`
    StartedAt   time.Time  `json:"started_at" db:"started_at"`
    CompletedAt *time.Time `json:"completed_at,omitempty" db:"completed_at"`
    CreatedAt   time.Time  `json:"created_at" db:"created_at"`
}
```

- [ ] **Step 3: Update CreatePipelineRun query**

In `internal/db/queries.go`, find `CreatePipelineRun` (around line 1544) and add `mode` to the INSERT:

```go
func (q *Queries) CreatePipelineRun(run *models.PipelineRun) error {
    _, err := q.db.Exec(
        `INSERT INTO pipeline_runs (id, project_id, mode, status, started_at) VALUES (?, ?, ?, ?, ?)`,
        run.ID, run.ProjectID, run.Mode, run.Status, run.StartedAt,
    )
    return err
}
```

- [ ] **Step 4: Add CountPipelineRunsByProject query**

In `internal/db/queries.go`, add after `ListPipelineRunsByProject`:

```go
func (q *Queries) CountPipelineRunsByProject(projectID string) (int, error) {
    var count int
    err := q.db.QueryRow(`SELECT COUNT(*) FROM pipeline_runs WHERE project_id = ?`, projectID).Scan(&count)
    return count, err
}
```

- [ ] **Step 5: Add ListPipelineRunsByProjectPaginated query**

```go
func (q *Queries) ListPipelineRunsByProjectPaginated(projectID string, limit, offset int) ([]*models.PipelineRun, error) {
    rows, err := q.db.Query(
        `SELECT id, project_id, mode, status, stage, error, started_at, completed_at, created_at
         FROM pipeline_runs WHERE project_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`,
        projectID, limit, offset,
    )
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var runs []*models.PipelineRun
    for rows.Next() {
        r := &models.PipelineRun{}
        if err := rows.Scan(&r.ID, &r.ProjectID, &r.Mode, &r.Status, &r.Stage, &r.Error, &r.StartedAt, &r.CompletedAt, &r.CreatedAt); err != nil {
            return nil, err
        }
        runs = append(runs, r)
    }
    return runs, rows.Err()
}
```

- [ ] **Step 6: Verify build compiles**

Run: `cd /Users/kun/DEV/p0m32kun && go build ./...`
Expected: No errors

- [ ] **Step 7: Commit**

```bash
git add internal/db/db.go internal/models/models.go internal/db/queries.go
git commit -m "feat(scan): add mode field to pipeline_runs with pagination support"
```

---

### Task 2: Fix Pipeline Task Linking

**Files:**
- Modify: `internal/workflow/pipeline.go`

- [ ] **Step 1: Set RunID in createAndRunTask**

In `pipeline.go`, find `createAndRunTask` (around line 954). Change the task creation to include `RunID`:

```go
func (p *Pipeline) createAndRunTask(ctx context.Context, tool string, args []string) (*models.ScanTask, []byte, error) {
    taskID := util.GenerateID()
    now := time.Now().UTC()
    task := &models.ScanTask{
        ID:              taskID,
        ProjectID:       p.projectID,
        RunID:           &p.runID,
        Tool:            tool,
        CommandTemplate: strings.Join(args, " "),
        Status:          models.TaskCreated,
        CreatedAt:       now,
    }
    p.queries.CreateScanTask(task)
    p.runner.Run(ctx, task.ID)
    stdout, err := p.readTaskStdout(task.ID)
    return task, stdout, nil
}
```

- [ ] **Step 2: Verify build compiles**

Run: `cd /Users/kun/DEV/p0m32kun && go build ./...`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add internal/workflow/pipeline.go
git commit -m "feat(scan): link pipeline tasks to run via RunID"
```

---

### Task 3: New Unified Scan Endpoint

**Files:**
- Modify: `internal/api/pipeline_handlers.go`
- Modify: `internal/api/server.go`

- [ ] **Step 1: Add ScanMode type and config builder**

At the top of `pipeline_handlers.go`, add the mode-based config builder:

```go
// ScanMode represents a predefined scan configuration
type ScanMode string

const (
    ScanModeQuick    ScanMode = "quick"
    ScanModeStandard ScanMode = "standard"
    ScanModeDeep     ScanMode = "deep"
    ScanModeCustom   ScanMode = "custom"
)

// buildConfigForMode returns a PipelineConfig preset for the given scan mode
func buildConfigForMode(mode ScanMode, base models.PipelineConfig) models.PipelineConfig {
    switch mode {
    case ScanModeQuick:
        return models.PipelineConfig{
            EnableFOFA:          false,
            EnableSubfinder:     false,
            EnableCDNFilter:     false,
            PortRange:           "top-1000",
            PortScanTimeout:     10,
            PortScanConcurrency: 25,
            EnableNerva:         false,
            EnableNuclei:        false,
        }
    case ScanModeStandard:
        return models.PipelineConfig{
            EnableFOFA:          false,
            EnableSubfinder:     true,
            SubfinderTimeout:    30,
            DNSConcurrency:      50,
            DNSTimeout:          5,
            EnableCDNFilter:     false,
            PortRange:           "top-1000",
            PortScanTimeout:     10,
            PortScanConcurrency: 25,
            EnableNerva:         true,
            NervaTimeout:        10,
            NervaConcurrency:    10,
            EnableNuclei:        true,
            NucleiRateLimit:     150,
            NucleiConcurrency:   25,
        }
    case ScanModeDeep:
        return models.PipelineConfig{
            EnableFOFA:          base.EnableFOFA,
            FofaResultLimit:     base.FofaResultLimit,
            FofaConcurrency:     base.FofaConcurrency,
            EnableSubfinder:     true,
            SubfinderTimeout:    60,
            DNSConcurrency:      100,
            DNSTimeout:          10,
            EnableCDNFilter:     true,
            PortRange:           base.PortRange,
            PortScanTimeout:     15,
            PortScanConcurrency: 50,
            EnableNerva:         true,
            NervaTimeout:        15,
            NervaConcurrency:    20,
            EnableNuclei:        true,
            NucleiRateLimit:     300,
            NucleiConcurrency:   50,
        }
    case ScanModeCustom:
        return base
    default:
        return models.DefaultPipelineConfig()
    }
}
```

- [ ] **Step 2: Add handleCreateScan handler**

Add this handler function in `pipeline_handlers.go`:

```go
func (s *Server) handleCreateScan(w http.ResponseWriter, r *http.Request) {
    projectID := r.PathValue("id")
    if projectID == "" {
        http.Error(w, `{"error":"missing project id"}`, http.StatusBadRequest)
        return
    }

    var req struct {
        Mode ScanMode `json:"mode"`
        Name string   `json:"name"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
        return
    }

    if req.Mode == "" {
        req.Mode = ScanModeStandard
    }

    project, err := s.queries.GetProject(projectID)
    if err != nil {
        http.Error(w, `{"error":"project not found"}`, http.StatusNotFound)
        return
    }

    // Build config based on mode
    baseConfig := models.DefaultPipelineConfig()
    if project.PipelineConfig != "" {
        json.Unmarshal([]byte(project.PipelineConfig), &baseConfig)
    }
    cfg := buildConfigForMode(req.Mode, baseConfig)

    // Create pipeline run with mode
    runID := util.GenerateID()
    now := time.Now().UTC()
    run := &models.PipelineRun{
        ID:        runID,
        ProjectID: projectID,
        Mode:      string(req.Mode),
        Status:    "running",
        StartedAt: now,
    }
    if err := s.queries.CreatePipelineRun(run); err != nil {
        http.Error(w, `{"error":"failed to create run"}`, http.StatusInternalServerError)
        return
    }

    // Launch pipeline in background
    go func() {
        ctx := context.Background()
        pipeline := workflow.NewPipeline(s.queries, s.worker, s.scopeEng, s.dataDir).
            WithConfig(cfg).
            WithRunID(runID).
            WithStageCallback(func(event models.PipelineRunStage) {
                s.broadcastProjectSSE(projectID, "pipeline_stage_change", event)
            })

        // Set FOFA credentials if deep mode and configured
        if req.Mode == ScanModeDeep && project.FofaEmail != "" && project.FofaAPIKey != "" {
            pipeline.WithFOFA(project.FofaEmail, project.FofaAPIKey)
        }

        err := pipeline.Run(ctx, projectID)
        if err != nil {
            s.queries.UpdatePipelineRunError(runID, err.Error())
        }
        s.queries.UpdatePipelineRunCompleted(runID)
        s.broadcastProjectSSE(projectID, "pipeline_complete", map[string]string{"run_id": runID})
    }()

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusAccepted)
    json.NewEncoder(w).Encode(map[string]string{
        "run_id": runID,
        "status": "accepted",
        "mode":   string(req.Mode),
    })
}
```

- [ ] **Step 3: Register new route in server.go**

In `server.go`, add the new route in the route registration section (around line 155-218):

```go
// Unified scan
mux.HandleFunc("POST /projects/{id}/scan", s.handleCreateScan)
```

- [ ] **Step 4: Add list pipeline runs handler with pagination**

Add a new handler that lists pipeline runs with pagination (compatible with the frontend's `PaginatedResponse` format):

```go
func (s *Server) handleListScanRuns(w http.ResponseWriter, r *http.Request) {
    projectID := r.PathValue("id")
    if projectID == "" {
        http.Error(w, `{"error":"missing project id"}`, http.StatusBadRequest)
        return
    }

    limit, offset := parsePagination(r)

    total, err := s.queries.CountPipelineRunsByProject(projectID)
    if err != nil {
        http.Error(w, `{"error":"failed to count runs"}`, http.StatusInternalServerError)
        return
    }

    runs, err := s.queries.ListPipelineRunsByProjectPaginated(projectID, limit, offset)
    if err != nil {
        http.Error(w, `{"error":"failed to list runs"}`, http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "data": runs,
        "pagination": map[string]interface{}{
            "total":  total,
            "limit":  limit,
            "offset": offset,
        },
    })
}
```

Register this route:
```go
mux.HandleFunc("GET /projects/{id}/scan/runs", s.handleListScanRuns)
```

- [ ] **Step 5: Verify build compiles**

Run: `cd /Users/kun/DEV/p0m32kun && go build ./...`
Expected: No errors

- [ ] **Step 6: Commit**

```bash
git add internal/api/pipeline_handlers.go internal/api/server.go
git commit -m "feat(scan): add unified POST /projects/{id}/scan endpoint with mode support"
```

---

### Task 4: Frontend API Client — Add Scan Methods

**Files:**
- Modify: `frontend/src/lib/api.ts`

- [ ] **Step 1: Add PipelineRun interface with mode**

Update the existing `PipelineRunStage` interface area to add/update `PipelineRun`:

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

- [ ] **Step 2: Add scan API methods**

Add these methods to the `api` object (after the existing pipeline methods):

```typescript
// Unified scan
async createScan(
  projectId: string,
  data: { mode: string; name?: string },
  signal?: AbortSignal
): Promise<{ run_id: string; status: string; mode: string }> {
  const res = await fetch(`${getApiBase()}/projects/${projectId}/scan`, {
    method: "POST",
    headers: { "Content-Type": "application/json", ...authHeaders() },
    body: JSON.stringify(data),
    signal,
  });
  if (!res.ok) throw await classifyError(res);
  return res.json();
},

async listScanRuns(
  projectId: string,
  pagination?: { limit: number; offset: number },
  signal?: AbortSignal
): Promise<PaginatedResponse<PipelineRun>> {
  const params = new URLSearchParams();
  if (pagination) {
    params.set("limit", String(pagination.limit));
    params.set("offset", String(pagination.offset));
  }
  const qs = params.toString();
  const url = `${getApiBase()}/projects/${projectId}/scan/runs${qs ? "?" + qs : ""}`;
  const res = await fetch(url, { headers: authHeaders(), signal });
  if (!res.ok) throw await classifyError(res);
  return res.json();
},

async getScanRun(
  projectId: string,
  runId: string,
  signal?: AbortSignal
): Promise<PipelineRun> {
  const res = await fetch(`${getApiBase()}/projects/${projectId}/pipeline/runs/${runId}`, {
    headers: authHeaders(),
    signal,
  });
  if (!res.ok) throw await classifyError(res);
  return res.json();
},
```

- [ ] **Step 3: Verify TypeScript compiles**

Run: `cd /Users/kun/DEV/p0m32kun/frontend && npx tsc --noEmit`
Expected: No new errors (pre-existing errors may remain)

- [ ] **Step 4: Commit**

```bash
git add frontend/src/lib/api.ts
git commit -m "feat(scan): add unified scan API methods to frontend client"
```

---

### Task 5: Frontend — New ScanModal Component

**Files:**
- Create: `frontend/src/components/ScanModal.tsx`

- [ ] **Step 1: Create ScanModal component**

```tsx
import { useState } from "react";
import Modal from "./Modal";
import { Button } from "./Button";

interface ScanModeOption {
  id: string;
  icon: React.ReactNode;
  name: string;
  description: string;
  tools: string;
  duration: string;
}

const SCAN_MODES: ScanModeOption[] = [
  {
    id: "quick",
    icon: (
      <svg className="w-6 h-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
        <path d="M13 2L3 14h9l-1 8 10-12h-9l1-8z" />
      </svg>
    ),
    name: "快速侦察",
    description: "快速了解目标暴露面",
    tools: "naabu → httpx",
    duration: "1-3 分钟",
  },
  {
    id: "standard",
    icon: (
      <svg className="w-6 h-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
        <circle cx="11" cy="11" r="8" />
        <path d="M21 21l-4.35-4.35" />
      </svg>
    ),
    name: "标准扫描",
    description: "常规安全评估",
    tools: "subfinder → naabu → nerva → httpx → nuclei",
    duration: "5-15 分钟",
  },
  {
    id: "deep",
    icon: (
      <svg className="w-6 h-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
        <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
      </svg>
    ),
    name: "深度扫描",
    description: "全面安全审计",
    tools: "FOFA → subfinder → cdncheck → naabu → nerva → httpx → nuclei",
    duration: "15-30 分钟",
  },
  {
    id: "custom",
    icon: (
      <svg className="w-6 h-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
        <circle cx="12" cy="12" r="3" />
        <path d="M19.4 15a1.65 1.65 0 00.33 1.82l.06.06a2 2 0 010 2.83 2 2 0 01-2.83 0l-.06-.06a1.65 1.65 0 00-1.82-.33 1.65 1.65 0 00-1 1.51V21a2 2 0 01-2 2 2 2 0 01-2-2v-.09A1.65 1.65 0 009 19.4a1.65 1.65 0 00-1.82.33l-.06.06a2 2 0 01-2.83 0 2 2 0 010-2.83l.06-.06A1.65 1.65 0 004.68 15a1.65 1.65 0 00-1.51-1H3a2 2 0 01-2-2 2 2 0 012-2h.09A1.65 1.65 0 004.6 9a1.65 1.65 0 00-.33-1.82l-.06-.06a2 2 0 010-2.83 2 2 0 012.83 0l.06.06A1.65 1.65 0 009 4.68a1.65 1.65 0 001-1.51V3a2 2 0 012-2 2 2 0 012 2v.09a1.65 1.65 0 001 1.51 1.65 1.65 0 001.82-.33l.06-.06a2 2 0 012.83 0 2 2 0 010 2.83l-.06.06A1.65 1.65 0 0019.32 9a1.65 1.65 0 001.51 1H21a2 2 0 012 2 2 2 0 01-2 2h-.09a1.65 1.65 0 00-1.51 1z" />
      </svg>
    ),
    name: "自定义",
    description: "高级用户手动配置工具组合",
    tools: "自选工具组合",
    duration: "视配置",
  },
];

interface ScanModalProps {
  open: boolean;
  onClose: () => void;
  onConfirm: (mode: string) => void;
  loading?: boolean;
  targetCount?: number;
}

export function ScanModal({ open, onClose, onConfirm, loading, targetCount }: ScanModalProps) {
  const [selectedMode, setSelectedMode] = useState<string>("standard");

  return (
    <Modal
      open={open}
      onClose={onClose}
      title="新建扫描"
      size="lg"
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={loading}>
            取消
          </Button>
          <Button
            variant="primary"
            onClick={() => onConfirm(selectedMode)}
            loading={loading}
          >
            开始扫描
          </Button>
        </>
      }
    >
      <div className="grid grid-cols-2 gap-3">
        {SCAN_MODES.map((mode) => (
          <button
            key={mode.id}
            onClick={() => setSelectedMode(mode.id)}
            className={`text-left p-4 rounded-2xl border transition-all duration-200 ${
              selectedMode === mode.id
                ? "border-brand-primary bg-brand-primary/10 shadow-[0_0_0_1px_rgba(47,129,247,0.3)]"
                : "border-white/[0.08] bg-white/[0.02] hover:bg-white/[0.04] hover:border-white/[0.12]"
            }`}
          >
            <div className={`mb-2 ${selectedMode === mode.id ? "text-brand-primary" : "text-text-secondary"}`}>
              {mode.icon}
            </div>
            <h4 className="text-sm font-semibold text-text-primary mb-0.5">{mode.name}</h4>
            <p className="text-xs text-text-tertiary mb-2">{mode.description}</p>
            <p className="text-[11px] text-text-quaternary font-mono mb-1">{mode.tools}</p>
            <p className="text-[11px] text-text-quaternary">{mode.duration}</p>
          </button>
        ))}
      </div>
      {targetCount !== undefined && (
        <p className="text-xs text-text-tertiary mt-3 text-center">
          目标: 已选 {targetCount} 个 target
        </p>
      )}
    </Modal>
  );
}
```

- [ ] **Step 2: Export from components/index.ts**

Add to `frontend/src/components/index.ts`:
```tsx
export { ScanModal } from "./ScanModal";
```

- [ ] **Step 3: Verify TypeScript compiles**

Run: `cd /Users/kun/DEV/p0m32kun/frontend && npx tsc --noEmit`

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/ScanModal.tsx frontend/src/components/index.ts
git commit -m "feat(scan): add ScanModal component with 4 scan modes"
```

---

### Task 6: Frontend — Rewrite RunsPage

**Files:**
- Modify: `frontend/src/pages/RunsPage.tsx`

This is the largest task. The RunsPage needs to:
1. List from `pipeline_runs` via `api.listScanRuns`
2. Use `ScanModal` instead of `CreateRunModal`
3. Show mode badge on each run
4. Show stage progress when a run is selected
5. Listen to SSE for `pipeline_stage_change` and `pipeline_complete`

- [ ] **Step 1: Update imports and add ScanModal**

Replace the import section at the top of `RunsPage.tsx`. Add:
```typescript
import { ScanModal } from "../components/ScanModal";
```

Update the `PipelineRun` import if needed to match the new interface with `mode` field.

- [ ] **Step 2: Replace run listing to use pipeline runs**

Replace the `loadRuns` function to use `api.listScanRuns`:

```typescript
const loadRuns = useCallback(async () => {
  if (!projectId) return;
  try {
    const res = await api.listScanRuns(projectId, { limit: 50, offset: 0 });
    setRuns(res.data || []);
  } catch {
    // fallback
  }
}, [projectId]);
```

- [ ] **Step 3: Replace run details loading**

Replace `loadRunDetails` to use pipeline run + stages + tasks:

```typescript
const loadRunDetails = useCallback(async (runId: string) => {
  if (!projectId) return;
  try {
    const [runDetail, stagesRes, tasksRes] = await Promise.all([
      api.getScanRun(projectId, runId),
      api.listPipelineRunStages(projectId, runId).catch(() => ({ stages: [] })),
      api.getRunTasks(runId).catch(() => []),
    ]);
    setSelectedRunDetail(runDetail);
    setStages(stagesRes.stages || []);
    setTasks(Array.isArray(tasksRes) ? tasksRes : []);
  } catch {
    toast("加载详情失败", "error");
  }
}, [projectId]);
```

- [ ] **Step 4: Replace CreateRunModal with ScanModal**

Replace the create run flow:

```typescript
const [scanModalOpen, setScanModalOpen] = useState(false);
const [scanLoading, setScanLoading] = useState(false);

const handleCreateScan = async (mode: string) => {
  if (!projectId) return;
  setScanLoading(true);
  try {
    await api.createScan(projectId, { mode });
    toast("扫描已启动");
    setScanModalOpen(false);
    loadRuns();
  } catch {
    toast("启动扫描失败", "error");
  } finally {
    setScanLoading(false);
  }
};
```

- [ ] **Step 5: Update run list rendering to show mode badge**

In the run list, add a mode badge:
```tsx
const modeLabels: Record<string, string> = {
  quick: "快速侦察",
  standard: "标准扫描",
  deep: "深度扫描",
  custom: "自定义",
};

const modeColors: Record<string, string> = {
  quick: "bg-accent-teal/10 text-accent-teal border-accent-teal/20",
  standard: "bg-brand-primary/10 text-brand-primary border-brand-primary/20",
  deep: "bg-brand-purple/10 text-brand-purple border-brand-purple/20",
  custom: "bg-white/[0.06] text-text-secondary border-white/[0.08]",
};
```

- [ ] **Step 6: Update SSE event handling**

Update the SSE listener to handle `pipeline_stage_change` and `pipeline_complete` events correctly, reloading runs and updating stage progress.

- [ ] **Step 7: Remove old CreateRunModal and template loading**

Remove the old `CreateRunModal` component reference and `templates` state. Replace the "新建扫描" button to open `ScanModal`.

- [ ] **Step 8: Verify TypeScript compiles**

Run: `cd /Users/kun/DEV/p0m32kun/frontend && npx tsc --noEmit`

- [ ] **Step 9: Commit**

```bash
git add frontend/src/pages/RunsPage.tsx
git commit -m "feat(scan): rewrite RunsPage to use unified pipeline scan"
```

---

### Task 7: Frontend — Simplify ScanConfigPage

**Files:**
- Modify: `frontend/src/pages/ScanConfigPage.tsx`

- [ ] **Step 1: Remove tool enable/disable toggles**

Remove the toggle switches for `enable_fofa`, `enable_subfinder`, `enable_cdn_filter`, `enable_nerva`, `enable_nuclei`. These are now controlled by scan mode selection.

Keep only the parameter sections:
- Port range
- Concurrency settings
- Timeout settings
- FOFA credentials
- nuclei rate limit and concurrency

- [ ] **Step 2: Add explanatory text**

Add a note at the top explaining that tool selection is now handled by scan mode:
```tsx
<div className="vision-glass p-4 mb-6">
  <p className="text-sm text-text-secondary">
    工具选择已移至「新建扫描」的模式选择中。此页面仅配置运行参数。
  </p>
</div>
```

- [ ] **Step 3: Verify TypeScript compiles**

Run: `cd /Users/kun/DEV/p0m32kun/frontend && npx tsc --noEmit`

- [ ] **Step 4: Commit**

```bash
git add frontend/src/pages/ScanConfigPage.tsx
git commit -m "feat(scan): simplify ScanConfigPage — remove tool toggles, keep params"
```

---

### Task 8: Build Verification & Integration Test

**Files:** None (verification only)

- [ ] **Step 1: Verify Go build**

Run: `cd /Users/kun/DEV/p0m32kun && go build ./...`
Expected: No errors

- [ ] **Step 2: Verify frontend build**

Run: `cd /Users/kun/DEV/p0m32kun/frontend && npx vite build`
Expected: Build succeeds

- [ ] **Step 3: Start backend and test scan endpoint**

Run: `cd /Users/kun/DEV/p0m32kun && go run main.go`

Test the new endpoint:
```bash
# Create a scan
curl -X POST http://localhost:17421/projects/{PROJECT_ID}/scan \
  -H "Content-Type: application/json" \
  -d '{"mode": "quick"}'
```

Expected: `202 Accepted` with `{ run_id, status, mode }`

- [ ] **Step 4: Verify run appears in listing**

```bash
curl http://localhost:17421/projects/{PROJECT_ID}/scan/runs
```

Expected: Paginated response with the new run

- [ ] **Step 5: Final commit if fixes needed**

```bash
git add -A
git commit -m "fix(scan): integration fixes after testing"
```
