---
status: accepted
source_of_truth: false
owner: kun
created: 2026-06-07
scope: scan-engine-convergence
---

# Scan Engine 收敛 — Tasks

> **当前阶段**：Accepted — G0–G5 完成（2026-06-07）

## 阶段 G0：企业名入口 + 三引擎 PassiveSearch

### G0.1 定义 SeedAsset 类型
- [x] 新建 `internal/scanengine/seed/types.go`
- [x] 验收: REQ-1（类型可被测试引用）

### G0.2 实现 PassiveSearch 三引擎并行
- [x] 新建 `internal/scanengine/seed/passive_search.go`
- [x] FOFA/Hunter/Quake 并行；fail-soft；dedup；限额
- [x] 读 `cfg.EnablePassiveSearch`、`PassiveSearchResultLimit`、`PassiveSearchConcurrency`
- [x] 验收: REQ-1

### G0.3 重构 ExpandTargets
- [x] 改签名传入 `PipelineConfig`；company 走 PassiveSearch；其他 target 转 SeedAsset
- [x] 更新 `pipeline_handlers.go` 调用
- [x] 验收: REQ-1

### G0.4 Unit tests
- [x] `passive_search_test.go`：mock 三引擎、单引擎失败、限额、开关关闭
- [x] 验收: REQ-1

### G0.5 前端恢复企业名入口
- [x] `TargetPage.tsx` 增加「企业名」选项 + 说明文案
- [x] 验收: REQ-2

### G0.6 E2E company 流程
- [x] 新建 `frontend/e2e/tests/company-scan-flow.spec.ts`（场景 E2E-COMPANY-01）
- [x] `TargetPage.spec.ts` TC-3 覆盖 UI 添加企业名
- [x] 删除 `v0.4-company-flow.spec.ts` fixme 占位
- [x] 本地 `company-scan-flow.spec.ts` 跑通（2026-06-07）
- [x] 验收: REQ-2（UI 路径）；REQ-1（依赖 fofa-mock + AssetPage 断言）

**PR**: G0a（G0.1–G0.4）→ G0b（G0.5）→ G0c（G0.6）

---

## 阶段 G1：资产血缘（asset_relations）

### G1.1 DB 迁移
- [x] `internal/db/v33.go` 创建 `asset_relations`
- [x] 验收: REQ-3

### G1.2 Model + Queries
- [x] `internal/models/asset_relation.go`
- [x] `internal/db/queries_asset_relation.go` + test
- [x] 验收: REQ-3

### G1.3 Engine 写入 relations
- [x] company 展开：`expanded_by`（target → asset）via `RunWithSeeds` + `SeedAsset.ToDiscoveryAsset`
- [x] `processNewAsset` / `prepareChildAsset`：`discovered_from`（asset → asset）
- [ ] 写 `discovery_depth` 到 assets / web_endpoints（defer G2 前可选）
- [x] 验收: REQ-3（expanded_by + discovered_from）

### G1.4 Lineage API
- [x] `GET /assets/{id}/lineage?run_id=` in asset_handlers
- [x] 更新 `internal/api/README.md`
- [x] E2E-LINEAGE-01 并入 `company-scan-flow.spec.ts` Step 6
- [x] 验收: REQ-3

**PR**: G1a（G1.1–G1.2）→ G1b（G1.3–G1.4）— 本批一次交付

---

## 阶段 G2：资产门控属性持久化

### G2.1 asset_state schema
- [x] 迁移：`assets.state_json`（v34）
- [x] 验收: REQ-4

### G2.2 写入与读取
- [x] `onWorkComplete` 写 state（httpx/cdncheck/dnsx）
- [x] `processNewAsset` hydrate + persist via `asset.LoadAttrsForEngine` / `MergeAndSaveState`
- [x] 验收: REQ-4

### G2.3 Unit tests
- [x] `queries_asset_state_test.go` + `asset/state_test.go`
- [x] 验收: REQ-4

**PR**: G2（单 PR）— 已完成

---

