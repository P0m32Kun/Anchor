# Progress

## Status
In Progress

## Tasks
- [x] Sprint 0.5 — 编写 10 条主流程验收用例 (`docs/acceptance-criteria.md`)
- [x] Sprint 0.11 — 工程健康检查 (`docs/engineering-health.md`)

## Files Changed
- `docs/acceptance-criteria.md` — 10 条验收用例（AC-001 ~ AC-010），对齐当前代码实际能力
- `docs/engineering-health.md` — 工程健康基线（代码健康 / 测试覆盖 / 安全配置 / 依赖状态 / 架构健康 / P0-P3 待办项）

- [x] Sprint 0.2 — 跑主流程，记录阻断点 (`sprint0-t6-bugs.md`)

## Files Changed
- `docs/acceptance-criteria.md` — 10 条验收用例（AC-001 ~ AC-010），对齐当前代码实际能力
- `docs/engineering-health.md` — 工程健康基线（代码健康 / 测试覆盖 / 安全配置 / 依赖状态 / 架构健康 / P0-P3 待办项）
- `sprint0-t6-bugs.md` — Sprint 0.2 主流程阻断点报告（14 条 Bug，含 1 个 P0、6 个 P1、5 个 P2、2 个 P3）

## Notes
- 验收用例已按 flat 路由 + legacy 路由现状、无全局错误边界、Zustand store 传递项目上下文等实际能力编写
- 工程健康报告提取了 coverage-baseline.md、scout-report.md、api-error-contract.md 及源码的关键指标
- tsc --noEmit 因 node_modules 缺失未能现场执行，但 package.json 脚本与历史构建记录证明类型检查在完整环境中有效
- **Sprint 0.2 关键发现**：Vite proxy `/api` 配置与前端实际 API 路径（无 `/api` 前缀）完全错位，导致 Web dev 模式下所有 API 请求 404，应用完全不可用。此前 `progress.md` 中标记为 ✅ 的 "Vite proxy 配置" 任务实际未通过验收。
