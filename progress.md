# Progress

## Status
In Progress

## Tasks

### Sprint 1.3 + 1.6 — 颜色审计 + 组件微调 ✅

- [x] 搜索 `frontend/src/` 下所有硬编码颜色值（`bg-yellow-500/15`, `text-yellow-300`, `bg-blue-500/15`, `bg-zinc-800/60` 等）
- [x] 将硬编码颜色替换为语义 Token（RunsPage / AssetPage / FindingsPage / TargetPage / WorkersPage / ReportsPage）
- [x] Badge StatusBadge 补充 run status 映射（running→info, failed→danger, completed→success, pending→warning）
- [x] 排查原生 alert/confirm，产出使用清单（共 12 处，供 Sprint 1.8 替换）
- [x] 产出 `docs/design-tokens.md`
- [x] `npm run typecheck` 零错误

## Files Changed

- `frontend/src/pages/RunsPage.tsx`
- `frontend/src/pages/AssetPage.tsx`
- `frontend/src/pages/FindingsPage.tsx`
- `frontend/src/pages/TargetPage.tsx`
- `frontend/src/pages/WorkersPage.tsx`
- `frontend/src/pages/ReportsPage.tsx`
- `frontend/src/components/Badge.tsx`
- `frontend/docs/design-tokens.md`

## Notes
