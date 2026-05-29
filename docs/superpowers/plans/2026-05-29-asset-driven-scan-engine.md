# 资产驱动扫描引擎 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将扫描执行从阶段流水线迁移为资产图 + Work(资产×动作) + 属性门控 + 收敛状态机；前端 Runs 以任务时间序为主、保留 Stage 聚合；资产页增加 Work 时间线抽屉。

**Architecture:** 新建 `internal/scanengine/`（core/work/queue/dedup/executor/stageagg）；`scan_work_items` 为调度真相；`pipeline_run_stages` 由 StageAggregator 投影；工具执行仍走 `toolrun` + `ResourceGovernor`；外网能力由 `ScanProfile.external` + 现有 `DefaultExternalPipelineConfig()` 驱动。

**Tech Stack:** Go 1.26、SQLite、React 18/TypeScript、Tauri、ProjectDiscovery CLI、现有 SSE

**Spec:** `docs/superpowers/specs/2026-05-29-asset-driven-scan-engine-design.md`

---

## File map

| 文件 | 职责 |
|------|------|
| `internal/db/migrations/00X_scan_work_items.sql` | 新表 + 列扩展 |
| `internal/models/scan_work.go` | `ScanWorkItem` 模型 |
| `internal/models/scan.go` | `PipelineRunStage` 扩展字段；`engine_state` |
| `internal/db/queries_scan_work.go` | Work CRUD |
| `internal/scanengine/core/*.go` | Asset、Action、Rules、Profile、Attrs |
| `internal/scanengine/work/store.go` | TryClaim、MarkDone、Terminal |
| `internal/scanengine/queue/priority.go` | 三级队列 |
| `internal/scanengine/dedup/run_dedup.go` | Run 级资产 dedup |
| `internal/scanengine/executor/*.go` | Batcher + httpx/nuclei/… |
| `internal/scanengine/stageagg/aggregator.go` | Work → Stage 行 + SSE |
| `internal/scanengine/engine.go` | ScanEngine 主循环 |
| `internal/api/pipeline_handlers.go` | 挂载 ScanEngine；metrics/works handlers |
| `internal/api/asset_handlers.go` | `GET /assets/{id}/works` |
| `frontend/src/lib/api.ts` | 类型 + API 方法 |
| `frontend/src/pages/RunsPage.tsx` | metrics 顶栏；任务主列表；移除时间窗 |
| `frontend/src/pages/AssetPage.tsx` | depth 列；Work 抽屉 |
| `docs/current/architecture.md` | 基线更新（A5） |

---

## Task A0: Migration + 模型（已确认设计）

**Files:**
- Create: `internal/db/migrations/014_scan_work_items.sql`
- Create: `internal/models/scan_work.go`
- Modify: `internal/models/scan.go`
- Create: `internal/db/queries_scan_work.go`
- Create: `internal/db/queries_scan_work_test.go`

- [ ] **Step 1: 写失败测试 — CreateScanWorkItem 唯一约束**

```go
// internal/db/queries_scan_work_test.go
func TestScanWorkItem_UniqueRunAssetAction(t *testing.T) {
	q, cleanup := testQueries(t)
	defer cleanup()
	runID := createTestPipelineRun(t, q)
	assetID := "asset-1"
	w := &models.ScanWorkItem{
		ID: util.GenerateID(), RunID: runID, ProjectID: "p1",
		AssetID: assetID, Action: "HTTPX_FINGERPRINT", Status: models.WorkStatusPending,
	}
	if err := q.CreateScanWorkItem(w); err != nil {
		t.Fatal(err)
	}
	w2 := *w
	w2.ID = util.GenerateID()
	err := q.CreateScanWorkItem(&w2)
	if err == nil {
		t.Fatal("expected unique violation")
	}
}
```

- [ ] **Step 2: 运行测试确认 FAIL**

