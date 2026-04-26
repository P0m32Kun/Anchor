# SecBench Wiki Index

> SecBench 项目知识库索引。Agent 应先读此文件了解知识库结构。

---

## 📂 目录

### 核心指令
- [`SCHEMA.md`](./SCHEMA.md) — **AI 指令文件**（必读，包含技术栈、架构决策、编码约定、安全红线）

### 架构决策 (ADR)
- [`decisions/001-tauri-go-communication.md`](./decisions/001-tauri-go-communication.md) — Tauri ↔ Go 通信模式
- [`decisions/002-sse-over-websocket.md`](./decisions/002-sse-over-websocket.md) — SSE 替代 WebSocket
- [`decisions/003-zustand-state-management.md`](./decisions/003-zustand-state-management.md) — Zustand 状态管理
- [`decisions/004-sqlite-wal.md`](./decisions/004-sqlite-wal.md) — SQLite WAL 模式
- [`decisions/005-worker-in-process.md`](./decisions/005-worker-in-process.md) — Worker 同进程模型
- [`decisions/006-scope-check-gate.md`](./decisions/006-scope-check-gate.md) — Scope Check 强制门控

### 编码约定
- [`conventions/frontend-conventions.md`](./conventions/frontend-conventions.md) — 前端编码约定
- [`conventions/backend-conventions.md`](./conventions/backend-conventions.md) — 后端编码约定
- [`conventions/api-contracts.md`](./conventions/api-contracts.md) — 前后端 API 契约

### 踩坑记录
- [`pitfalls/`](./pitfalls/) — 项目踩坑记录（按时间倒序）

### 变更日志
- [`log.md`](./log.md) — 项目变更日志

---

## 🔄 维护规则

1. **每次关键决策后**：在 `decisions/` 添加新的 ADR
2. **每次踩坑后**：在 `pitfalls/` 添加新记录
3. **每次里程碑完成后**：更新 `SCHEMA.md` 中的里程碑状态
4. **每次架构变更后**：更新 `SCHEMA.md` 中的活跃决策列表
5. **Agent 执行任务前**：先读 `SCHEMA.md` 了解项目上下文
