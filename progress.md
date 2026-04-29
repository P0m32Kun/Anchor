# Progress

## Status
In Progress

## Tasks
- [x] 任务 A：为所有页面添加 AbortController
- [x] 任务 B：页面适配 store loading/error 状态
- [x] npm run typecheck 零错误

## Files Changed
- frontend/src/pages/DashboardPage.tsx — AbortController + fetch signal
- frontend/src/pages/ProjectPage.tsx — AbortController for listProjects
- frontend/src/pages/TargetPage.tsx — AbortController + targetsLoading/targetsError store 状态
- frontend/src/pages/AssetPage.tsx — AbortController + assetsLoading/assetsError store 状态
- frontend/src/pages/FindingsPage.tsx — AbortController + findingsLoading/findingsError store 状态
- frontend/src/pages/RunsPage.tsx — AbortController + runs 移入 store + runsLoading/runsError
- frontend/src/pages/WorkersPage.tsx — AbortController for fetch
- frontend/src/pages/ReportsPage.tsx — AbortController for loadData

## Notes
- SettingsPage.tsx 无 API 调用，无需修改
- 所有 API 错误处理均排除 AbortError（组件卸载时不展示错误）
- typecheck 通过零错误