Run: `go test ./internal/db/ -run TestScanWorkItem_UniqueRunAssetAction -v`  
Expected: FAIL（表/方法不存在）

- [ ] **Step 3: 添加 migration `014_scan_work_items.sql`**

内容对齐 spec §6.1–6.4（`scan_work_items`、`scan_tasks` 列、`discovery_depth`、`engine_state`）。

- [ ] **Step 4: 实现 `ScanWorkItem` 与 queries**

```go
// internal/models/scan_work.go
type WorkStatus string
const (
	WorkStatusPending  WorkStatus = "pending"
	WorkStatusRunning  WorkStatus = "running"
	WorkStatusDone     WorkStatus = "done"
	WorkStatusSkipped  WorkStatus = "skipped"
	WorkStatusFailed   WorkStatus = "failed"
)

type ScanWorkItem struct {
	ID          string     `json:"id" db:"id"`
	RunID       string     `json:"run_id" db:"run_id"`
	ProjectID   string     `json:"project_id" db:"project_id"`
	AssetID     string     `json:"asset_id" db:"asset_id"`
	Action      string     `json:"action" db:"action"`
	Status      WorkStatus `json:"status" db:"status"`
	SkipReason  string     `json:"skip_reason,omitempty" db:"skip_reason"`
	Stage       string     `json:"stage,omitempty" db:"stage"`
	Error       string     `json:"error,omitempty" db:"error"`
	StartedAt   *time.Time `json:"started_at,omitempty" db:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
}
```

- [ ] **Step 5: 运行测试 PASS**

Run: `go test ./internal/db/ -run TestScanWorkItem -v`

- [ ] **Step 6: Commit**

```bash
git add internal/db/migrations/014_scan_work_items.sql internal/models/scan_work.go internal/db/queries_scan_work.go internal/db/queries_scan_work_test.go internal/models/scan.go
git commit -m "feat(db): add scan_work_items and engine_state columns"
```

---

## Task A1: scanengine 骨架 + domain httpx→nuclei

**Files:**
- Create: `internal/scanengine/core/asset.go`, `task.go`, `attrs.go`, `rules_internal.go`
- Create: `internal/scanengine/work/store.go`, `store_test.go`
- Create: `internal/scanengine/queue/priority.go`
- Create: `internal/scanengine/dedup/run_dedup.go`
- Create: `internal/scanengine/stageagg/aggregator.go`
- Create: `internal/scanengine/engine.go`, `config.go`
- Create: `internal/scanengine/executor/batcher.go`, `httpx.go`, `nuclei.go`
- Modify: `internal/api/pipeline_handlers.go`
- Create: `internal/api/scan_metrics_handlers.go`
- Modify: `frontend/src/lib/api.ts`
- Modify: `frontend/src/pages/RunsPage.tsx`（仅 metrics 顶栏 + SSE）

- [ ] **Step 1: 写失败测试 — DeriveEligibleWorks 指纹门控**

```go
// internal/scanengine/core/rules_internal_test.go
func TestDeriveEligibleWorks_NucleiRequiresFingerprint(t *testing.T) {
	cfg := ProfileInternal{RequireFingerprint: true}
	a := &DiscoveryAsset{
		Type: AssetHTTPService, DiscoveryDepth: 0,
		Attrs: AssetAttrs{Fingerprinted: false},
	}
	works := DeriveEligibleWorks(a, cfg)
	for _, w := range works {
		if w.Action == ActionNucleiScan {
			t.Fatal("nuclei should not be eligible without fingerprint")
		}
	}
	a.Attrs.Fingerprinted = true
	works = DeriveEligibleWorks(a, cfg)
	var hasNuclei bool
	for _, w := range works {
		if w.Action == ActionNucleiScan {
			hasNuclei = true
		}
	}
	if !hasNuclei {
		t.Fatal("expected nuclei after fingerprint")
	}
}
```

- [ ] **Step 2: 实现 `core` 包（MaxDiscoveryDepth=2）与 internal profile 规则**

- [ ] **Step 3: 实现 `work.Store` — TryClaim / MarkDone / AllTerminal**

- [ ] **Step 4: 实现 `engine.ScanEngine` — seed domain、processNewAsset、scheduler 2s tick、idle→wind_down**

- [ ] **Step 5: 实现 httpx + nuclei executor（toolrun）；onWorkComplete 二次派生**

- [ ] **Step 6: 实现 `stageagg` — HTTPX→stage `httpx`，NUCLEI→`vuln`；写 DB + callback SSE**

- [ ] **Step 7: `handleGetScanRunMetrics` + pipeline 启动走 ScanEngine（feature flag `ANCHOR_SCAN_ENGINE=1`）**

```go
// internal/api/scan_metrics_handlers.go
func (s *Server) handleGetScanRunMetrics(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("runId")
	m, err := s.queries.GetScanRunMetrics(runID)
	// ...
}
```

- [ ] **Step 8: 前端 api.ts + RunsPage 顶栏 metrics（订阅 `scan_metrics`）**

- [ ] **Step 9: 单域 E2E（现有 spec + `ANCHOR_SCAN_ENGINE=1`）**

Run: `go test ./internal/scanengine/... -v`  
Run: 相关 Playwright / pipeline e2e（项目既有命令）

- [ ] **Step 10: Commit**

```bash
git commit -m "feat(scanengine): A1 domain flow with httpx/nuclei and metrics API"
```

---

## Task A2: katana/ffuf + 深度 + wind_down + Works API

**Files:**
- Modify: `internal/scanengine/core/rules_internal.go`（katana/ffuf MaxDepth=1）
- Create: `internal/scanengine/executor/katana.go`, `ffuf.go`
- Modify: `internal/scanengine/engine.go`（wind_down 仅允许 nuclei/httpx）
- Create: `internal/api/scan_work_handlers.go`
- Modify: `frontend/src/pages/AssetPage.tsx`（抽屉 + depth 列）
- Modify: `frontend/src/lib/api.ts`

- [ ] **Step 1: 写失败测试 — katana 在 depth 2 不派生**

```go
func TestDeriveEligibleWorks_KatanaMaxDepth1(t *testing.T) {
	a := &DiscoveryAsset{Type: AssetHTTPService, DiscoveryDepth: 2}
	for _, w := range DeriveEligibleWorks(a, DefaultInternalProfile()) {
		if w.Action == ActionKatanaCrawl {
			t.Fatal("katana at depth 2")
		}
	}
}
```

- [ ] **Step 2: 实现 katana/ffuf executor + URL 解析 → processNewAsset**

- [ ] **Step 3: dedup 与 `asset/normalizer` 对齐；同 URL 不 +depth**

- [ ] **Step 4: `GET .../runs/{runId}/works` 与 `GET /assets/{id}/works?run_id=`**

- [ ] **Step 5: AssetPage 抽屉组件 `AssetWorkDrawer.tsx`**

```tsx
// 按 started_at 升序展示 ScanWorkItem；skipped 显示 skip_reason
```

- [ ] **Step 6: Commit**

```bash
git commit -m "feat(scanengine): A2 crawl/ffuf depth limits and work timeline API"
```

---

## Task A3: external profile + 被动种子

**Files:**
- Create: `internal/scanengine/core/rules_external.go`, `profile.go`
- Create: `internal/scanengine/seed/passive.go`
- Modify: `internal/scanengine/engine.go`
- Modify: `internal/models/engine.go`（确认 `DefaultExternalPipelineConfig` 字段）
- Create: `internal/scanengine/engine_external_test.go`

- [ ] **Step 1: 写失败测试 — CDN IP PORT_SCAN skipped**

```go
func TestPortScan_SkippedOnCDN(t *testing.T) {
	a := &DiscoveryAsset{Type: AssetIP, Attrs: AssetAttrs{Alive: ptr(true), IsCDN: ptr(true)}}
	works := DeriveEligibleWorks(a, ProfileExternal{SkipPortscanOnCDN: true})
	for _, w := range works {
		if w.Action == ActionPortScan {
			t.Fatal("should not enqueue port scan on CDN")
		}
	}
}
```

- [ ] **Step 2: 实现 passive 种子（FOFA/Hunter/Quake fail-soft、crt、gau）→ processNewAsset**

- [ ] **Step 3: external 规则启用集对齐 spec §9 与 `2026-05-25` §6.2 默认值**

- [ ] **Step 4: `go test ./internal/scanengine/... -run External -v`**

- [ ] **Step 5: Commit**

```bash
git commit -m "feat(scanengine): A3 external profile and passive seed injectors"
```

---

## Task A4: 切换主路径 + Runs 任务主列表

**Files:**
- Modify: `internal/api/pipeline_handlers.go`（默认 ScanEngine；旧 Pipeline 用 `ANCHOR_LEGACY_PIPELINE=1` 保留一版）
- Modify: `frontend/src/pages/RunsPage.tsx`
- Modify: `frontend/src/pages/RunsPage.test.tsx`
- Delete or deprecate: `tasksInStage` 时间窗逻辑

- [ ] **Step 1: 写失败测试 — mergeStageEvent 仍兼容 work 计数字段**

```ts
// RunsPage.test.tsx
it("merges stage event with work counts", () => {
  const next = mergeStageEvent([], {
    stage: "httpx",
    status: "running",
    work_total: 10,
    work_done: 3,
  });
  expect(next[0].work_total).toBe(10);
});
```

- [ ] **Step 2: 任务列表按 `started_at` 排序；stage 点击过滤**

- [ ] **Step 3: wind_down Banner**

- [ ] **Step 4: 移除 `STAGE_TOOL_ALLOWLIST` 时间窗路径**

- [ ] **Step 5: 全量 E2E**

- [ ] **Step 6: Commit**

```bash
git commit -m "feat(ui): A4 runs task-primary list and legacy pipeline flag"
```

---

## Task A5: 文档与收尾

**Files:**
- Modify: `docs/current/architecture.md`
- Modify: `docs/current/plan.md`
- Modify: `internal/api/README.md`
- Modify: `docs/superpowers/specs/2026-05-25-external-scan-pipeline-design.md`（文首 superseded 注记）

- [ ] **Step 1: architecture.md 增加「资产驱动扫描」章节（双轨可观测性、表、收敛）**

- [ ] **Step 2: API README 新路由与 Server 字段（若 handlers 引用新 deps）**

- [ ] **Step 3: `go vet ./internal/scanengine/ ./internal/api/`**

- [ ] **Step 4: Commit**

```bash
git commit -m "docs: asset-driven scan engine baseline"
```

---

## Spec coverage self-review

| Spec § | Task |
|--------|------|
| §2 Work + 两拍派生 | A1, A2 |
| §2.5 深度 katana/ffuf≤1 | A2 |
| §2.6 收敛 wind_down | A1, A2 |
| §4 Stage 聚合 | A1 |
| §6 数据模型 | A0 |
| §7 API/SSE | A1 metrics, A2 works, A4 SSE |
| §8 前端 | A1 metrics, A2 抽屉, A4 Runs |
| §9 外网 profile | A3 |
| §10 分期 | A0–A5 |

---

## 执行方式（任选）

**Plan 已保存至** `docs/superpowers/plans/2026-05-29-asset-driven-scan-engine.md`。

1. **Subagent-Driven（推荐）** — 每 Task 派生子 agent，Task 间人工/自动 review  
2. **Inline Execution** — 本会话按 A0→A5 连续实现，每期 checkpoint

请告知从 **A0** 开始实现，以及选用哪种执行方式。
