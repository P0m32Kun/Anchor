---
status: proposed
source_of_truth: false
owner: kun
created: 2026-06-13
scope: bounty-watch
---

# Bounty Watch — Tasks

> **执行约定**：每完成一个子阶段 → 跑该阶段验收命令 → 勾选 checkbox → 再进入下一阶段。  
> **完成定义**：对应 REQ 验收信号全绿 + E2E/functional-test 登记项通过。  
> **PR 策略**：建议每 BW 阶段 1～2 个 PR，避免超大 diff。

## 阶段总览

| 阶段 | 主题 | REQ | 建议工期 | 状态 |
|------|------|-----|---------|------|
| BW0 | 可选 Scope 边界 | REQ-BW0 | 3～5 天 | 未开始 |
| BW1 | Bounty Preset + Spoor | REQ-BW1 | 2～3 天 | 未开始 |
| BW2 | Signal Inbox | REQ-BW2 | 5～7 天 | 未开始 |
| BW3 | Delta API + 增量扫描 | REQ-BW3 | 4～5 天 | 未开始 |
| BW4 | Watch Mode | REQ-BW4 | 5～7 天 | 未开始 |
| BW5 | 边缘发现补强 | REQ-BW5 | 5～7 天 | 未开始 |

**MVP 里程碑**：BW0 + BW1 + BW2 完成 → 可「领赏金目标 → 限定 scope → 扫一遍 → 看 inbox」。

---

## 阶段 BW0：可选 Scope 边界

### BW0.1 DB 迁移

- [x] `internal/db/v35.go`：`projects.scope_boundary_mode DEFAULT 'off'`
- [x] `findings.scope_status DEFAULT 'in_scope'`
- [x] `internal/models/project.go` 增加 `ScopeBoundaryMode` + constants
- [x] `internal/models/finding.go` 增加 `ScopeStatus` + constants
- [x] `internal/db/queries_project.go` 更新所有查询 + `UpdateProjectScopeBoundaryMode`
- [x] `internal/db/queries_finding.go` 更新列定义、插入参数、扫描函数
- [x] `internal/db/ensure.go` 安全网
- [x] 验收: REQ-BW0（DB 行）

### BW0.2 Scope 评估逻辑

- [x] `internal/scope/scope.go`：`EvaluateBoundary(target, rules, mode)` + `EvaluateBoundaryForProject`
- [x] strict：exclude 优先 → include 必须命中（无 include 时仅 exclude）
- [x] 复用 `matchDomainRule` / `matchIP` / `matchURL`
- [x] `internal/scope/scope_test.go`：13 个 EvaluateBoundary 测试用例
- [x] 验收: REQ-BW0（Unit FT-SCOPE-BW-01）— 174 tests passed

### BW0.3 Gate A — Seed 过滤

- [x] `internal/scanengine/seed/boundary_filter.go`：`FilterSeedsByBoundary(seeds, eng, rules, mode)`
- [x] `pipeline_handlers.go` 在 strict 时调用 Gate A
- [x] log 跳过原因（fail-soft，不阻断 run）
- [x] `boundary_filter_test.go`：7 个测试用例
- [x] 验收: REQ-BW0

### BW0.4 Gate B — Engine Work 门控

- [x] `engine.scopeBoundaryMode` 字段，构造时从 project 获取
- [x] `off`：保持现网 exclude-only（excludeMgr + 项目 exclude 规则）
- [x] `strict`：调用 `EvaluateBoundary`；deny 时 log Gate B 原因
- [x] 验收: REQ-BW0

### BW0.5 Gate C — Finding scope 标记

- [x] `NucleiPersister.WithScope(eng, mode)` 注入 scope 引擎
- [x] `evaluateScopeStatus(projectID, host)` 检查 host 是否 in scope
- [x] Nuclei persist 时设置 `ScopeStatus`
- [x] Spoor persist 时检查 asset host 并设置 `ScopeStatus`
- [x] 验收: REQ-BW0

### BW0.6 API + 项目设置 UI

- [x] `PATCH /projects/{id}/scope-boundary-mode` 路由 + handler
- [x] `GET /projects/{id}` 返回当前 mode（自动包含）
- [x] `service.ProjectService.UpdateScopeBoundaryMode` + Repository + Mock
- [x] 更新 `internal/api/README.md`
- [x] 前端 `ProjectSettingsPage.tsx`：Scope 边界开关 + 模式说明 + 管理规则链接
- [x] 前端 API 客户端 `updateProjectScopeBoundaryMode` + `Project.scope_boundary_mode`
- [x] 前端 Navbar 添加「项目设置」导航链接
- [x] 前端路由 `App.tsx` 添加 `/projects/:projectId/settings`
- [ ] ScanModal 只读展示 scope 状态（defer，非 MVP 阻断项）
- [x] 验收: REQ-BW0（API + UI）