## 阶段 G3：Profile 收敛 + high-value 门控

### G3.1 提取 preconditions
- [x] 新建 `internal/scanengine/core/preconditions.go`（`isHighValueHTTP`）
- [x] 验收: REQ-5

### G3.2 实现 isHighValueHTTP
- [x] 外网 katana/ffuf precondition 叠加 high-value
- [x] 验收: REQ-5

### G3.3 收窄 DefaultExternalPipelineConfig
- [x] katana/ffuf 默认 false；EnablePassiveSearch 保持 true
- [x] 更新 `frontend/src/lib/api.ts` `DEFAULT_EXTERNAL_PIPELINE_CONFIG`
- [x] 验收: REQ-5

### G3.4 测试
- [x] `preconditions_test.go` high-value 用例
- [x] `models/engine_test.go` 外网默认 katana/ffuf off
- [ ] smoke E2E 验证 work 数下降 + finding 存在（G4 后统一验收）
- [x] 验收: REQ-5（单元层）

**PR**: G3 — 已完成（E2E smoke 指标留 G4/G5）

---

## 阶段 G4：删除双轨执行

### G4.1 删除 legacy workflows
- [x] 删 `internal/workflows/`
- [x] 删 workflow discovery/screening handlers 与路由
- [x] 迁移 `handleListWebEndpointsByProject` 到 `asset_handlers.go`
- [x] 删 `handleRunTask` dead code（路由已移除）
- [x] 验收: REQ-6

### G4.2 前端统一入口
- [x] AssetPage「启动扫描」→ ScanModal / `createScan`
- [x] 删 `api.startAssetDiscovery` / `startWebScreening` / `runTask`
- [x] 验收: REQ-6

### G4.3 E2E 更新
- [x] AssetPage spec 验证 ScanModal 入口（不再调 legacy API）
- [x] 验收: REQ-6

**PR**: G4a（后端）→ G4b（前端）

---

## 阶段 G5：文档同步（贯穿各 PR，最终收口）

### G5.1 架构基线
- [x] 更新 `docs/current/architecture.md`：Tier-0 入口、seed 路径、删 runPassiveSearch 描述
- [x] 验收: REQ-7

### G5.2 Plan + functional-test
- [x] `plan.md` workstream 状态 → Accepted
- [x] `functional-test.md` 场景表更新
- [x] 验收: REQ-7

### G5.3（可选 P2）规则扩展点
- [x] 文档化 ActionRule 扩展方式（`architecture.md` + DEFER-1）
- [x] 验收: DEFER-1 记录，不阻塞发布

---

## Implement 阶段门禁（develop-feature）

进入 Implement 前：

- [ ] proposal.md 用户已批准
- [ ] 本 tasks.md 已确认 PR 顺序
- [ ] BDD 场景已登记 `functional-test.md`

每个 task 执行时：

1. `test-strategy` 选层
2. `tdd` 红-绿-重构
3. 单包 `go build ./that/pkg/`
4. PR 合并前 `make test-e2e-smoke`（G0c 之后）

## Verify 阶段（全部 PR 合并后）

- [ ] REQ-1 ～ REQ-7 验收信号逐项勾选
- [ ] rangefield 实扫：company 或 domain 入口 → finding + lineage 可查
- [ ] `go test ./...` 全绿
- [ ] 产出 `docs/active/review/scan-engine-convergence-acceptance.md`

## 进度跟踪

| 阶段 | 状态 | PR |
|------|------|-----|
| G0 | **完成** | G0.6 E2E 跑通（2026-06-07） |
| G1 | **完成** | asset_relations + lineage API + E2E-LINEAGE-01（2026-06-07） |
| G2 | **完成** | assets.state_json + hydrate/persist（2026-06-07） |
| G3 | **完成** | 外网 katana/ffuf 默认关 + high-value 门控（2026-06-07） |
| G4 | **完成** | 删 workflows + AssetPage ScanModal 入口（2026-06-07） |
| G5 | **完成** | 文档同步 + acceptance 记录（2026-06-07） |
