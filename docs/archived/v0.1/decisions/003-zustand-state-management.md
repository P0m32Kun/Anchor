---
archived: true
archived_at: "2026-04-29"
archived_by: doc-archivist
version: "v0.1"
original_path: "wiki/decisions/003-zustand-state-management.md"
status: "completed"
reason: "v0.1 架构决策记录，v0.2 阶段结束"
---

# ADR-003: Zustand 状态管理

## 状态
✅ 已决策（2026-04-26）

## 背景
前端需要管理项目列表、目标列表、任务状态、工具健康状态等数据。

## 决策
**使用 Zustand 替代 Redux Toolkit。**

## 理由
| 因素 | Zustand | Redux Toolkit |
|------|---------|---------------|
| 包大小 | ✅ ~1KB | ~11KB + deps |
| API 简洁 | ✅ 无样板代码 | 需写 slice、reducer |
| TypeScript | ✅ 原生支持 | 需额外配置 |
| 中间件 | ✅ 简单组合 | 成熟生态 |
|  DevTools | ✅ 支持 | 更成熟 |
| 复杂状态 | ⚠️ 够用但有限 | ✅ 更强大 |

MVP 状态不复杂（主要是 CRUD 列表 + 任务状态），Zustand 更轻量、API 更简洁。

## 当前实现
- `src/lib/store.ts` — Zustand store 定义
- 按 domain 拆分：projectStore、targetStore、taskStore

## 风险
- 如果未来状态复杂度增加（如 undo/redo、时间旅行调试），可能需要迁移到 Redux Toolkit

## 迁移路径
如果状态复杂度超出 Zustand 舒适区，可以：
1. 保留 Zustand 用于简单状态
2. 引入 Redux Toolkit 处理复杂状态机
3. 或直接用 Zustand 的 middleware（如 `zustand/middleware` 的 persist、devtools）