### BW0.7 E2E

- [x] 新建 `frontend/e2e/tests/scope-boundary.spec.ts`（E2E-SCOPE-BW-01）
- [x] 6 个测试用例：页面渲染、strict 切换、off 切换、API 持久化、侧边栏导航、规则链接
- [x] `frontend/e2e/fixtures/api-helpers.ts` 添加 `updateProjectScopeBoundaryMode` + `createScopeRule`
- [x] `docs/functional-test.md` 登记 FT-SCOPE-BW-01 / E2E-SCOPE-BW-01 状态更新
- [x] 验收: REQ-BW0

**BW0 阶段验收命令**：

```bash
go test ./internal/scope/... -v -count=1
go vet ./internal/api/ ./internal/scanengine/seed/
# Docker E2E
cd frontend && npx playwright test e2e/tests/scope-boundary.spec.ts e2e/tests/external-scan-conv.spec.ts
```

**PR**：BW0a（BW0.1–BW0.3）→ BW0b（BW0.4–BW0.7）

---

## 阶段 BW1：Bounty Preset + Spoor 独立

### BW1.1 PipelineConfig

- [x] `EnableSpoor bool` 字段 + JSON 默认值
- [x] `DefaultBountyPipelineConfig()` in `internal/models/engine.go`
- [x] `internal/models/engine_test.go` bounty 默认值断言（`TestDefaultBountyPipelineConfig`）
- [x] 验收: REQ-BW1

### BW1.2 Profile + scan.config

- [x] `profile_config.go`：`ActionSpoorScan` 读 `EnableSpoor`（fallback 到 `EnableKatana`）
- [x] `rules_external.go`：增加 `ActionSpoorScan` rule + `isHTTPServiceOrPathHighValue` precondition
- [x] `configs/scan.config.yaml` presets.bounty（`enable_spoor: true`）
- [x] `scanconfig/config.go`：bounty preset 支持 + `GET /scan/defaults` 返回 bounty
- [x] `pipeline_handlers.go`：`presetDefaults` + `buildConfigForMode` 支持 bounty
- [x] `profile_config_test.go`：Spoor 独立性测试（`TestProfileFromConfig_SpoorIndependentOfKatana` + `TestProfileFromConfig_SpoorDisabledWhenBothOff`）
- [x] 验收: REQ-BW1（FT-SPOOR-BW-01）

### BW1.3 前端 ScanModal

- [x] `ScanMode` 类型添加 `"bounty"`
- [x] 扫描模式增加「赏金监控」选项（Sparkles 图标 + 黄色主题）
- [x] `frontend/src/lib/api.ts`：`PipelineConfig.enable_spoor` + `DEFAULT_BOUNTY_PIPELINE_CONFIG`
- [x] Step 2 显示 Spoor 独立开关（bounty 模式下可见）
- [x] 被动搜索垃圾过滤对 bounty 模式可见
- [x] 验收: REQ-BW1

### BW1.4 E2E

- [x] `frontend/e2e/tests/bounty-preset.spec.ts`（E2E-BOUNTY-PRESET-01）
- [x] 3 个测试用例：ScanModal 赏金选项、Spoor 开关、API bounty preset
- [x] `docs/functional-test.md` 登记 FT-SPOOR-BW-01 / E2E-BOUNTY-PRESET-01 状态更新
- [x] 验收: REQ-BW1

**BW1 阶段验收命令**：

```bash
go test ./internal/scanengine/core/... ./internal/models/... -v -count=1
cd frontend && npx playwright test e2e/tests/bounty-preset.spec.ts
```

**PR**：BW1（单 PR）

---

## 阶段 BW2：Signal Inbox

### BW2.1 DB + Model

- [x] `internal/db/v36.go`：`signals` 表（含索引）
- [x] `internal/models/signal.go`：Signal + SignalKind + SignalStatus
- [x] `internal/db/queries_signal.go`：CRUD + Upsert + List + Count
- [x] 验收: REQ-BW2

### BW2.2 写入管道

