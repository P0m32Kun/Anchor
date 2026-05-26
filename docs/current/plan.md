---
status: active
source_of_truth: true
owner: kun
last_updated: 2026-05-18
scope: repository-wide
---

# Current Plan

This is the only repository-wide implementation plan that agents should treat as current by default.

## Current Baseline

- The last clearly completed milestone is `v0.4` ("smart scan pipeline + multi-target-type + Nuclei layered scanning").
- Post-v0.4 increments (v2.1 selective ingestion PR1-3) merged to main 2026-05-16: batch finding buffer, resource governor, tool execution allowlist.
- v0.4 acceptance criteria and test mapping live in [`../active/review/v0.4-acceptance.md`](../active/review/v0.4-acceptance.md).
- If a document conflicts with this file, treat this file as the planning baseline unless the task explicitly says otherwise.

## Active Workstreams

| Workstream | Status | Source of truth? | Notes |
| --- | --- | --- | --- |
| Code health audit | Accepted | Yes | Superseded by Slimdown 2026 audit; see [`slimdown-2026-summary.md`](../active/review/slimdown-2026-summary.md) for delta |
| Documentation governance | Active | Yes | Consolidate navigation, mark document lifecycle, reduce agent confusion |
| Current architecture baseline | Active | Yes | Defined in [`architecture.md`](architecture.md) |
| Slimdown 2026 | Active | Yes | Repository-wide slimdown audit in [`slimdown-2026-handoff-for-implementer.md`](../active/review/slimdown-2026-handoff-for-implementer.md); T0-T6 reports at `docs/active/review/slimdown-*.md` |
| v0.4 scan pipeline | Accepted | Yes | Verified by `docs/active/review/v0.4-acceptance.md`; design doc archived at `docs/archived/v0.4/scan-pipeline.md` (superseded 2026-05-12, see `architecture.md` for current baseline) |
| Database migration notes for v0.4 | Accepted | Yes | Tied to v0.4; superseded plans now archived |
| Large-scale refactoring | Backlog | No | See [`../refactoring-plan.md`](../refactoring-plan.md) as an idea pool, not an approved release plan |
| Design v2.1 selective ingestion | Active | Yes | PR1-3 merged to main (batch buffer, governor, allowlist); PR4 (asset_relations) remains backlog |

## Rules For New Work

1. Promote only one repository-wide plan at a time.
2. Keep proposal documents in `docs/design/` until they are accepted and verified.
3. Move completed milestone plans to `docs/archived/`.
4. If a task needs a short-lived task plan, keep it scoped to that task and link back here instead of creating another competing top-level roadmap.
5. **改代码前先查执行流,不要手翻文件**:
   - 修改 `internal/api/` 下的 handler 前,先看 [`internal/api/README.md`](../../internal/api/README.md) 的「字段反向索引」和「Handler 文件总览」,定位需要读的最小代码集。
   - 修改任意 Go 符号前,先用 `gitnexus_query`(找执行流)或 `gitnexus_context`(看某个符号的 360° 视图)查清楚 blast radius,而不是 grep 或读全文件。
   - 详细执行约束见项目根目录 `CLAUDE.md` 的「Always Do」与「文档同步约束」。

## Exit Criteria For Promoting A Proposal

A proposal can replace this plan only when all of the following are true:

- its scope is approved,
- its implementation status is clear,
- the related E2E acceptance path is defined,
- superseded plans are explicitly marked.

## Design v2.1 Selective Ingestion

来源:外部 PRD `pentest-arsenal v2.1`(Python+Vue 重写型 4 周 MVP)。整体不吸收(技术栈不兼容、相对当前 Anchor 是功能子集)。仅吸收下列 4 项真正补齐 Anchor 现有缺口的点。每一项交付都要落到 E2E 实测,build/typecheck 不算完成。

| # | 吸收项 | 优先级 | Anchor 现状 | 落点 (相对路径) | 验收信号 |
|---|---|---|---|---|---|
| 1 | Findings 批量写入缓冲 | P0 | 单条 INSERT,无 buffer | `internal/db/queries_finding.go` 增加 BatchInsert;`internal/workflow/pipeline_result.go` 改走 buffer | 高产出场景写入吞吐显著提升;进程异常退出时缓冲已 flush(注册关停钩子),无 finding 丢失 |
| 2 | 资源治理(内存/CPU 阈值降级) | P1 | 无系统级资源保护,长扫易把笔记本拖死 | 新增 `internal/worker/resource_governor.go`;接入 `dispatcher.go` | 内存超阈值时新任务排队、CPU 超阈值时入队速率减半;阈值参数与上游工具单位一致(参考 [`feedback_no_unit_conversion`]) |
| 3 | 工具二进制 allowlist + 参数白名单 | P1 | `exec.Command(bin, args...)` 已天然规避 shell 注入;但二进制路径与参数缺少集中白名单 | 复核 `internal/health/health.go`、`internal/worker/worker.go`、`internal/worker/server.go`、`internal/cdn/detector.go`、`internal/nuclei/custom/git.go` 五个 `exec.Command` 调用点;统一通过 allowlist 解析 | 任一非白名单二进制/参数调用被拒;新增工具时强制走注册流程 |
| 4 | 资产关联图结构 (asset_relations) | P2 | 资产表扁平,无 (source, target, relation_type) 三元组 | 新增 `internal/db/v24.go` 迁移;模型 `internal/models/asset_relation.go`;查询 `internal/db/queries_asset_relation.go` | 至少能表达 domain→ip(belongs_to)、port→service(runs)、cidr→ip(contains);报告聚合时能挂载关联;先做后端,前端图谱可视化是 P3 再议 |

