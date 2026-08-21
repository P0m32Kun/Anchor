---
status: accepted
source_of_truth: true
owner: kun
created: 2026-06-17
scope: anchor-core-slimdown
---

# Anchor Core 收敛设计

## 决策

- **方案 A（激进瘦身）**，用户确认 **SRC 能力全部删除**。
- 北极星：分布式扫描器 — 内外网信息收集 + 指纹驱动 POC + Findings 输出。
- 扩展能力（Watch、Signal Inbox、赏金工作站）全部冻结或删除，稳定核心后再议。

## 保留

| 模块 | 理由 |
|------|------|
| ScanEngine 资产驱动引擎 | 核心执行模型 |
| Worker 分布式调度 | 内外网跳板刚需 |
| FOFA/Hunter/Quake 被动搜索 | 外网资产入口 |
| httpx 指纹 + Nuclei 路由 | POC 迭代主路径 |
| RBKD dict/templates/finger 同步 | 团队资源分发 |
| 字典 / 指纹 CRUD | ffuf / httpx 迭代 |
| Projects / Targets / Runs / Assets / Findings | 最小产品闭环 |
| Exclude domains | 降噪必需 |

## Phase 1 删除（本次）

### 后端整包删除

- `internal/bounty/`
- `internal/submission/`
- `internal/sources/`
- `internal/credentials/`
- `cmd/anchor-cred/`
- `internal/api/program_handlers.go`
- `internal/api/bounty_handlers.go`
- `internal/api/submission_handlers.go`
- `internal/api/credential_handlers.go`
- `internal/models/src_program.go`
- `internal/models/bounty_candidate.go`
- `internal/models/submission_pack.go`
- `internal/db/queries_src_program.go`
- `internal/db/queries_src_program_test.go`
- `internal/db/queries_bounty_candidate.go`
- `internal/db/queries_submission_pack.go`

### 前端删除

- `SRCProgramPage.tsx`
- `BountyQueuePage.tsx`
- `SubmissionPackPage.tsx`
- `frontend/e2e/tests/bounty-preset.spec.ts`
- `srcApi` 及关联类型（`api.ts`）

### API 路由删除

- `/projects/{id}/src-program*`
- `/src-programs`
- `/projects/{id}/bounty-candidates*`
- `/bounty-candidates/{id}*`
- `/submission-packs*`
- `/credentials*`
- `/sources*`

### 扫描模式重命名

- `src_low_noise` → `external_low`（UI 标签「外网低噪音」）
- 删除 `bounty` preset；`NormalizeScanMode` 仍将 legacy 值映射为 `external`+`low`
- DB 迁移 v29–v31 **保留**（历史库兼容，表闲置无害）

## 后续 Phase

| Phase | 内容 | 状态 |
|-------|------|------|
| P1 | Batch 调度接线（scanengine pool → engine.go） | ✅ 已落地 |
| P2 | 删除 legacy workflow、evaluation、scan-plans、slow-scan | ✅ 已完成 |
| P3 | ScanModal 三 preset + 高级折叠；Nuclei 管理简化 | ✅ 已完成 |
| P4 | E2E 核心集回归；架构/functional-test 文档同步 | 🔄 待验收 |

## 验收

- `go build ./...` 通过
- `go vet ./internal/api/...` 通过
- 前端 typecheck 通过
- 外网 / 内网 两种 ScanModal 模式可启动扫描
- 无 SRC / 赏金 / 提交包 UI 入口
