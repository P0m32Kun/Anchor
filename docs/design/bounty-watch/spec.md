---
status: proposed
source_of_truth: false
owner: kun
created: 2026-06-13
scope: bounty-watch
---

# Bounty Watch — Spec

## Purpose

定义 Anchor 从「一次性护网扫描」演进为「赏金攻击面持续监控底座」的可观察行为。代码须满足本 spec 全部 REQ；**build/typecheck alone 不算完成**；每 REQ 须映射 E2E 或 integration 验收。

## 产品叙事

```text
赏金目标 (ip/cidr/domain/company)
        │
        ▼
  [可选] Scope 边界过滤 ──► seed 注入
        │
        ▼
  ScanEngine / Watch tick ──► 资产图 + lineage
        │
        ▼
  指纹 / Spoor / Nuclei ──► Signal Inbox（评分排序）
        │
        ▼
  Delta feed（新资产 / 新信号）──► 用户 / 外部 Agent
```

---

## REQ-BW0: 可选 Scope 边界（项目级）

**原则**：默认**关闭**（护网行为）；用户手动开启后限定域名/IP 扫描范围。

**验收信号**：

- [ ] DB: `projects.scope_boundary_mode` 存在，默认 `off`
- [ ] API: `PATCH /projects/{id}` 可设置 `scope_boundary_mode`: `off` | `strict`
- [ ] UI: 项目设置页有 Scope 边界开关 + 说明（护网建议关闭；赏金建议开启）
- [ ] Given `scope_boundary_mode=off`，When 扫描扩面，Then include 规则**不**阻止子域/IP 注入（与当前护网 exclude-only 行为一致或可配置 exclude）
- [ ] Given `scope_boundary_mode=strict` 且 include=`*.example.com`，When passive 展开 `evil.com`，Then 该 seed **不**进入引擎
- [ ] Given `strict` + include=`*.example.com`，When 展开 `api.example.com`，Then seed 进入引擎
- [ ] Given `strict` + exclude=`staging.example.com`，When 发现该子域，Then Work 被 skip、资产标记 `scope_status=out_of_scope`
- [ ] Finding 入库：out-of-scope endpoint 的 finding **不**进入 inbox（可写 DB 但 `scope_status=out`，默认 API 不返回）
- [ ] Unit: `internal/scope/scope_test.go` wildcard include + exclude 冲突
- [ ] E2E: `E2E-SCOPE-BW-01` strict 模式下 OOS 子域不出现在 AssetPage

**场景**：

- Given 赏金项目 scope include=`*.acme.com`，exclude=`*.acme.cn`
- When  启动扫描且 passive 返回 `www.acme.com` 与 `www.acme.cn`
- Then  仅 `www.acme.com` 进入资产列表；`www.acme.cn` 标记 out-of-scope

---

## REQ-BW1: Bounty 扫描 Preset + Spoor 独立

**验收信号**：

- [ ] `DefaultBountyPipelineConfig()` 存在：`EnablePassiveSearch=true`；katana 对 high-value 可开；**Spoor 独立字段** `EnableSpoor=true`（不依赖 katana）
- [ ] `scan.config.yaml` presets 段含 `bounty`；`GET /scan/defaults` 返回 bounty preset
- [ ] ScanModal 扫描模式含「赏金监控」选项（或 bounty 子 preset）
- [ ] `profile_config.go`: `ActionSpoorScan` 读 `EnableSpoor`，不再绑定 `EnableKatana`
- [ ] Given 外网 bounty preset + high-value HTTP，When 扫描完成，Then 至少 1 条 Spoor work 执行（mock 环境）
- [ ] Unit: `engine_test.go` / `profile_config_test.go` Spoor 在 katana=false 时仍可启用
- [ ] E2E: `E2E-BOUNTY-PRESET-01` bounty 模式 Spoor stage 可见或 task 存在

---

## REQ-BW2: Signal Inbox（高价值汇聚）

**验收信号**：

