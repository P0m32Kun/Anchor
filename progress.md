# Progress

## Status
Sprint 2a-1 Done

## Tasks
- [x] api.ts 重构
  - [x] fetchJSON 改名为 fetchAPI
  - [x] 204 响应返回 null
  - [x] 添加 fetchBlob 函数（支持 signal、错误处理）
  - [x] APIError 扩展：code 类型细化、retryable 自动计算
  - [x] 错误分类细化：HTTP_5xx(retryable), HTTP_4xx, NON_JSON_RESPONSE
  - [x] api 对象所有方法添加 signal?: AbortSignal 参数
  - [x] exportReportMD/exportReportJSON 改用 fetchBlob
  - [x] importTargets 错误处理对齐新 APIError 规范
  - [x] App.tsx 全局错误处理器按 code 分类显示 Toast 标题
  - [x] `npm run typecheck` 零错误

## Files Changed
- frontend/src/lib/api.ts
- frontend/src/App.tsx

## Notes
APIError 现在提供结构化错误分类，便于 UI 根据 retryable 决定是否展示重试按钮。

---

## Status
Sprint 2a-3 Done

## Tasks
- [x] Zustand Store 改造
  - [x] 为 projects/targets/assets/runs/findings 添加 loading/error 状态
  - [x] 添加 runs 数组及 setter 到 store
  - [x] 改造 setCurrentProjectId：切换项目时清空所有子资源并重置 loading/error
  - [x] 改造 setCurrentProject：不再清空子资源（职责集中到 setCurrentProjectId）
  - [x] Persist 只持久化 currentProjectId
  - [x] `npm run typecheck` 零错误

## Files Changed
- frontend/src/lib/store.ts

## Notes
Store 现在支持各 slice 的独立 loading/error 状态，便于 UI 精细化展示骨架屏/错误提示。
