---
status: accepted
source_of_truth: false
owner: kun
created: 2026-06-17
scope: batch-scan-scheduling
---

# 批量扫描调度 — 验收记录

> 设计包：[`docs/design/batch-scan-scheduling/`](../../design/batch-scan-scheduling/proposal.md)  
> 架构基线：[`docs/current/architecture.md`](../../current/architecture.md) 「批量调度」章

## 结论

**P0–P3 代码与文档已于 2026-06-17 完成。** 批量 Pool + Stage 调度 + orphan 恢复 + 前端分页已落地；规模 E2E 与 orphan 单测已添加，**Docker 实扫需在 CI/本地 e2e 栈验证**。

## REQ 验收清单

| REQ | 摘要 | 证据 |
|-----|------|------|
| REQ-B1 | PopFairStaged + ComputeLimits + IPThrottler | `queue/fair.go`；`engine.go` tick；`scheduler/*_test.go` |
| REQ-B2 | Tier1 大池（dnsx/naabu/cdncheck/subfinder） | `engine_tier1.go`；`work/store_test.go` CreatePooledBatch |
| REQ-B3 | httpx 候选池 | `engine_tier2.go` httpPool；`pool/probe_target.go` |
| REQ-B4 | nmap IP port-set 聚合 | `pool/ip_port_agg.go`；`onBatchNmapComplete` |
| REQ-B5 | nuclei tech 分桶 | `scanconfig/nuclei_routing.go`；`pool/nuclei_buckets.go` |
| REQ-B6 | Tier3 单点工具保留 | katana/ffuf/spoor 仍 per-asset `store.Create` |
| REQ-B7 | 无默认 AbsoluteTimeout + orphan 恢复 | `DefaultEngineConfig`；`recovery/orphan.go` + `orphan_test.go` |
| REQ-B8 | CDN IP:port 归一化 | `hostWithoutPort`；`host_test.go` |
| REQ-B9 | works/tool-calls 分页 | `scan_work_handlers.go`；`RunsPage.tsx` |
| REQ-B10 | Profile 派生收敛 | `rules_external.go` DNS MaxDepth=1；Tier1/2 池化 |
| REQ-B11 | 文档同步 | 本文件 + `architecture.md` + `functional-test.md` |

## 阶段交付

| 阶段 | 交付物 |
|------|--------|
| P0 | fair queue、scheduler 接线、orphan、CDN、RunsPage 分页 |
| P1 | Tier1 Pool + v39 BatchWork schema |
| P2 | httpx/nmap/nuclei Tier2 池 + nuclei_tech_routing |
| P3 | E2E-BATCH-01、FT-BATCH-* 登记、架构文档 |

## 自动化验证（本地）

```bash
go test ./internal/scanengine/ ./internal/scanengine/pool/... ./internal/scanengine/recovery/... -count=1
go build .

# E2E（需 Docker e2e 栈）
cd frontend && npx playwright test e2e/tests/batch-scan-scale.spec.ts --project=chromium-scan
cd frontend && npx playwright test e2e/tests/external-scan-conv.spec.ts --project=chromium-scan
```

## 非功能指标（待实扫记录）

| 指标 | 目标 | 测量 |
|------|------|------|
| 1067 domain tool_calls | < 300 | `COUNT(tool_call_logs)` |
| 1067 domain works | < 1000 | `COUNT(scan_work_items)` |
| 120 domain E2E smoke | works ≤ 400, tool_calls ≤ 200 | E2E-BATCH-01 |

## 已知 defer

| ID | 内容 |
|----|------|
| DEFER-B1 | P2.4 custom nuclei RoutingPolicy 扩展 |
| DEFER-B2 | 调度↔rate 联动、Pool jitter（hw-scan AD-5 余项） |
| DEFER-B3 | 1067 domain 生产项目复扫对比数据（人工 FT） |

## Verify 待办

- [ ] Docker E2E：`batch-scan-scale.spec.ts` 绿
- [ ] 1067 domain 项目复扫填表（tool_calls / works / 前端 payload）
- [ ] `go test ./internal/scanconfig/...` 修复 config_test 与 PipelineConfig 字段漂移
