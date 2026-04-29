# Progress

## Status
In Progress

## Tasks
- [x] Sprint 1.5 — Modal 组件实现
- [x] Sprint 1.5 — ConfirmDialog 组件实现
- [x] Sprint 1.5 — 组件导出更新
- [x] Sprint 1.1 — ProjectLayout + 嵌套路由实现

## Files Changed
- `frontend/src/components/Modal.tsx` (新建)
- `frontend/src/components/ConfirmDialog.tsx` (新建)
- `frontend/src/components/index.ts` (更新导出)
- `frontend/src/components/ProjectLayout.tsx` (新建)
- `frontend/src/App.tsx` (修改路由，添加嵌套路由)
- `frontend/src/lib/store.ts` (添加 currentProjectId / setCurrentProjectId)
- `frontend/src/pages/TargetPage.tsx` (移除 useParams id，改用 useProjectId)
- `frontend/src/pages/AssetPage.tsx` (移除 useParams id，改用 useProjectId)
- `frontend/src/pages/FindingsPage.tsx` (移除 useParams id，改用 useProjectId)
- `frontend/src/pages/ReportsPage.tsx` (移除 useParams id，改用 useProjectId)
- `frontend/src/pages/RunsPage.tsx` (移除 useParams id，改用 useProjectId)
- `frontend/e2e/tests/ProjectLayout.e2e.md` (新增 E2E 测试计划)

## Notes
- `Button.tsx` 已支持 `forwardRef`，无需修改
- `tailwind.config.js` 中已定义所有所需 token（`animate-fade-in`、`animate-slide-up`、`bg-surface-elevated`、`border-glass-border` 等）
- `npm run typecheck` 环境中 SettingsPage 及基础组件存在已有类型错误，本次修改未引入新的结构性类型错误
- Legacy 路由 `/projects/:id/*` 保留，Sprint 1.11 统一移除
