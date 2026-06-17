---
status: draft
source_of_truth: false
owner: kun
created: 2026-06-17
scope: batch-scan-scheduling
---

# 批量扫描调度 — Tasks

> **当前阶段**：Verify — P3 文档/E2E 已提交，待 Docker 实扫  
> **设计**：[design.md](./design.md) | **验收**：[spec.md](./spec.md)

## PR 拆分原则

- 每 PR ≤400 行有效 diff
- 每 PR 后 `go build ./internal/scanengine/...` + 相关单测
- P0 完成后可独立合并；P1/P2 依赖 P0 调度接线

---

## P0：调度接线 + 生命周期 + CDN + 前端分页

> 不引入 BatchWork，先让现有 per-asset work 按 Stage/Bucket 正确调度。

### P0.1 实现 `PopFair` / `PopFairStaged`

- [x] 新建 `internal/scanengine/queue/fair.go`
- [x] `Item` 增加 `BucketKey` 字段（或 parallel map）
- [x] 跑绿 `queue/fair_test.go`、`queue/stage_rank_test.go`
- [x] 验收: REQ-B1（单测层）

### P0.2 `engine.tick` 接线 scheduler

- [x] import `scheduler`；`ComputeLimits` 替换固定 `BatchSize`
- [x] `Pop` → `PopFairStaged`
- [x] `executeWork` 集成 `IPThrottler`
- [x] seed → bucket 映射：`RunWithSeeds` 保存 seeds，`processNewAsset` 写 bucket
- [ ] 验收: REQ-B1（集成）

### P0.3 移除默认 AbsoluteTimeout

- [x] `DefaultEngineConfig` 去掉或禁用 `AbsoluteTimeout`
- [x] `finalizePipelineRun` 不再依赖 timeout 假完成
- [x] 单测：长 elapsed 不触发 stop
- [x] 验收: REQ-B7

### P0.4 Orphan run 恢复 + finalize 补强

- [x] 新建 `internal/scanengine/recovery/orphan.go`
- [x] `Server` 启动时调用：running run → failed；running work → failed
- [x] `finalizePipelineRun`：处理 running 僵尸
- [ ] 验收: REQ-B7（E2E）

### P0.5 CDN IP:port 归一化

- [x] `assetHostValue` / `buildParams` CDN：`SplitHostPort`
- [x] 单测 + 回归 cdncheck IP:port 场景
- [x] 验收: REQ-B8

### P0.6 API + 前端分页

- [x] `handleListScanRunWorks` / `handleListToolCallLogs` 分页 query
- [x] DB queries 增加 LIMIT/OFFSET + COUNT
- [x] `RunsPage.tsx`：轮询 metrics/summary；works/tool-calls 分页 lazy load
- [x] `internal/api/README.md` 同步
- [ ] 验收: REQ-B9（E2E）

**PR 建议**：P0.1–P0.2 一批 | P0.3–P0.4 一批 | P0.5 一批 | P0.6 一批

---

## P1：Tier1 大池（dnsx / naabu / cdncheck / subfinder）

### P1.1 通用 Pool 抽象

- [x] `internal/scanengine/pool/pool.go`：BatchSize、FlushTimeout、OnFlush 回调
- [x] 单测：dedup、timeout flush、满批 flush

### P1.2 HostPool + IPPool

- [x] `host_pool.go`（HostPool via pool.Config）
- [x] `ip_pool.go`（CDN + Port 分池）
- [x] flush → 创建 **BatchWorkItem**

### P1.3 BatchWork DB schema

- [x] 迁移 v39：`input_file`、`member_asset_ids`、`bucket_key`、`generation`、`batch_mode`
- [x] `work/store.go`：`CreatePooledBatch`
- [x] 验收: REQ-B2（数据层单测）

### P1.4 接入 engine：`processNewAsset` 改入池

- [x] Tier1 action 改 `pool.Add` + flush → batch work
- [x] `buildBatchParams` 读 `input_file`
- [x] `onBatch*Complete` 逐行回注
- [ ] 验收: REQ-B2（E2E 规模）

### P1.5 接入 domainpool（subfinder）

- [x] engine 启动 `domainpool` + `Start`
- [x] `SUBDOMAIN_ENUM` 改 batch `-dL`（`domain_file`）
- [x] Run 结束 `Stop` + final flush（`finalizeRun`）

### P1.6 Profile 收敛（部分）

- [x] CDN/DNS/Port/Subdomain 走 Tier1 池（不再 per-asset 单条 enqueue）
- [x] external DNS MaxDepth: -1 → 1
- [ ] 验收: REQ-B10（E2E）

**PR 建议**：P1.1–P1.3 一批 | P1.4–P1.5 一批 | P1.6 一批

---

## P2：Tier2（httpx 池 + nmap 聚合 + nuclei tech 分桶）

### P2.1 HTTPCandidatePool

