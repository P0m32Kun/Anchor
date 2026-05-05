---
status: active
source_of_truth: true
owner: kun
last_updated: 2026-05-05
scope: runtime-baseline
---

# Current Architecture Baseline

This file describes the current repository baseline that agents should assume unless a task explicitly opts into an in-review design.

## System Shape

- Desktop client: Tauri 2.x shell hosting a React 18 + TypeScript frontend
- Local/remote service: Go application providing API, orchestration, and worker-facing endpoints
- Persistence: SQLite in WAL mode
- Realtime updates: SSE
- Scan execution: worker processes running external security tools (subfinder, dnsx, httpx, naabu, nerva, cdncheck, nuclei)
- Pipeline configuration: mode-driven (`external`/`internal`) tool selection, per-tool speed params (rate limit, threads, timeout), port range presets
- Global engine credentials: FOFA/Hunter/Quake API keys stored in `engine_credentials` table, configured via `/engines/keys`

## Baseline Workflow

The stable product narrative remains:

`目标输入 -> Scope Check -> 资产发现 -> Web 初筛 -> 人工验证 -> 报告导出`

实际执行管线（当前已实现）：

```
目标导入 → 分类 → (FOFA/Subfinder) → DNSx 解析 → CDN 过滤 → Naabu 端口扫描 → nerva 服务指纹 → httpx Web 探活 → Nuclei 漏洞扫描
```

扫描模式由前端 `ScanModal` 选择：

- **外网扫描 (`external`)**：启用全部工具链（FOFA → Subfinder → DNSx → CDNCheck → Naabu → Nerva → HTTPX → Nuclei）
- **内网扫描 (`internal`)**：仅启用 Naabu → Nerva → HTTPX → Nuclei

各工具的速率限制、并发线程、超时参数在 `ScanModal` Step 2 中配置，通过 `POST /projects/{id}/scan` 的 `config` 字段传递。端口范围支持 top100 / top1000 / high-risk / full / custom 五种预设。

FOFA 凭证不再绑定到项目，而是从全局 `engine_credentials` 表读取。Hunter 和 Quake 通过独立的 `/engines/search` API 调用，结果统一为 `SearchResult` 格式。

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
