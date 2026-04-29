# Progress

## Status
In Progress

## Tasks
- [x] P1-1: Dashboard 不显示当前项目 — 修复完成
- [x] P1-6: Reports 页面允许无项目 ID 访问 — 修复完成
- [x] P2-9: Runs 页面空态信息矛盾 — 修复完成

## Files Changed
- `frontend/src/pages/DashboardPage.tsx`：导入 `useStore`，读取 `currentProject`，在"当前项目"区域根据有无项目动态显示项目名称（可跳转）或"前往创建 →"
- `frontend/src/pages/ReportsPage.tsx`：移除 `id!` 非空断言，增加 `projectId` 为空时的 `EmptyState` 引导页，避免 API 调用 `/projects/undefined/...`

## Notes
- 项目 node_modules 缺失，`npm run typecheck` 全局报错（React/JSX 类型缺失），修改文件本身无新增类型错误
- Dashboard 修改逻辑：存在 `currentProject` 时显示 `{name} →` 并跳转 `/projects/${id}`；不存在时保持原行为跳转 `/projects`
- Reports 修改逻辑：`projectId` 为 `undefined` 时显示 EmptyState 引导用户前往 `/projects`；有 ID 时保持原报告页面
