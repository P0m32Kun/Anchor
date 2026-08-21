---
status: accepted
source_of_truth: false
owner: kun
created: 2026-06-07
scope: scan-engine-convergence
---

# Scan Engine 收敛 — Spec

## Purpose

定义 Anchor 从「功能齐全」收敛为「护网可用攻击面扩展引擎」的可观察行为与验收信号。代码实现须满足本 spec 全部 REQ，build/typecheck alone 不算完成。

## 外网入口模型（Tier-0）

```text
company  ──► PassiveSearch (FOFA ∥ Hunter ∥ Quake)
domain   ──► [可选] crt.sh / gau
ip/cidr  ──► 直接进入 ScanEngine
url      ──► 直接进入 ScanEngine
                │
                ▼
         seed (domain / ip / url)
                │
                ▼
         ScanEngine 资产驱动循环
```

---

## REQ-1: 企业名经三引擎并行展开

**验收信号**：

- [x] Given 项目目标 type=company 且已配置 FOFA/Hunter/Quake 凭证，When 启动外网扫描，Then seed 包含三引擎结果的 domain/ip/url 并去重
- [x] Given `EnablePassiveSearch=false`，When 启动扫描，Then 不调用三引擎，company 目标 log 跳过原因
- [x] Given 单引擎失败，When 其他引擎成功，Then 扫描继续（fail-soft）
- [x] Given 结果超 `PassiveSearchResultLimit`，When 展开，Then 截断至限额
- [x] Unit: `internal/scanengine/seed/*_test.go` mock 三引擎全绿

**场景**：

- Given 目标「某某科技有限公司」且 FOFA mock 返回 domain+ip
- When  `ExpandTargets` 被调用且 `EnablePassiveSearch=true`
- Then  返回 seeds 含 FOFA 产出的 domain 与 ip

---

## REQ-2: 前端可显式添加企业名目标

**验收信号**：

- [x] UI: TargetPage 类型下拉含「企业名」
- [x] UI: 文案说明将通过 FOFA/Hunter/Quake 展开（需配置引擎密钥）
- [x] API: `POST /projects/{id}/targets` type=company 成功创建
- [x] E2E: `E2E-COMPANY-01` 从 UI 添加 company → 启动外网扫描 → Runs 页有 running run

**场景**：

- Given 用户在外网项目 TargetPage
- When  选择「企业名」并输入「TestCorp」提交
- Then  目标列表显示 type=company 的 TestCorp

---

## REQ-3: 资产血缘可追溯到入口

**验收信号**：

- [x] DB: `asset_relations` 表存在，含 `(project_id, source_id, target_id, relation_type, source_engine, run_id)`
- [x] Given company 展开产生 domain seed，When 扫描完成，Then relation `expanded_by` 从 company target 指向 domain asset
- [x] Given httpx 发现子路径，When 完成，Then relation `discovered_from` 从 parent asset 指向 child
- [x] API: `GET /assets/{id}/lineage?run_id=` 返回从 seed 到当前节点的链
- [x] Unit: relation 写入与查询测试全绿

**场景**：

- Given company 目标 + FOFA 展开 foo.com
- When  扫描产生 finding 关联 foo.com 下某 endpoint
- Then  lineage API 可从 endpoint 追到 company 目标

---

## REQ-4: 资产门控属性持久化

**验收信号**：

- [x] httpx 完成后 `fingerprinted=true` 与 technologies 落库（asset_state 或等价）
- [x] cdncheck 完成后 `is_cdn` 落库
- [x] Given 同 asset 二次 run，When 读 state，Then Nuclei 派生前可判断已 fingerprinted（可选 skip httpx）
- [x] Unit: state 读写测试全绿

---

## REQ-5: 外网护网 V1 Profile 收敛

**验收信号**：

- [x] `DefaultExternalPipelineConfig()`: `EnablePassiveSearch=true`；katana/ffuf 默认 **false**
- [x] high-value HTTP 才派生 katana/spoor/ffuf（precondition 可测）
- [x] `ExternalProfile.Rules()` 与 `PipelineConfig` 开关一致（无 rules=false 但 config=true 矛盾）
- [x] ScanModal 外网模式 UI 与默认值对齐
- [x] E2E smoke：work 总数下降，仍有 ≥1 finding

**high-value 判定（任一满足）**：

- Technologies 非空
- StatusCode ∈ [200, 399]
- Spoor Sensitivity == high

---

## REQ-6: 单一扫描执行路径

**验收信号**：

- [x] `POST /workflows/asset-discovery` 与 `web-screening` 路由移除
- [x] `internal/workflows/` 包删除
- [x] AssetPage「启动扫描」改为打开 ScanModal
- [x] `grep -r workflows/asset-discovery frontend/` 无匹配
- [x] E2E AssetPage 不再调 legacy API

---

## REQ-7: 文档与架构基线同步

**验收信号**：

- [x] `architecture.md` 被动搜索描述指向 `scanengine/seed`，删除 `runPassiveSearch` 旧引用
- [x] 删除 `ANCHOR_SCAN_ENGINE=1` 过时描述（ScanEngine 已是唯一路径）
- [x] `plan.md` workstream 状态更新
- [x] `functional-test.md` 场景注册表已登记 REQ 对应场景
- [x] 若改 handler：`internal/api/README.md` 同步

---

## 不在范围内（明确 defer）

| ID | 内容 | 原因 |
|----|------|------|
| DEFER-1 | YAML 规则引擎 | Phase G5 仅提取 preconditions |
| DEFER-2 | 资产图谱前端可视化 | lineage API 先行 |
| DEFER-3 | ports/services 表合并 | 方案 A（relations）优先 |

---

## BDD 场景映射

| 场景 ID | REQ | 自动化 |
|---------|-----|--------|
| E2E-COMPANY-01 | REQ-2, REQ-1 | `frontend/e2e/tests/company-scan-flow.spec.ts` |
| FT-PASSIVE-01 | REQ-1 | `internal/scanengine/seed/passive_search_test.go` |
| FT-LINEAGE-01 | REQ-3 | `internal/db/queries_asset_relation_test.go` + company-scan-flow Step 6 |
| E2E-ASSET-SCAN-01 | REQ-6 | `frontend/e2e/tests/AssetPage.spec.ts` |
| E2E-SCAN-CONV-01 | REQ-5 | `frontend/e2e/tests/external-scan-conv.spec.ts` |
