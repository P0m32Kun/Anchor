---
archived: true
archived_at: "2026-04-29"
archived_by: doc-archivist
version: "v0.1"
original_path: "wiki/decisions/001-tauri-go-communication.md"
status: "completed"
reason: "v0.1 架构决策记录，v0.2 阶段结束"
---

# ADR-001: Tauri ↔ Go 通信模式

## 状态
✅ 已决策（2026-04-26）

## 背景
Anchor 是 Tauri 桌面应用 + Go 本地服务的架构。需要确定前端（Tauri/React）与后端（Go）之间的通信方式。

## 决策
**MVP 使用 HTTP API (:8080)，后续考虑迁移到 Tauri Command 模式。**

## 理由
| 因素 | HTTP API | Tauri Command |
|------|----------|---------------|
| 开发调试 | ✅ 直接 curl/postman 调试 | ❌ 需要启动 Tauri 进程 |
| 实时推送 | ✅ SSE 简单实现 | ⚠️ 需要 Tauri Event |
| 类型安全 | ⚠️ 手动维护 | ✅ 自动生成 TypeScript 类型 |
| Native API 访问 | ❌ 无法调用 | ✅ 可调用 Rust API |
| 跨平台路径 | ✅ 统一 localhost | ⚠️ Tauri sidecar 有坑 |
| 进程管理 | ⚠️ 需自行管理 Go 进程 | ✅ Tauri 管理 sidecar |

MVP 优先开发速度，HTTP 模式足够。v0.2 再评估是否需要 Native API 能力。

## 当前实现
- Go HTTP server 监听 `:8080`
- 前端通过 `api.ts` 统一封装 HTTP 调用
- SSE 端点 `/events` 用于实时推送任务状态

## 风险
- Windows 上 Tauri sidecar 路径处理有已知坑（已记录，待验证）
- HTTP 模式无法利用 Tauri 的 Native API（文件系统、通知等）

## 替代方案
- **Tauri Command + Sidecar**：类型安全更好，Native API 可用，但开发调试慢
- **gRPC**：强类型 + 高性能，但引入额外依赖和复杂度

## 迁移路径
v0.2 时评估：如果不需要 Native API，保持 HTTP；如果需要，逐步迁移关键 API 到 Tauri Command。