不吸收(已对照过、不必再做):
- **Canonical Finding Schema**:`internal/models/finding.go` 的 Finding 模型已含 `source_tool` / `source_rule_id` / `dedup_key` / `severity` / `confidence` / `raw_request` 等,字段覆盖优于 PRD 提案,无需重做。
- **Adapter/Pipeline/SSE/扫描模板/报告引擎**:`internal/workflow/pipeline*.go` 与 `stageemitter.go` 已实现且更完整,设计文档的版本是退步。
- **Phase 0-3 路线图、技术栈替换、PyInstaller、单机启动密码**:与 Anchor 现有架构方向不一致,不采纳。

每项完成时,把验收信号写进 `docs/active/review/`,并把对应行从本节移到 "Active Workstreams" 表的 `Accepted` 状态。

### PR 进度（v2.1 selective ingestion）

| PR | 吸收项 | 状态 | 提交 |
|----|--------|------|------|
| PR1 | Findings 批量写入缓冲 | Merged to main | `709bfff` |
| PR2 | 资源治理（静态阈值） | Merged to main | `410738a` |
| PR3 | 工具二进制 allowlist + 参数白名单 | Merged to main | `f2818be` |

PR2 的实现细节:
- `internal/worker/resource_governor.go`:`Acquire(ctx)` 检查系统内存/CPU,内存超阈值轮询阻塞、CPU 超阈值 sleep 固定延迟。
- 接入点:`Runner.Run`(API 服务器任务入口)与 `WorkerServer.executeTask`(远端 worker 任务执行)。
- 采样实现:`github.com/shirou/gopsutil/v3`,通过 `ResourceSampler` 接口在测试中注入 fake。
- 阈值通过 `ANCHOR_GOVERNOR_*` 环境变量配置,详见 `docs/current/architecture.md` 的「资源治理」章节。

PR3 的实现细节:
- `internal/toolguard/allowlist.go`:`Allowlist` 结构提供二进制白名单 + 参数 shell 元字符检查。
- `Validate(binary, args)` 基于 `filepath.Base` 检查二进制名，拒绝任何含 shell 元字符的参数。
- 接入点:全部 5 个 `exec.Command` 调用文件（`worker.go`, `server.go`, `health.go`, `cdn/detector.go`, `nuclei/custom/git.go`）。
- `Allowlist.Allow(name)` 支持运行时扩展，新增工具强制走注册流程。

## GSTACK REVIEW REPORT

| Review | Trigger | Why | Runs | Status | Findings |
|--------|---------|-----|------|--------|----------|
| CEO Review | `/plan-ceo-review` | Scope & strategy | 1 | CLEAR (PLAN) | 8 proposals, 3 accepted, 5 deferred |
| Codex Review | `/codex review` | Independent 2nd opinion | 1 | issues_found | 27 missed problems |
| Eng Review | `/plan-eng-review` | Architecture & tests (required) | 2 | CLEAR | 16 issues resolved, 2 critical gaps fixed |
| Design Review | `/plan-design-review` | UI/UX gaps | 1 | CLEAR (FULL) | score: 4/10 → 8/10, 6 decisions |
| DX Review | `/plan-devex-review` | Developer experience gaps | 0 | — | — |

- **CODEX:** 27 missed problems including buffer durability (SIGKILL/OOM), SQLite write contention, resource governance overbuilt, allowlist environment attack surface, asset relation taxonomy issues
- **CROSS-MODEL:** Buffer persistence — review recommended shutdown hook, Codex argued for durable staging (pipeline boundary flush accepted). Resource governance — review recommended dynamic thresholds, Codex argued for static limits (accepted).
- **CRITICAL GAPS FIXED (2026-05-18):**
  1. **buffer flush silent loss** — `FindingBuffer.flushLocked()` now clears `b.buf` only after successful `BatchInsertFindings`; on failure, findings are retained and retried via timer (`flushTimerCallback` restarts timer on error). Tests: `TestFindingBuffer_FlushFailureRetainsData`, `TestFindingBuffer_TimerRetryAfterFailure`.
  2. **SIGKILL finding loss** — Reduced flush interval from 5s to 2s in `Pipeline.Run()`, shrinking the unflushed data window. SIGKILL itself remains uncatchable; mitigation is shorter flush interval + pipeline boundary flush (already in place).
- **UNRESOLVED:** 0
- **VERDICT:** Eng Review CLEARED — all critical gaps fixed, 16 architecture/code-quality/test/performance issues addressed, scope held at 3 PRs.

*Last reviewed: 2026-05-18*
