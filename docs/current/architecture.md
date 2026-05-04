---
status: active
source_of_truth: true
owner: kun
last_updated: 2026-05-04
scope: runtime-baseline
---

# Current Architecture Baseline

This file describes the current repository baseline that agents should assume unless a task explicitly opts into an in-review design.

## System Shape

- Desktop client: Tauri 2.x shell hosting a React 18 + TypeScript frontend
- Local/remote service: Go application providing API, orchestration, and worker-facing endpoints
- Persistence: SQLite in WAL mode
- Realtime updates: SSE
- Scan execution: worker processes running external security tools (subfinder, httpx, naabu, nerva, cdncheck, nuclei)
- Pipeline configuration: per-project config for stage toggles, port range presets, concurrency, and timeouts

## Baseline Workflow

The stable product narrative remains:

`目标输入 -> Scope Check -> 资产发现 -> Web 初筛 -> 人工验证 -> 报告导出`

实际执行管线（v0.3 已实现）：

```
目标导入 → 分类 → (FOFA/Subfinder) → DNS 解析 → CDN 过滤 → Naabu 端口扫描 → nerva 服务指纹 → httpx Web 探活 → Nuclei 漏洞扫描
```

各阶段开关、端口范围、并发与超时均通过项目级 `PipelineConfig` 配置，前端在 ScanConfigPage 编辑。

## What Is Not Baseline Yet

- `docs/design/v0.4-scan-pipeline.md` describes a newer pipeline direction, but it is still an in-review design document.
- `docs/design/v0.4-migrations.md` is migration support material for that same proposal.
- `docs/refactoring-plan.md` is a backlog/refactor inventory, not the current product architecture.

## How To Use This File

- Use this file for repo-level orientation.
- Use the implementation and tests to answer behavior questions.
- Use `docs/current/design/README.md` only when a task explicitly targets a proposal or review stream.

## Documentation Contract

If architecture changes materially, update this file first or in the same change set. Proposal documents should explain the delta from this baseline instead of redefining the entire system from scratch.
