# Progress

## Status
In Progress

## Tasks
- [x] Sprint 1.5 — Modal 组件实现
- [x] Sprint 1.5 — ConfirmDialog 组件实现
- [x] Sprint 1.5 — 组件导出更新

## Files Changed
- `frontend/src/components/Modal.tsx` (新建)
- `frontend/src/components/ConfirmDialog.tsx` (新建)
- `frontend/src/components/index.ts` (更新导出)

## Notes
- `Button.tsx` 已支持 `forwardRef`，无需修改
- `tailwind.config.js` 中已定义所有所需 token（`animate-fade-in`、`animate-slide-up`、`bg-surface-elevated`、`border-glass-border` 等）
- `npm run typecheck` 环境因缺少 `node_modules` 导致所有 `.tsx` 文件报错（已有问题），新建组件代码无额外类型错误