- [x] `internal/signal/upsert.go`：`UpsertFromFinding` / `UpsertFromSpoorFinding` / `UpsertFromAsset` / `UpsertFromEndpoint`
- [x] hook：engine.go Nuclei persist 后 `upsertSignalsFromRecentFindings`
- [x] hook：engine.go Spoor persist 后 `UpsertFromSpoorFinding`
- [x] strict scope：`ScopeOutOfScope` 的 finding 不写 inbox
- [x] 验收: REQ-BW2

### BW2.3 评分

- [x] `internal/signal/scorer.go`：`ScoreComponents` + 权重计算（severity 40% / novelty 25% / scope 20% / edge 15%）
- [x] `internal/signal/scorer_test.go`：14 个测试用例
- [x] 验收: REQ-BW2

### BW2.4 API Handlers

- [x] `internal/api/signal_handlers.go`：`handleListSignals` / `handleGetSignal` / `handlePatchSignal` / `handleCountSignals`
- [x] 路由注册 + `internal/api/README.md` 同步
- [x] 验收: REQ-BW2

### BW2.5 前端 Inbox

- [x] `SignalInboxPage.tsx`：列表（score、title、source kind、severity、scope status、dismiss/pin）
- [x] `api.ts`：Signal 接口 + `listSignals` / `countSignals` / `updateSignalStatus`
- [x] 侧边栏「Signal Inbox」入口 + 路由 `/projects/:projectId/signals`
- [x] 筛选：全部 / 新信号 / 已置顶
- [x] 验收: REQ-BW2

### BW2.6 E2E

- [x] `frontend/e2e/tests/signal-inbox.spec.ts`（E2E-INBOX-01）
- [x] 5 个测试用例：页面渲染、侧边栏导航、筛选切换、刷新按钮、API 验证
- [x] `docs/functional-test.md` 登记
- [x] 验收: REQ-BW2

**BW2 阶段验收命令**：

```bash
go test ./internal/signal/... ./internal/bounty/... ./internal/db/... -run Signal -v
go vet ./internal/api/
cd frontend && npx playwright test e2e/tests/signal-inbox.spec.ts
```

**PR**：BW2a（BW2.1–BW2.3）→ BW2b（BW2.4–BW2.6）

---

## 阶段 BW3：Delta API + 增量扫描

### BW3.1 查询 API

- [x] `ListAssetsByProjectSince` / `ListFindingsByProjectSince` / `ListSignalsByProjectSince`
- [x] Handler query params: `first_seen_after`, `created_after`, `since`
- [x] 验收: REQ-BW3

### BW3.2 Skip stable assets

- [x] `SkipStableAssetDays` in PipelineConfig；bounty 默认 7
- [x] `shouldSkipStableAsset` + `processNewAsset` 中跳过 HTTPX
- [x] 验收: REQ-BW3

### BW3.3 Run summary

- [x] `handleGetRunSummary`：统计 new_findings + new_signals
- [x] `GET /projects/{id}/pipeline/runs/{runId}/summary` 路由
- [x] Runs 页摘要卡片（新发现 + 新信号）
- [x] `api.ts`：`getRunSummary` 方法
- [x] 验收: REQ-BW3

### BW3.4 E2E

- [x] `frontend/e2e/tests/scan-delta.spec.ts`（E2E-DELTA-01）
- [x] 5 个测试用例：assets delta、findings delta、signals delta、run summary、invalid param
- [x] 验收: REQ-BW3

**BW3 阶段验收命令**：

```bash
go test ./internal/scanengine/... ./internal/db/... -v -count=1
cd frontend && npx playwright test e2e/tests/scan-delta.spec.ts
```

**PR**：BW3（单 PR）

---

## 阶段 BW4：Watch Mode

### BW4.1 Project watch 字段

- [x] DB v37：`watch_enabled`, `watch_interval_hours`, `watch_passive_only`, `watch_last_tick_at`
- [x] `Project` model + `queries_project.go` 更新
- [x] `PATCH /projects/{id}/watch-config` API
- [x] `service.ProjectService.UpdateWatchConfig` + Repository + Mock
- [x] 验收: REQ-BW4

### BW4.2 Scheduler

- [x] `internal/watch/scheduler.go`：`WatchStore` 接口 + `Scheduler`
- [x] `NewSchedulerFromQueries` + Server 启动注册
- [x] `isDue` 逻辑 + `tick` 触发
- [x] `triggerWatchRun`：passive-only config overlay + `watch_passive` mode
- [x] `scheduler_test.go`：4 个测试用例
- [x] 验收: REQ-BW4

