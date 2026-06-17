---
status: accepted
source_of_truth: false
owner: kun
created: 2026-06-07
scope: scan-engine-convergence
---

# Scan Engine 收敛 — 验收记录

> 设计包：[`docs/design/scan-engine-convergence/`](../../design/scan-engine-convergence/proposal.md)  
> 架构基线：[`docs/current/architecture.md`](../../current/architecture.md)

## 结论

**G0–G5 已于 2026-06-07 完成。** ScanEngine（`POST /projects/{id}/scan`）为唯一扫描执行路径；legacy `internal/workflows/` 与 workflow HTTP 路由已删除。

## REQ 验收清单

| REQ | 摘要 | 证据 |
|-----|------|------|
| REQ-1 | company 三引擎 PassiveSearch | `internal/scanengine/seed/passive_search_test.go`；E2E-COMPANY-01 |
| REQ-2 | TargetPage 企业名入口 | `TargetPage.tsx`；`company-scan-flow.spec.ts` |
| REQ-3 | asset_relations + lineage API | `v33` 迁移；`queries_asset_relation_test.go`；E2E Step 6 |
| REQ-4 | assets.state_json 门控持久化 | `v34`；`internal/asset/state_test.go` |
| REQ-5 | 外网 Profile 收敛 + high-value 门控 | `preconditions_test.go`；`models/engine_test.go` |
| REQ-6 | 单一执行路径 | workflows 包已删；AssetPage → ScanModal；`AssetPage.spec.ts` |
| REQ-7 | 文档同步 | 本文件 + `architecture.md` + `plan.md` + `functional-test.md` + `api/README.md` |

## 阶段交付

| 阶段 | 交付物 |
|------|--------|
| G0 | `scanengine/seed`、TargetPage 企业名、E2E-COMPANY-01 |
| G1 | `asset_relations`、lineage API、E2E-LINEAGE-01 |
| G2 | `assets.state_json`、hydrate/persist |
| G3 | 外网 katana/ffuf 默认关、`isHighValueHTTP` |
| G4 | 删 workflows、AssetPage ScanModal、删 legacy API |
| G5 | 架构/plan/functional-test/api-contracts 同步 |

## 自动化验证（本地）

```bash
go test ./internal/scanengine/ ./internal/db/ ./internal/asset/ ./internal/models/ ./internal/api/ -count=1
go build ./internal/api/

# E2E（需 Docker 栈或 ANCHOR_E2E_SKIP_DOCKER=1 + 已运行服务）
cd frontend && npx playwright test e2e/tests/company-scan-flow.spec.ts --project=chromium-scan
cd frontend && npx playwright test e2e/tests/AssetPage.spec.ts --project=chromium
```

## 已知 defer

| ID | 内容 |
|----|------|
| DEFER-1 | YAML 规则引擎（已文档化 ActionRule/precondition 扩展点） |
| DEFER-2 | 资产图谱前端可视化 |
| DEFER-3 | ports/services 表合并 |
| E2E-SCAN-CONV-01 | smoke 验证外网 work 数下降 | `external-scan-conv.spec.ts` 已自动化 |

## Verify 阶段待办（可选 follow-up）

- [ ] rangefield 全链路实扫：company → finding + lineage 人工勾选
- [ ] `make test-e2e-smoke` CI 绿
- [ ] `go test ./...` 全仓库绿