- [x] `probe_target.go`：subdomain/IP/IPPort → probe target 去重
- [x] `httpPool` flush → httpx batch（100/10s）
- [x] `processHTTPXOutput` → technologies 写入 `asset_state`
- [ ] 验收: REQ-B3（E2E）

### P2.2 IP port-set 聚合 → nmap

- [x] `ip_port_agg.go`：按 IP 聚合 ports
- [x] flush → 1 IP = 1 SERVICE_FINGERPRINT batch work
- [x] `onBatchNmapComplete` nmap XML 逐 port 回注
- [ ] 验收: REQ-B4（E2E）

### P2.3 Nuclei tech 路由

- [x] `configs/scan.config.yaml`：`nuclei_tech_routing`
- [x] `internal/scanconfig/nuclei_routing.go` 加载
- [x] `nuclei_buckets.go`：httpx 后分桶；noise_level 行为
- [x] 删除 per-asset 立即 nuclei enqueue（改入 TagBucket）
- [ ] 验收: REQ-B5（E2E）

### P2.4 Nuclei 与 custom template 协同

- [ ] RoutingPolicy 扩展 tech 映射（可选 P2.4）
- [x] Tier3 单点 template_path 路径保留（非 batch nuclei 仍走单条 work 时可用）
- [ ] 验收: REQ-B5、REQ-B6（E2E）

**PR 建议**：P2.1 | P2.2 | P2.3+P2.4

---

## P3：收尾、E2E、文档

### P3.1 规模 E2E

- [x] `frontend/e2e/tests/batch-scan-scale.spec.ts`
- [x] 120 target：assert tool_calls / works 低于阈值
- [ ] 验收: REQ-B2/B10（Docker 实跑）

### P3.2 functional-test 登记

- [x] FT-BATCH-01：100+ domain 扫描 tool call / work 阈值
- [x] FT-BATCH-02：nuclei 分桶（`nuclei_routing_test.go`）
- [x] FT-BATCH-03：orphan 恢复（`orphan_test.go`）
- [x] 验收: REQ-B11 登记

### P3.3 架构文档

- [x] `docs/current/architecture.md` 新增「批量调度」章
- [x] `docs/design/hw-scan-optimization/design.md` 标注 AD-3/AD-5 承接
- [x] 产出 `docs/active/review/batch-scan-scheduling-acceptance.md`
- [x] 验收: REQ-B11

### P3.4（可选 P2）同质端口 nmap 批、调度↔rate 联动、Pool jitter

- [ ] hw-scan-optimization AD-5 待做表

---

## 文件改动总表（Implement 时对照）

| 文件 | P0 | P1 | P2 | P3 |
|------|:--:|:--:|:--:|:--:|
| `internal/scanengine/engine.go` | ✓ | ✓ | ✓ | |
| `internal/scanengine/queue/fair.go` | ✓ | | | |
| `internal/scanengine/queue/priority.go` | ✓ | | | |
| `internal/scanengine/scheduler/*` | ✓ | | | |
| `internal/scanengine/recovery/orphan.go` | ✓ | | | |
| `internal/scanengine/pool/*` | | ✓ | ✓ | |
| `internal/scanengine/domainpool/pool.go` | | ✓ | | |
| `internal/scanengine/core/rules_external.go` | | ✓ | ✓ | |
| `internal/scanconfig/nuclei_routing.go` | | | ✓ | |
| `internal/scanengine/work/store.go` | | ✓ | ✓ | |
| `internal/db/v39+.go` | | ✓ | | |
| `internal/api/scan_work_handlers.go` | ✓ | | | |
| `internal/api/pipeline_handlers.go` | ✓ | | | |
| `internal/api/server.go` | ✓ | | | |
| `internal/api/README.md` | ✓ | | | ✓ |
| `configs/scan.config.yaml` | | | ✓ | |
| `frontend/src/pages/RunsPage.tsx` | ✓ | | | |
| `frontend/src/lib/api.ts` | ✓ | | | |
| `docs/current/architecture.md` | | | | ✓ |

---

## Verify 阶段（全部 P 完成后）

- [ ] REQ-B1 ～ REQ-B11 逐项勾选
- [ ] 1067 domain 项目复扫：tool_calls、works、前端卡顿对比记录
- [ ] `go test ./internal/scanengine/...` 全绿
- [ ] Docker E2E smoke 通过

---

## 进度跟踪

| 阶段 | 状态 | 备注 |
|------|------|------|
| P0 调度+生命周期+CDN+分页 | 代码完成 | 待 E2E 验收 |
| P1 Tier1 池 | 代码完成 | 待 E2E 验收 |
| P2 Tier2 httpx/nmap/nuclei | 代码完成 | 待 E2E 验收 |
| P3 E2E+文档 | 代码完成 | 待 Docker E2E 绿 |

---

## Implement 门禁

- [ ] [proposal.md](./proposal.md) 用户已批准
- [ ] 三分法 + nuclei 弃用策略 A 已确认
- [ ] 不设默认 AbsoluteTimeout 已确认
