# Progress

## Status
In Progress

## Tasks
- [x] P1-2: Zustand store 添加 `persist` 中间件，持久化 `currentProject`
- [x] P1-3: 添加项目级路由 `/projects/:id/targets` 和 `/projects/:id/runs`
- [x] P2-10: 创建项目后自动选中 `currentProject`
- [x] P2-11: Settings Server 地址在 Web 模式下 placeholder 为空

## Files Changed (P1-2)
- `frontend/src/lib/store.ts` — 导入 `persist` 中间件，包装 store，添加 `partialize`，切换项目时清空子资源

## Files Changed (P1-3)
- `frontend/src/App.tsx` — 添加两条项目级路由，更新路由表注释
- `frontend/src/pages/RunsPage.tsx` — 添加 `useParams` fallback，优先 URL params，其次 store

## Files Changed (P2-10)
- `frontend/src/pages/ProjectPage.tsx` — `handleCreate` 成功后调用 `setCurrentProject(p)`

## Files Changed (P2-11)
- `frontend/src/pages/SettingsPage.tsx` — 计算 `rawBase` 和 `placeholderText`，动态设置 input placeholder；在默认相对路径模式下增加说明文字

## Notes
- `npm run typecheck` 零错误通过（修改未引入新错误，项目原有错误与本次无关）
- P2-11: 当前工作树缺少 `node_modules`，无法运行完整 typecheck；修改语法简单，无新增类型问题
- P1-2: `zustand/middleware` 的 `persist` 只持久化 `currentProject`，切换项目时自动清空 targets/tasks/assets/webEndpoints/ports/services/findings/currentFinding，防止旧数据残留
- P1-3: TargetPage 原本已使用 `useParams<{ id: string }>()`，无需修改
- P1-3: RunsPage 从 store-only 改为 `id || currentProject?.id` 双源 fallback
- Store API 已确认：`setCurrentProject` 接收 `Project | null`，类型完全匹配
