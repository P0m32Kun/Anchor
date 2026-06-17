---
status: draft
source_of_truth: false
owner: kun
created: 2026-06-13
scope: bounty-watch
related_spec: docs/design/bounty-watch/spec.md
related_tasks: docs/design/bounty-watch/tasks.md
---

# Bounty Watch — Acceptance

> 全部 BW 阶段完成后逐项勾选。单项 build/typecheck 不算通过。

## BW0 — 可选 Scope 边界

- [ ] `projects.scope_boundary_mode` 默认 `off`
- [ ] 项目设置可切换 `strict`；ScanModal 只读展示
- [ ] strict + include `*.example.com`：in-scope 子域进入引擎，OOS 不进入
- [ ] off 模式：`external-scan-conv.spec.ts` 回归通过
- [ ] E2E-SCOPE-BW-01 绿
- [ ] FT-SCOPE-BW-01 绿

## BW1 — Bounty Preset + Spoor

- [ ] `DefaultBountyPipelineConfig()` + scan.config presets.bounty
- [ ] Spoor 独立于 katana；katana=false 时 Spoor 仍可跑
- [ ] ScanModal 赏金 preset 可选
- [ ] E2E-BOUNTY-PRESET-01 绿
- [ ] FT-SPOOR-BW-01 绿

## BW2 — Signal Inbox

- [ ] 扫描后自动生成 signals（无需手动 refresh）
- [ ] `GET /projects/{id}/signals` 按 score 排序
- [ ] strict scope：out-of-scope 默认不在 inbox
- [ ] UI Inbox 页可 dismiss
- [ ] E2E-INBOX-01 绿

## BW3 — Delta + 增量扫描

- [ ] `first_seen_after` / `created_after` / `since` 查询 API
- [ ] 二次 run skip stable httpx（可配置天数）
- [ ] run summary 含 new_* 计数
- [ ] E2E-DELTA-01 绿

## BW4 — Watch Mode

- [ ] watch_enabled + interval 配置
- [ ] 到期自动 passive tick
- [ ] SSE `asset.new` / `signal.new`
- [ ] E2E-WATCH-01 绿

## BW5 — 边缘发现

- [ ] gau query / crt SAN / JS_URL 至少两项落地
- [ ] parser unit tests 绿
- [ ] architecture.md 已同步

## 文档

- [ ] `internal/api/README.md` 与 handler 变更一致
- [ ] `docs/current/architecture.md` §Bounty Watch
- [ ] `docs/functional-test.md` 场景表完整
- [ ] `docs/current/plan.md` workstream → Accepted

## 最终回归

- [ ] 全量 BW E2E spec 绿
- [ ] `company-scan-flow` / `external-scan-conv` 无回归