### BW4.3 SSE 事件

- [x] engine `onNewAsset` 回调 + `SetOnNewAsset` 方法
- [x] `asset.new` SSE 事件广播（含 asset_id/value/type）
- [x] 主扫描 + watch 扫描均设置回调
- [x] 验收: REQ-BW4

### BW4.4 UI + E2E

- [x] `ProjectSettingsPage.tsx`：Watch Mode 配置卡片（开关/间隔/passive-only/last tick）
- [x] `api.ts`：`updateProjectWatchConfig` + `Project` 接口 watch 字段
- [x] `frontend/e2e/tests/watch-mode.spec.ts`（E2E-WATCH-01）
- [x] 4 个测试用例：渲染、配置、API 验证、关闭
- [x] 验收: REQ-BW4

**BW4 阶段验收命令**：

```bash
go test ./internal/watch/... -v
cd frontend && npx playwright test e2e/tests/watch-mode.spec.ts
```

**PR**：BW4a（BW4.1–BW4.2）→ BW4b（BW4.3–BW4.4）

---

## 阶段 BW5：边缘发现补强

### BW5.1 gau query 提取

- [x] `parser/gau.go`：`GauResult` 结构化结果（URL/Host/Path/Params/HasQuery）
- [x] `parser/gau_test.go`：4 个测试用例（query params/empty/invalid/dedup）
- [x] 验收: REQ-BW5

### BW5.2 crt SAN 子域

- [x] `passive/crt.go`：`FetchSubdomains` 已存在（crt.sh API + SAN 提取 + 通配符处理）
- [x] `seed/passive.go`：`InjectCrt` 已存在（注入为 AssetSubdomain）
- [x] 验收: REQ-BW5

### BW5.3 AssetJSURL 接线

- [x] `executor/katana.go`：`isJSURL` 检测 + JS URL 分类为 `AssetJSURL`
- [x] `core/preconditions.go`：`isSpoorEligible` 支持 `AssetJSURL`
- [x] `core/rules.go` + `rules_external.go`：Spoor rule 使用 `isSpoorEligible`
- [x] `executor/katana_test.go`：JS URL 分类测试 + `isJSURL` 测试
- [x] 验收: REQ-BW5

### BW5.4 文档

- [x] `docs/current/architecture.md`：资产类型表新增 `AssetJSURL`
- [x] 验收: REQ-BW5

**BW5 阶段验收命令**：

```bash
go test ./internal/scanengine/executor/... -v -count=1
```

**PR**：BW5（单 PR，可按 BW5.1/BW5.2/BW5.3 拆 commit）

---

## 全量完成验收

全部 BW0–BW5 checkbox 完成后：

- [ ] 更新 `docs/active/review/bounty-watch-acceptance.md` 逐项勾选
- [ ] 更新 `docs/current/plan.md` workstream 状态 → Accepted
- [ ] 更新 `docs/current/architecture.md` 新增 §Bounty Watch
- [ ] 运行全量 E2E regression（`frontend/e2e/tests/` 核心 spec）
- [ ] promote `docs/design/bounty-watch/*.md` status → accepted

```bash
# 最终回归（示例）
cd frontend && npx playwright test e2e/tests/scope-boundary.spec.ts \
  e2e/tests/bounty-preset.spec.ts \
  e2e/tests/signal-inbox.spec.ts \
  e2e/tests/scan-delta.spec.ts \
  e2e/tests/watch-mode.spec.ts \
  e2e/tests/external-scan-conv.spec.ts \
  e2e/tests/company-scan-flow.spec.ts
```

---

## 场景注册表（实现时写入 functional-test.md）

| 场景 ID | REQ | 摘要 |
|---------|-----|------|
| FT-SCOPE-BW-01 | BW0 | strict + wildcard include 单元测试 |
| E2E-SCOPE-BW-01 | BW0 | UI 开 strict，OOS 域不可见 |
| FT-SPOOR-BW-01 | BW1 | katana off + spoor on |
| E2E-BOUNTY-PRESET-01 | BW1 | ScanModal 赏金 preset |
| E2E-INBOX-01 | BW2 | 扫描后 inbox 有 signal |
| E2E-DELTA-01 | BW3 | 二次扫描 delta 下降 |
| E2E-WATCH-01 | BW4 | watch tick 自动 run |