- [ ] DB: `signals` 表或扩展 `bounty_candidates` 支持多 `source_kind`（finding / spoor / endpoint / asset_new）
- [ ] 扫描完成或 Spoor/Nuclei 入库时自动 upsert signal（无需手动 refresh）
- [ ] 评分维度：severity、novelty（7 天内 first_seen）、scope_score、edge_score（passive 来源加权）
- [ ] API: `GET /projects/{id}/signals?min_score=&since=&scope=in`
- [ ] API: `PATCH /signals/{id}` dismiss / pin
- [ ] UI: Inbox 页（或 Findings 页 Tab）按 score 降序；显示 lineage 摘要
- [ ] Given strict scope，When 生成 signal，Then out-of-scope 默认不在列表
- [ ] Unit: `internal/bounty/scorer_test.go` 覆盖各 source_kind
- [ ] E2E: `E2E-INBOX-01` 扫描后 Inbox 有 ≥1 条 signal

---

## REQ-BW3: Delta API + 增量扫描

**验收信号**：

- [ ] API: `GET /projects/{id}/assets?first_seen_after=` ISO8601
- [ ] API: `GET /projects/{id}/findings?created_after=`
- [ ] API: `GET /projects/{id}/signals?since=`
- [ ] Engine: 二次 run 时 hydrate `state_json`；已 fingerprinted 且 `last_seen` 在 N 天内 → skip httpx（可配置 `SkipStableAssetDays`）
- [ ] Run 摘要 API: `GET /projects/{id}/runs/{run_id}/summary` 含 `new_assets`、`new_findings`、`new_signals` 计数
- [ ] Unit: `asset/state_test.go` skip stable 逻辑
- [ ] E2E: `E2E-DELTA-01` 两次扫描第二次 new_assets 计数下降

---

## REQ-BW4: Watch Mode（持续监控）

**验收信号**：

- [ ] DB: `projects.watch_enabled`、`watch_interval_hours`、`watch_passive_only`
- [ ] Server 内置 scheduler（或 cron goroutine）：到期触发 passive-only scan
- [ ] passive-only tick：仅 `ExpandTargets` + 新 seed 注入 + light httpx（不 full portscan）
- [ ] SSE: `asset.new`、`signal.new` 事件含 project_id、run_id
- [ ] UI: 项目设置 Watch 开关 + interval
- [ ] Given watch_enabled + interval=24h，When 到期，Then 自动创建 run 且 mode 标记 `watch_passive`
- [ ] E2E: `E2E-WATCH-01`（可 mock 短 interval 或 API 手动 trigger tick）

---

## REQ-BW5: 边缘发现补强

**验收信号**：

- [ ] gau parser 提取带 query 的 URL → `HTTP_PATH` + params metadata
- [ ] crt.sh parser 提取 SAN 子域 → subdomain seed
- [ ] `AssetJSURL` 接线：katana 或专用 parser 产出 JS URL → 可选 Spoor pass
- [ ] Unit: parser tests 覆盖新字段
- [ ] 文档: architecture.md 更新资产类型表

**Defer（本阶段不做）**：

- subdomain takeover 专用 Action
- GitHub leak / 云 bucket 枚举

---

## BDD 场景映射

| 场景 ID | REQ | 自动化 |
|---------|-----|--------|
| E2E-SCOPE-BW-01 | BW0 | `frontend/e2e/tests/scope-boundary.spec.ts` |
| E2E-BOUNTY-PRESET-01 | BW1 | `frontend/e2e/tests/bounty-preset.spec.ts` |
| E2E-INBOX-01 | BW2 | `frontend/e2e/tests/signal-inbox.spec.ts` |
| E2E-DELTA-01 | BW3 | `frontend/e2e/tests/scan-delta.spec.ts` |
| E2E-WATCH-01 | BW4 | `frontend/e2e/tests/watch-mode.spec.ts` |
| FT-SCOPE-BW-01 | BW0 | `internal/scope/scope_boundary_test.go` |
| FT-SPOOR-BW-01 | BW1 | `internal/scanengine/core/profile_config_test.go` |

---

## 不在范围内（明确 defer）

| ID | 内容 |
|----|------|
| DEFER-BW-1 | Agent MCP / webhook 推送协议 |
| DEFER-BW-2 | 赏金平台 scope 自动导入 |
| DEFER-BW-3 | Subdomain takeover Action |
| DEFER-BW-4 | 多 project 全局 inbox |
