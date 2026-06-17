---
status: draft
source_of_truth: false
owner: kun
created: 2026-06-17
scope: batch-scan-scheduling
---

# 批量扫描调度 — Spec

## Purpose

定义批量扫描调度改造的可观察行为。build/typecheck alone 不算完成；须 Docker/E2E 实扫验证。

---

## REQ-B1：调度器接线（Stage Rank + 公平 + 动态并发）

**验收信号**：

- [ ] `engine.tick` 使用 `PopFairStaged`，非 `Pop` + `ClassifyAction` 优先级
- [ ] `ComputeLimits(seedCount, elapsed)` 驱动 sem 宽度（非固定 BatchSize=5）
- [ ] `IPThrottler` 在 `executeWork` 对 host/IP Acquire/Release
- [ ] 单测 `queue/stage_rank_test.go`、`scheduler/limits_test.go`、`scheduler/ip_throttle_test.go` 全绿
- [ ] 集成测试：100 seed 的 fake run 中，DNS stage work 先于 HTTPX stage 被 claim

**场景**：

- Given 同一 run 内既有 pending DNS 又有 pending HTTPX
- When scheduler pop
- Then 仅 pop DNS（或同级 stage），HTTPX 等待

---

## REQ-B2：Tier1 大池批量（dnsx / naabu / cdncheck / subfinder）

**验收信号**：

- [ ] 50 个 IP 的 PORT_SCAN 合并为 1 次 naabu 调用（`-l` 文件含 50 行）
- [ ] 100 个 host 的 DNS_RESOLVE 合并为 1 次 dnsx 调用
- [ ] subfinder 使用 `domainpool` 或等价 DomainPool，非 1 domain 1 进程
- [ ] `onWorkComplete` 逐行创建 IP:port / 子域资产，归因正确
- [ ] 1067 domain 外网 scan：tool call 总数 **< 500**（P1 完成后）

**场景**：

- Given 100 个 alive IP 进入 port scan 阶段
- When pool flush
- Then 创建 1 条 batch work；worker 日志仅 1 次 naabu 启动

---

## REQ-B3：Tier2 — httpx 候选池

**验收信号**：

- [ ] subdomain、IP、IPPort 去重后进入 HTTPCandidatePool
- [ ] flush 100 target/批，1 次 httpx
- [ ] httpx 输出 technologies 写入 `asset_state`（REQ-4 延续）

---

## REQ-B4：Tier2 — nmap IP port-set 聚合

**验收信号**：

- [ ] IP `1.2.3.4` 开放 80,443,8080 → **1 次** nmap `-p 80,443,8080`
- [ ] 非 3 次 nmap（每 port 一次）
- [ ] nmap XML 解析后每个 port 更新对应 IP:port 资产 service 属性

**场景**：

- Given naabu 发现 1.2.3.4:80 与 1.2.3.4:443
- When port scan 阶段完成
- Then 仅 1 条 SERVICE_FINGERPRINT work，params ports=[80,443]

---

## REQ-B5：Tier2 — Nuclei tech 路由分桶（禁止策略 A）

**验收信号**：

- [ ] `configs/scan.config.yaml` 含 `nuclei_tech_routing` 段
- [ ] httpx 完成后按 technologies 分桶，**不同 TemplateSet 不混批**
- [ ] `noise_level=low` 时无 tech URL **不创建** nuclei work
- [ ] 禁止默认 `-tags exposure,cve` 宽扫整批 URL
- [ ] 每桶 nuclei 启动前可统计 templates 数 ≤ `max_templates`
- [ ] finding 的 asset 归因正确（matched-at → asset_id）

**场景**：

- Given 10 个 jenkins URL 与 10 个 nginx URL 均已 fingerprint
- When nuclei 阶段调度
- Then 至少 2 次 nuclei 调用（jenkins 桶、nginx 桶），且 jenkins 桶不含 nginx-only templates

---

## REQ-B6：Tier3 单点工具

**验收信号**：

- [ ] katana/spoor/ffuf 仍为 1 URL/work
- [ ] 仅 `isHighValueHTTP` 资产派生（延续 G3）
- [ ] custom nuclei `template_path` 单点不进入 TagBucket

---

## REQ-B7：Run 生命周期（无默认硬超时）

**验收信号**：

- [ ] `DefaultEngineConfig` 无 `AbsoluteTimeout` 或值为 0/禁用
- [ ] 长 run（>2h fake clock 或集成测试）不因绝对超时自动 stop
- [ ] Server 重启后 orphan run 标为 failed/cancelled，非永久 running
- [ ] `finalizePipelineRun` 处理 running 僵尸 work
- [ ] Cancel 后 running work 在合理时间内 terminal

---

## REQ-B8：CDN IP:port 归一化

**验收信号**：

- [ ] 资产 value `1.2.3.4:443` 执行 CDN_CHECK 不再报 `requires IP, got domain`
- [ ] 单元测试：`assetHostValue` / `buildParams` CDN 路径

---

## REQ-B9：API 与前端分页

**验收信号**：

- [ ] `GET .../works?page=1&page_size=50` 返回分页结构，默认 page_size=50
- [ ] `GET .../tool-calls` 同上
- [ ] RunsPage 运行中轮询不拉全量 works/tool-calls
- [ ] 选中 run 后 works 分页加载；1000+ work 时页面无明显卡顿

---

## REQ-B10：Profile 派生收敛

**验收信号**：

- [ ] `DNS_RESOLVE` MaxDepth 非 -1（或等价「已 resolved 跳过」门控）
- [ ] CDN 不 per-asset enqueue，走 Tier1 池
- [ ] 1067 domain run：work item 总数 **< 1000**（P2 完成后，对比现 6597）

---

## REQ-B11：文档同步

**验收信号**：

- [ ] `docs/current/architecture.md` 描述 Pool/BatchWork/Stage 调度
- [ ] `internal/api/README.md` 分页参数
- [ ] `docs/functional-test.md` 登记 FT-BATCH-01～03
- [ ] `docs/active/review/batch-scan-scheduling-acceptance.md` 产出

---

## 非功能指标（参考）

| 指标 | 1067 domain 外网 low noise | 测量方式 |
|------|---------------------------|----------|
| tool call 总数 | < 300（P2 后） | `COUNT(tool_call_logs)` |
| work item 总数 | < 1000 | `COUNT(scan_work_items)` |
| 前端轮询 payload | < 50KB/3s | DevTools network |
| DNS 阶段完成率 @ httpx 启动 | > 95% | run summary phases |

---

## 与现有 REQ 关系

| 现有 | 关系 |
|------|------|
| scan-engine-convergence REQ-4 asset_state | REQ-B5 前置依赖 |
| scan-engine-convergence REQ-5 high-value | REQ-B6 延续 |
| hw-scan-optimization AD-3/AD-5 | 本 spec REQ-B1 落地 |
