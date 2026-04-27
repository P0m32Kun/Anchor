# Anchor Wiki Index

> Anchor 项目知识库索引。Agent 应先读此文件了解知识库结构。

---

## 📂 目录

### 核心指令
- [`SCHEMA.md`](./SCHEMA.md) — **AI 指令文件**（必读，包含技术栈、架构决策、编码约定、安全红线、里程碑状态）

### 架构决策 (ADR)
- [`decisions/001-tauri-go-communication.md`](./decisions/001-tauri-go-communication.md) — Tauri ↔ Go 通信模式
- [`decisions/002-sse-over-websocket.md`](./decisions/002-sse-over-websocket.md) — SSE 替代 WebSocket
- [`decisions/003-zustand-state-management.md`](./decisions/003-zustand-state-management.md) — Zustand 状态管理
- [`decisions/004-sqlite-wal.md`](./decisions/004-sqlite-wal.md) — SQLite WAL 模式
- [`decisions/005-worker-in-process.md`](./decisions/005-worker-in-process.md) — Worker 同进程模型
- [`decisions/006-scope-check-gate.md`](./decisions/006-scope-check-gate.md) — Scope Check 强制门控
- [`decisions/007-fingerprint-driven-nuclei-scanning.md`](./decisions/007-fingerprint-driven-nuclei-scanning.md) — 指纹驱动 Nuclei 模板精确筛选
- [`decisions/008-asset-normalization.md`](./decisions/008-asset-normalization.md) — 资产归一化与去重

### 编码约定
- [`conventions/frontend-conventions.md`](./conventions/frontend-conventions.md) — 前端编码约定
- [`conventions/backend-conventions.md`](./conventions/backend-conventions.md) — 后端编码约定
- [`conventions/api-contracts.md`](./conventions/api-contracts.md) — 前后端 API 契约

### 踩坑记录
- [`pitfalls/20260426-frontend-backend-field-mismatch.md`](./pitfalls/20260426-frontend-backend-field-mismatch.md) — 前端字段名与后端不匹配
- [`pitfalls/20260426-artifact-type-mismatch.md`](./pitfalls/20260426-artifact-type-mismatch.md) — Worker Artifact 类型与工作流不匹配
- [`pitfalls/20260426-asset-scope-check-missing.md`](./pitfalls/20260426-asset-scope-check-missing.md) — 发现的资产未过 Scope Check
- [`pitfalls/20260426-raw-artifact-redaction-loss.md`](./pitfalls/20260426-raw-artifact-redaction-loss.md) — 原始证据被脱敏覆盖
- [`pitfalls/20260427-null-scan-crash.md`](./pitfalls/20260427-null-scan-crash.md) — NULL 列 Scan 到 string 崩溃
- [`pitfalls/20260427-markdown-pipe-corruption.md`](./pitfalls/20260427-markdown-pipe-corruption.md) — Markdown 表格被 `|` 破坏

### 变更日志
- [`log.md`](./log.md) — 项目变更日志（按时间倒序）

---

## 🔄 维护规则

1. **每次关键决策后**：在 `decisions/` 添加新的 ADR
2. **每次踩坑后**：在 `pitfalls/` 添加新记录
3. **每次里程碑完成后**：更新 `SCHEMA.md` 中的里程碑状态
4. **每次架构变更后**：更新 `SCHEMA.md` 中的活跃决策列表
5. **Agent 执行任务前**：先读 `SCHEMA.md` 了解项目上下文
